package marketplace

import (
	"encoding/json"
	"testing"
	"time"
)

// --- NewX402Error tests ---

func TestNewX402Error_RetryableFlags(t *testing.T) {
	cases := []struct {
		code      X402ErrorCode
		name      string
		retryable bool
	}{
		{ErrInvalidMessage, "INVALID_MESSAGE", false},
		{ErrUnknownResource, "UNKNOWN_RESOURCE", false},
		{ErrPaymentRequired, "PAYMENT_REQUIRED", true},
		{ErrPaymentInvalid, "PAYMENT_INVALID", false},
		{ErrNonceMismatch, "PAYMENT_NONCE_MISMATCH", false},
		{ErrNonceExpired, "PAYMENT_NONCE_EXPIRED", true},
		{ErrNonceUsed, "PAYMENT_NONCE_USED", false},
		{ErrAmountWrong, "PAYMENT_AMOUNT_WRONG", false},
		{ErrSettlementFailed, "SETTLEMENT_FAILED", true},
		{ErrExecutionFailed, "EXECUTION_FAILED", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := NewX402Error("corr-1", tc.code, "test message")
			if e.ErrorCode != tc.code {
				t.Errorf("ErrorCode: got %d, want %d", e.ErrorCode, tc.code)
			}
			if e.ErrorName != tc.name {
				t.Errorf("ErrorName: got %q, want %q", e.ErrorName, tc.name)
			}
			if e.Retryable != tc.retryable {
				t.Errorf("Retryable: got %v, want %v", e.Retryable, tc.retryable)
			}
			if e.Version != X402LibP2PVersion {
				t.Errorf("Version: got %q, want %q", e.Version, X402LibP2PVersion)
			}
			if e.CorrelationID != "corr-1" {
				t.Errorf("CorrelationID: got %q, want corr-1", e.CorrelationID)
			}
		})
	}
}

func TestNewX402Error_UnknownCode(t *testing.T) {
	e := NewX402Error("", X402ErrorCode(9999), "oops")
	if e.ErrorName != "UNKNOWN_ERROR" {
		t.Errorf("expected UNKNOWN_ERROR, got %s", e.ErrorName)
	}
	if e.Retryable {
		t.Error("expected unknown error to be non-retryable")
	}
}

// --- JSON round-trip tests ---

func TestX402Request_JSONRoundTrip(t *testing.T) {
	original := X402Request{
		Version:       X402LibP2PVersion,
		CorrelationID: "test-corr",
		Resource:      "agent-123",
		Method:        "execute",
		Payment:       nil,
		Body:          []byte(`{"input":"hello"}`),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded X402Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Version != original.Version {
		t.Errorf("Version: got %q, want %q", decoded.Version, original.Version)
	}
	if decoded.CorrelationID != original.CorrelationID {
		t.Errorf("CorrelationID: got %q, want %q", decoded.CorrelationID, original.CorrelationID)
	}
	if decoded.Resource != original.Resource {
		t.Errorf("Resource: got %q, want %q", decoded.Resource, original.Resource)
	}
	if decoded.Payment != nil {
		t.Errorf("Payment: expected nil, got %+v", decoded.Payment)
	}
}

func TestX402PaymentRequired_JSONRoundTrip(t *testing.T) {
	pr := X402PaymentRequired{
		Version:            X402LibP2PVersion,
		CorrelationID:      "corr-42",
		ChallengeNonce:     "abc123",
		ChallengeExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
		PaymentRequirements: &PaymentRequirements{
			Scheme:  "exact",
			Network: NetworkBaseSepolia,
			Amount:  "1000",
			Asset:   USDCBaseSepolia,
			PayTo:   "0xSeller",
		},
		Message: "Payment required",
	}

	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded X402PaymentRequired
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.ChallengeNonce != pr.ChallengeNonce {
		t.Errorf("ChallengeNonce: got %q, want %q", decoded.ChallengeNonce, pr.ChallengeNonce)
	}
	if decoded.PaymentRequirements == nil {
		t.Fatal("PaymentRequirements is nil after round-trip")
	}
	if decoded.PaymentRequirements.Amount != "1000" {
		t.Errorf("Amount: got %q, want 1000", decoded.PaymentRequirements.Amount)
	}
}

func TestX402Error_JSONRoundTrip(t *testing.T) {
	e := NewX402Error("corr-99", ErrSettlementFailed, "facilitator timeout")

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded X402Error
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.ErrorCode != ErrSettlementFailed {
		t.Errorf("ErrorCode: got %d, want %d", decoded.ErrorCode, ErrSettlementFailed)
	}
	if !decoded.Retryable {
		t.Error("expected Retryable=true for SETTLEMENT_FAILED")
	}
}

// --- ChallengeStore tests ---

func newTestPaymentService() *PaymentService {
	return &PaymentService{
		challengeStore: newChallengeStore(),
	}
}

func TestChallengeStore_GenerateAndConsume(t *testing.T) {
	svc := newTestPaymentService()

	c, err := svc.GenerateChallenge("corr-1", 5*time.Minute)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}
	if len(c.Nonce) != 64 {
		t.Errorf("expected 64-char hex nonce, got len=%d", len(c.Nonce))
	}
	if c.CorrelationID != "corr-1" {
		t.Errorf("CorrelationID: got %q, want corr-1", c.CorrelationID)
	}

	consumed, err := svc.ConsumeChallenge("corr-1")
	if err != nil {
		t.Fatalf("ConsumeChallenge: %v", err)
	}
	if consumed.Nonce != c.Nonce {
		t.Errorf("nonce mismatch: got %q, want %q", consumed.Nonce, c.Nonce)
	}

	// Second consume should fail (deleted on first read).
	_, err = svc.ConsumeChallenge("corr-1")
	if err == nil {
		t.Error("expected error on double-consume, got nil")
	}
}

func TestChallengeStore_ConsumeUnknown(t *testing.T) {
	svc := newTestPaymentService()
	_, err := svc.ConsumeChallenge("does-not-exist")
	if err == nil {
		t.Error("expected error for unknown correlation ID")
	}
}

func TestChallengeStore_ConsumeExpired(t *testing.T) {
	svc := newTestPaymentService()

	// Generate with a very short TTL that has already passed.
	c, err := svc.GenerateChallenge("corr-exp", -1*time.Second)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}
	_ = c

	_, err = svc.ConsumeChallenge("corr-exp")
	if err == nil {
		t.Error("expected error for expired challenge")
	}
}

func TestChallengeStore_CleanupExpired(t *testing.T) {
	svc := newTestPaymentService()

	// One live challenge, one expired.
	_, _ = svc.GenerateChallenge("live", 5*time.Minute)
	_, _ = svc.GenerateChallenge("expired", -1*time.Second)

	svc.CleanupExpiredChallenges()

	// The live one should still be consumable.
	_, err := svc.ConsumeChallenge("live")
	if err != nil {
		t.Errorf("expected live challenge to survive cleanup: %v", err)
	}

	// The expired one should have been removed.
	_, err = svc.ConsumeChallenge("expired")
	if err == nil {
		t.Error("expected expired challenge to be cleaned up")
	}
}
