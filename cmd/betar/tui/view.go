package tui

import (
	"fmt"
	"strings"

	"github.com/asabya/betar/pkg/types"
	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	w := m.width
	if w == 0 {
		w = 100
	}
	h := m.height
	if h == 0 {
		h = 30
	}

	rightW := w / 3
	leftW := w - rightW

	logH := h * 2 / 3
	inputH := h - logH

	rightPanel := m.renderRightPanel(rightW, h)
	leftTop := m.renderLogs(leftW, logH)
	leftBottom := m.renderInput(leftW, inputH)

	leftCol := lipgloss.JoinVertical(lipgloss.Left, leftTop, leftBottom)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightPanel)
}

func (m model) renderRightPanel(width, height int) string {
	return PanelStyle.Width(width - 2).Height(height - 2).Render(m.rightViewport.View())
}

// buildRightPanelContent constructs the text content for the right panel viewport.
// Called from Update() when node info changes so the viewport content is set
// on the real model (not a View() copy).
func buildRightPanelContent(m model) string {
	peerID := m.peerID
	if peerID == "" {
		peerID = "(not started)"
	}
	walletAddr := m.walletAddr
	if walletAddr == "" {
		walletAddr = "(none)"
	}
	dataDir := m.dataDir
	if dataDir == "" {
		dataDir = "(unknown)"
	}

	content := TitleStyle.Render("Node Status") + "\n"
	content += "PeerID: " + peerID + "\n"
	content += "Addrs:  " + formatAddrs(m.addresses) + "\n"
	content += fmt.Sprintf("Peers:  %d\n", m.connectedPeers)
	content += "Wallet: " + walletAddr + "\n"
	content += "Data:   " + dataDir + "\n"

	content += "\n" + TitleStyle.Render("Local Agents") + "\n"
	if len(m.agents) == 0 {
		content += "  (none)\n"
	} else {
		for _, a := range m.agents {
			content += fmt.Sprintf("  %s\n  %s\n\n", a.Name, a.DID)
		}
	}

	content += "\n" + TitleStyle.Render("Network Agents") + "\n"
	if len(m.discoveredAgents) == 0 {
		content += "  (none)\n"
	} else {
		for _, a := range m.discoveredAgents {
			content += fmt.Sprintf("  %s\n  %s\n\n", a.Name, a.DID)
		}
	}

	content += "\n" + TitleStyle.Render("Incoming Tasks") + "\n"
	if len(m.pendingTasks) == 0 {
		content += "  (none)\n"
	} else {
		for _, t := range m.pendingTasks {
			content += "- " + t + "\n"
		}
	}

	content += "\n" + TitleStyle.Render("Active Workflows") + "\n"
	if orch := runtimeOrchestrator; orch != nil {
		workflows := orch.ListWorkflows()
		active := 0
		for _, wf := range workflows {
			if wf.Status == types.WorkflowStatusRunning || wf.Status == types.WorkflowStatusPending {
				completed := 0
				for _, s := range wf.Steps {
					if s.Status == types.StepStatusCompleted {
						completed++
					}
				}
				content += fmt.Sprintf("  %s: %d/%d steps\n", wf.ID[:8], completed, len(wf.Steps))
				active++
			}
		}
		if active == 0 {
			content += "  (none)\n"
		}
	} else {
		content += "  (none)\n"
	}

	return content
}

func (m model) renderLogs(width, height int) string {
	return PanelStyle.Width(width - 2).Height(height - 2).Render(m.logsViewport.View())
}

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
		content += SuggestionStyle.Width(width-6).Render(strings.Join(lines, "\n")) + "\n"
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

func formatAddrs(addrs []string) string {
	if len(addrs) == 0 {
		return "(none)"
	}
	var filtered []string
	for _, addr := range addrs {
		if !strings.HasPrefix(addr, "/127.0.0.1") && !strings.HasPrefix(addr, "/localhost") {
			filtered = append(filtered, addr)
		}
	}
	if len(filtered) == 0 {
		return "(local only)"
	}
	if len(filtered) > 2 {
		return strings.Join(filtered[:2], ", ") + "..."
	}
	return strings.Join(filtered, ", ")
}
