# Betar Presenter Script — PL Genesis: Frontiers of Collaboration

**Total Runtime: ~2:45**
**Tracks: Web3 + AI/AGI | Category: Existing Code**
**Status: Beta — Running on Base Sepolia testnet**

> This is the voiceover/presenter script. For the detailed recording guide with
> commands, see [video-script.md](video-script.md).

---

## HOOK — 0:00-0:15

[Visual: Two animated AI agents on opposite sides of the screen, connected by a glowing P2P line. A centralized server appears between them, blinks red, and vanishes. The agents reconnect directly. Text fades in: "What happens when there's no server in the room?"]

What happens when AI agents need to pay each other — and there's no server in the room?

No API gateway. No custodial wallet. No middleman holding the keys.

Just two peers, a stream, and a signed USDC transfer.

---

## PROBLEM — 0:15-0:45

[Visual: Split screen. Left side: thriving AI agent ecosystem. Right side: payment infrastructure — a tangle of HTTP servers, API keys, OAuth tokens, centralized payment processors, all bottlenecking through a single point.]

AI agents are everywhere. They write code, analyze markets, generate content, run multi-step workflows. And they're getting good enough to hire each other.

But when Agent A needs to pay Agent B for a service, everything breaks down. Today's payment rails require HTTP servers, API keys, custodial wallets, centralized gateways. Every transaction routes through someone else's infrastructure.

[Visual: The centralized bottleneck diagram shatters. Text appears: "The agent web is peer-to-peer. The payment layer isn't."]

The agent web is peer-to-peer. The payment layer isn't.

Until now.

---

## SOLUTION — 0:45-1:30

[Visual: Betar logo animates in. Tagline: "A decentralized marketplace where agents discover, execute, and pay each other."]

This is Betar — a decentralized P2P marketplace where autonomous agents discover each other, negotiate services, and settle payments. All without a central server.

Let me show you.

[Visual: Terminal — `bin/betar start --name "code-reviewer" --price 0.001 --port 4001`. Show libp2p host bootstrapping, DHT connecting, peer ID printed.]

**Step 1.** Start a Betar node. A libp2p host spins up with TCP and QUIC transports, connects to the Kademlia DHT, and begins mDNS discovery.

[Visual: Second terminal, second node. Show CRDT replication log: agent listing appears on the other node via GossipSub topic `betar/marketplace/crdt`.]

**Step 2.** Register an agent. The listing replicates across the network through a CRDT over GossipSub. No registry server. No API call. Just eventual consistency across every peer.

[Visual: Web UI or TUI showing agent discovery. List of available agents with names, descriptions, prices.]

**Step 3.** Discover agents by querying the local CRDT state. Every node has a full view of the marketplace.

[Visual: Execute command. Show the P2P stream opening, task sent, 402 response, signed payment, result returning. Highlight message types scrolling in logs: `x402.request`, `x402.payment_required`, `x402.paid_request`, `x402.response`.]

**Step 4.** Execute a task. The buyer opens a libp2p stream to the seller, sends a request, and the x402 protocol takes over. The seller responds with a 402 challenge. The buyer signs a USDC authorization, resends with payment attached. The seller verifies, executes, and returns the result. Payment settles on-chain.

All of that happened over a single P2P stream. No HTTP involved.

---

## DEEP DIVE — 1:30-2:15

[Visual: Protocol diagram — five message types as arrows between two peer boxes.]

The key innovation is the x402 protocol running natively over libp2p.

Protocol ID: `/x402/libp2p/1.0.0`. Same 402 semantics you know from HTTP, but transported in binary-framed libp2p stream messages instead of HTTP headers.

[Visual: Wire format: `[type_len : uint16 BE][type : UTF-8][data_len : uint32 BE][data]`.]

The wire format is minimal. Two bytes for the message type length, the type string, four bytes for the data length, then the payload. Five typed messages handle the entire payment lifecycle: request, payment_required, paid_request, response, and error.

[Visual: `X402PaymentRequired` struct with `ChallengeNonce` and `PaymentRequirements`. Then `X402PaidRequest` with `X402PaymentEnvelope` containing `EVMPayload`.]

When a seller requires payment, it issues a challenge nonce and returns payment requirements — network, amount, asset, payee address. The buyer signs an EIP-712 typed USDC authorization, binding the exact amount and nonce. No tokens leave the buyer's wallet until the facilitator settles the meta-transaction on Base Sepolia.

[Visual: "Buyer signs EIP-712" → "Seller verifies locally" → "Facilitator settles on-chain via transferWithAuthorization". Facilitator box: "HTTP (only this step)". Everything else: "libp2p (P2P)".]

To be precise: discovery, execution, and payment negotiation are fully peer-to-peer over libp2p. The one HTTP call is facilitator settlement — submitting the signed authorization to the USDC contract on-chain. Everything else is agents and math.

---

## PROTOCOL LABS — 2:15-2:30

[Visual: Tech stack diagram. Bottom: libp2p (TCP, QUIC, mDNS, Kademlia DHT, GossipSub). Middle: IPFS-lite (embedded). Top: go-ds-crdt (marketplace state).]

Betar is built entirely on Protocol Labs primitives. libp2p for networking and discovery. Kademlia DHT for peer routing. GossipSub for CRDT replication. IPFS-lite for content storage, sharing the same libp2p host.

We didn't just use libp2p. We extended it with a payment protocol that makes agent commerce native to the peer-to-peer web.

---

## CLOSE — 2:30-2:45

[Visual: Tagline types out letter by letter: "x402 Meets libp2p. Money Flows Peer-to-Peer." Then fade in: GitHub URL, PL Genesis logo.]

x402 meets libp2p. Money flows peer-to-peer.

Built for PL Genesis.

---

## Elevator Pitch (30 seconds)

Betar is a decentralized agent-to-agent marketplace where AI agents discover each other over libp2p, execute tasks through direct P2P streams, and pay with USDC using the x402 payment protocol — all without a central server. We took the HTTP 402 "Payment Required" concept and rebuilt it as a native libp2p stream protocol with binary-framed messages and EIP-712 signed USDC authorizations. Discovery runs on Kademlia and CRDT-over-GossipSub. Execution is direct peer-to-peer. The only HTTP call in the entire flow is facilitator settlement on Base Sepolia. Everything else is agents and math.

---

## Key Talking Points

- **x402 over libp2p is the core innovation.** `/x402/libp2p/1.0.0` carries the full HTTP 402 payment flow — request, challenge, signed payment, response — in binary-framed stream messages. Not a wrapper around HTTP. A native P2P payment protocol.

- **Five message types handle the entire payment lifecycle.** `x402.request`, `x402.payment_required`, `x402.paid_request`, `x402.response`, `x402.error`. Each is a typed JSON payload inside a length-prefixed binary frame. Simple, extensible, auditable.

- **EIP-712 signed USDC meta-transactions.** Buyers sign a typed authorization that the facilitator submits on-chain via `transferWithAuthorization` (EIP-3009). The buyer controls the exact amount, recipient, and time window.

- **Built entirely on Protocol Labs primitives.** libp2p, IPFS-lite, go-ds-crdt. No external infrastructure beyond an Ethereum RPC.

- **Marketplace state is a CRDT, not a database.** Listings replicate across all peers via `go-ds-crdt` over `betar/marketplace/crdt`. Every node has a full, eventually-consistent view. No indexer. No API server.

- **Honest about architecture.** Discovery and execution are fully P2P. Facilitator settlement is the only centralized call. We are transparent about this.

- **Single binary.** `bin/betar start` launches a full node — embedded IPFS, libp2p host, agent runtime, Web UI. No external daemons.
