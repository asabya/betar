package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AgentProfile holds the persistent configuration for a single agent.
type AgentProfile struct {
	Name          string  `yaml:"name"`
	Description   string  `yaml:"description"`
	Price         float64 `yaml:"price"`
	Model         string  `yaml:"model,omitempty"`          // falls back to GOOGLE_MODEL env
	APIKey        string  `yaml:"api_key,omitempty"`        // Google API key; falls back to GOOGLE_API_KEY env
	Provider      string  `yaml:"provider,omitempty"`       // "google", "openai", or "" for auto-detect
	OpenAIAPIKey  string  `yaml:"openai_api_key,omitempty"` // OpenAI-compatible API key
	OpenAIBaseURL string  `yaml:"openai_base_url,omitempty"` // OpenAI-compatible base URL
}

// AgentsConfig is the top-level structure for agents.yaml.
type AgentsConfig struct {
	Agents []AgentProfile `yaml:"agents"`
}

// AgentsConfigPath returns the path to the agents.yaml file.
func AgentsConfigPath(dataDir string) string {
	return filepath.Join(dataDir, "agents.yaml")
}

// LoadAgentsConfig loads the agents config from disk.
// Returns an empty config (not an error) if the file does not exist.
func LoadAgentsConfig(dataDir string) (*AgentsConfig, error) {
	path := AgentsConfigPath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &AgentsConfig{}, nil
		}
		return nil, fmt.Errorf("reading agents config: %w", err)
	}
	var cfg AgentsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing agents config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid agents config: %w", err)
	}
	return &cfg, nil
}

// SaveAgentsConfig marshals the config to YAML and writes it to disk.
func SaveAgentsConfig(dataDir string, cfg *AgentsConfig) error {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling agents config: %w", err)
	}
	path := AgentsConfigPath(dataDir)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing agents config: %w", err)
	}
	return nil
}

// Validate checks for empty names and duplicate names.
func (c *AgentsConfig) Validate() error {
	seen := make(map[string]bool)
	for _, p := range c.Agents {
		if p.Name == "" {
			return fmt.Errorf("agent profile has empty name")
		}
		if seen[p.Name] {
			return fmt.Errorf("duplicate agent profile name: %q", p.Name)
		}
		seen[p.Name] = true
	}
	return nil
}

// FindProfile returns the profile with the given name, or nil if not found.
func (c *AgentsConfig) FindProfile(name string) *AgentProfile {
	for i := range c.Agents {
		if c.Agents[i].Name == name {
			return &c.Agents[i]
		}
	}
	return nil
}

// AddProfile adds a new profile. Returns an error if the name already exists.
func (c *AgentsConfig) AddProfile(p AgentProfile) error {
	if p.Name == "" {
		return fmt.Errorf("agent profile name cannot be empty")
	}
	if c.FindProfile(p.Name) != nil {
		return fmt.Errorf("agent profile %q already exists", p.Name)
	}
	c.Agents = append(c.Agents, p)
	return nil
}

// DeleteProfile removes the profile with the given name.
// Returns an error if not found.
func (c *AgentsConfig) DeleteProfile(name string) error {
	for i, p := range c.Agents {
		if p.Name == name {
			c.Agents = append(c.Agents[:i], c.Agents[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("agent profile %q not found", name)
}

