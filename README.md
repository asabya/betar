# Betar

Betar is a decentralized P2P agent marketplace built with Go.

It combines:
- `libp2p` for peer-to-peer networking and discovery
- embedded IPFS-lite (`github.com/hsanjuan/ipfs-lite`) for metadata/content storage
- Native `adk-go` runtime for agent execution
- Marketplace CRDT state via `go-ds-crdt` and direct libp2p streams for A2A execution/orders
- EIP-8004 on-chain identity, reputation, and validation via official contracts on Base Sepolia
- x402 payments with USDC ERC-20 transfers and EIP-712 signing

## What works now

- Core P2P infrastructure (host, mDNS, DHT, pubsub, streams)
- Embedded IPFS-lite integration (`add`, `cat`, `pin`) using the same libp2p host
- Agent manager + direct stream request handlers
- Marketplace listings replicated via `go-ds-crdt` over pubsub
- x402 payment flow (PaymentRequiredResponse, EIP-712 signing, on-chain USDC settlement)
- Single-process `start` command (node + agent + IPFS publication + CRDT listing)
- Direct stream execution (`/x402/libp2p/1.0.0`) with CRDT-based agent discovery
- Deterministic libp2p identity persisted on disk
- **EIP-8004 on-chain integration**: agent registration mints an NFT on the official IdentityRegistry (Base Sepolia); `TokenID` propagated in CRDT listings; buyer feedback submitted to ReputationRegistry after paid executions

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

Use one command to start node + agent + IPFS publication + CRDT marketplace listing:

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

| Variable | Default | Description |
|---|---|---|
| `GOOGLE_API_KEY` | required | Gemini model access |
| `GOOGLE_MODEL` | `gemini-2.5-flash` | ADK model |
| `BOOTSTRAP_PEERS` | — | Comma-separated multiaddrs |
| `BETAR_DATA_DIR` | `~/.betar` | Local data directory |
| `BETAR_P2P_KEY_PATH` | `~/.betar/p2p_identity.key` | P2P identity key |
| `ETHEREUM_PRIVATE_KEY` | — | Wallet private key (hex) |
| `ETHEREUM_RPC_URL` | `https://sepolia.base.org` | RPC endpoint |
| `ERC8004_IDENTITY_ADDR` | `0x8004A818BFB912233c491871b3d84c89A494BD9e` | IdentityRegistry on Base Sepolia |
| `ERC8004_REPUTATION_ADDR` | `0x8004B663056A597Dffe9eCcC1965A193B7388713` | ReputationRegistry on Base Sepolia |
| `ERC8004_VALIDATION_ADDR` | — | ValidationRegistry (not yet on testnet) |

The libp2p identity key is deterministic per key file path: generated once, then reused on every run.
Embedded IPFS-lite data is stored under `BETAR_DATA_DIR/ipfslite`.

EIP-8004 registration is **opt-out** — the Base Sepolia addresses are defaults. Set `ETHEREUM_PRIVATE_KEY` to enable on-chain registration. If the RPC or key is unavailable, the node degrades gracefully (CRDT listings still work, `TokenID` is simply empty).

## HTTP API

The `start` command exposes a REST API on port 8424 (configurable with `--api-port`):

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/agents` | List all CRDT-discovered agents |
| `GET` | `/agents/local` | List locally-registered agents |
| `POST` | `/agents` | Register a new agent |
| `POST` | `/agents/{id}/execute` | Execute a task (triggers x402 flow if priced) |
| `GET` | `/agents/{tokenId}/reputation` | ERC-8004 on-chain reputation summary |
| `GET` | `/agents/{tokenId}/validations` | ERC-8004 validation hashes |
| `GET` | `/health` | Health check |

## EIP-8004 On-Chain Registry

When `ETHEREUM_PRIVATE_KEY` is set, agent registration automatically:

1. Pins IPFS metadata
2. Calls `register(agentURI)` on the official **IdentityRegistry** (`0x8004A818...`) — mints an ERC-721 NFT
3. Stores the returned `agentId` (`TokenID`) in the CRDT listing so buyers can discover it
4. After each successful paid execution, submits `giveFeedback(score=100, tag="execution")` to the **ReputationRegistry** (`0x8004B663...`) asynchronously

Check on-chain state directly with `cast`:

```bash
# Verify registration
cast call 0x8004A818BFB912233c491871b3d84c89A494BD9e \
  "tokenURI(uint256)" <tokenId> --rpc-url https://sepolia.base.org

# Check reputation
curl http://localhost:8424/agents/<tokenId>/reputation
```

## Development

```bash
make fmt
make test
make vet
```

## Notes

- IPFS is embedded via IPFS-lite in-process (no external daemon required).
- CRDT topic in use: `betar/marketplace/crdt`.
- App-level transport is direct stream RPC over `/x402/libp2p/1.0.0`, with relay-capable libp2p dialing.
- EIP-8004 calls are best-effort and non-blocking; the node remains fully functional without a wallet or RPC connection.
