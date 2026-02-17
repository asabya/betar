package marketplace

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/x402-go"
)

// PaymentRequirement is an alias for x402.PaymentRequirement
type PaymentRequirement = x402.PaymentRequirement

// PaymentHeader is embedded in P2P messages for payment
type PaymentHeader struct {
	Requirement x402.PaymentRequirement `json:"requirement"`
	Payer       string                  `json:"payer"`
	PaymentID   string                  `json:"payment_id"`
	Signature   string                  `json:"signature,omitempty"`

	Accepted *x402.PaymentRequirement `json:"accepted,omitempty"`
	Payload  *x402.EVMPayload         `json:"payload,omitempty"`
}

// TaskExecuteRequest is sent for task execution with payment
type TaskExecuteRequest struct {
	AgentID         string         `json:"agent_id"`
	Input           string         `json:"input"`
	PaymentHeader   *PaymentHeader `json:"payment_header,omitempty"`
	TransactionHash string         `json:"transaction_hash,omitempty"`
}

// TaskExecuteResponse is returned after task execution
type TaskExecuteResponse struct {
	RequestID       string `json:"request_id"`
	Output          string `json:"output,omitempty"`
	Error           string `json:"error,omitempty"`
	TransactionHash string `json:"transaction_hash,omitempty"`
}

// PaymentRequiredResponse is returned when a task requires payment before execution
type PaymentRequiredResponse struct {
	AgentID            string              `json:"agent_id"`
	RequestID          string              `json:"request_id"`
	Message            string              `json:"message"`
	PaymentRequirement *PaymentRequirement `json:"payment_requirement,omitempty"`
	RequiresPayment    bool                `json:"requires_payment"`
}

// PaymentHeaderFromJSON deserializes payment header from JSON
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
			"name":    "USDC",
			"version": "2",
		},
	}
}

const (
	NetworkEthSepolia  = "sepolia"
	NetworkEthMainnet  = "eip155:1"
	NetworkBaseSepolia = "base-sepolia"
	NetworkBaseMainnet = "eip155:8453"

	USDCETHSepolia  = "0x1c7d4b196cb0c7b01d743fbc6116a902379c7238"
	USDCETHMainnet  = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	USDCBaseSepolia = "0x036CbD53842c5426634e7929541eC2318f3dCF7e"
	USDCBaseMainnet = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
)

func GetUSDCAddress(network string) string {
	switch network {
	case NetworkEthSepolia:
		return USDCETHSepolia
	case NetworkEthMainnet:
		return USDCETHMainnet
	case NetworkBaseSepolia:
		return USDCBaseSepolia
	case NetworkBaseMainnet:
		return USDCBaseMainnet
	default:
		return USDCBaseSepolia
	}
}

func GetNetworkName(network string) string {
	switch network {
	case NetworkEthSepolia:
		return "ETH Sepolia"
	case NetworkEthMainnet:
		return "ETH Mainnet"
	case NetworkBaseSepolia:
		return "Base Sepolia"
	case NetworkBaseMainnet:
		return "Base Mainnet"
	default:
		return "Unknown"
	}
}
