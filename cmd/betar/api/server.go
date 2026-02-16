package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/asabya/betar/cmd/betar/api/handlers"
	"github.com/asabya/betar/internal/agent"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/internal/p2p"
	"github.com/gorilla/mux"
)

type Server struct {
	httpServer *http.Server
	port       int
}

func NewServer(port int, agentMgr *agent.Manager, listingSvc *marketplace.AgentListingService, orderSvc *marketplace.OrderService, p2pHost *p2p.Host) *Server {
	r := mux.NewRouter()

	// Add handlers
	handlers.RegisterAgentHandlers(r, agentMgr, listingSvc, p2pHost)
	handlers.RegisterWalletHandlers(r)
	handlers.RegisterOrderHandlers(r, orderSvc, listingSvc)

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	return &Server{
		port: port,
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      r,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
	}
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
