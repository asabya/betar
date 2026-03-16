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
		RPCUrl:         "https://sepolia.base.org",
		P2PPort:        4001,
		BootstrapPeers: []string{"/ip4/1.2.3.4/tcp/4001/p2p/Qm123"},
	}

	if err := SaveFileConfig(path, original); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadFileConfig(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.RPCUrl != "https://sepolia.base.org" {
		t.Fatalf("expected rpc_url 'https://sepolia.base.org', got %q", loaded.RPCUrl)
	}
	if loaded.P2PPort != 4001 {
		t.Fatalf("expected p2p_port 4001, got %d", loaded.P2PPort)
	}
	if len(loaded.BootstrapPeers) != 1 {
		t.Fatalf("expected 1 bootstrap peer, got %d", len(loaded.BootstrapPeers))
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
	if fc.RPCUrl != "" {
		t.Fatalf("expected empty rpc_url, got %q", fc.RPCUrl)
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
	t.Setenv("ETHEREUM_RPC_URL", "")

	fc := &FileConfig{
		RPCUrl:         "https://custom-rpc.example.com",
		P2PPort:        5001,
		BootstrapPeers: []string{"/ip4/1.2.3.4/tcp/4001/p2p/Qm123"},
	}
	if err := SaveFileConfig(FileConfigPath(dir), fc); err != nil {
		t.Fatalf("save: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.Ethereum.RPCURL != "https://custom-rpc.example.com" {
		t.Fatalf("expected RPCURL 'https://custom-rpc.example.com', got %q", cfg.Ethereum.RPCURL)
	}
	if cfg.P2P.Port != 5001 {
		t.Fatalf("expected P2P port 5001, got %d", cfg.P2P.Port)
	}
	if len(cfg.P2P.BootstrapPeers) != 1 {
		t.Fatalf("expected 1 bootstrap peer, got %d", len(cfg.P2P.BootstrapPeers))
	}
}
