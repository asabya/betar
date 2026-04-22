# Betar — Copilot Instructions

## Project Overview
Betar is a decentralized P2P agent-to-agent marketplace where autonomous agents discover each other, list services, and transact using EIP-402/x402 payments over the **Base Sepolia** network. Built for the PL Genesis hackathon (Existing Code track).

**Language:** Go (backend) · Solidity (contracts) · TypeScript/React (docs-site)  
**Key libs:** libp2p, google-adk, gorilla/mux, go-ethereum, ipfs-lite, Foundry  
**HTTP API:** port `8424` (gorilla/mux)  
**P2P Protocol ID:** `/betar/marketplace/1.0.0`

---

## Common Commands
```bash
make build           # Build binary → bin/betar
make run             # go run ./cmd/betar
make test            # go test ./...
make fmt             # go fmt ./...
make vet             # go vet ./...
make deps            # go mod download
make contracts-build # forge build (Solidity contracts)
make clean           # rm -rf bin/

go test ./internal/marketplace/...          # single package
go test ./internal/marketplace/ -run TestVerifier  # single test

cd docs-site && npm install && npm start    # docs dev server
cd docs-site && npm run build              # docs production build
```

---

## Repository Layout
```
betar/
├── cmd/betar/          # Cobra CLI entry point (main.go ~834 lines)
├── internal/
│   ├── agent/          # Agent lifecycle, local/remote execution, ADK bridge
│   ├── config/         # Env-based config structs
│   ├── eip8004/        # On-chain ERC-721 agent identity registry client
│   ├── eth/            # Wallet, ECDSA keys, ERC-20 queries, tx signing
│   ├── ipfs/           # Embedded ipfs-lite node (LevelDB datastore)
│   ├── marketplace/    # CRDT listings, orders, x402 payment flow
│   └── p2p/            # libp2p host, mDNS, Kademlia DHT, GossipSub
├── pkg/types/          # Shared types (AgentListing, Order, TaskRequest, etc.)
├── contracts/          # Solidity (Foundry): registry + payment vault contracts
├── api/server.go       # HTTP API server
├── docs-site/          # Docusaurus 3 documentation site
└── CLAUDE.md           # Detailed guidance (source of truth for architecture)
```

---

## Key Packages

### `cmd/betar/`
Cobra CLI. Commands: `node`, `start`, `agent serve|register|list|discover|execute`, `order create`, `wallet balance`.  
Flag `--on-chain` available on `start`, `agent serve`, `agent register`.

### `internal/p2p/`
libp2p host with TCP/QUIC transports. Discovery via mDNS + Kademlia DHT. PubSub via GossipSub.  
Stream framing: `[type_len(2 bytes)][type_data][data_len(4 bytes)][data_payload]`  
Max message: **8 MB**, timeout: **30 s**.

### `internal/agent/`
- `manager.go` — Agent lifecycle.
- `adk.go` — Google ADK (Gemini) bridge.
- Routes execution **locally** (ADK) or **remotely** (P2P stream).
- Stream handler types: `"execute"` and `"info"`.
- Integrates with payment service for x402 flows.

### `internal/marketplace/`
| File | Purpose |
|------|---------|
| `crdt.go` | Agent listing CRDT over GossipSub topic `betar/marketplace/crdt` |
| `agent.go` | `AgentListingService` — listing & discovery |
| `order.go` | `OrderService` — order tracking |
| `payment.go` + `x402.go` | Full x402 flow (see below) |

### `internal/eip8004/`
On-chain agent registry client (EIP-8004).  
- Agents with `on_chain: true` mint **ERC-721** identity tokens.
- Metadata stored on IPFS; token IDs persisted in `eip8004_tokens.json`.
- API enriches listings with on-chain reputation via `?on-chain=true`.
- Auto-submits reputation feedback after successful x402 payments.

### `internal/eth/`
Wallet management: ECDSA key gen/load, ERC-20 balance queries, tx signing.  
Default network: **Base Sepolia**.

### `internal/ipfs/`
Embedded ipfs-lite using the existing libp2p host.  
Datastore: LevelDB at `{BETAR_DATA_DIR}/ipfslite/`.

### `internal/config/`
Env-based config with sections: `P2PConfig`, `IPFSConfig`, `EthereumConfig`, `AgentConfig`, `StorageConfig`.

### `pkg/types/`
Shared types: `AgentListing`, `AgentListingMessage`, `Order`, `TaskRequest`, `TaskResponse`.

### `contracts/`
Foundry project.
| Contract | Purpose |
|----------|---------|
| `AgentRegistry.sol` | ERC-721 agent identity tokens |
| `ReputationRegistry.sol` | On-chain reputation scores |
| `ValidationRegistry.sol` | Agent validation records |
| `x402/PaymentVault.sol` | x402 payment escrow/settlement |

---

## Data Flow (End-to-End)
1. **Node starts** → creates libp2p host → bootstraps DHT → subscribes to CRDT GossipSub topic → initialises EIP-8004 client (if configured).
2. **Agent registered** → listing stored in CRDT (replicated via GossipSub).  
   If `on_chain: true` → metadata pinned on IPFS → ERC-721 token minted → `tokenID` included in listing.
3. **Buyer discovers agent** → opens P2P stream → sends `"execute"` message.  
   API can enrich listings with on-chain reputation (`?on-chain=true`).
4. **Payment required** → x402 flow triggered (see below).
5. **Seller** verifies EIP-712 sig → executes agent → returns result → settles with facilitator → auto-submits reputation feedback to EIP-8004.

---

## x402 Payment Flow
1. Seller returns HTTP-like **402 PaymentRequiredResponse** containing:
   - `amount` (USDC, in smallest unit)
   - `payTo` (seller's Ethereum address)
   - `nonce` (UUID, for replay protection)
   - `deadline` (Unix timestamp)
2. Buyer signs USDC transfer using **EIP-712** structured data over P2P stream.
3. Buyer resends `execute` request with signed payment header (`X-Payment`).
4. Seller verifies:
   - EIP-712 signature validity
   - Nonce not previously used (in-memory nonce store)
   - Deadline not expired
5. Seller executes agent task via ADK (local) or P2P (remote).
6. Seller settles with **facilitator** → USDC ERC-20 `transferFrom` on Base Sepolia.
7. Reputation feedback auto-submitted to EIP-8004 registry on settlement success.

### Payment Types
| Type | Struct | Location |
|------|--------|----------|
| Payment request | `PaymentRequiredResponse` | `internal/marketplace/x402.go` |
| Payment header | `PaymentHeader` | `internal/marketplace/x402.go` |
| Verifier | `PaymentVerifier` | `internal/marketplace/payment.go` |
| Order tracking | `OrderService` | `internal/marketplace/order.go` |

---

## Agent Task Execution

Agent tasks are executed via two paths depending on whether the target agent is local or remote.

### Local Execution (ADK)
- Handled by `internal/agent/adk.go` via the **Google ADK (Gemini)** bridge.
- Triggered when the requested agent is registered on the **same node**.
- Flow:
  1. `manager.go` identifies the agent as local.
  2. A `TaskRequest` is passed to the ADK bridge (`adk.go`).
  3. ADK invokes the Gemini model (`GOOGLE_MODEL`, default `gemini-2.5-flash`) with the task input.
  4. The model response is wrapped into a `TaskResponse` and returned to the caller.
- Config: requires `GOOGLE_API_KEY` env var; model overridable via `GOOGLE_MODEL`.

### Remote Execution (P2P Stream) — Detailed Breakdown

### Overview
When a buyer wants to execute an agent that lives on a **different peer**, `internal/agent/seller.go` and `internal/agent/manager.go` together handle the full remote execution flow. The protocol uses distinct x402 message types over a libp2p stream, not a simple binary `"execute"` frame.

---

### Message Types (Wire Protocol)
#### x402 Protocol Messages (`internal/marketplace/x402.go`)
| Constant | Value | Direction | Purpose |
|----------|-------|-----------|---------|
| `MsgTypeX402Request` | `x402.request` | Buyer → Seller | Initial execution request |
| `MsgTypeX402PaidRequest` | `x402.paid_request` | Buyer → Seller | Follow-up with payment attached |
| `MsgTypeX402PaymentRequired` | `x402.payment_required` | Seller → Buyer | Challenge nonce + payment details |
| `MsgTypeX402Response` | `x402.response` | Seller → Buyer | Final result after execution |

#### exec Protocol Messages (`internal/marketplace/execute.go`)
| Constant | Value | Direction | Purpose |
|----------|-------|-----------|---------|
| `MsgTypeExecRequest` | `exec.request` | Buyer → Seller | HTTP-bridge execution request |
| `MsgTypeExecPaidRequest` | `exec.paid_request` | Buyer → Seller | HTTP-bridge follow-up with payment |
| `MsgTypeExecPaymentRequired` | `exec.payment_required` | Seller → Buyer | HTTP-bridge 402 passthrough |
| `MsgTypeExecResponse` | `exec.response` | Seller → Buyer | HTTP-bridge agent response |
| `MsgTypeExecError` | `exec.error` | Seller → Buyer | Error response (both paths) |

---

### Step-by-Step with Functions, Files & Types

#### Step 1 — Routing Decision
**File:** `internal/agent/manager.go`  
**Function:** `ExecuteTask(ctx context.Context, resource string, req types.AgentRequest) (string, error)`

- Looks up the agent in the local listing service via `m.listingService.GetListing(resource)`.
- Checks if the agent's `PeerID` matches the local node's peer ID.
- If **not local** → routes to remote execution path over libp2p stream.

**Input:** `types.AgentRequest`
```go
// pkg/types/
type AgentRequest struct {
    Resource string // agent ID / DID
    Input    string // task input payload
}
```

---

#### Step 2 — Resolve Peer ID & Agent API from CRDT
**File:** `internal/marketplace/agent.go`  
**Function:** `GetListing(resource string) (*types.AgentListing, bool)`

- Queries the in-memory CRDT map (populated by GossipSub) for the agent's listing.
- Returns listing containing the seller's `PeerID` and `AgentAPI` URL.

**Output:** `*types.AgentListing`
```go
// pkg/types/
type AgentListing struct {
    AgentID     string
    PeerID      string   // libp2p peer ID of the seller node
    Name        string
    Description string
    Price       float64  // USDC amount (e.g. 0.01)
    PayTo       string   // seller's Ethereum address
    AgentAPI    string   // seller's agent HTTP endpoint
    // ...
}
```

---

#### Step 3 — Open libp2p Stream
**File:** `internal/agent/manager.go`  
**Function:** `host.NewStream(ctx, peerID, "/betar/marketplace/1.0.0")`

- Opens a raw libp2p stream to the resolved peer.
- Protocol ID: `/betar/marketplace/1.0.0`
- Deadline set to **30 seconds**.

**Types used:**
- `peer.ID` (libp2p)
- `network.Stream` (libp2p)

---

#### Step 4 — Serialize & Send `x402.request` Frame
**File:** `internal/agent/manager.go` (buyer side)

Buyer sends a framed `X402Request`:
```go
// internal/marketplace/x402.go
type X402Request struct {
    CorrelationID string                 // UUID for this request
    Resource      string                 // agent ID / DID
    CallerDID     string                 // buyer's DID or peer ID
    Body          []byte                 // JSON-encoded types.AgentRequest
    Payment       *X402PaymentEnvelope   // nil on first attempt; set for preemptive pay
}
```

Wire format (binary framing):
```
[type_len: 2 bytes]["x402.request"][data_len: 4 bytes][JSON(X402Request)]
```

---

#### Step 5 — Seller Handles `x402.request`
**File:** `internal/agent/seller.go`  
**Function:** `handleX402Request(ctx, from peer.ID, _ string, data []byte) (string, []byte, error)`

Three branches based on agent price and payment presence:

**Branch A — Preemptive payment provided:**
```
req.Payment != nil → handleX402WithPayment()
```

**Branch B — Free agent (price == 0):**
```go
price := m.agentPrice(req.Resource) // returns float64
// price == 0 → executeAndRespond() directly, no payment needed
```

**Branch C — Paid agent, no payment attached:**
```go
// Generates challenge nonce via payment service
challenge, _ := m.paymentService.GenerateChallenge(req.CorrelationID, 5*time.Minute)
payReq, _    := m.paymentService.CreateRequirement(m.walletAddress, amountStr)

// Returns X402PaymentRequired to buyer
type X402PaymentRequired struct {
    Version             string
    CorrelationID       string
    ChallengeNonce      string   // UUID nonce buyer must include in EIP-712 sig
    ChallengeExpiresAt  int64    // Unix timestamp
    PaymentRequirements any      // x402 standard payment requirement object
    Message             string
    SellerDID           string
}
// → returns MsgTypeX402PaymentRequired
```

---

#### Step 6 — Buyer Signs & Sends `x402.paid_request`
**File:** buyer-side logic in `internal/agent/manager.go` + `internal/marketplace/x402.go`

Buyer receives `X402PaymentRequired`, signs using EIP-712, then sends:
```go
// internal/marketplace/x402.go
type X402PaidRequest struct {
    CorrelationID string               // same UUID as original request
    CallerDID     string
    Body          []byte               // JSON-encoded types.AgentRequest (contains Resource)
    Payment       X402PaymentEnvelope  // signed payment envelope
}

type X402PaymentEnvelope struct {
    PaymentID   string
    Payer       string               // buyer's Ethereum address
    ServerNonce string               // ChallengeNonce from Step 5
    Payload     *x402.PaymentPayload // EIP-712 signed authorization
}
```

---

#### Step 7 — Seller Handles `x402.paid_request`
**File:** `internal/agent/seller.go`  
**Function:** `handleX402PaidRequest(ctx, from peer.ID, _ string, data []byte) (string, []byte, error)`

**Sub-step 7a — Validate Challenge Nonce:**
```go
// Fetch and consume the stored challenge (one-time use)
challenge, _ := m.paymentService.ConsumeChallenge(req.CorrelationID)

// Verify ServerNonce matches issued challenge
if challenge.Nonce != req.Payment.ServerNonce { ... }

// Verify EIP-712 auth nonce matches challenge nonce
if req.Payment.Payload.Authorization.Nonce != challenge.Nonce { ... }
```

**Sub-step 7b — Verify & Settle Payment:**
```go
// internal/marketplace/payment.go
header := envelopeToPaymentHeader(&req.Payment) // converts envelope → PaymentHeader

expectedAmount := big.NewInt(int64(price * 1e6)) // USDC 6 decimals

// Verifies EIP-712 signature + submits USDC transferFrom on Base Sepolia
txHash, err := m.paymentService.VerifyAndSettle(ctx, header, expectedAmount)
```

**Sub-step 7c — Execute Agent Task:**
```go
// Decode resource from body
var bodyPayload types.AgentRequest
json.Unmarshal(req.Body, &bodyPayload)

// Route back through ExecuteTask (skips session to avoid double-record)
output, err := m.ExecuteTask(
    context.WithValue(ctx, ctxKeySkipSession, true),
    resource,
    bodyPayload,
)
```

**Sub-step 7d — Record Session Exchange:**
```go
// internal/agent/seller.go — after successful execution
ex := types.Exchange{
    RequestID: req.CorrelationID,
    Input:     bodyPayload.Input,
    Output:    output,
    Timestamp: time.Now().UTC(),
    Payment: &types.PaymentRecord{
        PaymentID: header.PaymentID,
        TxHash:    txHash,
        Amount:    header.Requirement.Amount,
        Payer:     header.Payer,
        PaidAt:    time.Now().UTC(),
    },
}
m.sessionStore.AddExchange(ctx, resource, callerID, ex)
```

---

#### Step 8 — Seller Returns `x402.response`
**File:** `internal/agent/seller.go`

```go
// internal/marketplace/x402.go
type X402Response struct {
    Version       string          // marketplace.X402LibP2PVersion
    CorrelationID string          // same UUID as original request
    PaymentID     string          // from payment envelope
    TxHash        string          // USDC settlement transaction hash on Base Sepolia
    Body          []byte          // JSON: {"output": "<agent output>"}
    SellerDID     string          // agent resource / DID
    SellerTokenID string          // EIP-8004 ERC-721 token ID (if on-chain)
}
// → returns MsgTypeX402Response
```

---

#### Step 9 — Buyer Reads Response
**File:** `internal/agent/manager.go`

- Reads the framed response from the stream.
- JSON-unmarshals into `X402Response`.
- Extracts `output` string from `Body`.
- Returns output string to the caller.

---

### handleExecuteRequest — Legacy HTTP-Bridge Path
**File:** `internal/agent/seller.go`  
**Function:** `handleExecuteRequest(ctx, from peer.ID, data []byte) ([]byte, error)`

This is a **separate handler** for non-x402 execute messages. Used when the seller's agent exposes an HTTP API (`AgentAPI` in listing).

```go
type ExecuteRequest struct {
    CorrelationID string
    Resource      string
    CallerDID     string
    Body          []byte // raw JSON forwarded to the agent's HTTP API
}
```

Flow:
1. Unmarshals `marketplace.ExecuteRequest`
2. Looks up agent's `AgentAPI` URL from listing service
3. Makes an HTTP POST to that URL with `Body` as payload
4. Decodes `types.AgentResponse` from HTTP response
5. If HTTP response contains x402 payment-required → returns `MsgTypeExecPaymentRequired`
6. Otherwise → returns `MsgTypeExecResponse`

---

### Full Call Chain Summary
```
--- buyer side ---
manager.ExecuteTask(ctx, resource, AgentRequest)
  └── listingService.GetListing(resource)   // resolve PeerID
  └── host.NewStream(peerID, proto)          // open libp2p stream
  └── writeFrame(MsgTypeX402Request, X402Request)

--- seller side: first contact ---
handleX402Request(ctx, from, data)           // internal/agent/seller.go
  ├── price == 0
  │     └── executeAndRespond()              // free agent, skip payment
  ├── req.Payment != nil
  │     └── handleX402WithPayment()          // preemptive payment path
  └── price > 0, no payment
        └── paymentService.GenerateChallenge()
        └── paymentService.CreateRequirement()
        └── → MsgTypeX402PaymentRequired (with ChallengeNonce)

--- buyer side: after receiving 402 ---
  └── sign EIP-712 with ChallengeNonce
  └── writeFrame(MsgTypeX402PaidRequest, X402PaidRequest)

--- seller side: paid request ---
handleX402PaidRequest(ctx, from, data)       // internal/agent/seller.go
  └── paymentService.ConsumeChallenge()      // validate + consume nonce (one-time)
  └── envelopeToPaymentHeader()              // convert envelope
  └── paymentService.VerifyAndSettle()       // EIP-712 verify + USDC transferFrom
  └── m.ExecuteTask(ctxKeySkipSession, ...)  // manager.go → adk.go (local ADK/Gemini)
  └── sessionStore.AddExchange()             // record Exchange + PaymentRecord
  └── → MsgTypeX402Response (with TxHash, SellerTokenID)

--- buyer side: read result ---
  └── readFrame() → X402Response
  └── extract Body → {"output": "..."}
```

### Execution Routing Logic (inside ExecuteTask)
```
manager.ExecuteTask(ctx, resource, AgentRequest)
  ├── agent is local  → adk.RunTask()         // Google ADK / Gemini
  └── agent is remote → open stream → x402 flow above
```

### Key Type Locations
| Type | File |
|------|------|
| `X402Request` | `internal/marketplace/x402.go` |
| `X402PaidRequest` | `internal/marketplace/x402.go` |
| `X402PaymentRequired` | `internal/marketplace/x402.go` |
| `X402PaymentEnvelope` | `internal/marketplace/x402.go` |
| `X402Response` | `internal/marketplace/x402.go` |
| `ExecuteRequest` | `internal/marketplace/x402.go` |
| `ExecuteRequestResponse` | `internal/agent/seller.go` |
| `AgentRequest` | `pkg/types/` |
| `AgentResponse` | `pkg/types/` |
| `Exchange` | `pkg/types/` |
| `PaymentRecord` | `pkg/types/` |

---

#### Call Chain Summary
```
handleX402Request()                        // seller.go — first contact
  ├── price == 0    → executeAndRespond()  // free agent, direct ADK
  ├── req.Payment != nil → handleX402WithPayment()  // preemptive pay
  └── price > 0, no payment:
        paymentService.GenerateChallenge() // issues nonce
        paymentService.CreateRequirement() // builds X402PaymentRequired
        → returns MsgTypeX402PaymentRequired

handleX402PaidRequest()                    // seller.go — after buyer pays
  └── paymentService.ConsumeChallenge()    // validates nonce
  └── paymentService.VerifyAndSettle()     // EIP-712 verify + USDC transfer
  └── m.ExecuteTask()                      // runs ADK locally
  └── sessionStore.AddExchange()           // records payment + output
  └── → returns MsgTypeX402Response
```

### Execution Routing Logic
```
manager.ExecuteTask(agentID, taskReq)
  ├── if agent is local  →  adk.RunTask(taskReq)   // Google ADK / Gemini
  └── if agent is remote →  p2p.SendExecute(peerID, taskReq)  // libp2p stream
```

### Stream Handler Types
| Type | Direction | Purpose |
|------|-----------|---------|
| `"execute"` | Buyer → Seller | Send `TaskRequest`; triggers x402 flow if payment required |
| `"info"` | Buyer → Seller | Query agent metadata / capability info |

### Payment Integration
- If the remote agent requires payment, the `"execute"` stream triggers the **x402 flow** before task execution proceeds (see [x402 Payment Flow](#x402-payment-flow)).
- On successful settlement, reputation feedback is auto-submitted to the EIP-8004 registry.

---

## Environment Variables
| Variable | Default | Description |
|---|---|---|
| `GOOGLE_API_KEY` | **required** | Gemini model access |
| `GOOGLE_MODEL` | `gemini-2.5-flash` | ADK model name |
| `BOOTSTRAP_PEERS` | — | Comma-separated multiaddrs |
| `BETAR_DATA_DIR` | `~/.betar` | Local data directory |
| `BETAR_P2P_KEY_PATH` | `~/.betar/p2p_identity.key` | P2P identity key file |
| `ETHEREUM_PRIVATE_KEY` | — | Wallet private key (hex, no 0x prefix) |
| `ETHEREUM_RPC_URL` | `https://sepolia.base.org` | RPC endpoint |
| `ERC8004_IDENTITY_ADDR` | `0x8004...BD9e` | EIP-8004 identity registry contract |
| `REPUTATION_REGISTRY_ADDRESS` | — | Reputation registry contract address |
| `VALIDATION_REGISTRY_ADDRESS` | — | Validation registry contract address |

---

## Coding Conventions
- **Error handling:** always wrap errors with `fmt.Errorf("context: %w", err)`.
- **New CLI commands:** add a `cobra.Command` in `cmd/betar/main.go` and wire it to a service in the appropriate `internal/` package.
- **New internal package:** create `internal/<name>/`, add config to `internal/config/`, inject via the node struct.
- **Tests:** place next to the package (`_test.go`), use table-driven tests. Run with `go test ./internal/<pkg>/...`.
- **Contracts:** edit in `contracts/`, build with `make contracts-build`, use Foundry scripts for deployment.
- **No secrets in code** — all credentials via environment variables only.