package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadFileConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	original := &FileConfig{
		LLM: LLMFileConfig{
			Provider: "google",
			APIKey:   "test-key",
			Model:    "gemini-2.5-flash",
		},
		Wallet: WalletFileConfig{
			RPCURL: "https://sepolia.base.org",
		},
		P2P: P2PFileConfig{
			Port: 4001,
		},
		Agent: AgentFileConfig{
			Name:        "my-agent",
			Description: "test agent",
			Price:       0.001,
		},
	}

	if err := SaveFileConfig(path, original); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadFileConfig(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.LLM.Provider != "google" {
		t.Fatalf("expected provider 'google', got %q", loaded.LLM.Provider)
	}
	if loaded.LLM.APIKey != "test-key" {
		t.Fatalf("expected api_key 'test-key', got %q", loaded.LLM.APIKey)
	}
	if loaded.Agent.Name != "my-agent" {
		t.Fatalf("expected agent name 'my-agent', got %q", loaded.Agent.Name)
	}
	if loaded.Agent.Price != 0.001 {
		t.Fatalf("expected price 0.001, got %f", loaded.Agent.Price)
	}
}

func TestLoadFileConfigMissing(t *testing.T) {
	t.Parallel()
	fc, err := LoadFileConfig("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	if fc == nil {
		t.Fatal("expected non-nil empty FileConfig")
	}
	if fc.LLM.Provider != "" {
		t.Fatalf("expected empty provider, got %q", fc.LLM.Provider)
	}
}

func TestSaveFileConfigPermissions(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := SaveFileConfig(path, &FileConfig{}); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("expected 0600 permissions, got %o", perm)
	}
}

func TestFileConfigAppliedByLoadConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BETAR_DATA_DIR", dir)
	t.Setenv("BETAR_P2P_KEY_PATH", filepath.Join(dir, "p2p.key"))
	t.Setenv("BETAR_WALLET_KEY_PATH", filepath.Join(dir, "wallet.key"))
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GOOGLE_MODEL", "")
	t.Setenv("LLM_PROVIDER", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "")

	// Simulate what onboard writes
	fc := &FileConfig{
		LLM: LLMFileConfig{
			Provider: "openai",
			APIKey:   "sk-openai-test",
			Model:    "gpt-4o",
			BaseURL:  "http://localhost:11434/v1/",
		},
		P2P: P2PFileConfig{
			Port: 5001,
		},
	}
	if err := SaveFileConfig(FileConfigPath(dir), fc); err != nil {
		t.Fatalf("save: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Agent.Provider != "openai" {
		t.Fatalf("expected provider 'openai', got %q", cfg.Agent.Provider)
	}
	if cfg.Agent.OpenAIAPIKey != "sk-openai-test" {
		t.Fatalf("expected OpenAIAPIKey 'sk-openai-test', got %q", cfg.Agent.OpenAIAPIKey)
	}
	if cfg.Agent.OpenAIBaseURL != "http://localhost:11434/v1/" {
		t.Fatalf("expected OpenAIBaseURL, got %q", cfg.Agent.OpenAIBaseURL)
	}
}
