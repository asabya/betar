package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/pkg/types"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
)

type orderRPCResponse struct {
	Order *types.Order `json:"order,omitempty"`
	Error string       `json:"error,omitempty"`
}

// OrderService handles order management.
type OrderService struct {
	streamHandler *p2p.StreamHandler
	host          *p2p.Host
	peerID        peer.ID

	mu      sync.RWMutex
	orders  map[string]*types.Order
	pending map[string]chan *types.Order
}

// NewOrderService creates a new order service.
func NewOrderService(streamHandler *p2p.StreamHandler, host *p2p.Host, peerID peer.ID) *OrderService {
	s := &OrderService{
		streamHandler: streamHandler,
		host:          host,
		peerID:        peerID,
		orders:        make(map[string]*types.Order),
		pending:       make(map[string]chan *types.Order),
	}

	if streamHandler != nil {
		streamHandler.RegisterHandler(OrderCreateMessage, s.handleOrderCreate)
		streamHandler.RegisterHandler(OrderAcceptMessage, s.handleOrderAccept)
		streamHandler.RegisterHandler(OrderCompleteMessage, s.handleOrderComplete)
		streamHandler.RegisterHandler(OrderCancelMessage, s.handleOrderCancel)
	}

	return s
}

// CreateOrder creates a new order.
func (s *OrderService) CreateOrder(ctx context.Context, agentID, sellerID string, price float64) (*types.Order, error) {
	orderID := uuid.New().String()
	now := time.Now().Unix()

	order := &types.Order{
		ID:        orderID,
		AgentID:   agentID,
		BuyerID:   s.peerID.String(),
		SellerID:  sellerID,
		Price:     price,
		Status:    "pending",
		Timestamp: now,
	}

	s.mu.Lock()
	s.orders[orderID] = order
	s.mu.Unlock()

	if sellerID == "" || sellerID == s.peerID.String() {
		return order, nil
	}

	msg := &types.OrderMessage{
		Type:      "new",
		OrderID:   orderID,
		AgentID:   agentID,
		BuyerID:   s.peerID.String(),
		SellerID:  sellerID,
		Price:     price,
		Status:    "pending",
		Timestamp: now,
	}

	remoteOrder, err := s.sendOrderMessage(ctx, sellerID, OrderCreateMessage, msg)
	if err != nil {
		return nil, err
	}

	if remoteOrder != nil {
		s.mu.Lock()
		s.orders[orderID] = remoteOrder
		s.mu.Unlock()
		return remoteOrder, nil
	}

	return order, nil
}

// AcceptOrder accepts an order (seller side).
func (s *OrderService) AcceptOrder(ctx context.Context, orderID string) error {
	order, err := s.updateOrderStatus(orderID, "accepted")
	if err != nil {
		return err
	}
	return s.notifyCounterparty(ctx, order, OrderAcceptMessage, "accept")
}

// CompleteOrder completes an order.
func (s *OrderService) CompleteOrder(ctx context.Context, orderID string) error {
	order, err := s.updateOrderStatus(orderID, "completed")
	if err != nil {
		return err
	}
	return s.notifyCounterparty(ctx, order, OrderCompleteMessage, "complete")
}

// CancelOrder cancels an order.
func (s *OrderService) CancelOrder(ctx context.Context, orderID string) error {
	order, err := s.updateOrderStatus(orderID, "cancelled")
	if err != nil {
		return err
	}
	return s.notifyCounterparty(ctx, order, OrderCancelMessage, "cancel")
}

func (s *OrderService) notifyCounterparty(ctx context.Context, order *types.Order, msgType, eventType string) error {
	if order == nil {
		return fmt.Errorf("order is nil")
	}

	target := order.BuyerID
	if order.BuyerID == s.peerID.String() {
		target = order.SellerID
	}

	if target == "" || target == s.peerID.String() {
		return nil
	}

	msg := &types.OrderMessage{
		Type:      eventType,
		OrderID:   order.ID,
		AgentID:   order.AgentID,
		BuyerID:   order.BuyerID,
		SellerID:  order.SellerID,
		Price:     order.Price,
		Status:    order.Status,
		Timestamp: order.Timestamp,
	}

	_, err := s.sendOrderMessage(ctx, target, msgType, msg)
	return err
}

func (s *OrderService) sendOrderMessage(ctx context.Context, targetPeer, streamMessage string, msg *types.OrderMessage) (*types.Order, error) {
	if s.streamHandler == nil {
		return nil, fmt.Errorf("stream handler not configured")
	}
	if s.host == nil {
		return nil, fmt.Errorf("host not configured")
	}

	peerID, err := peer.Decode(targetPeer)
	if err != nil {
		return nil, fmt.Errorf("invalid peer ID %q: %w", targetPeer, err)
	}

	if err := s.host.Connect(ctx, peer.AddrInfo{ID: peerID}); err != nil {
		return nil, fmt.Errorf("failed to connect peer %s: %w", targetPeer, err)
	}

	payload, err := types.ToJSON(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode order message: %w", err)
	}

	respData, err := s.streamHandler.SendMessage(ctx, peerID, streamMessage, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to send order message: %w", err)
	}

	var resp orderRPCResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode order response: %w", err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("remote error: %s", resp.Error)
	}

	return resp.Order, nil
}

func (s *OrderService) handleOrderCreate(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
	return s.handleRemoteOrderMutation(ctx, from, data, "pending")
}

func (s *OrderService) handleOrderAccept(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
	return s.handleRemoteOrderMutation(ctx, from, data, "accepted")
}

func (s *OrderService) handleOrderComplete(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
	return s.handleRemoteOrderMutation(ctx, from, data, "completed")
}

func (s *OrderService) handleOrderCancel(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
	return s.handleRemoteOrderMutation(ctx, from, data, "cancelled")
}

func (s *OrderService) handleRemoteOrderMutation(ctx context.Context, from peer.ID, data []byte, expectedStatus string) ([]byte, error) {
	_ = ctx

	var msg types.OrderMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return encodeOrderRPCResponse(orderRPCResponse{Error: fmt.Sprintf("invalid request: %v", err)})
	}
	if msg.OrderID == "" {
		return encodeOrderRPCResponse(orderRPCResponse{Error: "order ID is required"})
	}

	if msg.BuyerID == "" {
		msg.BuyerID = from.String()
	}
	if msg.SellerID == "" {
		msg.SellerID = s.peerID.String()
	}
	if msg.Status == "" {
		msg.Status = expectedStatus
	}
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().Unix()
	}

	order := &types.Order{
		ID:        msg.OrderID,
		AgentID:   msg.AgentID,
		BuyerID:   msg.BuyerID,
		SellerID:  msg.SellerID,
		Price:     msg.Price,
		Status:    msg.Status,
		Timestamp: msg.Timestamp,
	}

	s.mu.Lock()
	s.orders[order.ID] = order
	if ch, ok := s.pending[order.ID]; ok {
		select {
		case ch <- order:
		default:
		}
	}
	s.mu.Unlock()

	return encodeOrderRPCResponse(orderRPCResponse{Order: order})
}

func encodeOrderRPCResponse(resp orderRPCResponse) ([]byte, error) {
	data, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *OrderService) updateOrderStatus(orderID, status string) (*types.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	order, ok := s.orders[orderID]
	if !ok {
		return nil, fmt.Errorf("order not found")
	}

	order.Status = status
	order.Timestamp = time.Now().Unix()
	return order, nil
}

// GetOrder gets an order by ID.
func (s *OrderService) GetOrder(orderID string) (*types.Order, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order, ok := s.orders[orderID]
	return order, ok
}

// ListOrders lists all orders.
func (s *OrderService) ListOrders() []*types.Order {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orders := make([]*types.Order, 0, len(s.orders))
	for _, o := range s.orders {
		orders = append(orders, o)
	}

	return orders
}

// WaitForOrder waits for an order to be updated.
func (s *OrderService) WaitForOrder(ctx context.Context, orderID string, timeout time.Duration) (*types.Order, error) {
	ch := make(chan *types.Order, 1)

	s.mu.Lock()
	if order, ok := s.orders[orderID]; ok {
		ch <- order
		s.mu.Unlock()
		return order, nil
	}
	s.pending[orderID] = ch
	s.mu.Unlock()

	select {
	case order := <-ch:
		s.mu.Lock()
		delete(s.pending, orderID)
		s.mu.Unlock()
		return order, nil
	case <-ctx.Done():
		s.mu.Lock()
		delete(s.pending, orderID)
		s.mu.Unlock()
		return nil, ctx.Err()
	case <-time.After(timeout):
		s.mu.Lock()
		delete(s.pending, orderID)
		s.mu.Unlock()
		return nil, fmt.Errorf("timeout waiting for order")
	}
}
