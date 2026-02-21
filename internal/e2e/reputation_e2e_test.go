package e2e

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/asabya/betar/internal/agent"
	"github.com/asabya/betar/internal/eip8004"
	"github.com/asabya/betar/internal/eth"
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

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	identityAddr   = "0x5FbDB2315678afecb367f032d93F642f64180aa3"
	reputationAddr = "0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0"
	usdcAddr       = "0x0165878A594ca255338adfa4d48449f69242Eb8F"
	rpcURL         = "http://localhost:8545"
	chainID        = 31337
)

var anvilKeys = []string{
	"ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
	"59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
	"5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a",
	"7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6",
	"47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
	"8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba",
	"92db14e403b83dfe3df233f83dfa3a0d7096f21ca9b0d6d6b8d88b2b4ec1564e",
	"4bbbf85ce3377467afe5d46f804f221813b2bb87f24d81f60f1fcdbf7cbf4356",
	"dbda1821b80551c9d65939329250298aa3472ba22feea921c0cf5d620ea67b97",
	"2a871d0798f97d79848a013d4936a73bf4cc922c825d33c1cf7073dff6d409c6",
}

type reputationPeer struct {
	host     host.Host
	pubsub   *pubsub.PubSub
	crdt     *marketplace.ListingCRDT
	stream   *p2p.StreamHandler
	dag      ipld.DAGService
	wallet   *eth.Wallet
	client   *eip8004.Client
	agentID  *big.Int
	manager  *agent.Manager
	address  common.Address
	tokenURI string
}

func (p *reputationPeer) Close() {
	if p.crdt != nil {
		p.crdt.Close()
	}
	if p.host != nil {
		p.host.Close()
	}
}

var (
	erc8004ContractsPath string
	rng                  = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func init() {
	cwd, _ := os.Getwd()
	erc8004ContractsPath = filepath.Join(cwd, "..", "..", "dev", "erc-8004-contracts")
}

func startAnvil(t *testing.T) *exec.Cmd {
	t.Helper()

	cmd := exec.Command("anvil")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = erc8004ContractsPath

	err := cmd.Start()
	require.NoError(t, err, "failed to start anvil")

	time.Sleep(2 * time.Second)

	return cmd
}

func deployContracts(t *testing.T) {
	t.Helper()

	forge := filepath.Join(os.Getenv("HOME"), ".foundry", "bin", "forge")
	if _, err := os.Stat(forge); err != nil {
		forge = "forge"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	privateKey := "0x" + anvilKeys[0]
	cmd := exec.CommandContext(ctx, forge, "script", "script/Deploy.s.sol:Deploy",
		"--rpc-url", rpcURL,
		"--broadcast",
		"--private-key", privateKey)
	cmd.Dir = erc8004ContractsPath
	cmd.Env = append(os.Environ(), "PRIVATE_KEY="+privateKey)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Deploy output: %s", string(out))
		require.NoError(t, err, "failed to deploy contracts")
	}
}

func mintUSDC(t *testing.T, to string, amount int64) {
	t.Helper()
}

func setupReputationPeers(t *testing.T, numPeers int) (mocknet.Mocknet, []*reputationPeer) {
	t.Helper()

	mn := mocknet.New()
	peers := make([]*reputationPeer, numPeers)

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

		peers[i] = &reputationPeer{
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

func (p *reputationPeer) setupCRDT(t *testing.T) {
	t.Helper()

	crdt, err := marketplace.NewListingCRDTWithPubSub(context.Background(), p.pubsub, p.dag)
	if err != nil {
		t.Fatalf("failed to create CRDT: %v", err)
	}
	p.crdt = crdt
}

func createWalletForPeer(t *testing.T, peerIndex int) (*eth.Wallet, common.Address) {
	t.Helper()

	wallet, err := eth.NewWallet(anvilKeys[peerIndex], rpcURL)
	require.NoError(t, err, "failed to create wallet for peer %d", peerIndex)

	return wallet, common.HexToAddress(wallet.AddressHex())
}

func registerIdentityForPeer(t *testing.T, wallet *eth.Wallet, peerIndex int) (*big.Int, string) {
	t.Helper()

	privKey := anvilKeys[peerIndex]
	client, err := eip8004.NewClient(
		context.Background(),
		rpcURL,
		privKey,
		common.HexToAddress(identityAddr),
		common.HexToAddress(reputationAddr),
		common.HexToAddress(""),
		int64(chainID),
	)
	require.NoError(t, err, "failed to create eip8004 client for peer %d", peerIndex)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	agentURI := fmt.Sprintf("ipfs://agent-%d", peerIndex)
	agentID, err := client.RegisterIdentity(ctx, agentURI)
	require.NoError(t, err, "failed to register identity for peer %d", peerIndex)

	return agentID, agentURI
}

func setupAgentForPeer(t *testing.T, ctx context.Context, p *reputationPeer, peerIndex int) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("betar-ipfs-peer-%d-*", peerIndex))
	require.NoError(t, err)

	ipfsClient, err := ipfs.NewClient(ctx, p.host, nil, tmpDir)
	require.NoError(t, err, "failed to create ipfs client for peer %d", peerIndex)

	responses := map[string]string{
		"hello":   fmt.Sprintf("Hello from agent %d!", peerIndex),
		"status":  "Agent is running",
		"compute": "Task computed successfully",
	}
	mockRuntime := agent.NewMockRuntime(responses)

	p.manager = agent.NewManagerWithRuntime(
		mockRuntime,
		ipfsClient,
		p2p.NewHostFromRaw(p.host),
		nil, nil, "", nil,
	)

	spec := agent.AgentSpec{
		Name:        fmt.Sprintf("Agent-%d", peerIndex),
		Description: fmt.Sprintf("Test agent %d", peerIndex),
		Price:       0,
	}
	localAgent, err := p.manager.RegisterAgent(ctx, spec)
	require.NoError(t, err, "failed to register agent for peer %d", peerIndex)

	agentID := p.host.ID().String() + "/" + localAgent.AgentID
	p.crdt.Apply(&types.AgentListingMessage{
		Type:      "list",
		AgentID:   agentID,
		Name:      fmt.Sprintf("Agent-%d", peerIndex),
		Price:     0,
		Metadata:  p.tokenURI,
		SellerID:  p.host.ID().String(),
		Addrs:     []string{p.host.Addrs()[0].String()},
		Timestamp: time.Now().Unix(),
		TokenID:   p.agentID.String(),
	})

	p.stream.RegisterHandler("execute", func(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
		result, err := p.manager.ExecuteTask(ctx, localAgent.AgentID, string(data))
		return []byte(result), err
	})
}

func executeTaskAndGiveFeedback(
	t *testing.T,
	ctx context.Context,
	buyer *reputationPeer,
	buyerIndex int,
	seller *reputationPeer,
	sellerIndex int,
	round int,
) {
	t.Helper()

	resp, err := buyer.stream.SendMessage(ctx, seller.host.ID(), "execute", []byte("hello"))
	require.NoError(t, err, "failed to execute task in round %d", round)
	assert.NotEmpty(t, resp, "response should not be empty")

	feedbackValue := int64(rng.Intn(100) + 1)
	err = buyer.client.GiveFeedback(ctx, seller.agentID,
		feedbackValue, 0,
		"quality", "",
		fmt.Sprintf("/task/%d", round),
		"",
		[32]byte{})
	require.NoError(t, err, "failed to give feedback in round %d", round)

	t.Logf("Round %d: Buyer %d -> Seller %d, feedback value: %d, response: %s",
		round, buyerIndex, sellerIndex, feedbackValue, string(resp))
}

func TestE2E_ReputationSystem(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Log("Starting Anvil...")
	anvil := startAnvil(t)
	defer func() {
		t.Log("Stopping Anvil...")
		anvil.Process.Kill()
		anvil.Wait()
	}()

	t.Log("Deploying contracts...")
	deployContracts(t)

	time.Sleep(2 * time.Second)

	t.Log("Minting USDC to all peers...")
	for i := 0; i < 10; i++ {
		wallet, _ := createWalletForPeer(t, i)
		mintUSDC(t, wallet.AddressHex(), 1000000000000)
	}

	t.Log("Setting up 10-peer mock P2P network...")
	mn, peers := setupReputationPeers(t, 10)
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

	t.Log("Registering identities and setting up agents...")
	for i, p := range peers {
		wallet, addr := createWalletForPeer(t, i)
		p.wallet = wallet
		p.address = addr

		client, err := eip8004.NewClient(ctx, rpcURL, anvilKeys[i],
			common.HexToAddress(identityAddr),
			common.HexToAddress(reputationAddr),
			common.HexToAddress(""),
			int64(chainID))
		require.NoError(t, err, "failed to create eip8004 client for peer %d", i)
		p.client = client

		agentID, agentURI := registerIdentityForPeer(t, wallet, i)
		p.agentID = agentID
		p.tokenURI = agentURI
		t.Logf("Peer %d: registered identity with agentID %s, tokenURI %s", i, agentID.String(), agentURI)

		setupAgentForPeer(t, ctx, p, i)
		t.Logf("Peer %d: agent registered and listed on CRDT", i)
	}

	t.Log("Waiting for CRDT convergence...")
	time.Sleep(2 * time.Second)

	for _, p := range peers {
		listings := p.crdt.List()
		assert.Equal(t, 10, len(listings), "each peer should see 10 listings")
	}

	t.Log("Verifying Identity Registry and CRDT data...")
	for i, p := range peers {
		listings := p.crdt.List()
		var myListing *types.AgentListing
		for _, l := range listings {
			if l.SellerID == p.host.ID().String() {
				myListing = l
				break
			}
		}
		require.NotNil(t, myListing, "peer %d should have its own listing in CRDT", i)

		assert.Equal(t, fmt.Sprintf("Agent-%d", i), myListing.Name, "peer %d: CRDT name should match", i)
		assert.Equal(t, p.host.ID().String(), myListing.SellerID, "peer %d: CRDT sellerID should match peer ID", i)
		assert.Equal(t, p.agentID.String(), myListing.TokenID, "peer %d: CRDT tokenID should match on-chain agentID", i)
		assert.Equal(t, p.tokenURI, myListing.Metadata, "peer %d: CRDT metadata should match tokenURI", i)

		onChainOwner, err := p.client.GetAgentOwner(ctx, p.agentID)
		require.NoError(t, err, "peer %d: failed to get on-chain owner", i)
		assert.Equal(t, p.address, onChainOwner, "peer %d: on-chain owner should match wallet address", i)

		onChainURI, err := p.client.GetAgentTokenURI(ctx, p.agentID)
		require.NoError(t, err, "peer %d: failed to get on-chain tokenURI", i)
		assert.Equal(t, p.tokenURI, onChainURI, "peer %d: on-chain tokenURI should match registered URI", i)

		t.Logf("Peer %d: CRDT listing verified - name=%s, sellerID=%s, tokenID=%s, metadata=%s",
			i, myListing.Name, myListing.SellerID, myListing.TokenID, myListing.Metadata)
		t.Logf("Peer %d: On-chain verified - owner=%s, tokenURI=%s",
			i, onChainOwner.Hex(), onChainURI)
	}

	t.Log("Executing tasks and giving feedback...")
	feedbackCounts := make(map[int]int)
	feedbackClients := make(map[int][]common.Address)
	rounds := 5
	for round := 0; round < rounds; round++ {
		for buyerIdx := 0; buyerIdx < 10; buyerIdx++ {
			sellerIdx := (buyerIdx + round + 1) % 10
			if buyerIdx == sellerIdx {
				continue
			}

			buyer := peers[buyerIdx]
			seller := peers[sellerIdx]

			executeTaskAndGiveFeedback(t, ctx, buyer, buyerIdx, seller, sellerIdx, round)
			feedbackCounts[sellerIdx]++
			feedbackClients[sellerIdx] = append(feedbackClients[sellerIdx], buyer.address)
		}
	}

	t.Log("Waiting for transactions to be mined...")
	time.Sleep(3 * time.Second)

	t.Log("Verifying reputation system...")
	for i, p := range peers {
		expectedFeedbacks := feedbackCounts[i]

		count, summaryValue, decimals, err := p.client.GetReputationSummaryForClients(ctx, p.agentID, feedbackClients[i], "", "")
		require.NoError(t, err, "failed to get reputation summary for peer %d", i)

		t.Logf("Peer %d: agentID=%s, feedback count=%d, summary value=%d, decimals=%d",
			i, p.agentID.String(), count, summaryValue, decimals)

		if expectedFeedbacks > 0 {
			assert.Greater(t, count, uint64(0), "peer %d should have feedback", i)
			assert.Greater(t, summaryValue, int64(0), "peer %d should have positive summary value", i)
		}
	}

	var totalFeedbacks uint64
	for i, p := range peers {
		count, _, _, err := p.client.GetReputationSummaryForClients(ctx, p.agentID, feedbackClients[i], "", "")
		require.NoError(t, err)
		totalFeedbacks += count
	}

	expectedTotal := 0
	for _, c := range feedbackCounts {
		expectedTotal += c
	}

	t.Logf("Total feedbacks across all agents: %d (expected: %d)", totalFeedbacks, expectedTotal)
	assert.Equal(t, uint64(expectedTotal), totalFeedbacks, "total feedbacks should match")

	t.Log("Verifying feedback by tag...")
	for i, p := range peers {
		if feedbackCounts[i] == 0 {
			continue
		}

		tagCount, tagValue, tagDecimals, err := p.client.GetReputationSummaryForClients(ctx, p.agentID, feedbackClients[i], "quality", "")
		require.NoError(t, err, "peer %d: failed to get quality tag summary", i)

		t.Logf("Peer %d: quality tag feedback count=%d, value=%d, decimals=%d", i, tagCount, tagValue, tagDecimals)

		totalCount, _, _, err := p.client.GetReputationSummaryForClients(ctx, p.agentID, feedbackClients[i], "", "")
		require.NoError(t, err)
		assert.Equal(t, tagCount, totalCount, "peer %d: quality tag count should equal total count", i)
	}

	t.Log("E2E Reputation System test PASSED!")
}
