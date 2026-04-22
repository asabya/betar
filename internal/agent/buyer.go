package agent

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/pkg/types"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

type AgentSuccessResponse struct {
	Root struct {
		MessageType string              `json:"message_type"`
		Message     types.AgentResponse `json:"message"`
	} `json:"root"`
}

// handleExecuteResponse
func (m *Manager) handleExecuteResponse(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
	fmt.Println("========>> Will handle execute responses from peers.")
	return nil, nil
}

// handleExecutePaymentRequired
func (m *Manager) handleExecutePaymentRequired(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
	fmt.Println("========>> Will handle execute payment required messages from peers.")
	return data, nil
}

// handleErrorResponse handles incoming error messages from peers.
func (m *Manager) handleErrorResponse(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
	fmt.Println("========>> Will handle error responses from peers.")
	return nil, nil
}

// DiscoverAgents discovers agents from the marketplace
func (m *Manager) DiscoverAgents(ctx context.Context) ([]types.AgentListing, error) {
	// In a real implementation, this would subscribe to pubsub and collect listings
	// For now, return empty list
	return []types.AgentListing{}, nil
}

// FindListingByAgentID is the exported version of findListingByAgentID.
func (m *Manager) FindListingByAgentID(agentID string) (*types.AgentListing, string) {
	return m.findListingByAgentID(agentID)
}

// connectToPeer connects to a remote peer by seller ID and addresses.
// Falls back to peer-only connection if no addresses are provided.
func (m *Manager) connectToPeer(ctx context.Context, sellerID string, addrs []string) (peer.ID, error) {
	fmt.Printf("[connectToPeer] Attempting to connect to peer: %s\n", sellerID)

	peerID, err := peer.Decode(sellerID)
	if err != nil {
		fmt.Printf("[connectToPeer] Invalid peer ID %s: %v\n", sellerID, err)
		return "", fmt.Errorf("invalid peer ID: %w", err)
	}
	fmt.Printf("[connectToPeer] Decoded peer ID: %s\n", peerID)

	// If no addresses provided, try peer-only connection
	if len(addrs) == 0 {
		fmt.Printf("[connectToPeer] No addresses provided, attempting peer-only connection\n")
		if err := m.p2pHost.Connect(ctx, peer.AddrInfo{ID: peerID}); err != nil {
			fmt.Printf("[connectToPeer] Peer-only connection failed: %v\n", err)
			return "", err
		}
		fmt.Printf("[connectToPeer] Peer-only connection successful\n")
		return peerID, nil
	}

	fmt.Printf("[connectToPeer] Attempting to connect using %d addresses\n", len(addrs))

	// Parse addresses and try to connect
	var connectErr error
	for i, rawAddr := range addrs {
		addr, err := multiaddr.NewMultiaddr(rawAddr)
		if err != nil {
			fmt.Printf("[connectToPeer] Failed to parse address %d (%s): %v\n", i, rawAddr, err)
			continue
		}
		fmt.Printf("[connectToPeer] Trying address %d: %s\n", i, rawAddr)
		info := peer.AddrInfo{ID: peerID, Addrs: []multiaddr.Multiaddr{addr}}
		if err := m.p2pHost.Connect(ctx, info); err == nil {
			fmt.Printf("[connectToPeer] Successfully connected via address %d\n", i)
			return peerID, nil
		} else {
			fmt.Printf("[connectToPeer] Connection failed for address %d: %v\n", i, err)
			connectErr = err
		}
	}

	fmt.Printf("[connectToPeer] All address connections failed, falling back to peer-only\n")

	// Fallback to peer-only connection if address connections fail
	if err := m.p2pHost.Connect(ctx, peer.AddrInfo{ID: peerID}); err != nil {
		fmt.Printf("[connectToPeer] Fallback peer-only connection failed: %v\n", err)
		if connectErr != nil {
			return "", connectErr
		}
		return "", err
	}

	fmt.Printf("[connectToPeer] Fallback peer-only connection successful\n")
	return peerID, nil
}

func (m *Manager) RemoteExecute(ctx context.Context, peerID peer.ID, agentID string, reqbody []byte) (string, error) {
	request := types.AgentRequest{}
	if err := json.Unmarshal(reqbody, &request); err != nil {
		return "", fmt.Errorf("failed to unmarshal agent request body: %w", err)
	}
	// TODO: can we use raw request instead of constructing AgentRequest here? need to match the expected format on the seller side
	if m.streamHandler == nil {
		return "", fmt.Errorf("stream handler not configured")
	}
	correlationID := uuid.New().String()
	req := marketplace.ExecuteRequest{
		Version:       marketplace.ExecLibP2PVersion,
		CorrelationID: correlationID,
		Resource:      agentID,
		Method:        "execute",
		Body:          reqbody,
		CallerDID:     m.nodeDID,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal execute request: %w", err)
	}
	fmt.Printf("[RemoteExecute] sending request to %s corr=%s\n", peerID, correlationID)
	resp, err := m.streamHandler.SendMessage(ctx, peerID, marketplace.MsgTypeExecRequest, reqData)

	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	// TODO: send proper formatted respose to match SendMessageSuccessResponse
	return string(resp), nil
}

// RemoteExecuteX402 executes a task on a remote agent using the /x402/libp2p/1.0.0 protocol.
// It performs the standard 2-trip flow: send x402.request → receive x402.payment_required →
// sign with challenge nonce → send x402.paid_request → receive x402.response.
func (m *Manager) RemoteExecuteX402(ctx context.Context, peerID peer.ID, agentID, input string) (string, error) {
	if m.x402StreamHandler == nil {
		return "", fmt.Errorf("x402 stream handler not configured")
	}

	correlationID := uuid.New().String()
	bodyPayload := map[string]string{"resource": agentID, "input": input}
	bodyBytes, _ := json.Marshal(bodyPayload)

	req := marketplace.X402Request{
		Version:       marketplace.X402LibP2PVersion,
		CorrelationID: correlationID,
		Resource:      agentID,
		Method:        "execute",
		Payment:       nil,
		Body:          bodyBytes,
		CallerDID:     m.nodeDID,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal x402.request: %w", err)
	}

	fmt.Printf("[RemoteExecuteX402] sending x402.request to %s corr=%s\n", peerID, correlationID)
	respType, respData, err := m.x402StreamHandler.SendX402Message(ctx, peerID, marketplace.MsgTypeX402Request, reqData)
	if err != nil {
		return "", fmt.Errorf("x402.request failed: %w", err)
	}

	switch respType {
	case marketplace.MsgTypeX402Response:
		output, err := extractX402Output(respData)
		if err == nil && m.sessionStore != nil {
			ex := types.Exchange{
				RequestID: correlationID,
				Input:     input,
				Output:    output,
				Timestamp: time.Now().UTC(),
			}
			_ = m.sessionStore.AddExchange(ctx, agentID, m.nodeDID, ex)
		}
		return output, err

	// Betar client handling PaymentRequired message and signing PaymentRequest
	case marketplace.MsgTypeX402PaymentRequired:
		var pr marketplace.X402PaymentRequired
		if err := json.Unmarshal(respData, &pr); err != nil {
			return "", fmt.Errorf("failed to unmarshal x402.payment_required: %w", err)
		}
		fmt.Printf("[RemoteExecuteX402] received payment_required challenge_nonce=%s\n", pr.ChallengeNonce)

		if m.paymentService == nil {
			return "", fmt.Errorf("payment service not configured; cannot pay for x402 agent")
		}

		header, err := m.paymentService.SignRequirementWithNonce(pr.PaymentRequirements, pr.ChallengeNonce)
		if err != nil {
			return "", fmt.Errorf("failed to sign payment with nonce: %w", err)
		}

		env := paymentHeaderToEnvelope(header, pr.ChallengeNonce)
		paidReq := marketplace.X402PaidRequest{
			Version:       marketplace.X402LibP2PVersion,
			CorrelationID: correlationID,
			Payment:       env,
			Body:          bodyBytes,
			CallerDID:     m.nodeDID,
		}
		paidData, err := json.Marshal(paidReq)
		if err != nil {
			return "", fmt.Errorf("failed to marshal x402.paid_request: %w", err)
		}

		fmt.Printf("[RemoteExecuteX402] sending x402.paid_request to %s corr=%s\n", peerID, correlationID)
		respType2, respData2, err := m.x402StreamHandler.SendX402Message(ctx, peerID, marketplace.MsgTypeX402PaidRequest, paidData)
		if err != nil {
			return "", fmt.Errorf("x402.paid_request failed: %w", err)
		}

		switch respType2 {
		case marketplace.MsgTypeX402Response:
			output, err := extractX402Output(respData2)
			if err == nil {
				var resp marketplace.X402Response
				_ = json.Unmarshal(respData2, &resp)
				if m.sessionStore != nil {
					ex := types.Exchange{
						RequestID: correlationID,
						Input:     input,
						Output:    output,
						Timestamp: time.Now().UTC(),
						Payment: &types.PaymentRecord{
							PaymentID: resp.PaymentID,
							TxHash:    resp.TxHash,
							Amount:    pr.PaymentRequirements.Amount,
							Payer:     m.walletAddress,
							PaidAt:    time.Now().UTC(),
						},
					}
					_ = m.sessionStore.AddExchange(ctx, agentID, m.nodeDID, ex)
				}
				// Auto-submit reputation feedback after paid execution
				if m.eip8004 != nil && resp.SellerTokenID != "" {
					go func() {
						tokenID := new(big.Int)
						if _, ok := tokenID.SetString(resp.SellerTokenID, 10); !ok {
							return
						}
						hash := sha256.Sum256([]byte(correlationID))
						var feedbackHash [32]byte
						copy(feedbackHash[:], hash[:])
						fbErr := m.eip8004.GiveFeedback(context.Background(), tokenID,
							100, 0, "execution", "", agentID, "", feedbackHash)
						if fbErr != nil {
							fmt.Printf("[RemoteExecuteX402] reputation feedback failed: %v\n", fbErr)
						} else {
							fmt.Printf("[RemoteExecuteX402] reputation feedback submitted for tokenID=%s\n", resp.SellerTokenID)
						}
					}()
				}
			}
			return output, err
		case marketplace.MsgTypeX402Error:
			return extractX402ErrorMessage(respData2)
		default:
			return "", fmt.Errorf("unexpected response type to paid_request: %s", respType2)
		}

	case marketplace.MsgTypeX402Error:
		return extractX402ErrorMessage(respData)

	default:
		return "", fmt.Errorf("unexpected response type to x402.request: %s", respType)
	}
}

// ConnectToAgent connects to a remote agent via P2P
func (m *Manager) ConnectToAgent(ctx context.Context, peerID peer.ID) error {
	return m.p2pHost.Connect(ctx, peer.AddrInfo{ID: peerID})
}
