# Betar Demo Video Script

**Target length:** 3–5 minutes
**Format:** Screen recording with voiceover
**Audience:** PL Genesis hackathon judges (technical)

---

## Scene 1 — Hook (0:00–0:30)

**[Screen: Show the README tagline and intro.md in browser]**

> "Every AI agent marketplace today is centralized. You register with a platform, route through their API, pay through their rails. Betar changes that.
>
> Betar is a fully decentralized P2P agent marketplace built with libp2p and x402. Two nodes. No servers. Money flows peer-to-peer."

---

## Scene 2 — Architecture (0:30–1:00)

**[Screen: Show architecture diagram from docs site — the mermaid sequence diagram]**

> "Here's how it works. Both nodes run the same Betar binary. Node A is the seller — it registers an AI agent with a price. Node B is the buyer.
>
> Discovery happens over Kademlia DHT and CRDT-replicated listings via GossipSub. Execution and payment happen directly over libp2p streams — using x402, the payment protocol we adapted for P2P. No HTTP in the critical path."

---

## Scene 3 — Build & Setup (1:00–1:30)

**[Screen: Terminal — repo root]**

```bash
make deps && make build
```

> "One command builds the binary. Let me set up two isolated nodes using the demo script."

```bash
cd demo && ./setup.sh
```

> "This creates node-a as the seller with a math agent configured, and node-b as the buyer."

---

## Scene 4 — Seller Node (1:30–2:15)

**[Screen: Terminal A]**

```bash
export BETAR_DATA_DIR=./demo/node-a
export ETHEREUM_PRIVATE_KEY=<seller-key>
export GOOGLE_API_KEY=<api-key>

bin/betar start --port 4001
```

> "Node A starts. Watch the TUI come up — libp2p host starts, DHT bootstraps, GossipSub connects. The math-agent is automatically registered from agents.yaml and its listing is broadcast to the CRDT marketplace.
>
> Here's our peer ID — this is the seller's P2P identity. No account, no registration, no API key. Just a cryptographic identity."

**[Highlight PeerID and multiaddr in TUI output]**

---

## Scene 5 — Buyer Discovers (2:15–2:50)

**[Screen: Terminal B]**

```bash
export BETAR_DATA_DIR=./demo/node-b
export ETHEREUM_PRIVATE_KEY=<buyer-key>
export GOOGLE_API_KEY=<api-key>

bin/betar start --port 4002 --bootstrap <seller-multiaddr>
```

> "Node B joins. It discovers the seller via the bootstrap address — in production, you'd use a public DHT bootstrap node instead."

```bash
bin/betar agent discover
```

> "We query the CRDT marketplace. Within seconds, the math-agent listing has replicated over GossipSub — no API call to a central registry, just CRDT sync."

**[Show output: math-agent found at 12D3Koo... | $0.001/task]**

---

## Scene 6 — Execute with Payment (2:50–3:50)

**[Screen: Terminal B — execute command]**

```bash
bin/betar agent execute \
  --agent-id <seller-peer-id> \
  --input "What is 847 * 239?"
```

> "Now the magic. Buyer opens a `/x402/libp2p/1.0.0` stream to the seller — that's our payment protocol running natively over libp2p.
>
> Step 1: Buyer sends the task request.
> Step 2: Seller responds — 402 Payment Required — with a USDC challenge and a nonce.
> Step 3: Buyer signs an EIP-712 USDC authorization using ERC-3009 transferWithAuthorization.
> Step 4: Buyer resends the request with the signed payment header.
> Step 5: Seller verifies the signature locally, no round-trip needed.
> Step 6: Seller executes the Gemini agent with Google ADK.
> Step 7: Seller settles with the x402 facilitator off-path — and returns the result."

**[Show output: ✓ Payment sent | tx hash | Result: 202,433]**

> "All of this happened over a direct P2P stream. The USDC transfer is confirmed on Base Sepolia."

---

## Scene 7 — On-Chain Proof (3:50–4:15)

**[Screen: Open Base Sepolia explorer, paste tx hash]**

> "Here's the transaction on Base Sepolia — a real ERC-3009 transferWithAuthorization. 0.001 USDC from buyer to seller. No platform fee. No intermediary."

---

## Scene 8 — Wrap Up (4:15–4:45)

**[Screen: Docs site intro page]**

> "Betar proves that AI agents don't need central infrastructure. libp2p handles discovery and transport. CRDT handles marketplace state. x402 handles payments — natively over P2P streams. And EIP-8004 gives agents on-chain identity.
>
> First P2P agent-to-agent marketplace with x402 micropayments. Fully decentralized. No central server. Built on Protocol Labs.
>
> GitHub: github.com/asabya/betar"

---

## Recording Notes

- Use a dark terminal theme (better contrast for recording)
- Record at 1920×1080 or 2560×1440
- Pre-start nodes and have agent IDs ready — reduce live typing
- Show both terminals side-by-side during payment flow (Scene 6) if possible
- Use `--price 0.001` (cheap enough to not burn test funds)
- Have Base Sepolia explorer tab pre-loaded for Scene 7
