package marketplace

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"io"

	v2 "github.com/mark3labs/x402-go"
	"github.com/mark3labs/x402-go/signers/evm"

	"github.com/asabya/betar/internal/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	FacilitatorURL = "https://facilitator.x402.rs"
	DefaultTimeout = 60
)

// PaymentService handles EIP-402 (x402) payments
type PaymentService struct {
	wallet      *eth.Wallet
	paymentAddr string
	network     string
	facilitator string
}

// FacilitatorVerifyRequest is sent to facilitator to verify payment (x402 v2 format)
type FacilitatorVerifyRequest struct {
	X402Version         int                   `json:"x402Version"`
	PaymentPayload      PaymentPayloadV2      `json:"paymentPayload"`
	PaymentRequirements v2.PaymentRequirement `json:"paymentRequirements"`
}

// PaymentPayloadV2 is the v2 payment payload structure
type PaymentPayloadV2 struct {
	X402Version int                   `json:"x402Version"`
	Accepted    v2.PaymentRequirement `json:"accepted"`
	Payload     v2.EVMPayload         `json:"payload"`
}

// FacilitatorVerifyResponse is returned by facilitator (x402 v2 format)
type FacilitatorVerifyResponse struct {
	IsValid        bool   `json:"isValid"`
	InvalidReason  string `json:"invalidReason,omitempty"`
	InvalidMessage string `json:"invalidMessage,omitempty"`
	Payer          string `json:"payer,omitempty"`
}

// NewPaymentService creates a new payment service
func NewPaymentService(wallet *eth.Wallet, paymentAddr string) *PaymentService {
	return &PaymentService{
		wallet:      wallet,
		paymentAddr: paymentAddr,
		network:     NetworkBaseSepolia, // Default to Base Sepolia
		facilitator: FacilitatorURL,
	}
}

// GetPaymentAddress returns the payment address for receiving payments
func (s *PaymentService) GetPaymentAddress() string {
	return s.paymentAddr
}

// GetUSDCBalance returns the USDC balance of the wallet
func (s *PaymentService) GetUSDCBalance(ctx context.Context) (*big.Int, error) {
	usdcAddress := GetUSDCAddress(s.network)
	return s.wallet.ERC20Balance(ctx, common.HexToAddress(usdcAddress))
}

// CreateRequirement creates an unsigned payment requirement (for seller to send to buyer)
func (s *PaymentService) CreateRequirement(payee string, amount string) (*PaymentRequirement, error) {
	usdcAddress := GetUSDCAddress(s.network)
	requirement := CreatePaymentRequirement(
		s.network,
		amount,
		usdcAddress,
		payee,
		DefaultTimeout,
	)
	return &requirement, nil
}

// CreatePayment creates a payment requirement for a buyer to sign
func (s *PaymentService) CreatePayment(ctx context.Context, payee string, amount string, orderID string) (*PaymentHeader, error) {
	// Create payment requirement
	usdcAddress := GetUSDCAddress(s.network)
	requirement := CreatePaymentRequirement(
		s.network,
		amount,
		usdcAddress,
		payee,
		DefaultTimeout,
	)

	// Create payment ID
	paymentID := generatePaymentID(
		s.wallet.AddressHex(),
		payee,
		amount,
		s.network,
	)

	// Create payment header
	header := &PaymentHeader{
		Requirement: requirement,
		Payer:       s.wallet.AddressHex(),
		PaymentID:   paymentID,
	}

	// Sign the requirement
	signature, err := s.signRequirement(&requirement, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to sign requirement: %w", err)
	}
	header.Signature = signature

	return header, nil
}

// SignRequirement signs a payment requirement and creates a PaymentHeader (used by buyer)
func (s *PaymentService) SignRequirement(req *PaymentRequirement, orderID string) (*PaymentHeader, error) {
	// Create payment ID
	paymentID := generatePaymentID(
		s.wallet.AddressHex(),
		req.PayTo,
		req.MaxAmountRequired,
		req.Network,
	)

	// Create payment header
	header := &PaymentHeader{
		Requirement: *req,
		Payer:       s.wallet.AddressHex(),
		PaymentID:   paymentID,
	}

	// Sign the requirement
	signature, err := s.signRequirement(req, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to sign requirement: %w", err)
	}
	header.Signature = signature

	return header, nil
}

// createX402Signer creates an x402-go EVM signer for EIP-3009 signatures
func (s *PaymentService) createX402Signer() (*evm.Signer, error) {
	privateKeyHex := s.wallet.PrivateKeyHex()

	usdcAddress := GetUSDCAddress(s.network)

	signer, err := evm.NewSigner(
		evm.WithPrivateKey(privateKeyHex),
		evm.WithNetwork(s.network),
		evm.WithToken(usdcAddress, "USDC", 6),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create x402 signer: %w", err)
	}

	return signer, nil
}

// signPayment creates an EIP-3009 authorization for the payment requirement
func (s *PaymentService) signPayment(req *PaymentRequirement, paymentID string) (*PaymentHeader, error) {
	signer, err := s.createX402Signer()
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	paymentPayload, err := signer.Sign(req)
	if err != nil {
		return nil, fmt.Errorf("failed to sign payment: %w", err)
	}

	evmPayload, ok := paymentPayload.Payload.(v2.EVMPayload)
	if !ok {
		return nil, fmt.Errorf("unexpected payload type: %T", paymentPayload.Payload)
	}

	header := &PaymentHeader{
		Requirement: *req,
		Accepted:    *req,
		Payer:       s.wallet.AddressHex(),
		PaymentID:   paymentID,
		Signature:   evmPayload.Signature,
		Payload:     &evmPayload,
	}

	return header, nil
}

// signRequirement creates an EIP-712 signature for the payment requirement (legacy)
func (s *PaymentService) signRequirement(req *PaymentRequirement, orderID string) (string, error) {
	// Create the payment payload to sign
	// Following x402 v2 EIP-712 structure
	sigData := fmt.Sprintf("%s:%s:%s:%s:%s:%d",
		req.Scheme,
		req.Network,
		req.MaxAmountRequired,
		req.Asset,
		req.PayTo,
		req.MaxTimeoutSeconds,
	)

	// Add order ID for replay protection
	sigData = fmt.Sprintf("%s:%s", sigData, orderID)

	// Hash the data
	hash := crypto.Keccak256Hash([]byte(sigData))

	// Sign with wallet's private key
	signature, err := s.wallet.Sign(hash.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to sign payment: %w", err)
	}

	return hex.EncodeToString(signature), nil
}

// VerifyAndSettle verifies payment with facilitator and settles on-chain
func (s *PaymentService) VerifyAndSettle(ctx context.Context, header *PaymentHeader) (string, error) {
	if header == nil {
		return "", fmt.Errorf("payment header is nil")
	}

	// Step 1: Validate payment header locally
	if err := s.validatePaymentHeader(header); err != nil {
		return "", fmt.Errorf("invalid payment header: %w", err)
	}

	// Step 2: Verify with facilitator
	verified, err := s.verifyWithFacilitator(ctx, header)
	if err != nil {
		return "", fmt.Errorf("facilitator verification failed: %w", err)
	}
	if !verified {
		return "", fmt.Errorf("payment verification failed")
	}

	// Step 3: Settle on-chain (transfer USDC to seller)
	fmt.Printf("[VerifyAndSettle] Settlement started for PaymentID: %s\n", header.PaymentID)
	txHash, err := s.settlePayment(ctx, header)
	if err != nil {
		fmt.Printf("[VerifyAndSettle] Settlement failed: %v\n", err)
		return "", fmt.Errorf("payment settlement failed: %w", err)
	}
	fmt.Printf("[VerifyAndSettle] Settlement successful. TxHash: %s\n", txHash)

	return txHash, nil
}

// validatePaymentHeader validates the payment header locally
func (s *PaymentService) validatePaymentHeader(header *PaymentHeader) error {
	if header.Requirement.Scheme != "exact" {
		fmt.Printf("[validatePaymentHeader] Unsupported scheme: %s\n", header.Requirement.Scheme)
		return fmt.Errorf("unsupported payment scheme: %s", header.Requirement.Scheme)
	}

	if header.Requirement.Network != s.network {
		fmt.Printf("[validatePaymentHeader] Unsupported network: %s (expected: %s)\n", header.Requirement.Network, s.network)
		return fmt.Errorf("unsupported network: %s (expected: %s)",
			header.Requirement.Network, s.network)
	}

	if header.Signature == "" {
		fmt.Printf("[validatePaymentHeader] Signature missing\n")
		return fmt.Errorf("payment signature missing")
	}

	if header.Requirement.PayTo != s.paymentAddr {
		fmt.Printf("[validatePaymentHeader] PayTo mismatch: %s (expected: %s)\n", header.Requirement.PayTo, s.paymentAddr)
		return fmt.Errorf("payment address mismatch: expected %s, got %s",
			s.paymentAddr, header.Requirement.PayTo)
	}

	// Verify USDC address
	expectedUSDC := GetUSDCAddress(s.network)
	if header.Requirement.Asset != expectedUSDC {
		fmt.Printf("[validatePaymentHeader] Asset mismatch: %s (expected: %s)\n", header.Requirement.Asset, expectedUSDC)
		return fmt.Errorf("invalid asset: expected %s, got %s", expectedUSDC, header.Requirement.Asset)
	}

	return nil
}

// verifyWithFacilitator sends payment to x402 facilitator for verification
func (s *PaymentService) verifyWithFacilitator(ctx context.Context, header *PaymentHeader) (bool, error) {
	req := FacilitatorVerifyRequest{
		X402Version: 2,
		PaymentPayload: PaymentPayloadV2{
			X402Version: 2,
			Accepted:    header.Accepted,
			Payload:     *header.Payload,
		},
		PaymentRequirements: header.Requirement,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("[verifyWithFacilitator] Failed to marshal request: %v\n", err)
		return false, fmt.Errorf("failed to marshal verify request: %w", err)
	}

	fmt.Printf("[verifyWithFacilitator] Verification Request URL: %s/verify\n", s.facilitator)
	fmt.Printf("[verifyWithFacilitator] Request Body: %s\n", string(reqBody))

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		s.facilitator+"/verify", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("[verifyWithFacilitator] Failed to create HTTP request: %v\n", err)
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Printf("[verifyWithFacilitator] HTTP request failed: %v\n", err)
		return false, fmt.Errorf("facilitator request failed: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("[verifyWithFacilitator] Response Status: %d\n", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		// Read body for error details
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("[verifyWithFacilitator] Error Response Body: %s\n", string(bodyBytes))
		return false, fmt.Errorf("facilitator returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var verifyResp FacilitatorVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		fmt.Printf("[verifyWithFacilitator] Failed to decode response: %v\n", err)
		return false, fmt.Errorf("failed to decode facilitator response: %w", err)
	}

	if !verifyResp.IsValid {
		fmt.Printf("[verifyWithFacilitator] Facilitator error: %s\n", verifyResp.InvalidMessage)
		return false, fmt.Errorf("payment invalid: %s", verifyResp.InvalidMessage)
	}

	fmt.Printf("[verifyWithFacilitator] Verification Result: Valid=%v, Payer=%s\n", verifyResp.IsValid, verifyResp.Payer)
	return verifyResp.IsValid, nil
}

// settlePayment executes the USDC transfer to the seller
func (s *PaymentService) settlePayment(ctx context.Context, header *PaymentHeader) (string, error) {
	amount, ok := new(big.Int).SetString(header.Requirement.MaxAmountRequired, 10)
	if !ok {
		return "", fmt.Errorf("invalid amount: %s", header.Requirement.MaxAmountRequired)
	}

	usdcAddress := common.HexToAddress(header.Requirement.Asset)
	toAddress := common.HexToAddress(header.Requirement.PayTo)

	txHash, err := s.wallet.TransferERC20(ctx, usdcAddress, toAddress, amount)
	if err != nil {
		return "", fmt.Errorf("transfer failed: %w", err)
	}

	return txHash.Hex(), nil
}

// VerifyPayment verifies a buyer's payment header without settling
// Used by sellers to confirm payment is valid before executing the task
func (s *PaymentService) VerifyPayment(header *PaymentHeader) error {
	if header == nil {
		return fmt.Errorf("payment header is nil")
	}

	if header.Requirement.Scheme != "exact" {
		return fmt.Errorf("unsupported payment scheme: %s", header.Requirement.Scheme)
	}

	if header.Signature == "" {
		return fmt.Errorf("payment signature missing")
	}

	return nil
}

// WaitForSettlement waits for the settlement transaction to be confirmed
func (s *PaymentService) WaitForSettlement(ctx context.Context, txHash string, timeout time.Duration) (bool, error) {
	hash := common.HexToHash(txHash)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err := s.wallet.WaitForTransaction(ctx, hash)
	if err != nil {
		return false, err
	}

	return true, nil
}

// generatePaymentID creates a unique payment ID
func generatePaymentID(payer, payee, amount, network string) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%d", payer, payee, amount, network, time.Now().UnixNano())
	hash := crypto.Keccak256Hash([]byte(data))
	return fmt.Sprintf("0x%x", hash)
}

// ParsePayment parses a payment header from JSON
func ParsePayment(data []byte) (*PaymentHeader, error) {
	var header PaymentHeader
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, err
	}
	return &header, nil
}

// ToJSON serializes payment header to JSON
func (p *PaymentHeader) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}
