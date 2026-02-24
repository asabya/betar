package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func RunTUI() error {
	m := NewModel()
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}
