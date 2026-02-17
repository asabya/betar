# Payment Flow Implementation Plan (P2P-only)

> **For Claude:** Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement payment flow for Agent-to-Agent over libp2p streams (no HTTP/MCP).

**Architecture:**
- Payment flows over libp2p direct streams (`/betar/marketplace/1.0.0`)
- Use x402-go/v2 for signing (`evm.NewSigner`, `signer.Sign`)
- Facilitator used only for verification (not as transport)
- Buyer signs EIP-3009 authorization, sends via P2P stream

**Tech Stack:** Go, x402-go/v2, libp2p streams, Ethereum/Base (CAIP-2)

---

## Payment Flow Diagram

```
Buyer (libp2p)                              Seller (libp2p)
     │                                             │
     │──── Execute {agentId, input} ──────────────►│
     │                                             │──► Get listing, check price > 0?
     │                                             │
     │◄─── 402 {payment_requirement} ──────────────│ (if price > 0 and no payment)
     │                                             │
     │  (Buyer: sign with evm.Signer)              │
     │                                             │
     │──── Execute {agentId, input, paymentHeader}►│
     │                                             │──► Verify (facilitator /verify)
     │                                             │──► Execute task
     │                                             │
     │◄─── {output, txHash} ─────────────────────│
```

---

## Current State

**What exists:**
- `internal/marketplace/x402.go`: Types (needs update for v2)
- `internal/marketplace/payment.go`: Partial service (needs update)
- `internal/eth/wallet.go`: Has signing capability

**What's broken:**
- Seller doesn't check payment before executing
- No PaymentRequiredResponse returned
- Buyer doesn't send payment header

---

## Task 1: Update x402 types for v2 API (No MCP)

**Files:**
- Modify: `internal/marketplace/x402.go`

**Step 1: Update imports and types**

```go
import (
    v2 "github.com/mark3labs/x402-go/v2"
)

// PaymentRequirement is v2.PaymentRequirements
type PaymentRequirement = v2.PaymentRequirements

// PaymentHeader for P2P messages
type PaymentHeader struct {
    Requirement v2.PaymentRequirements `json:"requirement"`
    Payer       string                 `json:"payer"`
    PaymentID   string                 `json:"payment_id"`
    Payload     *v2.EVMPayload         `json:"payload,omitempty"`
}

// TaskExecuteRequest with optional payment
type TaskExecuteRequest struct {
    AgentID       string          `json:"agent_id"`
    Input         string          `json:"input"`
    PaymentHeader *PaymentHeader  `json:"payment_header,omitempty"`
}

// PaymentRequiredResponse (402)
type PaymentRequiredResponse struct {
    AgentID            string               `json:"agent_id"`
    RequestID          string               `json:"request_id"`
    Message            string               `json:"message"`
    PaymentRequirement *v2.PaymentRequirements `json:"payment_requirement,omitempty"`
    RequiresPayment    bool                 `json:"requires_payment"`
}
```

**Step 2: Update network constants**

```go
const (
    NetworkBaseSepolia = "eip155:84532"
    USDCBaseSepolia    = "0x036CbD53842c5426634e7929541eC2318f3dCF7e"
)

func GetUSDCAddress(network string) string {
    if network == NetworkBaseSepolia {
        return USDCBaseSepolia
    }
    return USDCBaseSepolia
}

func CreatePaymentRequirement(network, amount, asset, payTo string, timeout int64) v2.PaymentRequirements {
    return v2.PaymentRequirements{
        Scheme:            "exact",
        Network:           network,
        Amount:            amount,  // v2 uses "Amount"
        Asset:             asset,
        PayTo:             payTo,
        MaxTimeoutSeconds: int(timeout),
    }
}
```

**Step 3: Verify**

```bash
go build ./...
```

---

## Task 2: Seller Side - Check Payment in handleExecuteRequest

**Files:**
- Modify: `internal/agent/manager.go:361-384`

**Step 1: Update handleExecuteRequest**

```go
func (m *Manager) handleExecuteRequest(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
    var req struct {
        AgentID       string                  `json:"agentId"`
        Input         string                  `json:"input"`
        PaymentHeader *marketplace.PaymentHeader `json:"paymentHeader,omitempty"`
    }

    if err := json.Unmarshal(data, &req); err != nil {
        return nil, fmt.Errorf("failed to unmarshal: %w", err)
    }

    // Get listing to check if payment required
    listing, ok := m.listingService.GetListing(req.AgentID)
    if !ok {
        fullID := from.String() + "/" + req.AgentID
        listing, ok = m.listingService.GetListing(fullID)
    }

    requiresPayment := ok && listing.Price > 0

    // If payment required but not provided, return 402
    if requiresPayment && req.PaymentHeader == nil {
        amount := fmt.Sprintf("%d", int(listing.Price*1e6)) // USDC 6 decimals

        paymentReq := marketplace.CreatePaymentRequirement(
            marketplace.NetworkBaseSepolia,
            amount,
            marketplace.USDCBaseSepolia,
            m.walletAddress,
            300,
        )

        resp := marketplace.PaymentRequiredResponse{
            AgentID:            req.AgentID,
            RequestID:          fmt.Sprintf("%d", time.Now().UnixNano()),
            Message:            "Payment required",
            PaymentRequirement: &paymentReq,
            RequiresPayment:    true,
        }

        return json.Marshal(resp)
    }

    // If payment provided, verify it
    if req.PaymentHeader != nil {
        if err := m.verifyPayment(ctx, req.PaymentHeader); err != nil {
            return nil, fmt.Errorf("payment verification failed: %w", err)
        }
    }

    // Execute task
    output, err := m.ExecuteTask(ctx, req.AgentID, req.Input)
    if err != nil {
        return json.Marshal(map[string]string{"error": err.Error()})
    }

    resp := map[string]interface{}{"output": output}
    if req.PaymentHeader != nil {
        resp["paymentId"] = req.PaymentHeader.PaymentID
    }
    return json.Marshal(resp)
}
```

**Step 2: Add verifyPayment method**

```go
func (m *Manager) verifyPayment(ctx context.Context, header *marketplace.PaymentHeader) error {
    if header.Payload == nil {
        return fmt.Errorf("no payment payload")
    }
    // Verify via facilitator
    return m.paymentService.Verify(ctx, header)
}
```

**Step 3: Add walletAddress to Manager struct**

Add `walletAddress string` to Manager struct in `internal/agent/manager.go`

**Step 4: Verify**

```bash
go build ./...
```

---

## Task 3: Update payment.go for v2 Signer

**Files:**
- Modify: `internal/marketplace/payment.go`

**Step 1: Rewrite to use v2 signer**

```go
import (
    v2 "github.com/mark3labs/x402-go/v2"
    "github.com/mark3labs/x402-go/v2/signers/evm"
)

type PaymentService struct {
    wallet      *eth.Wallet
    paymentAddr string
    network     string
    facilitator string
    signer      *evm.Signer
}

func NewPaymentService(wallet *eth.Wallet, paymentAddr string) (*PaymentService, error) {
    network := "eip155:84532" // Base Sepolia

    chainConfig, _ := v2.GetChainConfig(network)

    tokens := []v2.TokenConfig{{
        Address:  chainConfig.USDCAddress,
        Symbol:   "USDC",
        Decimals: 6,
    }}

    signer, err := evm.NewSigner(network, wallet.PrivateKeyHex(), tokens)
    if err != nil {
        return nil, err
    }

    return &PaymentService{
        wallet:      wallet,
        paymentAddr: paymentAddr,
        network:     network,
        facilitator: "https://facilitator.x402.rs",
        signer:      signer,
    }, nil
}

// Sign signs a payment requirement
func (s *PaymentService) Sign(req *v2.PaymentRequirements) (*PaymentHeader, error) {
    payload, err := s.signer.Sign(req)
    if err != nil {
        return nil, err
    }

    evmPayload, ok := payload.Payload.(v2.EVMPayload)
    if !ok {
        return nil, fmt.Errorf("unexpected payload type")
    }

    return &PaymentHeader{
        Requirement: *req,
        Payer:       s.wallet.AddressHex(),
        PaymentID:   generatePaymentID(s.wallet.AddressHex(), req.PayTo, req.Amount),
        Payload:     &evmPayload,
    }, nil
}

// Verify verifies payment with facilitator
func (s *PaymentService) Verify(ctx context.Context, header *PaymentHeader) error {
    req := map[string]interface{}{
        "x402Version":        1,
        "paymentPayload":     header.Payload,
        "paymentRequirements": header.Requirement,
    }
    // POST to facilitator /verify
    // Return error if not valid
}

// Settle settles payment on-chain
func (s *PaymentService) Settle(ctx context.Context, header *PaymentHeader) (string, error) {
    return s.signer.Settle(ctx, header.Payload)
}
```

**Step 2: Verify**

```bash
go build ./...
```

---

## Task 4: Buyer Side - RemoteExecute with Payment

**Files:**
- Modify: `internal/agent/manager.go`

**Step 1: Update RemoteExecute**

```go
func (m *Manager) RemoteExecute(ctx context.Context, peerID peer.ID, agentID, input string, paymentHeader *marketplace.PaymentHeader) (string, *marketplace.PaymentRequiredResponse, error) {
    req := map[string]interface{}{
        "agentId": agentID,
        "input":   input,
    }
    if paymentHeader != nil {
        req["paymentHeader"] = paymentHeader
    }

    reqData, _ := json.Marshal(req)
    resp, err := m.streamHandler.SendMessage(ctx, peerID, "execute", reqData)
    if err != nil {
        return "", nil, err
    }

    // Check if 402
    var payResp marketplace.PaymentRequiredResponse
    if json.Unmarshal(resp, &payResp); payResp.RequiresPayment {
        return "", &payResp, nil
    }

    var result map[string]interface{}
    json.Unmarshal(resp, &result)

    if errMsg, ok := result["error"]; ok {
        return "", nil, fmt.Errorf("remote error: %s", errMsg)
    }

    return result["output"].(string), nil, nil
}
```

**Step 2: Add ExecuteTask wrapper**

```go
func (m *Manager) ExecuteTask(ctx context.Context, agentID, input string, paymentHeader *marketplace.PaymentHeader) (string, *marketplace.PaymentRequiredResponse, error) {
    // ... existing local agent check ...

    // For remote, pass payment header
    return m.RemoteExecute(ctx, peerID, runtimeAgentID, input, paymentHeader)
}
```

**Step 3: Verify**

```bash
go build ./...
```

---

## Task 5: HTTP API for 402 Responses

**Files:**
- Modify: `cmd/betar/api/handlers/agents.go`

**Step 1: Update handler**

```go
func (h *agentHandler) executeAgent(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Input        string                   `json:"input"`
        PaymentHeader *marketplace.PaymentHeader `json:"paymentHeader,omitempty"`
    }
    json.NewDecoder(r.Body).Decode(&req)

    output, payResp, err := h.agentMgr.ExecuteTask(r.Context(), agentID, req.Input, req.PaymentHeader)

    if payResp != nil {
        w.WriteHeader(http.StatusPaymentRequired)
        json.NewEncoder(w).Encode(payResp)
        return
    }

    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]string{"output": output})
}
```

**Step 2: Verify**

```bash
go build ./...
```

---

## Testing

1. Start seller with `--price 0.01`
2. Buyer executes without payment → gets 402
3. Buyer signs payment, retries with header → gets output

---

## Success Criteria

- [ ] Seller returns 402 when payment needed but not provided
- [ ] Buyer can handle 402 and sign payment
- [ ] Payment verified via facilitator
- [ ] Full flow: execute → 402 → sign → execute with payment → output
