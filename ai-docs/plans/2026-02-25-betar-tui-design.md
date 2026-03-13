# Betar TUI Design

## Overview

Add a Bubble Tea-based TUI to betar so running `betar` (no subcommand) starts both the P2P server AND an interactive terminal UI with command input.

## Layout

```
┌─────────────────────────────────────┬────────────────────┐
│                                     │ Node Status        │
│           Logs                      │ - PeerID           │
│           (2/3 left, top)           │ - Addresses        │
│                                     │ - Connected Peers │
│                                     │ - Wallet Address   │
│                                     │ - DID              │
│─────────────────────────────────────│ ───────────────────│
│                                     │ Incoming Tasks     │
│           Input                     │ - [task list]      │
│           (1/3 left, bottom)        │                    │
│                                     │                    │
└─────────────────────────────────────┴────────────────────┘
```

- **Right Panel (1/3 width):**
  - Node Status: PeerID, non-loopback addresses, connected peers count, wallet address, DID
  - Incoming Tasks: List of pending tasks
- **Left Top (2/3 of left):** Logs (scrollable, auto-updating)
- **Left Bottom (1/3 of left):** Command input

## Colors

- Primary: `#ff8258` (orange)
- Secondary: `#a1a2ff` (lavender)

## TUI Commands

| Command | Description |
|---------|-------------|
| `/agent list` | List local agents |
| `/agent discover` | Discover marketplace agents |
| `/agent execute <id> <task>` | Execute task on agent |
| `/order create <agent-id> <price>` | Create order |
| `/wallet balance` | Check balance |
| `/peers` | Show connected peers |
| `/help` | Show available commands |
| `/exit` | Quit application |

## Behavior

- Real-time P2P events (new peer, incoming task, order) appear automatically in logs
- Enter key executes command
- Ctrl+C also exits
- Up/Down arrows for command history
- Tab for command autocomplete

## Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/charmbracelet/bubbles/viewport` - Scrollable logs
- `github.com/charmbracelet/bubbles/textinput` - Command input

## Architecture

### File Structure

```
cmd/betar/tui/
├── model.go       # State: logs, peers, tasks, input
├── view.go        # 3-panel layout rendering
├── update.go      # Message handling, keybindings
├── commands.go    # Command handlers
├── styles.go      # Color/theme definitions
└── main.go        # Entry point
```

### State Management

- TUI model holds references to runtime services
- Commands execute via existing services, output to viewport
- P2P events sent as tea.Msg → update → render
