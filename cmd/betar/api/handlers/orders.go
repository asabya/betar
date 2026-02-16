package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/asabya/betar/internal/marketplace"
	"github.com/gorilla/mux"
)

func RegisterOrderHandlers(r *mux.Router, orderSvc *marketplace.OrderService, listingSvc *marketplace.AgentListingService) {
	h := &orderHandler{orderSvc: orderSvc, listingSvc: listingSvc}

	r.HandleFunc("/orders", h.listOrders).Methods("GET")
	r.HandleFunc("/orders", h.createOrder).Methods("POST")
}

type orderHandler struct {
	orderSvc   *marketplace.OrderService
	listingSvc *marketplace.AgentListingService
}

func (h *orderHandler) listOrders(w http.ResponseWriter, r *http.Request) {
	if h.orderSvc == nil {
		http.Error(w, "order service not available", http.StatusServiceUnavailable)
		return
	}

	orders := h.orderSvc.ListOrders()
	json.NewEncoder(w).Encode(orders)
}

func (h *orderHandler) createOrder(w http.ResponseWriter, r *http.Request) {
	if h.orderSvc == nil {
		http.Error(w, "order service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		AgentID string  `json:"agentId"`
		Price   float64 `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.AgentID == "" {
		http.Error(w, "agentId is required", http.StatusBadRequest)
		return
	}

	// Get listing to find seller
	var sellerID string
	if h.listingSvc != nil {
		if listing, ok := h.listingSvc.GetListing(req.AgentID); ok {
			sellerID = listing.SellerID
		}
	}

	order, err := h.orderSvc.CreateOrder(r.Context(), req.AgentID, sellerID, req.Price)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(order)
}
