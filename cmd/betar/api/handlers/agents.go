package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/asabya/betar/internal/agent"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/internal/p2p"
	"github.com/gorilla/mux"
)

func RegisterAgentHandlers(r *mux.Router, agentMgr *agent.Manager, listingSvc *marketplace.AgentListingService, p2pHost *p2p.Host, paymentSvc *marketplace.PaymentService) {
	h := &agentHandler{agentMgr: agentMgr, listingSvc: listingSvc, p2pHost: p2pHost, paymentSvc: paymentSvc}

	r.HandleFunc("/agents", h.listAgents).Methods("GET")
	r.HandleFunc("/agents/local", h.listLocalAgents).Methods("GET")
	r.HandleFunc("/agents", h.registerAgent).Methods("POST")
	r.HandleFunc("/agents/{id}/execute", h.executeAgent).Methods("POST")
	r.HandleFunc("/payment/sign", h.signPayment).Methods("POST")
	r.HandleFunc("/payment/submit", h.submitPayment).Methods("POST")
}

type agentHandler struct {
	agentMgr   *agent.Manager
	listingSvc *marketplace.AgentListingService
	p2pHost    *p2p.Host
	paymentSvc *marketplace.PaymentService
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
		Input           string                     `json:"input"`
		PaymentHeader   *marketplace.PaymentHeader `json:"paymentHeader,omitempty"`
		TransactionHash string                     `json:"transactionHash,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	output, payResp, err := h.agentMgr.ExecuteTask(r.Context(), agentID, req.Input, req.PaymentHeader, req.TransactionHash)

	// If payment required, return 402
	if payResp != nil {
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(payResp)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"output": output})
}

func (h *agentHandler) signPayment(w http.ResponseWriter, r *http.Request) {
	if h.paymentSvc == nil {
		http.Error(w, "payment service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		PaymentRequirement marketplace.PaymentRequirement `json:"paymentRequirement"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("[signPayment] Signing payment requirement - Amount: %s %s, PayTo: %s\n",
		req.PaymentRequirement.MaxAmountRequired, req.PaymentRequirement.Asset, req.PaymentRequirement.PayTo)

	header, err := h.paymentSvc.SignRequirement(&req.PaymentRequirement, fmt.Sprintf("order-%d", time.Now().UnixNano()))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to sign payment: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Printf("[signPayment] Payment signed successfully - Payer: %s, PaymentID: %s\n", header.Payer, header.PaymentID)
	json.NewEncoder(w).Encode(header)
}

func (h *agentHandler) submitPayment(w http.ResponseWriter, r *http.Request) {
	if h.paymentSvc == nil {
		http.Error(w, "payment service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		PaymentHeader marketplace.PaymentHeader `json:"paymentHeader"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("[submitPayment] Submitting EIP-3009 transaction - Payer: %s, PayTo: %s\n",
		req.PaymentHeader.Payer, req.PaymentHeader.Requirement.PayTo)

	ctx := r.Context()
	txHash, err := h.paymentSvc.SubmitPayment(ctx, &req.PaymentHeader)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to submit payment: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Printf("[submitPayment] Transaction submitted - TxHash: %s\n", txHash)

	// Wait for confirmation
	fmt.Printf("[submitPayment] Waiting for transaction confirmation...\n")
	confirmed, err := h.paymentSvc.WaitForSettlement(ctx, txHash, 2*time.Minute)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to wait for confirmation: %v", err), http.StatusInternalServerError)
		return
	}

	if !confirmed {
		http.Error(w, "transaction not confirmed within timeout", http.StatusGatewayTimeout)
		return
	}

	fmt.Printf("[submitPayment] Transaction confirmed!\n")
	json.NewEncoder(w).Encode(map[string]string{"transactionHash": txHash, "status": "confirmed"})
}
