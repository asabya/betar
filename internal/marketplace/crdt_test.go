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
