package marketplace

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestSafe4337DecodeCalldata(t *testing.T) {
	// Build a known calldata: executeUserOp(to, value, calldata, operation)
	// where calldata = transferFrom(from, to, value)
	safeABI, err := abi.JSON(strings.NewReader(safe4337ModuleABI))
	if err != nil {
		t.Fatal(err)
	}
	erc20ABI, err := abi.JSON(strings.NewReader(erc20TransferFromABI))
	if err != nil {
		t.Fatal(err)
	}

	from := common.HexToAddress("0x1111111111111111111111111111111111111111")
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	amount := big.NewInt(1000000) // 1 USDC

	// Build inner transferFrom calldata
	transferCalldata, err := erc20ABI.Pack("transferFrom", from, to, amount)
	if err != nil {
		t.Fatal(err)
	}

	// Build outer executeUserOp calldata
	tokenAddr := common.HexToAddress("0x036CbD53842c5426634e7929541eC2318f3dCF7e") // USDC
	outerCalldata, err := safeABI.Pack("executeUserOp", tokenAddr, big.NewInt(0), transferCalldata, uint8(0))
	if err != nil {
		t.Fatal(err)
	}

	// Now decode it
	f := &Safe4337Facilitator{
		safe4337ABI: safeABI,
		erc20ABI:    erc20ABI,
	}

	decodedTo, decodedAmount, err := f.decodeUserOpCalldata(hexutil.Encode(outerCalldata))
	if err != nil {
		t.Fatalf("decodeUserOpCalldata failed: %v", err)
	}

	if decodedTo != to {
		t.Errorf("expected to=%s, got %s", to.Hex(), decodedTo.Hex())
	}
	if decodedAmount.Cmp(amount) != 0 {
		t.Errorf("expected amount=%s, got %s", amount.String(), decodedAmount.String())
	}
}

func TestExtractUserOpHash(t *testing.T) {
	tests := []struct {
		name    string
		payload interface{}
		want    string
		wantErr bool
	}{
		{
			name:    "map[string]interface{}",
			payload: map[string]interface{}{"userOpHash": "0xabc123"},
			want:    "0xabc123",
		},
		{
			name:    "map[string]string",
			payload: map[string]string{"userOpHash": "0xdef456"},
			want:    "0xdef456",
		},
		{
			name:    "json.RawMessage",
			payload: json.RawMessage(`{"userOpHash":"0x789abc"}`),
			want:    "0x789abc",
		},
		{
			name:    "missing hash",
			payload: map[string]interface{}{"other": "value"},
			wantErr: true,
		},
		{
			name:    "unsupported type",
			payload: "string",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractUserOpHash(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractUserOpHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractUserOpHash() = %v, want %v", got, tt.want)
			}
		})
	}
}
