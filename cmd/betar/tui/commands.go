package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/asabya/betar/internal/agent"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/internal/workflow"
	"github.com/asabya/betar/pkg/types"
)

// SessionReader is the TUI's view of the session store.
type SessionReader interface {
	ListByAgent(ctx context.Context, agentID string) ([]*types.Session, error)
	Get(ctx context.Context, agentID, callerID string) (*types.Session, error)
}

var (
	runtimeP2pHost        *p2p.Host
	runtimeAgentManager   *agent.Manager
	runtimeListingService *marketplace.AgentListingService
	runtimeOrderService   *marketplace.OrderService
	runtimeWalletAddr     string
	runtimeDataDir        string
	runtimeSessionStore   SessionReader
	runtimeOrchestrator   *workflow.Orchestrator
)

// knownCommands is the list used for autocomplete suggestions.
var knownCommands = []string{
	"/help",
	"/status",
	"/peers",
	"/exit",
	"/wallet balance",
	"/agent list",
	"/agent discover",
	"/agent execute ",
	"/order create ",
	"/session list ",
	"/session show ",
	"/workflow create ",
	"/workflow list",
	"/workflow status ",
}

func processCommand(cmd string) []string {
	switch {
	case cmd == "/help":
		return []string{
			"Available commands:",
			"  /agent list        - List local agents",
			"  /agent discover    - Discover marketplace agents",
			"  /agent execute <id> <task> - Execute task",
			"  /order create <agent-id> <price> - Create order",
			"  /wallet balance   - Check wallet balance",
			"  /peers            - Show connected peers",
			"  /status           - Show node status",
			"  /session list <agentID>            - List sessions for an agent",
			"  /session show <agentID> <callerID> - Show session exchanges",
			"  /workflow create <agents> <input>  - Run a multi-agent workflow",
			"  /workflow list                     - List all workflows",
			"  /workflow status <id>              - Show workflow status",
			"  /exit             - Quit application",
		}
	case cmd == "/agent list":
		return listAgents()
	case cmd == "/agent discover":
		return discoverAgents()
	case cmd == "/peers":
		return listPeers()
	case cmd == "/status":
		return showStatus()
	case cmd == "/wallet balance":
		return checkBalance()
	case strings.HasPrefix(cmd, "/agent execute "):
		return executeAgent(strings.TrimPrefix(cmd, "/agent execute "))
	case strings.HasPrefix(cmd, "/order create "):
		return createOrder(strings.TrimPrefix(cmd, "/order create "))
	case strings.HasPrefix(cmd, "/session list "):
		return listSessions(strings.TrimPrefix(cmd, "/session list "))
	case strings.HasPrefix(cmd, "/session show "):
		return showSession(strings.TrimPrefix(cmd, "/session show "))
	case cmd == "/workflow list":
		return listWorkflowsTUI()
	case strings.HasPrefix(cmd, "/workflow status "):
		return showWorkflowStatus(strings.TrimPrefix(cmd, "/workflow status "))
	default:
		return []string{"Unknown command: " + cmd + ". Type /help for available commands."}
	}
}

func listAgents() []string {
	if mgr := getAgentManager(); mgr != nil {
		agents := mgr.ListAgents()
		if len(agents) == 0 {
			return []string{"No local agents registered"}
		}
		var result []string
		result = append(result, "Local Agents:")
		for _, a := range agents {
			result = append(result, fmt.Sprintf("  - %s (%s) - %f ETH", a.Name, a.ID, a.Price))
		}
		return result
	}
	return []string{"Agent manager not initialized. Start node first."}
}

func discoverAgents() []string {
	if ls := getListingService(); ls != nil {
		listings := ls.ListListings()
		if len(listings) == 0 {
			return []string{"No agents discovered"}
		}
		var result []string
		result = append(result, "Discovered Agents:")
		for _, l := range listings {
			if l != nil {
				result = append(result, fmt.Sprintf("  - %s (%s) - %f ETH", l.Name, l.ID, l.Price))
			}
		}
		return result
	}
	return []string{"Listing service not initialized. Start node first."}
}

func listPeers() []string {
	if h := getP2PHost(); h != nil {
		peers := h.RawHost().Network().Peers()
		if len(peers) == 0 {
			return []string{"No connected peers"}
		}
		var result []string
		result = append(result, "Connected Peers:")
		for _, p := range peers {
			result = append(result, fmt.Sprintf("  - %s", p))
		}
		return result
	}
	return []string{"P2P host not initialized. Start node first."}
}

func showStatus() []string {
	var result []string
	if h := getP2PHost(); h != nil {
		result = append(result, "Node Status:")
		result = append(result, fmt.Sprintf("  Peer ID: %s", h.ID()))
		result = append(result, fmt.Sprintf("  Addresses: %v", h.AddrStrings()))
		peers := h.RawHost().Network().Peers()
		result = append(result, fmt.Sprintf("  Connected Peers: %d", len(peers)))
	}
	return result
}

func checkBalance() []string {
	return []string{"Wallet balance: (not implemented)"}
}

func executeAgent(args string) []string {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 || parts[0] == "" {
		return []string{"Usage: /agent execute <agent-id> <task>"}
	}

	mgr := getAgentManager()
	if mgr == nil {
		return []string{"Agent manager not initialized. Start node first."}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req := types.AgentRequest{Input: parts[1]}
	output, err := mgr.ExecuteTask(ctx, parts[0], req)
	if err != nil {
		return []string{fmt.Sprintf("Execution failed: %v", err)}
	}
	return []string{fmt.Sprintf("Agent %s result:", parts[0]), output}
}

func createOrder(args string) []string {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 || parts[0] == "" {
		return []string{"Usage: /order create <agent-id> <price>"}
	}
	agentID := parts[0]
	price := parts[1]
	return []string{fmt.Sprintf("Creating order for agent %s with price %s (not implemented)", agentID, price)}
}

func getAgentManager() *agent.Manager {
	return runtimeAgentManager
}

func getListingService() *marketplace.AgentListingService {
	return runtimeListingService
}

func getOrderService() *marketplace.OrderService {
	return runtimeOrderService
}

func getP2PHost() *p2p.Host {
	return runtimeP2pHost
}

func SetRuntime(p2pHost *p2p.Host, agentManager *agent.Manager, listingService *marketplace.AgentListingService, orderService *marketplace.OrderService) {
	runtimeP2pHost = p2pHost
	runtimeAgentManager = agentManager
	runtimeListingService = listingService
	runtimeOrderService = orderService
}

// SetWallet sets the Ethereum wallet address for display in the TUI.
func SetWallet(addr string) {
	runtimeWalletAddr = addr
}

func getWalletAddr() string {
	return runtimeWalletAddr
}

// SetDataDir sets the data directory path for display in the TUI.
func SetDataDir(dir string) {
	runtimeDataDir = dir
}

func getDataDir() string {
	return runtimeDataDir
}

// SetSessionStore sets the session store for TUI session commands.
func SetSessionStore(s SessionReader) {
	runtimeSessionStore = s
}

// SetOrchestrator sets the workflow orchestrator for TUI workflow commands.
func SetOrchestrator(o *workflow.Orchestrator) {
	runtimeOrchestrator = o
}

func listSessions(agentID string) []string {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return []string{"Usage: /session list <agentID>"}
	}
	if runtimeSessionStore == nil {
		return []string{"Session store not initialized."}
	}
	sessions, err := runtimeSessionStore.ListByAgent(context.Background(), agentID)
	if err != nil {
		return []string{"Error: " + err.Error()}
	}
	if len(sessions) == 0 {
		return []string{"No sessions found for agent: " + agentID}
	}
	var result []string
	result = append(result, fmt.Sprintf("Sessions for agent %s:", agentID))
	for _, s := range sessions {
		result = append(result, fmt.Sprintf("  Caller: %s | Exchanges: %d | Updated: %s",
			s.CallerID, len(s.Exchanges), s.UpdatedAt.Format("2006-01-02 15:04:05")))
	}
	return result
}

func showSession(args string) []string {
	parts := strings.SplitN(strings.TrimSpace(args), " ", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return []string{"Usage: /session show <agentID> <callerID>"}
	}
	if runtimeSessionStore == nil {
		return []string{"Session store not initialized."}
	}
	sess, err := runtimeSessionStore.Get(context.Background(), parts[0], parts[1])
	if err != nil {
		return []string{"Error: " + err.Error()}
	}
	if sess == nil {
		return []string{"Session not found."}
	}
	var result []string
	result = append(result, fmt.Sprintf("Session %s | Agent: %s | Caller: %s", sess.ID, sess.AgentID, sess.CallerID))
	for i, ex := range sess.Exchanges {
		result = append(result, fmt.Sprintf("  [%d] %s", i+1, ex.Timestamp.Format("15:04:05")))
		result = append(result, fmt.Sprintf("    IN:  %s", ex.Input))
		result = append(result, fmt.Sprintf("    OUT: %s", truncate(ex.Output, 120)))
		if ex.Error != "" {
			result = append(result, fmt.Sprintf("    ERR: %s", ex.Error))
		}
		if ex.Payment != nil {
			result = append(result, fmt.Sprintf("    PAY: %s USDC | tx=%s | payer=%s",
				ex.Payment.Amount, ex.Payment.TxHash[:min(10, len(ex.Payment.TxHash))], ex.Payment.Payer))
		}
	}
	return result
}

func listWorkflowsTUI() []string {
	if runtimeOrchestrator == nil {
		return []string{"Orchestrator not initialized. Start node first."}
	}
	workflows, _ := runtimeOrchestrator.ListWorkflows(context.Background())
	if len(workflows) == 0 {
		return []string{"No workflows found"}
	}
	var result []string
	result = append(result, "Workflows:")
	for _, wf := range workflows {
		input := truncate(wf.Input, 40)
		result = append(result, fmt.Sprintf("  %s  %s  %d steps  %s", wf.ID[:8], wf.Status, len(wf.Steps), input))
	}
	return result
}

func showWorkflowStatus(id string) []string {
	id = strings.TrimSpace(id)
	if id == "" {
		return []string{"Usage: /workflow status <id>"}
	}
	if runtimeOrchestrator == nil {
		return []string{"Orchestrator not initialized. Start node first."}
	}
	wf, err := runtimeOrchestrator.GetWorkflow(context.Background(), id)
	if err != nil {
		return []string{fmt.Sprintf("Error: %v", err)}
	}
	var result []string
	result = append(result, fmt.Sprintf("Workflow %s  Status: %s", wf.ID[:8], wf.Status))
	result = append(result, fmt.Sprintf("  Input: %s", truncate(wf.Input, 120)))
	for _, step := range wf.Steps {
		line := fmt.Sprintf("  Step %d (%s): %s", step.Index+1, step.AgentID, step.Status)
		if step.Error != "" {
			line += " - " + step.Error
		}
		if step.Output != "" {
			line += " -> " + truncate(step.Output, 80)
		}
		result = append(result, line)
	}
	if wf.Output != "" {
		result = append(result, fmt.Sprintf("  Output: %s", truncate(wf.Output, 200)))
	}
	return result
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
