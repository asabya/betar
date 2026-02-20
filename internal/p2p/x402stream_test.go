package p2p

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

func newTestHost(t *testing.T) host.Host {
	t.Helper()
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("failed to create libp2p host: %v", err)
	}
	t.Cleanup(func() { _ = h.Close() })
	return h
}

func connectHosts(t *testing.T, a, b host.Host) {
	t.Helper()
	bInfo := peer.AddrInfo{ID: b.ID(), Addrs: b.Addrs()}
	if err := a.Connect(context.Background(), bInfo); err != nil {
		t.Fatalf("failed to connect hosts: %v", err)
	}
}

// TestX402StreamHandler_RequestResponse verifies a complete 2-message exchange:
// the server first returns x402.payment_required, then x402.response on the paid request.
func TestX402StreamHandler_RequestResponse(t *testing.T) {
	serverHost := newTestHost(t)
	clientHost := newTestHost(t)
	connectHosts(t, clientHost, serverHost)

	serverHandler := NewX402StreamHandler(serverHost)
	clientHandler := NewX402StreamHandler(clientHost)

	type reqBody struct {
		Step int `json:"step"`
	}

	// Track calls to simulate the 2-trip flow.
	callCount := 0

	serverHandler.RegisterHandler("x402.request", func(ctx context.Context, from peer.ID, msgType string, data []byte) (string, []byte, error) {
		callCount++
		if callCount == 1 {
			resp := map[string]interface{}{
				"version":        "1.0",
				"correlation_id": "corr-1",
				"challenge_nonce": "abc123",
				"challenge_expires_at": time.Now().Add(5 * time.Minute).Unix(),
				"message":        "Payment required",
			}
			b, _ := json.Marshal(resp)
			return "x402.payment_required", b, nil
		}
		resp := map[string]interface{}{
			"version":        "1.0",
			"correlation_id": "corr-1",
			"tx_hash":        "0xdeadbeef",
			"body":           []byte(`{"output":"ok"}`),
		}
		b, _ := json.Marshal(resp)
		return "x402.response", b, nil
	})

	ctx := context.Background()
	reqData, _ := json.Marshal(reqBody{Step: 1})

	// First call → payment_required.
	respType, respData, err := clientHandler.SendX402Message(ctx, serverHost.ID(), "x402.request", reqData)
	if err != nil {
		t.Fatalf("SendX402Message (1): %v", err)
	}
	if respType != "x402.payment_required" {
		t.Errorf("first response type: got %q, want x402.payment_required", respType)
	}
	if len(respData) == 0 {
		t.Error("expected non-empty response data")
	}

	// Second call → response.
	respType2, respData2, err := clientHandler.SendX402Message(ctx, serverHost.ID(), "x402.request", reqData)
	if err != nil {
		t.Fatalf("SendX402Message (2): %v", err)
	}
	if respType2 != "x402.response" {
		t.Errorf("second response type: got %q, want x402.response", respType2)
	}
	if len(respData2) == 0 {
		t.Error("expected non-empty response data")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respData2, &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if result["tx_hash"] != "0xdeadbeef" {
		t.Errorf("tx_hash: got %v, want 0xdeadbeef", result["tx_hash"])
	}
}

// TestX402StreamHandler_UnknownMsgType checks that an unknown message type gets
// an x402.error response.
func TestX402StreamHandler_UnknownMsgType(t *testing.T) {
	serverHost := newTestHost(t)
	clientHost := newTestHost(t)
	connectHosts(t, clientHost, serverHost)

	_ = NewX402StreamHandler(serverHost)
	clientHandler := NewX402StreamHandler(clientHost)

	respType, _, err := clientHandler.SendX402Message(context.Background(), serverHost.ID(), "x402.request", []byte("{}"))
	if err != nil {
		t.Fatalf("SendX402Message: %v", err)
	}
	if respType != "x402.error" {
		t.Errorf("expected x402.error for unregistered handler, got %q", respType)
	}
}

// TestX402FrameRoundTrip verifies writeX402Frame / readX402Frame symmetry using a pipe.
func TestX402FrameRoundTrip(t *testing.T) {
	cases := []struct {
		msgType string
		data    []byte
	}{
		{"x402.request", []byte(`{"hello":"world"}`)},
		{"x402.payment_required", []byte{}},
		{"x402.error", []byte(`{"error_code":1000}`)},
	}

	for _, tc := range cases {
		t.Run(tc.msgType, func(t *testing.T) {
			r, w := newPipeConn()

			done := make(chan error, 1)
			go func() {
				done <- writeX402Frame(w, tc.msgType, tc.data)
			}()

			gotType, gotData, err := readX402Frame(r)
			if readErr := <-done; readErr != nil {
				t.Fatalf("writeX402Frame: %v", readErr)
			}
			if err != nil {
				t.Fatalf("readX402Frame: %v", err)
			}
			if gotType != tc.msgType {
				t.Errorf("type: got %q, want %q", gotType, tc.msgType)
			}
			if string(gotData) != string(tc.data) {
				t.Errorf("data: got %q, want %q", gotData, tc.data)
			}
		})
	}
}

// --- helpers ---

type pipeConn struct {
	r *pipeReader
	w *pipeWriter
}

type syncPipe struct {
	buf  []byte
	done bool
	ch   chan struct{}
}

func newPipeConn() (*pipeReader, *pipeWriter) {
	ch := make(chan []byte, 128)
	return &pipeReader{ch: ch}, &pipeWriter{ch: ch}
}

type pipeReader struct{ ch chan []byte; buf []byte }
type pipeWriter struct{ ch chan []byte }

func (pw *pipeWriter) Write(p []byte) (int, error) {
	cp := make([]byte, len(p))
	copy(cp, p)
	pw.ch <- cp
	return len(p), nil
}

func (pr *pipeReader) Read(p []byte) (int, error) {
	for len(pr.buf) < len(p) {
		chunk, ok := <-pr.ch
		if !ok {
			return 0, context.Canceled
		}
		pr.buf = append(pr.buf, chunk...)
	}
	n := copy(p, pr.buf)
	pr.buf = pr.buf[n:]
	return n, nil
}
