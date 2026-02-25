package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type logMsg string

type nodeInfoMsg struct {
	peerID         string
	addresses      []string
	connectedPeers int
	walletAddr     string
	agents         []agentInfo
}

type tickMsg time.Time

func waitForLog() tea.Cmd {
	return func() tea.Msg {
		return logMsg(<-logCh)
	}
}

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

func tickEvery5s() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Init() tea.Cmd {
	return tea.Batch(waitForLog(), fetchNodeInfo(), tickEvery5s())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		leftW := msg.Width * 2 / 3
		rightW := msg.Width - leftW
		logH := msg.Height * 2 / 3
		m.logsViewport = viewport.New(leftW-4, logH-2)
		m.rightViewport = viewport.New(rightW-4, msg.Height-4)
	case logMsg:
		line := string(msg)
		m.logs = append(m.logs, line)
		if len(m.logs) > 100 {
			m.logs = m.logs[len(m.logs)-100:]
		}
		m.logsViewport.SetContent(strings.Join(m.logs, "\n"))
		m.logsViewport.GotoBottom()
		return m, waitForLog()
	case nodeInfoMsg:
		m.peerID = msg.peerID
		m.addresses = msg.addresses
		m.connectedPeers = msg.connectedPeers
		m.walletAddr = msg.walletAddr
		m.agents = msg.agents
		return m, nil
	case tickMsg:
		return m, tea.Batch(fetchNodeInfo(), tickEvery5s())
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

	if cmd == "/exit" {
		return m, tea.Quit
	}

	m.cmdHistory = append(m.cmdHistory, cmd)
	m.historyIndex = len(m.cmdHistory)

	m.logs = append(m.logs, "> "+cmd)
	result := processCommand(cmd)
	m.logs = append(m.logs, result...)

	if len(m.logs) > 100 {
		m.logs = m.logs[len(m.logs)-100:]
	}

	m.logsViewport.SetContent(strings.Join(m.logs, "\n"))
	m.logsViewport.GotoBottom()

	return m, nil
}
