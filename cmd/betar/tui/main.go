package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func RunTUI() error {
	m := NewModel()
	p := tea.NewProgram(m)
	return p.Run()
}
