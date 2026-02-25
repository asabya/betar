package marketplace

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var usdcABI abi.ABI

const usdcABIJSON = `[
  {
    "type": "function",
    "name": "transferWithAuthorization",
    "inputs": [
      {"name": "from", "type": "address"},
      {"name": "to", "type": "address"},
      {"name": "value", "type": "uint256"},
      {"name": "validAfter", "type": "uint256"},
      {"name": "validBefore", "type": "uint256"},
      {"name": "nonce", "type": "bytes32"},
      {"name": "signature", "type": "bytes"}
    ],
    "outputs": [],
    "stateMutability": "nonpayable"
  },
  {
    "type": "function",
    "name": "balanceOf",
    "inputs": [{"name": "account", "type": "address"}],
    "outputs": [{"name": "", "type": "uint256"}],
    "stateMutability": "view"
  },
  {
    "type": "function",
    "name": "decimals",
    "inputs": [],
    "outputs": [{"name": "", "type": "uint8"}],
    "stateMutability": "view"
  }
]`

func init() {
	var err error
	usdcABI, err = abi.JSON(strings.NewReader(usdcABIJSON))
	if err != nil {
		panic("failed to parse USDC ABI: " + err.Error())
	}
}
