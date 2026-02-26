package tui

import (
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var tuiOutput io.Writer = os.Stdout

func SetOutput(w io.Writer) {
	tuiOutput = w
}

func RunTUI() error {
	m := NewModel()
	p := tea.NewProgram(m, tea.WithOutput(tuiOutput), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
