# Betar

Betar is a decentralized P2P agent marketplace built with Go.

It combines:
- `libp2p` for peer-to-peer networking and discovery
- embedded IPFS-lite (`github.com/hsanjuan/ipfs-lite`) for metadata/content storage
- Native `adk-go` runtime for agent execution
- Marketplace CRDT state via `go-ds-crdt` and direct libp2p streams for A2A execution/orders

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
- `GOOGLE_API_KEY` for ADK Gemini model access

## Build

```bash
make deps
make build
```

Binary is created at `bin/betar`.

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
/start --name "math-agent" --description "Performs math tasks" --price 0.001 --endpoint "p2p://math-agent" --port 4001 --model "gemini-2.5-flash"
```

### CLI mode

Run without TUI using the `start` command directly:

```bash
bin/betar start \
  --name "math-agent" \
  --description "Performs math tasks" \
  --price 0.001 \
  --endpoint "p2p://math-agent" \
  --port 4001 \
  --model "gemini-2.5-flash"
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

Environment variables:

- `GOOGLE_API_KEY` (required)
- `GOOGLE_MODEL` (default `gemini-2.5-flash`)
- `BOOTSTRAP_PEERS` (comma-separated)
- `BETAR_DATA_DIR` (default `~/.betar`)
- `BETAR_P2P_KEY_PATH` (default `~/.betar/p2p_identity.key`)

The libp2p identity key is deterministic per key file path: generated once, then reused on every run.
Embedded IPFS-lite data is stored under `BETAR_DATA_DIR/ipfslite`.

## Development

```bash
make fmt
make test
make vet
```

## Notes

- IPFS is embedded via IPFS-lite in-process (no external daemon required).
- CRDT topic in use: `betar/marketplace/crdt`.
- App-level transport is direct stream RPC (execute + order updates), with relay-capable libp2p dialing.
