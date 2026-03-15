package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileConfig represents ~/.betar/config.yaml written by `betar onboard`.
type FileConfig struct {
	LLM    LLMFileConfig    `yaml:"llm"`
	Wallet WalletFileConfig `yaml:"wallet"`
	P2P    P2PFileConfig    `yaml:"p2p"`
	Agent  AgentFileConfig  `yaml:"agent"`
}

// LLMFileConfig holds LLM provider settings.
type LLMFileConfig struct {
	Provider string `yaml:"provider"` // "google" or "openai"
	APIKey   string `yaml:"api_key"`
	Model    string `yaml:"model"`
	BaseURL  string `yaml:"base_url,omitempty"` // only for openai provider
}

// WalletFileConfig holds Ethereum wallet settings.
// Private keys are stored in wallet.key file, not in config.yaml.
type WalletFileConfig struct {
	RPCURL string `yaml:"rpc_url"`
}

// P2PFileConfig holds P2P network settings.
type P2PFileConfig struct {
	Port           int      `yaml:"port"`
	BootstrapPeers []string `yaml:"bootstrap_peers,omitempty"`
}

// AgentFileConfig holds default agent profile settings.
type AgentFileConfig struct {
	Name        string  `yaml:"name"`
	Description string  `yaml:"description"`
	Price       float64 `yaml:"price"`
}

// FileConfigPath returns the config.yaml path for the given data directory.
// FileConfigPath returns the config.yaml path for the given data directory.
func FileConfigPath(dataDir string) string {
	return filepath.Join(dataDir, "config.yaml")
}

// LoadFileConfig loads config from a YAML file.
// Returns an empty FileConfig (not an error) if the file does not exist.
func LoadFileConfig(path string) (*FileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &FileConfig{}, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	var fc FileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	return &fc, nil
}

// SaveFileConfig writes the config to a YAML file with 0600 permissions.
func SaveFileConfig(path string, fc *FileConfig) error {
	data, err := yaml.Marshal(fc)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}
