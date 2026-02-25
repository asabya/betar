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

	SuggestionStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#555555")).
			Padding(0, 1)

	SuggestionHighlightStyle = lipgloss.NewStyle().
					Foreground(PrimaryColor).
					Bold(true)

	GhostStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555"))
)
