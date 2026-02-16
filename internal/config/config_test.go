package config

import (
	"path/filepath"
	"testing"

	p2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
)

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
