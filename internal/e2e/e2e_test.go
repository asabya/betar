package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/asabya/betar/internal/agent"
	"github.com/asabya/betar/internal/ipfs"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/pkg/types"
	blockservice "github.com/ipfs/boxo/blockservice"
	blockstore "github.com/ipfs/boxo/blockstore"
	offline "github.com/ipfs/boxo/exchange/offline"
	merkledag "github.com/ipfs/boxo/ipld/merkledag"
	ds "github.com/ipfs/go-datastore"
	syncds "github.com/ipfs/go-datastore/sync"
	ipld "github.com/ipfs/go-ipld-format"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
)

type testPeer struct {
	host   host.Host
	pubsub *pubsub.PubSub
	crdt   *marketplace.ListingCRDT
	stream *p2p.StreamHandler
	dag    ipld.DAGService
}

func setupMockNet(t *testing.T, numPeers int) (mocknet.Mocknet, []*testPeer) {
	t.Helper()

	mn := mocknet.New()
	peers := make([]*testPeer, numPeers)

	sharedBs := blockstore.NewBlockstore(syncds.MutexWrap(ds.NewMapDatastore()))
	sharedDag := merkledag.NewDAGService(blockservice.New(sharedBs, offline.Exchange(sharedBs)))

	for i := 0; i < numPeers; i++ {
		h, err := mn.GenPeer()
		if err != nil {
			t.Fatalf("failed to generate peer %d: %v", i, err)
		}

		ps, err := pubsub.NewGossipSub(context.Background(), h,
			pubsub.WithMessageSigning(true),
			pubsub.WithStrictSignatureVerification(true),
		)
		if err != nil {
			t.Fatalf("failed to create pubsub for peer %d: %v", i, err)
		}

		peers[i] = &testPeer{
			host:   h,
			pubsub: ps,
			dag:    sharedDag,
			stream: p2p.NewStreamHandler(h),
		}
	}

	if err := mn.LinkAll(); err != nil {
		t.Fatalf("failed to link all peers: %v", err)
	}

	if err := mn.ConnectAllButSelf(); err != nil {
		t.Fatalf("failed to connect peers: %v", err)
	}

	return mn, peers
}

func (p *testPeer) setupCRDT(t *testing.T) {
	t.Helper()

	crdt, err := marketplace.NewListingCRDTWithPubSub(context.Background(), p.pubsub, p.dag)
	if err != nil {
		t.Fatalf("failed to create CRDT: %v", err)
	}
	p.crdt = crdt
}

func (p *testPeer) Close() {
	if p.crdt != nil {
		p.crdt.Close()
	}
	if p.host != nil {
		p.host.Close()
	}
}

func TestE2E_CRDTConvergence(t *testing.T) {
	t.Parallel()

	mn, peers := setupMockNet(t, 5)
	defer mn.Close()
	defer func() {
		for _, p := range peers {
			p.Close()
		}
	}()

	for i, p := range peers {
		p.setupCRDT(t)
		t.Logf("Peer %d: %s", i, p.host.ID().ShortString())
	}

	time.Sleep(500 * time.Millisecond)

	for i, p := range peers {
		agentID := fmt.Sprintf("%s/agent-%d", p.host.ID().ShortString(), i)
		msg := &types.AgentListingMessage{
			Type:      "list",
			AgentID:   agentID,
			Name:      fmt.Sprintf("Agent-%d", i),
			Price:     0.01 + float64(i)*0.001,
			SellerID:  p.host.ID().String(),
			Timestamp: time.Now().Unix(),
		}

		if !p.crdt.Apply(msg) {
			t.Fatalf("peer %d: failed to apply listing", i)
		}
		t.Logf("Peer %d listed agent: %s", i, agentID)
	}

	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

convergeLoop:
	for {
		select {
		case <-timeout:
			t.Fatalf("CRDT did not converge within timeout")
		case <-ticker.C:
			allConverged := true
			for i, p := range peers {
				listings := p.crdt.List()
				if len(listings) != 5 {
					allConverged = false
					t.Logf("Peer %d has %d listings, expected 5", i, len(listings))
					break
				}
			}
			if allConverged {
				break convergeLoop
			}
		}
	}

	for i, p := range peers {
		listings := p.crdt.List()
		if len(listings) != 5 {
			t.Errorf("peer %d: expected 5 listings, got %d", i, len(listings))
		}

		seen := make(map[string]bool)
		for _, l := range listings {
			seen[l.ID] = true
		}
		for j := 0; j < 5; j++ {
			expectedID := fmt.Sprintf("%s/agent-%d", peers[j].host.ID().ShortString(), j)
			if !seen[expectedID] {
				t.Errorf("peer %d: missing listing %s", i, expectedID)
			}
		}
	}
}

func TestE2E_StreamMessaging(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mn, peers := setupMockNet(t, 2)
	defer mn.Close()
	defer func() {
		for _, p := range peers {
			p.Close()
		}
	}()

	seller := peers[0]
	buyer := peers[1]

	seller.stream.RegisterHandler("execute", func(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
		return []byte("result:" + string(data)), nil
	})

	resp, err := buyer.stream.SendMessage(ctx, seller.host.ID(), "execute", []byte("hello"))
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	if string(resp) != "result:hello" {
		t.Fatalf("unexpected response: %s", string(resp))
	}
}

func TestE2E_AgentDiscoveryAndExecution(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mn, peers := setupMockNet(t, 3)
	defer mn.Close()
	defer func() {
		for _, p := range peers {
			p.Close()
		}
	}()

	for _, p := range peers {
		p.setupCRDT(t)
	}

	time.Sleep(500 * time.Millisecond)

	seller := peers[0]
	buyer1 := peers[1]
	buyer2 := peers[2]

	agentID := fmt.Sprintf("%s/agent-calculator", seller.host.ID().ShortString())
	seller.crdt.Apply(&types.AgentListingMessage{
		Type:      "list",
		AgentID:   agentID,
		Name:      "Calculator Agent",
		Price:     0.01,
		SellerID:  seller.host.ID().String(),
		Addrs:     []string{seller.host.Addrs()[0].String()},
		Protocols: []string{"/betar/marketplace/1.0.0"},
		Timestamp: time.Now().Unix(),
	})

	seller.stream.RegisterHandler("execute", func(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
		req := string(data)
		if req == "add:5+3" {
			return []byte("8"), nil
		}
		return []byte("unknown request"), nil
	})

	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

waitForListing:
	for {
		select {
		case <-timeout:
			t.Fatalf("listing not propagated")
		case <-ticker.C:
			if listing, ok := buyer1.crdt.Get(agentID); ok && listing != nil {
				break waitForListing
			}
		}
	}

	for _, buyer := range []*testPeer{buyer1, buyer2} {
		listing, ok := buyer.crdt.Get(agentID)
		if !ok || listing == nil {
			t.Fatalf("buyer %s: agent not found", buyer.host.ID().ShortString())
		}

		resp, err := buyer.stream.SendMessage(ctx, seller.host.ID(), "execute", []byte("add:5+3"))
		if err != nil {
			t.Fatalf("buyer %s: failed to execute: %v", buyer.host.ID().ShortString(), err)
		}
		if string(resp) != "8" {
			t.Fatalf("buyer %s: unexpected result: %s", buyer.host.ID().ShortString(), string(resp))
		}
	}
}

func TestE2E_DelistAgent(t *testing.T) {
	t.Parallel()

	mn, peers := setupMockNet(t, 2)
	defer mn.Close()
	defer func() {
		for _, p := range peers {
			p.Close()
		}
	}()

	for _, p := range peers {
		p.setupCRDT(t)
	}

	time.Sleep(300 * time.Millisecond)

	seller := peers[0]
	buyer := peers[1]

	agentID := fmt.Sprintf("%s/agent-temp", seller.host.ID().ShortString())
	seller.crdt.Apply(&types.AgentListingMessage{
		Type:      "list",
		AgentID:   agentID,
		Name:      "Temp Agent",
		Price:     0.01,
		SellerID:  seller.host.ID().String(),
		Timestamp: time.Now().Unix(),
	})

	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

waitForList:
	for {
		select {
		case <-timeout:
			t.Fatalf("listing not propagated")
		case <-ticker.C:
			if _, ok := buyer.crdt.Get(agentID); ok {
				break waitForList
			}
		}
	}

	if _, ok := buyer.crdt.Get(agentID); !ok {
		t.Fatalf("agent should be listed")
	}

	seller.crdt.Apply(&types.AgentListingMessage{
		Type:      "delist",
		AgentID:   agentID,
		SellerID:  seller.host.ID().String(),
		Timestamp: time.Now().Unix(),
	})

waitForDelist:
	for {
		select {
		case <-timeout:
			t.Fatalf("delist not propagated")
		case <-ticker.C:
			if _, ok := buyer.crdt.Get(agentID); !ok {
				break waitForDelist
			}
		}
	}

	if _, ok := buyer.crdt.Get(agentID); ok {
		t.Fatalf("agent should be delisted")
	}
}

func TestE2E_MultipleSellers(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mn, peers := setupMockNet(t, 4)
	defer mn.Close()
	defer func() {
		for _, p := range peers {
			p.Close()
		}
	}()

	for _, p := range peers {
		p.setupCRDT(t)
	}

	time.Sleep(500 * time.Millisecond)

	for i := 0; i < 3; i++ {
		seller := peers[i]
		agentID := fmt.Sprintf("%s/service-%d", seller.host.ID().ShortString(), i)
		seller.crdt.Apply(&types.AgentListingMessage{
			Type:      "list",
			AgentID:   agentID,
			Name:      fmt.Sprintf("Service %d", i),
			Price:     0.01 * float64(i+1),
			SellerID:  seller.host.ID().String(),
			Timestamp: time.Now().Unix(),
		})

		idx := i
		seller.stream.RegisterHandler("execute", func(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
			return []byte(fmt.Sprintf("service-%d-response", idx)), nil
		})
	}

	buyer := peers[3]

	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

waitForListings:
	for {
		select {
		case <-timeout:
			t.Fatalf("listings not propagated")
		case <-ticker.C:
			if len(buyer.crdt.List()) >= 3 {
				break waitForListings
			}
		}
	}

	listings := buyer.crdt.List()
	if len(listings) != 3 {
		t.Fatalf("expected 3 listings, got %d", len(listings))
	}

	for i := 0; i < 3; i++ {
		seller := peers[i]
		resp, err := buyer.stream.SendMessage(ctx, seller.host.ID(), "execute", []byte("test"))
		if err != nil {
			t.Fatalf("failed to execute on seller %d: %v", i, err)
		}
		expected := fmt.Sprintf("service-%d-response", i)
		if string(resp) != expected {
			t.Fatalf("seller %d: expected %s, got %s", i, expected, string(resp))
		}
	}
}

func TestE2E_MockLLMExecution(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mn, peers := setupMockNet(t, 2)
	defer mn.Close()
	defer func() {
		for _, p := range peers {
			p.Close()
		}
	}()

	for _, p := range peers {
		p.setupCRDT(t)
	}

	time.Sleep(500 * time.Millisecond)

	seller := peers[0]
	buyer := peers[1]

	tmpDir, err := os.MkdirTemp("", "betar-ipfs-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ipfsClient, err := ipfs.NewClient(ctx, seller.host, nil, tmpDir)
	if err != nil {
		t.Fatalf("failed to create ipfs client: %v", err)
	}
	defer ipfsClient.Close()

	responses := map[string]string{
		"sum":   "The sum is 15",
		"greet": "Hello! How can I help?",
	}
	mockRuntime := agent.NewMockRuntime(responses)
	mockMgr := agent.NewManagerWithRuntime(mockRuntime, ipfsClient, p2p.NewHostFromRaw(seller.host), nil, nil, "", nil)

	spec := agent.AgentSpec{
		Name:        "MathHelper",
		Description: "A math helper agent",
		Price:       0,
	}
	localAgent, err := mockMgr.RegisterAgent(ctx, spec)
	if err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	agentID := seller.host.ID().String() + "/" + localAgent.AgentID
	seller.crdt.Apply(&types.AgentListingMessage{
		Type:      "list",
		AgentID:   agentID,
		Name:      "MathHelper",
		Price:     0,
		SellerID:  seller.host.ID().String(),
		Addrs:     []string{seller.host.Addrs()[0].String()},
		Timestamp: time.Now().Unix(),
	})

	seller.stream.RegisterHandler("execute", func(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
		input := string(data)
		result, err := mockMgr.ExecuteTask(ctx, localAgent.AgentID, input)
		if err != nil {
			return nil, err
		}
		return []byte(result), nil
	})

	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

waitForListing:
	for {
		select {
		case <-timeout:
			t.Fatalf("listing not propagated")
		case <-ticker.C:
			if _, ok := buyer.crdt.Get(agentID); ok {
				break waitForListing
			}
		}
	}

	resp, err := buyer.stream.SendMessage(ctx, seller.host.ID(), "execute", []byte("sum of 5 and 10"))
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}
	if string(resp) != "The sum is 15" {
		t.Errorf("expected 'The sum is 15', got %q", string(resp))
	}
}
