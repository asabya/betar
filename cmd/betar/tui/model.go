package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

type model struct {
	p2pHost        interface{}
	agentManager   interface{}
	listingService interface{}
	orderService   interface{}

	logsViewport viewport.Model
	cmdInput     textinput.Model

	logs         []string
	cmdHistory   []string
	historyIndex int

	peerID         string
	addresses      []string
	connectedPeers int
	walletAddr     string
	did            string

	pendingTasks []string
}

func NewModel() model {
	vp := viewport.New(80, 20)
	ti := textinput.New()
	ti.Placeholder = "/agent list"
	ti.Prompt = "> "

	return model{
		logsViewport: vp,
		cmdInput:     ti,
		logs:         []string{"Betar TUI started. Type /help for commands."},
	}
}
