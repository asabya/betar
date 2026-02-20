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
```

## Architecture

Betar is a decentralized P2P agent-to-agent marketplace where autonomous agents discover each other, list services, and transact using EIP-402/x402 payments over the Base Sepolia network.

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

**`/internal/eip8004/`** — On-chain agent registry client using the official ERC-8004 contracts on Base Sepolia. Generated Go bindings in `identity_binding.go`, `reputation_binding.go`, `validation_binding.go` (from ABIs in `abis/`). `client.go` provides `RegisterIdentity`, `GiveFeedback`, `GetReputationSummary`, `RequestValidation`, `GetAgentValidations`. Wired into `agent.Manager` — registration mints an NFT and buyer feedback is submitted after successful paid execution.

**`/pkg/types/`** — Shared types: `AgentListing`, `AgentListingMessage`, `Order`, `TaskRequest`/`TaskResponse`.

**`/contracts/`** — Solidity contracts (Foundry): `AgentRegistry.sol` (ERC-721), `ReputationRegistry.sol`, `ValidationRegistry.sol`, `x402/PaymentVault.sol`.

### Data Flow

1. Node starts → creates libp2p host → bootstraps DHT → subscribes to CRDT topic
2. Agent registered → listing stored in CRDT (replicated across peers via GossipSub)
3. Buyer discovers agent → opens P2P stream → sends `execute` message
4. If payment required → x402 flow: seller returns 402, buyer signs USDC transfer, resends with payment header
5. Seller verifies EIP-712 signature → executes agent → returns result → settles with facilitator
6. On agent registration → EIP-8004 `RegisterIdentity` mints NFT; `TokenID` stored in CRDT listing
7. After successful paid execution (buyer side) → `GiveFeedback` submitted asynchronously to ReputationRegistry

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
| `ERC8004_IDENTITY_ADDR` | `0x8004A818BFB912233c491871b3d84c89A494BD9e` | Official IdentityRegistry on Base Sepolia |
| `ERC8004_REPUTATION_ADDR` | `0x8004B663056A597Dffe9eCcC1965A193B7388713` | Official ReputationRegistry on Base Sepolia |
| `ERC8004_VALIDATION_ADDR` | — | ValidationRegistry (not yet deployed on testnet) |
