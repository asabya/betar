package marketplace

import (
	"encoding/json"
	"fmt"

	"github.com/asabya/betar/internal/eth"

	"github.com/mark3labs/x402-go"
)

// PaymentRequirement is an alias for x402.PaymentRequirement (for backward compatibility)
type PaymentRequirement = x402.PaymentRequirement

// PaymentHeader is embedded in P2P messages for payment
type PaymentHeader struct {
	Requirement x402.PaymentRequirement `json:"requirement"`
	Payer       string                  `json:"payer"`      // Buyer wallet address
	PaymentID   string                  `json:"payment_id"` // Unique payment ID
	Signature   string                  `json:"signature"`  // EIP-712 signature (legacy)

	// v2 fields
	Accepted x402.PaymentRequirement `json:"accepted,omitempty"` // The requirement that was accepted
	Payload  *x402.EVMPayload        `json:"payload,omitempty"`  // EIP-3009 authorization
}

// TaskExecuteRequest is sent for task execution with payment
type TaskExecuteRequest struct {
	AgentID       string         `json:"agent_id"`
	Input         string         `json:"input"`
	PaymentHeader *PaymentHeader `json:"payment_header,omitempty"`
}

// TaskExecuteResponse is returned after task execution
type TaskExecuteResponse struct {
	RequestID       string `json:"request_id"`
	Output          string `json:"output,omitempty"`
	Error           string `json:"error,omitempty"`
	TransactionHash string `json:"transaction_hash,omitempty"` // Payment tx hash
}

// PaymentRequiredResponse is returned when a task requires payment before execution
type PaymentRequiredResponse struct {
	AgentID            string              `json:"agent_id"`
	RequestID          string              `json:"request_id"`
	Message            string              `json:"message"`
	PaymentRequirement *PaymentRequirement `json:"payment_requirement,omitempty"` // Unsigned requirement from seller
	RequiresPayment    bool                `json:"requires_payment"`
}

// FromJSON deserializes payment header from JSON
func PaymentHeaderFromJSON(data []byte) (*PaymentHeader, error) {
	var ph PaymentHeader
	if err := json.Unmarshal(data, &ph); err != nil {
		return nil, fmt.Errorf("failed to parse payment header: %w", err)
	}
	return &ph, nil
}

// CreatePaymentRequirement creates a payment requirement from agent listing
func CreatePaymentRequirement(network, amount, asset, payTo string, timeout int64) x402.PaymentRequirement {
	return x402.PaymentRequirement{
		Scheme:            "exact",
		Network:           network,
		MaxAmountRequired: amount,
		Asset:             asset,
		PayTo:             payTo,
		MaxTimeoutSeconds: int(timeout),
		Extra: map[string]interface{}{
			"name":    "USD Coin",
			"version": "1",
		},
	}
}

// SignRequirement signs a payment requirement with the wallet
func SignRequirement(wallet *eth.Wallet, req *PaymentRequirement) (string, error) {
	// In production, this would create an EIP-712 signature
	// For now, create a simple signature of the payment ID
	sigData := fmt.Sprintf("%s:%s:%s:%s", req.Network, req.MaxAmountRequired, req.Asset, req.PayTo)

	// TODO: Implement proper EIP-712 signing using x402-go library
	// This is a placeholder - actual implementation will use x402-go signers
	_ = sigData
	_ = wallet

	return "mock_signature", nil
}

// VerifyPayment verifies payment header (placeholder - will integrate x402-go)
func VerifyPayment(header *PaymentHeader) error {
	if header == nil {
		return fmt.Errorf("payment header is nil")
	}

	if header.Requirement.Scheme != "exact" {
		return fmt.Errorf("unsupported payment scheme: %s", header.Requirement.Scheme)
	}

	if header.Signature == "" {
		return fmt.Errorf("payment signature missing")
	}

	// TODO: Verify signature using x402-go library
	// This will call the facilitator to verify the payment intent

	return nil
}

// Default network constants
const (
	// Base Sepolia testnet
	NetworkBaseSepolia = "eip155:84532"
	// Base mainnet
	NetworkBaseMainnet = "eip155:8453"

	// USDC addresses
	USDCBaseSepolia = "0x036CbD53842c5426634e7929541eC2318f3dCF7e"
	USDCBaseMainnet = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
)

// GetUSDCAddress returns the USDC address for a network
func GetUSDCAddress(network string) string {
	switch network {
	case NetworkBaseSepolia:
		return USDCBaseSepolia
	case NetworkBaseMainnet:
		return USDCBaseMainnet
	default:
		return USDCBaseSepolia // Default to testnet
	}
}

// GetNetworkName returns a human-readable network name
func GetNetworkName(network string) string {
	switch network {
	case NetworkBaseSepolia:
		return "Base Sepolia"
	case NetworkBaseMainnet:
		return "Base Mainnet"
	default:
		return "Unknown"
	}
}
