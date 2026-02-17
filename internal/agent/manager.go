package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/asabya/betar/internal/ipfs"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/pkg/types"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// Manager manages agent lifecycle
type Manager struct {
	runtime        Runtime
	ipfsClient     *ipfs.Client
	p2pHost        *p2p.Host
	streamHandler  *p2p.StreamHandler
	listingService *marketplace.AgentListingService
	paymentService *marketplace.PaymentService
	walletAddress  string

	mu          sync.RWMutex
	localAgents map[string]*LocalAgent
}

// LocalAgent represents a local agent managed by this node
type LocalAgent struct {
	ID          string
	Name        string
	Description string
	Endpoint    string
	Price       float64
	MetadataCID string
	AgentID     string // ADK runtime agent ID
	Status      string
	CreatedAt   time.Time
}

// NewManager creates a new agent manager
func NewManager(runtimeCfg ADKConfig, ipfsClient *ipfs.Client, p2pHost *p2p.Host, streamHandler *p2p.StreamHandler, listingService *marketplace.AgentListingService, privKey crypto.PrivKey, paymentSvc *marketplace.PaymentService, walletAddr string) (*Manager, error) {
	if ipfsClient == nil {
		return nil, fmt.Errorf("ipfs client is required")
	}
	if p2pHost == nil {
		return nil, fmt.Errorf("p2p host is required")
	}

	// Pass the private key to runtime config for DID generation
	runtimeCfg.PrivKey = privKey
	runtime, err := NewADKRuntime(runtimeCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize runtime: %w", err)
	}

	m := &Manager{
		runtime:        runtime,
		ipfsClient:     ipfsClient,
		p2pHost:        p2pHost,
		streamHandler:  streamHandler,
		listingService: listingService,
		paymentService: paymentSvc,
		walletAddress:  walletAddr,
		localAgents:    make(map[string]*LocalAgent),
	}

	// Register stream handlers
	if streamHandler != nil {
		streamHandler.RegisterHandler("execute", m.handleExecuteRequest)
		streamHandler.RegisterHandler("info", m.handleInfoRequest)
	}

	return m, nil
}

// RegisterAgent registers a new agent locally and publishes to marketplace
func (m *Manager) RegisterAgent(ctx context.Context, spec AgentSpec) (*LocalAgent, error) {
	// Create agent in runtime
	runtimeAgentID, err := m.runtime.CreateAgent(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	// Store metadata on IPFS
	metadata := types.AgentRegistration{
		Type:        "Agent",
		Name:        spec.Name,
		Description: spec.Description,
		Image:       spec.Image,
		Services:    spec.Services,
		X402Support: spec.X402Support,
		Active:      true,
	}

	metadataCID, err := m.ipfsClient.AddJSON(ctx, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to store metadata on IPFS: %w", err)
	}

	// Pin the metadata
	if err := m.ipfsClient.Pin(ctx, metadataCID); err != nil {
		return nil, fmt.Errorf("failed to pin metadata: %w", err)
	}

	// Create local agent
	agent := &LocalAgent{
		ID:          m.p2pHost.ID().String() + "/" + runtimeAgentID,
		Name:        spec.Name,
		Description: spec.Description,
		Endpoint:    spec.Endpoint,
		Price:       spec.Price,
		MetadataCID: metadataCID,
		AgentID:     runtimeAgentID,
		Status:      "active",
		CreatedAt:   time.Now(),
	}

	// Store locally
	m.mu.Lock()
	m.localAgents[agent.AgentID] = agent
	m.mu.Unlock()

	return agent, nil
}

// ExecuteTask executes a task on a local agent or a remote agent from CRDT
func (m *Manager) ExecuteTask(ctx context.Context, agentID, input string, paymentHeader *marketplace.PaymentHeader, transactionHash string) (string, *marketplace.PaymentRequiredResponse, error) {
	fmt.Printf("[ExecuteTask] Starting task execution for agentID: %s\n", agentID)

	// Find local agent first
	m.mu.RLock()
	agent, ok := m.localAgents[agentID]
	m.mu.RUnlock()

	if ok {
		fmt.Printf("[ExecuteTask] Found local agent: %s (name: %s)\n", agent.AgentID, agent.Name)
		// Execute locally
		requestID := fmt.Sprintf("local-%d", time.Now().UnixNano())
		fmt.Printf("[ExecuteTask] Executing locally with requestID: %s\n", requestID)

		result, err := m.runtime.RunTask(ctx, types.TaskRequest{
			AgentID:   agent.AgentID,
			Input:     input,
			RequestID: requestID,
		})
		if err != nil {
			fmt.Printf("[ExecuteTask] Local execution failed: %v\n", err)
			return "", nil, fmt.Errorf("failed to run agent: %w", err)
		}

		if result.Error != "" {
			fmt.Printf("[ExecuteTask] Local execution returned error: %s\n", result.Error)
			return "", nil, fmt.Errorf("runtime error: %s", result.Error)
		}

		fmt.Printf("[ExecuteTask] Local execution successful, output length: %d\n", len(result.Output))
		return result.Output, nil, nil
	}

	fmt.Printf("[ExecuteTask] Agent not found locally, searching CRDT for remote agent\n")

	// Not found locally, try to find in CRDT and execute remotely
	if m.listingService != nil {
		listing, runtimeAgentID := m.findListingByAgentID(agentID)
		if listing != nil {
			fmt.Printf("[ExecuteTask] Found remote listing - PeerID: %s, RuntimeAgentID: %s\n", listing.SellerID, runtimeAgentID)
			fmt.Printf("[ExecuteTask] Listing has %d addresses\n", len(listing.Addrs))

			// Connect to remote peer and execute
			peerID, err := m.connectToPeer(ctx, listing.SellerID, listing.Addrs)
			if err != nil {
				fmt.Printf("[ExecuteTask] Failed to connect to remote peer %s: %v\n", listing.SellerID, err)
				return "", nil, fmt.Errorf("failed to connect to remote agent: %w", err)
			}

			fmt.Printf("[ExecuteTask] Connected to peer %s, executing remotely\n", peerID)
			return m.RemoteExecute(ctx, peerID, runtimeAgentID, input, paymentHeader, transactionHash)
		}
		fmt.Printf("[ExecuteTask] Agent not found in CRDT listings\n")
	} else {
		fmt.Printf("[ExecuteTask] Listing service not available, cannot search CRDT\n")
	}

	fmt.Printf("[ExecuteTask] Agent not found: %s\n", agentID)
	return "", nil, fmt.Errorf("agent not found: %s", agentID)
}

// findListingByAgentID looks up a listing by agent ID in the CRDT.
// It tries direct lookup first, then searches all listings.
// Returns the listing and the runtime agent ID extracted from the listing.
func (m *Manager) findListingByAgentID(agentID string) (*types.AgentListing, string) {
	fmt.Printf("[findListingByAgentID] Searching CRDT for agentID: %s\n", agentID)

	// Try direct lookup by full agent ID
	if listing, ok := m.listingService.GetListing(agentID); ok {
		runtimeID := m.extractRuntimeAgentID(listing)
		fmt.Printf("[findListingByAgentID] Direct CRDT lookup successful - ListingID: %s, RuntimeID: %s\n", listing.ID, runtimeID)
		return listing, runtimeID
	}
	fmt.Printf("[findListingByAgentID] Direct lookup failed, searching all listings\n")

	// Search all listings
	listings := m.listingService.ListListings()
	fmt.Printf("[findListingByAgentID] Total listings in CRDT: %d\n", len(listings))

	for i, listing := range listings {
		if listing == nil {
			continue
		}

		// Check if listing.ID matches
		if listing.ID == agentID {
			runtimeID := m.extractRuntimeAgentID(listing)
			fmt.Printf("[findListingByAgentID] Found match at index %d by listing.ID\n", i)
			return listing, runtimeID
		}

		// Check if the runtime agent ID matches
		runtimeID := m.extractRuntimeAgentID(listing)
		if runtimeID == agentID {
			fmt.Printf("[findListingByAgentID] Found match at index %d by runtime agent ID\n", i)
			return listing, runtimeID
		}
	}

	fmt.Printf("[findListingByAgentID] Agent not found in any CRDT listings\n")
	return nil, ""
}

// extractRuntimeAgentID extracts the runtime agent ID from a listing.
// The listing.ID format is "peerID/runtimeAgentID" or just the runtime agent ID.
func (m *Manager) extractRuntimeAgentID(listing *types.AgentListing) string {
	if listing == nil {
		return ""
	}
	if listing.SellerID == "" {
		return listing.ID
	}
	prefix := listing.SellerID + "/"
	if strings.HasPrefix(listing.ID, prefix) {
		return strings.TrimPrefix(listing.ID, prefix)
	}
	return listing.ID
}

// connectToPeer connects to a remote peer by seller ID and addresses.
// Falls back to peer-only connection if no addresses are provided.
func (m *Manager) connectToPeer(ctx context.Context, sellerID string, addrs []string) (peer.ID, error) {
	fmt.Printf("[connectToPeer] Attempting to connect to peer: %s\n", sellerID)

	peerID, err := peer.Decode(sellerID)
	if err != nil {
		fmt.Printf("[connectToPeer] Invalid peer ID %s: %v\n", sellerID, err)
		return "", fmt.Errorf("invalid peer ID: %w", err)
	}
	fmt.Printf("[connectToPeer] Decoded peer ID: %s\n", peerID)

	// If no addresses provided, try peer-only connection
	if len(addrs) == 0 {
		fmt.Printf("[connectToPeer] No addresses provided, attempting peer-only connection\n")
		if err := m.p2pHost.Connect(ctx, peer.AddrInfo{ID: peerID}); err != nil {
			fmt.Printf("[connectToPeer] Peer-only connection failed: %v\n", err)
			return "", err
		}
		fmt.Printf("[connectToPeer] Peer-only connection successful\n")
		return peerID, nil
	}

	fmt.Printf("[connectToPeer] Attempting to connect using %d addresses\n", len(addrs))

	// Parse addresses and try to connect
	var connectErr error
	for i, rawAddr := range addrs {
		addr, err := multiaddr.NewMultiaddr(rawAddr)
		if err != nil {
			fmt.Printf("[connectToPeer] Failed to parse address %d (%s): %v\n", i, rawAddr, err)
			continue
		}
		fmt.Printf("[connectToPeer] Trying address %d: %s\n", i, rawAddr)
		info := peer.AddrInfo{ID: peerID, Addrs: []multiaddr.Multiaddr{addr}}
		if err := m.p2pHost.Connect(ctx, info); err == nil {
			fmt.Printf("[connectToPeer] Successfully connected via address %d\n", i)
			return peerID, nil
		} else {
			fmt.Printf("[connectToPeer] Connection failed for address %d: %v\n", i, err)
			connectErr = err
		}
	}

	fmt.Printf("[connectToPeer] All address connections failed, falling back to peer-only\n")

	// Fallback to peer-only connection if address connections fail
	if err := m.p2pHost.Connect(ctx, peer.AddrInfo{ID: peerID}); err != nil {
		fmt.Printf("[connectToPeer] Fallback peer-only connection failed: %v\n", err)
		if connectErr != nil {
			return "", connectErr
		}
		return "", err
	}

	fmt.Printf("[connectToPeer] Fallback peer-only connection successful\n")
	return peerID, nil
}

// DiscoverAgents discovers agents from the marketplace
func (m *Manager) DiscoverAgents(ctx context.Context) ([]types.AgentListing, error) {
	// In a real implementation, this would subscribe to pubsub and collect listings
	// For now, return empty list
	return []types.AgentListing{}, nil
}

// GetAgent gets a local agent by ID
func (m *Manager) GetAgent(agentID string) (*LocalAgent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, ok := m.localAgents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	return agent, nil
}

// ListAgents lists all local agents
func (m *Manager) ListAgents() []*LocalAgent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]*LocalAgent, 0, len(m.localAgents))
	for _, a := range m.localAgents {
		agents = append(agents, a)
	}

	return agents
}

// DeregisterAgent removes an agent from local registry and runtime.
func (m *Manager) DeregisterAgent(ctx context.Context, agentID string) error {
	m.mu.Lock()
	agent, ok := m.localAgents[agentID]
	if ok {
		delete(m.localAgents, agentID)
	}
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	if err := m.runtime.DeleteAgent(ctx, agent.AgentID); err != nil {
		return fmt.Errorf("failed to delete runtime agent: %w", err)
	}

	return nil
}

// handleExecuteRequest handles incoming execution requests via P2P stream
func (m *Manager) handleExecuteRequest(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
	fmt.Printf("[handleExecuteRequest] Received execution request from peer %s, data length: %d\n", from, len(data))

	var req struct {
		AgentID         string                     `json:"agentId"`
		Input           string                     `json:"input"`
		PaymentHeader   *marketplace.PaymentHeader `json:"paymentHeader,omitempty"`
		TransactionHash string                     `json:"transactionHash,omitempty"`
	}

	if err := json.Unmarshal(data, &req); err != nil {
		fmt.Printf("[handleExecuteRequest] Failed to unmarshal request from %s: %v\n", from, err)
		return json.Marshal(map[string]string{"error": fmt.Sprintf("failed to unmarshal request: %v", err)})
	}

	fmt.Printf("[handleExecuteRequest] Received request - AgentID: %s, HasPaymentHeader: %v, TransactionHash: %s\n",
		req.AgentID, req.PaymentHeader != nil, req.TransactionHash)

	// Get listing to check if payment required
	// First check local agents, then CRDT listing
	fmt.Printf("[handleExecuteRequest] Looking up listing for agentID: %s\n", req.AgentID)

	// Try to find local agent first
	var localAgent *LocalAgent
	m.mu.RLock()
	localAgent, localOk := m.localAgents[req.AgentID]
	m.mu.RUnlock()

	var price float64

	if localOk && localAgent != nil {
		fmt.Printf("[handleExecuteRequest] Found local agent - Name: %s, Price: %.6f\n", localAgent.Name, localAgent.Price)
		price = localAgent.Price
	} else {
		// Try CRDT listing
		listing, crdtOk := m.listingService.GetListing(req.AgentID)
		if !crdtOk {
			fullID := from.String() + "/" + req.AgentID
			fmt.Printf("[handleExecuteRequest] Listing not found for %s, trying fullID: %s\n", req.AgentID, fullID)
			listing, crdtOk = m.listingService.GetListing(fullID)
		}

		if !crdtOk {
			fmt.Printf("[handleExecuteRequest] WARNING: Agent %s not found locally or in CRDT - payment will NOT be required\n", req.AgentID)
		} else {
			fmt.Printf("[handleExecuteRequest] Listing found - Name: %s, Price: %.6f, SellerID: %s\n", listing.Name, listing.Price, listing.SellerID)
			price = listing.Price
		}
	}

	requiresPayment := price > 0
	fmt.Printf("[handleExecuteRequest] Payment check: requiresPayment=%v (price=%.6f)\n", requiresPayment, price)

	// If payment required but not provided, return 402
	if requiresPayment && req.PaymentHeader == nil {
		// Warn if wallet address is not configured
		if m.walletAddress == "" {
			fmt.Printf("[handleExecuteRequest] WARNING: Agent has price=%.6f but seller wallet is not configured!\n", price)
			fmt.Printf("[handleExecuteRequest] Please configure Ethereum in your config file to receive payments\n")
		}
		fmt.Printf("[handleExecuteRequest] >>> RETURNING 402 Payment Required <<<\n")
		fmt.Printf("[handleExecuteRequest] Agent price: %.6f USDC, Seller wallet: %s\n", price, m.walletAddress)

		// Create payment requirement
		amount := fmt.Sprintf("%d", int(price*1e6)) // USDC 6 decimals
		fmt.Printf("[handleExecuteRequest] Creating payment requirement - Amount: %s USDC, PayTo: %s\n", amount, m.walletAddress)
		paymentReq := marketplace.CreatePaymentRequirement(
			marketplace.NetworkBaseSepolia,
			amount,
			marketplace.USDCBaseSepolia,
			m.walletAddress,
			300,
		)

		resp := marketplace.PaymentRequiredResponse{
			AgentID:            req.AgentID,
			RequestID:          fmt.Sprintf("%d", time.Now().UnixNano()),
			Message:            "Payment required for this agent",
			PaymentRequirement: &paymentReq,
			RequiresPayment:    true,
		}

		respData, _ := json.Marshal(resp)
		fmt.Printf("[handleExecuteRequest] Returning PaymentRequiredResponse to buyer, size: %d bytes\n", len(respData))
		return respData, nil
	}

	// If payment provided, verify it
	if req.PaymentHeader != nil {
		fmt.Printf("[handleExecuteRequest] Payment header provided, verifying...\n")
		fmt.Printf("[handleExecuteRequest] PaymentHeader - Payer: %s, PayTo: %s, Amount: %s USDC\n",
			req.PaymentHeader.Payer, req.PaymentHeader.Requirement.PayTo, req.PaymentHeader.Requirement.MaxAmountRequired)

		if m.paymentService == nil {
			return json.Marshal(map[string]string{"error": "payment service not configured"})
		}

		// Verify and settle payment (buyer submitted tx, seller verifies on-chain)
		txHash, err := m.paymentService.VerifyAndSettle(ctx, req.PaymentHeader, req.TransactionHash)
		if err != nil {
			fmt.Printf("[handleExecuteRequest] Payment verification failed: %v\n", err)
			return json.Marshal(map[string]string{"error": fmt.Sprintf("payment verification failed: %v", err)})
		}
		fmt.Printf("[handleExecuteRequest] Payment verified, tx: %s\n", txHash)
	}

	// Execute task
	output, _, err := m.ExecuteTask(ctx, req.AgentID, req.Input, nil, "")
	if err != nil {
		fmt.Printf("[handleExecuteRequest] Task execution failed for agent %s: %v\n", req.AgentID, err)
		return json.Marshal(map[string]string{"error": err.Error()})
	}

	fmt.Printf("[handleExecuteRequest] Task execution successful for agent %s, output length: %d\n", req.AgentID, len(output))

	resp := map[string]interface{}{"output": output}
	if req.PaymentHeader != nil {
		resp["paymentId"] = req.PaymentHeader.PaymentID
	}

	return json.Marshal(resp)
}

// handleInfoRequest handles info requests via P2P stream
func (m *Manager) handleInfoRequest(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
	var req struct {
		AgentID string `json:"agentId"`
	}

	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	agent, err := m.GetAgent(req.AgentID)
	if err != nil {
		return nil, err
	}

	return json.Marshal(agent)
}

// AgentSpec represents agent specification for registration
type AgentSpec struct {
	Name        string
	Description string
	Image       string
	Endpoint    string
	Price       float64
	Framework   string
	Model       string
	Services    []types.Service
	X402Support bool
}

// ConnectToAgent connects to a remote agent via P2P
func (m *Manager) ConnectToAgent(ctx context.Context, peerID peer.ID) error {
	return m.p2pHost.Connect(ctx, peer.AddrInfo{ID: peerID})
}

// RemoteExecute executes a task on a remote agent
func (m *Manager) RemoteExecute(ctx context.Context, peerID peer.ID, agentID, input string, paymentHeader *marketplace.PaymentHeader, transactionHash string) (string, *marketplace.PaymentRequiredResponse, error) {
	fmt.Printf("[RemoteExecute] Starting remote execution to peer %s for agent %s\n", peerID, agentID)

	if m.streamHandler == nil {
		fmt.Printf("[RemoteExecute] Stream handler not configured\n")
		return "", nil, fmt.Errorf("stream handler not configured")
	}

	fmt.Printf("[RemoteExecute] Preparing request with input length: %d\n", len(input))
	req := map[string]interface{}{
		"agentId":         agentID,
		"input":           input,
		"transactionHash": transactionHash,
	}
	if paymentHeader != nil {
		req["paymentHeader"] = paymentHeader
		fmt.Printf("[RemoteExecute] Payment header attached - Payer: %s, PayTo: %s, Amount: %s USDC\n",
			paymentHeader.Payer, paymentHeader.Requirement.PayTo, paymentHeader.Requirement.MaxAmountRequired)
	} else {
		fmt.Printf("[RemoteExecute] NO payment header attached - buyer is requesting without payment\n")
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		fmt.Printf("[RemoteExecute] Failed to marshal request: %v\n", err)
		return "", nil, err
	}
	fmt.Printf("[RemoteExecute] Sending message to peer %s via stream handler\n", peerID)

	resp, err := m.streamHandler.SendMessage(ctx, peerID, "execute", reqData)
	if err != nil {
		fmt.Printf("[RemoteExecute] Failed to send message to peer %s: %v\n", peerID, err)
		return "", nil, err
	}
	fmt.Printf("[RemoteExecute] Received response, length: %d\n", len(resp))

	// Check if payment required (402 response)
	var payResp marketplace.PaymentRequiredResponse
	if json.Unmarshal(resp, &payResp); payResp.RequiresPayment {
		fmt.Printf("[RemoteExecute] >>> RECEIVED 402 Payment Required from seller <<<\n")
		if payResp.PaymentRequirement != nil {
			fmt.Printf("[RemoteExecute] Payment requirement - Amount: %s %s, PayTo: %s\n",
				payResp.PaymentRequirement.MaxAmountRequired,
				payResp.PaymentRequirement.Asset,
				payResp.PaymentRequirement.PayTo)
		}
		return "", &payResp, nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		fmt.Printf("[RemoteExecute] Failed to unmarshal response: %v\n", err)
		return "", nil, err
	}

	if errMsg, ok := result["error"]; ok {
		fmt.Printf("[RemoteExecute] Remote agent returned error: %s\n", errMsg)
		return "", nil, fmt.Errorf("remote error: %s", errMsg)
	}

	output, _ := result["output"].(string)
	fmt.Printf("[RemoteExecute] Remote execution successful, output length: %d\n", len(output))
	return output, nil, nil
}
