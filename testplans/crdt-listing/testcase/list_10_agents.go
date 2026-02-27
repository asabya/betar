package testcase

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/asabya/betar/internal/config"
	"github.com/asabya/betar/internal/ipfs"
	"github.com/asabya/betar/internal/marketplace"
	"github.com/asabya/betar/internal/p2p"
	"github.com/asabya/betar/pkg/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

const (
	PeerTopicName               = "peer-addrs"
	ListingDoneState sync.State = "listing-done"
	VerifyState      sync.State = "verified"
)

type PeerAddr struct {
	PeerID string   `json:"peerID"`
	Addrs  []string `json:"addrs"`
	Seq    int64    `json:"seq"`
}

func List10Agents(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	runenv.RecordMessage("Starting list-10-agents test, instance %d/%d",
		initCtx.GlobalSeq, runenv.TestInstanceCount)

	tempDir := runenv.TestTempPath
	if tempDir == "" {
		tempDir = filepath.Join("/tmp", "betar-test", runenv.TestRun)
	}

	seqNum := initCtx.GlobalSeq
	agentID := fmt.Sprintf("agent-%d", seqNum)

	runenv.RecordMessage("Setting up local stack for %s", agentID)

	cfg := &config.P2PConfig{
		Port:        0,
		EnableMDNS:  false,
		EnableDHT:   false,
		EnableRelay: true,
	}

	h, err := p2p.NewHost(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create host: %w", err)
	}
	defer h.Close()

	ps, err := p2p.NewPubSub(ctx, h.RawHost())
	if err != nil {
		return fmt.Errorf("failed to create pubsub: %w", err)
	}
	defer ps.Close()

	ipfsClient, err := ipfs.NewClient(ctx, h.RawHost(), nil, tempDir)
	if err != nil {
		return fmt.Errorf("failed to create ipfs client: %w", err)
	}
	defer ipfsClient.Close()

	listingSvc, err := marketplace.NewAgentListingService(ctx, ipfsClient, ps, h.ID())
	if err != nil {
		return fmt.Errorf("failed to create listing service: %w", err)
	}
	defer listingSvc.Close()

	ownAddrs := h.AddrStrings()
	runenv.RecordMessage("My addresses: %v", ownAddrs)

	ownAddrs = append(ownAddrs, "/ip4/127.0.0.1/tcp/0")

	addrPayload := PeerAddr{
		PeerID: h.ID().String(),
		Addrs:  ownAddrs,
		Seq:    seqNum,
	}
	addrPayloadJSON, err := json.Marshal(addrPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal addr payload: %w", err)
	}

	runenv.RecordMessage("Publishing my address: %s", h.ID())

	peerTopic := sync.NewTopic(PeerTopicName, PeerAddr{})
	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()

	ch := make(chan PeerAddr)
	_, sub, err := initCtx.SyncClient.PublishSubscribe(subCtx, peerTopic, addrPayloadJSON, ch)
	if err != nil {
		return fmt.Errorf("failed to publish subscribe: %w", err)
	}
	_ = sub

	runenv.RecordMessage("Waiting for all peer addresses")

	allPeers := make(map[int64]PeerAddr)
	allPeers[seqNum] = addrPayload

	expectedPeers := runenv.TestInstanceCount
	received := 1

	for received < expectedPeers {
		select {
		case peerAddr := <-ch:
			if peerAddr.PeerID == h.ID().String() {
				continue
			}
			if _, exists := allPeers[peerAddr.Seq]; !exists {
				allPeers[peerAddr.Seq] = peerAddr
				received++
				runenv.RecordMessage("Received address from seq %d (%s), total %d/%d",
					peerAddr.Seq, peerAddr.PeerID, received, expectedPeers)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	runenv.RecordMessage("All peer addresses received, connecting for CRDT gossip")

	for seq, pa := range allPeers {
		if seq == seqNum {
			continue
		}

		pID, err := peer.Decode(pa.PeerID)
		if err != nil {
			runenv.RecordMessage("Failed to decode peer ID %s: %v", pa.PeerID, err)
			continue
		}

		var maddrs []multiaddr.Multiaddr
		for _, addr := range pa.Addrs {
			m, err := multiaddr.NewMultiaddr(addr)
			if err != nil {
				continue
			}
			maddrs = append(maddrs, m)
		}

		if len(maddrs) > 0 {
			pi := peer.AddrInfo{
				ID:    pID,
				Addrs: maddrs,
			}

			runenv.RecordMessage("Connecting to peer seq %d (%s)", seq, pID)
			if err := h.Connect(ctx, pi); err != nil {
				runenv.RecordMessage("Failed to connect to peer seq %d: %v", seq, err)
			} else {
				runenv.RecordMessage("Connected to peer seq %d (%s)", seq, pID)
			}
		}
	}

	time.Sleep(2 * time.Second)

	runenv.RecordMessage("Publishing my agent listing to CRDT")

	listingMsg := &types.AgentListingMessage{
		Type:      "list",
		AgentID:   fmt.Sprintf("%s/%s", h.ID().String(), agentID),
		Name:      agentID,
		Price:     0.01,
		Metadata:  "bafytest",
		SellerID:  h.ID().String(),
		Addrs:     ownAddrs,
		Protocols: []string{"/betar/execute/1.0.0"},
		Timestamp: time.Now().Unix(),
	}

	if err := listingSvc.ListAgent(ctx, listingMsg); err != nil {
		return fmt.Errorf("failed to list agent: %w", err)
	}

	runenv.RecordMessage("Listing published, signaling done")

	initCtx.SyncClient.MustSignalEntry(ctx, ListingDoneState)

	runenv.RecordMessage("Waiting for all instances to list their agents")

	initCtx.MustWaitAllInstancesInitialized(ctx)

	runenv.RecordMessage("All instances have listed, verifying CRDT convergence")

	if err := waitForConvergence(ctx, runenv, listingSvc, runenv.TestInstanceCount); err != nil {
		return err
	}

	runenv.RecordMessage("CRDT convergence verified, signaling completion")

	initCtx.SyncClient.MustSignalEntry(ctx, VerifyState)

	runenv.RecordMessage("Test completed successfully")

	return nil
}

func waitForConvergence(ctx context.Context, runenv *runtime.RunEnv,
	listingSvc *marketplace.AgentListingService, expectedCount int) error {

	maxAttempts := 30
	retryInterval := 2 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		listings := listingSvc.ListListings()
		uniqueListings := make(map[string]bool)
		for _, l := range listings {
			uniqueListings[l.ID] = true
		}

		count := len(uniqueListings)
		runenv.RecordMessage("Attempt %d/%d: found %d listings (expected %d)",
			attempt, maxAttempts, count, expectedCount)

		if count >= expectedCount {
			runenv.RecordMessage("CRDT converged: %d listings found", count)
			return nil
		}

		if attempt < maxAttempts {
			time.Sleep(retryInterval)
		}
	}

	listings := listingSvc.ListListings()
	return fmt.Errorf("CRDT convergence failed: expected at least %d listings, got %d. Listings: %v",
		expectedCount, len(listings), listings)
}
