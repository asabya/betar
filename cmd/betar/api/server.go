package api

import (
	"context"
	"encoding/json"
	"fmt"
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
		r.HandleFunc("/.well-known/agent.json", func(w http.ResponseWriter, r *http.Request) {
			listings := listingSvc.ListListings()
			var cards []*a2a.AgentCard
			for _, l := range listings {
				if l != nil {
					cards = append(cards, a2a.AgentListingToAgentCard(l))
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(cards)
		}).Methods("GET")
	}

	// Dashboard — embedded single-page UI
	r.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		data, err := dashboard.Files.ReadFile("index.html")
		if err != nil {
			http.Error(w, "dashboard not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	}).Methods("GET")

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
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
