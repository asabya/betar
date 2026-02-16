package eth

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Wallet manages Ethereum wallet operations
type Wallet struct {
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
	address    common.Address
	client     *ethclient.Client
	chainID    *big.Int
}

// NewWallet creates a new wallet from private key
func NewWallet(privateKeyHex string, rpcURL string) (*Wallet, error) {
	// Parse private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	// Derive public key and address
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	address := crypto.PubkeyToAddress(*publicKey)

	// Connect to Ethereum node
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum: %w", err)
	}

	// Get chain ID
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	return &Wallet{
		privateKey: privateKey,
		publicKey:  publicKey,
		address:    address,
		client:     client,
		chainID:    chainID,
	}, nil
}

// Address returns the wallet address
func (w *Wallet) Address() common.Address {
	return w.address
}

// AddressHex returns the wallet address as hex string
func (w *Wallet) AddressHex() string {
	return w.address.Hex()
}

// PrivateKeyHex returns the private key as hex string (without 0x prefix)
func (w *Wallet) PrivateKeyHex() string {
	return hex.EncodeToString(crypto.FromECDSA(w.privateKey))
}

// ChainID returns the chain ID
func (w *Wallet) ChainID() *big.Int {
	return w.chainID
}

// Client returns the Ethereum client
func (w *Wallet) Client() *ethclient.Client {
	return w.client
}

// GetTransactor creates a transactor for contract calls
func (w *Wallet) GetTransactor(ctx context.Context) (*bind.TransactOpts, error) {
	transactor, err := bind.NewKeyedTransactorWithChainID(w.privateKey, w.chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	nonce, err := w.client.PendingNonceAt(ctx, w.address)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}
	transactor.Nonce = big.NewInt(int64(nonce))

	gasPrice, err := w.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}
	transactor.GasPrice = gasPrice
	transactor.GasLimit = 3000000
	transactor.Context = ctx

	return transactor, nil
}

// GetCallOpts creates call options for read-only contract calls
func (w *Wallet) GetCallOpts(ctx context.Context) *bind.CallOpts {
	return &bind.CallOpts{
		From:    w.address,
		Context: ctx,
	}
}

// Balance returns the ETH balance of the wallet
func (w *Wallet) Balance(ctx context.Context) (*big.Int, error) {
	return w.client.BalanceAt(ctx, w.address, nil)
}

// PendingBalance returns the pending ETH balance of the wallet
func (w *Wallet) PendingBalance(ctx context.Context) (*big.Int, error) {
	return w.client.PendingBalanceAt(ctx, w.address)
}

// Transfer transfers ETH to an address
func (w *Wallet) Transfer(ctx context.Context, to common.Address, amount *big.Int) (common.Hash, error) {
	nonce, err := w.client.PendingNonceAt(ctx, w.address)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := w.client.SuggestGasPrice(ctx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get gas price: %w", err)
	}

	tx := gethtypes.NewTx(&gethtypes.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    amount,
		Gas:      21000,
		GasPrice: gasPrice,
	})

	signedTx, err := gethtypes.SignTx(tx, gethtypes.NewEIP155Signer(w.chainID), w.privateKey)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = w.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to send transaction: %w", err)
	}

	return signedTx.Hash(), nil
}

// WaitForTransaction waits for a transaction to be mined
func (w *Wallet) WaitForTransaction(ctx context.Context, hash common.Hash) (*gethtypes.Receipt, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		receipt, err := w.client.TransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// GenerateKey generates a new random private key
func GenerateKey() (*ecdsa.PrivateKey, error) {
	return crypto.GenerateKey()
}

// PrivateKeyToAddress derives address from private key
func PrivateKeyToAddress(privateKey *ecdsa.PrivateKey) common.Address {
	return crypto.PubkeyToAddress(*privateKey.Public().(*ecdsa.PublicKey))
}

// EtherToWei converts Ether to Wei
func EtherToWei(ether float64) *big.Int {
	value := new(big.Float).Mul(big.NewFloat(ether), big.NewFloat(1e18))
	wei := new(big.Int)
	value.Int(wei)
	return wei
}

// WeiToEther converts Wei to Ether
func WeiToEther(wei *big.Int) float64 {
	f := new(big.Float).SetInt(wei)
	f.Quo(f, new(big.Float).SetInt64(1e18))
	e, _ := f.Float64()
	return e
}
