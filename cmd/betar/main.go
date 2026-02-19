package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/asabya/betar/cmd/betar/api"
	"github.com/asabya/betar/internal/agent"
	"github.com/asabya/betar/internal/config"
	"github.com/asabya/betar/internal/eth"
	"github.com/asabya/betar/internal/ipfs"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/pkg/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
)

var (
	cfg            *config.Config
	ctx            context.Context
	cancel         context.CancelFunc
	p2pHost        *p2p.Host
	p2pPubSub      *p2p.PubSub
	streamHandler  *p2p.StreamHandler
	discovery      *p2p.Discovery
	agentManager   *agent.Manager
	listingService *marketplace.AgentListingService
	orderService   *marketplace.OrderService
	paymentService *marketplace.PaymentService
	ipfsClient     *ipfs.Client
	apiServer      *api.Server
)

var rootCmd = &cobra.Command{
	Use:   "betar",
	Short: "P2P Agent 2 Agent Marketplace",
	Long:  "A decentralized marketplace where AI agents can discover, list, and transact with each other",
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

	// Agent serve flags
	agentServeCmd.Flags().IntP("port", "p", 4001, "Port to listen on")
	agentServeCmd.Flags().StringSlice("bootstrap", []string{}, "Bootstrap peers")
	agentServeCmd.Flags().String("model", "gemini-2.5-flash", "ADK model name")
	agentServeCmd.Flags().StringP("name", "n", "", "Agent name")
	agentServeCmd.Flags().StringP("description", "d", "", "Agent description")
	agentServeCmd.Flags().Float64P("price", "r", 0, "Price per task")
	agentServeCmd.Flags().String("endpoint", "p2p://local", "Agent endpoint")
	agentServeCmd.Flags().String("framework", "adk", "Agent framework")
	agentServeCmd.Flags().Bool("x402", false, "Support EIP-402 payments")
	_ = agentServeCmd.MarkFlagRequired("name")

	// Unified start flags
	startCmd.Flags().IntP("port", "p", 4001, "Port to listen on")
	startCmd.Flags().StringSlice("bootstrap", []string{}, "Bootstrap peers")
	startCmd.Flags().String("model", "gemini-2.5-flash", "ADK model name")
	startCmd.Flags().StringP("name", "n", "", "Agent name")
	startCmd.Flags().StringP("description", "d", "", "Agent description")
	startCmd.Flags().Float64P("price", "r", 0, "Price per task")
	startCmd.Flags().String("endpoint", "p2p://local", "Agent endpoint")
	startCmd.Flags().String("framework", "adk", "Agent framework")
	startCmd.Flags().Bool("x402", false, "Support EIP-402 payments")
	startCmd.Flags().Duration("announce-interval", 30*time.Second, "How often to republish agent CRDT listing")
	startCmd.Flags().Int("api-port", 8424, "HTTP API server port")
	_ = startCmd.MarkFlagRequired("name")

	// Agent register flags
	agentRegisterCmd.Flags().StringP("name", "n", "", "Agent name")
	agentRegisterCmd.Flags().StringP("description", "d", "", "Agent description")
	agentRegisterCmd.Flags().Float64P("price", "p", 0, "Price per task")
	agentRegisterCmd.Flags().String("endpoint", "", "Agent endpoint")
	agentRegisterCmd.Flags().Bool("x402", false, "Support EIP-402 payments")

	// Agent list flags
	agentListCmd.Flags().String("api-url", "http://localhost:8424", "API server URL")

	// Agent discover flags
	agentDiscoverCmd.Flags().String("api-url", "http://localhost:8424", "API server URL")

	// Agent execute flags
	agentExecuteCmd.Flags().String("api-url", "http://localhost:8424", "API server URL")
	agentExecuteCmd.Flags().String("agent-id", "", "Agent ID")
	agentExecuteCmd.Flags().StringP("task", "t", "", "Task input")
	agentExecuteCmd.Flags().Float64("payment", 0, "Payment amount")

	// Order create flags
	orderCreateCmd.Flags().String("api-url", "http://localhost:8424", "API server URL")
	orderCreateCmd.Flags().String("agent-id", "", "Agent ID")
	orderCreateCmd.Flags().Float64("price", 0, "Price")

	// Wallet balance flags
	walletBalanceCmd.Flags().String("api-url", "http://localhost:8424", "API server URL")

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

	orderCmd.AddCommand(orderCreateCmd)
	walletCmd.AddCommand(walletBalanceCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
	endpoint, _ := cmd.Flags().GetString("endpoint")
	framework, _ := cmd.Flags().GetString("framework")
	model, _ := cmd.Flags().GetString("model")
	x402, _ := cmd.Flags().GetBool("x402")

	registered, err := agentManager.RegisterAgent(ctx, agent.AgentSpec{
		Name:        name,
		Description: description,
		Endpoint:    endpoint,
		Price:       price,
		Framework:   framework,
		Model:       model,
		X402Support: x402,
		Services: []types.Service{{
			Name:     "p2p",
			Endpoint: endpoint,
		}},
	})
	if err != nil {
		return fmt.Errorf("failed to register serving agent: %w", err)
	}

	if listingService != nil {
		listingService.UpsertLocalListing(&types.AgentListingMessage{
			Type:      "list",
			AgentID:   registered.ID,
			Name:      registered.Name,
			Price:     registered.Price,
			Metadata:  registered.MetadataCID,
			SellerID:  p2pHost.ID().String(),
			Addrs:     p2pHost.AddrStrings(),
			Protocols: []string{string(p2p.ProtocolID)},
			Timestamp: time.Now().Unix(),
		})
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

	ipfsReadyCID, err := publishNodePresence(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize IPFS presence: %w", err)
	}

	registered, listingMsg, err := registerLocalAgentFromFlags(ctx, cmd)
	if err != nil {
		return err
	}

	if listingService != nil {
		listingService.UpsertLocalListing(listingMsg)
	}

	apiPort, _ := cmd.Flags().GetInt("api-port")
	apiServer = api.NewServer(apiPort, agentManager, listingService, orderService, p2pHost, paymentService)
	if err := apiServer.Start(); err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}
	fmt.Printf("HTTP API server running on port %d\n", apiPort)

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

	fmt.Println("Betar Started (single process)")
	printRuntimeInfo()
	fmt.Printf("Agent ID: %s\n", registered.ID)
	fmt.Printf("Agent Name: %s\n", registered.Name)
	fmt.Printf("Metadata CID: %s\n", registered.MetadataCID)
	fmt.Printf("Node Presence CID: %s\n", ipfsReadyCID)
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

	agentManager, err = agent.NewManager(agent.ADKConfig{
		AppName:   "betar",
		ModelName: cfg.Agent.Model,
		APIKey:    cfg.Agent.APIKey,
	}, ipfsClient, p2pHost, streamHandler, listingService, cfg.P2P.PrivKey, paymentService, walletAddr)
	if err != nil {
		return fmt.Errorf("failed to create agent manager: %w", err)
	}

	orderService = marketplace.NewOrderService(streamHandler, p2pHost, p2pHost.ID())

	return nil
}

func publishNodePresence(ctx context.Context) (string, error) {
	if ipfsClient == nil || p2pHost == nil {
		return "", fmt.Errorf("runtime not initialized")
	}

	presence := map[string]interface{}{
		"kind":      "node-presence",
		"peerId":    p2pHost.ID().String(),
		"addresses": p2pHost.AddrStrings(),
		"timestamp": time.Now().Unix(),
	}

	cid, err := ipfsClient.AddJSON(ctx, presence)
	if err != nil {
		return "", err
	}
	if err := ipfsClient.Pin(ctx, cid); err != nil {
		return "", err
	}

	return cid, nil
}

func registerLocalAgentFromFlags(ctx context.Context, cmd *cobra.Command) (*agent.LocalAgent, *types.AgentListingMessage, error) {
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	price, _ := cmd.Flags().GetFloat64("price")
	endpoint, _ := cmd.Flags().GetString("endpoint")
	framework, _ := cmd.Flags().GetString("framework")
	model, _ := cmd.Flags().GetString("model")
	x402, _ := cmd.Flags().GetBool("x402")

	registered, err := agentManager.RegisterAgent(ctx, agent.AgentSpec{
		Name:        name,
		Description: description,
		Endpoint:    endpoint,
		Price:       price,
		Framework:   framework,
		Model:       model,
		X402Support: x402,
		Services: []types.Service{{
			Name:     "p2p",
			Endpoint: endpoint,
		}},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to register serving agent: %w", err)
	}

	listing := &types.AgentListingMessage{
		Type:      "list",
		AgentID:   registered.ID,
		Name:      registered.Name,
		Price:     registered.Price,
		Metadata:  registered.MetadataCID,
		SellerID:  p2pHost.ID().String(),
		Addrs:     p2pHost.AddrStrings(),
		Protocols: []string{string(p2p.ProtocolID)},
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
		}
	}
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
}

func registerAgent(cmd *cobra.Command, args []string) error {
	if p2pHost == nil || agentManager == nil {
		return fmt.Errorf("node not running in this process. use 'betar start' or 'betar agent serve'")
	}

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")
	price, _ := cmd.Flags().GetFloat64("price")
	endpoint, _ := cmd.Flags().GetString("endpoint")
	x402, _ := cmd.Flags().GetBool("x402")

	spec := agent.AgentSpec{
		Name:        name,
		Description: description,
		Price:       price,
		Endpoint:    endpoint,
		X402Support: x402,
		Services: []types.Service{
			{Name: "default", Endpoint: endpoint},
		},
	}

	agent, err := agentManager.RegisterAgent(ctx, spec)
	if err != nil {
		return fmt.Errorf("failed to register agent: %w", err)
	}

	fmt.Println("Agent registered successfully")
	fmt.Printf("Agent ID: %s\n", agent.ID)
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
	apiURL, _ := cmd.Flags().GetString("api-url")
	client := api.NewClient(apiURL)

	agentID, _ := cmd.Flags().GetString("agent-id")
	task, _ := cmd.Flags().GetString("task")

	if agentID == "" || task == "" {
		return fmt.Errorf("agent-id and task are required")
	}

	// First attempt - without payment
	var resp struct {
		Output string `json:"output"`
		Error  string `json:"error"`
	}

	payErr, err := client.PostWithPayment(fmt.Sprintf("/agents/%s/execute", agentID), map[string]string{"input": task}, &resp)
	// Check payment required first (even if err is nil, since 402 returns err=nil but payErr non-nil)
	if payErr != nil && payErr.RequiresPayment {
		fmt.Printf("\n=== Payment Required ===\n")
		fmt.Printf("Agent: %s\n", payErr.AgentID)
		fmt.Printf("Message: %s\n", payErr.Message)

		var displayedAtomicAmount string
		if payReq, ok := payErr.PaymentRequirement.(map[string]interface{}); ok {
			if atomicAmt, ok := payReq["amount"].(string); ok && atomicAmt != "" {
				displayedAtomicAmount = atomicAmt
				fmt.Printf("Amount: %s\n", marketplace.FormatUSDC(atomicAmt))
			} else {
				fmt.Printf("Amount: (unable to parse)\n")
			}
			if asset, ok := payReq["extra"].(map[string]interface{}); ok {
				fmt.Printf("Asset: %s\n", asset["name"])
			}
		}

		fmt.Printf("\nDo you want to proceed with payment? (y/N): ")

		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Payment cancelled by user")
			return nil
		}

		// Sign payment and retry
		fmt.Println("\nSigning payment...")

		// Convert to marketplace.PaymentRequirements
		var paymentReq marketplace.PaymentRequirements
		if reqBytes, err := json.Marshal(payErr.PaymentRequirement); err == nil {
			json.Unmarshal(reqBytes, &paymentReq)
		}

		// Call local API to sign payment (using the same client since we need local wallet)
		var paymentHeader marketplace.PaymentHeader
		if signErr := client.Post("/payment/sign", map[string]interface{}{
			"paymentRequirement": paymentReq,
		}, &paymentHeader); signErr != nil {
			return fmt.Errorf("failed to sign payment: %w", signErr)
		}

		// Validate that signed amount matches what was displayed to user
		if displayedAtomicAmount != "" && paymentHeader.Requirement.Amount != displayedAtomicAmount {
			return fmt.Errorf("SECURITY WARNING: signed amount (%s) does not match displayed amount (%s). Aborting.",
				paymentHeader.Requirement.Amount, displayedAtomicAmount)
		}

		fmt.Printf("Payment signed. PaymentID: %s, Amount: %s\n", paymentHeader.PaymentID, marketplace.FormatUSDC(paymentHeader.Requirement.Amount))

		// Retry with payment (seller will settle via facilitator)
		var respWithPayment struct {
			Output    string `json:"output"`
			Error     string `json:"error"`
			PaymentID string `json:"paymentId"`
		}
		if err := client.Post(fmt.Sprintf("/agents/%s/execute", agentID), map[string]interface{}{
			"input":         task,
			"paymentHeader": paymentHeader,
		}, &respWithPayment); err != nil {
			return err
		}

		if respWithPayment.Error != "" {
			return fmt.Errorf("execution error: %s", respWithPayment.Error)
		}

		fmt.Printf("Payment ID: %s\n", respWithPayment.PaymentID)
		fmt.Println("Task output:")
		fmt.Println(respWithPayment.Output)
		return nil
	} else if err != nil {
		return err
	}

	if resp.Error != "" {
		return fmt.Errorf("execution error: %s", resp.Error)
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

func findListingByAgentID(agentID string) (*types.AgentListing, bool) {
	if listingService == nil {
		return nil, false
	}

	if l, ok := listingService.GetListing(agentID); ok {
		return l, true
	}

	for _, l := range listingService.ListListings() {
		if l != nil && l.ID == agentID {
			return l, true
		}
	}

	return nil, false
}

func extractRuntimeAgentID(listing *types.AgentListing) string {
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

func connectToListingPeer(ctx context.Context, listing *types.AgentListing) error {
	if listing == nil {
		return fmt.Errorf("listing is nil")
	}
	if listing.SellerID == "" {
		return fmt.Errorf("listing seller ID is empty")
	}

	sellerPeerID, err := peer.Decode(listing.SellerID)
	if err != nil {
		return fmt.Errorf("invalid seller peer ID %q: %w", listing.SellerID, err)
	}

	if len(listing.Addrs) == 0 {
		return agentManager.ConnectToAgent(ctx, sellerPeerID)
	}

	addrInfos := make([]peer.AddrInfo, 0, len(listing.Addrs))
	for _, rawAddr := range listing.Addrs {
		addr, err := multiaddr.NewMultiaddr(rawAddr)
		if err != nil {
			continue
		}
		addrInfos = append(addrInfos, peer.AddrInfo{ID: sellerPeerID, Addrs: []multiaddr.Multiaddr{addr}})
	}

	if len(addrInfos) == 0 {
		return agentManager.ConnectToAgent(ctx, sellerPeerID)
	}

	var connectErr error
	for _, info := range addrInfos {
		if err := p2pHost.Connect(ctx, info); err == nil {
			return nil
		} else {
			connectErr = err
		}
	}

	if connectErr != nil {
		return connectErr
	}

	return agentManager.ConnectToAgent(ctx, sellerPeerID)
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
