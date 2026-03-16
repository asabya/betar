package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileConfig represents ~/.betar/config.yaml written by `betar onboard`.
// It holds flat network settings only; agent profiles live in agents.yaml.
type FileConfig struct {
	RPCUrl         string   `yaml:"rpc_url"`
	P2PPort        int      `yaml:"p2p_port"`
	BootstrapPeers []string `yaml:"bootstrap_peers,omitempty"`
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
