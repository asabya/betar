package handlers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/asabya/betar/internal/agent"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/internal/p2p"
	"github.com/gorilla/mux"
)

func RegisterAgentHandlers(r *mux.Router, agentMgr *agent.Manager, listingSvc *marketplace.AgentListingService, p2pHost *p2p.Host) {
	h := &agentHandler{agentMgr: agentMgr, listingSvc: listingSvc, p2pHost: p2pHost}

	r.HandleFunc("/agents", h.listAgents).Methods("GET")
	r.HandleFunc("/agents/local", h.listLocalAgents).Methods("GET")
	r.HandleFunc("/agents", h.registerAgent).Methods("POST")
	r.HandleFunc("/agents/{id}/execute", h.executeAgent).Methods("POST")
	r.HandleFunc("/agents/{id}/reputation", h.getReputation).Methods("GET")
	r.HandleFunc("/agents/{id}/validations", h.getValidations).Methods("GET")
}

type agentHandler struct {
	agentMgr   *agent.Manager
	listingSvc *marketplace.AgentListingService
	p2pHost    *p2p.Host
}

func (h *agentHandler) listAgents(w http.ResponseWriter, r *http.Request) {
	if h.listingSvc == nil {
		http.Error(w, "listing service not available", http.StatusServiceUnavailable)
		return
	}

	listings := h.listingSvc.ListListings()
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

	var req struct {
		Input string `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	output, err := h.agentMgr.ExecuteTask(r.Context(), agentID, req.Input)
	if err != nil {
		http.Error(w, fmt.Sprintf("execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"output": output})
}

// getReputation returns ERC-8004 on-chain reputation summary for an agent.
// The {id} path variable must be the decimal on-chain agentId (TokenID).
func (h *agentHandler) getReputation(w http.ResponseWriter, r *http.Request) {
	if h.agentMgr == nil {
		http.Error(w, "agent manager not available", http.StatusServiceUnavailable)
		return
	}
	eip := h.agentMgr.EIP8004Client()
	if eip == nil {
		http.Error(w, "EIP-8004 not configured", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	tokenID, ok := new(big.Int).SetString(vars["id"], 10)
	if !ok {
		http.Error(w, "invalid agentId", http.StatusBadRequest)
		return
	}

	count, summaryValue, decimals, err := eip.GetReputationSummary(r.Context(), tokenID, "", "")
	if err != nil {
		http.Error(w, fmt.Sprintf("reputation query failed: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"agentId":      tokenID.String(),
		"count":        count,
		"summaryValue": summaryValue,
		"decimals":     decimals,
	})
}

// getValidations returns ERC-8004 on-chain validation hashes for an agent.
// The {id} path variable must be the decimal on-chain agentId (TokenID).
func (h *agentHandler) getValidations(w http.ResponseWriter, r *http.Request) {
	if h.agentMgr == nil {
		http.Error(w, "agent manager not available", http.StatusServiceUnavailable)
		return
	}
	eip := h.agentMgr.EIP8004Client()
	if eip == nil {
		http.Error(w, "EIP-8004 not configured", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	tokenID, ok := new(big.Int).SetString(vars["id"], 10)
	if !ok {
		http.Error(w, "invalid agentId", http.StatusBadRequest)
		return
	}

	hashes, err := eip.GetAgentValidations(r.Context(), tokenID)
	if err != nil {
		http.Error(w, fmt.Sprintf("validations query failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert [][32]byte to hex strings for JSON readability.
	hexHashes := make([]string, len(hashes))
	for i, h := range hashes {
		hexHashes[i] = fmt.Sprintf("0x%x", h)
	}

	json.NewEncoder(w).Encode(map[string]any{
		"agentId":     tokenID.String(),
		"validations": hexHashes,
	})
}
