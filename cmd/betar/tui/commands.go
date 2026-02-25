package tui

import (
	"fmt"
	"strings"

	"github.com/asabya/betar/internal/agent"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/internal/p2p"
)

var (
	runtimeP2pHost        *p2p.Host
	runtimeAgentManager   *agent.Manager
	runtimeListingService *marketplace.AgentListingService
	runtimeOrderService   *marketplace.OrderService
	runtimeWalletAddr     string
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
	agentID := parts[0]
	task := parts[1]
	return []string{fmt.Sprintf("Executing task on agent %s: %s (not implemented)", agentID, task)}
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
