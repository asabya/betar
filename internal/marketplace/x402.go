package marketplace

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	x402v2 "github.com/mark3labs/x402-go/v2"
)

// X402LibP2PVersion is the protocol version for x402 over libp2p.
const X402LibP2PVersion = "1.0"

// PreemptiveNonce is used as server_nonce when the client pre-signs without a server challenge.
const PreemptiveNonce = "preemptive"

// X402ProtocolMessageTypes are the message type strings used in /x402/libp2p/1.0.0 frames.
const (
	MsgTypeX402Request         = "x402.request"
	MsgTypeX402PaymentRequired = "x402.payment_required"
	MsgTypeX402PaidRequest     = "x402.paid_request"
	MsgTypeX402Response        = "x402.response"
	MsgTypeX402Error           = "x402.error"
)

// X402ErrorCode is a typed integer for structured error codes in x402.error messages.
type X402ErrorCode int

const (
	ErrInvalidMessage   X402ErrorCode = 1000
	ErrUnknownResource  X402ErrorCode = 1001
	ErrPaymentRequired  X402ErrorCode = 2000
	ErrPaymentInvalid   X402ErrorCode = 2001
	ErrNonceMismatch    X402ErrorCode = 2002
	ErrNonceExpired     X402ErrorCode = 2003
	ErrNonceUsed        X402ErrorCode = 2004
	ErrAmountWrong      X402ErrorCode = 2005
	ErrSettlementFailed X402ErrorCode = 2007
	ErrExecutionFailed  X402ErrorCode = 3000
)

// X402PaymentEnvelope carries signed payment data inside x402.request or x402.paid_request.
type X402PaymentEnvelope struct {
	X402Version int         `json:"x402_version"`
	Scheme      string      `json:"scheme"`
	Network     string      `json:"network"`
	ServerNonce string      `json:"server_nonce"`
	Payer       string      `json:"payer"`
	Payload     *EVMPayload `json:"payload,omitempty"`
}

// X402Request is sent client → server to request resource execution.
// Payment is nil on the first attempt (standard flow) or non-nil for preemptive payment.
type X402Request struct {
	Version       string               `json:"version"`
	CorrelationID string               `json:"correlation_id"`
	Resource      string               `json:"resource"`
	Method        string               `json:"method"`
	Payment       *X402PaymentEnvelope `json:"payment"`
	Body          []byte               `json:"body"`
	CallerDID     string               `json:"caller_did,omitempty"`
}

// X402PaymentRequired is sent server → client when payment is required (analogous to HTTP 402).
type X402PaymentRequired struct {
	Version             string               `json:"version"`
	CorrelationID       string               `json:"correlation_id"`
	ChallengeNonce      string               `json:"challenge_nonce"`
	ChallengeExpiresAt  int64                `json:"challenge_expires_at"`
	PaymentRequirements *PaymentRequirements `json:"payment_requirements"`
	Message             string               `json:"message"`
	SellerDID           string               `json:"seller_did,omitempty"`
}

// X402PaidRequest is sent client → server with a signed payment attached.
type X402PaidRequest struct {
	Version       string              `json:"version"`
	CorrelationID string              `json:"correlation_id"`
	Payment       X402PaymentEnvelope `json:"payment"`
	Body          []byte              `json:"body"`
	CallerDID     string              `json:"caller_did,omitempty"`
}

// X402Response is sent server → client on successful execution.
type X402Response struct {
	Version       string `json:"version"`
	CorrelationID string `json:"correlation_id"`
	PaymentID     string `json:"payment_id"`
	TxHash        string `json:"tx_hash"`
	Body          []byte `json:"body"`
	SellerDID     string `json:"seller_did,omitempty"`
}

// X402Error is sent server → client when a typed error occurs.
type X402Error struct {
	Version       string        `json:"version"`
	CorrelationID string        `json:"correlation_id"`
	ErrorCode     X402ErrorCode `json:"error_code"`
	ErrorName     string        `json:"error_name"`
	Message       string        `json:"message"`
	Retryable     bool          `json:"retryable"`
}

// NewX402Error constructs an X402Error with the correct name and retryable flag for the given code.
func NewX402Error(correlationID string, code X402ErrorCode, message string) *X402Error {
	e := &X402Error{
		Version:       X402LibP2PVersion,
		CorrelationID: correlationID,
		ErrorCode:     code,
		Message:       message,
	}
	switch code {
	case ErrInvalidMessage:
		e.ErrorName = "INVALID_MESSAGE"
		e.Retryable = false
	case ErrUnknownResource:
		e.ErrorName = "UNKNOWN_RESOURCE"
		e.Retryable = false
	case ErrPaymentRequired:
		e.ErrorName = "PAYMENT_REQUIRED"
		e.Retryable = true
	case ErrPaymentInvalid:
		e.ErrorName = "PAYMENT_INVALID"
		e.Retryable = false
	case ErrNonceMismatch:
		e.ErrorName = "PAYMENT_NONCE_MISMATCH"
		e.Retryable = false
	case ErrNonceExpired:
		e.ErrorName = "PAYMENT_NONCE_EXPIRED"
		e.Retryable = true
	case ErrNonceUsed:
		e.ErrorName = "PAYMENT_NONCE_USED"
		e.Retryable = false
	case ErrAmountWrong:
		e.ErrorName = "PAYMENT_AMOUNT_WRONG"
		e.Retryable = false
	case ErrSettlementFailed:
		e.ErrorName = "SETTLEMENT_FAILED"
		e.Retryable = true
	case ErrExecutionFailed:
		e.ErrorName = "EXECUTION_FAILED"
		e.Retryable = false
	default:
		e.ErrorName = "UNKNOWN_ERROR"
		e.Retryable = false
	}
	return e
}

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
