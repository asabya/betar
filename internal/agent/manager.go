package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/asabya/betar/internal/eip8004"
	"github.com/asabya/betar/internal/ipfs"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/pkg/types"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// SessionStorer is the interface Manager uses to persist session data.
// Implemented by *session.Store.
type SessionStorer interface {
	AddExchange(ctx context.Context, agentID, callerID string, ex types.Exchange) error
}

type ctxKey string

const ctxKeySkipSession ctxKey = "skipSession"

// Manager manages agent lifecycle
type Manager struct {
	defaultCfg        ADKConfig
	runtimes          map[string]Runtime // agentDID → isolated Runtime
	ipfsClient        *ipfs.Client
	p2pHost           *p2p.Host
	streamHandler     *p2p.StreamHandler
	x402StreamHandler *p2p.X402StreamHandler
	listingService    *marketplace.AgentListingService
	paymentService    *marketplace.PaymentService
	walletAddress     string
	sessionStore      SessionStorer
	nodeDID           string // this node's DID (did:key:xxx) for identifying as caller
	eip8004           *eip8004.Client
	eip8004Tokens     *eip8004.TokenStore

	mu            sync.RWMutex
	localAgents   map[string]*LocalAgent
	customHandler func(ctx context.Context, agentID string, input []byte) (output []byte, err error)
}

// LocalAgent represents a local agent managed by this node
type LocalAgent struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	MetadataCID string    `json:"metadataCID"`
	AgentID     string    `json:"agentID"` // ADK runtime agent ID
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	TokenID     *big.Int  `json:"tokenId,omitempty"`  // EIP-8004 on-chain token ID
	AgentAPI    string    `json:"agentApi,omitempty"` // Optional API endpoint for the agent
}

// NewManager creates a new agent manager
func NewManager(runtimeCfg ADKConfig, ipfsClient *ipfs.Client, p2pHost *p2p.Host, listingService *marketplace.AgentListingService, privKey crypto.PrivKey, paymentSvc *marketplace.PaymentService, walletAddr string, sessionStore SessionStorer, eip8004Client ...*eip8004.Client) (*Manager, error) {
	if ipfsClient == nil {
		return nil, fmt.Errorf("ipfs client is required")
	}
	if p2pHost == nil {
		return nil, fmt.Errorf("p2p host is required")
	}

	// Store the private key in the default config for per-agent runtime creation.
	runtimeCfg.PrivKey = privKey

	m := &Manager{
		defaultCfg:     runtimeCfg,
		runtimes:       make(map[string]Runtime),
		ipfsClient:     ipfsClient,
		p2pHost:        p2pHost,
		listingService: listingService,
		paymentService: paymentSvc,
		walletAddress:  walletAddr,
		sessionStore:   sessionStore,
		nodeDID:        GenerateDID(privKey, runtimeCfg.AppName, "node"),
		localAgents:    make(map[string]*LocalAgent),
	}
	if len(eip8004Client) > 0 && eip8004Client[0] != nil {
		m.eip8004 = eip8004Client[0]
	}

	return m, nil
}

// RegisterX402Handlers wires up the x402 stream handler and stores a reference for client use.
func (m *Manager) RegisterX402Handlers(sh *p2p.X402StreamHandler) {
	m.x402StreamHandler = sh
	sh.RegisterHandler(marketplace.MsgTypeX402Request, m.handleX402Request)
	sh.RegisterHandler(marketplace.MsgTypeX402PaidRequest, m.handleX402PaidRequest)
}

// RegisterStreamHandlers registers marketplace stream handlers on the basic StreamHandler.
// Currently registers an "info" handler that returns all known local listings as JSON.
func (m *Manager) RegisterStreamHandlers(sh *p2p.StreamHandler) {
	m.streamHandler = sh
	sh.RegisterHandler("info", func(ctx context.Context, from peer.ID, _ []byte) ([]byte, error) {
		listings := m.listingService.ListListings()
		return json.Marshal(listings)
	})

	sh.RegisterHandler(marketplace.MsgTypeExecRequest, m.handleExecuteRequest)
	sh.RegisterHandler(marketplace.MsgTypeExecPaymentRequired, m.handleExecutePaymentRequired)
	sh.RegisterHandler(marketplace.MsgTypeExecResponse, m.handleExecuteResponse)
	sh.RegisterHandler(marketplace.MsgTypeExecError, m.handleErrorResponse)
}

// SetCustomHandler registers a custom task handler that is used instead of the
// ADK runtime for agents registered with CustomHandler: true.
func (m *Manager) SetCustomHandler(h func(ctx context.Context, agentID string, input []byte) (output []byte, err error)) {
	m.mu.Lock()
	m.customHandler = h
	m.mu.Unlock()
}

// SetTokenStore sets the EIP-8004 token store for persisting on-chain tokenIDs.
func (m *Manager) SetTokenStore(ts *eip8004.TokenStore) {
	m.eip8004Tokens = ts
}

// RegisterAgent registers a new agent locally and publishes to marketplace
func (m *Manager) RegisterAgent(ctx context.Context, spec AgentSpec) (*LocalAgent, error) {
	var runtimeAgentID string
	var rt *ADKRuntime

	if spec.CustomHandler {
		// Skip ADK runtime — agent will be served by a custom TaskHandler via sdk.Serve().
		runtimeAgentID = GenerateDID(m.defaultCfg.PrivKey, m.defaultCfg.AppName, spec.Name)
	} else {
		// Build per-agent runtime config, falling back to global defaults.
		apiKey := spec.APIKey
		if apiKey == "" {
			apiKey = m.defaultCfg.APIKey
		}
		model := spec.Model
		if model == "" {
			model = m.defaultCfg.ModelName
		}
		provider := spec.Provider
		if provider == "" {
			provider = m.defaultCfg.Provider
		}
		openAIAPIKey := spec.OpenAIAPIKey
		if openAIAPIKey == "" {
			openAIAPIKey = m.defaultCfg.OpenAIAPIKey
		}
		openAIBaseURL := spec.OpenAIBaseURL
		if openAIBaseURL == "" {
			openAIBaseURL = m.defaultCfg.OpenAIBaseURL
		}
		agentAPI := spec.AgentAPI
		var err error
		rt, err = NewADKRuntime(ADKConfig{
			AppName:       m.defaultCfg.AppName,
			ModelName:     model,
			APIKey:        apiKey,
			PrivKey:       m.defaultCfg.PrivKey,
			Provider:      provider,
			OpenAIAPIKey:  openAIAPIKey,
			OpenAIBaseURL: openAIBaseURL,
			AgentAPI:      agentAPI,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create agent runtime: %w", err)
		}

		// Create agent in the per-agent runtime.
		if agentAPI != "" {
			// For custom agent hosting, the "model" is actually the agent URL.
			runtimeAgentID, err = rt.CreateHTTPAgent(ctx, spec)
			for i := 0; i < len(rt.agents); i++ {
				fmt.Printf("Waiting for HTTP agent to be available in runtime... (agentID: %s)\n", runtimeAgentID)
			}
		} else {
			runtimeAgentID, err = rt.CreateAgent(ctx, spec)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create agent: %w", err)
		}
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
	agentID := m.p2pHost.ID().String() + "/" + runtimeAgentID
	if spec.CustomHandler {
		// DID is already globally unique — no peer ID prefix needed.
		agentID = runtimeAgentID
	}
	agent := &LocalAgent{
		ID:          agentID,
		Name:        spec.Name,
		Description: spec.Description,
		Price:       spec.Price,
		MetadataCID: metadataCID,
		AgentID:     runtimeAgentID,
		Status:      "active",
		CreatedAt:   time.Now(),
		AgentAPI:    spec.AgentAPI,
	}

	// Best-effort on-chain registration via EIP-8004 (only when explicitly requested)
	if m.eip8004 != nil && spec.OnChain {
		// Check if this agent was already registered on-chain (by name).
		if m.eip8004Tokens != nil {
			if existing := m.eip8004Tokens.Get(spec.Name); existing != nil {
				agent.TokenID = existing
				fmt.Printf("EIP-8004: agent %q already registered on-chain with tokenID=%s (skipping)\n", spec.Name, existing.String())
				goto skipOnChain
			}
		}
		{
			tokenID, err := m.eip8004.RegisterIdentity(ctx, metadataCID)
			if err != nil {
				fmt.Printf("warning: EIP-8004 on-chain registration failed: %v\n", err)
			} else if tokenID != nil {
				agent.TokenID = tokenID
				fmt.Printf("EIP-8004: agent registered on-chain with tokenID=%s\n", tokenID.String())
				if m.eip8004Tokens != nil {
					if err := m.eip8004Tokens.Put(spec.Name, tokenID); err != nil {
						fmt.Printf("warning: failed to persist EIP-8004 tokenID: %v\n", err)
					}
				}
			}
		}
	skipOnChain:
	}

	// Store locally, indexed by the runtime agent ID.
	m.mu.Lock()
	if rt != nil {
		m.runtimes[runtimeAgentID] = rt
	}
	m.localAgents[agent.AgentID] = agent
	m.mu.Unlock()

	return agent, nil
}

// ExecuteTask executes a task on a local agent or a remote agent from CRDT
func (m *Manager) ExecuteTask(ctx context.Context, agentID string, reqbody []byte) (string, error) {
	var request types.AgentRequest
	if err := json.Unmarshal(reqbody, &request); err != nil {
		return "", fmt.Errorf("failed to unmarshal request body: %w", err)
	}
	fmt.Printf("[ExecuteTask] Starting task execution for agentID: %s\n", agentID)

	// Find local agent first
	m.mu.RLock()
	agent, ok := m.localAgents[agentID]
	m.mu.RUnlock()

	if ok {
		fmt.Printf("[ExecuteTask] Found local agent: %s (name: %s)\n", agent.AgentID, agent.Name)
		// Execute locally using the per-agent runtime.
		m.mu.RLock()
		rt, rtok := m.runtimes[agent.AgentID]
		handler := m.customHandler
		m.mu.RUnlock()

		// If no runtime, try the custom handler (for CustomHandler agents).
		if !rtok && handler != nil {
			fmt.Printf("[ExecuteTask] No runtime, using custom handler for agent: %s\n", agent.AgentID)
			output, err := handler(ctx, agent.AgentID, []byte(request.Input))
			if err != nil {
				return "", fmt.Errorf("custom handler failed: %w", err)
			}
			return string(output), nil
		}
		if !rtok {
			return "", fmt.Errorf("runtime not found for agent: %s", agent.AgentID)
		}

		requestID := fmt.Sprintf("local-%d", time.Now().UnixNano())

		result, err := rt.RunTask(ctx, types.TaskRequest{
			AgentID:   agent.AgentID,
			Input:     request.Input,
			RequestID: requestID,
		})
		if err != nil {
			fmt.Printf("[ExecuteTask] Local execution failed: %v\n", err)
			return "", fmt.Errorf("failed to run agent: %w", err)
		}

		if result.Error != "" {
			fmt.Printf("[ExecuteTask] Local execution returned error: %s\n", result.Error)
			return "", fmt.Errorf("runtime error: %s", result.Error)
		}

		fmt.Printf("[ExecuteTask] Local execution successful, output length: %d\n", len(result.Output))

		if m.sessionStore != nil && ctx.Value(ctxKeySkipSession) == nil {
			ex := types.Exchange{
				RequestID: requestID,
				Input:     request.Input,
				Output:    result.Output,
				Timestamp: result.Timestamp,
			}
			_ = m.sessionStore.AddExchange(ctx, agentID, "local", ex)
		}

		return result.Output, nil
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
				return "", fmt.Errorf("failed to connect to remote agent: %w", err)
			}

			fmt.Printf("[ExecuteTask] Connected to peer %s, executing remotely via x402\n", peerID)
			// Request PaymentRequired
			// return m.RemoteExecuteX402(ctx, peerID, runtimeAgentID, input)
			// TODO: m.RemoteExecute
			return m.RemoteExecute(ctx, peerID, runtimeAgentID, reqbody)
		}
		fmt.Printf("[ExecuteTask] Agent not found in CRDT listings\n")
	} else {
		fmt.Printf("[ExecuteTask] Listing service not available, cannot search CRDT\n")
	}

	fmt.Printf("[ExecuteTask] Agent not found: %s\n", agentID)
	return "", fmt.Errorf("agent not found: %s", agentID)
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
	var rt Runtime
	var rtok bool
	if ok {
		rt, rtok = m.runtimes[agent.AgentID]
		delete(m.localAgents, agentID)
		delete(m.runtimes, agent.AgentID)
	}
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	if rtok {
		if err := rt.DeleteAgent(ctx, agent.AgentID); err != nil {
			return fmt.Errorf("failed to delete runtime agent: %w", err)
		}
	}

	return nil
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

// agentTokenIDString returns the string representation of a local agent's on-chain tokenID.
func (m *Manager) agentTokenIDString(agentID string) string {
	m.mu.RLock()
	la, ok := m.localAgents[agentID]
	m.mu.RUnlock()
	if ok && la != nil && la.TokenID != nil {
		return la.TokenID.String()
	}
	return ""
}

// AgentSpec represents agent specification for registration
type AgentSpec struct {
	Name        string
	Description string
	Image       string
	Price       float64
	Model       string
	APIKey      string // per-agent Google API key; empty = use global default
	Services    []types.Service
	X402Support bool
	OnChain     bool // if true, register on-chain via EIP-8004 (default: false)

	// Provider fields
	Provider        string // "google", "openai", or "" for auto-detect
	OpenAIAPIKey    string // OpenAI-compatible API key
	OpenAIBaseURL   string // OpenAI-compatible base URL
	AgentAPI        string // URL for custom agent hosting (must implement /execute endpoint)
	AgentCardSource string // "AgentCardSource" URL

	// CustomHandler skips ADK runtime creation. Use this when the agent
	// will be served via sdk.Serve() with a custom TaskHandler.
	CustomHandler bool
}
