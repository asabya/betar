package p2p

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	libp2phttp "github.com/libp2p/go-libp2p/p2p/http"
	"github.com/libp2p/go-libp2p/core/peer"
)

// HTTPHost wraps a libp2p HTTP host that serves x402 protocol over HTTP.
type HTTPHost struct {
	httpHost *libp2phttp.Host
	handler  *X402StreamHandler
	listener net.Listener
	mux      *http.ServeMux
	port     int
}

// NewHTTPHost creates an HTTP host that serves x402 handlers over HTTP.
// It reuses the same X402StreamHandler's handler map for dispatch.
func NewHTTPHost(x402Handler *X402StreamHandler, port int) (*HTTPHost, error) {
	mux := http.NewServeMux()
	h := &HTTPHost{
		httpHost: &libp2phttp.Host{
			ServeMux:          mux,
			InsecureAllowHTTP: true,
		},
		handler: x402Handler,
		port:    port,
		mux:     mux,
	}
	return h, nil
}

// Start begins listening for HTTP requests.
func (h *HTTPHost) Start() error {
	// Register the x402 handler on the libp2p HTTP host for protocol discovery.
	h.httpHost.SetHTTPHandlerAtPath(
		X402ProtocolID,
		"/x402/libp2p/1.0.0/",
		h.serveX402(),
	)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", h.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", h.port, err)
	}
	h.listener = listener

	// Serve using the mux which now has the x402 handler + well-known endpoint.
	go func() {
		if err := http.Serve(listener, h.mux); err != nil && !errors.Is(err, net.ErrClosed) {
			log.Printf("x402 HTTP host serve error: %v", err)
		}
	}()
	return nil
}

// serveX402 returns an http.Handler that:
// 1. Reads X-Message-Type header for message type
// 2. Reads JSON body as data
// 3. Dispatches to the existing handler map
// 4. Sets X-Message-Type response header and writes JSON response
func (h *HTTPHost) serveX402() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		msgType := r.Header.Get("X-Message-Type")
		if msgType == "" {
			http.Error(w, "X-Message-Type header required", http.StatusBadRequest)
			return
		}

		data, err := io.ReadAll(io.LimitReader(r.Body, int64(maxMessageDataLen)))
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		// Look up handler from the shared handler map.
		handler, ok := h.handler.GetHandler(msgType)

		if !ok {
			w.Header().Set("X-Message-Type", "x402.error")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"version":    "1.0",
				"error_code": 1000,
				"error_name": "INVALID_MESSAGE",
				"message":    fmt.Sprintf("unknown message type: %s", msgType),
				"retryable":  false,
			})
			return
		}

		// Use empty peer ID for HTTP clients (no P2P identity).
		respType, respData, err := handler(r.Context(), peer.ID(""), msgType, data)
		if err != nil {
			w.Header().Set("X-Message-Type", "x402.error")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			errPayload := marshalError("", err.Error())
			w.Write(errPayload)
			return
		}

		w.Header().Set("X-Message-Type", respType)
		w.Header().Set("Content-Type", "application/json")

		// Use HTTP 402 status for payment required responses.
		if respType == "x402.payment_required" {
			w.WriteHeader(http.StatusPaymentRequired)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		w.Write(respData)
	})
}

// Addr returns the HTTP listen address.
func (h *HTTPHost) Addr() string {
	if h.listener == nil {
		return ""
	}
	return h.listener.Addr().String()
}

// Close stops the HTTP host.
func (h *HTTPHost) Close() error {
	if h.listener != nil {
		return h.listener.Close()
	}
	return nil
}
