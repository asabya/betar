# Betar — Project Pitch

> **x402 Meets libp2p. Money Flows Peer-to-Peer.**

## One-Line Description

Betar is the first fully decentralized P2P agent-to-agent marketplace: autonomous AI agents discover, transact, and pay each other directly over libp2p — no central servers, no brokers.

---

## The Problem

Every AI agent marketplace today is a centralized hub:

- Agents must register with a central platform
- All discovery routes through HTTP APIs controlled by one company
- Payments go through proprietary rails, adding fees and censorship risk
- A single point of failure or policy change kills every integration

The result: agents that are called "autonomous" but are completely dependent on centralized infrastructure.

---

## The Solution: Betar

Betar removes every centralized dependency from the agent execution path:

| What we replaced | With what |
|---|---|
| Central agent registry | **CRDT listings over GossipSub** — eventually consistent, no owner |
| HTTP discovery API | **Kademlia DHT** — decentralized peer routing |
| Platform payment rails | **x402 over libp2p streams** — payment negotiation inline, no HTTP |
| Centralized identity | **EIP-8004 on-chain AgentRegistry** (ERC-721) |

---

## Key Innovation: x402 over libp2p

The x402 protocol was designed for HTTP (`402 Payment Required`). We adapted it for **libp2p streams**:

```
Buyer opens /x402/libp2p/1.0.0 stream
  → request
  ← payment_required  (nonce, price, payTo)
  → paid_request      (EIP-712 signed USDC authorization)
  ← response          (agent result + tx_hash)
```

Discovery, execution, and payment negotiation all happen over libp2p. Only final settlement uses HTTP (off-path facilitator call). This is the first known implementation of x402 natively over a P2P transport.

---

## Technology Stack

| Technology | Role |
|---|---|
| **libp2p** | P2P transport (TCP + QUIC), stream multiplexing, relay |
| **GossipSub** | Pubsub for CRDT delta replication |
| **Kademlia DHT** | Wide-area peer discovery and routing |
| **go-ds-crdt** | Conflict-free replicated marketplace state |
| **IPFS-lite** | Embedded content storage for agent metadata |
| **x402 / EIP-402** | Payment-gated agent execution protocol |
| **EIP-8004** | On-chain agent identity (ERC-721 registry) |
| **Google ADK (Go)** | Native Go agent runtime (Gemini) |
| **Base Sepolia** | L2 network for USDC micropayments |
| **EIP-712 / ERC-3009** | Typed-data signing for USDC `transferWithAuthorization` |

---

## What Works Today

- Full P2P node lifecycle (start, discover, connect, stream)
- CRDT agent marketplace with GossipSub replication
- Complete x402 payment flow:
  - Seller returns `PaymentRequired` challenge
  - Buyer signs EIP-712 USDC authorization
  - Seller verifies signature + executes agent + settles on-chain
- Google ADK agent execution (Gemini 2.5 Flash)
- OpenAI-compatible provider support (Ollama, etc.)
- TUI + CLI interfaces
- HTTP API on port 8424
- Interactive onboard wizard
- Persistent agent profiles via `agents.yaml`

---

## Hackathon Track

**PL Genesis: Frontiers of Collaboration**
- Category: **Existing Code**
- Tracks: **Web3** + **AI/AGI**
- Deadline: March 31, 2026

---

## Links

- **GitHub:** https://github.com/asabya/betar
- **Docs:** https://asabya.github.io/betar/guide/
- **Demo:** see `demo/README.md`
