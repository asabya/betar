package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/asabya/betar/internal/config"
	"github.com/asabya/betar/internal/eth"
	"github.com/gorilla/mux"
)

func RegisterWalletHandlers(r *mux.Router) {
	r.HandleFunc("/wallet/balance", getBalance).Methods("GET")
}

func getBalance(w http.ResponseWriter, r *http.Request) {
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

	json.NewEncoder(w).Encode(map[string]interface{}{
		"address": wallet.AddressHex(),
		"balance": eth.WeiToEther(balance),
	})
}
