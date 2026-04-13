package agent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/pkg/types"
)

// sendX402Error is a convenience helper that marshals an X402Error and returns the typed tuple.
func sendX402Error(correlationID string, code marketplace.X402ErrorCode, message string) (string, []byte, error) {
	e := marketplace.NewX402Error(correlationID, code, message)
	data, _ := json.Marshal(e)
	return marketplace.MsgTypeX402Error, data, nil
}

// envelopeToPaymentHeader converts an X402PaymentEnvelope to the legacy PaymentHeader type
// used by PaymentService.VerifyAndSettle.
func envelopeToPaymentHeader(env *marketplace.X402PaymentEnvelope) *marketplace.PaymentHeader {
	if env == nil {
		return nil
	}
	req := marketplace.CreatePaymentRequirements(
		env.Network,
		"", // Amount comes from Payload.Authorization.Value
		marketplace.GetUSDCAddress(env.Network),
		"", // PayTo comes from Payload.Authorization.To
		marketplace.DefaultTimeout,
	)
	if env.Payload != nil {
		req.Amount = env.Payload.Authorization.Value
		req.PayTo = env.Payload.Authorization.To
	}
	var sig string
	if env.Payload != nil {
		sig = env.Payload.Signature
	}
	return &marketplace.PaymentHeader{
		Requirement: req,
		Accepted:    &req,
		Payer:       env.Payer,
		PaymentID:   "",
		Signature:   sig,
		Payload:     env.Payload,
	}
}

// paymentHeaderToEnvelope converts a PaymentHeader to an X402PaymentEnvelope for the wire.
func paymentHeaderToEnvelope(ph *marketplace.PaymentHeader, serverNonce string) marketplace.X402PaymentEnvelope {
	return marketplace.X402PaymentEnvelope{
		X402Version: 2,
		Scheme:      ph.Requirement.Scheme,
		Network:     ph.Requirement.Network,
		ServerNonce: serverNonce,
		Payer:       ph.Payer,
		Payload:     ph.Payload,
	}
}

// extractX402Output parses an x402.response and returns the output string.
func extractX402Output(data []byte) (string, error) {
	var resp marketplace.X402Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal x402.response: %w", err)
	}
	fmt.Printf("[RemoteExecuteX402] tx_hash=%s payment_id=%s\n", resp.TxHash, resp.PaymentID)
	if len(resp.Body) == 0 {
		return "", nil
	}
	// Body is a JSON object with an "output" key; decode it.
	var body map[string]string
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		// Fallback: treat as base64-encoded raw string.
		decoded, err2 := base64.StdEncoding.DecodeString(string(resp.Body))
		if err2 != nil {
			return string(resp.Body), nil
		}
		return string(decoded), nil
	}
	return body["output"], nil
}

// extractX402ErrorMessage parses an x402.error and returns an error.
func extractX402ErrorMessage(data []byte) (string, error) {
	var e marketplace.X402Error
	if err := json.Unmarshal(data, &e); err != nil {
		return "", fmt.Errorf("x402 error (unparseable): %s", string(data))
	}
	return "", fmt.Errorf("x402 error %d (%s): %s", e.ErrorCode, e.ErrorName, e.Message)
}

func hasX402PaymentRequired(resp types.AgentResponse) bool {
	// Guard each level to avoid nil pointer dereference
	if resp.Result.ContextID == "" {
		return false
	}
	if resp.Result.ID == "" {
		return false
	}
	if resp.Result.Kind == "" {
		return false
	}
	if resp.Result.Status.State == "" {
		return false
	}
	if resp.Result.Status.Message.Kind == "" {
		return false
	}
	if resp.Result.Status.Message.MessageID == "" {
		return false
	}
	return true
}

// findListingByAgentID looks up a listing by agent ID in the CRDT.
// It tries direct lookup first, then searches all listings.
// Returns the listing and the runtime agent ID extracted from the listing.
func (m *Manager) findListingByAgentID(agentID string) (*types.AgentListing, string) {
	fmt.Printf("[findListingByAgentID] Searching CRDT for agentID: %s\n", agentID)

	// Try direct lookup by full agent ID
	if listing, ok := m.listingService.GetListing(agentID); ok {
		runtimeID := m.extractRuntimeAgentID(listing)
		fmt.Printf("[findListingByAgentID] Direct CRDT lookup successful - ListingID: %s, RuntimeID: %s\n", listing.ID, runtimeID)
		return listing, runtimeID
	}
	fmt.Printf("[findListingByAgentID] Direct lookup failed, searching all listings\n")

	// Search all listings
	listings := m.listingService.ListListings()
	fmt.Printf("[findListingByAgentID] Total listings in CRDT: %d\n", len(listings))

	for i, listing := range listings {
		if listing == nil {
			continue
		}

		// Check if listing.ID matches
		if listing.ID == agentID {
			runtimeID := m.extractRuntimeAgentID(listing)
			fmt.Printf("[findListingByAgentID] Found match at index %d by listing.ID\n", i)
			return listing, runtimeID
		}

		// Check if the runtime agent ID matches
		runtimeID := m.extractRuntimeAgentID(listing)
		if runtimeID == agentID {
			fmt.Printf("[findListingByAgentID] Found match at index %d by runtime agent ID\n", i)
			return listing, runtimeID
		}
	}

	fmt.Printf("[findListingByAgentID] Agent not found in any CRDT listings\n")
	return nil, ""
}

// extractRuntimeAgentID extracts the runtime agent ID from a listing.
// The listing.ID format is "peerID/runtimeAgentID" or just the runtime agent ID.
func (m *Manager) extractRuntimeAgentID(listing *types.AgentListing) string {
	if listing == nil {
		return ""
	}
	if listing.SellerID == "" {
		return listing.ID
	}
	prefix := listing.SellerID + "/"
	if strings.HasPrefix(listing.ID, prefix) {
		return strings.TrimPrefix(listing.ID, prefix)
	}
	return listing.ID
}
