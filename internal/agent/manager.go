package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/asabya/betar/internal/ipfs"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/pkg/types"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// Manager manages agent lifecycle
type Manager struct {
	runtime            Runtime
	ipfsClient         *ipfs.Client
	p2pHost            *p2p.Host
	streamHandler      *p2p.StreamHandler
	x402StreamHandler  *p2p.X402StreamHandler
	listingService     *marketplace.AgentListingService
	paymentService     *marketplace.PaymentService
	walletAddress      string

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

			// Auto-negotiate: prefer x402 if the listing advertises it.
			for _, proto := range listing.Protocols {
				if proto == p2p.X402ProtocolID {
					fmt.Printf("[ExecuteTask] Listing supports x402, using /x402/libp2p/1.0.0\n")
					output, err := m.RemoteExecuteX402(ctx, peerID, runtimeAgentID, input)
					return output, nil, err
				}
			}
			return m.RemoteExecute(ctx, peerID, runtimeAgentID, input, paymentHeader, transactionHash)
		}
		fmt.Printf("[ExecuteTask] Agent not found in CRDT listings\n")
	} else {
		fmt.Printf("[ExecuteTask] Listing service not available, cannot search CRDT\n")
	}

	fmt.Printf("[ExecuteTask] Agent not found: %s\n", agentID)
	return "", nil, fmt.Errorf("agent not found: %s", agentID)
}

// FindListingByAgentID is the exported version of findListingByAgentID.
func (m *Manager) FindListingByAgentID(agentID string) (*types.AgentListing, string) {
	return m.findListingByAgentID(agentID)
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
		fmt.Printf("[handleExecuteRequest] PaymentHeader - Payer: %s, PayTo: %s, Amount: %s\n",
			req.PaymentHeader.Payer, req.PaymentHeader.Requirement.PayTo, marketplace.FormatUSDC(req.PaymentHeader.Requirement.Amount))

		if m.paymentService == nil {
			return json.Marshal(map[string]string{"error": "payment service not configured"})
		}

		// Calculate expected amount from agent price
		expectedAmount := big.NewInt(int64(price * 1e6))

		// Verify and settle payment (facilitator settles, seller waits for confirmation)
		txHash, err := m.paymentService.VerifyAndSettle(ctx, req.PaymentHeader, expectedAmount)
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

// RegisterX402Handlers wires up the x402 stream handler and stores a reference for client use.
func (m *Manager) RegisterX402Handlers(sh *p2p.X402StreamHandler) {
	m.x402StreamHandler = sh
	sh.RegisterHandler(marketplace.MsgTypeX402Request, m.handleX402Request)
	sh.RegisterHandler(marketplace.MsgTypeX402PaidRequest, m.handleX402PaidRequest)
}

// handleX402Request is the server-side handler for x402.request messages.
// If the agent requires payment and the request carries no payment, it issues a challenge nonce
// and returns x402.payment_required. If payment is already attached (preemptive), it is
// forwarded to handleX402WithPayment. Free agents are executed directly.
func (m *Manager) handleX402Request(ctx context.Context, from peer.ID, _ string, data []byte) (string, []byte, error) {
	var req marketplace.X402Request
	if err := json.Unmarshal(data, &req); err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrInvalidMessage,
			fmt.Sprintf("failed to unmarshal x402.request: %v", err))
	}

	fmt.Printf("[handleX402Request] peer=%s corr=%s resource=%s\n", from, req.CorrelationID, req.Resource)

	price := m.agentPrice(req.Resource)

	// Preemptive payment provided by the client.
	if req.Payment != nil {
		return m.handleX402WithPayment(ctx, from, &req, price, req.Payment)
	}

	// No payment — free agent, execute directly.
	if price == 0 {
		return m.executeAndRespond(ctx, req.CorrelationID, req.Resource, req.Body)
	}

	// Payment required: generate challenge nonce.
	if m.paymentService == nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrPaymentRequired,
			"payment service not configured on seller")
	}

	challenge, err := m.paymentService.GenerateChallenge(req.CorrelationID, 5*time.Minute)
	if err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrPaymentRequired,
			fmt.Sprintf("failed to generate challenge: %v", err))
	}

	payReq, err := m.paymentService.CreateRequirement(m.walletAddress,
		fmt.Sprintf("%d", int(price*1e6)))
	if err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrPaymentRequired,
			fmt.Sprintf("failed to create payment requirement: %v", err))
	}

	pr := marketplace.X402PaymentRequired{
		Version:             marketplace.X402LibP2PVersion,
		CorrelationID:       req.CorrelationID,
		ChallengeNonce:      challenge.Nonce,
		ChallengeExpiresAt:  challenge.ExpiresAt.Unix(),
		PaymentRequirements: payReq,
		Message:             "Payment required",
	}
	respData, err := json.Marshal(pr)
	if err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrPaymentRequired, err.Error())
	}

	fmt.Printf("[handleX402Request] issued challenge nonce=%s corr=%s\n", challenge.Nonce, req.CorrelationID)
	return marketplace.MsgTypeX402PaymentRequired, respData, nil
}

// handleX402PaidRequest is the server-side handler for x402.paid_request messages.
func (m *Manager) handleX402PaidRequest(ctx context.Context, from peer.ID, _ string, data []byte) (string, []byte, error) {
	var req marketplace.X402PaidRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrInvalidMessage,
			fmt.Sprintf("failed to unmarshal x402.paid_request: %v", err))
	}

	fmt.Printf("[handleX402PaidRequest] peer=%s corr=%s server_nonce=%s\n", from, req.CorrelationID, req.Payment.ServerNonce)

	price := m.agentPrice(req.Payment.Payer) // fallback; actual agent is embedded in body
	// The resource is not in paid_request directly; determine from the original x402.request
	// which stored it before the challenge. We'll use the payer/resource from the payment envelope.
	// NOTE: The agent resource is encoded in req.Body (decoded below).

	// Standard flow: validate challenge nonce matches what was issued.
	if req.Payment.ServerNonce != marketplace.PreemptiveNonce {
		challenge, err := m.paymentService.ConsumeChallenge(req.CorrelationID)
		if err != nil {
			return sendX402Error(req.CorrelationID, marketplace.ErrNonceExpired,
				fmt.Sprintf("challenge expired or unknown: %v", err))
		}
		if challenge.Nonce != req.Payment.ServerNonce {
			return sendX402Error(req.CorrelationID, marketplace.ErrNonceMismatch,
				fmt.Sprintf("nonce mismatch: expected %s, got %s", challenge.Nonce, req.Payment.ServerNonce))
		}
		// Also verify the EIP-712 auth nonce matches.
		if req.Payment.Payload != nil && req.Payment.Payload.Authorization.Nonce != "" {
			authNonce := req.Payment.Payload.Authorization.Nonce
			if strings.HasPrefix(authNonce, "0x") || strings.HasPrefix(authNonce, "0X") {
				authNonce = authNonce[2:]
			}
			if authNonce != challenge.Nonce {
				return sendX402Error(req.CorrelationID, marketplace.ErrNonceMismatch,
					"EIP-712 auth nonce does not match challenge nonce")
			}
		}
	}

	header := envelopeToPaymentHeader(&req.Payment)

	// Decode the body to find the resource (agent ID) and input.
	var bodyPayload struct {
		Resource string `json:"resource"`
		Input    string `json:"input"`
	}
	if len(req.Body) > 0 {
		_ = json.Unmarshal(req.Body, &bodyPayload)
	}
	resource := bodyPayload.Resource
	if resource == "" {
		return sendX402Error(req.CorrelationID, marketplace.ErrInvalidMessage, "missing resource in body")
	}

	price = m.agentPrice(resource)

	if m.paymentService == nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrPaymentInvalid, "payment service not configured")
	}

	expectedAmount := big.NewInt(int64(price * 1e6))
	txHash, err := m.paymentService.VerifyAndSettle(ctx, header, expectedAmount)
	if err != nil {
		fmt.Printf("[handleX402PaidRequest] VerifyAndSettle failed: %v\n", err)
		return sendX402Error(req.CorrelationID, marketplace.ErrSettlementFailed,
			fmt.Sprintf("payment verification/settlement failed: %v", err))
	}

	fmt.Printf("[handleX402PaidRequest] payment settled txHash=%s\n", txHash)

	output, _, err := m.ExecuteTask(ctx, resource, bodyPayload.Input, nil, txHash)
	if err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrExecutionFailed, err.Error())
	}

	respBody, _ := json.Marshal(map[string]string{"output": output})
	resp := marketplace.X402Response{
		Version:       marketplace.X402LibP2PVersion,
		CorrelationID: req.CorrelationID,
		PaymentID:     header.PaymentID,
		TxHash:        txHash,
		Body:          respBody,
	}
	respData, err := json.Marshal(resp)
	if err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrExecutionFailed, err.Error())
	}
	return marketplace.MsgTypeX402Response, respData, nil
}

// handleX402WithPayment handles a preemptive-payment path from handleX402Request.
func (m *Manager) handleX402WithPayment(ctx context.Context, from peer.ID, req *marketplace.X402Request, price float64, env *marketplace.X402PaymentEnvelope) (string, []byte, error) {
	fmt.Printf("[handleX402WithPayment] preemptive payment corr=%s\n", req.CorrelationID)

	if m.paymentService == nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrPaymentInvalid, "payment service not configured")
	}

	header := envelopeToPaymentHeader(env)
	expectedAmount := big.NewInt(int64(price * 1e6))

	txHash, err := m.paymentService.VerifyAndSettle(ctx, header, expectedAmount)
	if err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrSettlementFailed,
			fmt.Sprintf("settlement failed: %v", err))
	}

	var bodyPayload struct {
		Input string `json:"input"`
	}
	if len(req.Body) > 0 {
		_ = json.Unmarshal(req.Body, &bodyPayload)
	}

	output, _, err := m.ExecuteTask(ctx, req.Resource, bodyPayload.Input, nil, txHash)
	if err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrExecutionFailed, err.Error())
	}

	respBody, _ := json.Marshal(map[string]string{"output": output})
	resp := marketplace.X402Response{
		Version:       marketplace.X402LibP2PVersion,
		CorrelationID: req.CorrelationID,
		PaymentID:     header.PaymentID,
		TxHash:        txHash,
		Body:          respBody,
	}
	respData, _ := json.Marshal(resp)
	return marketplace.MsgTypeX402Response, respData, nil
}

// executeAndRespond executes a free agent and returns an x402.response.
func (m *Manager) executeAndRespond(ctx context.Context, correlationID, resource string, rawBody []byte) (string, []byte, error) {
	var bodyPayload struct {
		Input string `json:"input"`
	}
	if len(rawBody) > 0 {
		_ = json.Unmarshal(rawBody, &bodyPayload)
	}

	output, _, err := m.ExecuteTask(ctx, resource, bodyPayload.Input, nil, "")
	if err != nil {
		return sendX402Error(correlationID, marketplace.ErrExecutionFailed, err.Error())
	}

	respBody, _ := json.Marshal(map[string]string{"output": output})
	resp := marketplace.X402Response{
		Version:       marketplace.X402LibP2PVersion,
		CorrelationID: correlationID,
		Body:          respBody,
	}
	respData, _ := json.Marshal(resp)
	return marketplace.MsgTypeX402Response, respData, nil
}

// RemoteExecuteX402 executes a task on a remote agent using the /x402/libp2p/1.0.0 protocol.
// It performs the standard 2-trip flow: send x402.request → receive x402.payment_required →
// sign with challenge nonce → send x402.paid_request → receive x402.response.
func (m *Manager) RemoteExecuteX402(ctx context.Context, peerID peer.ID, agentID, input string) (string, error) {
	if m.x402StreamHandler == nil {
		return "", fmt.Errorf("x402 stream handler not configured")
	}

	correlationID := uuid.New().String()
	bodyPayload := map[string]string{"resource": agentID, "input": input}
	bodyBytes, _ := json.Marshal(bodyPayload)

	req := marketplace.X402Request{
		Version:       marketplace.X402LibP2PVersion,
		CorrelationID: correlationID,
		Resource:      agentID,
		Method:        "execute",
		Payment:       nil,
		Body:          bodyBytes,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal x402.request: %w", err)
	}

	fmt.Printf("[RemoteExecuteX402] sending x402.request to %s corr=%s\n", peerID, correlationID)
	respType, respData, err := m.x402StreamHandler.SendX402Message(ctx, peerID, marketplace.MsgTypeX402Request, reqData)
	if err != nil {
		return "", fmt.Errorf("x402.request failed: %w", err)
	}

	switch respType {
	case marketplace.MsgTypeX402Response:
		return extractX402Output(respData)

	case marketplace.MsgTypeX402PaymentRequired:
		var pr marketplace.X402PaymentRequired
		if err := json.Unmarshal(respData, &pr); err != nil {
			return "", fmt.Errorf("failed to unmarshal x402.payment_required: %w", err)
		}
		fmt.Printf("[RemoteExecuteX402] received payment_required challenge_nonce=%s\n", pr.ChallengeNonce)

		if m.paymentService == nil {
			return "", fmt.Errorf("payment service not configured; cannot pay for x402 agent")
		}

		header, err := m.paymentService.SignRequirementWithNonce(pr.PaymentRequirements, pr.ChallengeNonce)
		if err != nil {
			return "", fmt.Errorf("failed to sign payment with nonce: %w", err)
		}

		env := paymentHeaderToEnvelope(header, pr.ChallengeNonce)
		paidReq := marketplace.X402PaidRequest{
			Version:       marketplace.X402LibP2PVersion,
			CorrelationID: correlationID,
			Payment:       env,
			Body:          bodyBytes,
		}
		paidData, err := json.Marshal(paidReq)
		if err != nil {
			return "", fmt.Errorf("failed to marshal x402.paid_request: %w", err)
		}

		fmt.Printf("[RemoteExecuteX402] sending x402.paid_request to %s corr=%s\n", peerID, correlationID)
		respType2, respData2, err := m.x402StreamHandler.SendX402Message(ctx, peerID, marketplace.MsgTypeX402PaidRequest, paidData)
		if err != nil {
			return "", fmt.Errorf("x402.paid_request failed: %w", err)
		}

		switch respType2 {
		case marketplace.MsgTypeX402Response:
			return extractX402Output(respData2)
		case marketplace.MsgTypeX402Error:
			return extractX402ErrorMessage(respData2)
		default:
			return "", fmt.Errorf("unexpected response type to paid_request: %s", respType2)
		}

	case marketplace.MsgTypeX402Error:
		return extractX402ErrorMessage(respData)

	default:
		return "", fmt.Errorf("unexpected response type to x402.request: %s", respType)
	}
}

// agentPrice returns the price for a given agent ID (checks local then CRDT).
func (m *Manager) agentPrice(agentID string) float64 {
	m.mu.RLock()
	la, ok := m.localAgents[agentID]
	m.mu.RUnlock()
	if ok && la != nil {
		return la.Price
	}
	if m.listingService != nil {
		if listing, ok := m.listingService.GetListing(agentID); ok {
			return listing.Price
		}
	}
	return 0
}

// sendX402Error is a convenience helper that marshals an X402Error and returns the typed tuple.
func sendX402Error(correlationID string, code marketplace.X402ErrorCode, message string) (string, []byte, error) {
	e := marketplace.NewX402Error(correlationID, code, message)
	data, _ := json.Marshal(e)
	return marketplace.MsgTypeX402Error, data, nil
}

// envelopeToPaymentHeader converts an X402PaymentEnvelope to the legacy PaymentHeader type
// used by PaymentService.VerifyAndSettle.
func envelopeToPaymentHeader(env *marketplace.X402PaymentEnvelope) *marketplace.PaymentHeader {
	if env == nil {
		return nil
	}
	req := marketplace.CreatePaymentRequirements(
		env.Network,
		"", // Amount comes from Payload.Authorization.Value
		marketplace.GetUSDCAddress(env.Network),
		"", // PayTo comes from Payload.Authorization.To
		marketplace.DefaultTimeout,
	)
	if env.Payload != nil {
		req.Amount = env.Payload.Authorization.Value
		req.PayTo = env.Payload.Authorization.To
	}
	var sig string
	if env.Payload != nil {
		sig = env.Payload.Signature
	}
	return &marketplace.PaymentHeader{
		Requirement: req,
		Accepted:    &req,
		Payer:       env.Payer,
		PaymentID:   "",
		Signature:   sig,
		Payload:     env.Payload,
	}
}

// paymentHeaderToEnvelope converts a PaymentHeader to an X402PaymentEnvelope for the wire.
func paymentHeaderToEnvelope(ph *marketplace.PaymentHeader, serverNonce string) marketplace.X402PaymentEnvelope {
	return marketplace.X402PaymentEnvelope{
		X402Version: 2,
		Scheme:      ph.Requirement.Scheme,
		Network:     ph.Requirement.Network,
		ServerNonce: serverNonce,
		Payer:       ph.Payer,
		Payload:     ph.Payload,
	}
}

// extractX402Output parses an x402.response and returns the output string.
func extractX402Output(data []byte) (string, error) {
	var resp marketplace.X402Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal x402.response: %w", err)
	}
	fmt.Printf("[RemoteExecuteX402] tx_hash=%s payment_id=%s\n", resp.TxHash, resp.PaymentID)
	if len(resp.Body) == 0 {
		return "", nil
	}
	// Body is a JSON object with an "output" key; decode it.
	var body map[string]string
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		// Fallback: treat as base64-encoded raw string.
		decoded, err2 := base64.StdEncoding.DecodeString(string(resp.Body))
		if err2 != nil {
			return string(resp.Body), nil
		}
		return string(decoded), nil
	}
	return body["output"], nil
}

// extractX402ErrorMessage parses an x402.error and returns an error.
func extractX402ErrorMessage(data []byte) (string, error) {
	var e marketplace.X402Error
	if err := json.Unmarshal(data, &e); err != nil {
		return "", fmt.Errorf("x402 error (unparseable): %s", string(data))
	}
	return "", fmt.Errorf("x402 error %d (%s): %s", e.ErrorCode, e.ErrorName, e.Message)
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
			paymentHeader.Payer, paymentHeader.Requirement.PayTo, paymentHeader.Requirement.Amount)
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
			fmt.Printf("[RemoteExecute] Payment requirement - Amount: %s, PayTo: %s\n",
				marketplace.FormatUSDC(payResp.PaymentRequirement.Amount),
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
