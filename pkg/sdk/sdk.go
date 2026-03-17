// Package sdk provides a simple Go client library for participating in the
// Betar decentralized agent marketplace. It wraps the internal P2P, IPFS,
// marketplace, and payment subsystems behind an ergonomic API so that an
// external developer can register, discover, and execute agents in under
// 20 lines of code.
//
// Quick start:
//
//	c, _ := sdk.NewClient(sdk.Config{})
//	defer c.Close()
//
//	c.Register(ctx, sdk.AgentSpec{Name: "my-agent", Description: "does things"})
//	results, _ := c.Discover(ctx, "")
//	output, _ := c.Execute(ctx, results[0].ID, "hello")
package sdk

import (
	"context"
	"fmt"
	"sync"

	"github.com/asabya/betar/internal/agent"
	"github.com/asabya/betar/internal/config"
	"github.com/asabya/betar/internal/eth"
	"github.com/asabya/betar/internal/ipfs"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/internal/session"
	"github.com/asabya/betar/pkg/types"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Config configures a Betar SDK client. Zero-value fields fall back to
// environment variables and sensible defaults (same as the CLI).
type Config struct {
	// DataDir is the local data directory. Default: ~/.betar
	DataDir string

	// P2PPort overrides the libp2p listen port. Default: 4001.
	P2PPort int

	// BootstrapPeers is a list of multiaddr strings for DHT bootstrap.
	BootstrapPeers []string

	// EthereumRPC is the JSON-RPC endpoint for the settlement chain.
	// Default: https://sepolia.base.org
	EthereumRPC string

	// EthereumKey is a hex-encoded secp256k1 private key for signing
	// payments. If empty, a key is loaded/generated at DataDir/wallet.key.
	EthereumKey string

	// LLMProvider selects the LLM backend: "google" or "openai".
	// Empty string auto-detects from available API keys.
	LLMProvider string

	// GoogleAPIKey is the API key for Gemini models.
	GoogleAPIKey string

	// GoogleModel overrides the Gemini model name. Default: gemini-2.5-flash.
	GoogleModel string

	// OpenAIAPIKey is the API key for OpenAI-compatible providers.
	OpenAIAPIKey string

	// OpenAIBaseURL is the base URL for OpenAI-compatible providers.
	OpenAIBaseURL string
}

// Client is the main entry point for the Betar SDK. It manages a full P2P
// node, IPFS-lite store, marketplace CRDT, payment service, and agent runtime.
// A Client is safe for concurrent use.
type Client struct {
	mu sync.Mutex

	cfg          *config.Config
	ctx          context.Context
	cancel       context.CancelFunc
	p2pHost      *p2p.Host
	pubsub       *p2p.PubSub
	discovery    *p2p.Discovery
	ipfs         *ipfs.Client
	listing      *marketplace.AgentListingService
	payment      *marketplace.PaymentService
	manager      *agent.Manager
	sessionStore *session.Store
	stream       *p2p.StreamHandler
	x402Stream   *p2p.X402StreamHandler
	walletAddr   string
	serveHandler TaskHandler
	serveMu      sync.RWMutex
}

// AgentSpec describes an agent to register on the marketplace.
type AgentSpec = agent.AgentSpec

// AgentListing is a marketplace agent listing.
type AgentListing = types.AgentListing

// TaskHandler is a function that handles inbound execution requests for a
// served agent. It receives the task input string and returns the output
// string or an error.
type TaskHandler func(ctx context.Context, agentID, input string) (output string, err error)

// NewClient creates a new Betar SDK client. It initialises the P2P host,
// IPFS-lite node, marketplace services, and payment system. Call Close when
// done to release resources.
func NewClient(sdkCfg Config) (*Client, error) {
	// Load base config from environment, then apply overrides.
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("sdk: load config: %w", err)
	}
	applyOverrides(cfg, &sdkCfg)

	ctx, cancel := context.WithCancel(context.Background())

	c := &Client{
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
	}

	if err := c.init(); err != nil {
		cancel()
		return nil, err
	}

	return c, nil
}

// Close shuts down the client and releases all resources.
func (c *Client) Close() error {
	c.cancel()
	if c.sessionStore != nil {
		c.sessionStore.Close()
	}
	if c.p2pHost != nil {
		return c.p2pHost.Close()
	}
	return nil
}

// PeerID returns the libp2p peer ID of this node.
func (c *Client) PeerID() peer.ID {
	return c.p2pHost.ID()
}

// Addrs returns the multiaddr strings this node is listening on.
func (c *Client) Addrs() []string {
	return c.p2pHost.AddrStrings()
}

// WalletAddress returns the Ethereum wallet address used for payments.
func (c *Client) WalletAddress() string {
	return c.walletAddr
}

// Register registers a new agent on the local node and publishes it to the
// marketplace CRDT so that other peers can discover and call it.
func (c *Client) Register(ctx context.Context, spec AgentSpec) (*agent.LocalAgent, error) {
	la, err := c.manager.RegisterAgent(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("sdk: register: %w", err)
	}

	// Publish to marketplace CRDT.
	msg := &types.AgentListingMessage{
		Type:     "list",
		AgentID:  la.ID,
		Name:     la.Name,
		Price:    la.Price,
		Metadata: la.MetadataCID,
		SellerID: c.p2pHost.ID().String(),
		Addrs:    c.p2pHost.AddrStrings(),
	}
	if la.TokenID != nil {
		msg.TokenID = la.TokenID.String()
	}
	if err := c.listing.ListAgent(ctx, msg); err != nil {
		return nil, fmt.Errorf("sdk: publish listing: %w", err)
	}

	return la, nil
}

// Discover returns all agent listings known to this node. If query is
// non-empty, only listings whose Name contains the query are returned.
func (c *Client) Discover(ctx context.Context, query string) ([]AgentListing, error) {
	_ = ctx
	all := c.listing.ListListings()
	if query == "" {
		out := make([]AgentListing, 0, len(all))
		for _, l := range all {
			if l != nil {
				out = append(out, *l)
			}
		}
		return out, nil
	}

	var out []AgentListing
	for _, l := range all {
		if l != nil && contains(l.Name, query) {
			out = append(out, *l)
		}
	}
	return out, nil
}

// Execute calls a remote (or local) agent by its marketplace ID. If the
// agent requires payment, the x402 flow is handled automatically using the
// configured Ethereum wallet.
func (c *Client) Execute(ctx context.Context, agentID, input string) (string, error) {
	output, err := c.manager.ExecuteTask(ctx, agentID, input)
	if err != nil {
		return "", fmt.Errorf("sdk: execute: %w", err)
	}
	return output, nil
}

// Serve registers a custom handler that will be invoked for all inbound
// execution requests targeting agents on this node. This is useful for
// agents that don't use the ADK runtime and want to handle requests with
// custom Go code.
//
// The handler replaces the default ADK-based execution for any agent
// registered on this node.
func (c *Client) Serve(handler TaskHandler) {
	c.serveMu.Lock()
	c.serveHandler = handler
	c.serveMu.Unlock()
}

// init performs the full node bootstrap sequence.
func (c *Client) init() error {
	cfg := c.cfg
	ctx := c.ctx

	// P2P host
	var err error
	c.p2pHost, err = p2p.NewHost(ctx, cfg.P2P)
	if err != nil {
		return fmt.Errorf("sdk: p2p host: %w", err)
	}

	// PubSub
	c.pubsub, err = p2p.NewPubSub(ctx, c.p2pHost.RawHost())
	if err != nil {
		return fmt.Errorf("sdk: pubsub: %w", err)
	}

	// Stream handlers
	c.stream = p2p.NewStreamHandler(c.p2pHost.RawHost())
	c.x402Stream = p2p.NewX402StreamHandler(c.p2pHost.RawHost())

	// Discovery
	c.discovery, err = p2p.NewDiscovery(ctx, c.p2pHost.RawHost(), cfg.P2P)
	if err != nil {
		return fmt.Errorf("sdk: discovery: %w", err)
	}
	if err := c.discovery.DiscoverPeers(ctx, cfg.P2P.BootstrapPeers); err != nil {
		// Non-fatal — discovery may partially succeed.
		fmt.Printf("sdk: discovery bootstrap warning: %v\n", err)
	}

	// IPFS-lite
	c.ipfs, err = ipfs.NewClient(ctx, c.p2pHost.RawHost(), c.discovery.Routing(), cfg.Storage.DataDir)
	if err != nil {
		return fmt.Errorf("sdk: ipfs: %w", err)
	}

	// Listing service (CRDT)
	c.listing, err = marketplace.NewAgentListingService(ctx, c.ipfs, c.pubsub, c.p2pHost.ID())
	if err != nil {
		return fmt.Errorf("sdk: listing service: %w", err)
	}

	// Payment service (optional — requires Ethereum config)
	if cfg.Ethereum != nil && cfg.Ethereum.PrivateKey != "" {
		c.walletAddr, _ = config.GetAddressFromKey(cfg.Ethereum.PrivateKey)
		if cfg.Ethereum.RPCURL != "" {
			wallet, err := eth.NewWallet(cfg.Ethereum.PrivateKey, cfg.Ethereum.RPCURL)
			if err == nil {
				c.payment = marketplace.NewPaymentService(wallet, c.walletAddr)
			}
		}
	}

	// Session store
	c.sessionStore, err = session.NewStore(cfg.Storage.SessionsDir)
	if err != nil {
		c.sessionStore = nil
	}

	// Agent manager
	c.manager, err = agent.NewManager(agent.ADKConfig{
		AppName:       "betar",
		ModelName:     cfg.Agent.Model,
		APIKey:        cfg.Agent.APIKey,
		Provider:      cfg.Agent.Provider,
		OpenAIAPIKey:  cfg.Agent.OpenAIAPIKey,
		OpenAIBaseURL: cfg.Agent.OpenAIBaseURL,
	}, c.ipfs, c.p2pHost, c.listing, cfg.P2P.PrivKey, c.payment, c.walletAddr, c.sessionStore)
	if err != nil {
		return fmt.Errorf("sdk: agent manager: %w", err)
	}

	// Wire handlers
	c.manager.RegisterX402Handlers(c.x402Stream)
	c.manager.RegisterStreamHandlers(c.stream)

	// Start announcement listener for remote agent discovery.
	c.listing.StartAnnouncementListener(ctx, c.pubsub)

	return nil
}

// applyOverrides merges SDK config overrides into the loaded config.
func applyOverrides(cfg *config.Config, sdk *Config) {
	if sdk.DataDir != "" {
		cfg.Storage.DataDir = sdk.DataDir
	}
	if sdk.P2PPort != 0 {
		cfg.P2P.Port = sdk.P2PPort
	}
	if len(sdk.BootstrapPeers) > 0 {
		cfg.P2P.BootstrapPeers = sdk.BootstrapPeers
	}
	if sdk.EthereumRPC != "" {
		cfg.Ethereum.RPCURL = sdk.EthereumRPC
	}
	if sdk.EthereumKey != "" {
		cfg.Ethereum.PrivateKey = sdk.EthereumKey
	}
	if sdk.LLMProvider != "" {
		cfg.Agent.Provider = sdk.LLMProvider
	}
	if sdk.GoogleAPIKey != "" {
		cfg.Agent.APIKey = sdk.GoogleAPIKey
	}
	if sdk.GoogleModel != "" {
		cfg.Agent.Model = sdk.GoogleModel
	}
	if sdk.OpenAIAPIKey != "" {
		cfg.Agent.OpenAIAPIKey = sdk.OpenAIAPIKey
	}
	if sdk.OpenAIBaseURL != "" {
		cfg.Agent.OpenAIBaseURL = sdk.OpenAIBaseURL
	}
}

// contains reports whether s contains substr (case-sensitive).
func contains(s, substr string) bool {
	return len(substr) <= len(s) && searchString(s, substr) >= 0
}

func searchString(s, substr string) int {
	n := len(substr)
	for i := 0; i <= len(s)-n; i++ {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}
