package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/asabya/betar/cmd/betar/api/handlers"
	"github.com/asabya/betar/cmd/betar/dashboard"
	"github.com/asabya/betar/internal/agent"
	"github.com/asabya/betar/internal/eip8004"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/internal/workflow"
	"github.com/asabya/betar/pkg/a2a"
	"github.com/gorilla/mux"
)

type Server struct {
	httpServer     *http.Server
	port           int
	paymentService *marketplace.PaymentService
}

type Route struct {
	Path    string
	Method  string
	Handler http.HandlerFunc
}

func NewServer(port int, agentMgr *agent.Manager, listingSvc *marketplace.AgentListingService, orderSvc *marketplace.OrderService, p2pHost *p2p.Host, paymentSvc *marketplace.PaymentService, sessionStore handlers.SessionQuerier, orch *workflow.Orchestrator, walletAddr, dataDir string, eip8004Client ...*eip8004.Client) *Server {
	r := mux.NewRouter()

	// Add handlers
	if len(eip8004Client) > 0 {
		handlers.RegisterAgentHandlers(r, agentMgr, listingSvc, p2pHost, eip8004Client[0])
	} else {
		handlers.RegisterAgentHandlers(r, agentMgr, listingSvc, p2pHost)
	}
	handlers.RegisterWalletHandlers(r, paymentSvc)
	handlers.RegisterOrderHandlers(r, orderSvc, listingSvc)
	handlers.RegisterSessionHandlers(r, sessionStore)
	handlers.RegisterStatusHandlers(r, p2pHost, walletAddr, dataDir)
	if orch != nil {
		handlers.RegisterWorkflowHandlers(r, orch)
	}

	// Register reputation endpoint if eip8004 client is available
	if len(eip8004Client) > 0 && eip8004Client[0] != nil {
		handlers.RegisterReputationHandlers(r, eip8004Client[0])
	}

	// A2A Agent Card discovery
	if listingSvc != nil {
		r.HandleFunc("/{agentName}/.well-known/agent-card.json", func(w http.ResponseWriter, r *http.Request) {
			listings := listingSvc.ListListings()
			vars := mux.Vars(r)
			agentName := vars["agentName"]

			for _, listing := range listings {
				if listing == nil {
					continue
				}
				if listing.Name == agentName {
					card := a2a.AgentListingToAgentCard(listing, port)
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(card)
					return
				}
			}
			// http.Error(w, `{"error":"agent not found"}`, http.StatusNotFound)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "agent not found"})
		}).Methods("GET")
	}

	// Dashboard — embedded React SPA
	distFS, _ := fs.Sub(dashboard.Files, "dist")
	spaHandler := spaFileServer(distFS)
	r.PathPrefix("/dashboard").Handler(http.StripPrefix("/dashboard", spaHandler))

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// CORS middleware
	handler := corsMiddleware(r)

	return &Server{
		port: port,
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      handler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		paymentService: paymentSvc,
	}
}

// spaFileServer serves static files from the embedded FS and falls back to
// index.html for any path that doesn't match a real file (SPA client-side routing).
func spaFileServer(root fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to open the requested file
		path := r.URL.Path
		if path == "/" || path == "" {
			path = "index.html"
		} else if path[0] == '/' {
			path = path[1:]
		}
		if _, err := fs.Stat(root, path); err != nil {
			// File not found — serve index.html for SPA routing
			data, readErr := fs.ReadFile(root, "index.html")
			if readErr != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) Start() error {
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP API server error: %v\n", err)
		}
	}()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
