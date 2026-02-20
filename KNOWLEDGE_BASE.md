# Betar Knowledge Base

This document contains research findings and implementation status for building a P2P Agent-to-Agent marketplace.

---

## Table of Contents

1. [Implementation Status](#implementation-status)
2. [libp2p - P2P Networking](#libp2p---p2p-networking)
3. [Google ADK (Go) - Agent Framework](#google-adk-go---agent-framework)
4. [EIP-8004 - Agent Registration](#eip-8004---agent-registration)
5. [IPFS - Distributed Storage](#ipfs---distributed-storage)
6. [Marketplace CRDT - go-ds-crdt](#marketplace-crdt---go-ds-crdt)
7. [EIP-402 / x402 Payments](#eip-402--x402-payments)
8. [Architecture Summary](#architecture-summary)

---

## Implementation Status

| Component | Status | Notes |
|-----------|--------|-------|
| P2P Infrastructure | ✅ Done | host, mDNS, DHT, pubsub, streams |
| IPFS Integration | ✅ Done | embedded ipfs-lite |
| Agent Manager | ✅ Done | ADK Go runtime |
| Marketplace CRDT | ✅ Done | listing state via go-ds-crdt |
| Order Management | ✅ Done | order service |
| x402 Payments | ✅ Done | payment flow implemented |
| Wallet/ETH | ✅ Done | ERC20 transfers |
| Smart Contracts | ✅ Done | AgentRegistry, ReputationRegistry, ValidationRegistry, PaymentVault |
| HTTP API Server | ✅ Done | gorilla/mux |
| EIP-8004 Client | ❌ Missing | on-chain registration not integrated |

### Known Gaps

1. **EIP-8004 Client**: On-chain agent registration not wired to marketplace
2. **Documentation**: This file needs to be updated as implementation evolves

---

## libp2p - P2P Networking

**Repository:** https://github.com/libp2p/go-libp2p

### Implementation: `internal/p2p/`

| File | Purpose |
|------|---------|
| `host.go` | libp2p host creation with TCP/QUIC transports |
| `discovery.go` | mDNS (local) + DHT (network) peer discovery |
| `pubsub.go` | GossipSub for CRDT replication |
| `stream.go` | Direct P2P streams for agent execution |

### Key Code Patterns

#### Host Creation

```go
// internal/p2p/host.go
host, err := p2p.NewHost(ctx, cfg.P2P)
```

#### PubSub

```go
// internal/p2p/pubsub.go
ps, err := p2p.NewPubSub(ctx, host.RawHost())
```

#### Stream Protocol (`/betar/marketplace/1.0.0`)

```go
// Sending via stream (buyer side)
resp, err := streamHandler.SendMessage(ctx, peerID, "execute", reqData)

// Receiving (seller side)
streamHandler.RegisterHandler("execute", m.handleExecuteRequest)
```

---

## Google ADK (Go) - Agent Framework

**Module:** `google.golang.org/adk@v0.4.0`

### Implementation: `internal/agent/`

| File | Purpose |
|------|---------|
| `adk.go` | ADK runtime wrapper |
| `manager.go` | Agent lifecycle + P2P execution |
| `worker.go` | Legacy pubsub worker (not used) |

### Agent Creation

```go
// internal/agent/manager.go
func NewManager(runtimeCfg ADKConfig, ...) (*Manager, error) {
    runtime, err := NewADKRuntime(runtimeCfg)
    // ...
}
```

### Local Execution

```go
result, err := m.runtime.RunTask(ctx, types.TaskRequest{
    AgentID:   agent.AgentID,
    Input:     input,
    RequestID: requestID,
})
```

### Remote Execution

```go
// Connect to peer, then send via stream
resp, err := m.streamHandler.SendMessage(ctx, peerID, "execute", reqData)
```

---

## EIP-8004 - Agent Registration

**Specification:** https://eips.ethereum.org/EIPS/eip-8004

### Smart Contracts: `contracts/src/`

| File | Purpose |
|------|---------|
| `AgentRegistry.sol` | ERC-721 based agent identity |
| `ReputationRegistry.sol` | Feedback system |
| `ValidationRegistry.sol` | Agent validation |
| `x402/PaymentVault.sol` | EIP-402 payment vault |

### Registration Schema

```go
// pkg/types/types.go
type AgentRegistration struct {
    Type        string    `json:"type"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Image       string    `json:"image,omitempty"`
    Services    []Service `json:"services"`
    X402Support bool      `json:"x402Support"`
    Active      bool      `json:"active"`
}
```

### Status

⚠️ **Not Integrated**: On-chain registration via EIP-8004 is not wired to marketplace. The contracts exist but the Go client (`internal/eip8004/`) is a stub.

---

## IPFS - Distributed Storage

**Implementation:** `github.com/hsanjuan/ipfs-lite`

### Implementation: `internal/ipfs/client.go`

Betar runs embedded IPFS-lite peer in-process (no external daemon).

```go
// Creating client
ipfsClient, err := ipfs.NewClient(ctx, p2pHost.RawHost(), discovery.Routing(), cfg.Storage.DataDir)

// Adding data
cid, err := ipfsClient.AddJSON(ctx, data)

// Retrieving data
err := ipfsClient.GetJSON(ctx, cid, &result)

// Pinning
ipfsClient.Pin(ctx, cid)
```

---

## Marketplace CRDT - go-ds-crdt

**Implementation:** `internal/marketplace/crdt.go`

### Topic: `betar/marketplace/crdt`

```go
// internal/marketplace/crdt.go
const CRDTTopic = "betar/marketplace/crdt"
```

### Usage

```go
listingService, err := marketplace.NewAgentListingService(ctx, ipfsClient, p2pPubSub, p2pHost.ID())
listingService.UpsertLocalListing(&types.AgentListingMessage{...})
listingService.UpdateListing(ctx, listing)
listings := listingService.ListListings()
```

---

## EIP-402 / x402 Payments

**Implementation:** `internal/marketplace/payment.go`, `internal/marketplace/x402.go`

**Library:** `github.com/mark3labs/x402-go/v2`

### Networks & Tokens

```go
const NetworkBaseSepolia = "eip155:84532"
const NetworkBaseMainnet  = "eip155:8453"

const USDCBaseSepolia = "0x036CbD53842c5426634e7929541eC2318f3dCF7e"
const USDCBaseMainnet  = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
```

### Key Types

```go
// PaymentHeader — embedded in P2P execute messages
type PaymentHeader struct {
    Requirement PaymentRequirements  `json:"requirement"`
    Payer       string               `json:"payer"`
    PaymentID   string               `json:"payment_id"`
    Signature   string               `json:"signature,omitempty"`
    Accepted    *PaymentRequirements `json:"accepted,omitempty"`
    Payload     *EVMPayload          `json:"payload,omitempty"`
}

// TaskExecuteRequest — sent from buyer to seller over P2P stream
type TaskExecuteRequest struct {
    AgentID         string         `json:"agent_id"`
    Input           string         `json:"input"`
    PaymentHeader   *PaymentHeader `json:"payment_header,omitempty"`
    TransactionHash string         `json:"transaction_hash,omitempty"`
}

// PaymentRequiredResponse — returned by seller when payment is required
type PaymentRequiredResponse struct {
    AgentID            string               `json:"agent_id"`
    RequestID          string               `json:"request_id"`
    Message            string               `json:"message"`
    PaymentRequirement *PaymentRequirements `json:"payment_requirement,omitempty"`
    RequiresPayment    bool                 `json:"requires_payment"`
}
```

### Implemented Payment Flow

```
Buyer                              Seller
  │                                   │
  │── Execute Request ───────────────►│ {agentId, input, paymentHeader?}
  │                                   │──► Check: agent requires payment?
  │                                   │──► No payment header → return 402
  │                                   │
  │◄── PaymentRequired ───────────────│ {requires_payment: true, payment_requirement}
  │                                   │
  │  Buyer signs EIP-712 USDC auth    │
  │                                   │
  │── Execute + Payment ──────────────►│ {agentId, input, paymentHeader}
  │                                   │──► Step 1: Local validation
  │                                   │     (signature, timestamp, nonce, payTo, asset)
  │                                   │──► Step 2: Settle with facilitator (5 retries)
  │                                   │     POST {facilitator}/settle
  │                                   │──► Step 3: Wait for on-chain confirmation
  │                                   │──► Step 4: Execute ADK task
  │                                   │
  │◄── Output + TxHash ───────────────│ {output, transaction_hash}
```

### PaymentService API

```go
// Seller side
svc.CreateRequirement(payee, amount)        // build PaymentRequirements to return in 402
svc.VerifyAndSettle(ctx, header, amount)    // validate → settle → confirm; returns txHash

// Buyer side
svc.CreatePayment(ctx, payee, amount, orderID)   // sign and return PaymentHeader
svc.SignRequirement(req, orderID)                // sign an existing requirement
```

### Facilitator

- Default URL: `http://localhost:8080` (configurable)
- Endpoints: `POST /verify`, `POST /settle`
- Settlement uses exponential back-off retry (5 attempts, starting at 500 ms)
- On-chain mechanism: EIP-3009 `transferWithAuthorization` on the USDC contract

---

## Architecture Summary

```
┌─────────────────────────────────────────────────────────────────┐
│                        Marketplace Node                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │   libp2p    │  │   IPFS       │  │   ADK Agent Runtime  │  │
│  │   Host       │  │   Client     │  │   (Native Go)        │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│  ┌──────────────┐                                              │
│  │ Marketplace  │                                              │
│  │ CRDT Store   │                                              │
│  │ (go-ds-crdt) │                                              │
│  └──────────────┘                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  Ethereum    │  │  x402         │  │   Wallet             │  │
│  │  Client      │  │  Payments    │  │   (ERC20)           │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│  ┌──────────────┐                                              │
│  │  HTTP API    │                                              │
│  │  (gorilla)   │                                              │
│  └──────────────┘                                              │
└─────────────────────────────────────────────────────────────────┘
```

### CLI Commands

```bash
# Start node + agent (recommended)
betar start --name "math-agent" --price 0.001 --port 4001

# Start marketplace node only
betar node --port 4001

# Run agent
betar agent serve --name "my-agent"

# Create order
betar order create --agent-id <peerID> --price 0.001

# Check wallet balance
betar wallet balance
```

---

## References

- libp2p: https://github.com/libp2p/go-libp2p
- libp2p Docs: https://docs.libp2p.io
- ADK Go module: https://pkg.go.dev/google.golang.org/adk@v0.4.0
- ADK Go repository: https://github.com/google/adk-go
- EIP-8004: https://eips.ethereum.org/EIPS/eip-8004
- IPFS-lite: https://github.com/hsanjuan/ipfs-lite
- x402-go: https://github.com/mark3labs/x402-go
