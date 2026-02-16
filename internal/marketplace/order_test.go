package marketplace

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
)

func TestOrderServiceLifecycleWithoutStream(t *testing.T) {
	t.Parallel()

	svc := NewOrderService(nil, nil, peer.ID("buyer-1"))

	order, err := svc.CreateOrder(context.Background(), "agent-1", "", 0.002)
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}
	if order.Status != "pending" {
		t.Fatalf("expected pending status, got %s", order.Status)
	}

	if err := svc.AcceptOrder(context.Background(), order.ID); err != nil {
		t.Fatalf("AcceptOrder failed: %v", err)
	}
	if got, _ := svc.GetOrder(order.ID); got.Status != "accepted" {
		t.Fatalf("expected accepted status, got %s", got.Status)
	}

	if err := svc.CompleteOrder(context.Background(), order.ID); err != nil {
		t.Fatalf("CompleteOrder failed: %v", err)
	}
	if got, _ := svc.GetOrder(order.ID); got.Status != "completed" {
		t.Fatalf("expected completed status, got %s", got.Status)
	}
}
