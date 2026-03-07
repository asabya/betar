package p2p

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// X402ProtocolID is the libp2p protocol identifier for the x402-over-libp2p standard.
const X402ProtocolID = "/x402/libp2p/1.0.0"

// X402MessageHandler handles typed x402 messages.
// It returns (responseType, responseBytes, error). The error causes an x402.error to be sent;
// normal typed responses set responseType to the desired message type (e.g., x402.response).
type X402MessageHandler func(ctx context.Context, from peer.ID, msgType string, data []byte) (string, []byte, error)

// X402StreamHandler manages streams using the /x402/libp2p/1.0.0 protocol.
// Unlike the existing StreamHandler, both requests and responses carry a type field,
// enabling a single stream to exchange multiple distinct message types.
type X402StreamHandler struct {
	host     host.Host
	handlers map[string]X402MessageHandler
	mu       sync.RWMutex
}

// NewX402StreamHandler creates and registers the x402 stream handler on the given host.
func NewX402StreamHandler(h host.Host) *X402StreamHandler {
	s := &X402StreamHandler{
		host:     h,
		handlers: make(map[string]X402MessageHandler),
	}
	h.SetStreamHandler(X402ProtocolID, s.handleStream)
	return s
}

// RegisterHandler registers a handler for the given message type.
func (s *X402StreamHandler) RegisterHandler(msgType string, fn X402MessageHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[msgType] = fn
}

// SendX402Message opens a new stream to the target peer, writes a typed frame, reads the
// typed response frame, and returns (responseType, responseData, error).
func (s *X402StreamHandler) SendX402Message(ctx context.Context, to peer.ID, msgType string, payload []byte) (string, []byte, error) {
	if msgType == "" {
		return "", nil, fmt.Errorf("msgType is required")
	}
	if len(msgType) > maxMessageTypeLen {
		return "", nil, fmt.Errorf("msgType too long")
	}
	if len(payload) > maxMessageDataLen {
		return "", nil, fmt.Errorf("payload too large")
	}

	stream, err := s.host.NewStream(ctx, to, X402ProtocolID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer stream.Close()

	stream.SetDeadline(time.Now().Add(30 * time.Second))

	if err := writeX402Frame(stream, msgType, payload); err != nil {
		return "", nil, fmt.Errorf("failed to write frame: %w", err)
	}

	respType, respData, err := readX402Frame(stream)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read response frame: %w", err)
	}

	return respType, respData, nil
}

// handleStream is the libp2p stream handler for incoming /x402/libp2p/1.0.0 streams.
func (s *X402StreamHandler) handleStream(stream network.Stream) {
	go func() {
		defer stream.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		msgType, data, err := readX402Frame(stream)
		if err != nil {
			return
		}

		s.mu.RLock()
		handler, ok := s.handlers[msgType]
		s.mu.RUnlock()

		var respType string
		var respData []byte

		if !ok {
			// Unknown message type — write x402.error back.
			errPayload := marshalError("", fmt.Sprintf("unknown message type: %s", msgType))
			_ = writeX402Frame(stream, "x402.error", errPayload)
			return
		}

		respType, respData, err = handler(ctx, stream.Conn().RemotePeer(), msgType, data)
		if err != nil {
			errPayload := marshalError("", err.Error())
			_ = writeX402Frame(stream, "x402.error", errPayload)
			return
		}

		if err := writeX402Frame(stream, respType, respData); err != nil {
			return
		}
	}()
}

// writeX402Frame writes a typed frame to w:
//
//	[type_len : uint16 BE][type : UTF-8][data_len : uint32 BE][data]
func writeX402Frame(w io.Writer, msgType string, data []byte) error {
	typeBytes := []byte(msgType)
	if len(typeBytes) > maxMessageTypeLen {
		return fmt.Errorf("message type too long: %d > %d", len(typeBytes), maxMessageTypeLen)
	}
	if len(data) > maxMessageDataLen {
		return fmt.Errorf("data too large: %d > %d", len(data), maxMessageDataLen)
	}

	if err := binary.Write(w, binary.BigEndian, uint16(len(typeBytes))); err != nil {
		return err
	}
	if _, err := w.(io.Writer).Write(typeBytes); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}
	if len(data) > 0 {
		if _, err := w.(io.Writer).Write(data); err != nil {
			return err
		}
	}
	return nil
}

// readX402Frame reads a typed frame from r and returns (msgType, data, error).
func readX402Frame(r io.Reader) (string, []byte, error) {
	var typeLen uint16
	if err := binary.Read(r, binary.BigEndian, &typeLen); err != nil {
		return "", nil, fmt.Errorf("failed to read type length: %w", err)
	}
	if typeLen == 0 || typeLen > maxMessageTypeLen {
		return "", nil, fmt.Errorf("invalid type length: %d", typeLen)
	}

	typeBytes := make([]byte, typeLen)
	if _, err := io.ReadFull(r, typeBytes); err != nil {
		return "", nil, fmt.Errorf("failed to read type: %w", err)
	}

	var dataLen uint32
	if err := binary.Read(r, binary.BigEndian, &dataLen); err != nil {
		return "", nil, fmt.Errorf("failed to read data length: %w", err)
	}
	if dataLen > maxMessageDataLen {
		return "", nil, fmt.Errorf("data too large: %d", dataLen)
	}

	data := make([]byte, dataLen)
	if dataLen > 0 {
		if _, err := io.ReadFull(r, data); err != nil {
			return "", nil, fmt.Errorf("failed to read data: %w", err)
		}
	}

	return string(typeBytes), data, nil
}

// marshalError produces a JSON x402.error body for unexpected internal errors.
func marshalError(correlationID, message string) []byte {
	e := struct {
		Version       string `json:"version"`
		CorrelationID string `json:"correlation_id"`
		ErrorCode     int    `json:"error_code"`
		ErrorName     string `json:"error_name"`
		Message       string `json:"message"`
		Retryable     bool   `json:"retryable"`
	}{
		Version:       "1.0",
		CorrelationID: correlationID,
		ErrorCode:     1000,
		ErrorName:     "INVALID_MESSAGE",
		Message:       message,
		Retryable:     false,
	}
	data, _ := json.Marshal(e)
	return data
}
