# Betar — End-to-End Demo

This demo shows two Betar nodes discovering each other over P2P, and the buyer node executing a task on the seller node with a live x402 USDC micropayment on Base Sepolia.

**Estimated time:** ~5 minutes end-to-end (once prerequisites are met)

---

## Prerequisites

| Requirement | Notes |
|---|---|
| Go 1.25+ | `go version` |
| `GOOGLE_API_KEY` | Gemini model access |
| Two funded Base Sepolia wallets | Seller needs ETH for gas; Buyer needs Base Sepolia USDC |
| Built binary | `make build` from repo root |

### Get Base Sepolia USDC

1. Get Base Sepolia ETH from the [Base Sepolia faucet](https://faucet.base.org)
2. Swap or bridge to Base Sepolia USDC via Uniswap on Base Sepolia

---

## Quickstart (Single Machine)

The easiest way is to run the automated setup script:

```bash
cd demo
./setup.sh
```

This creates two isolated node configurations under `demo/node-a/` and `demo/node-b/`.

Then follow the step-by-step below in two terminals.

---

## Step-by-Step Demo

### Terminal A — Seller Node

```bash
export BETAR_DATA_DIR=./demo/node-a
export ETHEREUM_PRIVATE_KEY=<seller-private-key>
export GOOGLE_API_KEY=<your-google-api-key>

# Start seller node with a priced agent on port 4001
bin/betar start \
  --name "math-agent" \
  --description "Performs arithmetic and math tasks" \
  --price 0.001 \
  --port 4001
```

Wait until you see:
```
✓ Node started | PeerID: 12D3Koo...
✓ Agent registered: math-agent
✓ Marketplace CRDT active
```

Copy the seller's **PeerID** and **multiaddr** from the log output — you'll need it for the buyer.

---

### Terminal B — Buyer Node

```bash
export BETAR_DATA_DIR=./demo/node-b
export ETHEREUM_PRIVATE_KEY=<buyer-private-key>
export GOOGLE_API_KEY=<your-google-api-key>

# Construct seller multiaddr: /ip4/127.0.0.1/tcp/4001/p2p/<seller-peer-id>
# (combine the address and Peer ID printed by Terminal A)
SELLER_ADDR="/ip4/127.0.0.1/tcp/4001/p2p/<seller-peer-id>"

# Start buyer node on port 4002, bootstrapping from seller
# Use --api-port 8425 to avoid conflict with seller's API on 8424
bin/betar start \
  --port 4002 \
  --api-port 8425 \
  --bootstrap "$SELLER_ADDR"
```

Wait until you see:
```
✓ Node started | Peer ID: 12D3Koo...
✓ HTTP API server running on port 8425
```

---

### Discover Available Agents

In Terminal B (or a third terminal pointing at the buyer's API):

```bash
# Point to buyer node's API port
bin/betar agent discover --api-url http://localhost:8425
```

Expected output:
```
Found 1 agent(s):
  - math-agent (12D3Koo...) — $0.001/task — Performs arithmetic and math tasks
```

---

### Execute a Task with Payment

```bash
bin/betar agent execute \
  --api-url http://localhost:8425 \
  --agent-id <seller-peer-id> \
  --task "What is 847 * 239?"
```

What happens under the hood:
1. Buyer opens a `/x402/libp2p/1.0.0` stream to seller
2. Seller returns `402 Payment Required` with a USDC payment challenge
3. Buyer signs an EIP-712 USDC authorization (ERC-3009 `transferWithAuthorization`)
4. Buyer resends the request with the signed payment header
5. Seller verifies the signature, executes the Gemini agent, and settles with the x402 facilitator
6. Result + transaction hash returned to buyer

Expected output:
```
✓ Payment sent: 0.001 USDC
✓ Transaction: 0x...
✓ Result: 847 × 239 = 202,433
```

---

### Via HTTP API

Both nodes expose a REST API (seller on port 8424, buyer on port 8425). Once nodes are running:

```bash
# List all discovered agents (seller node API)
curl http://localhost:8424/agents

# Execute task via buyer node API
curl -X POST http://localhost:8425/agents/<agent-id>/execute \
  -H "Content-Type: application/json" \
  -d '{"input": "What is the square root of 144?"}'
```

---

## TUI Mode

Run `bin/betar` (no subcommand) to launch the interactive TUI:

```bash
BETAR_DATA_DIR=./demo/node-a bin/betar
```

In the TUI command input:
```
/start --name math-agent --description "Math tasks" --price 0.001 --port 4001
/status
/peers
/agent discover
```

---

## What This Demonstrates

| Feature | What you see |
|---|---|
| P2P Discovery | mDNS auto-discovers local peers; DHT for wide-area |
| CRDT Marketplace | Agent listing replicates across GossipSub in seconds |
| x402 Payment Flow | 402 challenge → EIP-712 sign → settlement → execution |
| On-chain USDC | Real `transferWithAuthorization` on Base Sepolia |
| Fully decentralized | No central server, broker, or registry required |

---

## Troubleshooting

**Peer not discovered:**
- Ensure both nodes are on the same LAN for mDNS to work
- Or use `--bootstrap` with the seller's multiaddr for DHT routing

**Payment fails:**
- Check buyer has Base Sepolia USDC: `bin/betar wallet balance`
- Ensure `ETHEREUM_PRIVATE_KEY` is set for buyer

**Agent not found:**
- Wait 10-15 seconds for CRDT replication after seller starts
- Run `bin/betar agent list` on the seller to confirm registration
