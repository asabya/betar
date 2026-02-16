# P2P Agent 2 Agent Marketplace - Implementation Plan

## Context

Building a decentralized P2P marketplace where AI agents can discover, list, and transact with each other. The marketplace will use:
- **libp2p** for P2P networking and peer discovery
- **Google ADK for Go (`google.golang.org/adk@v0.4.0`)** for native Go agent creation and execution
- **EIP-8004** for agent registration and discovery (with on-chain contracts)
- **EIP-402** for payments
- **Embedded IPFS-lite** for storing agent metadata off-chain
- **`go-ds-crdt`** for replicated off-chain marketplace listing state

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Marketplace Node                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │   libp2p     │  │   IPFS       │  │   ADK Agent Runtime  │  │
│  │   Host       │  │   Client     │  │   (in-process Go)    │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  Ethereum    │  │  EIP-8004    │  │   EIP-402            │  │
│  │  Client      │  │  Registry    │  │   Payments           │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│           │               │                    │                │
│           └───────────────┴────────────────────┘                │
│                          │                                       │
│                   ┌──────┴──────┐                                │
│                   │  Commands   │                                │
│                   │  & Queries  │                                │
│                   └─────────────┘                                │
└─────────────────────────────────────────────────────────────────┘
```

## Project Structure

```
betar/
├── cmd/
│   └── betar/
│       └── main.go              # Entry point & CLI
├── contracts/
│   ├── src/
│   │   ├── AgentRegistry.sol   # EIP-8004 Identity Registry (ERC-721)
│   │   ├── ReputationRegistry.sol  # EIP-8004 Feedback
│   │   ├── ValidationRegistry.sol   # EIP-8004 Validation
│   │   └── x402/
│   │       └── PaymentVault.sol     # EIP-402 Payments
│   ├── script/
│   │   └── Deploy.s.sol        # Deployment scripts
│   └── test/
│       └── *.t.sol             # Contract tests
├── internal/
│   ├── p2p/
│   │   ├── host.go             # libp2p host setup
│   │   ├── discovery.go        # Peer discovery (mDNS, DHT)
│   │   ├── pubsub.go           # GossipSub for CRDT internals
│   │   └── stream.go           # Direct stream handling
│   ├── ipfs/
│   │   └── client.go           # Embedded IPFS-lite client
│   ├── agent/
│   │   ├── manager.go          # Agent lifecycle management
│   │   ├── adk.go              # ADK agent construction/runtime
│   │   └── worker.go           # Legacy pubsub task worker (deprecated)
│   ├── marketplace/
│   │   ├── agent.go            # Agent listing/registration
│   │   ├── crdt.go             # go-ds-crdt listing state
│   │   ├── order.go            # Order management
│   │   ├── payment.go          # EIP-402 payment handling
│   │   └── protocol.go         # Marketplace protocols
│   ├── eip8004/
│   │   ├── registration.go     # On-chain registration
│   │   ├── metadata.go         # Metadata schemas
│   │   └── client.go           # Ethereum/ERC-20/ERC-721 interactions
│   └── eth/
│       └── wallet.go           # Ethereum wallet integration
├── api/
│   └── grpc/
│       └── marketplace.proto   # gRPC API definitions
├── pkg/
│   └── types/
│       └── types.go            # Shared type definitions
├── go.mod
└── go.sum
```

## Implementation Phases

### Phase 1: Core P2P Infrastructure

**Files to create:**
- `internal/p2p/host.go` - Create libp2p host with TCP/QUIC transports
- `internal/p2p/discovery.go` - mDNS (local) + DHT (network) discovery
- `internal/p2p/pubsub.go` - GossipSub transport for CRDT replication
- `internal/p2p/stream.go` - Direct peer-to-peer streams

**Key functions:**
```go
// internal/p2p/host.go
func NewHost(ctx context.Context, port int, bootstrapPeers []string) (*Host, error)
func (h *Host) ID() peer.ID
func (h *Host) Addrs() []multiaddr.Multiaddr
func (h *Host) Connect(ctx context.Context, pi peer.AddrInfo) error

// internal/p2p/pubsub.go
func NewPubSub(ctx context.Context, h host.Host) (*pubsub.PubSub, error)
func (ps *PubSub) JoinTopic(topic string) (*pubsub.Topic, error)
func (ps *PubSub) Subscribe(topic string) (*pubsub.Subscription, error)
```

### Phase 2: IPFS Integration

**Files to create:**
- `internal/ipfs/client.go` - Embedded IPFS-lite client

**Key functions:**
```go
// internal/ipfs/client.go
func NewClient(ctx context.Context, h host.Host, router routing.Routing, dataDir string) (*Client, error)
func (c *Client) Add(ctx context.Context, data []byte) (string, error)  // Returns CID
func (c *Client) Get(ctx context.Context, cid string) ([]byte, error)
func (c *Client) Pin(ctx context.Context, cid string) error
```

### Phase 3: EIP-8004 Smart Contracts

**Files to create (Solidity):**
- `contracts/src/AgentRegistry.sol` - ERC-721 based agent identity
- `contracts/src/ReputationRegistry.sol` - Feedback system
- `contracts/src/ValidationRegistry.sol` - Agent validation
- `contracts/src/x402/PaymentVault.sol` - EIP-402 payment handling

**Go bindings:**
- `internal/eip8004/client.go` - Contract interactions via go-ethereum
- `internal/eip8004/metadata.go` - Registration metadata schema

**EIP-8004 Registration Schema:**
```go
type Registration struct {
    Type        string    `json:"type"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Image       string    `json:"image,omitempty"`
    Services    []Service `json:"services"`
    X402Support bool      `json:"x402Support"`
    Active      bool      `json:"active"`
}

type Service struct {
    Name     string `json:"name"`
    Endpoint string `json:"endpoint"`
    Version  string `json:"version,omitempty"`
}
```

### Phase 4: Agent Runtime (Google ADK Go Integration)

**Files to create:**
- `internal/agent/adk.go` - ADK agent builder and runner wiring
- `internal/agent/worker.go` - legacy pubsub task execution loop (no longer default)
- `internal/agent/manager.go` - Agent lifecycle management

**Key insight:** use Go-native ADK (`google.golang.org/adk@v0.4.0`) in-process. Agents are invoked via direct libp2p stream RPC after seller discovery from CRDT listings.

**Token efficiency strategy:** keep pubsub task envelopes compact, move large context to IPFS (CID references), enforce per-task token budgets, and avoid replaying full chat history on each run.

```go
// internal/agent/adk.go
type ADKRuntime struct {
    agent   *llmagent.Agent
    runner  *runner.Runner
}

func NewADKRuntime(ctx context.Context, cfg Config) (*ADKRuntime, error)
func (r *ADKRuntime) RunTask(ctx context.Context, req TaskRequest) (*TaskResult, error)

// internal/agent/manager.go
type AgentManager struct {
    runtime  *ADKRuntime
    ipfs     *ipfs.Client
    host     *p2p.Host
}

func (m *AgentManager) RegisterAgent(ctx context.Context, spec AgentSpec) (*RegisteredAgent, error)
func (m *AgentManager) DiscoverAgents(ctx context.Context) ([]AgentListing, error)
func (m *AgentManager) ExecuteTask(ctx context.Context, agentID string, task string) (string, error)

// internal/agent/worker.go
func (w *Worker) Start(ctx context.Context) error
func (w *Worker) HandleMessage(ctx context.Context, msg *pubsub.Message) error
```

### Phase 5: Marketplace Protocol & Payments

**Files to create:**
- `internal/marketplace/agent.go` - Agent listing and discovery
- `internal/marketplace/crdt.go` - CRDT-backed listing datastore (`go-ds-crdt`)
- `internal/marketplace/order.go` - Order management
- `internal/marketplace/payment.go` - EIP-402 payment integration
- `internal/marketplace/protocol.go` - P2P marketplace protocols

**Recent design update:** agent listings are replicated through `go-ds-crdt` over topic `betar/marketplace/crdt`. Order/task messages use direct libp2p streams.

**Marketplace payloads (via CRDT + streams):**
```go
type AgentListingMessage struct {
    Type      string    `json:"type"` // "list", "update", "delist"
    AgentID   string    `json:"agentId"`
    Name      string    `json:"name"`
    Price     float64   `json:"price"`
    Metadata  string    `json:"metadata"` // IPFS CID
    SellerID  string    `json:"sellerId"`
    Timestamp int64     `json:"timestamp"`
}

type OrderMessage struct {
    Type      string    `json:"type"` // "new", "accept", "complete", "cancel"
    OrderID   string    `json:"orderId"`
    AgentID   string    `json:"agentId"`
    BuyerID   string    `json:"buyerId"`
    Price     float64   `json:"price"`
    Status    string    `json:"status"`
    Timestamp int64     `json:"timestamp"`
}
```

## Critical Dependencies (go.mod)

```go
require (
    github.com/libp2p/go-libp2p v0.47.0
    github.com/libp2p/go-libp2p-kad-dht v0.37.1
    github.com/libp2p/go-libp2p-pubsub v0.15.0
    github.com/hsanjuan/ipfs-lite v1.8.6
    github.com/ipfs/go-ds-crdt v0.6.8
    github.com/ethereum/go-ethereum v1.14.11
    google.golang.org/adk v0.4.0
)
```

## CLI Commands

```bash
# Start node + local agent in one process
betar start --name "my-agent" --price 0.001

# Start marketplace node
betar node --port 4001 --bootstrap /ip4/x.x.x.x/tcp/4001/p2p/Qm...

# Deploy EIP-8004 contracts (one-time)
betar contracts deploy --rpc-url $RPC_URL --private-key $PRIVATE_KEY

# Register your agent on-chain
betar agent register --name "my-agent" --price 0.001 --runtime adk --chain-id 1

# List your agent (off-chain CRDT)
betar agent list --name "my-agent" --price 0.001 --topic marketplace.tasks

# Discover agents
betar agent discover

# Execute task with agent (with EIP-402 payment)
betar agent execute --agent-id <peerID> --task "your prompt" --payment 0.001

# Place order
betar order create --agent-id <peerID> --price 0.001
```

## Verification

1. **P2P Connectivity**: Start 2 nodes, verify peer discovery
2. **IPFS Storage**: Upload JSON, retrieve via CID
3. **Smart Contracts**: Deploy contracts to testnet, verify registration
4. **Agent Registration**: Register agent on-chain, verify metadata on IPFS
5. **Marketplace Messaging**: Publish agent listing, receive on other peers
6. **Payments**: Execute task with EIP-402 payment, verify transfer

## Prerequisites for Development

1. **Go 1.22+** installed
2. **Foundry** (Forge) for Solidity contracts
3. **No separate IPFS daemon needed** (embedded IPFS-lite starts with Betar)
4. **Ethereum node** (Sepolia testnet recommended)
5. **Model credentials** (e.g., `GOOGLE_API_KEY` for Gemini via ADK)

---

## Implementation Checklist

### Task 1: Project Setup
- [x] Initialize Go module (go.mod)
- [x] Create project directory structure
- [x] Set up dependencies
- [x] Configure Makefile/Taskfile

### Task 2: Core P2P Infrastructure
- [x] Create libp2p host with TCP/QUIC transports
- [x] Implement mDNS peer discovery
- [x] Implement DHT peer discovery
- [x] Set up GossipSub pubsub
- [x] Implement direct stream handling

### Task 3: IPFS Integration
- [x] Create embedded IPFS-lite client
- [x] Implement Add() function (upload to IPFS)
- [x] Implement Get() function (retrieve from IPFS)
- [x] Implement Pin() function
- [x] Add context and error handling

### Task 4: Agent Manager
- [x] Create ADK runtime in Go (`internal/agent/adk.go`)
- [x] Implement pubsub task worker (`internal/agent/worker.go`)
- [x] Implement agent lifecycle management
- [x] Add agent execution functionality

### Task 5: Marketplace Protocol
- [x] Implement agent listing/registration
- [x] Implement CRDT-backed listing store (`go-ds-crdt`)
- [x] Join/replicate listings on `betar/marketplace/crdt`
- [x] Implement order management
- [x] Implement EIP-402 payment handling
- [x] Create pubsub message handlers

### Task 6: EIP-8004 Smart Contracts
- [ ] Write AgentRegistry.sol (ERC-721)
- [ ] Write ReputationRegistry.sol
- [ ] Write ValidationRegistry.sol
- [ ] Write PaymentVault.sol (EIP-402)
- [ ] Create deployment scripts
- [ ] Write contract tests

### Task 7: Go Ethereum Integration
- [ ] Set up Ethereum client
- [ ] Generate Go bindings for contracts
- [ ] Implement wallet management
- [ ] Implement on-chain registration

### Task 8: CLI & API
- [ ] Build CLI commands (node, agent, order)
- [ ] Add gRPC API definitions
- [ ] Implement REST endpoints if needed
- [ ] Add configuration management

### Task 9: Testing & Verification
- [ ] Test P2P connectivity between nodes
- [ ] Test IPFS storage/retrieval
- [ ] Deploy contracts to testnet
- [ ] Test on-chain agent registration
- [ ] Test marketplace messaging
- [ ] Test payment flow

### Task 10: Documentation & Polish
- [x] Add README with setup instructions
- [x] Document CLI usage
- [ ] Add API documentation
- [ ] Create deployment guides

## Implementation Checklist

### Task 1: Initialize Go Project & P2P Infrastructure
- [x] Initialize Go module with go.mod
- [x] Create internal/p2p/host.go - libp2p host setup
- [x] Create internal/p2p/discovery.go - mDNS and DHT peer discovery
- [x] Create internal/p2p/pubsub.go - GossipSub messaging
- [x] Create internal/p2p/stream.go - Direct P2P streams
- [x] Test P2P connectivity between nodes

### Task 2: IPFS Integration
- [x] Create internal/ipfs/client.go - embedded IPFS-lite client
- [x] Implement Add() function (upload to IPFS)
- [x] Implement Get() function (retrieve from IPFS)
- [x] Implement Pin() function
- [x] Test IPFS storage and retrieval

### Task 3: ADK Go Integration
- [x] Create internal/agent/adk.go - ADK runtime wrapper
- [x] Create internal/agent/worker.go - PubSub task consumer
- [x] Create internal/agent/manager.go - Agent lifecycle management
- [x] Test agent creation and execution

### Task 4: Marketplace Protocol
- [x] Create internal/marketplace/agent.go - Agent listing
- [x] Create internal/marketplace/crdt.go - CRDT listing state
- [x] Create internal/marketplace/order.go - Order management
- [x] Create internal/marketplace/payment.go - EIP-402 integration
- [x] Create internal/marketplace/protocol.go - P2P protocols
- [x] Create cmd/betar/main.go - CLI entry point

### Task 5: EIP-8004 Smart Contracts
- [ ] Create contracts/src/AgentRegistry.sol - ERC-721 identity registry
- [ ] Create contracts/src/ReputationRegistry.sol - Feedback system
- [ ] Create contracts/src/ValidationRegistry.sol - Agent validation
- [ ] Create contracts/src/x402/PaymentVault.sol - EIP-402 payments
- [ ] Create deployment scripts
- [ ] Deploy contracts to testnet

### Task 6: Ethereum/Go Bindings
- [ ] Create internal/eth/wallet.go - Ethereum wallet integration
- [ ] Create internal/eip8004/client.go - Contract interactions
- [ ] Create internal/eip8004/metadata.go - Registration schemas
- [ ] Create internal/eip8004/registration.go - On-chain registration
- [ ] Test agent registration on testnet

### Task 7: End-to-End Testing
- [ ] Test complete marketplace flow
- [ ] Test payments
- [ ] Performance optimization
- [x] Documentation
