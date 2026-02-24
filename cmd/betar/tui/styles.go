package tui

import "github.com/charmbracelet/lipgloss"

var (
	PrimaryColor   = lipgloss.Color("#ff8258")
	SecondaryColor = lipgloss.Color("#a1a2ff")

	TitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true)

	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(SecondaryColor)

	InputStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor)
)
