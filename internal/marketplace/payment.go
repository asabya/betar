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

	x402v2 "github.com/mark3labs/x402-go/v2"
	"github.com/mark3labs/x402-go/v2/signers/evm"

	"github.com/asabya/betar/internal/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	FacilitatorURL = "http://localhost:8080"
	DefaultTimeout = 60
)

type PaymentService struct {
	wallet      *eth.Wallet
	paymentAddr string
	network     string
	facilitator string
	verifier    *PaymentVerifier
}

type FacilitatorVerifyRequest struct {
	X402Version         int                   `json:"x402Version"`
	PaymentPayload      x402v2.PaymentPayload `json:"paymentPayload"`
	PaymentRequirements PaymentRequirements   `json:"paymentRequirements"`
}

type FacilitatorVerifyResponse struct {
	IsValid        bool   `json:"isValid"`
	InvalidReason  string `json:"invalidReason,omitempty"`
	InvalidMessage string `json:"invalidMessage,omitempty"`
	Payer          string `json:"payer,omitempty"`
}

func NewPaymentService(wallet *eth.Wallet, paymentAddr string) *PaymentService {
	return &PaymentService{
		wallet:      wallet,
		paymentAddr: paymentAddr,
		network:     NetworkBaseSepolia,
		facilitator: FacilitatorURL,
		verifier:    NewPaymentVerifier(NetworkBaseSepolia),
	}
}

func (s *PaymentService) GetPaymentAddress() string {
	return s.paymentAddr
}

func (s *PaymentService) GetUSDCBalance(ctx context.Context) (*big.Int, error) {
	usdcAddress := GetUSDCAddress(s.network)
	return s.wallet.ERC20Balance(ctx, common.HexToAddress(usdcAddress))
}

func (s *PaymentService) CreateRequirement(payee string, amount string) (*PaymentRequirements, error) {
	usdcAddress := GetUSDCAddress(s.network)
	requirement := CreatePaymentRequirements(
		s.network,
		amount,
		usdcAddress,
		payee,
		DefaultTimeout,
	)
	return &requirement, nil
}

func (s *PaymentService) CreatePayment(ctx context.Context, payee string, amount string, orderID string) (*PaymentHeader, error) {
	usdcAddress := GetUSDCAddress(s.network)
	requirement := CreatePaymentRequirements(
		s.network,
		amount,
		usdcAddress,
		payee,
		DefaultTimeout,
	)

	paymentID := generatePaymentID(
		s.wallet.AddressHex(),
		payee,
		amount,
		s.network,
	)

	return s.signPayment(&requirement, paymentID)
}

func (s *PaymentService) SignRequirement(req *PaymentRequirements, orderID string) (*PaymentHeader, error) {
	paymentID := generatePaymentID(
		s.wallet.AddressHex(),
		req.PayTo,
		req.Amount,
		req.Network,
	)

	return s.signPayment(req, paymentID)
}

func (s *PaymentService) createX402Signer() (*evm.Signer, error) {
	privateKeyHex := s.wallet.PrivateKeyHex()

	tokens := []x402v2.TokenConfig{
		{
			Address:  GetUSDCAddress(s.network),
			Symbol:   "USDC",
			Decimals: 6,
		},
	}

	signer, err := evm.NewSigner(s.network, privateKeyHex, tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to create x402 signer: %w", err)
	}

	return signer, nil
}

func (s *PaymentService) signPayment(req *PaymentRequirements, paymentID string) (*PaymentHeader, error) {
	fmt.Printf("[signPayment] Starting payment signing - PayTo: %s, Amount: %s, Network: %s\n", req.PayTo, req.Amount, req.Network)

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

	evmPayload, ok := paymentPayload.Payload.(x402v2.EVMPayload)
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

func (s *PaymentService) signRequirement(req *PaymentRequirements, orderID string) (string, error) {
	sigData := fmt.Sprintf("%s:%s:%s:%s:%s:%d",
		req.Scheme,
		req.Network,
		req.Amount,
		req.Asset,
		req.PayTo,
		req.MaxTimeoutSeconds,
	)

	sigData = fmt.Sprintf("%s:%s", sigData, orderID)

	hash := crypto.Keccak256Hash([]byte(sigData))

	signature, err := s.wallet.Sign(hash.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to sign payment: %w", err)
	}

	return hex.EncodeToString(signature), nil
}

func (s *PaymentService) VerifyAndSettle(ctx context.Context, header *PaymentHeader, expectedAmount *big.Int) (string, error) {
	if header == nil {
		return "", fmt.Errorf("payment header is nil")
	}

	fmt.Printf("[VerifyAndSettle] >>> Starting payment verification for PaymentID: %s <<<\n", header.PaymentID)
	fmt.Printf("[VerifyAndSettle] Payer: %s, PayTo: %s, Amount: %s USDC\n", header.Payer, header.Requirement.PayTo, header.Requirement.Amount)

	fmt.Printf("[VerifyAndSettle] Step 1: Comprehensive local validation (signature, timestamps, nonce)...\n")
	if err := s.verifier.ValidatePayment(header, s.paymentAddr, expectedAmount); err != nil {
		return "", fmt.Errorf("payment validation failed: %w", err)
	}
	fmt.Printf("[VerifyAndSettle] Step 1: Local validation passed (signature verified, nonce checked)\n")

	fmt.Printf("[VerifyAndSettle] Step 2: Settling with facilitator...\n")
	txHash, err := s.settleWithFacilitatorWithRetry(ctx, header)
	if err != nil {
		return "", fmt.Errorf("facilitator settlement failed: %w", err)
	}
	fmt.Printf("[VerifyAndSettle] Step 2: Facilitator settlement successful, txHash: %s\n", txHash)

	fmt.Printf("[VerifyAndSettle] Step 3: Waiting for transaction confirmation...\n")
	hash := common.HexToHash(txHash)

	receipt, err := s.wallet.WaitForTransaction(ctx, hash)
	if err != nil {
		fmt.Printf("[VerifyAndSettle] Transaction confirmation failed: %v\n", err)
		return "", fmt.Errorf("transaction not confirmed: %w", err)
	}

	if receipt.Status != 1 {
		return "", fmt.Errorf("transaction failed on-chain")
	}

	fmt.Printf("[VerifyAndSettle] >>> Transaction confirmed. TxHash: %s, Block: %d <<<\n", txHash, receipt.BlockNumber)

	return txHash, nil
}

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

	expectedUSDC := GetUSDCAddress(s.network)
	if header.Requirement.Asset != expectedUSDC {
		fmt.Printf("[validatePaymentHeader] Asset mismatch: %s (expected: %s)\n", header.Requirement.Asset, expectedUSDC)
		return fmt.Errorf("invalid asset: expected %s, got %s", expectedUSDC, header.Requirement.Asset)
	}

	return nil
}

func (s *PaymentService) verifyWithFacilitator(ctx context.Context, header *PaymentHeader) (bool, error) {
	acceptedReq := header.Requirement
	if header.Accepted != nil {
		acceptedReq = *header.Accepted
	}

	payload := x402v2.PaymentPayload{
		X402Version: x402v2.X402Version,
		Accepted:    acceptedReq,
		Payload:     *header.Payload,
	}

	req := FacilitatorVerifyRequest{
		X402Version:         x402v2.X402Version,
		PaymentPayload:      payload,
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

type FacilitatorSettleRequest struct {
	X402Version         int                   `json:"x402Version"`
	PaymentPayload      x402v2.PaymentPayload `json:"paymentPayload"`
	PaymentRequirements PaymentRequirements   `json:"paymentRequirements"`
}

type FacilitatorSettleResponse struct {
	Success       bool   `json:"success"`
	Transaction   string `json:"transaction,omitempty"`
	InvalidReason string `json:"invalidReason,omitempty"`
}

func (s *PaymentService) settleWithFacilitator(ctx context.Context, header *PaymentHeader) (string, error) {
	acceptedReq := header.Requirement
	if header.Accepted != nil {
		acceptedReq = *header.Accepted
	}

	payload := x402v2.PaymentPayload{
		X402Version: x402v2.X402Version,
		Accepted:    acceptedReq,
		Payload:     *header.Payload,
	}

	req := FacilitatorSettleRequest{
		X402Version:         x402v2.X402Version,
		PaymentPayload:      payload,
		PaymentRequirements: header.Requirement,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("[settleWithFacilitator] Failed to marshal request: %v\n", err)
		return "", fmt.Errorf("failed to marshal settle request: %w", err)
	}

	fmt.Printf("[settleWithFacilitator] Settle Request URL: %s/settle\n", s.facilitator)
	fmt.Printf("[settleWithFacilitator] Request Body: %s\n", string(reqBody))

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		s.facilitator+"/settle", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Printf("[settleWithFacilitator] Failed to create HTTP request: %v\n", err)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Printf("[settleWithFacilitator] HTTP request failed: %v\n", err)
		return "", fmt.Errorf("facilitator settle request failed: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("[settleWithFacilitator] Response Status: %d\n", resp.StatusCode)

	bodyBytes, _ := io.ReadAll(resp.Body)
	fmt.Printf("[settleWithFacilitator] Response Body: %s\n", string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("facilitator returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var settleResp FacilitatorSettleResponse
	if err := json.Unmarshal(bodyBytes, &settleResp); err != nil {
		return "", fmt.Errorf("failed to decode facilitator response: %w", err)
	}

	if !settleResp.Success {
		return "", fmt.Errorf("settlement failed: %s", settleResp.InvalidReason)
	}

	fmt.Printf("[settleWithFacilitator] Settlement successful. TxHash: %s\n", settleResp.Transaction)
	return settleResp.Transaction, nil
}

func (s *PaymentService) settleWithFacilitatorWithRetry(ctx context.Context, header *PaymentHeader) (string, error) {
	const maxRetries = 5
	baseDelay := 500 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			fmt.Printf("[settleWithFacilitatorWithRetry] Attempt %d failed, retrying in %v...\n", attempt, delay)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}

		txHash, err := s.settleWithFacilitator(ctx, header)
		if err == nil {
			return txHash, nil
		}
		lastErr = err
		fmt.Printf("[settleWithFacilitatorWithRetry] Attempt %d/%d failed: %v\n", attempt+1, maxRetries, err)
	}

	return "", fmt.Errorf("settlement failed after %d attempts: %w", maxRetries, lastErr)
}

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

	value, ok := new(big.Int).SetString(auth.Value, 10)
	if !ok {
		return "", fmt.Errorf("invalid amount: %s", auth.Value)
	}

	validAfter, ok := new(big.Int).SetString(auth.ValidAfter, 10)
	if !ok {
		return "", fmt.Errorf("invalid validAfter: %s", auth.ValidAfter)
	}

	validBefore, ok := new(big.Int).SetString(auth.ValidBefore, 10)
	if !ok {
		return "", fmt.Errorf("invalid validBefore: %s", auth.ValidBefore)
	}

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

func generatePaymentID(payer, payee, amount, network string) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%d", payer, payee, amount, network, time.Now().UnixNano())
	hash := crypto.Keccak256Hash([]byte(data))
	return fmt.Sprintf("0x%x", hash)
}

func ParsePayment(data []byte) (*PaymentHeader, error) {
	var header PaymentHeader
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, err
	}
	return &header, nil
}

func (p *PaymentHeader) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}
