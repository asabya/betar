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

Interactive wizard using `charmbracelet/huh` (already a dependency via TUI). Runs through three steps:

**Step 1 — LLM Provider:**
- Select provider: Google (Gemini) or OpenAI/Ollama
- Enter API key
- Optionally override model (default: `gemini-2.5-flash`)

**Step 2 — Wallet (optional):**
- Generate new wallet, import existing private key, or skip
- If generated, saves key to `~/.betar/wallet.key`

**Step 3 — Agent Profile (optional):**
- Agent name, description, price per task in USDC
- All optional with sensible defaults

**Output:**
- Writes `~/.betar/config.yaml`
- Prints summary and `betar start` as next step

### 2. Unified Config File (`~/.betar/config.yaml`)

```yaml
llm:
  provider: google          # google | openai
  api_key: "sk-..."
  model: "gemini-2.5-flash"
  # openai-specific (optional)
  # base_url: "http://localhost:11434/v1/"

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

**Precedence order:** CLI flags > env vars > config.yaml > defaults

Existing users with env vars are unaffected — config.yaml is a convenient alternative, not a replacement.

### 3. Integration with Existing Code

**New files:**
- `cmd/betar/onboard.go` — Cobra command with huh forms, writes config.yaml

**Modified files:**
- `internal/config/config.go` — After loading env vars, fill unset fields from `~/.betar/config.yaml`
- `cmd/betar/main.go` — Register `onboard` command; on `betar start` with no config, hint to run `betar onboard`

**No changes to:**
- P2P, marketplace, agent, IPFS, eth packages
- TUI, API server
- Existing `agents.yaml` flow (still works alongside)

### 4. User Experience

```bash
# First time
go install github.com/asabya/betar/cmd/betar@latest
betar onboard        # Interactive wizard → writes ~/.betar/config.yaml
betar start          # Just works

# Re-configure anytime
betar onboard        # Overwrites config.yaml

# Existing users (no changes needed)
export GOOGLE_API_KEY=...
betar start --name my-agent   # Still works exactly as before
```

## Scope

- ~2 new files, ~2 modified files
- No breaking changes to existing behavior
- Total estimated additions: ~300-400 lines
