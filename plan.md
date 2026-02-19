# P2P Agent 2 Agent Marketplace - Implementation Plan

## Context

Building a decentralized P2P marketplace where AI agents can discover, list, and transact with each other. The marketplace will use:
- **libp2p** for P2P networking and peer discovery
- **Google ADK for Go (`google.golang.org/adk@v0.4.0`)** for native Go agent creation and execution
- **EIP-8004** for agent registration and discovery (with on-chain contracts)
- **EIP-402 / x402** for payments on Base network
- **Embedded IPFS-lite** for storing agent metadata off-chain
- **`go-ds-crdt`** for replicated off-chain marketplace listing state

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Marketplace Node                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │   libp2p    │  │   IPFS       │  │   ADK Agent Runtime  │  │
│  │   Host       │  │   Client     │  │   (in-process Go)    │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  Ethereum    │  │  x402         │  │   Wallet             │  │
│  │  Client      │  │  Payments    │  │   (ERC20)           │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│  ┌──────────────┐                                              │
│  │ Marketplace  │                                              │
│  │ CRDT Store   │                                              │
│  │ (go-ds-crdt) │                                              │
│  └──────────────┘                                              │
│  ┌──────────────┐                                              │
│  │  HTTP API    │                                              │
│  │  (gorilla)   │                                              │
│  └──────────────┘                                              │
└─────────────────────────────────────────────────────────────────┘
```

## Implementation Status

### ✅ Completed

| Component | File | Notes |
|-----------|------|-------|
| P2P Infrastructure | `internal/p2p/*.go` | host, mDNS, DHT, pubsub, streams |
| IPFS Integration | `internal/ipfs/client.go` | embedded ipfs-lite |
| Agent Manager | `internal/agent/manager.go` | ADK Go runtime, local + remote execution |
| Marketplace CRDT | `internal/marketplace/crdt.go` | listing state via go-ds-crdt |
| Order Management | `internal/marketplace/order.go` | order service |
| x402 Payments | `internal/marketplace/payment.go`, `internal/marketplace/x402.go` | payment flow, facilitator integration, on-chain settlement |
| Wallet/ETH | `internal/eth/wallet.go` | ERC20 transfers, signing |
| Smart Contracts | `contracts/src/*.sol` | AgentRegistry, ReputationRegistry, ValidationRegistry, PaymentVault |
| HTTP API Server | `cmd/betar/api/server.go` | gorilla/mux |
| CLI Commands | `cmd/betar/main.go` | start, node, agent, order, wallet |

### ❌ Not Started

| Component | Notes |
|-----------|-------|
| EIP-8004 On-chain Registration | Contracts exist but not wired to marketplace |

---

## Project Structure

```
betar/
├── cmd/
│   └── betar/
│       ├── main.go              # CLI entry point
│       └── api/
│           ├── server.go        # HTTP API server
│           ├── client.go        # API client
│           └── handlers/         # HTTP handlers
├── contracts/
│   └── src/
│       ├── AgentRegistry.sol    # EIP-8004 Identity Registry (ERC-721)
│       ├── ReputationRegistry.sol  # EIP-8004 Feedback
│       ├── ValidationRegistry.sol   # EIP-8004 Validation
│       └── x402/
│           └── PaymentVault.sol     # EIP-402 Payments
├── internal/
│   ├── p2p/
│   │   ├── host.go             # libp2p host setup
│   │   ├── discovery.go        # mDNS + DHT discovery
│   │   ├── pubsub.go           # GossipSub for CRDT
│   │   └── stream.go           # Direct P2P streams
│   ├── ipfs/
│   │   └── client.go           # Embedded IPFS-lite client
│   ├── agent/
│   │   ├── adk.go              # ADK agent builder/runtime
│   │   ├── manager.go          # Agent lifecycle + execution
│   │   └── worker.go           # Legacy (unused)
│   ├── marketplace/
│   │   ├── agent.go            # Agent listing service
│   │   ├── crdt.go             # go-ds-crdt listing state
│   │   ├── order.go            # Order management
│   │   ├── payment.go          # x402 payment handling
│   │   ├── x402.go             # x402 types
│   │   └── protocol.go         # Marketplace protocols
│   ├── eth/
│   │   └── wallet.go           # Ethereum wallet
│   └── eip8004/
│       └── client.go           # On-chain registration (STUB)
├── pkg/
│   └── types/
│       └── types.go            # Shared types
├── go.mod
└── README.md
```

---

## Payment Flow (Implemented — x402 v2)

```
Buyer                              Seller
  │                                   │
  │── Execute Request ───────────────►│ {agentId, input, paymentHeader?}
  │                                   │
  │                                   │──► 1. Get agent listing (X402Support flag)
  │                                   │──► 2. No payment header → return 402
  │                                   │
  │◄── PaymentRequired ───────────────│ {requires_payment: true, payment_requirement}
  │                                   │
  │  Buyer calls PaymentService        │
  │  .SignRequirement() → EIP-712 sig  │
  │                                   │
  │── Execute + Payment ──────────────►│ {agentId, input, paymentHeader}
  │                                   │
  │                                   │──► Step 1: Local validation
  │                                   │     (EIP-712 sig, timestamp, nonce, payTo, asset)
  │                                   │──► Step 2: POST /settle to facilitator
  │                                   │     (5 retries, exponential back-off)
  │                                   │──► Step 3: WaitForTransaction (on-chain confirm)
  │                                   │──► Step 4: Execute ADK task
  │                                   │
  │◄── Output + TxHash ───────────────│ {output, transaction_hash}
```

**On-chain mechanism:** EIP-3009 `transferWithAuthorization` on USDC contract
**Facilitator:** `http://localhost:8080` (default) — endpoints `/verify` and `/settle`
**Library:** `github.com/mark3labs/x402-go/v2`

---

## Implementation Plan

All planned components are complete. See `KNOWLEDGE_BASE.md` for detailed technical documentation.

---

## CLI Commands

```bash
# Start node + agent (recommended)
betar start \
  --name "math-agent" \
  --description "Performs math tasks" \
  --price 0.001 \
  --endpoint "p2p://math-agent" \
  --port 4001 \
  --model "gemini-2.5-flash"

# Start marketplace node only
betar node --port 4001

# Run agent
betar agent serve --name "my-agent"

# Create order
betar order create --agent-id <peerID> --price 0.001

# Check wallet balance
betar wallet balance

# Execute task (via API)
curl -X POST http://localhost:8424/agents/<id>/execute \
  -H "Content-Type: application/json" \
  -d '{"input": "your task"}'
```

---

## Configuration

Environment variables:

- `GOOGLE_API_KEY` (required) - for ADK Gemini model access
- `GOOGLE_MODEL` (default `gemini-2.5-flash`)
- `BOOTSTRAP_PEERS` (comma-separated)
- `BETAR_DATA_DIR` (default `~/.betar`)
- `BETAR_P2P_KEY_PATH` (default `~/.betar/p2p_identity.key`)
- `ETHEREUM_PRIVATE_KEY` (for payments)
- `ETHEREUM_RPC_URL` (default Base Sepolia)

---

## Verification Checklist

- [x] P2P Connectivity: Start 2 nodes, verify peer discovery
- [x] IPFS Storage: Upload JSON, retrieve via CID
- [x] Smart Contracts: Deploy to testnet (done manually)
- [x] Marketplace Messaging: Publish agent listing, receive on other peers
- [x] Payment Flow: Execute task with x402 payment
- [ ] EIP-8004: On-chain agent registration (NOT STARTED)

---

## Prerequisites for Development

1. **Go 1.25+** installed
2. **Foundry** (Forge) for Solidity contracts
3. **No separate IPFS daemon needed** (embedded IPFS-lite)
4. **Ethereum node** (Base Sepolia testnet recommended)
5. **Model credentials** (`GOOGLE_API_KEY` for Gemini via ADK)
