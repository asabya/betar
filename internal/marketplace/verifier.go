package marketplace

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type PaymentVerifier struct {
	network      string
	usedNonces   map[string]bool
	nonceTracker map[string]int64
	mu           sync.RWMutex
}

func NewPaymentVerifier(network string) *PaymentVerifier {
	return &PaymentVerifier{
		network:      network,
		usedNonces:   make(map[string]bool),
		nonceTracker: make(map[string]int64),
	}
}

func (v *PaymentVerifier) VerifyEIP712Signature(header *PaymentHeader) error {
	if header == nil {
		return fmt.Errorf("payment header is nil")
	}

	if header.Payload == nil {
		return fmt.Errorf("payment payload is nil")
	}

	auth := header.Payload.Authorization
	sig := header.Payload.Signature

	if sig == "" {
		return fmt.Errorf("signature is empty")
	}

	if auth.From == "" {
		return fmt.Errorf("authorization 'from' address is empty")
	}

	chainID := v.getChainID()
	usdcAddr := common.HexToAddress(GetUSDCAddress(v.network))

	typedData := v.buildTypedData(usdcAddr, chainID, &auth)

	digest, err := v.computeDigest(typedData)
	if err != nil {
		return fmt.Errorf("failed to compute digest: %w", err)
	}

	sigBytes, err := hexDecode(sig)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	if len(sigBytes) != 65 {
		return fmt.Errorf("invalid signature length: expected 65 bytes, got %d", len(sigBytes))
	}

	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}

	pubKey, err := crypto.SigToPub(digest, sigBytes)
	if err != nil {
		return fmt.Errorf("failed to recover public key: %w", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	expectedAddr := common.HexToAddress(auth.From)

	if recoveredAddr != expectedAddr {
		return fmt.Errorf("signature mismatch: recovered %s, expected %s", recoveredAddr.Hex(), expectedAddr.Hex())
	}

	return nil
}

func (v *PaymentVerifier) buildTypedData(tokenAddr common.Address, chainID *big.Int, auth *EVMAuthorization) apitypes.TypedData {
	value, _ := new(big.Int).SetString(auth.Value, 10)
	validAfter, _ := new(big.Int).SetString(auth.ValidAfter, 10)
	validBefore, _ := new(big.Int).SetString(auth.ValidBefore, 10)

	return apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"TransferWithAuthorization": []apitypes.Type{
				{Name: "from", Type: "address"},
				{Name: "to", Type: "address"},
				{Name: "value", Type: "uint256"},
				{Name: "validAfter", Type: "uint256"},
				{Name: "validBefore", Type: "uint256"},
				{Name: "nonce", Type: "bytes32"},
			},
		},
		PrimaryType: "TransferWithAuthorization",
		Domain: apitypes.TypedDataDomain{
			Name:              "USDC",
			Version:           "2",
			ChainId:           math.NewHexOrDecimal256(chainID.Int64()),
			VerifyingContract: tokenAddr.Hex(),
		},
		Message: apitypes.TypedDataMessage{
			"from":        auth.From,
			"to":          auth.To,
			"value":       math.NewHexOrDecimal256(value.Int64()),
			"validAfter":  math.NewHexOrDecimal256(validAfter.Int64()),
			"validBefore": math.NewHexOrDecimal256(validBefore.Int64()),
			"nonce":       auth.Nonce,
		},
	}
}

func (v *PaymentVerifier) computeDigest(typedData apitypes.TypedData) ([]byte, error) {
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, fmt.Errorf("failed to hash domain: %w", err)
	}

	messageHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to hash message: %w", err)
	}

	rawData := append([]byte("\x19\x01"), domainSeparator...)
	rawData = append(rawData, messageHash...)

	return crypto.Keccak256(rawData), nil
}

func (v *PaymentVerifier) getChainID() *big.Int {
	switch v.network {
	case NetworkBaseSepolia:
		return big.NewInt(84532)
	case NetworkBaseMainnet:
		return big.NewInt(8453)
	default:
		return big.NewInt(84532)
	}
}

func hexDecode(s string) ([]byte, error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}
	if len(s)%2 != 0 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}

func (v *PaymentVerifier) CheckAndMarkNonce(nonce string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.usedNonces[nonce] {
		return fmt.Errorf("nonce already used: %s", nonce)
	}

	v.usedNonces[nonce] = true
	v.nonceTracker[nonce] = time.Now().Unix()
	return nil
}

func (v *PaymentVerifier) CleanupOldNonces(maxAgeSeconds int64) {
	v.mu.Lock()
	defer v.mu.Unlock()

	now := time.Now().Unix()
	for nonce, timestamp := range v.nonceTracker {
		if now-timestamp > maxAgeSeconds {
			delete(v.usedNonces, nonce)
			delete(v.nonceTracker, nonce)
		}
	}
}

func (v *PaymentVerifier) ValidateTimestamps(auth *EVMAuthorization) error {
	validAfter, ok := new(big.Int).SetString(auth.ValidAfter, 10)
	if !ok {
		return fmt.Errorf("invalid validAfter: %s", auth.ValidAfter)
	}

	validBefore, ok := new(big.Int).SetString(auth.ValidBefore, 10)
	if !ok {
		return fmt.Errorf("invalid validBefore: %s", auth.ValidBefore)
	}

	now := time.Now().Unix()

	if now < validAfter.Int64() {
		return fmt.Errorf("authorization not yet valid: now=%d, validAfter=%d", now, validAfter.Int64())
	}

	if now > validBefore.Int64() {
		return fmt.Errorf("authorization expired: now=%d, validBefore=%d", now, validBefore.Int64())
	}

	return nil
}

func (v *PaymentVerifier) ValidateTransactionData(header *PaymentHeader, txData []byte) error {
	if header == nil {
		return fmt.Errorf("payment header is nil")
	}

	if header.Payload == nil {
		return fmt.Errorf("payment payload is nil")
	}

	minLen := 4 + 32*6
	if len(txData) < minLen {
		return fmt.Errorf("transaction data too short: expected at least %d bytes, got %d", minLen, len(txData))
	}

	expectedSelector := crypto.Keccak256([]byte("transferWithAuthorization(address,address,uint256,uint256,uint256,bytes32,bytes)"))[:4]
	selector := txData[0:4]
	if !equalBytes(selector, expectedSelector) {
		return fmt.Errorf("method selector mismatch: got 0x%s, expected 0x%s", hex.EncodeToString(selector), hex.EncodeToString(expectedSelector))
	}

	auth := header.Payload.Authorization

	fromAddr := common.BytesToAddress(txData[4+12 : 36])
	expectedFrom := common.HexToAddress(auth.From)
	if fromAddr != expectedFrom {
		return fmt.Errorf("from address mismatch: got %s, expected %s", fromAddr.Hex(), expectedFrom.Hex())
	}

	toAddr := common.BytesToAddress(txData[36+12 : 68])
	expectedTo := common.HexToAddress(auth.To)
	if toAddr != expectedTo {
		return fmt.Errorf("to address mismatch: got %s, expected %s", toAddr.Hex(), expectedTo.Hex())
	}

	value := new(big.Int).SetBytes(txData[68:100])
	expectedValue, ok := new(big.Int).SetString(auth.Value, 10)
	if !ok {
		return fmt.Errorf("invalid authorization value: %s", auth.Value)
	}
	if value.Cmp(expectedValue) != 0 {
		return fmt.Errorf("value mismatch: got %s, expected %s", value.String(), expectedValue.String())
	}

	validAfter := new(big.Int).SetBytes(txData[100:132])
	expectedValidAfter, ok := new(big.Int).SetString(auth.ValidAfter, 10)
	if !ok {
		return fmt.Errorf("invalid authorization validAfter: %s", auth.ValidAfter)
	}
	if validAfter.Cmp(expectedValidAfter) != 0 {
		return fmt.Errorf("validAfter mismatch: got %s, expected %s", validAfter.String(), expectedValidAfter.String())
	}

	validBefore := new(big.Int).SetBytes(txData[132:164])
	expectedValidBefore, ok := new(big.Int).SetString(auth.ValidBefore, 10)
	if !ok {
		return fmt.Errorf("invalid authorization validBefore: %s", auth.ValidBefore)
	}
	if validBefore.Cmp(expectedValidBefore) != 0 {
		return fmt.Errorf("validBefore mismatch: got %s, expected %s", validBefore.String(), expectedValidBefore.String())
	}

	nonce := txData[164:196]
	expectedNonce, err := hexDecode(auth.Nonce)
	if err != nil {
		return fmt.Errorf("invalid authorization nonce: %s", auth.Nonce)
	}
	if !equalBytes(nonce, expectedNonce) {
		return fmt.Errorf("nonce mismatch: got 0x%s, expected 0x%s", hex.EncodeToString(nonce), hex.EncodeToString(expectedNonce))
	}

	return nil
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (v *PaymentVerifier) ValidatePaymentHeader(header *PaymentHeader, expectedPayTo string, expectedAmount *big.Int) error {
	if header == nil {
		return fmt.Errorf("payment header is nil")
	}

	if header.Requirement.Scheme != "exact" {
		return fmt.Errorf("invalid scheme: expected 'exact', got '%s'", header.Requirement.Scheme)
	}

	if header.Requirement.Network != v.network {
		return fmt.Errorf("network mismatch: expected '%s', got '%s'", v.network, header.Requirement.Network)
	}

	if header.Signature == "" {
		return fmt.Errorf("signature is empty")
	}

	if header.Requirement.PayTo != expectedPayTo {
		return fmt.Errorf("payTo mismatch: expected '%s', got '%s'", expectedPayTo, header.Requirement.PayTo)
	}

	maxAmount, ok := new(big.Int).SetString(header.Requirement.Amount, 10)
	if !ok {
		return fmt.Errorf("invalid amount: %s", header.Requirement.Amount)
	}

	if maxAmount.Cmp(expectedAmount) != 0 {
		return fmt.Errorf("amount mismatch: expected '%s', got '%s'", expectedAmount.String(), maxAmount.String())
	}

	expectedAsset := GetUSDCAddress(v.network)
	if !strings.EqualFold(header.Requirement.Asset, expectedAsset) {
		return fmt.Errorf("asset mismatch: expected '%s', got '%s'", expectedAsset, header.Requirement.Asset)
	}

	return nil
}

func (v *PaymentVerifier) ValidatePayerConsistency(header *PaymentHeader) error {
	if header == nil {
		return fmt.Errorf("payment header is nil")
	}

	if header.Payload == nil {
		return fmt.Errorf("payment payload is nil")
	}

	if header.Payer != header.Payload.Authorization.From {
		return fmt.Errorf("payer mismatch: header.Payer='%s', authorization.From='%s'", header.Payer, header.Payload.Authorization.From)
	}

	return nil
}

func (v *PaymentVerifier) ValidatePayment(header *PaymentHeader, expectedPayTo string, expectedAmount *big.Int) error {
	if err := v.ValidatePaymentHeader(header, expectedPayTo, expectedAmount); err != nil {
		return err
	}

	if err := v.ValidatePayerConsistency(header); err != nil {
		return err
	}

	if err := v.ValidateTimestamps(&header.Payload.Authorization); err != nil {
		return err
	}

	if err := v.CheckAndMarkNonce(header.Payload.Authorization.Nonce); err != nil {
		return err
	}

	if err := v.VerifyEIP712Signature(header); err != nil {
		return err
	}

	return nil
}
