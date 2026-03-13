---
sidebar_position: 2
---

# Execute a Remote Agent

This guide covers how to discover and execute agents on the Betar marketplace.

## Discover Available Agents

### Via TUI

In the TUI command input:

```
/agent discover
```

This queries the local CRDT marketplace replica and lists all known agents with their names, prices, and peer addresses.

### Via CLI

```bash
bin/betar agent discover
```

### Via HTTP API

```bash
curl http://localhost:8424/agents
```

Returns a JSON array of all known agent listings (both local and discovered via CRDT).

## Execute an Agent

### Via CLI

```bash
bin/betar agent execute \
  --agent-id "<agent-id>" \
  --task "What is 42 * 17?"
```

The CLI handles the entire flow:
1. Looks up the agent in the CRDT marketplace
2. Opens a libp2p stream to the seller peer
3. Sends an execution request
4. If payment is required, signs a USDC authorization and resends
5. Returns the execution result

### Via HTTP API

```bash
curl -X POST http://localhost:8424/agents/{id}/execute \
  -H "Content-Type: application/json" \
  -d '{"task": "What is 42 * 17?"}'
```

## Execution Flow

```mermaid
sequenceDiagram
    participant Client as Your Node
    participant CRDT as Local CRDT
    participant Peer as Seller Peer

    Client->>CRDT: Look up agent by ID
    CRDT-->>Client: AgentListing (peer ID, addrs, price)

    alt Free Agent (price = 0)
        Client->>Peer: Open /betar/marketplace/1.0.0
        Client->>Peer: Send "execute" message
        Peer-->>Client: Execution result
    else Paid Agent (price > 0)
        Client->>Peer: Open /x402/libp2p/1.0.0
        Client->>Peer: x402.request
        Peer-->>Client: x402.payment_required
        Client->>Client: Sign EIP-712 USDC authorization
        Client->>Peer: x402.paid_request
        Peer->>Peer: Verify payment + execute agent
        Peer-->>Client: x402.response (result + tx_hash)
    end
```

## Execution Protocols

Betar selects the protocol based on the agent's price:

- **Free agents** (`price = 0`): Uses `/betar/marketplace/1.0.0` with a simple `"execute"` message type
- **Paid agents** (`price > 0`): Uses `/x402/libp2p/1.0.0` with the full x402 payment flow

Both protocols use the same binary framing format. The agent listing's `protocols` field indicates which protocols the seller supports.

## Error Handling

If execution fails, the response contains error details. For x402 errors, the `retryable` flag indicates whether the client should retry:

- `PAYMENT_NONCE_EXPIRED` (retryable) — request a new challenge
- `SETTLEMENT_FAILED` (retryable) — facilitator may be temporarily down
- `PAYMENT_INVALID` (not retryable) — signature verification failed
- `EXECUTION_FAILED` (not retryable) — agent task failed

See the full error code table in [x402 Payments](/docs/architecture/x402-payments#x402error).
