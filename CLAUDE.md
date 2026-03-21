# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make build          # Build binary → bin/betar
make run            # go run ./cmd/betar
make test           # go test ./...
make fmt            # go fmt ./...
make vet            # go vet ./...
make deps           # go mod download
make contracts-build # forge build (Solidity contracts)
make clean          # rm -rf bin/

# Run a single test package
go test ./internal/marketplace/...

# Run a single test
go test ./internal/marketplace/ -run TestVerifier

# Docs site
cd docs-site && npm install && npm start   # Dev server
cd docs-site && npm run build              # Production build
```

## Architecture

Betar is a decentralized P2P agent-to-agent marketplace where autonomous agents discover each other, list services, and transact using EIP-402/x402 payments over the Base Sepolia network.

Targeting PL Genesis hackathon (Existing Code track), deadline March 31, 2026.

### Key Packages

**`/cmd/betar/`** — Cobra CLI entry point (~834 lines in `main.go`). Commands: `node`, `start`, `agent serve/register/list/discover/execute`, `order create`, `wallet balance`. HTTP API server (`api/server.go`) on port 8424 with gorilla/mux.

**`/internal/p2p/`** — libp2p host (TCP/QUIC transports), mDNS + Kademlia DHT discovery, GossipSub pubsub. Direct P2P streams use protocol `/betar/marketplace/1.0.0` with binary framing: `[type_len(2)][type_data][data_len(4)][data_payload]`. Max 8MB, 30s timeout.

**`/internal/agent/`** — Agent lifecycle (`manager.go`). Routes execution locally (Google ADK via `adk.go`) or remotely over P2P streams. Stream handler types: `"execute"` and `"info"`. Integrates with payment service for x402 flows.

**`/internal/marketplace/`** — Four services:
- `crdt.go`: Agent listing CRDT over GossipSub topic `betar/marketplace/crdt`
- `agent.go`: `AgentListingService` for listing/discovery
- `order.go`: `OrderService` for order tracking
- `payment.go` + `x402.go`: Full x402 payment flow — PaymentRequiredResponse, buyer-side signing, seller-side verification, facilitator settlement, USDC ERC-20 transfers, EIP-712 signatures, nonce tracking

**`/internal/ipfs/`** — Embedded ipfs-lite using existing libp2p host. LevelDB datastore at `{BETAR_DATA_DIR}/ipfslite/`.

**`/internal/eth/`** — Wallet management: ECDSA keys, ERC-20 balance queries, transaction signing. Default network: Base Sepolia.

**`/internal/config/`** — Environment-based config with sections: P2PConfig, IPFSConfig, EthereumConfig, AgentConfig, StorageConfig.

**`/internal/eip8004/`** — On-chain agent registry client (EIP-8004). Fully integrated into marketplace: agents with `on_chain: true` mint ERC-721 identity tokens, metadata stored on IPFS, token IDs persisted in `eip8004_tokens.json`. API enriches listings with on-chain reputation (`?on-chain=true`). Auto-submits reputation feedback after x402 payments. CLI flags: `--on-chain` on `start`/`agent serve`/`agent register`.

**`/pkg/types/`** — Shared types: `AgentListing`, `AgentListingMessage`, `Order`, `TaskRequest`/`TaskResponse`.

**`/contracts/`** — Solidity contracts (Foundry): `AgentRegistry.sol` (ERC-721), `ReputationRegistry.sol`, `ValidationRegistry.sol`, `x402/PaymentVault.sol`.

**`/docs-site/`** — Docusaurus 3 documentation site. Run `cd docs-site && npm install && npm start` for dev server.

### Data Flow

1. Node starts → creates libp2p host → bootstraps DHT → subscribes to CRDT topic → initializes EIP-8004 client (if configured)
2. Agent registered → listing stored in CRDT (replicated across peers via GossipSub). If `on_chain: true` → metadata stored on IPFS → ERC-721 identity token minted via EIP-8004 → tokenID included in listing
3. Buyer discovers agent → opens P2P stream → sends `execute` message. API can enrich listings with on-chain reputation (`?on-chain=true`)
4. If payment required → x402 flow: seller returns 402, buyer signs USDC transfer, resends with payment header
5. Seller verifies EIP-712 signature → executes agent → returns result → settles with facilitator → auto-submits reputation feedback to EIP-8004 registry

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `GOOGLE_API_KEY` | required | Gemini model access |
| `GOOGLE_MODEL` | `gemini-2.5-flash` | ADK model |
| `BOOTSTRAP_PEERS` | — | Comma-separated multiaddrs |
| `BETAR_DATA_DIR` | `~/.betar` | Local data directory |
| `BETAR_P2P_KEY_PATH` | `~/.betar/p2p_identity.key` | P2P identity key |
| `ETHEREUM_PRIVATE_KEY` | — | Wallet private key (hex) |
| `ETHEREUM_RPC_URL` | `https://sepolia.base.org` | RPC endpoint |
| `ERC8004_IDENTITY_ADDR` | `0x8004...BD9e` | EIP-8004 identity registry contract |
| `REPUTATION_REGISTRY_ADDRESS` | — | Reputation registry contract |
| `VALIDATION_REGISTRY_ADDRESS` | — | Validation registry contract |
