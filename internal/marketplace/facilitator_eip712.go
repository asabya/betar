package marketplace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	x402v2 "github.com/mark3labs/x402-go/v2"
)

// EIP712Facilitator delegates to facilitator.x402.rs for EIP-712 ECDSA payments.
type EIP712Facilitator struct {
	facilitatorURL string
	network        string
}

// NewEIP712Facilitator creates a new EIP-712 facilitator that delegates to the given URL.
func NewEIP712Facilitator(facilitatorURL, network string) *EIP712Facilitator {
	return &EIP712Facilitator{facilitatorURL: facilitatorURL, network: network}
}

func (f *EIP712Facilitator) Scheme() string { return "exact" }

func (f *EIP712Facilitator) Verify(ctx context.Context, payload *FacilitatorPayload) (*FacilitatorVerifyResult, error) {
	evmPayload, ok := payload.PaymentPayload.(*EVMPayload)
	if !ok {
		return &FacilitatorVerifyResult{IsValid: false, InvalidReason: "payload is not EVMPayload"}, nil
	}

	reqBody := FacilitatorVerifyRequest{
		X402Version: x402v2.X402Version,
		PaymentPayload: x402v2.PaymentPayload{
			X402Version: x402v2.X402Version,
			Accepted:    payload.PaymentRequirements,
			Payload:     *evmPayload,
		},
		PaymentRequirements: payload.PaymentRequirements,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal verify request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		f.facilitatorURL+"/verify", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("facilitator request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("facilitator returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var verifyResp FacilitatorVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return nil, fmt.Errorf("failed to decode facilitator response: %w", err)
	}

	return &FacilitatorVerifyResult{
		IsValid:       verifyResp.IsValid,
		InvalidReason: verifyResp.InvalidMessage,
		Payer:         verifyResp.Payer,
	}, nil
}

func (f *EIP712Facilitator) Settle(ctx context.Context, payload *FacilitatorPayload) (*FacilitatorSettleResult, error) {
	evmPayload, ok := payload.PaymentPayload.(*EVMPayload)
	if !ok {
		return &FacilitatorSettleResult{Success: false, ErrorReason: "payload is not EVMPayload"}, nil
	}

	reqBody := FacilitatorSettleRequest{
		X402Version: x402v2.X402Version,
		PaymentPayload: x402v2.PaymentPayload{
			X402Version: x402v2.X402Version,
			Accepted:    payload.PaymentRequirements,
			Payload:     *evmPayload,
		},
		PaymentRequirements: payload.PaymentRequirements,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settle request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		f.facilitatorURL+"/settle", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("facilitator settle request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("facilitator returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var settleResp FacilitatorSettleResponse
	if err := json.Unmarshal(respBody, &settleResp); err != nil {
		return nil, fmt.Errorf("failed to decode facilitator response: %w", err)
	}

	return &FacilitatorSettleResult{
		Success:     settleResp.Success,
		TxHash:      settleResp.Transaction,
		ErrorReason: settleResp.InvalidReason,
	}, nil
}
