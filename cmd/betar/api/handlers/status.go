package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/asabya/betar/internal/p2p"
	"github.com/gorilla/mux"
)

func RegisterStatusHandlers(r *mux.Router, p2pHost *p2p.Host, walletAddr, dataDir string) {
	h := &statusHandler{p2pHost: p2pHost, walletAddr: walletAddr, dataDir: dataDir}
	r.HandleFunc("/status", h.getStatus).Methods("GET")
	r.HandleFunc("/peers", h.getPeers).Methods("GET")
}

type statusHandler struct {
	p2pHost    *p2p.Host
	walletAddr string
	dataDir    string
}

type statusResponse struct {
	PeerID         string   `json:"peerId"`
	Addresses      []string `json:"addresses"`
	ConnectedPeers int      `json:"connectedPeers"`
	WalletAddress  string   `json:"walletAddress"`
	DataDir        string   `json:"dataDir"`
}

type peerInfo struct {
	ID    string   `json:"id"`
	Addrs []string `json:"addrs"`
}

func (h *statusHandler) getStatus(w http.ResponseWriter, r *http.Request) {
	resp := statusResponse{
		WalletAddress: h.walletAddr,
		DataDir:       h.dataDir,
	}

	if h.p2pHost != nil {
		resp.PeerID = h.p2pHost.ID().String()
		resp.Addresses = h.p2pHost.AddrStrings()
		resp.ConnectedPeers = len(h.p2pHost.RawHost().Network().Peers())
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *statusHandler) getPeers(w http.ResponseWriter, r *http.Request) {
	var peers []peerInfo

	if h.p2pHost != nil {
		for _, p := range h.p2pHost.RawHost().Network().Peers() {
			addrs := h.p2pHost.Peerstore().Addrs(p)
			addrStrs := make([]string, len(addrs))
			for i, a := range addrs {
				addrStrs[i] = a.String()
			}
			peers = append(peers, peerInfo{ID: p.String(), Addrs: addrStrs})
		}
	}

	if peers == nil {
		peers = []peerInfo{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(peers)
}
