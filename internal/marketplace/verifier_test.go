package marketplace

import (
	"fmt"
	"math/big"
	"testing"
	"time"
)

func testPaymentHeader() *PaymentHeader {
	return &PaymentHeader{
		Payer: "0x1111111111111111111111111111111111111111",
		Requirement: PaymentRequirements{
			Scheme:  "exact",
			Network: NetworkBaseSepolia,
			Amount:  "1000000",
			PayTo:   "0x2222222222222222222222222222222222222222",
			Asset:   USDCBaseSepolia,
		},
		Signature: "0xsig",
		Payload: &EVMPayload{
			Authorization: EVMAuthorization{
				From:        "0x1111111111111111111111111111111111111111",
				To:          "0x2222222222222222222222222222222222222222",
				Value:       "1000000",
				ValidAfter:  fmt.Sprintf("%d", time.Now().Unix()-10),
				ValidBefore: fmt.Sprintf("%d", time.Now().Unix()+60),
				Nonce:       "0x0000000000000000000000000000000000000000000000000000000000000001",
			},
			Signature: "0xsig",
		},
	}
}

func TestValidateTimestamps(t *testing.T) {
	v := NewPaymentVerifier(NetworkBaseSepolia)

	t.Run("valid window", func(t *testing.T) {
		auth := &EVMAuthorization{
			ValidAfter:  fmt.Sprintf("%d", time.Now().Unix()-10),
			ValidBefore: fmt.Sprintf("%d", time.Now().Unix()+60),
		}
		if err := v.ValidateTimestamps(auth); err != nil {
			t.Errorf("expected pass, got error: %v", err)
		}
	})

	t.Run("expired", func(t *testing.T) {
		auth := &EVMAuthorization{
			ValidAfter:  fmt.Sprintf("%d", time.Now().Unix()-120),
			ValidBefore: fmt.Sprintf("%d", time.Now().Unix()-60),
		}
		if err := v.ValidateTimestamps(auth); err == nil {
			t.Error("expected error for expired authorization")
		}
	})

	t.Run("not yet valid", func(t *testing.T) {
		auth := &EVMAuthorization{
			ValidAfter:  fmt.Sprintf("%d", time.Now().Unix()+10),
			ValidBefore: fmt.Sprintf("%d", time.Now().Unix()+60),
		}
		if err := v.ValidateTimestamps(auth); err == nil {
			t.Error("expected error for not yet valid authorization")
		}
	})
}

func TestNonceTracking(t *testing.T) {
	v := NewPaymentVerifier(NetworkBaseSepolia)

	t.Run("first use succeeds", func(t *testing.T) {
		if err := v.CheckAndMarkNonce("nonce-1"); err != nil {
			t.Errorf("expected pass, got error: %v", err)
		}
	})

	t.Run("second use fails", func(t *testing.T) {
		_ = v.CheckAndMarkNonce("nonce-2")
		if err := v.CheckAndMarkNonce("nonce-2"); err == nil {
			t.Error("expected error for reused nonce")
		}
	})

	t.Run("different nonces both succeed", func(t *testing.T) {
		if err := v.CheckAndMarkNonce("nonce-3"); err != nil {
			t.Errorf("expected pass for nonce-3, got error: %v", err)
		}
		if err := v.CheckAndMarkNonce("nonce-4"); err != nil {
			t.Errorf("expected pass for nonce-4, got error: %v", err)
		}
	})
}

func TestValidatePayerConsistency(t *testing.T) {
	v := NewPaymentVerifier(NetworkBaseSepolia)

	t.Run("matching payer", func(t *testing.T) {
		header := testPaymentHeader()
		if err := v.ValidatePayerConsistency(header); err != nil {
			t.Errorf("expected pass, got error: %v", err)
		}
	})

	t.Run("mismatched payer", func(t *testing.T) {
		header := testPaymentHeader()
		header.Payer = "0x3333333333333333333333333333333333333333"
		if err := v.ValidatePayerConsistency(header); err == nil {
			t.Error("expected error for mismatched payer")
		}
	})

	t.Run("nil header", func(t *testing.T) {
		if err := v.ValidatePayerConsistency(nil); err == nil {
			t.Error("expected error for nil header")
		}
	})
}

func TestValidatePaymentHeader(t *testing.T) {
	v := NewPaymentVerifier(NetworkBaseSepolia)
	expectedPayTo := "0x2222222222222222222222222222222222222222"
	expectedAmount := big.NewInt(1000000)

	t.Run("valid header", func(t *testing.T) {
		header := testPaymentHeader()
		if err := v.ValidatePaymentHeader(header, expectedPayTo, expectedAmount); err != nil {
			t.Errorf("expected pass, got error: %v", err)
		}
	})

	t.Run("wrong scheme", func(t *testing.T) {
		header := testPaymentHeader()
		header.Requirement.Scheme = "invalid"
		if err := v.ValidatePaymentHeader(header, expectedPayTo, expectedAmount); err == nil {
			t.Error("expected error for wrong scheme")
		}
	})

	t.Run("wrong network", func(t *testing.T) {
		header := testPaymentHeader()
		header.Requirement.Network = "invalid-network"
		if err := v.ValidatePaymentHeader(header, expectedPayTo, expectedAmount); err == nil {
			t.Error("expected error for wrong network")
		}
	})

	t.Run("empty signature", func(t *testing.T) {
		header := testPaymentHeader()
		header.Signature = ""
		if err := v.ValidatePaymentHeader(header, expectedPayTo, expectedAmount); err == nil {
			t.Error("expected error for empty signature")
		}
	})

	t.Run("wrong payTo", func(t *testing.T) {
		header := testPaymentHeader()
		header.Requirement.PayTo = "0x9999999999999999999999999999999999999999"
		if err := v.ValidatePaymentHeader(header, expectedPayTo, expectedAmount); err == nil {
			t.Error("expected error for wrong payTo")
		}
	})

	t.Run("wrong amount", func(t *testing.T) {
		header := testPaymentHeader()
		if err := v.ValidatePaymentHeader(header, expectedPayTo, big.NewInt(999)); err == nil {
			t.Error("expected error for wrong amount")
		}
	})

	t.Run("wrong asset", func(t *testing.T) {
		header := testPaymentHeader()
		header.Requirement.Asset = "0x0000000000000000000000000000000000000000"
		if err := v.ValidatePaymentHeader(header, expectedPayTo, expectedAmount); err == nil {
			t.Error("expected error for wrong asset")
		}
	})
}

func TestValidatePayment(t *testing.T) {
	expectedPayTo := "0x2222222222222222222222222222222222222222"
	expectedAmount := big.NewInt(1000000)

	t.Run("header validation passes", func(t *testing.T) {
		v := NewPaymentVerifier(NetworkBaseSepolia)
		header := testPaymentHeader()
		if err := v.ValidatePaymentHeader(header, expectedPayTo, expectedAmount); err != nil {
			t.Errorf("ValidatePaymentHeader: expected pass, got error: %v", err)
		}
		if err := v.ValidatePayerConsistency(header); err != nil {
			t.Errorf("ValidatePayerConsistency: expected pass, got error: %v", err)
		}
		if err := v.ValidateTimestamps(&header.Payload.Authorization); err != nil {
			t.Errorf("ValidateTimestamps: expected pass, got error: %v", err)
		}
		if err := v.CheckAndMarkNonce(header.Payload.Authorization.Nonce); err != nil {
			t.Errorf("CheckAndMarkNonce: expected pass, got error: %v", err)
		}
	})

	t.Run("timestamp failure", func(t *testing.T) {
		v := NewPaymentVerifier(NetworkBaseSepolia)
		header := testPaymentHeader()
		header.Payload.Authorization.ValidAfter = fmt.Sprintf("%d", time.Now().Unix()+100)
		if err := v.ValidateTimestamps(&header.Payload.Authorization); err == nil {
			t.Error("expected error for timestamp failure")
		}
	})

	t.Run("nonce failure", func(t *testing.T) {
		v := NewPaymentVerifier(NetworkBaseSepolia)
		header := testPaymentHeader()
		nonce := header.Payload.Authorization.Nonce

		if err := v.CheckAndMarkNonce(nonce); err != nil {
			t.Fatalf("first nonce check failed: %v", err)
		}

		if err := v.CheckAndMarkNonce(nonce); err == nil {
			t.Error("expected error for reused nonce")
		}
	})
}
