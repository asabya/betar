package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/asabya/betar/internal/ipfs"
	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/pkg/types"
	"github.com/libp2p/go-libp2p/core/peer"
)

// AgentListingService handles agent listing and discovery
type AgentListingService struct {
	ipfsClient *ipfs.Client
	peerID     peer.ID
	crdt       *ListingCRDT
}

// NewAgentListingService creates a new agent listing service
func NewAgentListingService(ctx context.Context, ipfsClient *ipfs.Client, p2pPubSub *p2p.PubSub, peerID peer.ID) (*AgentListingService, error) {
	listingCRDT, err := NewListingCRDT(ctx, p2pPubSub, ipfsClient)
	if err != nil {
		return nil, err
	}

	s := &AgentListingService{
		ipfsClient: ipfsClient,
		peerID:     peerID,
		crdt:       listingCRDT,
	}

	return s, nil
}

// handleAgentListingSubscription handles incoming agent listing messages
func (s *AgentListingService) handleAgentListingMessage(ctx context.Context, listing *types.AgentListingMessage) error {
	_ = ctx
	if listing == nil {
		return nil
	}

	s.crdt.Apply(listing)
	return nil
}

// ListAgent lists an agent on the marketplace
func (s *AgentListingService) ListAgent(ctx context.Context, listing *types.AgentListingMessage) error {
	_ = ctx
	_ = s.crdt.Apply(listing)
	return nil
}

// UpsertLocalListing applies listing to local CRDT state without publishing.
func (s *AgentListingService) UpsertLocalListing(listing *types.AgentListingMessage) {
	if listing == nil {
		return
	}
	_ = s.crdt.Apply(listing)
}

// UpdateListing updates an agent listing
func (s *AgentListingService) UpdateListing(ctx context.Context, listing *types.AgentListingMessage) error {
	listing.Type = "update"
	return s.ListAgent(ctx, listing)
}

// Delist removes an agent from the marketplace
func (s *AgentListingService) Delist(ctx context.Context, agentID string) error {
	listing := &types.AgentListingMessage{
		Type:      "delist",
		AgentID:   agentID,
		Timestamp: time.Now().Unix(),
	}

	_ = s.crdt.Apply(listing)
	_ = ctx
	listing.SellerID = s.peerID.String()
	return nil
}

// GetListing gets a listing by agent ID
func (s *AgentListingService) GetListing(agentID string) (*types.AgentListing, bool) {
	return s.crdt.Get(agentID)
}

// ListListings returns all listings
func (s *AgentListingService) ListListings() []*types.AgentListing {
	return s.crdt.List()
}

// GetAgentMetadata retrieves agent metadata from IPFS
func (s *AgentListingService) GetAgentMetadata(ctx context.Context, cid string) (*types.AgentRegistration, error) {
	var reg types.AgentRegistration
	err := s.ipfsClient.GetJSON(ctx, cid, &reg)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata from IPFS: %w", err)
	}
	return &reg, nil
}

// DiscoverAgents discovers agents from the network
func (s *AgentListingService) DiscoverAgents(ctx context.Context) ([]*types.AgentListing, error) {
	_ = ctx
	time.Sleep(200 * time.Millisecond)
	return s.ListListings(), nil
}

// Close closes marketplace listing resources.
func (s *AgentListingService) Close() error {
	if s == nil || s.crdt == nil {
		return nil
	}
	return s.crdt.Close()
}
