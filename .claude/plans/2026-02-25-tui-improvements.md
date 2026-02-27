# TUI Improvements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add three improvements to the Betar Bubbletea TUI: wallet+agents display in scrollable right panel, TUI behaves like `start` command when `--name` flag is set, and dual autocomplete (dropdown + ghost text).

**Architecture:** All changes stay within `cmd/betar/tui/` and `cmd/betar/main.go`. No new files. Model gains `agentInfo` type + `rightViewport`, `agents`, `suggestions`, `suggestionIdx` fields. Commands.go gains wallet setter and known-commands list. Main.go gains optional startup flags and agent lifecycle.

**Tech Stack:** Bubbletea, lipgloss, charmbracelet/bubbles/viewport, go-ethereum crypto

---

### Task 1: Add agentInfo type and new model fields

**Files:**
- Modify: `cmd/betar/tui/model.go`

The model currently has `did string` which we replace with `agents []agentInfo`. We also add `rightViewport`, `suggestions`, and `suggestionIdx`.

**Step 1: Open the file**

Read `cmd/betar/tui/model.go`. It currently looks like:

```go
package tui

import (
    "strings"

    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/bubbles/viewport"
)

var logCh = make(chan string, 256)

func SendLog(msg string) {
    select {
    case logCh <- msg:
    default:
    }
}

type model struct {
    p2pHost        interface{}
    agentManager   interface{}
    listingService interface{}
    orderService   interface{}

    logsViewport viewport.Model
    cmdInput     textinput.Model

    logs         []string
    cmdHistory   []string
    historyIndex int

    width  int
    height int

    peerID         string
    addresses      []string
    connectedPeers int
    walletAddr     string
    did            string

    pendingTasks []string
}

func NewModel() model {
    vp := viewport.New(80, 20)
    ti := textinput.New()
    ti.Placeholder = "/agent list"
    ti.Prompt = "> "
    ti.Focus()

    initLogs := []string{"Betar TUI started. Type /help for commands."}
    vp.SetContent(strings.Join(initLogs, "\n"))

    return model{
        logsViewport: vp,
        cmdInput:     ti,
        logs:         initLogs,
    }
}
```

**Step 2: Replace the file contents**

Replace `cmd/betar/tui/model.go` with:

```go
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

var logCh = make(chan string, 256)

func SendLog(msg string) {
	select {
	case logCh <- msg:
	default:
	}
}

// agentInfo holds display info for a local agent in the right panel.
type agentInfo struct {
	Name string
	DID  string
}

type model struct {
	p2pHost        interface{}
	agentManager   interface{}
	listingService interface{}
	orderService   interface{}

	logsViewport  viewport.Model
	rightViewport viewport.Model
	cmdInput      textinput.Model

	logs         []string
	cmdHistory   []string
	historyIndex int

	width  int
	height int

	peerID         string
	addresses      []string
	connectedPeers int
	walletAddr     string
	agents         []agentInfo

	suggestions   []string
	suggestionIdx int

	pendingTasks []string
}

func NewModel() model {
	vp := viewport.New(80, 20)
	rvp := viewport.New(30, 20)
	ti := textinput.New()
	ti.Placeholder = "/agent list"
	ti.Prompt = "> "
	ti.Focus()

	initLogs := []string{"Betar TUI started. Type /help for commands."}
	vp.SetContent(strings.Join(initLogs, "\n"))

	return model{
		logsViewport:  vp,
		rightViewport: rvp,
		cmdInput:      ti,
		logs:          initLogs,
	}
}
```

**Step 3: Verify it builds**

```bash
cd /Users/sabyasachipatra/go/src/github.com/asabya/betar && make build
```

Expected: **FAIL** — `view.go` references `m.did` which no longer exists.

**Step 4: Fix the compile error in view.go**

In `cmd/betar/tui/view.go`, remove the `did` display. Replace the entire `renderRightPanel` function with a stub that uses the viewport (we'll fill it properly in Task 5):

Find this block in `view.go`:

```go
func (m model) renderRightPanel(width, height int) string {
	peerID := m.peerID
	if peerID == "" {
		peerID = "(not started)"
	}
	walletAddr := m.walletAddr
	if walletAddr == "" {
		walletAddr = "(none)"
	}
	did := m.did
	if did == "" {
		did = "(none)"
	}

	nodeStatus := TitleStyle.Render("Node Status") + "\n"
	nodeStatus += "PeerID: " + peerID + "\n"
	nodeStatus += "Addrs:  " + formatAddrs(m.addresses) + "\n"
	nodeStatus += fmt.Sprintf("Peers:  %d\n", m.connectedPeers)
	nodeStatus += "Wallet: " + walletAddr + "\n"
	nodeStatus += "DID:    " + did + "\n"

	tasks := "\n" + TitleStyle.Render("Incoming Tasks") + "\n"
	if len(m.pendingTasks) == 0 {
		tasks += "  (none)"
	} else {
		for _, t := range m.pendingTasks {
			tasks += "- " + t + "\n"
		}
	}

	return PanelStyle.Width(width - 2).Height(height - 2).Render(nodeStatus + tasks)
}
```

Replace with:

```go
func (m model) renderRightPanel(width, height int) string {
	peerID := m.peerID
	if peerID == "" {
		peerID = "(not started)"
	}
	walletAddr := m.walletAddr
	if walletAddr == "" {
		walletAddr = "(none)"
	}

	content := TitleStyle.Render("Node Status") + "\n"
	content += "PeerID: " + peerID + "\n"
	content += "Addrs:  " + formatAddrs(m.addresses) + "\n"
	content += fmt.Sprintf("Peers:  %d\n", m.connectedPeers)
	content += "Wallet: " + walletAddr + "\n"

	content += "\n" + TitleStyle.Render("Local Agents") + "\n"
	if len(m.agents) == 0 {
		content += "  (none)\n"
	} else {
		for _, a := range m.agents {
			content += fmt.Sprintf("  %s\n  %s\n\n", a.Name, a.DID)
		}
	}

	content += "\n" + TitleStyle.Render("Incoming Tasks") + "\n"
	if len(m.pendingTasks) == 0 {
		content += "  (none)"
	} else {
		for _, t := range m.pendingTasks {
			content += "- " + t + "\n"
		}
	}

	m.rightViewport.SetContent(content)
	return PanelStyle.Width(width - 2).Height(height - 2).Render(m.rightViewport.View())
}
```

**Step 5: Verify it builds**

```bash
make build
```

Expected: **PASS** — clean build with no errors.

**Step 6: Commit**

```bash
cd /Users/sabyasachipatra/go/src/github.com/asabya/betar
git add cmd/betar/tui/model.go cmd/betar/tui/view.go
git commit -m "feat(tui): add agentInfo type, rightViewport, suggestions fields to model"
```

---

### Task 2: Add wallet setter and knownCommands to commands.go

**Files:**
- Modify: `cmd/betar/tui/commands.go`

**Step 1: Add wallet var, setter, getter, and knownCommands after the existing runtime vars**

In `cmd/betar/tui/commands.go`, find the existing var block:

```go
var (
	runtimeP2pHost        *p2p.Host
	runtimeAgentManager   *agent.Manager
	runtimeListingService *marketplace.AgentListingService
	runtimeOrderService   *marketplace.OrderService
)
```

Replace it with:

```go
var (
	runtimeP2pHost        *p2p.Host
	runtimeAgentManager   *agent.Manager
	runtimeListingService *marketplace.AgentListingService
	runtimeOrderService   *marketplace.OrderService
	runtimeWalletAddr     string
)

// knownCommands is the list used for autocomplete suggestions.
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

**Step 2: Add SetWallet and getWalletAddr functions after the existing SetRuntime function**

Find the `SetRuntime` function at the bottom of `commands.go`:

```go
func SetRuntime(p2pHost *p2p.Host, agentManager *agent.Manager, listingService *marketplace.AgentListingService, orderService *marketplace.OrderService) {
	runtimeP2pHost = p2pHost
	runtimeAgentManager = agentManager
	runtimeListingService = listingService
	runtimeOrderService = orderService
}
```

Add the following two functions immediately after it:

```go
// SetWallet sets the Ethereum wallet address for display in the TUI.
func SetWallet(addr string) {
	runtimeWalletAddr = addr
}

func getWalletAddr() string {
	return runtimeWalletAddr
}
```

**Step 3: Also remove the now-unused `hasPrefix` helper** (it wraps `strings.HasPrefix` — dead code)

Find and delete:

```go
func hasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}
```

**Step 4: Verify it builds**

```bash
make build
```

Expected: **PASS**

**Step 5: Commit**

```bash
git add cmd/betar/tui/commands.go
git commit -m "feat(tui): add SetWallet, getWalletAddr, knownCommands for autocomplete"
```

---

### Task 3: Expand nodeInfoMsg and fetchNodeInfo to include wallet + agents

**Files:**
- Modify: `cmd/betar/tui/update.go`

**Step 1: Update the nodeInfoMsg struct**

Find the current `nodeInfoMsg` struct in `update.go`:

```go
type nodeInfoMsg struct {
	peerID         string
	addresses      []string
	connectedPeers int
}
```

Replace with:

```go
type nodeInfoMsg struct {
	peerID         string
	addresses      []string
	connectedPeers int
	walletAddr     string
	agents         []agentInfo
}
```

**Step 2: Expand fetchNodeInfo to also collect wallet and agents**

Find the current `fetchNodeInfo` function:

```go
func fetchNodeInfo() tea.Cmd {
	return func() tea.Msg {
		h := getP2PHost()
		if h == nil {
			return nil
		}
		peers := h.RawHost().Network().Peers()
		return nodeInfoMsg{
			peerID:         h.ID().String(),
			addresses:      h.AddrStrings(),
			connectedPeers: len(peers),
		}
	}
}
```

Replace with:

```go
func fetchNodeInfo() tea.Cmd {
	return func() tea.Msg {
		h := getP2PHost()
		if h == nil {
			return nil
		}
		peers := h.RawHost().Network().Peers()

		var agents []agentInfo
		if mgr := getAgentManager(); mgr != nil {
			for _, a := range mgr.ListAgents() {
				agents = append(agents, agentInfo{Name: a.Name, DID: a.AgentID})
			}
		}

		return nodeInfoMsg{
			peerID:         h.ID().String(),
			addresses:      h.AddrStrings(),
			connectedPeers: len(peers),
			walletAddr:     getWalletAddr(),
			agents:         agents,
		}
	}
}
```

**Step 3: Update the nodeInfoMsg case in Update() to set wallet and agents**

Find this section in `Update()`:

```go
case nodeInfoMsg:
    m.peerID = msg.peerID
    m.addresses = msg.addresses
    m.connectedPeers = msg.connectedPeers
    return m, nil
```

Replace with:

```go
case nodeInfoMsg:
    m.peerID = msg.peerID
    m.addresses = msg.addresses
    m.connectedPeers = msg.connectedPeers
    m.walletAddr = msg.walletAddr
    m.agents = msg.agents
    return m, nil
```

**Step 4: Update WindowSizeMsg to also resize rightViewport**

Find this block in `Update()`:

```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    leftW := msg.Width * 2 / 3
    logH := msg.Height * 2 / 3
    m.logsViewport = viewport.New(leftW-4, logH-2)
```

Replace with:

```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    leftW := msg.Width * 2 / 3
    rightW := msg.Width - leftW
    logH := msg.Height * 2 / 3
    m.logsViewport = viewport.New(leftW-4, logH-2)
    m.rightViewport = viewport.New(rightW-4, msg.Height-4)
```

**Step 5: Verify it builds**

```bash
make build
```

Expected: **PASS**

**Step 6: Commit**

```bash
git add cmd/betar/tui/update.go
git commit -m "feat(tui): expand nodeInfoMsg to include wallet addr and local agents"
```

---

### Task 4: Add autocomplete key handling in update.go

**Files:**
- Modify: `cmd/betar/tui/update.go`

This task wires up Tab, Escape, and the suggestion-aware Up/Down in `Update()`, and recomputes suggestions after any input change.

**Step 1: Add the computeSuggestions helper function** at the bottom of `update.go`:

```go
// computeSuggestions returns commands from knownCommands that start with
// the given input string but are not equal to it (i.e., the input is a prefix).
func computeSuggestions(input string) []string {
	if input == "" {
		return nil
	}
	var result []string
	for _, cmd := range knownCommands {
		if strings.HasPrefix(cmd, input) && cmd != input {
			result = append(result, cmd)
		}
	}
	return result
}
```

**Step 2: Replace the key-handling block in Update()**

Find the full key-handling section:

```go
case tea.KeyMsg:
    switch msg.String() {
    case "ctrl+c":
        return m, tea.Quit
    case "enter":
        return m.handleCommand()
    case "up":
        if m.historyIndex > 0 {
            m.historyIndex--
            if m.historyIndex < len(m.cmdHistory) {
                m.cmdInput.SetValue(m.cmdHistory[m.historyIndex])
            }
        }
        return m, nil
    case "down":
        if m.historyIndex < len(m.cmdHistory) {
            m.historyIndex++
            if m.historyIndex < len(m.cmdHistory) {
                m.cmdInput.SetValue(m.cmdHistory[m.historyIndex])
            } else {
                m.cmdInput.SetValue("")
            }
        }
        return m, nil
    }
```

Replace with:

```go
case tea.KeyMsg:
    switch msg.String() {
    case "ctrl+c":
        return m, tea.Quit
    case "enter":
        // If a suggestion is highlighted, fill it in instead of executing.
        if len(m.suggestions) > 0 {
            m.cmdInput.SetValue(m.suggestions[m.suggestionIdx])
            m.suggestions = nil
            m.suggestionIdx = 0
            return m, nil
        }
        return m.handleCommand()
    case "tab":
        if len(m.suggestions) > 0 {
            m.cmdInput.SetValue(m.suggestions[m.suggestionIdx])
            m.suggestionIdx = (m.suggestionIdx + 1) % len(m.suggestions)
        }
        return m, nil
    case "esc":
        m.suggestions = nil
        m.suggestionIdx = 0
        m.cmdInput.SetValue("")
        return m, nil
    case "up":
        if len(m.suggestions) > 0 {
            if m.suggestionIdx > 0 {
                m.suggestionIdx--
            } else {
                m.suggestionIdx = len(m.suggestions) - 1
            }
            return m, nil
        }
        if m.historyIndex > 0 {
            m.historyIndex--
            if m.historyIndex < len(m.cmdHistory) {
                m.cmdInput.SetValue(m.cmdHistory[m.historyIndex])
            }
        }
        return m, nil
    case "down":
        if len(m.suggestions) > 0 {
            m.suggestionIdx = (m.suggestionIdx + 1) % len(m.suggestions)
            return m, nil
        }
        if m.historyIndex < len(m.cmdHistory) {
            m.historyIndex++
            if m.historyIndex < len(m.cmdHistory) {
                m.cmdInput.SetValue(m.cmdHistory[m.historyIndex])
            } else {
                m.cmdInput.SetValue("")
            }
        }
        return m, nil
    case "pgup":
        m.rightViewport.HalfViewUp()
        return m, nil
    case "pgdown":
        m.rightViewport.HalfViewDown()
        return m, nil
    }
```

**Step 3: Recompute suggestions after every input update**

At the very bottom of `Update()`, find the final two lines:

```go
m.cmdInput, cmd = m.cmdInput.Update(msg)
return m, cmd
```

Replace with:

```go
m.cmdInput, cmd = m.cmdInput.Update(msg)
// Recompute autocomplete suggestions whenever input changes.
newSuggestions := computeSuggestions(m.cmdInput.Value())
if len(newSuggestions) != len(m.suggestions) {
    m.suggestions = newSuggestions
    m.suggestionIdx = 0
}
return m, cmd
```

**Step 4: Ensure `strings` is imported in update.go**

`update.go` already imports `"strings"`. Verify the import block looks like:

```go
import (
    "strings"
    "time"

    "github.com/charmbracelet/bubbles/viewport"
    tea "github.com/charmbracelet/bubbletea"
)
```

**Step 5: Verify it builds**

```bash
make build
```

Expected: **PASS**

**Step 6: Commit**

```bash
git add cmd/betar/tui/update.go
git commit -m "feat(tui): add autocomplete key handling (Tab, Esc, Up/Down with suggestions)"
```

---

### Task 5: Update view.go — right viewport render + suggestion dropdown + ghost text

**Files:**
- Modify: `cmd/betar/tui/view.go`
- Modify: `cmd/betar/tui/styles.go`

**Step 1: Add suggestion styles to styles.go**

Append to `cmd/betar/tui/styles.go` after the existing styles:

```go
	SuggestionStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#555555")).
			Padding(0, 1)

	SuggestionHighlightStyle = lipgloss.NewStyle().
					Foreground(PrimaryColor).
					Bold(true)

	GhostStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555"))
```

The full `styles.go` should now look like:

```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	PrimaryColor   = lipgloss.Color("#ff8258")
	SecondaryColor = lipgloss.Color("#a1a2ff")

	TitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true)

	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(SecondaryColor)

	InputStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor)

	SuggestionStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#555555")).
			Padding(0, 1)

	SuggestionHighlightStyle = lipgloss.NewStyle().
					Foreground(PrimaryColor).
					Bold(true)

	GhostStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555"))
)
```

**Step 2: Update the renderInput function in view.go to include suggestions and ghost text**

Find the current `renderInput`:

```go
func (m model) renderInput(width, height int) string {
	return InputStyle.Width(width - 2).Height(height - 2).Render(m.cmdInput.View())
}
```

Replace with:

```go
func (m model) renderInput(width, height int) string {
	content := ""

	// Render suggestion dropdown above the input if there are matches.
	if len(m.suggestions) > 0 {
		max := 5
		if len(m.suggestions) < max {
			max = len(m.suggestions)
		}
		var lines []string
		for i := 0; i < max; i++ {
			line := m.suggestions[i]
			if i == m.suggestionIdx%max {
				line = SuggestionHighlightStyle.Render("▶ " + line)
			} else {
				line = "  " + line
			}
			lines = append(lines, line)
		}
		content += SuggestionStyle.Width(width - 6).Render(strings.Join(lines, "\n")) + "\n"
	}

	// Render the input with optional ghost text suffix.
	inputView := m.cmdInput.View()
	if len(m.suggestions) > 0 {
		ghost := strings.TrimPrefix(m.suggestions[m.suggestionIdx%len(m.suggestions)], m.cmdInput.Value())
		if ghost != "" {
			inputView += GhostStyle.Render(ghost)
		}
	}
	content += inputView

	return InputStyle.Width(width - 2).Height(height - 2).Render(content)
}
```

**Step 3: Update renderRightPanel to properly use rightViewport**

The current `renderRightPanel` (updated in Task 1) calls `m.rightViewport.SetContent(content)` inside a `View()` call. In Bubbletea's architecture, setting content inside `View()` is a side effect and won't persist since models are value types. We need to move content updates to `Update()`.

Fix this by making `renderRightPanel` use the viewport's existing content (set during `Update`) and NOT call `SetContent` in `View()`:

Replace the current `renderRightPanel` in `view.go` with:

```go
func (m model) renderRightPanel(width, height int) string {
	return PanelStyle.Width(width - 2).Height(height - 2).Render(m.rightViewport.View())
}
```

Then add a helper function `buildRightPanelContent` that is called from `Update()`:

```go
// buildRightPanelContent constructs the text content for the right panel viewport.
// Call this in Update() when node info changes, then use m.rightViewport.SetContent().
func buildRightPanelContent(m model) string {
	peerID := m.peerID
	if peerID == "" {
		peerID = "(not started)"
	}
	walletAddr := m.walletAddr
	if walletAddr == "" {
		walletAddr = "(none)"
	}

	content := TitleStyle.Render("Node Status") + "\n"
	content += "PeerID: " + peerID + "\n"
	content += "Addrs:  " + formatAddrs(m.addresses) + "\n"
	content += fmt.Sprintf("Peers:  %d\n", m.connectedPeers)
	content += "Wallet: " + walletAddr + "\n"

	content += "\n" + TitleStyle.Render("Local Agents") + "\n"
	if len(m.agents) == 0 {
		content += "  (none)\n"
	} else {
		for _, a := range m.agents {
			content += fmt.Sprintf("  %s\n  %s\n\n", a.Name, a.DID)
		}
	}

	content += "\n" + TitleStyle.Render("Incoming Tasks") + "\n"
	if len(m.pendingTasks) == 0 {
		content += "  (none)"
	} else {
		for _, t := range m.pendingTasks {
			content += "- " + t + "\n"
		}
	}

	return content
}
```

**Step 4: Call buildRightPanelContent from Update() in update.go**

In `update.go`, update the `nodeInfoMsg` case to also update the right viewport:

Find:

```go
case nodeInfoMsg:
    m.peerID = msg.peerID
    m.addresses = msg.addresses
    m.connectedPeers = msg.connectedPeers
    m.walletAddr = msg.walletAddr
    m.agents = msg.agents
    return m, nil
```

Replace with:

```go
case nodeInfoMsg:
    m.peerID = msg.peerID
    m.addresses = msg.addresses
    m.connectedPeers = msg.connectedPeers
    m.walletAddr = msg.walletAddr
    m.agents = msg.agents
    m.rightViewport.SetContent(buildRightPanelContent(m))
    return m, nil
```

**Step 5: Ensure view.go imports `strings` and `fmt`**

`view.go` already imports both. Verify the import block is:

```go
import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss"
)
```

**Step 6: Verify it builds**

```bash
make build
```

Expected: **PASS**

**Step 7: Commit**

```bash
git add cmd/betar/tui/view.go cmd/betar/tui/styles.go cmd/betar/tui/update.go
git commit -m "feat(tui): scrollable right panel viewport, suggestion dropdown, ghost text autocomplete"
```

---

### Task 6: Add startup flags + full agent lifecycle to runTUI in main.go

**Files:**
- Modify: `cmd/betar/main.go`

**Step 1: Add optional flags to the root command in init()**

In `cmd/betar/main.go`, find the `func init()` block. After the existing flag registrations (e.g., the `walletBalanceCmd` flags), and just before `rootCmd.AddCommand(nodeCmd)`, add:

```go
// TUI flags — same as startCmd but all optional
rootCmd.Flags().IntP("port", "p", 4001, "Port to listen on")
rootCmd.Flags().StringSlice("bootstrap", []string{}, "Bootstrap peers")
rootCmd.Flags().String("model", "gemini-2.5-flash", "ADK model name")
rootCmd.Flags().StringP("name", "n", "", "Agent name (optional; registers agent on startup if set)")
rootCmd.Flags().StringP("description", "d", "", "Agent description")
rootCmd.Flags().Float64P("price", "r", 0, "Price per task")
rootCmd.Flags().String("endpoint", "p2p://local", "Agent endpoint")
rootCmd.Flags().String("framework", "adk", "Agent framework")
rootCmd.Flags().Bool("x402", false, "Support EIP-402 payments")
rootCmd.Flags().Duration("announce-interval", 30*time.Second, "How often to republish agent CRDT listing")
rootCmd.Flags().Int("api-port", 8424, "HTTP API server port")
```

**Step 2: Add imports for go-ethereum crypto in main.go**

Find the current import block in `main.go`. It starts with:

```go
import (
    "bufio"
    "context"
    "errors"
    "fmt"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "time"
    ...
)
```

Add two imports:

```go
"crypto/ecdsa"

"github.com/ethereum/go-ethereum/crypto"
```

So the import block includes (among others):

```go
import (
    "bufio"
    "context"
    "crypto/ecdsa"
    "errors"
    "fmt"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "time"

    "github.com/asabya/betar/cmd/betar/api"
    "github.com/asabya/betar/cmd/betar/tui"
    "github.com/asabya/betar/internal/agent"
    "github.com/asabya/betar/internal/config"
    "github.com/asabya/betar/internal/ipfs"
    "github.com/asabya/betar/internal/marketplace"
    "github.com/asabya/betar/internal/p2p"
    "github.com/asabya/betar/pkg/types"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/libp2p/go-libp2p/core/peer"
    "github.com/multiformats/go-multiaddr"
    "github.com/spf13/cobra"
)
```

**Step 3: Add a deriveWalletAddress helper function**

Add this small helper anywhere in `main.go` (e.g., just before `printRuntimeInfo`):

```go
// deriveWalletAddress derives the Ethereum address hex from a private key hex string.
// Returns empty string if the key is not set or invalid.
func deriveWalletAddress(privKeyHex string) string {
    if privKeyHex == "" {
        return ""
    }
    pk, err := crypto.HexToECDSA(privKeyHex)
    if err != nil {
        return ""
    }
    pub, ok := pk.Public().(*ecdsa.PublicKey)
    if !ok {
        return ""
    }
    return crypto.PubkeyToAddress(*pub).Hex()
}
```

**Step 4: Replace the runTUI function**

Find the current `runTUI`:

```go
func runTUI(cmd *cobra.Command, args []string) error {
    if err := initRuntime(cmd); err != nil {
        return err
    }
    defer shutdownRuntime()

    tui.SetRuntime(p2pHost, agentManager, listingService, orderService)

    origStdout := os.Stdout
    r, w, err := os.Pipe()
    if err == nil {
        os.Stdout = w
        tui.SetOutput(origStdout)
        go func() {
            scanner := bufio.NewScanner(r)
            for scanner.Scan() {
                tui.SendLog(scanner.Text())
            }
        }()
    }

    fmt.Println("Starting Betar TUI...")
    printRuntimeInfo()

    return tui.RunTUI()
}
```

Replace with:

```go
func runTUI(cmd *cobra.Command, args []string) error {
    if err := initRuntime(cmd); err != nil {
        return err
    }
    defer shutdownRuntime()

    tui.SetRuntime(p2pHost, agentManager, listingService, orderService)
    tui.SetWallet(deriveWalletAddress(cfg.Ethereum.PrivateKey))

    // Redirect stdout into the TUI log panel.
    origStdout := os.Stdout
    r, w, pipeErr := os.Pipe()
    if pipeErr == nil {
        os.Stdout = w
        tui.SetOutput(origStdout)
        go func() {
            scanner := bufio.NewScanner(r)
            for scanner.Scan() {
                tui.SendLog(scanner.Text())
            }
        }()
    }

    // If --name is provided, run the full agent lifecycle (like `start`).
    name, _ := cmd.Flags().GetString("name")
    if name != "" {
        registered, listingMsg, err := registerLocalAgentFromFlags(ctx, cmd)
        if err != nil {
            return err
        }

        if listingService != nil {
            listingService.UpsertLocalListing(listingMsg)
        }

        apiPort, _ := cmd.Flags().GetInt("api-port")
        apiServer = api.NewServer(apiPort, agentManager, listingService, orderService, p2pHost)
        if err := apiServer.Start(); err != nil {
            fmt.Printf("warning: failed to start API server: %v\n", err)
        } else {
            fmt.Printf("HTTP API server running on port %d\n", apiPort)
        }

        announceInterval, _ := cmd.Flags().GetDuration("announce-interval")
        if announceInterval < 5*time.Second {
            announceInterval = 5 * time.Second
        }
        go runListingAnnouncer(ctx, announceInterval, func(ts int64) *types.AgentListingMessage {
            msg := *listingMsg
            msg.Type = "update"
            msg.Timestamp = ts
            return &msg
        })

        fmt.Printf("Agent registered: %s (%s)\n", registered.Name, registered.AgentID)
    }

    fmt.Println("Starting Betar TUI...")
    printRuntimeInfo()

    return tui.RunTUI()
}
```

**Step 5: Verify it builds**

```bash
make build
```

Expected: **PASS** — clean build.

**Step 6: Commit**

```bash
git add cmd/betar/main.go
git commit -m "feat: add startup flags to TUI, full agent lifecycle when --name is set"
```

---

### Task 7: Final verification

**Step 1: Clean build**

```bash
make build
```

Expected output: no errors, binary at `bin/betar`.

**Step 2: Verify binary exists and help text includes new flags**

```bash
./bin/betar --help
```

Expected: Output includes `--name`, `--price`, `--api-port`, `--announce-interval` flags.

**Step 3: Run TUI without flags (baseline test)**

```bash
./bin/betar
```

Expected: TUI starts, right panel shows `(not started)` for PeerID initially then updates, wallet shows `(none)` (no `ETHEREUM_PRIVATE_KEY` set), agents list shows `(none)`. Log panel shows "Starting Betar TUI..." and peer info. Typing `/ag` shows autocomplete dropdown with agent commands. Tab fills in first suggestion. Ghost text appears.

**Step 4: Verify Ctrl+C exits cleanly**

Press Ctrl+C in the TUI.
Expected: clean exit, no panic.

**Step 5: Final commit if any cleanup needed**

```bash
git add -p  # review any unstaged changes
git commit -m "chore: verify TUI improvements complete"
```

---

## Summary of All Changed Files

| File | What Changed |
|------|-------------|
| `cmd/betar/tui/model.go` | Added `agentInfo` type; replaced `did` with `agents []agentInfo`; added `rightViewport`, `suggestions`, `suggestionIdx`; init `rightViewport` in `NewModel()` |
| `cmd/betar/tui/styles.go` | Added `SuggestionStyle`, `SuggestionHighlightStyle`, `GhostStyle` |
| `cmd/betar/tui/commands.go` | Added `runtimeWalletAddr`, `SetWallet()`, `getWalletAddr()`, `knownCommands`; removed dead `hasPrefix()` |
| `cmd/betar/tui/update.go` | Expanded `nodeInfoMsg`; updated `fetchNodeInfo()`; Tab/Esc/pgup/pgdown key cases; suggestion-aware Up/Down; `computeSuggestions()`; right viewport update on `nodeInfoMsg`; resize right viewport on `WindowSizeMsg` |
| `cmd/betar/tui/view.go` | `renderRightPanel` uses viewport; `buildRightPanelContent()` helper; `renderInput` with suggestion dropdown + ghost text |
| `cmd/betar/main.go` | Optional flags on root cmd; `deriveWalletAddress()`; full agent lifecycle in `runTUI` when `--name` set |
