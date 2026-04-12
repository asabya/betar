package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/asabya/betar/internal/agent"
	"github.com/asabya/betar/internal/eip8004"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/pkg/types"
	"github.com/gorilla/mux"
)

func RegisterAgentHandlers(r *mux.Router, agentMgr *agent.Manager, listingSvc *marketplace.AgentListingService, p2pHost *p2p.Host, eip8004Client ...*eip8004.Client) {
	h := &agentHandler{agentMgr: agentMgr, listingSvc: listingSvc, p2pHost: p2pHost}
	if len(eip8004Client) > 0 {
		h.eip8004 = eip8004Client[0]
	}

	r.HandleFunc("/agents", h.listAgents).Methods("GET")
	r.HandleFunc("/agents/local", h.listLocalAgents).Methods("GET")
	r.HandleFunc("/agents", h.registerAgent).Methods("POST")
	r.HandleFunc("/agents/{id}/execute", h.executeAgent).Methods("POST")
}

type agentHandler struct {
	agentMgr   *agent.Manager
	listingSvc *marketplace.AgentListingService
	p2pHost    *p2p.Host
	eip8004    *eip8004.Client
}

func (h *agentHandler) listAgents(w http.ResponseWriter, r *http.Request) {
	if h.listingSvc == nil {
		http.Error(w, "listing service not available", http.StatusServiceUnavailable)
		return
	}

	listings := h.listingSvc.ListListings()

	// Optionally enrich with on-chain reputation when ?on-chain=true is set.
	if r.URL.Query().Get("on-chain") == "true" && h.eip8004 != nil {
		enriched := make([]*types.AgentListing, len(listings))
		copy(enriched, listings)
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		for i, l := range enriched {
			if l.TokenID == "" {
				continue
			}
			tokenID := new(big.Int)
			if _, ok := tokenID.SetString(l.TokenID, 10); !ok {
				continue
			}
			count, score, decimals, err := h.eip8004.GetReputationSummary(ctx, tokenID, "", "")
			if err != nil {
				continue
			}
			// Clone the listing to avoid mutating the CRDT copy.
			copy := *l
			copy.OnChainReputation = &types.OnChainReputation{
				Count:    count,
				Score:    score,
				Decimals: decimals,
			}
			enriched[i] = &copy
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(enriched)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listings)
}

func (h *agentHandler) listLocalAgents(w http.ResponseWriter, r *http.Request) {
	if h.agentMgr == nil {
		http.Error(w, "agent manager not available", http.StatusServiceUnavailable)
		return
	}

	agents := h.agentMgr.ListAgents()
	json.NewEncoder(w).Encode(agents)
}

func (h *agentHandler) registerAgent(w http.ResponseWriter, r *http.Request) {
	if h.agentMgr == nil {
		http.Error(w, "agent manager not available", http.StatusServiceUnavailable)
		return
	}

	var spec agent.AgentSpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	registered, err := h.agentMgr.RegisterAgent(r.Context(), spec)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(registered)
}

func (h *agentHandler) executeAgent(w http.ResponseWriter, r *http.Request) {
	if h.agentMgr == nil {
		http.Error(w, "agent manager not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	agentID := vars["id"]

	var req *types.AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If input is provided then we look for configured agents and execute
	// Otherwise, we forward http request to configured agent API and return the response as is.

	if req.Input != "" {
		output, err := h.agentMgr.ExecuteTask(r.Context(), agentID, types.AgentRequest{
			ID:       req.ID,
			Input:    req.Input,
			Params:   req.Params,
			Jsonrpc:  req.Jsonrpc,
			Method:   req.Method,
			Resource: req.Resource,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("execution failed: %v", err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"output": output})
		return
	}

	if !(len(req.Params.Message.Parts) > 0) {
		http.Error(w, "request body must contain 'parts' array when passing params in body.", http.StatusBadRequest)
		return
	}

	output, err := h.agentMgr.ExecuteTask(r.Context(), agentID, types.AgentRequest{
		ID:       req.ID,
		Params:   req.Params,
		Jsonrpc:  req.Jsonrpc,
		Method:   req.Method,
		Resource: req.Resource,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("execution failed: %v", err), http.StatusInternalServerError)
		return
	}
	var jsonOutput *struct {
		MessageType string          `json:"message_type"`
		Message     json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal([]byte(output), &jsonOutput); err == nil {
		// This is to match SendMessageSuccessResponse format in adk-client
		json.NewEncoder(w).Encode(jsonOutput.Message)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"output": output})
}

func RegisterReputationHandlers(r *mux.Router, eip8004Client *eip8004.Client) {
	r.HandleFunc("/agents/reputation/{tokenId}", handleGetReputation(eip8004Client)).Methods("GET")
}

func handleGetReputation(eip8004Client *eip8004.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenIDStr := mux.Vars(r)["tokenId"]
		tokenID := new(big.Int)
		if _, ok := tokenID.SetString(tokenIDStr, 10); !ok {
			http.Error(w, "invalid tokenId", http.StatusBadRequest)
			return
		}
		count, value, decimals, err := eip8004Client.GetReputationSummary(r.Context(), tokenID, "", "")
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get reputation: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":    count,
			"value":    value,
			"decimals": decimals,
		})
	}
}
