package marketplace

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/asabya/betar/internal/ipfs"
	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/pkg/types"
	ds "github.com/ipfs/go-datastore"
	query "github.com/ipfs/go-datastore/query"
	syncds "github.com/ipfs/go-datastore/sync"
	crdt "github.com/ipfs/go-ds-crdt"
	ipld "github.com/ipfs/go-ipld-format"
)

const (
	listingNamespace = "/marketplace"
	listingPrefix    = "/agents"
)

// ListingCRDT stores marketplace listings using go-ds-crdt.
type ListingCRDT struct {
	store *crdt.Datastore
	mu    sync.Mutex
}

// NewListingCRDT creates a ListingCRDT backed by go-ds-crdt.
func NewListingCRDT(ctx context.Context, ps *p2p.PubSub, ipfsClient *ipfs.Client) (*ListingCRDT, error) {
	if ps == nil || ps.Raw() == nil {
		return nil, fmt.Errorf("pubsub is required for marketplace crdt")
	}
	if ipfsClient == nil {
		return nil, fmt.Errorf("ipfs client is required for marketplace crdt")
	}

	dagService := ipfsClient.DAGService()
	if dagService == nil {
		return nil, fmt.Errorf("ipfs dag service is required for marketplace crdt")
	}
	return newListingCRDT(ctx, ps, dagService)
}

func newListingCRDT(ctx context.Context, ps *p2p.PubSub, dagService ipld.DAGService) (*ListingCRDT, error) {
	metaStore := syncds.MutexWrap(ds.NewMapDatastore())

	broadcaster, err := crdt.NewPubSubBroadcaster(ctx, ps.Raw(), CRDTTopic)
	if err != nil {
		return nil, fmt.Errorf("failed to create CRDT broadcaster: %w", err)
	}

	opts := crdt.DefaultOptions()
	opts.NumWorkers = 2

	store, err := crdt.New(metaStore, ds.NewKey(listingNamespace), dagService, broadcaster, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create listing CRDT store: %w", err)
	}

	return &ListingCRDT{store: store}, nil
}

// Apply applies a listing mutation and returns true if state changed.
func (c *ListingCRDT) Apply(msg *types.AgentListingMessage) bool {
	if c == nil || c.store == nil || msg == nil || msg.AgentID == "" {
		return false
	}

	ctx := context.Background()
	key := c.keyForAgent(msg.AgentID)

	c.mu.Lock()
	defer c.mu.Unlock()

	if msg.Type == "delist" {
		if err := c.store.Delete(ctx, key); err != nil {
			return false
		}
		return true
	}

	listing := types.AgentListing{
		ID:        msg.AgentID,
		Name:      msg.Name,
		Price:     msg.Price,
		Metadata:  msg.Metadata,
		SellerID:  msg.SellerID,
		Addrs:     msg.Addrs,
		Protocols: msg.Protocols,
		Timestamp: msg.Timestamp,
	}

	payload, err := json.Marshal(listing)
	if err != nil {
		return false
	}

	if err := c.store.Put(ctx, key, payload); err != nil {
		return false
	}

	return true
}

func (c *ListingCRDT) Get(agentID string) (*types.AgentListing, bool) {
	if c == nil || c.store == nil || agentID == "" {
		return nil, false
	}

	ctx := context.Background()
	data, err := c.store.Get(ctx, c.keyForAgent(agentID))
	if err != nil {
		return nil, false
	}

	var listing types.AgentListing
	if err := json.Unmarshal(data, &listing); err != nil {
		return nil, false
	}
	return &listing, true
}

func (c *ListingCRDT) List() []*types.AgentListing {
	if c == nil || c.store == nil {
		return nil
	}

	ctx := context.Background()
	results, err := c.store.Query(ctx, query.Query{Prefix: listingPrefix})
	if err != nil {
		return nil
	}
	defer results.Close()

	listings := make([]*types.AgentListing, 0)
	for r := range results.Next() {
		if r.Error != nil {
			continue
		}
		var listing types.AgentListing
		if err := json.Unmarshal(r.Value, &listing); err != nil {
			continue
		}
		copy := listing
		listings = append(listings, &copy)
	}

	return listings
}

func (c *ListingCRDT) Close() error {
	if c == nil || c.store == nil {
		return nil
	}
	return c.store.Close()
}

func (c *ListingCRDT) keyForAgent(agentID string) ds.Key {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(agentID))
	return ds.NewKey(listingPrefix).ChildString(encoded)
}
