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
	// Clear env vars that would normally provide these
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GOOGLE_MODEL", "")
	t.Setenv("LLM_PROVIDER", "")

	// Write a config.yaml with LLM settings
	fc := &FileConfig{
		LLM: LLMFileConfig{
			Provider: "google",
			APIKey:   "from-config-yaml",
			Model:    "gemini-2.0-flash",
		},
		Agent: AgentFileConfig{
			Name: "yaml-agent",
		},
	}
	if err := SaveFileConfig(FileConfigPath(dir), fc); err != nil {
		t.Fatalf("save file config: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Agent.APIKey != "from-config-yaml" {
		t.Fatalf("expected APIKey 'from-config-yaml', got %q", cfg.Agent.APIKey)
	}
	if cfg.Agent.Model != "gemini-2.0-flash" {
		t.Fatalf("expected Model 'gemini-2.0-flash', got %q", cfg.Agent.Model)
	}
}

func TestLoadConfigEnvOverridesFileConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BETAR_DATA_DIR", dir)
	t.Setenv("BETAR_P2P_KEY_PATH", filepath.Join(dir, "p2p.key"))
	t.Setenv("BETAR_WALLET_KEY_PATH", filepath.Join(dir, "wallet.key"))
	t.Setenv("GOOGLE_API_KEY", "from-env")
	t.Setenv("GOOGLE_MODEL", "")
	t.Setenv("LLM_PROVIDER", "")

	fc := &FileConfig{
		LLM: LLMFileConfig{
			Provider: "google",
			APIKey:   "from-config-yaml",
			Model:    "gemini-2.0-flash",
		},
	}
	if err := SaveFileConfig(FileConfigPath(dir), fc); err != nil {
		t.Fatalf("save file config: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Env var should win over config.yaml
	if cfg.Agent.APIKey != "from-env" {
		t.Fatalf("expected APIKey 'from-env' (env wins), got %q", cfg.Agent.APIKey)
	}
	// But model should come from config.yaml since env is empty
	if cfg.Agent.Model != "gemini-2.0-flash" {
		t.Fatalf("expected Model 'gemini-2.0-flash' (from yaml), got %q", cfg.Agent.Model)
	}
}
