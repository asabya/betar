package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
)

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
			if m.historyIndex > 0 {
				m.historyIndex--
				if m.historyIndex < len(m.cmdHistory) {
					m.cmdInput.SetValue(m.cmdHistory[m.historyIndex])
				}
			}
		case "down":
			if m.historyIndex < len(m.cmdHistory) {
				m.historyIndex++
				if m.historyIndex < len(m.cmdHistory) {
					m.cmdInput.SetValue(m.cmdHistory[m.historyIndex])
				} else {
					m.cmdInput.SetValue("")
				}
			}
		}
	case tea.WindowSizeMsg:
		m.logsViewport = viewport.New(msg.Width/2, msg.Height-5)
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

	m.cmdHistory = append(m.cmdHistory, cmd)
	m.historyIndex = len(m.cmdHistory)

	m.logs = append(m.logs, "> "+cmd)
	result := processCommand(cmd)
	m.logs = append(m.logs, result...)

	if len(m.logs) > 100 {
		m.logs = m.logs[len(m.logs)-100:]
	}

	return m, nil
}
