package config

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	p2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
)

// Config is the main configuration
type Config struct {
	P2P      *P2PConfig
	IPFS     *IPFSConfig
	Ethereum *EthereumConfig
	Agent    *AgentConfig
	Storage  *StorageConfig
}

// P2PConfig holds P2P configuration
type P2PConfig struct {
	Port           int
	BootstrapPeers []string
	EnableMDNS     bool
	EnableDHT      bool
	EnableRelay    bool
	PrivKey        p2pcrypto.PrivKey
	MinConnections int
	MaxConnections int
}

// IPFSConfig holds IPFS configuration
type IPFSConfig struct {
	APIURL string
}

// EthereumConfig holds Ethereum configuration
type EthereumConfig struct {
	RPCURL          string
	PrivateKey      string
	ChainID         int64
	RegistryAddr    string // AgentRegistry (ERC-721) contract address
	PaymentVaultAddr string // x402 PaymentVault contract address
	IdentityAddr    string // ERC-8004 IdentityRegistry contract address
	ReputationAddr  string // ReputationRegistry contract address
	ValidationAddr  string // ValidationRegistry contract address
}

// AgentConfig holds agent configuration
type AgentConfig struct {
	Model         string
	APIKey        string // Google API key (env: GOOGLE_API_KEY)
	Provider      string // "google", "openai", or "" for auto-detect (env: LLM_PROVIDER)
	OpenAIAPIKey  string // OpenAI-compatible API key (env: OPENAI_API_KEY)
	OpenAIBaseURL string // OpenAI-compatible base URL (env: OPENAI_BASE_URL)
}

// StorageConfig holds local persistent storage configuration
type StorageConfig struct {
	DataDir       string
	P2PKeyPath    string
	WalletKeyPath string
	SessionsDir   string // base dir for session store (NewStore appends /sessions)
}

// LoadConfig loads configuration from environment
func LoadConfig() (*Config, error) {
	cfg := &Config{
		P2P: &P2PConfig{
			Port:           4001,
			EnableMDNS:     true,
			EnableDHT:      true,
			EnableRelay:    getEnv("P2P_ENABLE_RELAY", "true") != "false",
			MinConnections: 2,
			MaxConnections: 10,
		},
		IPFS: &IPFSConfig{
			APIURL: getEnv("IPFS_API_URL", "http://localhost:5001"),
		},
		Ethereum: &EthereumConfig{
			RPCURL:           getEnv("ETHEREUM_RPC_URL", "https://sepolia.base.org"),
			PrivateKey:       getEnv("ETHEREUM_PRIVATE_KEY", ""),
			ChainID:          84532, // Base Sepolia
			RegistryAddr:     getEnv("AGENT_REGISTRY_ADDRESS", "0x81DdC4fAA728d555e44baAD65025067Ac7fcE903"),
			PaymentVaultAddr: getEnv("PAYMENT_VAULT_ADDRESS", "0x58E29Ab998C8c2ea456D29fe77C25fF67D44852b"),
			IdentityAddr:     getEnv("ERC8004_IDENTITY_ADDR", "0x8004A818BFB912233c491871b3d84c89A494BD9e"),
			ReputationAddr:   getEnv("REPUTATION_REGISTRY_ADDRESS", "0x36Cae8C9FD52B588c956f502f707CF27E063b702"),
			ValidationAddr:   getEnv("VALIDATION_REGISTRY_ADDRESS", "0xD0094DfEfC37f77e015D8A051fE6b7B885910757"),
		},
		Agent: &AgentConfig{
			Model:         getEnv("GOOGLE_MODEL", "gemini-2.5-flash"),
			APIKey:        getEnv("GOOGLE_API_KEY", ""),
			Provider:      getEnv("LLM_PROVIDER", ""),
			OpenAIAPIKey:  getEnv("OPENAI_API_KEY", ""),
			OpenAIBaseURL: getEnv("OPENAI_BASE_URL", ""),
		},
		Storage: &StorageConfig{},
	}

	dataDir := getEnv("BETAR_DATA_DIR", defaultDataDir())
	keyPath := getEnv("BETAR_P2P_KEY_PATH", filepath.Join(dataDir, "p2p_identity.key"))
	walletKeyPath := getEnv("BETAR_WALLET_KEY_PATH", filepath.Join(dataDir, "wallet.key"))

	cfg.Storage.DataDir = dataDir
	cfg.Storage.P2PKeyPath = keyPath
	cfg.Storage.WalletKeyPath = walletKeyPath
	cfg.Storage.SessionsDir = dataDir

	// If no private key from env, load or generate wallet key
	if cfg.Ethereum.PrivateKey == "" {
		walletKey, err := loadOrCreateWalletKey(cfg.Storage.WalletKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load or create wallet key: %w", err)
		}
		cfg.Ethereum.PrivateKey = walletKey
	}

	// Parse bootstrap peers
	bootstrapPeersStr := getEnv("BOOTSTRAP_PEERS", "")
	if bootstrapPeersStr != "" {
		cfg.P2P.BootstrapPeers = splitComma(bootstrapPeersStr)
	}

	privKey, err := loadOrCreateP2PIdentity(cfg.Storage.P2PKeyPath)
	if err != nil {
		return nil, err
	}
	cfg.P2P.PrivKey = privKey

	applyFileConfig(cfg)

	return cfg, nil
}

func applyFileConfig(cfg *Config) {
	fc, err := LoadFileConfig(FileConfigPath(cfg.Storage.DataDir))
	if err != nil || fc == nil {
		return
	}

	// LLM settings — only fill if env var didn't set them.
	// Provider
	if cfg.Agent.Provider == "" && fc.LLM.Provider != "" {
		cfg.Agent.Provider = fc.LLM.Provider
	}
	// API key — route to correct field based on provider
	provider := fc.LLM.Provider
	if provider == "openai" {
		if cfg.Agent.OpenAIAPIKey == "" && fc.LLM.APIKey != "" {
			cfg.Agent.OpenAIAPIKey = fc.LLM.APIKey
		}
		if cfg.Agent.OpenAIBaseURL == "" && fc.LLM.BaseURL != "" {
			cfg.Agent.OpenAIBaseURL = fc.LLM.BaseURL
		}
	} else {
		if cfg.Agent.APIKey == "" && fc.LLM.APIKey != "" {
			cfg.Agent.APIKey = fc.LLM.APIKey
		}
	}
	// Model — only apply if env var GOOGLE_MODEL was not set
	if os.Getenv("GOOGLE_MODEL") == "" && fc.LLM.Model != "" {
		cfg.Agent.Model = fc.LLM.Model
	}

	// Wallet — RPC URL only (private key is managed via wallet.key file)
	if fc.Wallet.RPCURL != "" && os.Getenv("ETHEREUM_RPC_URL") == "" {
		cfg.Ethereum.RPCURL = fc.Wallet.RPCURL
	}

	// P2P
	if fc.P2P.Port != 0 {
		cfg.P2P.Port = fc.P2P.Port
	}
	if len(cfg.P2P.BootstrapPeers) == 0 && len(fc.P2P.BootstrapPeers) > 0 {
		cfg.P2P.BootstrapPeers = fc.P2P.BootstrapPeers
	}
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".betar"
	}
	return filepath.Join(home, ".betar")
}

func loadOrCreateP2PIdentity(path string) (p2pcrypto.PrivKey, error) {
	if path == "" {
		return nil, fmt.Errorf("empty p2p key path")
	}

	keyBytes, err := os.ReadFile(path)
	if err == nil {
		pk, err := p2pcrypto.UnmarshalPrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse p2p private key at %s: %w", path, err)
		}
		return pk, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read p2p private key at %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("failed to create data dir for p2p key: %w", err)
	}

	pk, _, err := p2pcrypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate p2p private key: %w", err)
	}

	encoded, err := p2pcrypto.MarshalPrivateKey(pk)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal p2p private key: %w", err)
	}

	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return nil, fmt.Errorf("failed to persist p2p private key at %s: %w", path, err)
	}

	return pk, nil
}

func loadOrCreateWalletKey(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty wallet key path")
	}

	keyBytes, err := os.ReadFile(path)
	if err == nil {
		keyHex := strings.TrimSpace(string(keyBytes))

		if _, err := crypto.HexToECDSA(keyHex); err != nil {
			return "", fmt.Errorf("invalid wallet key at %s: %w", path, err)
		}

		return keyHex, nil
	}

	if !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read wallet key at %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("failed to create data dir: %w", err)
	}

	privateKey, err := crypto.GenerateKey() // secp256k1
	if err != nil {
		return "", fmt.Errorf("failed to generate wallet key: %w", err)
	}

	keyHex := hex.EncodeToString(crypto.FromECDSA(privateKey))

	if err := os.WriteFile(path, []byte(keyHex+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("failed to persist wallet key: %w", err)
	}

	return keyHex, nil
}

func GetAddressFromKey(keyHex string) (string, error) {
	keyHex = strings.TrimSpace(keyHex)

	privateKey, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %w", err)
	}

	publicKey, ok := privateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("invalid public key type")
	}

	address := crypto.PubkeyToAddress(*publicKey)
	return address.Hex(), nil
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func splitComma(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, part := range split(s, ",") {
		if part = trim(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func split(s, sep string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
