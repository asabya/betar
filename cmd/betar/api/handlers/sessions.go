package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/asabya/betar/pkg/types"
	"github.com/gorilla/mux"
)

// SessionQuerier is the subset of session.Store used by these handlers.
type SessionQuerier interface {
	ListByAgent(ctx context.Context, agentID string) ([]*types.Session, error)
	Get(ctx context.Context, agentID, callerID string) (*types.Session, error)
}

func RegisterSessionHandlers(r *mux.Router, store SessionQuerier) {
	if store == nil {
		return
	}
	r.HandleFunc("/sessions/{agentID}", listSessionsHandler(store)).Methods("GET")
	r.HandleFunc("/sessions/{agentID}/{callerID}", getSessionHandler(store)).Methods("GET")
}

func listSessionsHandler(store SessionQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := mux.Vars(r)["agentID"]
		sessions, err := store.ListByAgent(r.Context(), agentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if sessions == nil {
			sessions = []*types.Session{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)
	}
}

func getSessionHandler(store SessionQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		sess, err := store.Get(r.Context(), vars["agentID"], vars["callerID"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if sess == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sess)
	}
}
