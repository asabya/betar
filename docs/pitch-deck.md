# Betar Pitch Deck

---

## Slide 1: Betar

### x402 Meets libp2p. Money Flows Peer-to-Peer.

**PL Genesis: Frontiers of Collaboration** | March 2026

Track: Web3 + AI/AGI | Category: Existing Code | **Beta — Base Sepolia Testnet**

Team: [TBD]

> **Speaker notes:** Open with the tagline. Betar is a decentralized peer-to-peer agent marketplace built in Go. We took the x402 payment standard -- designed for HTTP -- and brought it to libp2p. The result: autonomous AI agents that discover, execute, and pay each other without a single HTTP request in the critical path.

---

## Slide 2: The Problem

### AI agents are exploding. Their payment rails are stuck in 2015.

- **2.4 billion** API calls per day across major AI platforms -- and growing
- Every agent-to-agent transaction today routes through a centralized intermediary
- Current payment options for agents:
  - Custodial wallets managed by platform operators
  - HTTP APIs gated by API keys and corporate billing
  - Manual invoicing and settlement
- Agents cannot autonomously discover services, negotiate prices, or transfer value
- They are trapped behind HTTP walls, dependent on DNS, TLS certificates, and corporate gatekeepers just to find each other

The "agentic web" everyone talks about has no native payment layer. Agents can think, but they cannot transact.

> **Speaker notes:** Paint the picture: imagine an AI agent that needs a translation service. Today, it has to know a specific API endpoint, authenticate with an API key provisioned by a human, and pay through a billing account set up manually. There is no open marketplace. There is no peer-to-peer discovery. There is no way for two agents to autonomously agree on a price and settle payment. The plumbing simply does not exist.

---

## Slide 3: The Insight

### Payments should be part of the communication protocol, not bolted on top.

- The x402 standard (HTTP 402 Payment Required) already exists -- Coinbase shipped it for HTTP
- It works: server returns 402, client signs a payment, resends the request with a payment header
- But x402 is coupled to HTTP. It assumes URLs, DNS, TLS, client-server topology
- Meanwhile, libp2p already solves discovery, identity, and transport for decentralized networks

**The question we asked:**

If agents discover each other over libp2p, why should they pay each other over HTTP?

What if x402 ran natively on libp2p streams?

> **Speaker notes:** This is the key insight. x402 is a good protocol -- it turns payment into a first-class part of the request/response cycle. But it was designed for the web as it exists today: centralized, DNS-dependent, HTTP-based. We realized that agents operating in a P2P network should not have to break out of that network just to pay each other. The payment should flow through the same stream as the task execution.

---

## Slide 4: The Solution -- Betar

### The payment layer for the agent web.

Betar is a decentralized P2P agent marketplace where autonomous agents:

1. **Discover** each other via Kademlia DHT, mDNS, and CRDT-replicated listings
2. **Execute** tasks over direct libp2p streams
3. **Pay** each other using x402 adapted for libp2p -- USDC on Base Sepolia
4. **Build reputation** through on-chain feedback and task history

No HTTP servers. No API keys. No custodial wallets. No middlemen.

One binary. Pure peer-to-peer.

> **Speaker notes:** Betar is a single Go binary that launches a libp2p node, discovers peers, replicates marketplace state via CRDT, and handles task execution and payment over direct streams. An agent operator runs `betar start --name "my-agent" --price 0.001` and they are live on the network. Another agent discovers them through the replicated marketplace, opens a stream, and transacts -- all without touching HTTP.

---

## Slide 5: How It Works

### End-to-end flow -- zero HTTP in the critical path

```
Node Start
    |
    v
libp2p Host (TCP/QUIC)
    |
    v
DHT Bootstrap
    |
    v
CRDT Marketplace Subscription (GossipSub topic: betar/marketplace/crdt)
    |
    v
Agent Registration --> Listing replicated to all peers
    |
    v
Buyer discovers agent in CRDT state
    |
    v
Opens /x402/libp2p/1.0.0 stream to seller peer
    |
    v
x402.request --> x402.payment_required (402 equivalent)
    |
    v
Buyer signs EIP-712 USDC transfer
    |
    v
x402.paid_request --> Seller verifies --> Executes agent
    |
    v
x402.response (result + tx_hash)
    |
    v
Off-path facilitator settles USDC on-chain
```

The only HTTP in the entire system is the off-path facilitator settlement call and the Ethereum RPC endpoint. Discovery, execution, and payment negotiation are pure libp2p.

> **Speaker notes:** Walk through the diagram. Emphasize that the facilitator is intentionally off the critical path -- it handles on-chain settlement asynchronously. The agent-to-agent interaction is entirely peer-to-peer. This is not "P2P with an HTTP escape hatch" -- it is P2P with an async settlement layer.

---

## Slide 6: x402 over libp2p -- The Innovation

### This is what we built. This is why it matters.

**Protocol ID:** `/x402/libp2p/1.0.0`

**Five message types over a single stream:**

| Step | Message Type | Direction | Purpose |
|------|-------------|-----------|---------|
| 1 | `x402.request` | Client --> Server | Request resource execution (optionally with preemptive payment) |
| 2 | `x402.payment_required` | Server --> Client | Challenge: price, nonce, payment requirements (the "402") |
| 3 | `x402.paid_request` | Client --> Server | Signed EIP-712 USDC transfer + original request body |
| 4 | `x402.response` | Server --> Client | Execution result + payment ID + tx hash |
| 5 | `x402.error` | Either direction | Structured error with typed codes (10 error types, retryable flags) |

**Binary wire format:**

```
[type_len : uint16 BE][type : UTF-8 string][data_len : uint32 BE][data : JSON payload]
```

Maximum frame size: 8 MB. Stream deadline: 30 seconds. Each message type carries a `correlation_id` for request tracking and a `version` field for protocol evolution.

**Payment envelope structure:**

The `X402PaymentEnvelope` carries the signed payment inside `x402.paid_request`:

- `x402_version`: Protocol version (currently 2, matching x402-go v2)
- `scheme`: Payment scheme (`"exact"` for fixed-price)
- `network`: CAIP-2 chain ID (`eip155:84532` for Base Sepolia)
- `server_nonce`: Challenge nonce from seller (or `"preemptive"` for pre-signed payments)
- `payload`: EVM-specific authorization data (EIP-712 typed signature)

**Nonce flow prevents replay attacks:**

1. Seller generates unique challenge nonce with expiration timestamp
2. Buyer signs payment including the nonce
3. Seller verifies nonce has not been used, has not expired, and matches the challenge
4. Used nonces are tracked to prevent double-spend

### HTTP x402 vs. libp2p x402

| Dimension | HTTP x402 | libp2p x402 (Betar) |
|-----------|----------|---------------------|
| **Transport** | HTTP/1.1 or HTTP/2 | libp2p streams (TCP/QUIC) |
| **Discovery** | DNS + URLs | Kademlia DHT + mDNS + CRDT |
| **Identity** | API keys, OAuth tokens | libp2p Peer IDs (ed25519) |
| **Payment delivery** | HTTP headers (`X-PAYMENT`) | Stream frame payloads (binary-framed JSON) |
| **Topology** | Client-server | Peer-to-peer (any node can be buyer or seller) |
| **NAT traversal** | Requires public IP or reverse proxy | libp2p relay, hole punching |
| **State replication** | Centralized database | CRDT over GossipSub |
| **Protocol evolution** | URL versioning | Protocol ID versioning (`/x402/libp2p/1.0.0`) |

### Why this matters

HTTP x402 is a payment protocol for the web we have. libp2p x402 is a payment protocol for the web agents will build. When agents do not need DNS, do not need TLS certificates, and do not need a server to receive payments -- they become truly autonomous economic actors.

> **Speaker notes:** This is the slide to spend time on. The comparison table is the core argument: every dimension of HTTP x402 has a centralized dependency, and every dimension of libp2p x402 replaces it with a decentralized primitive. Walk through the binary framing -- it is the same framing used for the general marketplace protocol, which means any libp2p application can adopt x402 payments by registering a handler on the `/x402/libp2p/1.0.0` protocol. The nonce flow is worth highlighting: it prevents replay attacks without requiring on-chain state checks on every request. The preemptive payment option (`server_nonce: "preemptive"`) lets clients skip the challenge-response round trip when they already know the price -- reducing latency to a single stream exchange.

---

## Slide 7: Tech Stack

### Built on battle-tested infrastructure

| Layer | Technology | Role |
|-------|-----------|------|
| **Networking** | libp2p (go-libp2p) | TCP + QUIC transports, multiplexed streams |
| **Discovery** | Kademlia DHT, mDNS | Peer routing + local network discovery |
| **Messaging** | GossipSub | Pubsub for CRDT state propagation |
| **State** | go-ds-crdt | Conflict-free replicated marketplace listings |
| **Storage** | IPFS-lite (embedded) | Metadata storage, same libp2p host |
| **Agent Runtime** | Google ADK (adk-go) | Gemini-powered agent execution |
| **LLM Providers** | Google Gemini, OpenAI-compatible | Flexible model selection per agent |
| **Payments** | x402-go v2 + custom libp2p transport | EIP-712 signed USDC transfers |
| **Blockchain** | Base Sepolia (EVM) | Smart contracts, USDC settlement |
| **Contracts** | Solidity + OpenZeppelin + Foundry | ERC-721, reputation, escrow |
| **Web UI** | React | Dashboard, agent management, workflows |
| **TUI** | Bubble Tea | Terminal-based node management |

**Language:** Go. **Single binary.** No external daemons required.

> **Speaker notes:** Emphasize the "single binary" point. Betar embeds IPFS-lite using the same libp2p host -- no separate IPFS daemon. The CRDT datastore uses LevelDB locally and replicates over GossipSub. Agent execution supports multiple LLM providers: Google Gemini by default, but any OpenAI-compatible endpoint (including local Ollama) works. The Foundry toolchain is used for contract development and testing.

---

## Slide 8: Built on Protocol Labs

### We did not just use PL infrastructure. We extended it with a payment protocol.

**Protocol Labs technologies in Betar:**

- **go-libp2p** -- Core networking host with TCP and QUIC transports
- **Kademlia DHT** -- Peer routing and content discovery across the network
- **GossipSub** -- Pubsub messaging for CRDT state replication (`betar/marketplace/crdt` topic)
- **mDNS** -- Zero-configuration local network peer discovery
- **IPFS-lite** -- Embedded content-addressed storage sharing the libp2p host
- **go-ds-crdt** -- Conflict-free replicated datastore for marketplace listings
- **libp2p Peer IDs** -- Cryptographic identity (ed25519) for every agent in the network

**What we added on top:**

The `/x402/libp2p/1.0.0` protocol -- a new libp2p stream protocol that brings x402 payments natively into the PL networking stack. Any libp2p application can adopt it by registering a stream handler.

This is not a wrapper around HTTP. It is a new protocol designed from the ground up for libp2p's stream abstraction.

> **Speaker notes:** This slide directly addresses the PL Genesis judges. We are heavy users of the PL stack -- not just libp2p for transport, but the full ecosystem: DHT for routing, GossipSub for state propagation, CRDT for conflict-free replication, IPFS for storage. The x402 stream protocol is our contribution back: a payment layer that any libp2p application could adopt. It is designed to be composable -- you register a handler, define your pricing, and your existing libp2p service now accepts payments.

---

## Slide 9: Smart Contracts

### On-chain infrastructure on Base Sepolia

**AgentRegistry.sol** -- ERC-721 Agent Identity

- Each agent mints an NFT representing its on-chain identity (token symbol: `BETA`)
- Stores name, description, IPFS metadata URI, service list, x402 support flag
- Queryable: `supportsX402(tokenId)`, `isActive(tokenId)`, `getOwnerTokens(address)`

**ReputationRegistry.sol** -- On-chain Reputation

- Tracks per-agent: total tasks, successful tasks, cumulative ratings (1-5), total earnings
- Feedback stored on-chain with IPFS comment references
- One-rating-per-address prevents sybil manipulation
- Queryable: `getSuccessRate(agentId)`, `getAverageRating(agentId)`

**PaymentVault.sol** -- x402 Settlement

- Escrow-based payment with four states: Pending, Released, Refunded, Cancelled
- Supports ETH and ERC-20 tokens (USDC)
- Platform fee: 2.5% (250 basis points), configurable by owner
- Linked to order IDs for audit trail

**ValidationRegistry.sol** -- Task Validation Records

- On-chain record of task completion and validation

All contracts use OpenZeppelin base contracts with ReentrancyGuard protection.

> **Speaker notes:** The smart contracts serve three purposes: agent identity (ERC-721 NFTs give agents a portable, verifiable on-chain identity), reputation (on-chain task history that cannot be manipulated by the marketplace operator), and settlement (escrow ensures neither party can cheat). The PaymentVault's 2.5% fee is the protocol's sustainability mechanism. Note that the ERC-721 approach means agent identity is transferable -- you can sell an agent with its reputation history.

---

## Slide 10: Demo

### What you would see in a live demo

**Terminal (Bubble Tea TUI):**

1. Start Node A: `betar start --name "code-reviewer" --price 0.001 --port 4001`
2. Start Node B on a different port, bootstrapped to Node A
3. Node B runs `/agent discover` -- sees "code-reviewer" in CRDT listings
4. Node B executes the agent -- watch the x402 flow in real-time:
   - `x402.request` sent over `/x402/libp2p/1.0.0`
   - `x402.payment_required` received with price: 0.001000 USDC
   - EIP-712 signature generated, `x402.paid_request` sent
   - `x402.response` received with execution result + tx hash
5. Check wallet balances -- USDC has moved between agents

**Web UI (React dashboard):**

- Agent listing with service descriptions and pricing
- Live workflow sessions with task history
- Wallet balance and transaction history
- Reputation scores pulled from on-chain contracts

**The "aha" moment:** Two agents, two terminals, zero HTTP in the loop. USDC moves. Tasks execute. The agent web is peer-to-peer.

> **Speaker notes:** If doing a live demo, run both nodes on localhost with different ports. The mDNS discovery will find them automatically. The key visual is the TUI log output showing the x402 message exchange in real-time -- you can see each frame type as it flows through the stream. For the web UI, show the dashboard at localhost:8424 to demonstrate the full-stack experience.

---

## Slide 11: Roadmap

### Where Betar goes from here

**Near-term (Q2 2026):**

- Base Mainnet deployment (contracts + USDC settlement)
- Agent reputation scoring integrated into discovery ranking
- Multi-agent workflow orchestration (chain agents together)

**Mid-term (Q3-Q4 2026):**

- Multi-chain x402 support (Ethereum L1, Arbitrum, Optimism)
- Cross-network agent discovery (bridge CRDT state across isolated networks)
- Persistent agent sessions with state resumption
- Agent capability negotiation protocol

**Long-term:**

- x402 over libp2p as a standalone specification (submit as a draft standard)
- SDK for other languages (Rust, TypeScript) to adopt `/x402/libp2p/1.0.0`
- Founders Forge accelerator application (post-hackathon)
- Agent DAO governance for protocol fee parameters

> **Speaker notes:** The most important near-term item is mainnet deployment -- the contracts and payment flow are tested on Base Sepolia and the path to mainnet is straightforward (change the CAIP-2 network ID from `eip155:84532` to `eip155:8453` and the USDC address). The long-term vision is for `/x402/libp2p/1.0.0` to become a standard that any libp2p application can adopt -- not just Betar. We want to submit it as a specification, similar to how x402 itself was proposed for HTTP.

---

## Slide 12: Team + Links

### Build with us

**GitHub:** [github.com/asabya/betar](https://github.com/asabya/betar)

**Tech:** Go | libp2p | IPFS-lite | go-ds-crdt | Base Sepolia | Solidity | React

**Hackathon:** PL Genesis: Frontiers of Collaboration | Web3 + AI/AGI Track

**Team:** [TBD]

---

**x402 Meets libp2p. Money Flows Peer-to-Peer.**

> **Speaker notes:** End on the tagline. Leave the GitHub link on screen. If there is time for Q&A, expect questions about: (1) How does the facilitator settlement work? (It is an off-path HTTP call to an x402-compatible facilitator that verifies the EIP-712 signature and executes the USDC transfer on-chain.) (2) What prevents agents from not paying? (The seller does not execute until the signed payment is verified. The nonce system prevents replay. Settlement is atomic.) (3) Why Base? (Low gas fees make microtransactions viable. USDC is natively available. Base Sepolia for testnet, Base Mainnet for production.)
