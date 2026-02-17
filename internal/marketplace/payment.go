package marketplace

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/x402-go"
	"github.com/mark3labs/x402-go/signers/evm"

	"github.com/asabya/betar/internal/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	FacilitatorURL = "https://facilitator.x402endpoints.online"
	DefaultTimeout = 60
)

// PaymentService handles EIP-402 (x402) payments
type PaymentService struct {
	wallet          *eth.Wallet
	paymentAddr     string
	network         string
	facilitator     string
	skipFacilitator bool
}

// PaymentServiceOption is a functional option for PaymentService
type PaymentServiceOption func(*PaymentService)

// WithSkipFacilitator sets whether to skip facilitator verification
func WithSkipFacilitator(skip bool) PaymentServiceOption {
	return func(s *PaymentService) {
		s.skipFacilitator = skip
	}
}

// FacilitatorVerifyRequest is sent to facilitator to verify payment (x402 v2 format)
type FacilitatorVerifyRequest struct {
	X402Version         int                     `json:"x402Version"`
	PaymentPayload      PaymentPayload          `json:"paymentPayload"`
	PaymentRequirements x402.PaymentRequirement `json:"paymentRequirements"`
}

// PaymentPayload is the payment payload structure (v2 format)
type PaymentPayload struct {
	X402Version int                     `json:"x402Version"`
	Scheme      string                  `json:"scheme"`
	Network     string                  `json:"network"`
	Accepted    x402.PaymentRequirement `json:"accepted"`
	Payload     x402.EVMPayload         `json:"payload"`
}

// FacilitatorVerifyResponse is returned by facilitator (x402 v2 format)
type FacilitatorVerifyResponse struct {
	IsValid        bool   `json:"isValid"`
	InvalidReason  string `json:"invalidReason,omitempty"`
	InvalidMessage string `json:"invalidMessage,omitempty"`
	Payer          string `json:"payer,omitempty"`
}

// NewPaymentService creates a new payment service
func NewPaymentService(wallet *eth.Wallet, paymentAddr string, opts ...PaymentServiceOption) *PaymentService {
	s := &PaymentService{
		wallet:          wallet,
		paymentAddr:     paymentAddr,
		network:         NetworkBaseSepolia,
		facilitator:     FacilitatorURL,
		skipFacilitator: true,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
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

	return s.signPayment(&requirement, paymentID)
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

	return s.signPayment(req, paymentID)
}

// createX402Signer creates an x402-go EVM signer for EIP-3009 signatures
func (s *PaymentService) createX402Signer() (*evm.Signer, error) {
	privateKeyHex := s.wallet.PrivateKeyHex()

	signer, err := evm.NewSigner(
		evm.WithPrivateKey("0x"+privateKeyHex),
		evm.WithNetwork(x402.BaseSepolia.NetworkID),
		evm.WithToken(x402.BaseSepolia.USDCAddress, x402.BaseSepolia.EIP3009Name, int(x402.BaseSepolia.Decimals)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create x402 signer: %w", err)
	}

	return signer, nil
}

// signPayment creates an EIP-3009 authorization for the payment requirement
func (s *PaymentService) signPayment(req *PaymentRequirement, paymentID string) (*PaymentHeader, error) {
	fmt.Printf("[signPayment] Starting payment signing - PayTo: %s, Amount: %s, Network: %s\n", req.PayTo, req.MaxAmountRequired, req.Network)

	signer, err := s.createX402Signer()
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}
	fmt.Printf("[signPayment] X402 signer created successfully\n")

	paymentPayload, err := signer.Sign(req)
	if err != nil {
		return nil, fmt.Errorf("failed to sign payment: %w", err)
	}
	fmt.Printf("[signPayment] Payment signed successfully %+v\n", paymentPayload)

	evmPayload, ok := paymentPayload.Payload.(x402.EVMPayload)
	if !ok {
		return nil, fmt.Errorf("unexpected payload type: %T", paymentPayload.Payload)
	}

	header := &PaymentHeader{
		Requirement: *req,
		Accepted:    req,
		Payer:       s.wallet.AddressHex(),
		PaymentID:   paymentID,
		Signature:   evmPayload.Signature,
		Payload:     &evmPayload,
	}

	fmt.Printf("[signPayment] Payment header created - Payer: %s, PaymentID: %s\n", header.Payer, header.PaymentID)
	fmt.Printf("[signPayment] Payment header: %+v\n", header)
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

// VerifyAndSettle verifies payment header and checks if transaction is confirmed on-chain
// For the new flow where buyer submits the transaction directly
func (s *PaymentService) VerifyAndSettle(ctx context.Context, header *PaymentHeader, buyerTxHash string) (string, error) {
	if header == nil {
		return "", fmt.Errorf("payment header is nil")
	}

	fmt.Printf("[VerifyAndSettle] >>> Starting payment verification for PaymentID: %s <<<\n", header.PaymentID)
	fmt.Printf("[VerifyAndSettle] Payer: %s, PayTo: %s, Amount: %s USDC\n", header.Payer, header.Requirement.PayTo, header.Requirement.MaxAmountRequired)

	// Step 1: Validate payment header locally
	fmt.Printf("[VerifyAndSettle] Step 1: Validating payment header locally...\n")
	if err := s.validatePaymentHeader(header); err != nil {
		return "", fmt.Errorf("invalid payment header: %w", err)
	}
	fmt.Printf("[VerifyAndSettle] Step 1: Local validation passed\n")

	// Step 2: Verify with facilitator (skip if flag is set)
	if s.skipFacilitator {
		fmt.Printf("[VerifyAndSettle] Step 2: Skipping facilitator verification (skipFacilitator=true)\n")
	} else {
		fmt.Printf("[VerifyAndSettle] Step 2: Verifying with facilitator...\n")
		verified, err := s.verifyWithFacilitator(ctx, header)
		if err != nil {
			return "", fmt.Errorf("facilitator verification failed: %w", err)
		}
		if !verified {
			return "", fmt.Errorf("payment verification failed")
		}
		fmt.Printf("[VerifyAndSettle] Step 2: Facilitator verification passed\n")
	}

	// Step 3: Verify buyer's on-chain transaction (buyer submitted it directly)
	if buyerTxHash == "" {
		return "", fmt.Errorf("no transaction hash provided - buyer must submit payment transaction")
	}

	fmt.Printf("[VerifyAndSettle] Step 3: Verifying buyer's on-chain transaction...\n")
	txHash := common.HexToHash(buyerTxHash)
	receipt, err := s.wallet.WaitForTransaction(ctx, txHash)
	if err != nil {
		fmt.Printf("[VerifyAndSettle] Transaction verification failed: %v\n", err)
		return "", fmt.Errorf("transaction not confirmed: %w", err)
	}

	if receipt.Status != 1 {
		return "", fmt.Errorf("transaction failed on-chain")
	}

	fmt.Printf("[VerifyAndSettle] >>> Transaction confirmed. TxHash: %s, Block: %d <<<\n", buyerTxHash, receipt.BlockNumber)

	return buyerTxHash, nil
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
	var acceptedReq x402.PaymentRequirement
	if header.Accepted != nil {
		acceptedReq = *header.Accepted
	}

	req := FacilitatorVerifyRequest{
		X402Version: 2,
		PaymentPayload: PaymentPayload{
			X402Version: 2,
			Scheme:      header.Requirement.Scheme,
			Network:     header.Requirement.Network,
			Accepted:    acceptedReq,
			Payload:     *header.Payload,
		},
		PaymentRequirements: header.Requirement,
	}

	// Set dummy resource URL (v2 requires a valid URL)
	req.PaymentRequirements.Resource = "https://betar.agent/execute"

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

// SubmitPayment submits the EIP-3009 transferWithAuthorization transaction
// This is called by the BUYER after signing to submit the on-chain transaction
func (s *PaymentService) SubmitPayment(ctx context.Context, header *PaymentHeader) (string, error) {
	if header == nil {
		return "", fmt.Errorf("payment header is nil")
	}

	if header.Payload == nil {
		return "", fmt.Errorf("payment payload is nil")
	}

	auth := header.Payload.Authorization
	from := common.HexToAddress(auth.From)
	to := common.HexToAddress(auth.To)

	// Value is a uint256 in decimal string
	value, ok := new(big.Int).SetString(auth.Value, 10)
	if !ok {
		return "", fmt.Errorf("invalid amount: %s", auth.Value)
	}

	// ValidAfter and ValidBefore are uint256 timestamps in decimal string
	validAfter, ok := new(big.Int).SetString(auth.ValidAfter, 10)
	if !ok {
		return "", fmt.Errorf("invalid validAfter: %s", auth.ValidAfter)
	}

	validBefore, ok := new(big.Int).SetString(auth.ValidBefore, 10)
	if !ok {
		return "", fmt.Errorf("invalid validBefore: %s", auth.ValidBefore)
	}

	// Nonce is a bytes32 - convert hex string (may have 0x prefix) to 32 bytes
	nonceStr := auth.Nonce
	if strings.HasPrefix(nonceStr, "0x") {
		nonceStr = nonceStr[2:]
	}
	nonceBytes, err := hex.DecodeString(nonceStr)
	if err != nil {
		return "", fmt.Errorf("invalid nonce hex: %w", err)
	}
	if len(nonceBytes) != 32 {
		return "", fmt.Errorf("nonce must be 32 bytes, got %d", len(nonceBytes))
	}

	// Signature is hex - may have 0x prefix
	sigStr := header.Payload.Signature
	if strings.HasPrefix(sigStr, "0x") {
		sigStr = sigStr[2:]
	}
	signature, err := hex.DecodeString(sigStr)
	if err != nil {
		return "", fmt.Errorf("invalid signature: %w", err)
	}

	usdcAddress := common.HexToAddress(header.Requirement.Asset)

	fmt.Printf("[SubmitPayment] Submitting EIP-3009 transferWithAuthorization\n")
	fmt.Printf("[SubmitPayment] From: %s, To: %s, Amount: %s\n", from.Hex(), to.Hex(), value.String())
	fmt.Printf("[SubmitPayment] USDC Contract: %s\n", usdcAddress.Hex())
	fmt.Printf("[SubmitPayment] ValidAfter: %s, ValidBefore: %s\n", validAfter.String(), validBefore.String())
	fmt.Printf("[SubmitPayment] Nonce (hex): %x\n", nonceBytes)

	callData, err := s.createTransferWithAuthorizationData(from, to, value, validAfter, validBefore, nonceBytes, signature)
	if err != nil {
		return "", fmt.Errorf("failed to create tx data: %w", err)
	}

	txHash, err := s.wallet.SubmitTransaction(ctx, usdcAddress, big.NewInt(0), callData)
	if err != nil {
		return "", fmt.Errorf("failed to submit transaction: %w", err)
	}

	fmt.Printf("[SubmitPayment] Transaction submitted. TxHash: %s\n", txHash.Hex())
	return txHash.Hex(), nil
}

// createTransferWithAuthorizationData creates the call data for EIP-3009 transferWithAuthorization
func (s *PaymentService) createTransferWithAuthorizationData(
	from, to common.Address,
	amount, validAfter, validBefore *big.Int,
	nonceBytes []byte,
	signature []byte,
) ([]byte, error) {
	if len(signature) != 65 {
		return nil, fmt.Errorf("signature must be 65 bytes, got %d", len(signature))
	}

	if len(nonceBytes) != 32 {
		return nil, fmt.Errorf("nonce must be 32 bytes, got %d", len(nonceBytes))
	}

	var nonce [32]byte
	copy(nonce[:], nonceBytes)

	callData, err := usdcABI.Pack(
		"transferWithAuthorization",
		from,
		to,
		amount,
		validAfter,
		validBefore,
		nonce,
		signature,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pack ABI: %w", err)
	}

	return callData, nil
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
