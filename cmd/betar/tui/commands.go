package tui

import (
	"fmt"
	"strings"

	"github.com/asabya/betar/internal/agent"
	"github.com/asabya/betar/internal/marketplace"
)

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
		return executeAgent(cmd)
	case strings.HasPrefix(cmd, "/order create "):
		return createOrder(cmd)
	default:
		return []string{"Unknown command: " + cmd + ". Type /help for available commands."}
	}
}

func hasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

func listAgents() []string {
	// Access global agentManager from main package
	if agentManager := getAgentManager(); agentManager != nil {
		agents := agentManager.ListAgents()
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
	if listingService := getListingService(); listingService != nil {
		listings := listingService.ListListings()
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
	if p2pHost := getP2PHost(); p2pHost != nil {
		peers := p2pHost.GetPeers()
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
	if p2pHost := getP2PHost(); p2pHost != nil {
		result = append(result, "Node Status:")
		result = append(result, fmt.Sprintf("  Peer ID: %s", p2pHost.ID()))
		result = append(result, fmt.Sprintf("  Addresses: %v", p2pHost.AddrStrings()))
		peers := p2pHost.GetPeers()
		result = append(result, fmt.Sprintf("  Connected Peers: %d", len(peers)))
	}
	return result
}

func checkBalance() []string {
	return []string{"Wallet balance: (not implemented)"}
}

func executeAgent(cmd string) []string {
	parts := strings.SplitN(cmd, " ", 3)
	if len(parts) < 3 {
		return []string{"Usage: /agent execute <agent-id> <task>"}
	}
	agentID := parts[1]
	task := parts[2]
	return []string{fmt.Sprintf("Executing task on agent %s: %s (not implemented)", agentID, task)}
}

func createOrder(cmd string) []string {
	parts := strings.SplitN(cmd, " ", 3)
	if len(parts) < 3 {
		return []string{"Usage: /order create <agent-id> <price>"}
	}
	agentID := parts[1]
	price := parts[2]
	return []string{fmt.Sprintf("Creating order for agent %s with price %s (not implemented)", agentID, price)}
}

// Placeholder functions - will be replaced with actual runtime references
// These are called from update.go and need to access the global variables from main.go

func getAgentManager() *agent.Manager {
	return nil
}

func getListingService() *marketplace.AgentListingService {
	return nil
}

func getOrderService() *marketplace.OrderService {
	return nil
}

func getP2PHost() interface{} {
	return nil
}
