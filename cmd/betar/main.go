package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/asabya/betar/cmd/betar/api"
	"github.com/asabya/betar/cmd/betar/tui"
	"github.com/asabya/betar/internal/agent"
	"github.com/asabya/betar/internal/config"
	"github.com/asabya/betar/internal/eth"
	"github.com/asabya/betar/internal/ipfs"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/internal/session"
	"github.com/asabya/betar/pkg/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/spf13/cobra"
)

var (
	cfg               *config.Config
	ctx               context.Context
	cancel            context.CancelFunc
	p2pHost           *p2p.Host
	p2pPubSub         *p2p.PubSub
	streamHandler     *p2p.StreamHandler
	x402StreamHandler *p2p.X402StreamHandler
	discovery         *p2p.Discovery
	agentManager      *agent.Manager
	listingService    *marketplace.AgentListingService
	orderService      *marketplace.OrderService
	paymentService    *marketplace.PaymentService
	ipfsClient        *ipfs.Client
	apiServer         *api.Server
	sessionStore      *session.Store
)

var rootCmd = &cobra.Command{
	Use:   "betar",
	Short: "P2P Agent 2 Agent Marketplace",
	Long:  "A decentralized marketplace where AI agents can discover, list, and transact with each other",
	RunE:  runTUI,
}

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Start a marketplace node",
	RunE:  runNode,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start node, agent, and marketplace in one process",
	RunE:  runStart,
}

func runTUI(cmd *cobra.Command, args []string) error {
	if err := initRuntime(cmd); err != nil {
		return err
	}
	defer shutdownRuntime()

	tui.SetRuntime(p2pHost, agentManager, listingService, orderService)
	tui.SetWallet(deriveWalletAddress(cfg.Ethereum.PrivateKey))
	tui.SetDataDir(cfg.Storage.DataDir)

	// Redirect stdout into the TUI log panel.
	origStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr == nil {
		os.Stdout = w
		tui.SetOutput(origStdout)
		go func() {
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				tui.SendLog(scanner.Text())
			}
		}()
	}

	// If --name is provided, run the full agent lifecycle (like `start`).
	name, _ := cmd.Flags().GetString("name")
	if name != "" {
		registered, listingMsg, err := registerLocalAgentFromFlags(ctx, cmd)
		if err != nil {
			return err
		}

		if listingService != nil {
			listingService.UpsertLocalListing(listingMsg)
			if data, err := json.Marshal(listingMsg); err == nil {
				_ = p2pPubSub.Publish(ctx, marketplace.AnnounceTopic, data)
			}
		}

		apiPort, _ := cmd.Flags().GetInt("api-port")
		apiServer = api.NewServer(apiPort, agentManager, listingService, orderService, p2pHost, paymentService, sessionStore)
		if err := apiServer.Start(); err != nil {
			fmt.Printf("warning: failed to start API server: %v\n", err)
		} else {
			fmt.Printf("HTTP API server running on port %d\n", apiPort)
		}

		announceInterval, _ := cmd.Flags().GetDuration("announce-interval")
		if announceInterval < 5*time.Second {
			announceInterval = 5 * time.Second
		}
		go runListingAnnouncer(ctx, announceInterval, func(ts int64) *types.AgentListingMessage {
			msg := *listingMsg
			msg.Type = "update"
			msg.Timestamp = ts
			return &msg
		})

		fmt.Printf("Agent registered: %s (%s)\n", registered.Name, registered.AgentID)
	}

	// Auto-load agents from agents.yaml.
	announceInterval, _ := cmd.Flags().GetDuration("announce-interval")
	if announceInterval < 5*time.Second {
		announceInterval = 5 * time.Second
	}
	if err := loadAndRegisterAgentsFromConfig(ctx, announceInterval); err != nil {
		fmt.Printf("warning: %v\n", err)
	}

	fmt.Println("Starting Betar TUI...")
	printRuntimeInfo()

	tuiErr := tui.RunTUI()
	if pipeErr == nil {
		_ = w.Close()
		os.Stdout = origStdout
	}
	return tuiErr
}

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agents",
}

var agentServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run a P2P node and serve one local agent",
	RunE:  serveAgent,
}

var agentRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register an agent",
	RunE:  registerAgent,
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List agents",
	RunE:  listAgents,
}

var agentDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover agents",
	RunE:  discoverAgents,
}

var agentExecuteCmd = &cobra.Command{
	Use:   "execute",
	Short: "Execute a task with an agent",
	RunE:  executeAgent,
}

var agentConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage agent configurations",
}

var agentConfigListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured agent profiles",
	RunE:  agentConfigList,
}

var agentConfigAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new agent profile",
	RunE:  agentConfigAdd,
}

var agentConfigDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete an agent profile",
	Args:  cobra.ExactArgs(1),
	RunE:  agentConfigDelete,
}

var agentConfigEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit an agent profile",
	Args:  cobra.ExactArgs(1),
	RunE:  agentConfigEdit,
}

var orderCmd = &cobra.Command{
	Use:   "order",
	Short: "Manage orders",
}

var orderCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an order",
	RunE:  createOrder,
}

var walletCmd = &cobra.Command{
	Use:   "wallet",
	Short: "Manage wallet",
}

var walletBalanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Check wallet balance",
	RunE:  checkBalance,
}

func init() {
	// Node flags
	nodeCmd.Flags().IntP("port", "p", 4001, "Port to listen on")
	nodeCmd.Flags().StringSlice("bootstrap", []string{}, "Bootstrap peers")
	nodeCmd.Flags().String("model", "gemini-2.5-flash", "ADK model name")
	nodeCmd.Flags().String("provider", "", "LLM provider: google, openai, or empty for auto-detect")
	nodeCmd.Flags().String("openai-api-key", "", "OpenAI-compatible API key (overrides OPENAI_API_KEY)")
	nodeCmd.Flags().String("openai-base-url", "", "OpenAI-compatible base URL, e.g. http://localhost:11434/v1/")

	// Agent serve flags
	agentServeCmd.Flags().IntP("port", "p", 4001, "Port to listen on")
	agentServeCmd.Flags().StringSlice("bootstrap", []string{}, "Bootstrap peers")
	agentServeCmd.Flags().String("model", "gemini-2.5-flash", "ADK model name")
	agentServeCmd.Flags().StringP("name", "n", "", "Agent name")
	agentServeCmd.Flags().StringP("description", "d", "", "Agent description")
	agentServeCmd.Flags().Float64P("price", "r", 0, "Price per task")
	agentServeCmd.Flags().String("provider", "", "LLM provider: google, openai, or empty for auto-detect")
	agentServeCmd.Flags().String("openai-api-key", "", "OpenAI-compatible API key (overrides OPENAI_API_KEY)")
	agentServeCmd.Flags().String("openai-base-url", "", "OpenAI-compatible base URL, e.g. http://localhost:11434/v1/")
	agentServeCmd.Flags().Duration("announce-interval", 30*time.Second, "How often to republish agent CRDT listing")
	_ = agentServeCmd.MarkFlagRequired("name")

	// Unified start flags
	startCmd.Flags().IntP("port", "p", 4001, "Port to listen on")
	startCmd.Flags().StringSlice("bootstrap", []string{}, "Bootstrap peers")
	startCmd.Flags().String("model", "gemini-2.5-flash", "ADK model name")
	startCmd.Flags().StringP("name", "n", "", "Agent name")
	startCmd.Flags().StringP("description", "d", "", "Agent description")
	startCmd.Flags().Float64P("price", "r", 0, "Price per task")
	startCmd.Flags().Duration("announce-interval", 30*time.Second, "How often to republish agent CRDT listing")
	startCmd.Flags().Int("api-port", 8424, "HTTP API server port")
	startCmd.Flags().String("provider", "", "LLM provider: google, openai, or empty for auto-detect")
	startCmd.Flags().String("openai-api-key", "", "OpenAI-compatible API key (overrides OPENAI_API_KEY)")
	startCmd.Flags().String("openai-base-url", "", "OpenAI-compatible base URL, e.g. http://localhost:11434/v1/")
	// --name is optional for startCmd; agents can be loaded from agents.yaml

	// Agent register flags
	agentRegisterCmd.Flags().StringP("name", "n", "", "Agent name")
	agentRegisterCmd.Flags().StringP("description", "d", "", "Agent description")
	agentRegisterCmd.Flags().Float64P("price", "p", 0, "Price per task")

	// Agent list flags
	agentListCmd.Flags().String("api-url", "http://localhost:8424", "API server URL")

	// Agent discover flags
	agentDiscoverCmd.Flags().String("api-url", "http://localhost:8424", "API server URL")

	// Agent execute flags
	agentExecuteCmd.Flags().String("api-url", "http://localhost:8424", "API server URL")
	agentExecuteCmd.Flags().String("agent-id", "", "Agent ID")
	agentExecuteCmd.Flags().StringP("task", "t", "", "Task input")

	// Order create flags
	orderCreateCmd.Flags().String("api-url", "http://localhost:8424", "API server URL")
	orderCreateCmd.Flags().String("agent-id", "", "Agent ID")
	orderCreateCmd.Flags().Float64("price", 0, "Price")

	// Agent config add flags
	agentConfigAddCmd.Flags().StringP("name", "n", "", "Agent name")
	agentConfigAddCmd.Flags().StringP("description", "d", "", "Agent description")
	agentConfigAddCmd.Flags().Float64P("price", "r", 0, "Price per task")
	agentConfigAddCmd.Flags().String("model", "", "ADK model name (overrides global GOOGLE_MODEL)")
	agentConfigAddCmd.Flags().String("api-key", "", "Google API key (overrides global GOOGLE_API_KEY)")
	agentConfigAddCmd.Flags().String("provider", "", "LLM provider: google, openai, or empty for auto-detect")
	agentConfigAddCmd.Flags().String("openai-api-key", "", "OpenAI-compatible API key")
	agentConfigAddCmd.Flags().String("openai-base-url", "", "OpenAI-compatible base URL")
	_ = agentConfigAddCmd.MarkFlagRequired("name")

	// Agent config edit flags
	agentConfigEditCmd.Flags().StringP("description", "d", "", "Agent description")
	agentConfigEditCmd.Flags().Float64P("price", "r", 0, "Price per task")
	agentConfigEditCmd.Flags().String("model", "", "ADK model name")
	agentConfigEditCmd.Flags().String("api-key", "", "Google API key")
	agentConfigEditCmd.Flags().String("provider", "", "LLM provider: google, openai, or empty for auto-detect")
	agentConfigEditCmd.Flags().String("openai-api-key", "", "OpenAI-compatible API key")
	agentConfigEditCmd.Flags().String("openai-base-url", "", "OpenAI-compatible base URL")

	// Wallet balance flags
	walletBalanceCmd.Flags().String("api-url", "http://localhost:8424", "API server URL")

	// TUI flags — same as startCmd but all optional
	rootCmd.Flags().IntP("port", "p", 4001, "Port to listen on")
	rootCmd.Flags().StringSlice("bootstrap", []string{}, "Bootstrap peers")
	rootCmd.Flags().String("model", "gemini-2.5-flash", "ADK model name")
	rootCmd.Flags().StringP("name", "n", "", "Agent name (optional; registers agent on startup if set)")
	rootCmd.Flags().StringP("description", "d", "", "Agent description")
	rootCmd.Flags().Float64P("price", "r", 0, "Price per task")
	rootCmd.Flags().Bool("x402", false, "Support EIP-402 payments")
	rootCmd.Flags().Duration("announce-interval", 30*time.Second, "How often to republish agent CRDT listing")
	rootCmd.Flags().Int("api-port", 8424, "HTTP API server port")
	rootCmd.Flags().String("provider", "", "LLM provider: google, openai, or empty for auto-detect")
	rootCmd.Flags().String("openai-api-key", "", "OpenAI-compatible API key (overrides OPENAI_API_KEY)")
	rootCmd.Flags().String("openai-base-url", "", "OpenAI-compatible base URL, e.g. http://localhost:11434/v1/")

	// Add commands
	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(orderCmd)
	rootCmd.AddCommand(walletCmd)

	agentCmd.AddCommand(agentRegisterCmd)
	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentDiscoverCmd)
	agentCmd.AddCommand(agentExecuteCmd)
	agentCmd.AddCommand(agentServeCmd)
	agentCmd.AddCommand(agentConfigCmd)
	agentConfigCmd.AddCommand(agentConfigListCmd, agentConfigAddCmd, agentConfigDeleteCmd, agentConfigEditCmd)

	orderCmd.AddCommand(orderCreateCmd)
	walletCmd.AddCommand(walletBalanceCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// getOptionalFlag reads a flag if it's registered on the command, otherwise returns "".
func getOptionalFlag(cmd *cobra.Command, name string) string {
	if f := cmd.Flags().Lookup(name); f != nil {
		return f.Value.String()
	}
	return ""
}

func runNode(cmd *cobra.Command, args []string) error {
	if err := initRuntime(cmd); err != nil {
		return err
	}
	defer shutdownRuntime()

	fmt.Println("Marketplace Node Started")
	printRuntimeInfo()
	waitForShutdown()
	return nil
}

func serveAgent(cmd *cobra.Command, args []string) error {
	if err := initRuntime(cmd); err != nil {
		return err
	}
	defer shutdownRuntime()

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	price, _ := cmd.Flags().GetFloat64("price")
	model, _ := cmd.Flags().GetString("model")

	registered, err := agentManager.RegisterAgent(ctx, agent.AgentSpec{
		Name:          name,
		Description:   description,
		Price:         price,
		Model:         model,
		X402Support:   true,
		Services:      []types.Service{{Name: name, Version: "1.0.0"}},
		Provider:      getOptionalFlag(cmd, "provider"),
		OpenAIAPIKey:  getOptionalFlag(cmd, "openai-api-key"),
		OpenAIBaseURL: getOptionalFlag(cmd, "openai-base-url"),
	})
	if err != nil {
		return fmt.Errorf("failed to register serving agent: %w", err)
	}

	if listingService != nil {
		serveMsg := &types.AgentListingMessage{
			Type:      "list",
			AgentID:   registered.AgentID,
			Name:      registered.Name,
			Price:     registered.Price,
			Metadata:  registered.MetadataCID,
			SellerID:  p2pHost.ID().String(),
			Addrs:     p2pHost.AddrStrings(),
			Protocols: []string{p2p.X402ProtocolID},
			Timestamp: time.Now().Unix(),
		}
		listingService.UpsertLocalListing(serveMsg)
		if data, err := json.Marshal(serveMsg); err == nil {
			_ = p2pPubSub.Publish(ctx, marketplace.AnnounceTopic, data)
		}
	}

	// Auto-load additional agents from agents.yaml.
	announceInterval, _ := cmd.Flags().GetDuration("announce-interval")
	if announceInterval < 5*time.Second {
		announceInterval = 5 * time.Second
	}
	if err := loadAndRegisterAgentsFromConfig(ctx, announceInterval); err != nil {
		fmt.Printf("warning: %v\n", err)
	}

	fmt.Println("P2P Agent Serving")
	printRuntimeInfo()
	fmt.Printf("ID: %s\n", registered.ID)
	fmt.Printf("Agent ID: %s\n", registered.AgentID)
	fmt.Printf("Agent Name: %s\n", registered.Name)
	fmt.Printf("Metadata CID: %s\n", registered.MetadataCID)
	waitForShutdown()
	return nil
}

func runStart(cmd *cobra.Command, args []string) error {
	if err := initRuntime(cmd); err != nil {
		return err
	}
	defer shutdownRuntime()

	announceInterval, _ := cmd.Flags().GetDuration("announce-interval")
	if announceInterval < 5*time.Second {
		announceInterval = 5 * time.Second
	}

	// Register agent from CLI flags if --name is set.
	name, _ := cmd.Flags().GetString("name")
	if name != "" {
		registered, listingMsg, err := registerLocalAgentFromFlags(ctx, cmd)
		if err != nil {
			return err
		}
		if listingService != nil {
			listingService.UpsertLocalListing(listingMsg)
			if data, err := json.Marshal(listingMsg); err == nil {
				_ = p2pPubSub.Publish(ctx, marketplace.AnnounceTopic, data)
			}
		}
		go runListingAnnouncer(ctx, announceInterval, func(ts int64) *types.AgentListingMessage {
			msg := *listingMsg
			msg.Type = "update"
			msg.Timestamp = ts
			return &msg
		})
		fmt.Printf("Agent registered: %s (%s)\n", registered.Name, registered.AgentID)
	}

	// Auto-load agents from agents.yaml.
	if err := loadAndRegisterAgentsFromConfig(ctx, announceInterval); err != nil {
		fmt.Printf("warning: %v\n", err)
	}

	apiPort, _ := cmd.Flags().GetInt("api-port")
	apiServer = api.NewServer(apiPort, agentManager, listingService, orderService, p2pHost, paymentService, sessionStore)
	if err := apiServer.Start(); err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}
	fmt.Printf("HTTP API server running on port %d\n", apiPort)

	if paymentService != nil {
		go func() {
			ticker := time.NewTicker(2 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					paymentService.CleanupExpiredChallenges()
				}
			}
		}()
	}

	fmt.Println("Betar Started (single process)")
	printRuntimeInfo()
	fmt.Println("Marketplace mode: CRDT directory + direct stream RPC")
	waitForShutdown()
	return nil
}

func initRuntime(cmd *cobra.Command) error {
	var err error
	cfg, err = config.LoadConfig()
	if err != nil {
		return err
	}

	port, _ := cmd.Flags().GetInt("port")
	bootstrap, _ := cmd.Flags().GetStringSlice("bootstrap")
	modelName, _ := cmd.Flags().GetString("model")

	cfg.P2P.Port = port
	cfg.P2P.BootstrapPeers = bootstrap
	cfg.Agent.Model = modelName

	if provider := getOptionalFlag(cmd, "provider"); provider != "" {
		cfg.Agent.Provider = provider
	}
	if key := getOptionalFlag(cmd, "openai-api-key"); key != "" {
		cfg.Agent.OpenAIAPIKey = key
	}
	if url := getOptionalFlag(cmd, "openai-base-url"); url != "" {
		cfg.Agent.OpenAIBaseURL = url
	}

	ctx, cancel = context.WithCancel(context.Background())

	p2pHost, err = p2p.NewHost(ctx, cfg.P2P)
	if err != nil {
		return fmt.Errorf("failed to create P2P host: %w", err)
	}

	p2pPubSub, err = p2p.NewPubSub(ctx, p2pHost.RawHost())
	if err != nil {
		return fmt.Errorf("failed to create pubsub: %w", err)
	}

	streamHandler = p2p.NewStreamHandler(p2pHost.RawHost())
	x402StreamHandler = p2p.NewX402StreamHandler(p2pHost.RawHost())

	discovery, err = p2p.NewDiscovery(ctx, p2pHost.RawHost(), cfg.P2P)
	if err != nil {
		return fmt.Errorf("failed to create discovery service: %w", err)
	}
	if err := discovery.DiscoverPeers(ctx, cfg.P2P.BootstrapPeers); err != nil {
		fmt.Printf("warning: discovery bootstrap had errors: %v\n", err)
	}

	ipfsClient, err = ipfs.NewClient(ctx, p2pHost.RawHost(), discovery.Routing(), cfg.Storage.DataDir)
	if err != nil {
		return fmt.Errorf("failed to create embedded ipfs-lite node: %w", err)
	}

	// Create agent listing service first (needed for remote agent lookup)
	listingService, err = marketplace.NewAgentListingService(ctx, ipfsClient, p2pPubSub, p2pHost.ID())
	if err != nil {
		return fmt.Errorf("failed to create listing service: %w", err)
	}

	// Create payment service if Ethereum is configured
	var walletAddr string
	if cfg.Ethereum != nil && cfg.Ethereum.PrivateKey != "" {
		walletAddr, _ = config.GetAddressFromKey(cfg.Ethereum.PrivateKey)
		if cfg.Ethereum.RPCURL != "" {
			wallet, err := eth.NewWallet(cfg.Ethereum.PrivateKey, cfg.Ethereum.RPCURL)
			if err != nil {
				fmt.Printf("Warning: failed to create wallet: %v (payments disabled)\n", err)
				fmt.Printf("Wallet address: %s (RPC URL required for payments)\n", walletAddr)
			} else {
				fmt.Printf("Wallet address: %s\n", walletAddr)
				paymentService = marketplace.NewPaymentService(wallet, walletAddr)
				if paymentService == nil {
					fmt.Printf("Warning: failed to create payment service (payments disabled)\n")
				}
			}
		} else {
			fmt.Printf("Wallet address: %s (RPC URL required for payments)\n", walletAddr)
		}
	}

	sessionStore, err = session.NewStore(cfg.Storage.SessionsDir)
	if err != nil {
		fmt.Printf("warning: failed to open session store: %v (sessions disabled)\n", err)
		sessionStore = nil
	}

	agentManager, err = agent.NewManager(agent.ADKConfig{
		AppName:       "betar",
		ModelName:     cfg.Agent.Model,
		APIKey:        cfg.Agent.APIKey,
		Provider:      cfg.Agent.Provider,
		OpenAIAPIKey:  cfg.Agent.OpenAIAPIKey,
		OpenAIBaseURL: cfg.Agent.OpenAIBaseURL,
	}, ipfsClient, p2pHost, listingService, cfg.P2P.PrivKey, paymentService, walletAddr, sessionStore)
	if err != nil {
		return fmt.Errorf("failed to create agent manager: %w", err)
	}

	// Wire up the x402 stream handler so the agent manager can serve /x402/libp2p/1.0.0 requests.
	agentManager.RegisterX402Handlers(x402StreamHandler)

	// Register "info" handler on the basic marketplace stream handler.
	agentManager.RegisterStreamHandlers(streamHandler)

	// Start GossipSub listener for remote agent announcements.
	listingService.StartAnnouncementListener(ctx, p2pPubSub)

	// Pull listings from every newly-connected peer via the "info" stream.
	p2pHost.RawHost().Network().Notify(&network.NotifyBundle{
		ConnectedF: func(_ network.Network, c network.Conn) {
			go func() {
				resp, err := streamHandler.SendMessage(ctx, c.RemotePeer(), "info", nil)
				if err != nil {
					return
				}
				var listings []*types.AgentListing
				if json.Unmarshal(resp, &listings) != nil {
					return
				}
				for _, l := range listings {
					listingService.UpsertLocalListing(&types.AgentListingMessage{
						Type:      "list",
						AgentID:   l.ID,
						Name:      l.Name,
						Price:     l.Price,
						Metadata:  l.Metadata,
						SellerID:  l.SellerID,
						Addrs:     l.Addrs,
						Protocols: l.Protocols,
						Timestamp: l.Timestamp,
					})
				}
			}()
		},
	})

	orderService = marketplace.NewOrderService(streamHandler, p2pHost, p2pHost.ID())

	return nil
}

func registerLocalAgentFromFlags(ctx context.Context, cmd *cobra.Command) (*agent.LocalAgent, *types.AgentListingMessage, error) {
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	price, _ := cmd.Flags().GetFloat64("price")
	model, _ := cmd.Flags().GetString("model")

	registered, err := agentManager.RegisterAgent(ctx, agent.AgentSpec{
		Name:          name,
		Description:   description,
		Price:         price,
		Model:         model,
		X402Support:   true,
		Services:      []types.Service{{Name: name, Version: "1.0.0"}},
		Provider:      getOptionalFlag(cmd, "provider"),
		OpenAIAPIKey:  getOptionalFlag(cmd, "openai-api-key"),
		OpenAIBaseURL: getOptionalFlag(cmd, "openai-base-url"),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to register serving agent: %w", err)
	}

	protocols := []string{p2p.X402ProtocolID}

	listing := &types.AgentListingMessage{
		Type:      "list",
		AgentID:   registered.AgentID,
		Name:      registered.Name,
		Price:     registered.Price,
		Metadata:  registered.MetadataCID,
		SellerID:  p2pHost.ID().String(),
		Addrs:     p2pHost.AddrStrings(),
		Protocols: protocols,
		Timestamp: time.Now().Unix(),
	}

	return registered, listing, nil
}

func runListingAnnouncer(ctx context.Context, interval time.Duration, next func(int64) *types.AgentListingMessage) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			if listingService == nil {
				continue
			}
			msg := next(t.Unix())
			if err := listingService.UpdateListing(ctx, msg); err != nil && !errors.Is(err, context.Canceled) {
				fmt.Printf("warning: failed to republish listing: %v\n", err)
			}
			if data, err := json.Marshal(msg); err == nil {
				_ = p2pPubSub.Publish(ctx, marketplace.AnnounceTopic, data)
			}
		}
	}
}

// deriveWalletAddress derives the Ethereum address hex from a private key hex string.
// Returns empty string if the key is not set or invalid.
func deriveWalletAddress(privKeyHex string) string {
	if privKeyHex == "" {
		return ""
	}
	pk, err := crypto.HexToECDSA(privKeyHex)
	if err != nil {
		return ""
	}
	pub, ok := pk.Public().(*ecdsa.PublicKey)
	if !ok {
		return ""
	}
	return crypto.PubkeyToAddress(*pub).Hex()
}

func printRuntimeInfo() {
	fmt.Printf("Peer ID: %s\n", p2pHost.ID())
	fmt.Printf("Addresses: %v\n", p2pHost.AddrStrings())
	fmt.Printf("IPFS: embedded ipfs-lite (%s/ipfslite)\n", cfg.Storage.DataDir)
	fmt.Printf("ADK Model: %s\n", cfg.Agent.Model)
	fmt.Printf("Identity Key: %s\n", cfg.Storage.P2PKeyPath)
}

func waitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	fmt.Println("\nShutting down...")
}

func shutdownRuntime() {
	if cancel != nil {
		cancel()
	}
	if discovery != nil {
		_ = discovery.Close()
	}
	if p2pPubSub != nil {
		_ = p2pPubSub.Close()
	}
	if listingService != nil {
		_ = listingService.Close()
	}
	if ipfsClient != nil {
		_ = ipfsClient.Close()
	}
	if apiServer != nil {
		_ = apiServer.Stop(context.Background())
	}
	if p2pHost != nil {
		_ = p2pHost.Close()
	}
	if sessionStore != nil {
		_ = sessionStore.Close()
	}
}

func registerAgent(cmd *cobra.Command, args []string) error {
	if p2pHost == nil || agentManager == nil {
		return fmt.Errorf("node not running in this process. use 'betar start' or 'betar agent serve'")
	}

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	price, _ := cmd.Flags().GetFloat64("price")

	spec := agent.AgentSpec{
		Name:        name,
		Description: description,
		Price:       price,
		X402Support: true,
		Services:    []types.Service{{Name: name, Version: "1.0.0"}},
	}

	agent, err := agentManager.RegisterAgent(ctx, spec)
	if err != nil {
		return fmt.Errorf("failed to register agent: %w", err)
	}

	fmt.Println("Agent registered successfully")
	fmt.Printf("Agent ID: %s\n", agent.AgentID)
	fmt.Printf("Name: %s\n", agent.Name)
	fmt.Printf("Price: %f ETH\n", agent.Price)

	return nil
}

func listAgents(cmd *cobra.Command, args []string) error {
	apiURL, _ := cmd.Flags().GetString("api-url")
	client := api.NewClient(apiURL)

	var agents []*agent.LocalAgent
	if err := client.Get("/agents/local", &agents); err != nil {
		return err
	}

	if len(agents) == 0 {
		fmt.Println("No agents registered")
		return nil
	}

	fmt.Println("Local Agents:")
	for _, a := range agents {
		fmt.Printf("  - %s (%s) - %f ETH\n", a.Name, a.ID, a.Price)
	}

	return nil
}

func discoverAgents(cmd *cobra.Command, args []string) error {
	apiURL, _ := cmd.Flags().GetString("api-url")
	client := api.NewClient(apiURL)

	var listings []*types.AgentListing
	if err := client.Get("/agents", &listings); err != nil {
		return err
	}

	if len(listings) == 0 {
		fmt.Println("No agents discovered")
		return nil
	}

	fmt.Println("Discovered Agents:")
	for _, l := range listings {
		fmt.Printf("  - %s (%s) - %f ETH\n", l.Name, l.ID, l.Price)
	}

	return nil
}

func executeAgent(cmd *cobra.Command, args []string) error {
	agentID, _ := cmd.Flags().GetString("agent-id")
	task, _ := cmd.Flags().GetString("task")

	if agentID == "" || task == "" {
		return fmt.Errorf("agent-id and task are required")
	}

	apiURL, _ := cmd.Flags().GetString("api-url")
	client := api.NewClient(apiURL)

	var resp struct {
		Output string `json:"output"`
	}
	if err := client.Post(fmt.Sprintf("/agents/%s/execute", agentID), map[string]string{"input": task}, &resp); err != nil {
		return err
	}

	fmt.Println("Task output:")
	fmt.Println(resp.Output)
	return nil
}

func createOrder(cmd *cobra.Command, args []string) error {
	apiURL, _ := cmd.Flags().GetString("api-url")
	client := api.NewClient(apiURL)

	agentID, _ := cmd.Flags().GetString("agent-id")
	price, _ := cmd.Flags().GetFloat64("price")

	if agentID == "" {
		return fmt.Errorf("agent-id is required")
	}

	var order types.Order
	if err := client.Post("/orders", map[string]interface{}{"agentId": agentID, "price": price}, &order); err != nil {
		return err
	}

	fmt.Println("Order created")
	fmt.Printf("Order ID: %s\n", order.ID)
	fmt.Printf("Agent ID: %s\n", order.AgentID)
	fmt.Printf("Price: %f ETH\n", order.Price)
	if order.SellerID != "" {
		fmt.Printf("Seller ID: %s\n", order.SellerID)
	}
	if strings.TrimSpace(order.BuyerID) != "" {
		fmt.Printf("Buyer ID: %s\n", order.BuyerID)
	}

	return nil
}

// loadAndRegisterAgentsFromConfig reads agents.yaml and registers each profile.
func loadAndRegisterAgentsFromConfig(ctx context.Context, announceInterval time.Duration) error {
	agentsCfg, err := config.LoadAgentsConfig(cfg.Storage.DataDir)
	if err != nil {
		return fmt.Errorf("loading agents config: %w", err)
	}
	if len(agentsCfg.Agents) == 0 {
		return nil
	}

	for _, profile := range agentsCfg.Agents {
		registered, err := agentManager.RegisterAgent(ctx, agent.AgentSpec{
			Name:          profile.Name,
			Description:   profile.Description,
			Price:         profile.Price,
			Model:         profile.Model,
			APIKey:        profile.APIKey,
			X402Support:   true,
			Services:      []types.Service{{Name: profile.Name, Version: "1.0.0"}},
			Provider:      profile.Provider,
			OpenAIAPIKey:  profile.OpenAIAPIKey,
			OpenAIBaseURL: profile.OpenAIBaseURL,
		})
		if err != nil {
			fmt.Printf("warning: failed to register agent %q from config: %v\n", profile.Name, err)
			continue
		}

		if listingService != nil {
			msg := &types.AgentListingMessage{
				Type:      "list",
				AgentID:   registered.AgentID,
				Name:      registered.Name,
				Price:     registered.Price,
				Metadata:  registered.MetadataCID,
				SellerID:  p2pHost.ID().String(),
				Addrs:     p2pHost.AddrStrings(),
				Protocols: []string{p2p.X402ProtocolID},
				Timestamp: time.Now().Unix(),
			}
			listingService.UpsertLocalListing(msg)
			if data, err := json.Marshal(msg); err == nil {
				_ = p2pPubSub.Publish(ctx, marketplace.AnnounceTopic, data)
			}
			go runListingAnnouncer(ctx, announceInterval, func(ts int64) *types.AgentListingMessage {
				m := *msg
				m.Type = "update"
				m.Timestamp = ts
				return &m
			})
		}

		fmt.Printf("Agent loaded from config: %s (%s)\n", registered.Name, registered.AgentID)
	}
	return nil
}

// agentConfigList prints all configured agent profiles.
func agentConfigList(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	agentsCfg, err := config.LoadAgentsConfig(cfg.Storage.DataDir)
	if err != nil {
		return err
	}
	if len(agentsCfg.Agents) == 0 {
		fmt.Println("No agent profiles configured.")
		return nil
	}
	fmt.Printf("%-20s %-40s %10s  %-20s  %-10s\n", "NAME", "DESCRIPTION", "PRICE", "MODEL", "PROVIDER")
	for _, p := range agentsCfg.Agents {
		model := p.Model
		if model == "" {
			model = "(global)"
		}
		provider := p.Provider
		if provider == "" {
			provider = "(auto)"
		}
		fmt.Printf("%-20s %-40s %10.6f  %-20s  %-10s\n", p.Name, p.Description, p.Price, model, provider)
	}
	return nil
}

// agentConfigAdd adds a new agent profile to agents.yaml.
func agentConfigAdd(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	agentsCfg, err := config.LoadAgentsConfig(cfg.Storage.DataDir)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	price, _ := cmd.Flags().GetFloat64("price")
	model, _ := cmd.Flags().GetString("model")
	apiKey, _ := cmd.Flags().GetString("api-key")
	provider, _ := cmd.Flags().GetString("provider")
	openAIAPIKey, _ := cmd.Flags().GetString("openai-api-key")
	openAIBaseURL, _ := cmd.Flags().GetString("openai-base-url")

	profile := config.AgentProfile{
		Name:          name,
		Description:   description,
		Price:         price,
		Model:         model,
		APIKey:        apiKey,
		Provider:      provider,
		OpenAIAPIKey:  openAIAPIKey,
		OpenAIBaseURL: openAIBaseURL,
	}

	if err := agentsCfg.AddProfile(profile); err != nil {
		return err
	}
	if err := config.SaveAgentsConfig(cfg.Storage.DataDir, agentsCfg); err != nil {
		return err
	}
	fmt.Printf("Agent profile %q added.\n", name)
	return nil
}

// agentConfigDelete removes an agent profile from agents.yaml.
func agentConfigDelete(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	agentsCfg, err := config.LoadAgentsConfig(cfg.Storage.DataDir)
	if err != nil {
		return err
	}
	name := args[0]
	if err := agentsCfg.DeleteProfile(name); err != nil {
		return err
	}
	if err := config.SaveAgentsConfig(cfg.Storage.DataDir, agentsCfg); err != nil {
		return err
	}
	fmt.Printf("Agent profile %q deleted.\n", name)
	return nil
}

// agentConfigEdit updates fields of an existing agent profile in agents.yaml.
func agentConfigEdit(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	agentsCfg, err := config.LoadAgentsConfig(cfg.Storage.DataDir)
	if err != nil {
		return err
	}
	name := args[0]
	p := agentsCfg.FindProfile(name)
	if p == nil {
		return fmt.Errorf("agent profile %q not found", name)
	}
	if cmd.Flags().Changed("description") {
		p.Description, _ = cmd.Flags().GetString("description")
	}
	if cmd.Flags().Changed("price") {
		p.Price, _ = cmd.Flags().GetFloat64("price")
	}
	if cmd.Flags().Changed("model") {
		p.Model, _ = cmd.Flags().GetString("model")
	}
	if cmd.Flags().Changed("api-key") {
		p.APIKey, _ = cmd.Flags().GetString("api-key")
	}
	if cmd.Flags().Changed("provider") {
		p.Provider, _ = cmd.Flags().GetString("provider")
	}
	if cmd.Flags().Changed("openai-api-key") {
		p.OpenAIAPIKey, _ = cmd.Flags().GetString("openai-api-key")
	}
	if cmd.Flags().Changed("openai-base-url") {
		p.OpenAIBaseURL, _ = cmd.Flags().GetString("openai-base-url")
	}
	if err := config.SaveAgentsConfig(cfg.Storage.DataDir, agentsCfg); err != nil {
		return err
	}
	fmt.Printf("Agent profile %q updated.\n", name)
	return nil
}

func checkBalance(cmd *cobra.Command, args []string) error {
	apiURL, _ := cmd.Flags().GetString("api-url")
	client := api.NewClient(apiURL)

	var resp struct {
		Address string  `json:"address"`
		Balance float64 `json:"balance"`
	}
	if err := client.Get("/wallet/balance", &resp); err != nil {
		return err
	}

	fmt.Printf("Address: %s\n", resp.Address)
	fmt.Printf("Balance: %f ETH\n", resp.Balance)

	return nil
}
