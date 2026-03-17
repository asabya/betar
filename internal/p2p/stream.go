package p2p

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-msgio"
)

// ProtocolID is the marketplace protocol identifier
const ProtocolID = "/betar/marketplace/1.0.0"

const (
	maxMessageTypeLen = 128
	maxMessageDataLen = 8 * 1024 * 1024
)

// StreamHandler handles direct P2P streams
type StreamHandler struct {
	host     host.Host
	handlers map[string]MessageHandler
	mu       sync.RWMutex
}

// MessageHandler handles incoming messages
type MessageHandler func(ctx context.Context, from peer.ID, data []byte) ([]byte, error)

// NewStreamHandler creates a new stream handler
func NewStreamHandler(h host.Host) *StreamHandler {
	s := &StreamHandler{
		host:     h,
		handlers: make(map[string]MessageHandler),
	}

	// Set stream handler for marketplace protocol
	h.SetStreamHandler(ProtocolID, s.handleStream)

	return s
}

// RegisterHandler registers a message handler
func (s *StreamHandler) RegisterHandler(msgType string, handler MessageHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[msgType] = handler
}

// SendMessage sends a message to a peer via stream
func (s *StreamHandler) SendMessage(ctx context.Context, to peer.ID, msgType string, data []byte) ([]byte, error) {
	if msgType == "" {
		return nil, fmt.Errorf("msgType is required")
	}
	if len(msgType) > maxMessageTypeLen {
		return nil, fmt.Errorf("msgType too long")
	}
	if len(data) > maxMessageDataLen {
		return nil, fmt.Errorf("message payload too large")
	}

	// Create new stream
	stream, err := s.host.NewStream(ctx, to, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()

	// Set read/write deadlines
	stream.SetDeadline(time.Now().Add(30 * time.Second))

	// Write message type
	msgLen := len(msgType)
	if err := binary.Write(stream, binary.BigEndian, uint16(msgLen)); err != nil {
		return nil, fmt.Errorf("failed to write msg type length: %w", err)
	}
	if _, err := stream.Write([]byte(msgType)); err != nil {
		return nil, fmt.Errorf("failed to write msg type: %w", err)
	}

	// Write message data
	if err := binary.Write(stream, binary.BigEndian, uint32(len(data))); err != nil {
		return nil, fmt.Errorf("failed to write data length: %w", err)
	}
	if _, err := stream.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write data: %w", err)
	}

	// Read response
	var respLen uint32
	if err := binary.Read(stream, binary.BigEndian, &respLen); err != nil {
		return nil, fmt.Errorf("failed to read response length: %w", err)
	}
	if respLen > maxMessageDataLen {
		return nil, fmt.Errorf("response payload too large")
	}

	respData := make([]byte, respLen)
	if _, err := io.ReadFull(stream, respData); err != nil {
		return nil, fmt.Errorf("failed to read response data: %w", err)
	}

	return respData, nil
}

// handleStream handles incoming streams
func (s *StreamHandler) handleStream(stream network.Stream) {
	go func() {
		defer stream.Close()

		stream.SetDeadline(time.Now().Add(30 * time.Second))

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Read message type
		var msgTypeLen uint16
		if err := binary.Read(stream, binary.BigEndian, &msgTypeLen); err != nil {
			return
		}
		if msgTypeLen == 0 || msgTypeLen > maxMessageTypeLen {
			return
		}

		msgTypeBytes := make([]byte, msgTypeLen)
		if _, err := io.ReadFull(stream, msgTypeBytes); err != nil {
			return
		}
		msgType := string(msgTypeBytes)

		// Read message data
		var dataLen uint32
		if err := binary.Read(stream, binary.BigEndian, &dataLen); err != nil {
			return
		}
		if dataLen > maxMessageDataLen {
			return
		}

		data := make([]byte, dataLen)
		if _, err := io.ReadFull(stream, data); err != nil {
			return
		}

		// Find handler
		s.mu.RLock()
		handler, ok := s.handlers[msgType]
		s.mu.RUnlock()

		var respData []byte
		var respErr error

		if ok {
			respData, respErr = handler(ctx, stream.Conn().RemotePeer(), data)
		} else {
			respErr = fmt.Errorf("unknown message type: %s", msgType)
		}

		// Write response
		if respErr != nil {
			respData = []byte(respErr.Error())
		}

		var respLen = uint32(len(respData))
		if err := binary.Write(stream, binary.BigEndian, respLen); err != nil {
			return
		}
		if _, err := stream.Write(respData); err != nil {
			return
		}
	}()
}

// ReadMessage reads a message from a stream using msgio
func ReadMessage(r io.Reader) (string, []byte, error) {
	mr := msgio.NewReader(r)
	msg, err := mr.ReadMsg()
	if err != nil {
		return "", nil, err
	}
	defer mr.ReleaseMsg(msg)

	// Simple protocol: type\ndata
	// Find the separator
	var msgType string
	var data []byte
	for i, b := range msg {
		if b == '\n' {
			msgType = string(msg[:i])
			data = msg[i+1:]
			break
		}
	}

	if msgType == "" {
		msgType = string(msg)
		data = nil
	}

	return msgType, data, nil
}

// WriteMessage writes a message to a stream using msgio
func WriteMessage(w io.Writer, msgType string, data []byte) error {
	mw := msgio.NewWriter(w)
	msg := append([]byte(msgType), '\n')
	msg = append(msg, data...)
	return mw.WriteMsg(msg)
}
