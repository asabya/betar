# x402 over libp2p — Protocol Specification

**Protocol ID:** `/x402/libp2p/1.0.0`
**Status:** Draft
**Reference implementation:** [asabya/betar](https://github.com/asabya/betar)

---

## 1. Overview

This document specifies how the [x402 payment standard](https://x402.org) is carried over [libp2p](https://libp2p.io) streams. It is the peer-to-peer analogue of the HTTP x402 standard: instead of HTTP request/response headers, payment information travels inside typed binary frames over a dedicated libp2p protocol stream.

The key improvements over ad-hoc embedded-payment approaches are:

- **Typed frames** — both request and response carry a message-type field, so a single handler can return different message types (`x402.response` vs `x402.error`) and the caller knows which it received.
- **Server-issued nonce challenge** — the server generates a fresh 32-byte nonce per interaction that becomes the EIP-3009 `TransferWithAuthorization.nonce`, binding the payment cryptographically to the specific interaction.
- **Preemptive payment (1-trip)** — clients that have cached payment requirements can skip the challenge round-trip by self-generating a nonce and marking it `"preemptive"`.
- **Structured error codes** — a typed error taxonomy with a `retryable` flag.

### Relationship to HTTP x402

HTTP x402 uses the `402 Payment Required` HTTP status code and `X-Payment` / `X-Payment-Requirements` headers. This spec replaces those HTTP primitives with a binary-framed, bidirectional libp2p stream while reusing the same EIP-712 / EIP-3009 payment payload format (`EVMPayload`, `EVMAuthorization`, `PaymentRequirements`).

---

## 2. Terminology

| Term | Definition |
|---|---|
| **client** | The libp2p peer initiating a resource request |
| **server** | The libp2p peer hosting the resource (agent) |
| **facilitator** | Third-party settlement service that calls `transferWithAuthorization` on-chain |
| **resource** | An agent or service identified by its agent ID |
| **challenge nonce** | A 32-byte random value generated fresh by the server per interaction |
| **correlation ID** | A client-generated UUID v4 identifying a logical request across multiple stream messages |

---

## 3. Wire Framing

Every message in both directions uses the same binary frame format:

```
┌─────────────────────────┬──────────────────────────────────────────────────────┐
│  type_len : uint16 BE   │  type : UTF-8 string (length = type_len)             │
├─────────────────────────┼──────────────────────────────────────────────────────┤
│  data_len : uint32 BE   │  data : JSON bytes (length = data_len)               │
└─────────────────────────┴──────────────────────────────────────────────────────┘
```

**Limits:**
- `type_len` ≤ 128 bytes
- `data_len` ≤ 8,388,608 bytes (8 MiB)

Both the request **and** the response use this framing, which means a single stream handler can write back a `x402.payment_required` frame OR a `x402.response` frame and the caller decodes the type first.

**Stream lifecycle:** Each stream MUST carry exactly one request frame (client → server) and exactly one response frame (server → client). The server MUST close the stream after writing the response. For the standard 2-trip flow, the client opens two separate streams — one for `x402.request` / `x402.payment_required` and one for `x402.paid_request` / `x402.response`.

---

## 4. Message Types

| `msg_type` | Direction | Purpose |
|---|---|---|
| `x402.request` | client → server | Resource request; `payment` is null on first attempt |
| `x402.payment_required` | server → client | Server's 402 equivalent; carries challenge nonce + requirements |
| `x402.paid_request` | client → server | Retry with signed payment embedding server nonce |
| `x402.response` | server → client | Success; carries result body + tx hash |
| `x402.error` | server → client | Typed error with code and `retryable` flag |

---

## 5. Message Schemas

All `body` fields carry opaque application payload. The value MUST be the JSON encoding of the raw application bytes, which renders as a **base64-encoded string** in the JSON wire format (standard Go `[]byte` → JSON marshalling). The server MUST reject a `body` that is not valid base64 or whose decoded content is not valid JSON with `INVALID_MESSAGE (1000)`.

### 5.1 `x402.request`

> `correlation_id` MUST be a UUID v4 string. The server MUST reject a request whose `correlation_id` is empty or not a valid UUID v4 with `INVALID_MESSAGE (1000)`.

```jsonc
{
  "version": "1.0",
  "correlation_id": "<uuid-v4>",
  "resource": "<agent-id>",
  "method": "execute",
  "payment": null,            // null = first attempt (standard flow)
                              // or X402PaymentEnvelope for preemptive
  "body": <bytes>             // application payload (JSON-encoded)
}
```

### 5.2 `x402.payment_required`

```jsonc
{
  "version": "1.0",
  "correlation_id": "<same as request>",
  "challenge_nonce": "<64 lowercase hex chars = 32 bytes>",
  "challenge_expires_at": <unix timestamp>,
  "payment_requirements": {
    "scheme": "exact",
    "network": "eip155:84532",
    "amount": "<atomic USDC units>",
    "asset": "<USDC contract address>",
    "pay_to": "<seller Ethereum address>",
    "max_timeout_seconds": 300,
    "extra": { "name": "USDC", "version": "2" }
  },
  "message": "Payment required"
}
```

### 5.3 `x402.paid_request`

```jsonc
{
  "version": "1.0",
  "correlation_id": "<same>",
  "payment": {
    "x402_version": 2,
    "scheme": "exact",
    "network": "eip155:84532",
    "server_nonce": "<challenge_nonce or 'preemptive'>",
    "payer": "0x...",
    "payload": {
      "signature": "0x...",        // EIP-712 signature over TransferWithAuthorization
      "authorization": {
        "from": "0x...",
        "to": "0x...",             // seller address
        "value": "<atomic>",
        "valid_after": "<unix ts>",
        "valid_before": "<unix ts>",
        "nonce": "0x<challenge_nonce>"  // MUST equal challenge_nonce for standard flow
      }
    }
  },
  "body": <bytes>                  // identical to the body in x402.request
}
```

### 5.4 `x402.response`

```jsonc
{
  "version": "1.0",
  "correlation_id": "<same>",
  "payment_id": "<payer:txhash>",
  "tx_hash": "0x...",
  "body": <bytes>                  // application response
}
```

### 5.5 `x402.error`

```jsonc
{
  "version": "1.0",
  "correlation_id": "<string>",
  "error_code": <int>,
  "error_name": "<string>",
  "message": "<human-readable>",
  "retryable": <bool>
}
```

---

## 6. Error Codes

| Code | Name | Retryable | Description |
|---|---|---|---|
| 1000 | `INVALID_MESSAGE` | false | Malformed or unrecognised frame |
| 1001 | `UNKNOWN_RESOURCE` | false | The requested agent/resource does not exist |
| 2000 | `PAYMENT_REQUIRED` | true | No payment was provided; retry after paying |
| 2001 | `PAYMENT_INVALID` | false | Signature or field validation failed |
| 2002 | `PAYMENT_NONCE_MISMATCH` | false | EIP-712 auth nonce ≠ challenge nonce |
| 2003 | `PAYMENT_NONCE_EXPIRED` | true | Challenge expired; request a new one |
| 2004 | `PAYMENT_NONCE_USED` | false | Nonce already consumed (replay attempt) |
| 2005 | `PAYMENT_AMOUNT_WRONG` | false | Signed amount doesn't match requirement |
| 2007 | `SETTLEMENT_FAILED` | true | Facilitator settlement failed; safe to retry |
| 3000 | `EXECUTION_FAILED` | false | Agent execution error (payment already settled) |

---

## 7. Protocol Flows

> **Settlement invariant:** The server MUST complete facilitator settlement before invoking resource execution. If settlement succeeds but execution fails, the server MUST return `EXECUTION_FAILED (3000)`. The server MUST NOT attempt resource execution if settlement fails.

### 7.1 Standard 2-Trip Flow (unknown price)

```
Client                                         Server
  |                                              |
  |  x402.request (payment=null)                |
  |--------------------------------------------->|
  |                                              |
  |  x402.payment_required                       |
  |    challenge_nonce = <32-byte random>        |
  |    payment_requirements = { amount, payTo }  |
  |<---------------------------------------------|
  |                                              |
  | [Client signs EIP-712 TransferWithAuthorization
  |  with Authorization.Nonce = challenge_nonce]  |
  |                                              |
  |  x402.paid_request                           |
  |    payment.server_nonce = challenge_nonce    |
  |    payment.payload.authorization.nonce = 0x<challenge_nonce>
  |--------------------------------------------->|
  |                                              |
  | [Server: validate nonce match, VerifyAndSettle, execute]
  |                                              |
  |  x402.response (tx_hash, body)               |
  |<---------------------------------------------|
```

### 7.2 Optimised 1-Trip Flow (cached requirements)

```
Client                                         Server
  |                                              |
  | [Client already knows payment_requirements]  |
  | [Client generates 32-byte random nonce R]    |
  | [Client signs EIP-712 with Nonce = R]        |
  |                                              |
  |  x402.request                                |
  |    payment.server_nonce = "preemptive"       |
  |    payment.payload.authorization.nonce = 0xR |
  |--------------------------------------------->|
  |                                              |
  | [Server: no challenge lookup needed,         |
  |  validate R not in usedNonces,               |
  |  VerifyAndSettle, execute]                   |
  |                                              |
  |  x402.response (tx_hash, body)               |
  |<---------------------------------------------|
```

---

## 8. Nonce Security

The `challenge_nonce` serves a dual role:

1. **Server challenge** — Binds the payment to this specific interaction, preventing a client from re-using a signed payment from a previous interaction.
2. **EIP-3009 on-chain nonce** — The same 32-byte value becomes `TransferWithAuthorization.nonce`. Once `transferWithAuthorization` executes on-chain, the nonce is permanently consumed by the USDC contract, making the payment unreplayable at the blockchain level.

**Challenge TTL:** The server keeps a challenge alive for 5 minutes (configurable). If `ChallengeExpiresAt` passes before the client sends `x402.paid_request`, the server returns `PAYMENT_NONCE_EXPIRED` (retryable=true) and the client must restart the flow.

**Preemptive nonce security:** When `server_nonce = "preemptive"`, the server performs standard EIP-712 signature verification (which checks timestamp validity and recovers the signer address), and the on-chain USDC nonce replay protection ensures the same signed payload cannot be reused.

**Validation order (standard flow):** Before attempting settlement, the server MUST verify in order:
1. The challenge nonce has not expired (`challenge_expires_at`).
2. `authorization.valid_after` ≤ current time ≤ `authorization.valid_before`.
3. The `authorization.nonce` matches the stored challenge nonce.

All three checks MUST pass or the server MUST return the appropriate error without calling the facilitator.

**Preemptive nonce rules:** When `server_nonce = "preemptive"`:
- The `authorization.nonce` MUST be globally unique (never previously used).
- The server MUST mark the nonce consumed **before** calling the facilitator.
- The nonce MUST NOT be reused even if settlement subsequently fails.

---

## 9. Design Rationale

### JSON over protobuf

This specification uses JSON because:
- The existing x402 Go and TypeScript libraries use JSON for `EVMPayload`.
- Interoperability with the HTTP x402 ecosystem is easier with a shared serialization format.
- The payload sizes (< 1 KB for most messages) make the overhead negligible.

---

## 10. Implementation Notes

- `challenge_nonce` is stored as lowercase hex without `0x` prefix in `X402PaymentRequired.ChallengeNonce`.
- `EVMAuthorization.Nonce` uses `0x`-prefixed hex (EIP-712 bytes32 encoding).
- The server MUST delete a challenge from its store after one consumption attempt (whether successful or not), preventing replay via repeated `x402.paid_request` messages.
- Facilitator settlement is performed by the server; the client only needs to provide a valid EIP-712 signature.
