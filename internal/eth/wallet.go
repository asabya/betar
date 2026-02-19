package eth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/crypto/pbkdf2"
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

// ChainID returns the chain ID
func (w *Wallet) ChainID() *big.Int {
	return w.chainID
}

// PrivateKeyHex returns the private key as hex string
func (w *Wallet) PrivateKeyHex() string {
	return hex.EncodeToString(w.privateKey.D.Bytes())
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

// GetTransaction retrieves a transaction by hash
func (w *Wallet) GetTransaction(ctx context.Context, hash common.Hash) (*gethtypes.Transaction, bool, error) {
	tx, isPending, err := w.client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get transaction: %w", err)
	}
	return tx, isPending, nil
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

// EncryptedWallet represents an encrypted wallet file
type EncryptedWallet struct {
	Address    string `json:"address"`
	Ciphertext string `json:"ciphertext"`
	Salt       string `json:"salt"`
	IV         string `json:"iv"`
}

// encrypt encrypts data with password using PBKDF2 and AES-GCM
func encrypt(data []byte, password string) (ciphertext, salt, iv []byte, err error) {
	// Generate salt
	salt = make([]byte, 32)
	if _, err = rand.Read(salt); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)

	// Generate IV
	iv = make([]byte, 12)
	if _, err = rand.Read(iv); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Encrypt using GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext = gcm.Seal(nil, iv, data, nil)
	return ciphertext, salt, iv, nil
}

// decrypt decrypts ciphertext with password using PBKDF2 and AES-GCM
func decrypt(ciphertext, salt, iv []byte, password string) ([]byte, error) {
	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Decrypt using GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// LoadOrCreateWallet loads existing wallet or creates new one
// walletPath: path to wallet file (e.g., ~/.betar/wallet)
// password: encryption password
// rpcURL: Ethereum RPC endpoint
func LoadOrCreateWallet(walletPath, password, rpcURL string) (*Wallet, error) {
	// Ensure directory exists
	dir := filepath.Dir(walletPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create wallet directory: %w", err)
	}

	// Check if wallet exists
	if _, err := os.Stat(walletPath); err == nil {
		// Load existing wallet
		data, err := os.ReadFile(walletPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read wallet file: %w", err)
		}

		// Parse encrypted wallet (simple format: hex encoded)
		if len(data) < 64 {
			return nil, fmt.Errorf("invalid wallet file format")
		}

		// Format: salt(64 hex) + iv(24 hex) + ciphertext
		salt, err := hex.DecodeString(string(data[:64]))
		if err != nil {
			return nil, fmt.Errorf("failed to decode salt: %w", err)
		}

		iv, err := hex.DecodeString(string(data[64:88]))
		if err != nil {
			return nil, fmt.Errorf("failed to decode IV: %w", err)
		}

		ciphertext, err := hex.DecodeString(string(data[88:]))
		if err != nil {
			return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
		}

		// Decrypt private key
		privateKeyHex, err := decrypt(ciphertext, salt, iv, password)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt wallet (wrong password?): %w", err)
		}

		return NewWallet(string(privateKeyHex), rpcURL)
	}

	// Create new wallet
	privateKey, err := GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Encode private key
	privateKeyBytes := crypto.FromECDSA(privateKey)
	privateKeyHex := hex.EncodeToString(privateKeyBytes)

	// Encrypt private key
	ciphertext, salt, iv, err := encrypt([]byte(privateKeyHex), password)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt wallet: %w", err)
	}

	// Save wallet file
	data := hex.EncodeToString(salt) + hex.EncodeToString(iv) + hex.EncodeToString(ciphertext)
	if err := os.WriteFile(walletPath, []byte(data), 0600); err != nil {
		return nil, fmt.Errorf("failed to save wallet: %w", err)
	}

	return NewWallet(privateKeyHex, rpcURL)
}

// Sign signs data with the wallet's private key
func (w *Wallet) Sign(data []byte) ([]byte, error) {
	hash := crypto.Keccak256Hash(data)
	signature, err := crypto.Sign(hash.Bytes(), w.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}
	return signature, nil
}

// ERC20Balance returns the balance of an ERC20 token for the wallet
func (w *Wallet) ERC20Balance(ctx context.Context, tokenAddress common.Address) (*big.Int, error) {
	// Standard ERC20 balanceOf call
	// We need to create the call data for balanceOf(address)
	// balanceOf signature: 0x70a08231
	// balanceOf(address): 0x70a08231 + 000000000000000000000000 + address

	callData := make([]byte, 36)
	copy(callData[:4], []byte{0x70, 0xa0, 0x82, 0x31}) // balanceOf selector
	copy(callData[4:], w.address.Bytes())

	result, err := w.client.CallContract(ctx, ethereum.CallMsg{
		To:   &tokenAddress,
		Data: callData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call balanceOf: %w", err)
	}

	if len(result) == 0 {
		return big.NewInt(0), nil
	}

	balance := new(big.Int).SetBytes(result)
	return balance, nil
}

// TransferERC20 transfers ERC20 tokens to an address
func (w *Wallet) TransferERC20(ctx context.Context, tokenAddress, to common.Address, amount *big.Int) (common.Hash, error) {
	// Create the transaction data for transfer(to, amount)
	// transfer(address,uint256) selector: 0xa9059cbb

	callData := make([]byte, 68)
	copy(callData[:4], []byte{0xa9, 0x05, 0x9c, 0xbb}) // transfer selector
	copy(callData[4:24], to.Bytes())
	copy(callData[36:], amount.Bytes())

	// Get nonce
	nnonce, err := w.client.PendingNonceAt(ctx, w.address)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := w.client.SuggestGasPrice(ctx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Create transaction
	tx := gethtypes.NewTx(&gethtypes.LegacyTx{
		Nonce:    nnonce,
		To:       &tokenAddress,
		Value:    big.NewInt(0),
		Gas:      65000, // ERC20 transfer gas limit
		GasPrice: gasPrice,
		Data:     callData,
	})

	// Sign transaction
	signedTx, err := gethtypes.SignTx(tx, gethtypes.NewEIP155Signer(w.chainID), w.privateKey)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	err = w.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to send transaction: %w", err)
	}

	return signedTx.Hash(), nil
}

// SubmitTransaction submits a generic transaction to a contract
func (w *Wallet) SubmitTransaction(ctx context.Context, to common.Address, value *big.Int, data []byte) (common.Hash, error) {
	// Get nonce
	nonce, err := w.client.PendingNonceAt(ctx, w.address)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := w.client.SuggestGasPrice(ctx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Estimate gas
	gasLimit, err := w.client.EstimateGas(ctx, ethereum.CallMsg{
		From:  w.address,
		To:    &to,
		Value: value,
		Data:  data,
	})
	if err != nil {
		gasLimit = 100000 // fallback
	}

	// Create transaction
	tx := gethtypes.NewTx(&gethtypes.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    value,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})

	// Sign transaction
	signedTx, err := gethtypes.SignTx(tx, gethtypes.NewEIP155Signer(w.chainID), w.privateKey)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	err = w.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to send transaction: %w", err)
	}

	return signedTx.Hash(), nil
}
