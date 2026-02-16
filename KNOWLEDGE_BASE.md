# Betar Knowledge Base

This document contains research findings for building a P2P Agent-to-Agent marketplace.

---

## Table of Contents

1. [libp2p - P2P Networking](#libp2p---p2p-networking)
2. [Google ADK (Go) - Agent Framework](#google-adk-go---agent-framework)
3. [EIP-8004 - Agent Registration](#eip-8004---agent-registration)
4. [IPFS - Distributed Storage](#ipfs---distributed-storage)
5. [Marketplace CRDT - go-ds-crdt](#marketplace-crdt---go-ds-crdt)
6. [EIP-402 - Payments](#eip-402---payments)

---

## libp2p - P2P Networking

**Repository:** https://github.com/libp2p/go-libp2p

### Basic Host Creation

```go
import (
    "context"
    "github.com/libp2p/go-libp2p"
)

func main() {
    host, err := libp2p.New()
    if err != nil {
        panic(err)
    }
    defer host.Close()

    fmt.Println("Peer ID:", host.ID())
    fmt.Println("Listen addresses:", host.Addrs())
}
```

### Customized Host Configuration

```go
import (
    "crypto/rand"
    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p/crypto"
    "github.com/libp2p/go-libp2p/p2p/transport/tcp"
    "github.com/libp2p/go-libp2p/p2p/muxer/yamux"
    "github.com/libp2p/go-libp2p/p2p/security/noise"
    "github.com/libp2p/go-libp2p/p2p/security/tls"
)

func createCustomHost(listenPort int) (host.Host, error) {
    privKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
    if err != nil {
        return nil, err
    }

    node, err := libp2p.New(
        libp2p.Identity(privKey),
        libp2p.ListenAddrStrings(
            fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort),
            fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", listenPort),
        ),
        libp2p.Transport(tcp.NewTCPTransport),
        libp2p.Muxer("/yamux/1.0.0", yamux.DefaultConfig),
        libp2p.Security(noise.ID, noise.New),
        libp2p.Security(tls.ID, tls.New),
        libp2p.NATPortMap(),
        libp2p.EnableRelay(),
        libp2p.EnableHolePunching(),
    )

    return node, err
}
```

### Key Configuration Options

| Option | Description |
|--------|-------------|
| `Identity(sk)` | Set peer identity |
| `ListenAddrStrings(s...)` | Configure listen addresses |
| `Transport(constructor)` | Add custom transport (TCP, QUIC, WebSocket, WebRTC) |
| `Muxer(name, config)` | Add stream multiplexer (yamux, mplex) |
| `Security(name, constructor)` | Add security transport (TLS, Noise) |
| `NATPortMap()` | Enable UPnP port mapping |
| `EnableRelay()` | Enable circuit relay transport |
| `EnableHolePunching()` | Enable NAT traversal |

### Peer Discovery

#### mDNS (Local Network Discovery)

```go
import (
    "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

func setupMdnsDiscovery(ctx context.Context, h host.Host, serviceName string) error {
    mdnsService := mdns.NewMdnsService(h, serviceName)
    if err := mdnsService.Start(); err != nil {
        return err
    }

    peerChan := make(chan *peer.AddrInfo, 10)
    mdnsService.RegisterNotifee(peerChan)

    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case addrInfo := <-peerChan:
                h.Peerstore().AddAddrs(addrInfo.ID, addrInfo.Addrs, peerstore.PermanentAddrTTL)
                h.Connect(ctx, *addrInfo)
            }
        }
    }()

    return nil
}
```

#### DHT (Decentralized Discovery)

```go
import (
    "github.com/libp2p/go-libp2p/p2p/discovery/routing"
    "github.com/libp2p/go-libp2p-kad-dht"
)

func setupDHT(ctx context.Context, h host.Host) (*dht.IpfsDHT, error) {
    kademliaDHT := dht.NewDHT(ctx, h, dht.Mode(dht.ServerMode))

    // Bootstrap with known peers
    bootstrapPeers := []string{
        "/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
        "/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
        "/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zp5i9cM2m2E1r4NkHeF7NhU9gBbz3K",
        "/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ2wBb1jzYp5VCxQGtEex9kK",
    }

    for _, addr := range bootstrapPeers {
        pi, _ := peer.AddrInfoFromString(addr)
        h.Peerstore().AddAddrs(pi.ID, pi.Addrs, peerstore.PermanentAddrTTL)
        h.Connect(ctx, *pi)
    }

    return kademliaDHT, nil
}
```

### PubSub (GossipSub)

```go
import (
    "github.com/libp2p/go-libp2p/pubsub"
    "github.com/libp2p/go-libp2p/pubsub/gossipsub"
)

func setupPubSub(ctx context.Context, h host.Host) (*pubsub.PubSub, error) {
    gs := gossipsub.NewGossipSub(
        ctx,
        h,
        gossipsub.WithMessageSigning(true),
        gossipsub.WithStrictSignatureVerification(true),
    )
    return gs, nil
}

// Publishing messages
func publishOrder(ctx context.Context, ps *pubsub.PubSub, topicStr string, data []byte) error {
    topic, err := ps.Join(topicStr)
    if err != nil {
        return err
    }
    return topic.Publish(ctx, data)
}

// Subscribing to messages
func subscribeToOrders(ctx context.Context, ps *pubsub.PubSub, topicStr string) (*pubsub.Subscription, error) {
    topic, err := ps.Join(topicStr)
    if err != nil {
        return nil, err
    }
    return topic.Subscribe()
}
```

### Recommended Dependencies

```
github.com/libp2p/go-libp2p v0.45.0
github.com/libp2p/go-libp2p-kad-dht v0.25.0
github.com/libp2p/go-libp2p-pubsub v0.11.0
github.com/libp2p/go-libp2p-noise v0.5.0
github.com/libp2p/go-libp2p-tls v0.5.0
```

---

## Google ADK (Go) - Agent Framework

**Module:** https://pkg.go.dev/google.golang.org/adk@v0.4.0  
**Repository:** https://github.com/google/adk-go

### Important Finding

`google.golang.org/adk@v0.4.0` provides a **native Go agent framework**, so we can remove the Python sidecar and run agents in the same Go process as the marketplace node.

Benefits for this project:
1. No cross-language HTTP bridge for local execution paths
2. Lower overhead for task dispatch (pubsub message -> Go handler -> ADK run)
3. Better token hygiene: keep prompts small, structured, and event-driven from pubsub payloads

### Core ADK Packages (v0.4.0)

| Package | Purpose |
|--------|---------|
| `agent/llmagent` | Build LLM-backed agents |
| `model/gemini` | Gemini model client implementation |
| `tool/functiontool` | Wrap Go functions as callable tools |
| `runner` | Runtime for orchestrating agent execution |
| `server/adka2a`, `server/adkrest` | Expose agents via A2A/REST when needed |

### Agent Creation (Go)

```go
import (
    "context"
    "os"

    "google.golang.org/genai"

    "google.golang.org/adk/agent/llmagent"
    "google.golang.org/adk/model/gemini"
    "google.golang.org/adk/tool"
    "google.golang.org/adk/tool/geminitool"
)

func newAgent(ctx context.Context) (*llmagent.Agent, error) {
    model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{
        APIKey: os.Getenv("GOOGLE_API_KEY"),
    })
    if err != nil {
        return nil, err
    }

    return llmagent.New(llmagent.Config{
        Name:        "marketplace_agent",
        Model:       model,
        Description: "Executes marketplace tasks from p2p queue",
        Instruction: "Process only marketplace task payloads.",
        Tools: []tool.Tool{
            geminitool.GoogleSearch{},
        },
    })
}
```

### Stream-Driven Execution Pattern (Current Runtime)

Betar uses direct libp2p streams for agent execution and order transitions:

1. Discover seller peer via CRDT listing (`SellerID` + `Addrs`)
2. Connect over libp2p (direct path first, relay-capable fallback)
3. Send compact execution/order payload over stream protocol
4. Process response as request/response ACK

### Token Efficiency Practices

- Keep stream payload schema minimal (`task`, `contextRef`, `constraints`)
- Store large context in IPFS and pass CID pointers, not full text
- Reuse short system instructions and avoid verbose conversation replay
- Enforce max input/output tokens per task class

### Recommended Dependency

```go
google.golang.org/adk v0.4.0
```

---

## EIP-8004 - Agent Registration

**Specification:** https://eips.ethereum.org/EIPS/eip-8004

EIP-8004 defines three lightweight registries for agent discovery and interaction.

### Agent Identity

Agents identified by:
- `agentRegistry`: Format `{namespace}:{chainId}:{identityRegistry}` (e.g., `eip155:1:0x742...`)
- `agentId`: ERC-721 tokenId assigned incrementally

### Registration File Schema

```json
{
  "type": "https://eips.ethereum.org/EIPS/eip-8004#registration-v1",
  "name": "myAgentName",
  "description": "Description of the Agent",
  "image": "https://example.com/agentimage.png",
  "services": [
    { "name": "web", "endpoint": "https://web.agentxyz.com/" },
    { "name": "A2A", "endpoint": "...", "version": "0.3.0" },
    { "name": "MCP", "endpoint": "...", "version": "2025-06-18" }
  ],
  "x402Support": false,
  "active": true,
  "registrations": [
    { "agentId": 22, "agentRegistry": "eip155:1:0x742..." }
  ],
  "supportedTrust": ["reputation", "crypto-economic", "tee-attestation"]
}
```

### Agent URI

Supports:
- `ipfs://{cid}` - IPFS content address
- `https://example.com/agent.json` - HTTPS
- `data:application/json;base64,...` - Base64 encoded

### Smart Contract Functions

```solidity
// AgentRegistry
function register(string agentURI, MetadataEntry[] calldata metadata) external returns (uint256 agentId)
function setAgentURI(uint256 agentId, string calldata newURI) external
function getMetadata(uint256 agentId, string memory metadataKey) external view returns (bytes memory)

// ReputationRegistry
function giveFeedback(
    uint256 agentId,
    int128 value,
    uint8 valueDecimals,
    string calldata tag1,
    string calldata tag2,
    string calldata endpoint,
    string calldata feedbackURI,
    bytes32 feedbackHash
) external
```

### Feedback Value Examples

| tag1 | Measurement | Example | value | valueDecimals |
|------|-------------|---------|-------|---------------|
| starred | Quality rating (0-100) | 87/100 | 87 | 0 |
| reachable | Endpoint reachable (binary) | true | 1 | 0 |
| uptime | Endpoint uptime (%) | 99.77% | 9977 | 2 |
| successRate | Success rate (%) | 89% | 89 | 0 |
| responseTime | Response time (ms) | 560ms | 560 | 0 |

---

## IPFS - Distributed Storage

**Implementation:** https://github.com/hsanjuan/ipfs-lite

Betar runs an embedded IPFS-lite peer in-process (no external daemon).
The IPFS-lite node is wired to the same libp2p host used for marketplace networking.

### Go Integration

```go
import (
    "bytes"
    "io"

    ipfslite "github.com/hsanjuan/ipfs-lite"
)

func addToIPFS(p *ipfslite.Peer, fileData []byte) (string, error) {
    node, err := p.AddFile(context.Background(), bytes.NewReader(fileData), nil)
    if err != nil {
        return "", err
    }
    return node.Cid().String(), nil
}

func catFromIPFS(p *ipfslite.Peer, cid cid.Cid) ([]byte, error) {
    r, err := p.GetFile(context.Background(), cid)
    if err != nil {
        return nil, err
    }
    defer r.Close()
    return io.ReadAll(r)
}
```

### CRDT DAG Sync

For `go-ds-crdt`, replicas need an IPLD `DAGService`. Betar passes IPFS-lite's `DAGService` directly into `crdt.New(...)`.
This allows CRDT state exchange over pubsub while persisting/fetching DAG blocks through the embedded node.

---

## Marketplace CRDT - go-ds-crdt

**Repository:** https://github.com/ipfs/go-ds-crdt

`go-ds-crdt` provides a Merkle-CRDT datastore. Betar uses it as the marketplace listing state layer:

- Topic for CRDT head gossip: `betar/marketplace/crdt`
- Transport: libp2p GossipSub broadcaster (`NewPubSubBroadcaster`)
- Persistence model: key-value entries under `/marketplace/agents/...`
- Value payload: serialized `AgentListing`

### Why this model

1. Convergent replicated listings without a central coordinator
2. Eventual consistency across peers even with intermittent connectivity
3. Native IPLD DAG history that can be repaired/replayed by CRDT datastore

### Implementation notes in Betar

- CRDT datastore is created with `crdt.New(...)`
- Broadcaster is created from existing libp2p pubsub
- DAG operations are served by embedded IPFS-lite (`ipld.DAGService`)
- CLI `betar start` continuously updates local agent listing entries; CRDT pubsub subscription is owned by `NewPubSubBroadcaster`
- Agent execution and order lifecycle are not app-level pubsub topics; they run on direct libp2p streams.

---

## EIP-402 - Payments

EIP-402 defines a payment standard for agent services, often implemented as a Payment Vault.

### Basic Payment Flow

1. Buyer deposits funds into Payment Vault
2. Buyer requests service from agent
3. Agent verifies payment and provides service
4. Funds released to agent wallet upon completion

### Example Payment Contract Structure

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

contract PaymentVault {
    mapping(address => uint256) public balances;
    mapping(bytes32 => PaymentRequest) public paymentRequests;

    struct PaymentRequest {
        address buyer;
        address seller;
        uint256 amount;
        bool released;
    }

    event Deposit(address indexed user, uint256 amount);
    event PaymentRequested(bytes32 indexed requestId, address buyer, address seller, uint256 amount);
    event PaymentReleased(bytes32 indexed requestId, uint256 amount);

    function deposit(uint256 amount) external {
        require(IERC20(token).transferFrom(msg.sender, address(this), amount));
        balances[msg.sender] += amount;
        emit Deposit(msg.sender, amount);
    }

    function requestPayment(
        address seller,
        uint256 amount
    ) external returns (bytes32 requestId) {
        require(balances[msg.sender] >= amount, "Insufficient balance");
        balances[msg.sender] -= amount;

        requestId = keccak256(abi.encodePacked(msg.sender, seller, amount, block.timestamp));
        paymentRequests[requestId] = PaymentRequest({
            buyer: msg.sender,
            seller: seller,
            amount: amount,
            released: false
        });

        emit PaymentRequested(requestId, msg.sender, seller, amount);
    }

    function releasePayment(bytes32 requestId) external {
        PaymentRequest storage req = paymentRequests[requestId];
        require(!req.released, "Already released");

        req.released = true;
        balances[req.seller] += req.amount;

        emit PaymentReleased(requestId, req.amount);
    }
}
```

---

## Architecture Summary

```
┌─────────────────────────────────────────────────────────────────┐
│                        Marketplace Node                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │   libp2p     │  │   IPFS       │  │   ADK Agent Runtime  │  │
│  │   Host       │  │   Client     │  │   (Native Go)        │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│  ┌──────────────┐                                              │
│  │ Marketplace  │                                              │
│  │ CRDT Store   │                                              │
│  │ (go-ds-crdt) │                                              │
│  └──────────────┘                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  Ethereum    │  │  EIP-8004    │  │   EIP-402            │  │
│  │  Client      │  │  Registry    │  │   Payments           │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Development Prerequisites

1. **Go 1.22+** - Main application and ADK agent runtime
2. **Foundry** - Solidity contracts
3. **Embedded IPFS-lite** - Started by Betar (no separate daemon)
4. **Ethereum node** - Sepolia testnet recommended
5. **Model API credentials** - e.g. `GOOGLE_API_KEY` for Gemini

---

## References

- libp2p: https://github.com/libp2p/go-libp2p
- libp2p Docs: https://docs.libp2p.io
- ADK Go module: https://pkg.go.dev/google.golang.org/adk@v0.4.0
- ADK Go repository: https://github.com/google/adk-go
- EIP-8004: https://eips.ethereum.org/EIPS/eip-8004
- IPFS-lite: https://github.com/hsanjuan/ipfs-lite
