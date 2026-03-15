# Onboard Wizard & Unified Config Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `betar onboard` interactive wizard and unified `~/.betar/config.yaml` so users can get started with two commands.

**Architecture:** New `FileConfig` struct parsed from YAML, loaded as fallback in existing `LoadConfig()`. New `onboard` cobra command using `charmbracelet/huh` forms writes the config file. Existing code is untouched except for config loading and command registration.

**Tech Stack:** Go, Cobra, charmbracelet/huh, gopkg.in/yaml.v3

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/config/fileconfig.go` | Create | `FileConfig` struct, `LoadFileConfig()`, `SaveFileConfig()` |
| `internal/config/fileconfig_test.go` | Create | Tests for file config load/save/merge |
| `internal/config/config.go` | Modify | Call `LoadFileConfig()` to fill unset fields after env vars |
| `internal/config/config_test.go` | Modify | Add test for config.yaml fallback |
| `cmd/betar/onboard.go` | Create | `betar onboard` cobra command with huh wizard |
| `cmd/betar/main.go` | Modify | Register `onboard` command, add startup hint |

---

## Chunk 1: FileConfig (load/save/merge)

### Task 1: FileConfig struct and save/load

**Files:**
- Create: `internal/config/fileconfig.go`
- Create: `internal/config/fileconfig_test.go`

- [ ] **Step 1: Write the failing test for SaveFileConfig + LoadFileConfig round-trip**

In `internal/config/fileconfig_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestSaveAndLoadFileConfig -v`
Expected: FAIL — `FileConfig` type not defined

- [ ] **Step 3: Write the FileConfig implementation**

In `internal/config/fileconfig.go`:

```go
package config

import (
	"errors"
	"fmt"
	"os"

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
type WalletFileConfig struct {
	PrivateKey string `yaml:"private_key,omitempty"` // empty = use wallet.key file
	RPCURL     string `yaml:"rpc_url"`
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
func FileConfigPath(dataDir string) string {
	return dataDir + "/config.yaml"
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
```

- [ ] **Step 4: Run all three tests to verify they pass**

Run: `go test ./internal/config/ -run "TestSaveAndLoadFileConfig|TestLoadFileConfigMissing|TestSaveFileConfigPermissions" -v`
Expected: PASS (3 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/config/fileconfig.go internal/config/fileconfig_test.go
git commit -m "feat: add FileConfig struct for unified config.yaml"
```

### Task 2: Merge FileConfig into LoadConfig

**Files:**
- Modify: `internal/config/config.go` (lines 71-133, `LoadConfig()`)
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test for config.yaml fallback**

Append to `internal/config/config_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -run "TestLoadConfigFallsBackToFileConfig|TestLoadConfigEnvOverridesFileConfig" -v`
Expected: FAIL — config.yaml not being read yet

- [ ] **Step 3: Add file config merging to LoadConfig**

In `internal/config/config.go`, add a new function and call it at the end of `LoadConfig()`, just before `return cfg, nil` (line 132):

```go
// Add this call just before "return cfg, nil" in LoadConfig():
	applyFileConfig(cfg)

// Add this new function:
func applyFileConfig(cfg *Config) {
	fc, err := LoadFileConfig(FileConfigPath(cfg.Storage.DataDir))
	if err != nil || fc == nil {
		return
	}

	// LLM settings — only fill if env var didn't set them.
	// Provider
	if cfg.Agent.Provider == "" && fc.LLM.Provider != "" {
		cfg.Agent.Provider = fc.LLM.Provider
	}
	// API key — route to correct field based on provider
	provider := fc.LLM.Provider
	if provider == "openai" {
		if cfg.Agent.OpenAIAPIKey == "" && fc.LLM.APIKey != "" {
			cfg.Agent.OpenAIAPIKey = fc.LLM.APIKey
		}
		if cfg.Agent.OpenAIBaseURL == "" && fc.LLM.BaseURL != "" {
			cfg.Agent.OpenAIBaseURL = fc.LLM.BaseURL
		}
	} else {
		if cfg.Agent.APIKey == "" && fc.LLM.APIKey != "" {
			cfg.Agent.APIKey = fc.LLM.APIKey
		}
	}
	// Model — only apply if env var GOOGLE_MODEL was not set
	if os.Getenv("GOOGLE_MODEL") == "" && fc.LLM.Model != "" {
		cfg.Agent.Model = fc.LLM.Model
	}

	// Wallet — only fill if env var didn't set it
	if cfg.Ethereum.PrivateKey == "" && fc.Wallet.PrivateKey != "" {
		cfg.Ethereum.PrivateKey = fc.Wallet.PrivateKey
	}
	if fc.Wallet.RPCURL != "" && os.Getenv("ETHEREUM_RPC_URL") == "" {
		cfg.Ethereum.RPCURL = fc.Wallet.RPCURL
	}

	// P2P — bootstrap peers only (port is handled by CLI flags in initRuntime)
	if len(cfg.P2P.BootstrapPeers) == 0 && len(fc.P2P.BootstrapPeers) > 0 {
		cfg.P2P.BootstrapPeers = fc.P2P.BootstrapPeers
	}
}
```

Note: The `Agent` section of `FileConfig` (name, description, price) is not applied in `LoadConfig()` — it is consumed by the `betar start` / `betar onboard` commands directly. `LoadConfig` only handles global settings.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -run "TestLoadConfigFallsBackToFileConfig|TestLoadConfigEnvOverridesFileConfig" -v`
Expected: PASS

- [ ] **Step 5: Run all config tests to ensure no regressions**

Run: `go test ./internal/config/ -v`
Expected: All tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: merge config.yaml as fallback in LoadConfig"
```

---

## Chunk 2: Onboard Command

### Task 3: Add charmbracelet/huh dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the dependency**

Run: `go get github.com/charmbracelet/huh@latest`

- [ ] **Step 2: Tidy modules**

Run: `go mod tidy`

- [ ] **Step 3: Verify it compiles**

Run: `go build ./...`
Expected: Success

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add charmbracelet/huh for onboard wizard"
```

### Task 4: Build the onboard command

**Files:**
- Create: `cmd/betar/onboard.go`
- Modify: `cmd/betar/main.go`

- [ ] **Step 1: Create the onboard command**

In `cmd/betar/onboard.go`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/asabya/betar/internal/config"
	"github.com/charmbracelet/huh"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Interactive setup wizard for Betar",
	Long:  "Walk through LLM provider, wallet, and agent setup. Writes ~/.betar/config.yaml.",
	RunE:  runOnboard,
}

func runOnboard(cmd *cobra.Command, args []string) error {
	dataDir := os.Getenv("BETAR_DATA_DIR")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".betar")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("cannot create data directory: %w", err)
	}

	cfgPath := config.FileConfigPath(dataDir)

	// Load existing config for pre-populating defaults on re-run
	existing, _ := config.LoadFileConfig(cfgPath)
	if existing == nil {
		existing = &config.FileConfig{}
	}

	fc := &config.FileConfig{}

	// --- Step 1: LLM Provider ---
	provider := existing.LLM.Provider
	if provider == "" {
		provider = "google"
	}
	if err := huh.NewSelect[string]().
		Title("Select your AI provider").
		Options(
			huh.NewOption("Google (Gemini)", "google"),
			huh.NewOption("OpenAI / Ollama", "openai"),
		).
		Value(&provider).
		Run(); err != nil {
		return err
	}
	fc.LLM.Provider = provider

	apiKeyDefault := existing.LLM.APIKey
	var apiKey string
	if err := huh.NewInput().
		Title("Enter your API key").
		Value(&apiKey).
		Placeholder(maskKey(apiKeyDefault)).
		Run(); err != nil {
		return err
	}
	if apiKey == "" {
		apiKey = apiKeyDefault
	}
	fc.LLM.APIKey = apiKey

	if provider == "openai" {
		baseURLDefault := existing.LLM.BaseURL
		if baseURLDefault == "" {
			baseURLDefault = "https://api.openai.com/v1/"
		}
		var baseURL string
		if err := huh.NewInput().
			Title("Base URL").
			Description("For Ollama use http://localhost:11434/v1/").
			Value(&baseURL).
			Placeholder(baseURLDefault).
			Run(); err != nil {
			return err
		}
		if baseURL == "" {
			baseURL = baseURLDefault
		}
		fc.LLM.BaseURL = baseURL
	}

	modelDefault := existing.LLM.Model
	if modelDefault == "" {
		if provider == "google" {
			modelDefault = "gemini-2.5-flash"
		} else {
			modelDefault = "gpt-4o"
		}
	}
	var model string
	if err := huh.NewInput().
		Title("Model").
		Value(&model).
		Placeholder(modelDefault).
		Run(); err != nil {
		return err
	}
	if model == "" {
		model = modelDefault
	}
	fc.LLM.Model = model

	// --- Step 2: Wallet ---
	walletKeyPath := filepath.Join(dataDir, "wallet.key")
	walletExists := fileExists(walletKeyPath)

	walletChoice := "skip"
	walletOptions := []huh.Option[string]{
		huh.NewOption("Generate new wallet", "generate"),
		huh.NewOption("Import existing private key", "import"),
		huh.NewOption("Skip for now", "skip"),
	}
	if walletExists {
		walletOptions = []huh.Option[string]{
			huh.NewOption("Keep existing wallet", "keep"),
			huh.NewOption("Generate new wallet (replaces current)", "generate"),
			huh.NewOption("Import existing private key", "import"),
			huh.NewOption("Skip for now", "skip"),
		}
		walletChoice = "keep"
	}

	if err := huh.NewSelect[string]().
		Title("Set up an Ethereum wallet for payments?").
		Options(walletOptions...).
		Value(&walletChoice).
		Run(); err != nil {
		return err
	}

	switch walletChoice {
	case "generate":
		if walletExists {
			var confirm bool
			if err := huh.NewConfirm().
				Title("This will replace your existing wallet key. Any funds in the old wallet will be inaccessible. Continue?").
				Affirmative("Yes, replace").
				Negative("No, keep existing").
				Value(&confirm).
				Run(); err != nil {
				return err
			}
			if !confirm {
				fmt.Println("Keeping existing wallet.")
				break
			}
		}
		key, err := crypto.GenerateKey()
		if err != nil {
			return fmt.Errorf("failed to generate wallet key: %w", err)
		}
		keyHex := fmt.Sprintf("%x", crypto.FromECDSA(key))
		if err := os.WriteFile(walletKeyPath, []byte(keyHex+"\n"), 0o600); err != nil {
			return fmt.Errorf("failed to save wallet key: %w", err)
		}
		addr := crypto.PubkeyToAddress(key.PublicKey).Hex()
		fmt.Printf("Generated wallet: %s\n", addr)
		fmt.Printf("Private key saved to %s\n", walletKeyPath)

	case "import":
		var keyHex string
		if err := huh.NewInput().
			Title("Enter your private key (hex, without 0x prefix)").
			Value(&keyHex).
			EchoMode(huh.EchoModePassword).
			Run(); err != nil {
			return err
		}
		keyHex = strings.TrimPrefix(strings.TrimSpace(keyHex), "0x")
		if _, err := crypto.HexToECDSA(keyHex); err != nil {
			return fmt.Errorf("invalid private key: %w", err)
		}
		if err := os.WriteFile(walletKeyPath, []byte(keyHex+"\n"), 0o600); err != nil {
			return fmt.Errorf("failed to save wallet key: %w", err)
		}
		fmt.Println("Wallet key imported.")

	case "keep":
		fmt.Println("Keeping existing wallet.")
	}

	rpcDefault := existing.Wallet.RPCURL
	if rpcDefault == "" {
		rpcDefault = "https://sepolia.base.org"
	}
	var rpcURL string
	if err := huh.NewInput().
		Title("Ethereum RPC URL").
		Value(&rpcURL).
		Placeholder(rpcDefault).
		Run(); err != nil {
		return err
	}
	if rpcURL == "" {
		rpcURL = rpcDefault
	}
	fc.Wallet.RPCURL = rpcURL

	// --- Step 3: Agent Profile ---
	var setupAgent bool
	if err := huh.NewConfirm().
		Title("Set up a default agent?").
		Affirmative("Yes").
		Negative("No").
		Value(&setupAgent).
		Run(); err != nil {
		return err
	}

	if setupAgent {
		nameDefault := existing.Agent.Name
		if nameDefault == "" {
			nameDefault = "my-agent"
		}
		var agentName string
		if err := huh.NewInput().
			Title("Agent name").
			Value(&agentName).
			Placeholder(nameDefault).
			Run(); err != nil {
			return err
		}
		if agentName == "" {
			agentName = nameDefault
		}
		fc.Agent.Name = agentName

		var agentDesc string
		if err := huh.NewInput().
			Title("Agent description").
			Value(&agentDesc).
			Placeholder(existing.Agent.Description).
			Run(); err != nil {
			return err
		}
		if agentDesc == "" {
			agentDesc = existing.Agent.Description
		}
		fc.Agent.Description = agentDesc

		// Price — use string input, parse to float
		priceDefault := "0"
		if existing.Agent.Price > 0 {
			priceDefault = fmt.Sprintf("%g", existing.Agent.Price)
		}
		var priceStr string
		if err := huh.NewInput().
			Title("Price per task in USDC").
			Value(&priceStr).
			Placeholder(priceDefault).
			Run(); err != nil {
			return err
		}
		if priceStr == "" {
			priceStr = priceDefault
		}
		var price float64
		fmt.Sscanf(priceStr, "%f", &price)
		fc.Agent.Price = price
	}

	// P2P defaults
	fc.P2P.Port = 4001
	if existing.P2P.Port != 0 {
		fc.P2P.Port = existing.P2P.Port
	}
	fc.P2P.BootstrapPeers = existing.P2P.BootstrapPeers

	// --- Save ---
	if err := config.SaveFileConfig(cfgPath, fc); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Printf("Config saved to %s\n", cfgPath)
	fmt.Println()
	fmt.Println("Get started:")
	fmt.Println("  betar start          # Start your agent")
	fmt.Println("  betar onboard        # Re-run setup anytime")

	return nil
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return key
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
```

- [ ] **Step 2: Register the onboard command in main.go**

In `cmd/betar/main.go`, in the `init()` function, add after the existing `rootCmd.AddCommand(walletCmd)` line (around line 397):

```go
	rootCmd.AddCommand(onboardCmd)
```

- [ ] **Step 3: Add startup hint and fix flag overrides in initRuntime**

In `cmd/betar/main.go`, in `initRuntime()`:

First, right after `cfg, err = config.LoadConfig()` succeeds (line 587), add the hint:

```go
	// Hint to run onboard if no LLM key is configured
	if cfg.Agent.APIKey == "" && cfg.Agent.OpenAIAPIKey == "" {
		fmt.Println("Tip: Run 'betar onboard' to set up your LLM provider and agent.")
	}
```

Then fix the flag overrides (lines 589-605) to only apply when explicitly set by the user. This prevents CLI flag defaults from clobbering config.yaml values:

Replace:
```go
	port, _ := cmd.Flags().GetInt("port")
	bootstrap, _ := cmd.Flags().GetStringSlice("bootstrap")
	modelName, _ := cmd.Flags().GetString("model")

	cfg.P2P.Port = port
	cfg.P2P.BootstrapPeers = bootstrap
	cfg.Agent.Model = modelName
```

With:
```go
	if cmd.Flags().Changed("port") {
		port, _ := cmd.Flags().GetInt("port")
		cfg.P2P.Port = port
	}
	if cmd.Flags().Changed("bootstrap") {
		bootstrap, _ := cmd.Flags().GetStringSlice("bootstrap")
		cfg.P2P.BootstrapPeers = bootstrap
	}
	if cmd.Flags().Changed("model") {
		modelName, _ := cmd.Flags().GetString("model")
		cfg.Agent.Model = modelName
	}
```

This ensures: CLI flags > env vars > config.yaml > defaults.

- [ ] **Step 4: Verify it compiles**

Run: `go build ./cmd/betar/`
Expected: Success

- [ ] **Step 5: Verify the command appears in help**

Run: `go run ./cmd/betar/ --help`
Expected: `onboard` appears in the available commands list

- [ ] **Step 6: Run all existing tests to check for regressions**

Run: `go test ./...`
Expected: All tests PASS

- [ ] **Step 7: Commit**

```bash
git add cmd/betar/onboard.go cmd/betar/main.go
git commit -m "feat: add betar onboard interactive setup wizard"
```

### Task 5: Integration test — onboard writes config that LoadConfig reads

**Files:**
- Modify: `internal/config/fileconfig_test.go`

- [ ] **Step 1: Write end-to-end config flow test**

Append to `internal/config/fileconfig_test.go`:

```go
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
```

- [ ] **Step 2: Run the test**

Run: `go test ./internal/config/ -run TestFileConfigAppliedByLoadConfig -v`
Expected: PASS

- [ ] **Step 3: Run full test suite**

Run: `go test ./...`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add internal/config/fileconfig_test.go
git commit -m "test: add integration test for config.yaml → LoadConfig flow"
```

### Task 6: Update startCmd to use agent config from config.yaml

**Files:**
- Modify: `cmd/betar/main.go`

- [ ] **Step 1: In `runStart`, load agent name from config.yaml if not provided via flag**

In `cmd/betar/main.go`, find the `runStart` function. After `initRuntime(cmd)` succeeds, before agent registration, add logic to fall back to config.yaml agent settings:

```go
	// Fall back to config.yaml agent settings if no --name flag provided
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		fc, _ := config.LoadFileConfig(config.FileConfigPath(cfg.Storage.DataDir))
		if fc != nil && fc.Agent.Name != "" {
			name = fc.Agent.Name
			// Also apply description and price if not set via flags
			if desc, _ := cmd.Flags().GetString("description"); desc == "" {
				cmd.Flags().Set("description", fc.Agent.Description)
			}
			if price, _ := cmd.Flags().GetFloat64("price"); price == 0 {
				cmd.Flags().Set("price", fmt.Sprintf("%g", fc.Agent.Price))
			}
			cmd.Flags().Set("name", name)
		}
	}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./cmd/betar/`
Expected: Success

- [ ] **Step 3: Run all tests**

Run: `go test ./...`
Expected: All PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/betar/main.go
git commit -m "feat: betar start falls back to config.yaml agent settings"
```
