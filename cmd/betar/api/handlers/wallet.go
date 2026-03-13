package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/asabya/betar/internal/config"
	"github.com/asabya/betar/internal/eth"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/gorilla/mux"
)

func RegisterWalletHandlers(r *mux.Router, paymentSvc *marketplace.PaymentService) {
	h := &walletHandler{paymentSvc: paymentSvc}
	r.HandleFunc("/wallet/balance", h.getBalance).Methods("GET")
}

type walletHandler struct {
	paymentSvc *marketplace.PaymentService
}

func (h *walletHandler) getBalance(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if cfg.Ethereum.PrivateKey == "" {
		http.Error(w, "no private key configured", http.StatusBadRequest)
		return
	}

	wallet, err := eth.NewWallet(cfg.Ethereum.PrivateKey, cfg.Ethereum.RPCURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	balance, err := wallet.Balance(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"address": wallet.AddressHex(),
		"balance": eth.WeiToEther(balance),
	}

	// Add USDC balance if payment service is available
	if h.paymentSvc != nil {
		usdcBal, err := h.paymentSvc.GetUSDCBalance(r.Context())
		if err == nil && usdcBal != nil {
			usdcStr := marketplace.AtomicToUSDC(usdcBal.String())
			if usdcFloat, parseErr := strconv.ParseFloat(usdcStr, 64); parseErr == nil {
				resp["usdcBalance"] = usdcFloat
			}
		}
	}

	json.NewEncoder(w).Encode(resp)
}
