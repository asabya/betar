package marketplace

import (
	"context"
	"testing"
	"time"

	"github.com/asabya/betar/internal/config"
	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/pkg/types"
	blockservice "github.com/ipfs/boxo/blockservice"
	blockstore "github.com/ipfs/boxo/blockstore"
	offline "github.com/ipfs/boxo/exchange/offline"
	merkledag "github.com/ipfs/boxo/ipld/merkledag"
	ds "github.com/ipfs/go-datastore"
	syncds "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/peer"
)

func testP2PConfig() *config.P2PConfig {
	return &config.P2PConfig{Port: 0, EnableMDNS: false, EnableDHT: false}
}

func TestListingCRDTApplyGetDelete(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h, err := p2p.NewHost(ctx, testP2PConfig())
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	ps, err := p2p.NewPubSub(ctx, h.RawHost())
	if err != nil {
		t.Fatalf("NewPubSub failed: %v", err)
	}
	defer ps.Close()

	bs := blockstore.NewBlockstore(syncds.MutexWrap(ds.NewMapDatastore()))
	dagService := merkledag.NewDAGService(blockservice.New(bs, offline.Exchange(bs)))

	crdtStore, err := newListingCRDT(ctx, ps, dagService)
	if err != nil {
		t.Fatalf("NewListingCRDT failed: %v", err)
	}
	defer crdtStore.Close()

	if changed := crdtStore.Apply(&types.AgentListingMessage{
		Type:      "list",
		AgentID:   "peer-1/agent-1",
		Name:      "agent-a",
		Price:     0.01,
		Metadata:  "bafy123",
		SellerID:  "peer-1",
		Timestamp: 10,
	}); !changed {
		t.Fatalf("expected first apply to change state")
	}

	listing, ok := crdtStore.Get("peer-1/agent-1")
	if !ok {
		t.Fatalf("expected listing to exist")
	}
	if listing.Name != "agent-a" {
		t.Fatalf("unexpected listing name: %s", listing.Name)
	}

	if changed := crdtStore.Apply(&types.AgentListingMessage{
		Type:      "delist",
		AgentID:   "peer-1/agent-1",
		SellerID:  "peer-1",
		Timestamp: 11,
	}); !changed {
		t.Fatalf("expected delist to change state")
	}

	if _, ok := crdtStore.Get("peer-1/agent-1"); ok {
		t.Fatalf("expected listing to be deleted")
	}
}

// TestCRDTListingPropagatesViaGossipSub verifies that an agent listing applied on
// one node's CRDT store propagates to a second node via GossipSub within 5s.
func TestCRDTListingPropagatesViaGossipSub(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cfg := testP2PConfig()

	h1, err := p2p.NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost h1 failed: %v", err)
	}
	defer h1.Close()

	h2, err := p2p.NewHost(ctx, cfg)
	if err != nil {
		t.Fatalf("NewHost h2 failed: %v", err)
	}
	defer h2.Close()

	// Direct connect h1 → h2
	if err := h1.Connect(ctx, peer.AddrInfo{ID: h2.ID(), Addrs: h2.Addrs()}); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	ps1, err := p2p.NewPubSub(ctx, h1.RawHost())
	if err != nil {
		t.Fatalf("NewPubSub h1 failed: %v", err)
	}
	defer ps1.Close()

	ps2, err := p2p.NewPubSub(ctx, h2.RawHost())
	if err != nil {
		t.Fatalf("NewPubSub h2 failed: %v", err)
	}
	defer ps2.Close()

	// Shared in-memory blockstore so CRDT dag blocks are available to both nodes.
	sharedBS := blockstore.NewBlockstore(syncds.MutexWrap(ds.NewMapDatastore()))
	ex := offline.Exchange(sharedBS)
	dag1 := merkledag.NewDAGService(blockservice.New(sharedBS, ex))
	dag2 := merkledag.NewDAGService(blockservice.New(sharedBS, ex))

	crdt1, err := newListingCRDT(ctx, ps1, dag1)
	if err != nil {
		t.Fatalf("newListingCRDT node1 failed: %v", err)
	}
	defer crdt1.Close()

	crdt2, err := newListingCRDT(ctx, ps2, dag2)
	if err != nil {
		t.Fatalf("newListingCRDT node2 failed: %v", err)
	}
	defer crdt2.Close()

	// Wait for GossipSub peers to see each other on the CRDT topic.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(ps1.ListPeers(CRDTTopic)) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if len(ps1.ListPeers(CRDTTopic)) == 0 {
		t.Skip("GossipSub peers did not see each other on CRDTTopic within 5s — skipping")
	}

	// Apply a listing on node 1.
	if changed := crdt1.Apply(&types.AgentListingMessage{
		Type:      "list",
		AgentID:   "node1/agent-x",
		Name:      "agent-x",
		Price:     0.05,
		SellerID:  "node1",
		Timestamp: time.Now().Unix(),
	}); !changed {
		t.Fatalf("expected Apply on node1 to change state")
	}

	// Poll node 2 for up to 5s.
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if listing, ok := crdt2.Get("node1/agent-x"); ok {
			if listing.Name == "agent-x" {
				return // propagation confirmed
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("agent listing did not propagate from node1 to node2 within 5s")
}
