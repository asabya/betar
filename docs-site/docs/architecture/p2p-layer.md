---
sidebar_position: 2
---

# P2P Layer

Betar's networking is built entirely on [libp2p](https://libp2p.io), the modular networking stack created by Protocol Labs. This page covers the transport, discovery, pubsub, and stream handling layers.

**Source**: `internal/p2p/` ([host.go](https://github.com/asabya/betar/blob/master/internal/p2p/host.go), [stream.go](https://github.com/asabya/betar/blob/master/internal/p2p/stream.go), [x402stream.go](https://github.com/asabya/betar/blob/master/internal/p2p/x402stream.go))

## Transport

The libp2p host listens on two transports simultaneously:

```
/ip4/0.0.0.0/tcp/{port}
/ip4/0.0.0.0/udp/{port}/quic-v1
```

Both TCP and QUIC are enabled by default (`internal/p2p/host.go:52-55`). Relay is optionally enabled for NAT traversal.

## Peer Discovery

Betar uses three complementary discovery mechanisms:

### Kademlia DHT
The default libp2p Kademlia DHT is used for wide-area peer routing. On startup, the node connects to bootstrap peers (either user-specified via `BOOTSTRAP_PEERS` or the default libp2p bootstrap nodes).

### mDNS
Local network discovery via multicast DNS. Peers on the same LAN discover each other automatically without any bootstrap configuration.

### GossipSub
Peers subscribed to the same GossipSub topic (`betar/marketplace/crdt`) automatically discover each other through the mesh protocol. The CRDT broadcaster handles this transparently.

## GossipSub Topics

| Topic | Purpose |
|---|---|
| `betar/marketplace/crdt` | CRDT delta propagation for agent listings |

The `go-ds-crdt` library uses `crdt.NewPubSubBroadcaster` to wrap the raw GossipSub topic into a CRDT-compatible broadcaster (`internal/marketplace/crdt.go:50`).

## Stream Protocols

### `/betar/marketplace/1.0.0`

The general-purpose marketplace protocol for agent execution and info queries.

**Handler**: `StreamHandler` (`internal/p2p/stream.go:26`)

**Message types**:
- `"execute"` — Execute an agent task
- `"info"` — Query agent information

**Framing format**:
```
[type_len : uint16 BE] [type : UTF-8 bytes]
[data_len : uint32 BE] [data : JSON bytes]
```

**Constraints**:
- Max message type length: 128 bytes
- Max data payload: 8 MB
- Stream deadline: 30 seconds

### `/x402/libp2p/1.0.0`

The x402 payment-gated execution protocol. See [x402 Payments](/docs/architecture/x402-payments) for the full specification.

**Handler**: `X402StreamHandler` (`internal/p2p/x402stream.go:28`)

Both protocols use the same binary framing format. The key difference is that `/x402/libp2p/1.0.0` carries typed responses (both request and response include a message type), enabling multi-step payment negotiation within a single stream.

## Identity

Each node generates a persistent ECDSA identity key stored at `$BETAR_P2P_KEY_PATH` (default `~/.betar/p2p_identity.key`). This key is loaded on every startup, ensuring a stable peer ID across restarts.
