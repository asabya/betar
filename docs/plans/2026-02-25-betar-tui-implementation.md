# Betar TUI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add Bubble Tea TUI to betar with 3-panel layout (right panel for node status/tasks, left top for logs, left bottom for input)

**Architecture:** Add Bubble Tea as dependency, create new tui package, integrate with existing runtime services, modify main.go to default to TUI mode

**Tech Stack:** Bubble Tea, Lipgloss, Bubbles (viewport, textinput)

---

## Task 1: Add Dependencies

**Files:**
- Modify: `go.mod`

**Step 1: Add Bubble Tea dependencies**

Run: `go get github.com/charmbracelet/bubbletea github.com/charmbracelet/lipgloss github.com/charmbracelet/bubbles/viewport github.com/charmbracelet/bubbles/textinput github.com/charmbracelet/bubbles/list`

**Step 2: Run go mod tidy**

Run: `go mod tidy`

**Step 3: Commit**

Run: `git add go.mod go.sum && git commit -m "deps: add bubble tea and lipgloss dependencies"`

---

## Task 2: Create TUI Package Structure

**Files:**
- Create: `cmd/betar/tui/styles.go`
- Create: `cmd/betar/tui/model.go`
- Create: `cmd/betar/tui/view.go`
- Create: `cmd/betar/tui/update.go`
- Create: `cmd/betar/tui/commands.go`
- Create: `cmd/betar/tui/main.go`

**Step 1: Create cmd/betar/tui directory**

Run: `mkdir -p cmd/betar/tui`

**Step 2: Create styles.go with colors**

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
)
```

**Step 3: Commit**

Run: `git add cmd/betar/tui/styles.go && git commit -m "feat: create tui package with styles"`

---

## Task 3: Implement Model

**Files:**
- Modify: `cmd/betar/tui/model.go`

**Step 1: Write the model struct**

```go
package tui

import (
    "github.com/charmbracelet/bubbles/viewport"
    "github.com/charmbracelet/bubbles/textinput"
)

type model struct {
    // Runtime references (set after initialization)
    p2pHost        interface{} // *p2p.Host
    agentManager   interface{} // *agent.Manager
    listingService interface{} // *marketplace.AgentListingService
    orderService   interface{} // *marketplace.OrderService
    
    // UI components
    logsViewport viewport.Model
    cmdInput    textinput.Model
    
    // State
    logs         []string
    cmdHistory   []string
    historyIndex int
    
    // Status (updated from runtime)
    peerID       string
    addresses    []string
    connectedPeers int
    walletAddr   string
    did          string
    
    // Incoming tasks
    pendingTasks []string
}
```

**Step 2: Create NewModel function**

```go
func NewModel() model {
    vp := viewport.New(80, 20)
    ti := textinput.New()
    ti.Placeholder = "/agent list"
    ti.Prompt = "> "
    
    return model{
        logsViewport: vp,
        cmdInput:    ti,
        logs:       []string{"Betar TUI started. Type /help for commands."},
    }
}
```

**Step 3: Commit**

Run: `git add cmd/betar/tui/model.go && git commit -m "feat: add TUI model struct"`

---

## Task 4: Implement View

**Files:**
- Modify: `cmd/betar/tui/view.go`

**Step 1: Write View method**

```go
package tui

func (m model) View() string {
    // Calculate panel widths based on terminal size
    width := 100 // get from viewport
    rightPanelWidth := width / 3
    leftPanelWidth := width - rightPanelWidth
    
    // Right panel - Node Status + Tasks
    rightPanel := m.renderRightPanel(rightPanelWidth)
    
    // Left top - Logs
    leftTop := m.renderLogs(leftPanelWidth)
    
    // Left bottom - Input
    leftBottom := m.renderInput(leftPanelWidth)
    
    return lipgloss.JoinVertical(
        lipgloss.Left,
        lipgloss.JoinHorizontal(lipgloss.Top, leftTop, rightPanel),
        lipgloss.JoinHorizontal(lipgloss.Bottom, leftBottom, rightPanel),
    )
}

func (m model) renderRightPanel(width int) string {
    nodeStatus := TitleStyle.Render("Node Status") + "\n"
    nodeStatus += "PeerID: " + m.peerID + "\n"
    nodeStatus += "Addresses: " + formatAddrs(m.addresses) + "\n"
    nodeStatus += "Peers: " + string(rune('0' + m.connectedPeers)) + "\n"
    nodeStatus += "Wallet: " + m.walletAddr + "\n"
    nodeStatus += "DID: " + m.did + "\n"
    
    tasks := TitleStyle.Render("Incoming Tasks") + "\n"
    for _, t := range m.pendingTasks {
        tasks += "- " + t + "\n"
    }
    
    return PanelStyle.Width(width).Render(nodeStatus + "\n" + tasks)
}

func (m model) renderLogs(width int) string {
    return PanelStyle.Width(width).Render(m.logsViewport.View())
}

func (m model) renderInput(width int) string {
    return m.cmdInput.View()
}

func formatAddrs(addrs []string) string {
    // Filter non-loopback, format for display
}
```

**Step 2: Commit**

Run: `git add cmd/betar/tui/view.go && git commit -m "feat: add TUI view rendering"`

---

## Task 5: Implement Update

**Files:**
- Modify: `cmd/betar/tui/update.go`

**Step 1: Write Update method**

```go
package tui

import "github.com/charmbracelet/bubbletea"

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "/exit":
            return m, tea.Quit
        case "enter":
            return m.handleCommand()
        case "up":
            // Command history up
        case "down":
            // Command history down
        }
    case tea.WindowSizeMsg:
        // Resize panels
    }
    
    m.cmdInput, cmd = m.cmdInput.Update(msg)
    return m, cmd
}

func (m model) handleCommand() (model, tea.Cmd) {
    cmd := m.cmdInput.Value()
    m.cmdInput.SetValue("")
    
    if cmd == "" {
        return m, nil
    }
    
    // Add to history
    m.cmdHistory = append(m.cmdHistory, cmd)
    m.historyIndex = len(m.cmdHistory)
    
    // Process command
    m.logs = append(m.logs, "> "+cmd)
    result := processCommand(cmd)
    m.logs = append(m.logs, result...)
    
    return m, nil
}
```

**Step 2: Commit**

Run: `git add cmd/betar/tui/update.go && git commit -m "feat: add TUI update handling"`

---

## Task 6: Implement Commands

**Files:**
- Modify: `cmd/betar/tui/commands.go`

**Step 1: Write command handlers**

```go
package tui

func processCommand(cmd string) []string {
    switch {
    case cmd == "/help":
        return []string{
            "Available commands:",
            "  /agent list        - List local agents",
            "  /agent discover    - Discover marketplace agents",
            "  /agent execute <id> <task> - Execute task",
            "  /order create <agent-id> <price> - Create order",
            "  /wallet balance    - Check wallet balance",
            "  /peers             - Show connected peers",
            "  /exit              - Quit application",
        }
    case cmd == "/agent list":
        return listAgents()
    case cmd == "/agent discover":
        return discoverAgents()
    case cmd == "/peers":
        return listPeers()
    case cmd == "/wallet balance":
        return checkBalance()
    case hasPrefix(cmd, "/agent execute"):
        return executeAgent(cmd)
    case hasPrefix(cmd, "/order create"):
        return createOrder(cmd)
    default:
        return []string{"Unknown command: " + cmd + ". Type /help for available commands."}
    }
}
```

**Step 2: Implement each handler to call existing CLI functions**

```go
func listAgents() []string {
    // Call API: GET /agents/local
    // Format output
}

func discoverAgents() []string {
    // Call API: GET /agents
    // Format output
}

func listPeers() []string {
    // Get from p2pHost.GetPeers()
    // Format output
}
```

**Step 3: Commit**

Run: `git add cmd/betar/tui/commands.go && git commit -m "feat: add TUI command handlers"`

---

## Task 7: Create TUI Entry Point

**Files:**
- Modify: `cmd/betar/tui/main.go`

**Step 1: Write TUI main**

```go
package tui

import (
    tea "github.com/charmbracelet/bubbletea"
)

func RunTUI() error {
    m := NewModel()
    p := tea.NewProgram(m)
    return p.Run()
}
```

**Step 2: Commit**

Run: `git add cmd/betar/tui/main.go && git commit -m "feat: add TUI entry point"`

---

## Task 8: Integrate with Main

**Files:**
- Modify: `cmd/betar/main.go`

**Step 1: Modify rootCmd to default to TUI**

```go
var rootCmd = &cobra.Command{
    Use:   "betar",
    Short: "P2P Agent 2 Agent Marketplace",
    Long:  "A decentralized marketplace where AI agents can discover, list, and transact with each other",
    RunE:  runTUI,
}
```

**Step 2: Implement runTUI**

```go
func runTUI(cmd *cobra.Command, args []string) error {
    // Initialize runtime (reuse initRuntime)
    if err := initRuntime(cmd); err != nil {
        return err
    }
    
    // Start TUI with runtime references
    return tui.RunTUI()
}
```

**Step 3: Modify shutdownRuntime to handle TUI exit**

Add graceful shutdown when TUI quits.

**Step 4: Commit**

Run: `git add cmd/betar/main.go && git commit -m "feat: integrate TUI with main CLI"`

---

## Task 9: Build and Test

**Step 1: Build binary**

Run: `go build -o bin/betar ./cmd/betar`

**Step 2: Test TUI starts**

Run: `./bin/betar` (may need timeout)

**Step 3: Verify all commands work**

- `/help`
- `/agent list`
- `/peers`

**Step 4: Commit**

Run: `git add . && git commit -m "feat: complete TUI integration"`

---

## Plan complete and saved to `docs/plans/2026-02-25-betar-tui-implementation.md`. 

Two execution options:

1. **Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

2. **Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
