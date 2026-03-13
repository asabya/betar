package eip8004

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
)

// TokenStore persists EIP-8004 tokenIDs so agents aren't re-registered on restart.
// Keyed by agent name (from agents.yaml) since that's the stable identifier across restarts.
type TokenStore struct {
	path   string
	mu     sync.Mutex
	tokens map[string]*big.Int // agent name → on-chain tokenID
}

// NewTokenStore loads or creates a token store at dataDir/eip8004_tokens.json.
func NewTokenStore(dataDir string) (*TokenStore, error) {
	path := filepath.Join(dataDir, "eip8004_tokens.json")
	s := &TokenStore{
		path:   path,
		tokens: make(map[string]*big.Int),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("reading token store: %w", err)
	}

	// Stored as map[string]string for JSON compat with big.Int
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing token store: %w", err)
	}
	for name, val := range raw {
		n := new(big.Int)
		if _, ok := n.SetString(val, 10); ok {
			s.tokens[name] = n
		}
	}
	return s, nil
}

// Get returns the tokenID for the given agent name, or nil if not stored.
func (s *TokenStore) Get(agentName string) *big.Int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tokens[agentName]
}

// Put stores a tokenID for the given agent name and persists to disk.
func (s *TokenStore) Put(agentName string, tokenID *big.Int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[agentName] = tokenID

	raw := make(map[string]string, len(s.tokens))
	for k, v := range s.tokens {
		raw[k] = v.String()
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}
