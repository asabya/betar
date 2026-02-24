package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	width := 100
	rightPanelWidth := width / 3
	leftPanelWidth := width - rightPanelWidth

	rightPanel := m.renderRightPanel(rightPanelWidth)
	leftTop := m.renderLogs(leftPanelWidth)
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
	nodeStatus += "Peers: " + string(rune('0'+m.connectedPeers)) + "\n"
	nodeStatus += "Wallet: " + m.walletAddr + "\n"
	nodeStatus += "DID: " + m.did + "\n"

	tasks := TitleStyle.Render("Incoming Tasks") + "\n"
	if len(m.pendingTasks) == 0 {
		tasks += "  (none)"
	} else {
		for _, t := range m.pendingTasks {
			tasks += "- " + t + "\n"
		}
	}

	return PanelStyle.Width(width).Render(nodeStatus + "\n" + tasks)
}

func (m model) renderLogs(width int) string {
	logContent := strings.Join(m.logs, "\n")
	if logContent == "" {
		logContent = "(no logs)"
	}
	return PanelStyle.Width(width).Render(logContent)
}

func (m model) renderInput(width int) string {
	return InputStyle.Width(width).Render(m.cmdInput.View())
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
