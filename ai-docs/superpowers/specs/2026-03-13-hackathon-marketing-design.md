# Betar Hackathon Marketing Package — Design Spec

**Date**: 2026-03-13
**Hackathon**: PL Genesis: Frontiers of Collaboration
**Category**: Existing Code
**Tracks**: Web3 + AI/AGI
**Deadline**: March 31, 2026
**Judging**: April 1-3, 2026

## Catchphrase

**"x402 Meets libp2p. Money Flows Peer-to-Peer."**

## Core Selling Point

Betar implements x402 payments natively over libp2p streams — not HTTP. Agents discover each other via CRDT-replicated listings over GossipSub, execute tasks over direct P2P streams, and settle payments using EIP-712 signed USDC transfers embedded in the stream protocol (`/x402/libp2p/1.0.0`). No centralized server for discovery or execution — settlement uses an off-path facilitator service.

**Dual-protocol architecture**: `/betar/marketplace/1.0.0` handles general agent execution streams; `/x402/libp2p/1.0.0` adds payment-gated execution with typed x402 message flows. This shows protocol evolution from plain execution to paid execution.

This is submitted to Protocol Labs — the creators of libp2p — making the narrative especially strong.

## Deliverables

### 1. Documentation Site (Docusaurus)

**Tech**: Docusaurus 3, React, MDX
**Location**: `docs-site/` in repo root

**Structure**:
```
docs-site/
├── docusaurus.config.js
├── sidebars.js
├── docs/
│   ├── intro.md                — What is Betar, catchphrase, value prop
│   ├── getting-started.md      — Prerequisites, build, quickstart
│   ├── architecture/
│   │   ├── overview.md         — System architecture, data flow diagram
│   │   ├── p2p-layer.md        — libp2p host, DHT, mDNS, GossipSub, streams
│   │   ├── x402-payments.md    — x402 over libp2p deep-dive (THE differentiator)
│   │   └── crdt-marketplace.md — Agent listing CRDT, discovery, replication
│   ├── guides/
│   │   ├── register-agent.md   — Agent registration (CLI + agents.yaml)
│   │   ├── execute-agent.md    — Discovering and executing remote agents
│   │   └── payment-flow.md     — End-to-end x402 payment walkthrough
│   ├── contracts/
│   │   ├── agent-registry.md   — AgentRegistry.sol (ERC-721)
│   │   ├── reputation.md       — ReputationRegistry.sol
│   │   └── payment-vault.md    — PaymentVault.sol, x402 settlement
│   └── api-reference.md        — HTTP API endpoints (port 8424)
├── src/
│   └── pages/
│       └── index.tsx            — Landing page with hero, catchphrase, features
├── static/
│   └── img/                     — Architecture diagrams, logo
├── package.json
└── README.md
```

**Landing page sections**:
- Hero: "x402 Meets libp2p. Money Flows Peer-to-Peer." + one-line description
- Three feature cards: P2P Discovery, x402 Payments, Agent Marketplace
- Architecture diagram (Mermaid in MDX — Docusaurus supports it natively)
- Quick links to docs
- Built with Protocol Labs tech badge

**Key content priorities**:
- `x402-payments.md` is the most important page — explain the protocol, message types, EIP-712 flow, comparison to HTTP x402
- `overview.md` should have a clear Mermaid data flow diagram matching CLAUDE.md's 5-step flow
- All pages should reference the actual Go source paths for credibility

### 2. Pitch Deck (Markdown outline for slides)

**Location**: `docs/hackathon/pitch-deck.md`
**Format**: Markdown with slide separators, convertible to Google Slides / Keynote

**Slides**:

1. **Title Slide**
   - "Betar" — "x402 Meets libp2p. Money Flows Peer-to-Peer."
   - PL Genesis Hackathon, March 2026
   - Team name [TBD]

2. **The Problem**
   - AI agents are exploding but have no native payment rails
   - Current solutions: centralized APIs, custodial wallets, HTTP-dependent
   - Agents can't autonomously transact without a middleman

3. **The Insight**
   - Payments should be part of the communication protocol, not bolted on
   - If agents discover each other P2P, they should pay each other P2P
   - x402 (HTTP 402 Payment Required) already exists — but only for HTTP

4. **The Solution: Betar**
   - Decentralized P2P agent marketplace
   - Agents discover, execute, and pay each other over libp2p
   - x402 payment protocol adapted for libp2p streams

5. **How It Works**
   - Visual data flow: Node start → DHT bootstrap → CRDT listing → P2P discovery → Stream execution → x402 payment → Settlement
   - Emphasis: no HTTP servers for agent discovery or task execution (settlement uses an off-path facilitator)

6. **x402 over libp2p: The Innovation**
   - Protocol ID: `/x402/libp2p/1.0.0`
   - Core message types: request → payment_required → paid_request → response (+ error)
   - EIP-712 signed USDC transfers in stream frames
   - Binary framing: `[type_len(2)][type_data][data_len(4)][data_payload]`
   - Comparison table: HTTP x402 vs libp2p x402

7. **Tech Stack**
   - libp2p (TCP/QUIC, Kademlia DHT, GossipSub, mDNS)
   - go-ds-crdt for replicated marketplace state
   - IPFS-lite for metadata storage
   - Google ADK for agent execution
   - Solidity contracts on Base Sepolia
   - React web UI + Bubble Tea TUI

8. **Built on Protocol Labs**
   - libp2p host, DHT, GossipSub, pubsub — all PL primitives
   - IPFS-lite integration using same libp2p host
   - Extending PL infrastructure into AI agent commerce

9. **Smart Contracts**
   - AgentRegistry.sol — ERC-721 agent identity
   - ReputationRegistry.sol — On-chain reputation
   - ValidationRegistry.sol — Task validation
   - PaymentVault.sol — x402 settlement

10. **Demo**
    - TUI screenshot
    - Web UI screenshot
    - Two-node execution flow

11. **Roadmap**
    - Mainnet deployment
    - Multi-chain x402 support
    - Agent reputation scoring
    - Cross-network agent discovery

12. **Team + Links**
    - GitHub, docs site, contact
    - "x402 Meets libp2p. Money Flows Peer-to-Peer."

### 3. Video Script (2-3 minutes)

**Location**: `docs/hackathon/video-script.md`
**Tone**: Technical but accessible, confident, slightly irreverent
**Target**: PL Genesis judges familiar with libp2p/IPFS

**Script outline**:

**[HOOK — 0:00-0:15]**
"What happens when AI agents need to pay each other — and there's no server in the room?"

**[PROBLEM — 0:15-0:45]**
- AI agents are everywhere. They can write code, book flights, analyze data.
- But when one agent needs to hire another? It hits a wall.
- Today's payment rails need HTTP servers, API keys, custodial wallets.
- The agent web is peer-to-peer. The payment layer isn't. Until now.

**[SOLUTION — 0:45-1:30]**
- Betar: a decentralized marketplace where agents discover, execute, and pay each other.
- Demo walkthrough:
  1. Start a node — libp2p host bootstraps, connects to DHT
  2. Register an agent — listing replicates via CRDT over GossipSub
  3. Discover agents — query the CRDT, find services across the network
  4. Execute — open a P2P stream, send a task
  5. Pay — x402 kicks in: seller returns 402, buyer signs USDC transfer with EIP-712, resends with payment header
  6. Result — agent executes, returns result, payment settles

**[DEEP-DIVE — 1:30-2:15]**
- The x402 protocol over libp2p is the key innovation
- Protocol ID: `/x402/libp2p/1.0.0`
- Same 402 semantics as HTTP, but in libp2p stream frames
- Binary-framed messages: typed request/response pairs
- EIP-712 signatures for USDC authorization — meta-transaction via facilitator
- On-chain settlement via PaymentVault contract on Base Sepolia
- No HTTP for discovery or execution. Settlement via off-path facilitator. Just agents and math.

**[PROTOCOL LABS ALIGNMENT — 2:15-2:30]**
- Built entirely on Protocol Labs primitives: libp2p, Kademlia, GossipSub, IPFS-lite
- We didn't just use libp2p — we extended it with a payment protocol that makes agent commerce native to the P2P web

**[CLOSE — 2:30-2:45]**
- "x402 Meets libp2p. Money Flows Peer-to-Peer."
- GitHub link, docs site, built for PL Genesis

### 4. README.md Updates

- Add catchphrase "x402 Meets libp2p. Money Flows Peer-to-Peer." as subtitle after "# Betar"
- Add "Built for PL Genesis: Frontiers of Collaboration" note in intro
- Add link to docs site (once deployed)
- Add "Built on Protocol Labs" section listing libp2p, IPFS-lite, GossipSub
- Keep all existing content intact

### 5. CLAUDE.md Updates

- Add `docs-site/` package description: "Docusaurus 3 documentation site — `cd docs-site && npm install && npm start`"
- Add to Commands section: `cd docs-site && npm start` (dev server) and `cd docs-site && npm run build` (production build)
- Note hackathon context: "Targeting PL Genesis hackathon (Existing Code track), deadline March 31, 2026"

## Success Criteria

- Documentation site builds and serves locally
- Pitch deck covers all 12 slides with compelling narrative
- Video script is 2-3 minutes when read aloud
- Catchphrase appears consistently across all deliverables
- x402-over-libp2p is clearly positioned as the key innovation
- Protocol Labs alignment is emphasized throughout

## Non-Goals

- Actually recording the video (script only)
- Creating final slide visuals (outline only)
- Deploying docs site to production
- Writing new code features
