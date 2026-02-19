package marketplace

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	x402v2 "github.com/mark3labs/x402-go/v2"
)

// PaymentRequirements is an alias for x402 v2 PaymentRequirements
type PaymentRequirements = x402v2.PaymentRequirements

// EVMPayload is an alias for x402 v2 EVMPayload
type EVMPayload = x402v2.EVMPayload

// EVMAuthorization is an alias for x402 v2 EVMAuthorization
type EVMAuthorization = x402v2.EVMAuthorization

// PaymentHeader is embedded in P2P messages for payment
type PaymentHeader struct {
	Requirement PaymentRequirements `json:"requirement"`
	Payer       string              `json:"payer"`
	PaymentID   string              `json:"payment_id"`
	Signature   string              `json:"signature,omitempty"`

	Accepted *PaymentRequirements `json:"accepted,omitempty"`
	Payload  *EVMPayload          `json:"payload,omitempty"`
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
	AgentID            string               `json:"agent_id"`
	RequestID          string               `json:"request_id"`
	Message            string               `json:"message"`
	PaymentRequirement *PaymentRequirements `json:"payment_requirement,omitempty"`
	RequiresPayment    bool                 `json:"requires_payment"`
}

// PaymentHeaderFromJSON deserializes payment header from JSON
func PaymentHeaderFromJSON(data []byte) (*PaymentHeader, error) {
	var ph PaymentHeader
	if err := json.Unmarshal(data, &ph); err != nil {
		return nil, fmt.Errorf("failed to parse payment header: %w", err)
	}
	return &ph, nil
}

// CreatePaymentRequirements creates a payment requirement from agent listing (x402 v2 format)
func CreatePaymentRequirements(network, amount, asset, payTo string, timeout int64) PaymentRequirements {
	return PaymentRequirements{
		Scheme:            "exact",
		Network:           network,
		Amount:            amount,
		Asset:             asset,
		PayTo:             payTo,
		MaxTimeoutSeconds: int(timeout),
		Extra: map[string]interface{}{
			"name":    "USDC",
			"version": "2",
		},
	}
}

// CreatePaymentRequirement creates a payment requirement (alias for backward compatibility)
func CreatePaymentRequirement(network, amount, asset, payTo string, timeout int64) PaymentRequirements {
	return CreatePaymentRequirements(network, amount, asset, payTo, timeout)
}

// CAIP-2 network identifiers for x402 v2
const (
	NetworkBaseSepolia = "eip155:84532"
	NetworkBaseMainnet = "eip155:8453"

	USDCBaseSepolia = "0x036CbD53842c5426634e7929541eC2318f3dCF7e"
	USDCBaseMainnet = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
)

func GetUSDCAddress(network string) string {
	switch network {
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
	case NetworkBaseSepolia:
		return "Base Sepolia"
	case NetworkBaseMainnet:
		return "Base Mainnet"
	default:
		return "Unknown"
	}
}

const USDCDecimals = 6

func AtomicToUSDC(atomicAmount string) string {
	amount, err := strconv.ParseInt(atomicAmount, 10, 64)
	if err != nil {
		return "0.000000"
	}
	usdc := float64(amount) / float64(int64(1e6))
	return fmt.Sprintf("%.6f", usdc)
}

func USDCToAtomic(usdcAmount float64) string {
	atomic := int64(usdcAmount * 1e6)
	return strconv.FormatInt(atomic, 10)
}

func FormatUSDC(atomicAmount string) string {
	return AtomicToUSDC(atomicAmount) + " USDC"
}

func ParseAtomicAmount(amountStr string) (int64, error) {
	amountStr = strings.TrimSpace(amountStr)
	return strconv.ParseInt(amountStr, 10, 64)
}
