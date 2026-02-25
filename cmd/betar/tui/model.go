package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
)

var logCh = make(chan string, 256)

func SendLog(msg string) {
	select {
	case logCh <- msg:
	default:
	}
}

// agentInfo holds display info for a local agent in the right panel.
type agentInfo struct {
	Name string
	DID  string
}

type model struct {
	p2pHost        interface{}
	agentManager   interface{}
	listingService interface{}
	orderService   interface{}

	logsViewport  viewport.Model
	rightViewport viewport.Model
	cmdInput      textinput.Model

	logs         []string
	cmdHistory   []string
	historyIndex int

	width  int
	height int

	peerID         string
	addresses      []string
	connectedPeers int
	walletAddr     string
	agents         []agentInfo

	suggestions   []string
	suggestionIdx int

	pendingTasks []string
}

func NewModel() model {
	vp := viewport.New(80, 20)
	rvp := viewport.New(30, 20)
	ti := textinput.New()
	ti.Placeholder = "/agent list"
	ti.Prompt = "> "
	ti.Focus()

	initLogs := []string{"Betar TUI started. Type /help for commands."}
	vp.SetContent(strings.Join(initLogs, "\n"))

	return model{
		logsViewport:  vp,
		rightViewport: rvp,
		cmdInput:      ti,
		logs:          initLogs,
	}
}
