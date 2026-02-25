package tui

import (
	"fmt"
	"strings"

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

func (m model) renderLogs(width, height int) string {
	return PanelStyle.Width(width - 2).Height(height - 2).Render(m.logsViewport.View())
}

func (m model) renderInput(width, height int) string {
	return InputStyle.Width(width - 2).Height(height - 2).Render(m.cmdInput.View())
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
