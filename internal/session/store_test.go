package session_test

import (
	"context"
	"testing"
	"time"

	"github.com/asabya/betar/internal/session"
	"github.com/asabya/betar/pkg/types"
)

func TestStore_AddAndGet(t *testing.T) {
	dir := t.TempDir()
	store, err := session.NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	agentID := "agent-1"
	callerID := "caller-1"

	ex := types.Exchange{
		RequestID: "req-1",
		Input:     "hello",
		Output:    "world",
		Timestamp: time.Now().UTC(),
	}

	if err := store.AddExchange(ctx, agentID, callerID, ex); err != nil {
		t.Fatalf("AddExchange: %v", err)
	}

	sess, err := store.Get(ctx, agentID, callerID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(sess.Exchanges) != 1 {
		t.Fatalf("expected 1 exchange, got %d", len(sess.Exchanges))
	}
	if sess.Exchanges[0].Input != "hello" {
		t.Errorf("expected input 'hello', got %q", sess.Exchanges[0].Input)
	}
}

func TestStore_ListByAgent(t *testing.T) {
	dir := t.TempDir()
	store, err := session.NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	agentID := "agent-2"

	for _, callerID := range []string{"caller-a", "caller-b"} {
		ex := types.Exchange{RequestID: callerID + "-req", Input: "in", Output: "out", Timestamp: time.Now().UTC()}
		if err := store.AddExchange(ctx, agentID, callerID, ex); err != nil {
			t.Fatalf("AddExchange for %s: %v", callerID, err)
		}
	}

	sessions, err := store.ListByAgent(ctx, agentID)
	if err != nil {
		t.Fatalf("ListByAgent: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestStore_WithPayment(t *testing.T) {
	dir := t.TempDir()
	store, err := session.NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	ex := types.Exchange{
		RequestID: "req-paid",
		Input:     "task",
		Output:    "result",
		Timestamp: time.Now().UTC(),
		Payment: &types.PaymentRecord{
			PaymentID: "pay-1",
			TxHash:    "0xabc",
			Amount:    "1000000",
			Payer:     "0xDEAD",
			PaidAt:    time.Now().UTC(),
		},
	}

	if err := store.AddExchange(ctx, "agent-3", "buyer-1", ex); err != nil {
		t.Fatalf("AddExchange: %v", err)
	}

	sess, err := store.Get(ctx, "agent-3", "buyer-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if sess.Exchanges[0].Payment == nil {
		t.Fatal("expected payment record, got nil")
	}
	if sess.Exchanges[0].Payment.TxHash != "0xabc" {
		t.Errorf("unexpected txHash: %s", sess.Exchanges[0].Payment.TxHash)
	}
}
