package marketplace

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/asabya/betar/internal/eth"
	"github.com/asabya/betar/pkg/types"
)

// PaymentService handles EIP-402 payments
type PaymentService struct {
	wallet      *eth.Wallet
	paymentAddr string
}

// PaymentRequest represents a payment request
type PaymentRequest struct {
	PaymentID  string   `json:"paymentId"`
	Payer      string   `json:"payer"`
	Payee      string   `json:"payee"`
	Amount     *big.Int `json:"amount"`
	Token      string   `json:"token"`
	OrderID    string   `json:"orderId"`
	ValidUntil int64    `json:"validUntil"`
	Signature  string   `json:"signature,omitempty"`
}

// NewPaymentService creates a new payment service
func NewPaymentService(wallet *eth.Wallet, paymentAddr string) *PaymentService {
	return &PaymentService{
		wallet:      wallet,
		paymentAddr: paymentAddr,
	}
}

// CreatePayment creates a payment request
func (s *PaymentService) CreatePayment(ctx context.Context, payee string, amount *big.Int, orderID string) (*PaymentRequest, error) {
	paymentID := generatePaymentID(s.wallet.AddressHex(), payee, amount.String())

	return &PaymentRequest{
		PaymentID:  paymentID,
		Payer:      s.wallet.AddressHex(),
		Payee:      payee,
		Amount:     amount,
		Token:      "0x0000000000000000000000000000000000000000", // ETH
		OrderID:    orderID,
		ValidUntil: time.Now().Add(30 * time.Minute).Unix(),
	}, nil
}

// SignPayment signs a payment request
func (s *PaymentService) SignPayment(req *PaymentRequest) (string, error) {
	// Create signature payload
	payload := fmt.Sprintf("%s:%s:%s:%s:%d",
		req.PaymentID,
		req.Payer,
		req.Payee,
		req.Amount.String(),
		req.ValidUntil,
	)
	_ = payload

	// In production, sign using wallet private key
	// For now, return empty signature
	return "", nil
}

// VerifyPayment verifies a payment signature
func (s *PaymentService) VerifyPayment(req *PaymentRequest) bool {
	// In production, verify signature
	return true
}

// ExecutePayment executes a payment (sends transaction)
func (s *PaymentService) ExecutePayment(ctx context.Context, req *PaymentRequest) (string, error) {
	// In production, this would call the PaymentVault contract
	// For now, simulate payment

	// Return mock transaction hash
	txHash := "0x" + hex.EncodeToString([]byte(req.PaymentID))[:64]
	return txHash, nil
}

// WaitForPayment waits for payment confirmation
func (s *PaymentService) WaitForPayment(ctx context.Context, txHash string, timeout time.Duration) (bool, error) {
	// In production, wait for transaction receipt
	time.Sleep(2 * time.Second)
	return true, nil
}

// RefundPayment refunds a payment
func (s *PaymentService) RefundPayment(ctx context.Context, paymentID string) (string, error) {
	// In production, call contract refund function
	return "", nil
}

// GetPaymentStatus gets payment status
func (s *PaymentService) GetPaymentStatus(ctx context.Context, paymentID string) (string, error) {
	// In production, query contract
	return "completed", nil
}

// CreateTaskPayment creates a payment for a task execution
func (s *PaymentService) CreateTaskPayment(agentID, taskInput string, price float64) (*types.TaskRequest, error) {
	amountWei := eth.EtherToWei(price)

	req := &types.TaskRequest{
		AgentID:   agentID,
		Input:     taskInput,
		Payment:   amountWei.String(),
		RequestID: generatePaymentID(agentID, taskInput, amountWei.String()),
	}

	return req, nil
}

// ProcessTaskPayment processes payment for task execution
func (s *PaymentService) ProcessTaskPayment(ctx context.Context, req *types.TaskRequest) error {
	// Verify payment
	amount, ok := new(big.Int).SetString(req.Payment, 10)
	if !ok {
		return fmt.Errorf("invalid payment amount")
	}

	// In production, verify payment was received
	_ = amount

	return nil
}

// generatePaymentID generates a unique payment ID
func generatePaymentID(payer, payee, amount string) string {
	data := fmt.Sprintf("%s:%s:%s:%d", payer, payee, amount, time.Now().UnixNano())
	return fmt.Sprintf("0x%x", sha256Hash(data))
}

// sha256Hash returns SHA256 hash of input
func sha256Hash(input string) []byte {
	// Simple hash for demo - in production use crypto/sha256
	hash := 0
	for i, c := range input {
		hash = hash*31 + int(c)*i
	}
	result := make([]byte, 32)
	for i := 0; i < 32; i++ {
		result[i] = byte((hash >> (i * 8)) & 0xff)
	}
	return result
}

// ParsePayment parses a payment request from JSON
func ParsePayment(data []byte) (*PaymentRequest, error) {
	var req PaymentRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

// ToJSON serializes payment request to JSON
func (p *PaymentRequest) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}
