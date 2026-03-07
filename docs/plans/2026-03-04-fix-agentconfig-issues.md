# Fix agentconfig PR Issues Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix five issues flagged in the PR #11 code review: remove the dead `--framework` flag, fix price-reset bug in `edit`, fix hardcoded announce interval in `serveAgent`, remove redundant fallback logic, add tests for `config/agents.go`, and update README env table.

**Architecture:** Changes are isolated across three layers — CLI flags (`cmd/betar/main.go`), config CRUD (`internal/config/agents.go` + new `agents_test.go`), and docs (`README.md`). No cross-cutting refactors required; each task is self-contained.

**Tech Stack:** Go 1.25, Cobra CLI, `gopkg.in/yaml.v3`, standard `testing` package.

---

### Task 1: Remove `--framework` flag and `Framework` field everywhere

The `Framework` field is stored and passed but never read. Removing it prevents confusion.

**Files:**
- Modify: `cmd/betar/main.go`
- Modify: `internal/agent/manager.go` (line ~828–835, AgentSpec struct)
- Modify: `internal/config/agents.go` (line ~12–21, AgentProfile struct; line ~136–138, UpdateProfile)
- Modify: `agents.example.yaml`
- Modify: `README.md`

**Step 1: Remove `Framework` from `AgentSpec` in `internal/agent/manager.go`**

Find the `AgentSpec` struct (around line 828) and remove the `Framework` field:

```go
// Before
type AgentSpec struct {
	Name        string
	Description string
	Image       string
	Price       float64
	Framework   string   // ← remove this line
	Model       string
	...
}
```

**Step 2: Remove `Framework` from `AgentProfile` in `internal/config/agents.go`**

```go
// Before
type AgentProfile struct {
	Name          string  `yaml:"name"`
	Description   string  `yaml:"description"`
	Price         float64 `yaml:"price"`
	Model         string  `yaml:"model,omitempty"`
	APIKey        string  `yaml:"api_key,omitempty"`
	Framework     string  `yaml:"framework,omitempty"`   // ← remove
	Provider      string  `yaml:"provider,omitempty"`
	OpenAIAPIKey  string  `yaml:"openai_api_key,omitempty"`
	OpenAIBaseURL string  `yaml:"openai_base_url,omitempty"`
}
```

Also remove the `Framework` block from `UpdateProfile` (lines ~136–138):
```go
// Remove this block:
if updates.Framework != "" {
    p.Framework = updates.Framework
}
```

**Step 3: Remove `--framework` flag registrations from `cmd/betar/main.go`**

Remove these four lines from `init()`:
```go
agentServeCmd.Flags().String("framework", "adk", "Agent framework")      // line ~251
startCmd.Flags().String("framework", "adk", "Agent framework")           // line ~264
agentConfigAddCmd.Flags().String("framework", "google-adk", "Agent framework") // line ~299
rootCmd.Flags().String("framework", "adk", "Agent framework")            // line ~324
```

**Step 4: Remove all reads of `--framework` flag in `cmd/betar/main.go`**

Remove `framework` local variable reads and struct fields in:
- `serveAgent` (line ~387–394): remove `framework, _ := cmd.Flags().GetString("framework")` and `Framework: framework`
- `registerLocalAgentFromFlags` (line ~645–653): remove `framework, _ := cmd.Flags().GetString("framework")` and `Framework: framework`
- `loadAndRegisterAgentsFromConfig` (line ~923): remove `Framework: profile.Framework`
- `agentConfigAdd` (line ~634,648): remove `framework, _ := cmd.Flags().GetString("framework")` and `Framework: framework` from the profile struct

**Step 5: Update `agents.example.yaml`**

Remove the `framework: google-adk` line and its comment.

**Step 6: Update `README.md`**

Remove `framework: google-adk      # optional, default google-adk` from the agents.yaml example block (line ~191).

**Step 7: Run tests and build**

```bash
go build ./...
go test ./...
```
Expected: compiles cleanly, all tests pass.

**Step 8: Commit**

```bash
git add cmd/betar/main.go internal/agent/manager.go internal/config/agents.go agents.example.yaml README.md
git commit -m "chore: remove dead --framework flag (never read by agent runtime)"
```

---

### Task 2: Fix price-reset bug in `agentConfigEdit`

`UpdateProfile` uses `if updates.Price != 0` so `--price 0` is a silent no-op. Fix by using `cmd.Flags().Changed()` in the edit handler and bypassing `UpdateProfile`'s zero-value guard.

**Files:**
- Modify: `cmd/betar/main.go` — `agentConfigEdit` function (around line 682)

**Step 1: Rewrite `agentConfigEdit` to use `Changed()` detection**

Replace the current implementation of `agentConfigEdit`:

```go
func agentConfigEdit(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	agentsCfg, err := config.LoadAgentsConfig(cfg.Storage.DataDir)
	if err != nil {
		return err
	}
	name := args[0]
	p := agentsCfg.FindProfile(name)
	if p == nil {
		return fmt.Errorf("agent profile %q not found", name)
	}
	if cmd.Flags().Changed("description") {
		p.Description, _ = cmd.Flags().GetString("description")
	}
	if cmd.Flags().Changed("price") {
		p.Price, _ = cmd.Flags().GetFloat64("price")
	}
	if cmd.Flags().Changed("model") {
		p.Model, _ = cmd.Flags().GetString("model")
	}
	if cmd.Flags().Changed("api-key") {
		p.APIKey, _ = cmd.Flags().GetString("api-key")
	}
	if cmd.Flags().Changed("provider") {
		p.Provider, _ = cmd.Flags().GetString("provider")
	}
	if cmd.Flags().Changed("openai-api-key") {
		p.OpenAIAPIKey, _ = cmd.Flags().GetString("openai-api-key")
	}
	if cmd.Flags().Changed("openai-base-url") {
		p.OpenAIBaseURL, _ = cmd.Flags().GetString("openai-base-url")
	}
	if err := config.SaveAgentsConfig(cfg.Storage.DataDir, agentsCfg); err != nil {
		return err
	}
	fmt.Printf("Agent profile %q updated.\n", name)
	return nil
}
```

Note: this no longer calls `UpdateProfile` — we mutate the pointer returned by `FindProfile` directly (which modifies the slice in place).

**Step 2: Run tests**

```bash
go test ./internal/config/...
go build ./cmd/betar/
```
Expected: compiles cleanly.

**Step 3: Commit**

```bash
git add cmd/betar/main.go
git commit -m "fix: use flag Changed() detection in agentConfigEdit so --price 0 works"
```

---

### Task 3: Fix hardcoded 30s announce interval in `serveAgent`

`serveAgent` calls `loadAndRegisterAgentsFromConfig(ctx, 30*time.Second)` but the `--announce-interval` flag is only on `startCmd`, not `agentServeCmd`.

**Files:**
- Modify: `cmd/betar/main.go` — `init()` and `serveAgent`

**Step 1: Add `--announce-interval` flag to `agentServeCmd` in `init()`**

After the existing `agentServeCmd` flags (around line ~254), add:
```go
agentServeCmd.Flags().Duration("announce-interval", 30*time.Second, "How often to republish agent CRDT listing")
```

**Step 2: Read the flag in `serveAgent` and use it**

In `serveAgent`, before the `loadAndRegisterAgentsFromConfig` call, add:
```go
announceInterval, _ := cmd.Flags().GetDuration("announce-interval")
if announceInterval < 5*time.Second {
    announceInterval = 5 * time.Second
}
```

Then change the hardcoded call:
```go
// Before:
if err := loadAndRegisterAgentsFromConfig(ctx, 30*time.Second); err != nil {

// After:
if err := loadAndRegisterAgentsFromConfig(ctx, announceInterval); err != nil {
```

**Step 3: Run build**

```bash
go build ./cmd/betar/
```
Expected: compiles cleanly.

**Step 4: Commit**

```bash
git add cmd/betar/main.go
git commit -m "fix: respect --announce-interval in serveAgent (was hardcoded to 30s)"
```

---

### Task 4: Remove redundant fallback logic in `loadAndRegisterAgentsFromConfig`

The ~20-line block that merges `profile.*` with `cfg.Agent.*` before calling `RegisterAgent` is dead code: `RegisterAgent` already does the identical fallback via `m.defaultCfg`. Remove it and pass `profile.*` fields directly.

**Files:**
- Modify: `cmd/betar/main.go` — `loadAndRegisterAgentsFromConfig` (line ~888)

**Step 1: Simplify the loop body**

Replace the entire per-profile block (from the local variable declarations through `RegisterAgent` call) with:

```go
for _, profile := range agentsCfg.Agents {
    registered, err := agentManager.RegisterAgent(ctx, agent.AgentSpec{
        Name:          profile.Name,
        Description:   profile.Description,
        Price:         profile.Price,
        Model:         profile.Model,
        APIKey:        profile.APIKey,
        X402Support:   true,
        Services:      []types.Service{{Name: profile.Name, Version: "1.0.0"}},
        Provider:      profile.Provider,
        OpenAIAPIKey:  profile.OpenAIAPIKey,
        OpenAIBaseURL: profile.OpenAIBaseURL,
    })
    if err != nil {
        fmt.Printf("warning: failed to register agent %q from config: %v\n", profile.Name, err)
        continue
    }
    // ... rest of listing/announcer code unchanged
```

**Step 2: Run tests and build**

```bash
go build ./...
go test ./...
```
Expected: compiles and passes.

**Step 3: Commit**

```bash
git add cmd/betar/main.go
git commit -m "refactor: remove redundant fallback in loadAndRegisterAgentsFromConfig (RegisterAgent handles it)"
```

---

### Task 5: Add tests for `internal/config/agents.go`

This is new I/O and CRUD code with no test coverage.

**Files:**
- Create: `internal/config/agents_test.go`

**Step 1: Write the test file**

```go
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
	// UpdateProfile only updates non-zero — price 0 is NOT updated (known limitation).
	// agentConfigEdit bypasses this using Changed() detection.
	if err := cfg.UpdateProfile("bot", AgentProfile{Price: 0.005}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if cfg.Agents[0].Price != 0.005 {
		t.Fatalf("expected 0.005, got %f", cfg.Agents[0].Price)
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
	if err := os.WriteFile(path, []byte(":::invalid yaml:::"), 0o600); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	_, err := LoadAgentsConfig(dir)
	if err == nil {
		t.Fatal("expected parse error for invalid YAML, got nil")
	}
}
```

**Step 2: Run the new tests to verify they pass**

```bash
go test ./internal/config/... -v -run TestLoad
go test ./internal/config/... -v -run TestSave
go test ./internal/config/... -v -run TestAdd
go test ./internal/config/... -v -run TestDelete
go test ./internal/config/... -v -run TestUpdate
go test ./internal/config/... -v -run TestValidate
```
Expected: all PASS.

**Step 3: Run full test suite**

```bash
go test ./...
```
Expected: all PASS.

**Step 4: Commit**

```bash
git add internal/config/agents_test.go
git commit -m "test: add unit tests for config/agents.go CRUD and I/O"
```

---

### Task 6: Add missing env vars to README

`LLM_PROVIDER`, `OPENAI_API_KEY`, `OPENAI_BASE_URL` are loaded by `LoadConfig` but absent from the README env table.

**Files:**
- Modify: `README.md` (around line 167–175)

**Step 1: Update the env table**

```markdown
| Variable | Default | Description |
|---|---|---|
| `GOOGLE_API_KEY` | — | Gemini model access (required for Google provider) |
| `GOOGLE_MODEL` | `gemini-2.5-flash` | Default ADK model |
| `LLM_PROVIDER` | — | `google`, `openai`, or empty for auto-detect |
| `OPENAI_API_KEY` | — | OpenAI-compatible API key |
| `OPENAI_BASE_URL` | — | OpenAI-compatible base URL (e.g. Ollama) |
| `BOOTSTRAP_PEERS` | — | Comma-separated multiaddrs |
| `BETAR_DATA_DIR` | `~/.betar` | Local data directory |
| `BETAR_P2P_KEY_PATH` | `~/.betar/p2p_identity.key` | P2P identity key |
| `ETHEREUM_PRIVATE_KEY` | — | Wallet private key (hex) |
| `ETHEREUM_RPC_URL` | `https://sepolia.base.org` | RPC endpoint |
```

Also remove `framework: google-adk` line from the agents.yaml example in the README (line ~191) since framework is being removed in Task 1 — coordinate with Task 1 if running independently.

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add LLM_PROVIDER, OPENAI_API_KEY, OPENAI_BASE_URL to env table in README"
```

---

## Summary of commits (in order)

1. `chore: remove dead --framework flag (never read by agent runtime)`
2. `fix: use flag Changed() detection in agentConfigEdit so --price 0 works`
3. `fix: respect --announce-interval in serveAgent (was hardcoded to 30s)`
4. `refactor: remove redundant fallback in loadAndRegisterAgentsFromConfig`
5. `test: add unit tests for config/agents.go CRUD and I/O`
6. `docs: add LLM_PROVIDER, OPENAI_API_KEY, OPENAI_BASE_URL to env table in README`

Tasks 1 and 6 both touch README — if implementing sequentially, merge the README edits into one commit, or do Task 6 first and amend it into Task 1's commit.
