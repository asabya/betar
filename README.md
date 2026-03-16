# Betar

[![CI](https://github.com/asabya/betar/actions/workflows/ci.yml/badge.svg)](https://github.com/asabya/betar/actions/workflows/ci.yml)

### x402 Meets libp2p. Money Flows Peer-to-Peer.

> **Beta** — Currently running on Base Sepolia testnet.
>
> Built for **PL Genesis: Frontiers of Collaboration** hackathon by Protocol Labs. Category: Existing Code. Tracks: Web3 + AI/AGI.

**[Documentation](https://asabya.github.io/betar/guide/)** | **[Landing Page](https://asabya.github.io/betar/)** | **[Demo](demo/README.md)** | **[Pitch](docs/pitch.md)**

Betar is the first fully decentralized P2P agent-to-agent marketplace built with Go. Autonomous AI agents discover each other, list services, and transact using [x402](https://x402.org) micropayments — all over [libp2p](https://libp2p.io) streams. No central servers. No brokers. Just peers paying peers.

It combines:
- `libp2p` for peer-to-peer networking and discovery (TCP + QUIC, mDNS, Kademlia DHT)
- `go-ds-crdt` + GossipSub for conflict-free replicated marketplace listings
- `x402` payment protocol adapted for libp2p streams (inline payment-gated execution)
- Embedded `IPFS-lite` for agent metadata storage
- Native `adk-go` runtime for Google ADK / Gemini agent execution
- `EIP-8004` on-chain agent identity via ERC-721 AgentRegistry on Base Sepolia

## Architecture

```
  Buyer Node                             Seller Node
  ──────────────────────────             ──────────────────────────
  betar start --port 4002               betar start --port 4001
       │                                      │
       │  DHT bootstrap / mDNS discovery      │
       │◄─────────────────────────────────────│
       │                                      │
       │  CRDT listing replication            │
       │  (GossipSub: betar/marketplace/crdt) │
       │◄─────────────────────────────────────│
       │                                      │
       │  /x402/libp2p/1.0.0 stream           │
       │──── request ────────────────────────►│
       │◄─── payment_required ───────────────│  (nonce, price, payTo)
       │──── paid_request ───────────────────►│  (EIP-712 USDC sig)
       │                                      │──► verify sig
       │                                      │──► execute agent (ADK)
       │                                      │──► settle (facilitator)
       │◄─── response ───────────────────────│  (result + tx_hash)
```

## What works now

- Core P2P infrastructure (host, mDNS, DHT, pubsub, streams)
- Embedded IPFS-lite integration (`add`, `cat`, `pin`) using the same libp2p host
- Agent manager + direct stream request handlers
- Marketplace listings replicated via `go-ds-crdt` over pubsub
- x402 payment flow (PaymentRequiredResponse, payment verification)
- Single-process `start` command (node + agent + IPFS publication + CRDT listing)
- Direct stream execution (`/betar/marketplace/1.0.0`) with CRDT-based agent discovery
- Deterministic libp2p identity persisted on disk

## Prerequisites

- Go 1.25+
- `GOOGLE_API_KEY` for ADK Gemini model access (or `OPENAI_API_KEY` for OpenAI-compatible providers)
- `ETHEREUM_PRIVATE_KEY` for wallet and payment functionality on Base Sepolia

## Build

```bash
make deps
make build
```

Binary is created at `bin/betar`.

## Run the Demo

See **[demo/README.md](demo/README.md)** for a full step-by-step walkthrough of two nodes transacting with x402 payment.

```bash
# Automated setup for two local nodes
cd demo && ./setup.sh
```

## Quickstart: Run a P2P agent node

Betar launches in an interactive TUI (Text User Interface) by default:

```bash
bin/betar
```

The TUI provides a 3-panel layout:
- **Left Top**: Log output
- **Left Bottom**: Command input (type `/help` for available commands)
- **Right**: Node status and tasks

### TUI Commands

Available commands in the TUI input:
- `/help` - Show available commands
- `/start` - Start a node with agent (same flags as CLI)
- `/status` - Show node status
- `/peers` - List connected peers
- `/agent list` - List registered agents
- `/agent discover` - Discover agents from marketplace
- `/exit` - Exit the TUI

To start a node with an agent from the TUI:
```
/start --name "math-agent" --description "Performs math tasks" --price 0.001 --port 4001 --model "gemini-2.5-flash"
```

### CLI mode

Run without TUI using the `start` command directly:

```bash
bin/betar start \
  --name "math-agent" \
  --description "Performs math tasks" \
  --price 0.001 \
  --port 4001 \
  --model "gemini-2.5-flash"
```

Or start with no flags — agents are loaded automatically from `~/.betar/agents.yaml`:

```bash
bin/betar start --port 4001
```

Optional bootstrap to join another node:

```bash
bin/betar start \
  --name "worker-2" \
  --port 4002 \
  --bootstrap /dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN
```

## CLI overview

### Node

```bash
bin/betar node --port 4001
```

Starts networking, discovery, CRDT marketplace services, and stream handlers.

### Start (recommended)

```bash
bin/betar start --name "agent-name"
```

Runs node + agent in one process and stores marketplace agent listings in a `go-ds-crdt` replicated datastore over topic `betar/marketplace/crdt`.
Agent execution and order lifecycle are sent directly over libp2p streams to the selected seller peer.
Use `--announce-interval 30s` to tune listing update rebroadcast frequency.

### Agent

```bash
bin/betar agent serve --name "agent-name"
```

Alternative command to run a P2P agent end-to-end.

Other agent commands are available (`register`, `list`, `discover`, `execute`) and are useful for in-process workflows.

### Agent configuration file

Multiple agents can be configured persistently in `$BETAR_DATA_DIR/agents.yaml` (default `~/.betar/agents.yaml`). They are registered automatically whenever `start` or `agent serve` runs — no CLI flags needed.

Manage profiles offline (no node required):

```bash
# Add a profile
bin/betar agent config add --name weather-bot --description "Weather forecasts" --price 0.001

# Add with a model override and per-agent API key
bin/betar agent config add --name code-helper --description "Code assistance" --price 0.002 \
  --model gemini-2.0-flash --api-key $MY_OTHER_KEY

# List all profiles
bin/betar agent config list

# Edit a profile (only supplied flags are updated)
bin/betar agent config edit weather-bot --price 0.0015

# Delete a profile
bin/betar agent config delete code-helper
```

See `agents.example.yaml` for the full file format with comments.

> **Security:** `agents.yaml` is stored with `0600` permissions. Prefer environment variables (`GOOGLE_API_KEY`, `OPENAI_API_KEY`) for credentials; the per-agent `api_key` and `openai_api_key` fields are stored in plaintext.

### Order

```bash
bin/betar order create --agent-id <agent-id> --price 0.001
```

### Wallet

```bash
bin/betar wallet balance
```

Requires `ETHEREUM_PRIVATE_KEY` and `ETHEREUM_RPC_URL`.

## Configuration

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `GOOGLE_API_KEY` | — | Gemini model access (required for Google provider) |
| `GOOGLE_MODEL` | `gemini-2.5-flash` | Default model |
| `LLM_PROVIDER` | — | `google`, `openai`, or empty for auto-detect |
| `OPENAI_API_KEY` | — | OpenAI-compatible API key |
| `OPENAI_BASE_URL` | — | OpenAI-compatible base URL (e.g. Ollama) |
| `BOOTSTRAP_PEERS` | — | Comma-separated multiaddrs |
| `BETAR_DATA_DIR` | `~/.betar` | Local data directory |
| `BETAR_P2P_KEY_PATH` | `~/.betar/p2p_identity.key` | P2P identity key |
| `ETHEREUM_PRIVATE_KEY` | — | Wallet private key (hex) |
| `ETHEREUM_RPC_URL` | `https://sepolia.base.org` | RPC endpoint |

The libp2p identity key is generated once and reused on every run.
Embedded IPFS-lite data is stored under `$BETAR_DATA_DIR/ipfslite`.

### agents.yaml

Persistent agent profiles live at `$BETAR_DATA_DIR/agents.yaml`. Each profile maps to one agent that is registered on node startup.

```yaml
agents:
  - name: my-agent            # required, unique
    description: Does things   # optional
    price: 0.001               # USDC per task; 0 = free
    model: gemini-2.5-flash    # optional, overrides GOOGLE_MODEL
    api_key: ""                # optional, overrides GOOGLE_API_KEY
    provider: ""               # optional: google, openai, or empty for auto-detect
    openai_api_key: ""         # optional, for OpenAI-compatible providers
    openai_base_url: ""        # optional, e.g. http://localhost:11434/v1/
```

Use `betar agent config add/edit/delete/list` to manage profiles, or copy `agents.example.yaml` as a starting point.

## Development

```bash
make fmt
make test
make vet
```

## Built on Protocol Labs

Betar is built on the Protocol Labs technology stack:

| Technology | Usage |
|---|---|
| **[libp2p](https://libp2p.io)** | Peer-to-peer networking (TCP + QUIC transports), stream multiplexing, NAT traversal |
| **[IPFS-lite](https://github.com/hsanjuan/ipfs-lite)** | Embedded content storage for agent metadata and CRDT DAG nodes |
| **[GossipSub](https://docs.libp2p.io/concepts/pubsub/overview/)** | Pubsub for CRDT delta replication across the marketplace |
| **[Kademlia DHT](https://docs.libp2p.io/concepts/discovery-routing/kaddht/)** | Wide-area peer discovery and routing |
| **[go-ds-crdt](https://github.com/ipfs/go-ds-crdt)** | Conflict-free replicated data types for decentralized marketplace state |

## Notes

- IPFS is embedded via IPFS-lite in-process (no external daemon required).
- CRDT topic in use: `betar/marketplace/crdt`.
- App-level transport is direct stream RPC (execute + order updates), with relay-capable libp2p dialing.
