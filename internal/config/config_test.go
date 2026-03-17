package config

import (
	"path/filepath"
	"testing"

	p2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
)

func TestAgentConfigNewProviderFields(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("OPENAI_BASE_URL", "http://localhost:11434/v1/")
	t.Setenv("BETAR_P2P_KEY_PATH", t.TempDir()+"/p2p.key")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Agent.Provider != "openai" {
		t.Fatalf("Provider not loaded: %q", cfg.Agent.Provider)
	}
	if cfg.Agent.OpenAIAPIKey != "sk-test" {
		t.Fatalf("OpenAIAPIKey not loaded: %q", cfg.Agent.OpenAIAPIKey)
	}
	if cfg.Agent.OpenAIBaseURL != "http://localhost:11434/v1/" {
		t.Fatalf("OpenAIBaseURL not loaded: %q", cfg.Agent.OpenAIBaseURL)
	}
}

func TestLoadConfigPersistsP2PIdentity(t *testing.T) {
	t.Setenv("BOOTSTRAP_PEERS", "")

	keyPath := filepath.Join(t.TempDir(), "p2p.key")
	t.Setenv("BETAR_P2P_KEY_PATH", keyPath)

	cfg1, err := LoadConfig()
	if err != nil {
		t.Fatalf("first LoadConfig failed: %v", err)
	}

	first, err := p2pcrypto.MarshalPrivateKey(cfg1.P2P.PrivKey)
	if err != nil {
		t.Fatalf("marshal first key failed: %v", err)
	}

	cfg2, err := LoadConfig()
	if err != nil {
		t.Fatalf("second LoadConfig failed: %v", err)
	}

	second, err := p2pcrypto.MarshalPrivateKey(cfg2.P2P.PrivKey)
	if err != nil {
		t.Fatalf("marshal second key failed: %v", err)
	}

	if string(first) != string(second) {
		t.Fatalf("expected deterministic persisted p2p identity across loads")
	}
}

func TestLoadConfigFallsBackToFileConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BETAR_DATA_DIR", dir)
	t.Setenv("BETAR_P2P_KEY_PATH", filepath.Join(dir, "p2p.key"))
	t.Setenv("BETAR_WALLET_KEY_PATH", filepath.Join(dir, "wallet.key"))
	t.Setenv("ETHEREUM_RPC_URL", "")

	// Write a flat config.yaml with network settings
	fc := &FileConfig{
		RPCUrl:  "https://custom-rpc.example.com",
		P2PPort: 5555,
	}
	if err := SaveFileConfig(FileConfigPath(dir), fc); err != nil {
		t.Fatalf("save file config: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Ethereum.RPCURL != "https://custom-rpc.example.com" {
		t.Fatalf("expected RPCURL 'https://custom-rpc.example.com', got %q", cfg.Ethereum.RPCURL)
	}
	if cfg.P2P.Port != 5555 {
		t.Fatalf("expected P2P port 5555, got %d", cfg.P2P.Port)
	}
}

func TestLoadConfigEnvOverridesFileConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BETAR_DATA_DIR", dir)
	t.Setenv("BETAR_P2P_KEY_PATH", filepath.Join(dir, "p2p.key"))
	t.Setenv("BETAR_WALLET_KEY_PATH", filepath.Join(dir, "wallet.key"))
	t.Setenv("ETHEREUM_RPC_URL", "https://from-env.example.com")

	fc := &FileConfig{
		RPCUrl: "https://from-yaml.example.com",
	}
	if err := SaveFileConfig(FileConfigPath(dir), fc); err != nil {
		t.Fatalf("save file config: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Env var should win over config.yaml
	if cfg.Ethereum.RPCURL != "https://from-env.example.com" {
		t.Fatalf("expected RPCURL 'https://from-env.example.com' (env wins), got %q", cfg.Ethereum.RPCURL)
	}
}
