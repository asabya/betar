package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/asabya/betar/pkg/types"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type logMsg string

type nodeInfoMsg struct {
	peerID           string
	addresses        []string
	connectedPeers   int
	walletAddr       string
	dataDir          string
	agents           []agentInfo
	discoveredAgents []agentInfo
}

type tickMsg time.Time

type executeResultMsg struct {
	agentID string
	output  string
	err     error
}

type workflowResultMsg struct {
	workflow *types.Workflow
	err      error
}

func executeTaskCmd(agentID, task string) tea.Cmd {
	return func() tea.Msg {
		mgr := getAgentManager()
		if mgr == nil {
			return executeResultMsg{agentID: agentID, err: fmt.Errorf("agent manager not initialized")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		req := types.AgentRequest{Input: task}
		output, err := mgr.ExecuteTask(ctx, agentID, []byte(req.Input))
		return executeResultMsg{agentID: agentID, output: string(output), err: err}
	}
}

func tuiWorkflowCreateCmd(agents []string, input string) tea.Cmd {
	return func() tea.Msg {
		orch := runtimeOrchestrator
		if orch == nil {
			return workflowResultMsg{err: fmt.Errorf("orchestrator not initialized")}
		}
		wf, err := orch.CreateWorkflow(context.Background(), types.WorkflowDefinition{AgentIDs: agents, Input: input})
		if err != nil {
			return workflowResultMsg{err: err}
		}
		result, err := orch.RunWorkflow(context.Background(), wf.ID)
		if err != nil {
			return workflowResultMsg{err: err}
		}
		return workflowResultMsg{workflow: result}
	}
}

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

		localPeerID := h.ID().String()

		var agents []agentInfo
		if mgr := getAgentManager(); mgr != nil {
			for _, a := range mgr.ListAgents() {
				agents = append(agents, agentInfo{Name: a.Name, DID: a.AgentID})
			}
		}

		var discoveredAgents []agentInfo
		if ls := getListingService(); ls != nil {
			listings, _ := ls.DiscoverAgents(context.Background())
			for _, l := range listings {
				if l == nil || l.SellerID == localPeerID {
					continue
				}
				discoveredAgents = append(discoveredAgents, agentInfo{Name: l.Name, DID: l.ID})
			}
		}

		return nodeInfoMsg{
			peerID:           localPeerID,
			addresses:        h.AddrStrings(),
			connectedPeers:   len(peers),
			walletAddr:       getWalletAddr(),
			dataDir:          getDataDir(),
			agents:           agents,
			discoveredAgents: discoveredAgents,
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
			// If a suggestion is highlighted, fill it in instead of executing.
			if len(m.suggestions) > 0 {
				m.cmdInput.SetValue(m.suggestions[m.suggestionIdx%len(m.suggestions)])
				m.suggestions = nil
				m.suggestionIdx = 0
				return m, nil
			}
			return m.handleCommand()
		case "tab":
			if len(m.suggestions) > 0 {
				m.cmdInput.SetValue(m.suggestions[m.suggestionIdx%len(m.suggestions)])
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		leftW := msg.Width * 2 / 3
		rightW := msg.Width - leftW
		logH := msg.Height * 2 / 3
		m.logsViewport = viewport.New(leftW-4, logH-2)
		m.logsViewport.SetContent(strings.Join(m.logs, "\n"))
		m.logsViewport.GotoBottom()
		m.rightViewport = viewport.New(rightW-4, msg.Height-4)
		m.rightViewport.SetContent(buildRightPanelContent(m))
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
		m.dataDir = msg.dataDir
		m.agents = msg.agents
		m.discoveredAgents = msg.discoveredAgents
		m.rightViewport.SetContent(buildRightPanelContent(m))
		return m, nil
	case tickMsg:
		return m, tea.Batch(fetchNodeInfo(), tickEvery5s())
	case executeResultMsg:
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("[%s] error: %v", msg.agentID, msg.err))
		} else {
			m.logs = append(m.logs, fmt.Sprintf("[%s] output: %s", msg.agentID, msg.output))
		}
		if len(m.logs) > 100 {
			m.logs = m.logs[len(m.logs)-100:]
		}
		m.logsViewport.SetContent(strings.Join(m.logs, "\n"))
		m.logsViewport.GotoBottom()
		return m, nil
	case workflowResultMsg:
		if msg.err != nil {
			m.logs = append(m.logs, fmt.Sprintf("[workflow] error: %v", msg.err))
		} else {
			wf := msg.workflow
			m.logs = append(m.logs, fmt.Sprintf("[workflow %s] %s", wf.ID[:8], wf.Status))
			for _, step := range wf.Steps {
				status := string(step.Status)
				line := fmt.Sprintf("  Step %d (%s): %s", step.Index+1, step.AgentID, status)
				if step.Error != "" {
					line += " - " + step.Error
				}
				m.logs = append(m.logs, line)
			}
			if wf.Output != "" {
				m.logs = append(m.logs, fmt.Sprintf("[workflow] output: %s", truncate(wf.Output, 200)))
			}
		}
		if len(m.logs) > 100 {
			m.logs = m.logs[len(m.logs)-100:]
		}
		m.logsViewport.SetContent(strings.Join(m.logs, "\n"))
		m.logsViewport.GotoBottom()
		return m, nil
	}

	m.cmdInput, cmd = m.cmdInput.Update(msg)
	// Recompute autocomplete suggestions whenever input changes.
	newSuggestions := computeSuggestions(m.cmdInput.Value())
	if len(newSuggestions) != len(m.suggestions) {
		m.suggestionIdx = 0
	}
	m.suggestions = newSuggestions
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

	// Async: intercept /agent execute before processCommand
	if strings.HasPrefix(cmd, "/agent execute ") {
		args := strings.TrimPrefix(cmd, "/agent execute ")
		parts := strings.SplitN(args, " ", 2)
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			m.logs = append(m.logs, "Usage: /agent execute <agent-id> <task>")
		} else {
			agentID, task := parts[0], parts[1]
			m.logs = append(m.logs, fmt.Sprintf("Executing task on agent %s...", agentID))
			if len(m.logs) > 100 {
				m.logs = m.logs[len(m.logs)-100:]
			}
			m.logsViewport.SetContent(strings.Join(m.logs, "\n"))
			m.logsViewport.GotoBottom()
			return m, executeTaskCmd(agentID, task)
		}
	} else if strings.HasPrefix(cmd, "/workflow create ") {
		args := strings.TrimPrefix(cmd, "/workflow create ")
		parts := strings.SplitN(args, " ", 2)
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			m.logs = append(m.logs, "Usage: /workflow create <agent1,agent2,...> <input>")
		} else {
			agents := strings.Split(parts[0], ",")
			m.logs = append(m.logs, fmt.Sprintf("Starting workflow with %d agents...", len(agents)))
			if len(m.logs) > 100 {
				m.logs = m.logs[len(m.logs)-100:]
			}
			m.logsViewport.SetContent(strings.Join(m.logs, "\n"))
			m.logsViewport.GotoBottom()
			return m, tuiWorkflowCreateCmd(agents, parts[1])
		}
	} else {
		result := processCommand(cmd)
		m.logs = append(m.logs, result...)
	}

	if len(m.logs) > 100 {
		m.logs = m.logs[len(m.logs)-100:]
	}

	m.logsViewport.SetContent(strings.Join(m.logs, "\n"))
	m.logsViewport.GotoBottom()

	return m, nil
}

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
