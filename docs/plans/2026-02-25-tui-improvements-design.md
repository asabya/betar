# TUI Improvements Design

**Date:** 2026-02-25
**Branch:** feature/betar-tui
**Status:** Approved

---

## Overview

Three improvements to the Betar Bubbletea TUI:

1. **Right panel: wallet address + scrollable local agents list** — populate the already-rendered but always-empty `walletAddr`/`did` fields; replace single DID with a per-agent list showing Name + DID, scrollable via viewport.
2. **TUI = `start` equivalent** — add optional agent-startup flags to root command so running `betar --name my-agent` behaves like `betar start --name my-agent` (registers agent, publishes IPFS presence, starts API server, runs listing announcer).
3. **Dual autocomplete** — dropdown suggestion list + inline ghost text as user types commands. Tab fills ghost text / cycles matches; Up/Down navigates dropdown.

---

## Feature 1: Right Panel — Wallet + Scrollable Agents

### What's broken today

`view.go:renderRightPanel` already references `m.walletAddr`, `m.did`, but these are never set. The right panel is a static string, not scrollable.

### Design

**New model fields:**
```go
type agentInfo struct {
    Name string
    DID  string
}

// in model struct:
rightViewport viewport.Model
agents        []agentInfo
// walletAddr already exists
// remove: did string (replaced by per-agent DIDs)
```

**`commands.go`:**
- Add `runtimeWalletAddr string` package var
- Add `SetWallet(addr string)` exported function
- Add `getWalletAddr() string` helper

**`update.go` — expand `nodeInfoMsg`:**
```go
type nodeInfoMsg struct {
    peerID         string
    addresses      []string
    connectedPeers int
    walletAddr     string
    agents         []agentInfo
}
```
`fetchNodeInfo()` calls `getAgentManager().ListAgents()` and `getWalletAddr()` in addition to P2P info.

In `Update()` `nodeInfoMsg` case: set `m.walletAddr`, `m.agents`, then rebuild right panel content string and call `m.rightViewport.SetContent(...)`.

In `tea.WindowSizeMsg` case: also resize `m.rightViewport`.

**`view.go`:**
- `renderRightPanel` builds content string (node status + agents list) and renders `m.rightViewport.View()` inside the panel style.
- Right viewport is scrollable (user can scroll with Up/Down when suggestions are not active — or Page Up/Down).

**`main.go`:**
- After `initRuntime`, derive wallet address from `cfg.Ethereum.PrivateKey` if non-empty using `crypto.PubkeyToAddress`. Pass via `tui.SetWallet(addr)`.

---

## Feature 2: TUI = `start` Equivalent

### What's missing today

`runTUI` calls `initRuntime` but skips: `publishNodePresence`, `registerLocalAgentFromFlags`, API server start, and listing announcer goroutine.

### Design

**Add optional flags to root command in `init()`:**
```
--name             Agent name (optional; if set, agent is registered on boot)
--description      Agent description
--price            Price per task (default 0)
--endpoint         Agent endpoint (default "p2p://local")
--framework        Agent framework (default "adk")
--model            ADK model name (default "gemini-2.5-flash")
--x402             Support EIP-402 payments
--announce-interval  How often to republish listing (default 30s)
--api-port         HTTP API server port (default 8424)
```

**`runTUI()` flow after this change:**
```
initRuntime(cmd)
SetRuntime(p2pHost, agentManager, listingService, orderService)
SetWallet(walletAddr)           // new
stdout pipe redirect

if --name is set:
    publishNodePresence(ctx)
    registerLocalAgentFromFlags(ctx, cmd)
    listingService.UpsertLocalListing(listingMsg)
    apiServer = api.NewServer(...); apiServer.Start()
    go runListingAnnouncer(ctx, interval, ...)

fmt.Println("Starting Betar TUI...")
printRuntimeInfo()
tui.RunTUI()
```

No changes needed to `shutdownRuntime()` — it already handles `apiServer` nil-safely.

---

## Feature 3: Dual Autocomplete

### Design

**Known commands list** (`commands.go`):
```go
var knownCommands = []string{
    "/help",
    "/status",
    "/peers",
    "/exit",
    "/wallet balance",
    "/agent list",
    "/agent discover",
    "/agent execute ",
    "/order create ",
}
```

**New model fields:**
```go
suggestions    []string
suggestionIdx  int
```

**Suggestion logic** (`update.go`):
- After `m.cmdInput.Update(msg)` for any key that isn't Enter/Up/Down/Tab/Escape: recompute `m.suggestions` by filtering `knownCommands` where `strings.HasPrefix(cmd, input)` and `input != cmd`. Reset `m.suggestionIdx = 0`.
- **Tab key**: if suggestions exist, fill `m.cmdInput` with `m.suggestions[m.suggestionIdx]`; increment `m.suggestionIdx` (mod len); do not submit. If no suggestions, do nothing.
- **Up key**: if suggestions visible, decrement `m.suggestionIdx` (mod len), consume event. Otherwise, history nav (existing behavior).
- **Down key**: if suggestions visible, increment `m.suggestionIdx` (mod len), consume event. Otherwise, history nav.
- **Escape**: clear `m.suggestions`, clear input.
- **Enter**: if suggestions visible and input exactly matches a suggestion prefix (not the full command), fill with the highlighted suggestion instead of executing. If input is a full/valid command, execute normally.

**Ghost text** (`view.go`):
- `renderInput` checks `len(m.suggestions) > 0`. If so, computes `ghost = strings.TrimPrefix(m.suggestions[m.suggestionIdx], m.cmdInput.Value())` and appends it in a dimmed style after the input view.

**Dropdown** (`view.go`):
- `renderSuggestions(width int)` renders a small lipgloss-styled box listing `m.suggestions` (up to 5). The `m.suggestionIdx` entry is highlighted. Rendered between log panel and input panel.

**Layout change** in `View()`:
```
leftCol = JoinVertical(logs, suggestions box [conditional], input)
```

---

## Files Changed

| File | Changes |
|------|---------|
| `cmd/betar/main.go` | Add flags to root cmd; expand `runTUI()` to do full start when `--name` set; derive + pass wallet addr |
| `cmd/betar/tui/commands.go` | Add `runtimeWalletAddr`, `SetWallet()`, `getWalletAddr()`, `knownCommands` list |
| `cmd/betar/tui/model.go` | Add `rightViewport`, `agents []agentInfo`, `suggestions`, `suggestionIdx`; remove `did` field; init rightViewport in `NewModel()` |
| `cmd/betar/tui/update.go` | Expand `nodeInfoMsg`; update `fetchNodeInfo()`; handle Tab/Up/Down/Escape for suggestions; update right viewport on resize; recompute suggestions on input change |
| `cmd/betar/tui/view.go` | `renderRightPanel` uses viewport; add `renderSuggestions()`; ghost text in `renderInput`; join suggestions into left column |

No new files needed.

---

## Verification

1. `make build` — clean build
2. `./bin/betar --name test-agent --price 0.01` — right panel shows wallet address and registered agent with DID; API server starts
3. Type `/ag` in TUI — dropdown shows `/agent list`, `/agent discover`, `/agent execute `; ghost text shows `/ent list`
4. Tab fills `/agent list`; Tab again cycles to `/agent discover`
5. Up/Down navigates dropdown when visible
6. Escape clears suggestions
7. Right panel scrollable with Page Up/Down
