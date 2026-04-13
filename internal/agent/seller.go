package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/pkg/types"
	"github.com/libp2p/go-libp2p/core/peer"
)

type ExecuteRequestResponse struct {
	MessageType string `json:"message_type"`
	Message     any    `json:"message"`
}

// handleExecuteRequest
func (m *Manager) handleExecuteRequest(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
	var req marketplace.ExecuteRequest
	if err := json.Unmarshal(data, &req); err != nil {
		fmt.Println("Error:", err.Error())
		resp := ExecuteRequestResponse{
			MessageType: marketplace.MsgTypeExecError,
			Message:     fmt.Sprintf("failed to unmarshal execute request: %v", err),
		}
		respData, _ := json.Marshal(resp)
		return respData, fmt.Errorf("failed to unmarshal execute request: %w", err)
	}
	callerID := req.CallerDID
	msgType, msgData, err := m.httpExecuteAndRespond(ctx, req.CorrelationID, req.Resource, req.Body, callerID)
	if err != nil {
		fmt.Println("Error in httpExecuteAndRespond:", err.Error())
		resp := ExecuteRequestResponse{
			MessageType: marketplace.MsgTypeExecError,
			Message:     fmt.Sprintf("execution failed: %v", err),
		}
		respData, _ := json.Marshal(resp)
		return respData, fmt.Errorf("execution failed: %w", err)
	}
	resp := ExecuteRequestResponse{
		MessageType: msgType,
		Message:     json.RawMessage(msgData),
	}
	respData, _ := json.Marshal(resp)
	m.streamHandler.SendMessage(ctx, from, msgType, msgData)
	return respData, nil
}

// handleX402Request is the server-side handler for x402.request messages.
// If the agent requires payment and the request carries no payment, it issues a challenge nonce
// and returns x402.payment_required. If payment is already attached (preemptive), it is
// forwarded to handleX402WithPayment. Free agents are executed directly.
func (m *Manager) handleX402Request(ctx context.Context, from peer.ID, _ string, data []byte) (string, []byte, error) {
	var req marketplace.X402Request
	if err := json.Unmarshal(data, &req); err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrInvalidMessage,
			fmt.Sprintf("failed to unmarshal x402.request: %v", err))
	}

	fmt.Printf("[handleX402Request] peer=%s corr=%s resource=%s\n", from, req.CorrelationID, req.Resource)

	price := m.agentPrice(req.Resource)

	// Preemptive payment provided by the client.
	if req.Payment != nil {
		return m.handleX402WithPayment(ctx, from, &req, price, req.Payment)
	}

	// No payment — free agent, execute directly.
	if price == 0 {
		callerID := req.CallerDID
		if callerID == "" {
			callerID = from.String()
		}
		return m.executeAndRespond(ctx, req.CorrelationID, req.Resource, req.Body, callerID)
	}

	// Payment required: generate challenge nonce.
	if m.paymentService == nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrPaymentRequired,
			"payment service not configured on seller")
	}

	challenge, err := m.paymentService.GenerateChallenge(req.CorrelationID, 5*time.Minute)
	if err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrPaymentRequired,
			fmt.Sprintf("failed to generate challenge: %v", err))
	}

	payReq, err := m.paymentService.CreateRequirement(m.walletAddress,
		fmt.Sprintf("%d", int(price*1e6)))
	if err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrPaymentRequired,
			fmt.Sprintf("failed to create payment requirement: %v", err))
	}

	pr := marketplace.X402PaymentRequired{
		Version:             marketplace.X402LibP2PVersion,
		CorrelationID:       req.CorrelationID,
		ChallengeNonce:      challenge.Nonce,
		ChallengeExpiresAt:  challenge.ExpiresAt.Unix(),
		PaymentRequirements: payReq,
		Message:             "Payment required",
		SellerDID:           req.Resource,
	}
	respData, err := json.Marshal(pr)
	if err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrPaymentRequired, err.Error())
	}

	fmt.Printf("[handleX402Request] issued challenge nonce=%s corr=%s\n", challenge.Nonce, req.CorrelationID)
	return marketplace.MsgTypeX402PaymentRequired, respData, nil
}

// handleX402PaidRequest is the server-side handler for x402.paid_request messages.
func (m *Manager) handleX402PaidRequest(ctx context.Context, from peer.ID, _ string, data []byte) (string, []byte, error) {
	var req marketplace.X402PaidRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrInvalidMessage,
			fmt.Sprintf("failed to unmarshal x402.paid_request: %v", err))
	}

	fmt.Printf("[handleX402PaidRequest] peer=%s corr=%s server_nonce=%s\n", from, req.CorrelationID, req.Payment.ServerNonce)

	price := m.agentPrice(req.Payment.Payer) // fallback; actual agent is embedded in body
	// The resource is not in paid_request directly; determine from the original x402.request
	// which stored it before the challenge. We'll use the payer/resource from the payment envelope.
	// NOTE: The agent resource is encoded in req.Body (decoded below).

	// Standard flow: validate challenge nonce matches what was issued.
	if req.Payment.ServerNonce != marketplace.PreemptiveNonce {
		challenge, err := m.paymentService.ConsumeChallenge(req.CorrelationID)
		if err != nil {
			return sendX402Error(req.CorrelationID, marketplace.ErrNonceExpired,
				fmt.Sprintf("challenge expired or unknown: %v", err))
		}
		if challenge.Nonce != req.Payment.ServerNonce {
			return sendX402Error(req.CorrelationID, marketplace.ErrNonceMismatch,
				fmt.Sprintf("nonce mismatch: expected %s, got %s", challenge.Nonce, req.Payment.ServerNonce))
		}
		// Also verify the EIP-712 auth nonce matches.
		if req.Payment.Payload != nil && req.Payment.Payload.Authorization.Nonce != "" {
			authNonce := req.Payment.Payload.Authorization.Nonce
			if strings.HasPrefix(authNonce, "0x") || strings.HasPrefix(authNonce, "0X") {
				authNonce = authNonce[2:]
			}
			if authNonce != challenge.Nonce {
				return sendX402Error(req.CorrelationID, marketplace.ErrNonceMismatch,
					"EIP-712 auth nonce does not match challenge nonce")
			}
		}
	}

	header := envelopeToPaymentHeader(&req.Payment)

	// Decode the body to find the resource (agent ID) and input.
	var bodyPayload types.AgentRequest
	if len(req.Body) > 0 {
		_ = json.Unmarshal(req.Body, &bodyPayload)
	}
	resource := bodyPayload.Resource
	if resource == "" {
		return sendX402Error(req.CorrelationID, marketplace.ErrInvalidMessage, "missing resource in body")
	}

	price = m.agentPrice(resource)

	if m.paymentService == nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrPaymentInvalid, "payment service not configured")
	}

	expectedAmount := big.NewInt(int64(price * 1e6))
	txHash, err := m.paymentService.VerifyAndSettle(ctx, header, expectedAmount)
	if err != nil {
		fmt.Printf("[handleX402PaidRequest] VerifyAndSettle failed: %v\n", err)
		return sendX402Error(req.CorrelationID, marketplace.ErrSettlementFailed,
			fmt.Sprintf("payment verification/settlement failed: %v", err))
	}

	fmt.Printf("[handleX402PaidRequest] payment settled txHash=%s\n", txHash)

	output, err := m.ExecuteTask(context.WithValue(ctx, ctxKeySkipSession, true), resource, bodyPayload)
	if err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrExecutionFailed, err.Error())
	}

	callerID := req.CallerDID
	if callerID == "" {
		callerID = from.String()
	}

	if m.sessionStore != nil {
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
		_ = m.sessionStore.AddExchange(ctx, resource, callerID, ex)
	}

	respBody, _ := json.Marshal(map[string]string{"output": output})
	resp := marketplace.X402Response{
		Version:       marketplace.X402LibP2PVersion,
		CorrelationID: req.CorrelationID,
		PaymentID:     header.PaymentID,
		TxHash:        txHash,
		Body:          respBody,
		SellerDID:     resource,
		SellerTokenID: m.agentTokenIDString(resource),
	}
	respData, err := json.Marshal(resp)
	if err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrExecutionFailed, err.Error())
	}
	return marketplace.MsgTypeX402Response, respData, nil
}

// handleX402WithPayment handles a preemptive-payment path from handleX402Request.
func (m *Manager) handleX402WithPayment(ctx context.Context, from peer.ID, req *marketplace.X402Request, price float64, env *marketplace.X402PaymentEnvelope) (string, []byte, error) {
	fmt.Printf("[handleX402WithPayment] preemptive payment corr=%s\n", req.CorrelationID)

	if m.paymentService == nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrPaymentInvalid, "payment service not configured")
	}

	header := envelopeToPaymentHeader(env)
	expectedAmount := big.NewInt(int64(price * 1e6))

	txHash, err := m.paymentService.VerifyAndSettle(ctx, header, expectedAmount)
	if err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrSettlementFailed,
			fmt.Sprintf("settlement failed: %v", err))
	}

	var bodyPayload types.AgentRequest
	if len(req.Body) > 0 {
		_ = json.Unmarshal(req.Body, &bodyPayload)
	}

	output, err := m.ExecuteTask(context.WithValue(ctx, ctxKeySkipSession, true), req.Resource, bodyPayload)
	if err != nil {
		return sendX402Error(req.CorrelationID, marketplace.ErrExecutionFailed, err.Error())
	}

	callerID := req.CallerDID
	if callerID == "" {
		callerID = from.String()
	}

	if m.sessionStore != nil {
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
		_ = m.sessionStore.AddExchange(ctx, req.Resource, callerID, ex)
	}

	respBody, _ := json.Marshal(map[string]string{"output": output})
	resp := marketplace.X402Response{
		Version:       marketplace.X402LibP2PVersion,
		CorrelationID: req.CorrelationID,
		PaymentID:     header.PaymentID,
		TxHash:        txHash,
		Body:          respBody,
		SellerDID:     req.Resource,
		SellerTokenID: m.agentTokenIDString(req.Resource),
	}
	respData, _ := json.Marshal(resp)
	return marketplace.MsgTypeX402Response, respData, nil
}

// httpExecuteAndRespond is the HTTP handler for executing an agent task via REST.
func (m *Manager) httpExecuteAndRespond(ctx context.Context, correlationID, resource string, rawBody []byte, callerID string) (string, []byte, error) {
	var bodyPayload types.AgentRequest
	if len(rawBody) > 0 {
		_ = json.Unmarshal(rawBody, &bodyPayload)
	}
	apiURL := ""
	if m.listingService != nil {
		if listing, ok := m.listingService.GetListing(resource); ok {
			apiURL = listing.AgentAPI
		}
	}

	if apiURL == "" {
		return marketplace.MsgTypeExecError, nil, fmt.Errorf("agent API URL not found for resource: %s", resource)
	}

	resp, err := http.Post(apiURL, "application/json", strings.NewReader(string(rawBody)))
	if err != nil {
		fmt.Println("Error making HTTP request:", err.Error())
		return marketplace.MsgTypeExecError, nil, fmt.Errorf("error making HTTP request: %w", err)
	}
	defer resp.Body.Close()
	var execResp types.AgentResponse
	if err := json.NewDecoder(resp.Body).Decode(&execResp); err != nil {
		fmt.Println("Error decoding HTTP response:", err.Error())
		return marketplace.MsgTypeExecError, nil, fmt.Errorf("error decoding HTTP response: %w", err)
	}
	// Send HTTP response back to the peer through libp2p.
	if hasX402PaymentRequired(execResp) {
		output, err := json.Marshal(execResp)
		if err != nil {
			return marketplace.MsgTypeExecError, nil, fmt.Errorf("error marshaling payment required response: %w", err)
		}
		return marketplace.MsgTypeExecPaymentRequired, output, nil
	}
	return marketplace.MsgTypeExecPaymentRequired, nil, nil
}

// executeAndRespond executes a free agent and returns an x402.response.
func (m *Manager) executeAndRespond(ctx context.Context, correlationID, resource string, rawBody []byte, callerID string) (string, []byte, error) {
	var bodyPayload types.AgentRequest
	if len(rawBody) > 0 {
		_ = json.Unmarshal(rawBody, &bodyPayload)
	}

	output, err := m.ExecuteTask(context.WithValue(ctx, ctxKeySkipSession, true), resource, bodyPayload)
	if err != nil {
		return sendX402Error(correlationID, marketplace.ErrExecutionFailed, err.Error())
	}

	if m.sessionStore != nil {
		ex := types.Exchange{
			RequestID: correlationID,
			Input:     bodyPayload.Input,
			Output:    output,
			Timestamp: time.Now().UTC(),
		}
		_ = m.sessionStore.AddExchange(ctx, resource, callerID, ex)
	}

	respBody, _ := json.Marshal(map[string]string{"output": output})
	resp := marketplace.X402Response{
		Version:       marketplace.X402LibP2PVersion,
		CorrelationID: correlationID,
		Body:          respBody,
		SellerDID:     resource,
		SellerTokenID: m.agentTokenIDString(resource),
	}
	respData, _ := json.Marshal(resp)
	return marketplace.MsgTypeX402Response, respData, nil
}
