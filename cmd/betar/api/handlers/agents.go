package handlers

import (
	"encoding/json"
	"fmt"
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
