---
sidebar_position: 3
---

# Payment Flow Walkthrough

This guide walks through an end-to-end x402 payment, referencing the actual Go source code at each step.

## Prerequisites

- Both buyer and seller nodes are running with `ETHEREUM_PRIVATE_KEY` configured
- Buyer has USDC balance on Base Sepolia
- Seller has registered an agent with `price > 0`

## Step 1: Buyer Opens Stream

The buyer discovers the agent in the CRDT marketplace and opens a `/x402/libp2p/1.0.0` stream to the seller peer.

**Source**: `internal/p2p/x402stream.go:53-82` (`SendX402Message`)

```go
stream, err := s.host.NewStream(ctx, to, X402ProtocolID)
```

## Step 2: Buyer Sends x402.request

The buyer sends an `x402.request` message with the resource (agent ID) and method (`"execute"`). No payment is attached on the first attempt.

**Source**: `internal/marketplace/x402.go:54-62` (`X402Request`)

The request is serialized to JSON and written using the binary frame format:
```
[type_len=12][x402.request][data_len=N][JSON payload]
```

## Step 3: Seller Returns x402.payment_required

The seller's stream handler receives the request, determines payment is required, and generates a challenge.

**Source**: `internal/marketplace/payment.go:83-101` (`GenerateChallenge`)

```go
nonceBytes := make([]byte, 32)
rand.Read(nonceBytes)
nonce := hex.EncodeToString(nonceBytes)
```

The challenge nonce is stored in a `ChallengeStore` keyed by `correlation_id`, with a TTL. The seller returns:

```json
{
  "version": "1.0",
  "challenge_nonce": "0xabc123...",
  "challenge_expires_at": 1711900060,
  "payment_requirements": {
    "scheme": "exact",
    "network": "eip155:84532",
    "amount": "1000",
    "asset": "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
    "payTo": "0x<seller-address>",
    "maxTimeoutSeconds": 60
  }
}
```

## Step 4: Buyer Signs Payment

The buyer constructs an EIP-712 typed data structure for USDC's `transferWithAuthorization` and signs it with their Ethereum private key.

**Source**: `internal/marketplace/payment.go:138-194` (`SignRequirementWithNonce`)

Key fields in the EIP-712 authorization:

| Field | Value |
|---|---|
| `from` | Buyer's Ethereum address |
| `to` | Seller's `payTo` address |
| `value` | Amount in atomic USDC (6 decimals) |
| `validAfter` | `now - 5s` (clock skew buffer) |
| `validBefore` | `now + maxTimeoutSeconds` |
| `nonce` | The seller's challenge nonce (hex-encoded, 0x-prefixed) |

The buyer computes the EIP-712 digest and signs it:

```go
digest, err := s.verifier.ComputeEIP712Digest(auth)
sig, err := s.wallet.SignRaw(digest)
```

## Step 5: Buyer Sends x402.paid_request

The signed payment envelope is wrapped in an `X402PaidRequest` and sent back to the seller over the same stream.

**Source**: `internal/marketplace/x402.go:76-82` (`X402PaidRequest`)

## Step 6: Seller Verifies Payment

The seller performs local verification:

1. **Consume challenge**: Retrieves and deletes the challenge by `correlation_id`, checks expiry (`internal/marketplace/payment.go:105-120`)
2. **Validate payment**: Checks scheme, network, asset address, payTo, amount, and recovers the signer from the EIP-712 signature
3. **Execute agent**: Runs the task via Google ADK

**Source**: `internal/marketplace/payment.go:323-360` (`VerifyAndSettle`)

## Step 7: Settlement with Facilitator

After successful verification and execution, the seller settles the payment with the x402 facilitator.

**Source**: `internal/marketplace/payment.go:473-537` (`settleWithFacilitator`)

```go
httpReq, err := http.NewRequestWithContext(ctx, "POST",
    s.facilitator+"/settle", bytes.NewBuffer(reqBody))
```

The facilitator submits the `transferWithAuthorization` transaction to the USDC contract on Base Sepolia and returns the transaction hash.

:::info Off-path settlement
Settlement is the only HTTP call in the entire flow. Discovery, execution, and payment negotiation all happen over libp2p streams. Settlement includes exponential backoff retry (up to 5 attempts).
:::

## Step 8: Seller Returns x402.response

The seller sends the execution result along with the settlement transaction hash:

```json
{
  "version": "1.0",
  "correlation_id": "uuid-v4",
  "payment_id": "0x<keccak256>",
  "tx_hash": "0x<on-chain-tx>",
  "body": "<execution result>"
}
```

The buyer can independently verify the transaction on Base Sepolia using the `tx_hash`.

## Summary

| Step | Who | What | Where (HTTP or P2P?) |
|---|---|---|---|
| 1 | Buyer | Open libp2p stream | P2P |
| 2 | Buyer | Send x402.request | P2P |
| 3 | Seller | Return x402.payment_required | P2P |
| 4 | Buyer | Sign EIP-712 authorization | Local |
| 5 | Buyer | Send x402.paid_request | P2P |
| 6 | Seller | Verify signature + execute | Local |
| 7 | Seller | Settle with facilitator | HTTP (off-path) |
| 8 | Seller | Return x402.response | P2P |
