package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// SAFE 4337 module ABI fragments (only the parts we need for decoding).
const safe4337ModuleABI = `[{"name":"executeUserOp","inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"calldata","type":"bytes"},{"name":"operation","type":"uint8"}],"outputs":[],"type":"function"}]`

const erc20TransferFromABI = `[{"name":"transferFrom","inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"outputs":[{"name":"","type":"bool"}],"type":"function"}]`

// Function selectors
const (
	executeUserOpSelector = "0x7bb37428"
	transferFromSelector  = "0x23b872dd"
)

// Safe4337Facilitator verifies SAFE UserOperation payments on-chain.
type Safe4337Facilitator struct {
	rpcURL       string
	rpcClient    *rpc.Client
	safe4337ABI  abi.ABI
	erc20ABI     abi.ABI
}

// NewSafe4337Facilitator creates a new SAFE 4337 facilitator connected to the given RPC.
func NewSafe4337Facilitator(rpcURL string) (*Safe4337Facilitator, error) {
	client, err := rpc.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	safeABI, err := abi.JSON(strings.NewReader(safe4337ModuleABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SAFE 4337 ABI: %w", err)
	}

	erc20, err := abi.JSON(strings.NewReader(erc20TransferFromABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ERC-20 ABI: %w", err)
	}

	return &Safe4337Facilitator{
		rpcURL:      rpcURL,
		rpcClient:   client,
		safe4337ABI: safeABI,
		erc20ABI:    erc20,
	}, nil
}

func (f *Safe4337Facilitator) Scheme() string { return "safe-4337" }

// userOpResult represents the result of eth_getUserOperationByHash.
type userOpResult struct {
	UserOperation struct {
		Sender   string `json:"sender"`
		CallData string `json:"callData"`
		Nonce    string `json:"nonce"`
	} `json:"userOperation"`
	EntryPoint string `json:"entryPoint"`
	BlockHash  string `json:"blockHash"`
	TxHash     string `json:"transactionHash"`
}

func (f *Safe4337Facilitator) Verify(ctx context.Context, payload *FacilitatorPayload) (*FacilitatorVerifyResult, error) {
	userOpHash, err := extractUserOpHash(payload.PaymentPayload)
	if err != nil {
		return &FacilitatorVerifyResult{IsValid: false, InvalidReason: err.Error()}, nil
	}

	// Query eth_getUserOperationByHash
	var result userOpResult
	err = f.rpcClient.CallContext(ctx, &result, "eth_getUserOperationByHash", userOpHash)
	if err != nil {
		return nil, fmt.Errorf("eth_getUserOperationByHash failed: %w", err)
	}

	if result.UserOperation.CallData == "" {
		return &FacilitatorVerifyResult{
			IsValid:       false,
			InvalidReason: "UserOperation not found or pending",
		}, nil
	}

	// Decode the SAFE 4337 module calldata
	to, amount, err := f.decodeUserOpCalldata(result.UserOperation.CallData)
	if err != nil {
		return &FacilitatorVerifyResult{
			IsValid:       false,
			InvalidReason: fmt.Sprintf("failed to decode UserOp calldata: %v", err),
		}, nil
	}

	// Validate: recipient must match payTo
	expectedPayTo := common.HexToAddress(payload.PaymentRequirements.PayTo)
	if to != expectedPayTo {
		return &FacilitatorVerifyResult{
			IsValid:       false,
			InvalidReason: fmt.Sprintf("recipient mismatch: expected %s, got %s", expectedPayTo.Hex(), to.Hex()),
		}, nil
	}

	// Validate: amount >= required
	requiredAmount, ok := new(big.Int).SetString(payload.PaymentRequirements.Amount, 10)
	if !ok {
		return &FacilitatorVerifyResult{
			IsValid:       false,
			InvalidReason: fmt.Sprintf("invalid required amount: %s", payload.PaymentRequirements.Amount),
		}, nil
	}
	if amount.Cmp(requiredAmount) < 0 {
		return &FacilitatorVerifyResult{
			IsValid:       false,
			InvalidReason: fmt.Sprintf("amount too low: got %s, need %s", amount.String(), requiredAmount.String()),
		}, nil
	}

	return &FacilitatorVerifyResult{
		IsValid: true,
		Payer:   result.UserOperation.Sender,
	}, nil
}

func (f *Safe4337Facilitator) Settle(ctx context.Context, payload *FacilitatorPayload) (*FacilitatorSettleResult, error) {
	// For SAFE 4337, the UserOp IS the settlement (already submitted by the client).
	// We just need to verify it and return the transaction hash.
	userOpHash, err := extractUserOpHash(payload.PaymentPayload)
	if err != nil {
		return &FacilitatorSettleResult{Success: false, ErrorReason: err.Error()}, nil
	}

	// Query eth_getUserOperationByHash to get the actual transaction hash.
	var result userOpResult
	err = f.rpcClient.CallContext(ctx, &result, "eth_getUserOperationByHash", userOpHash)
	if err != nil {
		return nil, fmt.Errorf("eth_getUserOperationByHash failed: %w", err)
	}

	if result.TxHash == "" {
		return &FacilitatorSettleResult{
			Success:     false,
			ErrorReason: "UserOperation not yet included in a block",
		}, nil
	}

	// Verify the calldata before accepting
	verifyResult, err := f.Verify(ctx, payload)
	if err != nil {
		return nil, err
	}
	if !verifyResult.IsValid {
		return &FacilitatorSettleResult{
			Success:     false,
			ErrorReason: verifyResult.InvalidReason,
		}, nil
	}

	return &FacilitatorSettleResult{
		Success: true,
		TxHash:  result.TxHash,
	}, nil
}

// decodeUserOpCalldata decodes the SAFE 4337 module calldata to extract the
// ERC-20 transfer recipient and amount.
// Expected structure: executeUserOp(to, value, calldata, operation)
//   where calldata is transferFrom(from, to, value)
func (f *Safe4337Facilitator) decodeUserOpCalldata(calldataHex string) (common.Address, *big.Int, error) {
	calldata, err := hexutil.Decode(calldataHex)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to decode calldata hex: %w", err)
	}

	if len(calldata) < 4 {
		return common.Address{}, nil, fmt.Errorf("calldata too short")
	}

	// Check for executeUserOp selector
	selector := hexutil.Encode(calldata[:4])
	if selector != executeUserOpSelector {
		return common.Address{}, nil, fmt.Errorf("unexpected selector: %s (expected executeUserOp %s)", selector, executeUserOpSelector)
	}

	// Decode executeUserOp arguments
	method, err := f.safe4337ABI.MethodById(calldata[:4])
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to find method: %w", err)
	}

	args, err := method.Inputs.Unpack(calldata[4:])
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to unpack executeUserOp args: %w", err)
	}

	if len(args) < 3 {
		return common.Address{}, nil, fmt.Errorf("executeUserOp has %d args, expected at least 3", len(args))
	}

	// args[2] is the inner calldata (bytes)
	innerCalldata, ok := args[2].([]byte)
	if !ok {
		return common.Address{}, nil, fmt.Errorf("inner calldata is not bytes")
	}

	if len(innerCalldata) < 4 {
		return common.Address{}, nil, fmt.Errorf("inner calldata too short")
	}

	// Check for transferFrom selector
	innerSelector := hexutil.Encode(innerCalldata[:4])
	if innerSelector != transferFromSelector {
		return common.Address{}, nil, fmt.Errorf("unexpected inner selector: %s (expected transferFrom %s)", innerSelector, transferFromSelector)
	}

	// Decode transferFrom arguments
	transferMethod, err := f.erc20ABI.MethodById(innerCalldata[:4])
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to find transferFrom method: %w", err)
	}

	transferArgs, err := transferMethod.Inputs.Unpack(innerCalldata[4:])
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("failed to unpack transferFrom args: %w", err)
	}

	if len(transferArgs) < 3 {
		return common.Address{}, nil, fmt.Errorf("transferFrom has %d args, expected 3", len(transferArgs))
	}

	// transferFrom(from, to, value) — we want to and value
	to, ok := transferArgs[1].(common.Address)
	if !ok {
		return common.Address{}, nil, fmt.Errorf("transferFrom 'to' is not address")
	}

	value, ok := transferArgs[2].(*big.Int)
	if !ok {
		return common.Address{}, nil, fmt.Errorf("transferFrom 'value' is not *big.Int")
	}

	return to, value, nil
}

// extractUserOpHash extracts the userOpHash from the payment payload.
// The payload is expected to be a map with a "userOpHash" key, or a json.RawMessage.
func extractUserOpHash(payload interface{}) (string, error) {
	switch p := payload.(type) {
	case map[string]interface{}:
		hash, ok := p["userOpHash"].(string)
		if !ok || hash == "" {
			return "", fmt.Errorf("missing userOpHash in payload")
		}
		return hash, nil
	case map[string]string:
		hash, ok := p["userOpHash"]
		if !ok || hash == "" {
			return "", fmt.Errorf("missing userOpHash in payload")
		}
		return hash, nil
	case json.RawMessage:
		var m map[string]string
		if err := json.Unmarshal(p, &m); err != nil {
			return "", fmt.Errorf("failed to unmarshal payload: %w", err)
		}
		hash, ok := m["userOpHash"]
		if !ok || hash == "" {
			return "", fmt.Errorf("missing userOpHash in payload")
		}
		return hash, nil
	default:
		return "", fmt.Errorf("unsupported payload type for safe-4337: %T", payload)
	}
}
