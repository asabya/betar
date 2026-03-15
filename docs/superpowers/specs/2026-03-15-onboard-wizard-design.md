# Betar Onboard Wizard & Unified Config

**Date:** 2026-03-15
**Issue:** #20
**Approach:** Wizard Layer — add onboarding on top of existing code

## Problem

Betar requires users to manually set 9+ environment variables, understand key management, and know which components are optional. There is no guided setup, and config is scattered across env vars, YAML, CLI flags, and key files.

## Solution

Add a `betar onboard` interactive wizard that writes a unified `~/.betar/config.yaml`, and teach existing config loading to read from it as a fallback.

## Design

### 1. `betar onboard` Command

Interactive wizard using `charmbracelet/huh` (new dependency — `bubbles`/`bubbletea` are already in the project but `huh` provides higher-level form primitives better suited for wizards). Runs through three steps:

**Step 1 — LLM Provider:**
- Select provider: Google (Gemini) or OpenAI/Ollama
- Enter API key
- If OpenAI/Ollama: also prompt for base URL (default: `https://api.openai.com/v1/`, suggest `http://localhost:11434/v1/` for Ollama)
- Optionally override model (default: `gemini-2.5-flash` for Google, `gpt-4o` for OpenAI)

**Step 2 — Wallet (optional):**
- Generate new wallet, import existing private key, or skip
- If generated, saves key to `~/.betar/wallet.key`
- If existing `wallet.key` detected, inform user and offer to keep it or regenerate

**Step 3 — Agent Profile (optional):**
- Agent name, description, price per task in USDC
- All optional with sensible defaults

**Re-run behavior:**
- If `config.yaml` already exists, pre-populate all fields with current values as defaults (user presses Enter to keep)
- If existing `wallet.key` is detected and user chooses "generate new wallet," warn explicitly: "This will replace your existing wallet key. Any funds in the old wallet will be inaccessible. Continue? [y/N]"

**Output:**
- Writes `~/.betar/config.yaml`
- Prints summary and `betar start` as next step

### 2. Unified Config File (`~/.betar/config.yaml`)

Located at `$BETAR_DATA_DIR/config.yaml` (defaults to `~/.betar/config.yaml`).

```yaml
llm:
  provider: google          # google | openai
  api_key: "sk-..."
  model: "gemini-2.5-flash"
  base_url: ""              # only for openai provider

wallet:
  private_key: ""           # empty = use ~/.betar/wallet.key file
  rpc_url: "https://sepolia.base.org"

p2p:
  port: 4001
  bootstrap_peers: []

agent:
  name: "my-agent"
  description: "A helpful assistant"
  price: 0.001
```

**Go struct** (new, in `internal/config/`):

```go
type FileConfig struct {
    LLM    LLMFileConfig    `yaml:"llm"`
    Wallet WalletFileConfig `yaml:"wallet"`
    P2P    P2PFileConfig    `yaml:"p2p"`
    Agent  AgentFileConfig  `yaml:"agent"`
}

type LLMFileConfig struct {
    Provider string `yaml:"provider"`
    APIKey   string `yaml:"api_key"`
    Model    string `yaml:"model"`
    BaseURL  string `yaml:"base_url"`
}

type WalletFileConfig struct {
    PrivateKey string `yaml:"private_key"`
    RPCURL     string `yaml:"rpc_url"`
}

type P2PFileConfig struct {
    Port           int      `yaml:"port"`
    BootstrapPeers []string `yaml:"bootstrap_peers"`
}

type AgentFileConfig struct {
    Name        string  `yaml:"name"`
    Description string  `yaml:"description"`
    Price       float64 `yaml:"price"`
}
```

**Mapping to existing `Config` fields:**
- `LLM.Provider` → determines which env-var-equivalent to set (`google` → `AgentConfig.APIKey`, `openai` → `AgentConfig.OpenAIAPIKey`)
- `LLM.APIKey` → `AgentConfig.APIKey` (Google) or `AgentConfig.OpenAIAPIKey` (OpenAI)
- `LLM.Model` → `AgentConfig.Model`
- `LLM.BaseURL` → `AgentConfig.OpenAIBaseURL`
- `Wallet.PrivateKey` → `EthereumConfig.PrivateKey`
- `Wallet.RPCURL` → `EthereumConfig.RPCURL`
- `P2P.Port` → `P2PConfig.Port`
- `P2P.BootstrapPeers` → `P2PConfig.BootstrapPeers`
- `Agent.*` → used by `betar start` to auto-register an agent (same as current `--name`/`--description`/`--price` flags)

Parsed using `gopkg.in/yaml.v3` (already in project via `agents.go`).

**Precedence order:** CLI flags > env vars > config.yaml > defaults

Existing users with env vars are unaffected — config.yaml is a convenient alternative, not a replacement.

### 3. Relationship with `agents.yaml`

`config.yaml` and `agents.yaml` serve different purposes:
- **`config.yaml`** — global node config: LLM credentials, wallet, P2P settings, and a single default agent profile
- **`agents.yaml`** — multi-agent definitions for users running multiple agents on one node

When both exist: `agents.yaml` agent definitions take priority over `config.yaml`'s `agent` section. If `agents.yaml` has entries, they are used. If it's absent or empty, `config.yaml`'s `agent` section is used as a single-agent fallback.

LLM credentials in `config.yaml` serve as defaults for agents in `agents.yaml` that don't specify their own `api_key`.

### 4. Integration with Existing Code

**New files:**
- `cmd/betar/onboard.go` — Cobra command with huh forms, writes config.yaml
- `internal/config/fileconfig.go` — `FileConfig` struct, load/save functions

**Modified files:**
- `internal/config/config.go` — After loading env vars, call `LoadFileConfig()` to fill unset fields
- `cmd/betar/main.go` — Register `onboard` command; on `betar start` with no config, hint to run `betar onboard`

**New dependency:**
- `github.com/charmbracelet/huh` — form library for interactive wizard

**No changes to:**
- P2P, marketplace, agent, IPFS, eth packages
- TUI, API server
- Existing `agents.yaml` flow (still works alongside)

### 5. User Experience

```bash
# First time
go install github.com/asabya/betar/cmd/betar@latest
betar onboard        # Interactive wizard → writes ~/.betar/config.yaml
betar start          # Just works

# Re-configure anytime
betar onboard        # Pre-populates with current values, confirms before overwriting wallet

# Existing users (no changes needed)
export GOOGLE_API_KEY=...
betar start --name my-agent   # Still works exactly as before
```

## Out of Scope

- Non-interactive mode (`betar onboard --api-key=...`) — deferred for later
- Refactoring `initRuntime()` or splitting `main.go` — tracked in #20/#21
- `betar doctor` health check command — tracked in #21

## Scope

- ~3 new files, ~2 modified files
- 1 new dependency (`charmbracelet/huh`)
- No breaking changes to existing behavior
- Total estimated additions: ~400-500 lines
