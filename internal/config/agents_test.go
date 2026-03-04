package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAgentsConfigMissingFile(t *testing.T) {
	t.Parallel()
	cfg, err := LoadAgentsConfig(t.TempDir())
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(cfg.Agents) != 0 {
		t.Fatalf("expected empty agents, got %d", len(cfg.Agents))
	}
}

func TestSaveAndLoadAgentsConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := &AgentsConfig{
		Agents: []AgentProfile{
			{Name: "bot-a", Description: "alpha", Price: 0.001, Model: "gemini-2.5-flash"},
			{Name: "bot-b", Description: "beta", Price: 0.002},
		},
	}
	if err := SaveAgentsConfig(dir, cfg); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	loaded, err := LoadAgentsConfig(dir)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(loaded.Agents))
	}
	if loaded.Agents[0].Name != "bot-a" || loaded.Agents[0].Price != 0.001 {
		t.Fatalf("unexpected agent[0]: %+v", loaded.Agents[0])
	}
}

func TestSaveAgentsConfigFilePermissions(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := &AgentsConfig{Agents: []AgentProfile{{Name: "x", Price: 0}}}
	if err := SaveAgentsConfig(dir, cfg); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "agents.yaml"))
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("expected 0600 permissions, got %o", perm)
	}
}

func TestAddProfileDuplicate(t *testing.T) {
	t.Parallel()
	cfg := &AgentsConfig{}
	if err := cfg.AddProfile(AgentProfile{Name: "bot"}); err != nil {
		t.Fatalf("first add failed: %v", err)
	}
	if err := cfg.AddProfile(AgentProfile{Name: "bot"}); err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
}

func TestAddProfileEmptyName(t *testing.T) {
	t.Parallel()
	cfg := &AgentsConfig{}
	if err := cfg.AddProfile(AgentProfile{Name: ""}); err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

func TestDeleteProfile(t *testing.T) {
	t.Parallel()
	cfg := &AgentsConfig{Agents: []AgentProfile{{Name: "a"}, {Name: "b"}, {Name: "c"}}}
	if err := cfg.DeleteProfile("b"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if len(cfg.Agents) != 2 {
		t.Fatalf("expected 2 agents after delete, got %d", len(cfg.Agents))
	}
	if cfg.FindProfile("b") != nil {
		t.Fatal("deleted profile still found")
	}
}

func TestDeleteProfileNotFound(t *testing.T) {
	t.Parallel()
	cfg := &AgentsConfig{}
	if err := cfg.DeleteProfile("ghost"); err == nil {
		t.Fatal("expected error for missing profile, got nil")
	}
}

func TestUpdateProfilePrice(t *testing.T) {
	t.Parallel()
	cfg := &AgentsConfig{Agents: []AgentProfile{{Name: "bot", Price: 0.001}}}
	if err := cfg.UpdateProfile("bot", AgentProfile{Price: 0.005}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if cfg.Agents[0].Price != 0.005 {
		t.Fatalf("expected 0.005, got %f", cfg.Agents[0].Price)
	}
}

// TestUpdateProfilePriceZeroIsNoop documents the known limitation of UpdateProfile:
// setting Price to 0 via UpdateProfile is a no-op due to the zero-value guard.
// The CLI works around this by using cmd.Flags().Changed() in agentConfigEdit directly.
func TestUpdateProfilePriceZeroIsNoop(t *testing.T) {
	t.Parallel()
	cfg := &AgentsConfig{Agents: []AgentProfile{{Name: "bot", Price: 0.001}}}
	if err := cfg.UpdateProfile("bot", AgentProfile{Price: 0}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	// Price stays at 0.001 because UpdateProfile skips zero values.
	if cfg.Agents[0].Price != 0.001 {
		t.Fatalf("expected 0.001 (unchanged), got %f", cfg.Agents[0].Price)
	}
}

func TestUpdateProfileNotFound(t *testing.T) {
	t.Parallel()
	cfg := &AgentsConfig{}
	if err := cfg.UpdateProfile("ghost", AgentProfile{Description: "x"}); err == nil {
		t.Fatal("expected error for missing profile, got nil")
	}
}

func TestValidateDuplicateNames(t *testing.T) {
	t.Parallel()
	cfg := &AgentsConfig{Agents: []AgentProfile{{Name: "dup"}, {Name: "dup"}}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for duplicate names, got nil")
	}
}

func TestLoadAgentsConfigInvalidYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "agents.yaml")
	if err := os.WriteFile(path, []byte("agents:\n  - {name: [bad"), 0o600); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	_, err := LoadAgentsConfig(dir)
	if err == nil {
		t.Fatal("expected parse error for invalid YAML, got nil")
	}
}
