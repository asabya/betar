package p2p

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/asabya/betar/internal/config"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

func testP2PConfig() *config.P2PConfig {
	return &config.P2PConfig{
		Port:       0,
		EnableMDNS: false,
		EnableDHT:  false,
	}
}

func TestMDNSPeerDiscovery(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mdnsCfg := &config.P2PConfig{Port: 0, EnableMDNS: true, EnableDHT: false}

	h1, err := NewHost(ctx, mdnsCfg)
	if err != nil {
		t.Fatalf("NewHost h1 failed: %v", err)
	}
	defer h1.Close()

	h2, err := NewHost(ctx, mdnsCfg)
	if err != nil {
		t.Fatalf("NewHost h2 failed: %v", err)
	}
	defer h2.Close()

	d1, err := NewDiscovery(ctx, h1.RawHost(), mdnsCfg)
	if err != nil {
		t.Fatalf("NewDiscovery h1 failed: %v", err)
	}
	defer d1.Close()

	d2, err := NewDiscovery(ctx, h2.RawHost(), mdnsCfg)
	if err != nil {
		t.Fatalf("NewDiscovery h2 failed: %v", err)
	}
	defer d2.Close()

	if err := d1.DiscoverPeers(ctx, nil); err != nil {
		t.Fatalf("DiscoverPeers h1 failed: %v", err)
	}
	if err := d2.DiscoverPeers(ctx, nil); err != nil {
		t.Fatalf("DiscoverPeers h2 failed: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if h1.RawHost().Network().Connectedness(h2.ID()) == network.Connected {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("mDNS: h1 did not discover h2 within 5s")
}

func TestHostConnect(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h1, err := NewHost(ctx, testP2PConfig())
	if err != nil {
		t.Fatalf("NewHost h1 failed: %v", err)
	}
	defer h1.Close()

	h2, err := NewHost(ctx, testP2PConfig())
	if err != nil {
		t.Fatalf("NewHost h2 failed: %v", err)
	}
	defer h2.Close()

	pi := peer.AddrInfo{ID: h2.ID(), Addrs: h2.Addrs()}
	if err := h1.Connect(ctx, pi); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	if got := h1.RawHost().Network().Connectedness(h2.ID()); got != network.Connected {
		t.Fatalf("expected connectedness %q, got %q", network.Connected, got)
	}
}

func TestStreamHandlerSendMessage(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h1, err := NewHost(ctx, testP2PConfig())
	if err != nil {
		t.Fatalf("NewHost h1 failed: %v", err)
	}
	defer h1.Close()

	h2, err := NewHost(ctx, testP2PConfig())
	if err != nil {
		t.Fatalf("NewHost h2 failed: %v", err)
	}
	defer h2.Close()

	if err := h1.Connect(ctx, peer.AddrInfo{ID: h2.ID(), Addrs: h2.Addrs()}); err != nil {
		t.Fatalf("connect failed: %v", err)
	}

	s1 := NewStreamHandler(h1.RawHost())
	s2 := NewStreamHandler(h2.RawHost())

	s2.RegisterHandler("echo", func(ctx context.Context, from peer.ID, data []byte) ([]byte, error) {
		return []byte("ok:" + string(data)), nil
	})

	resp, err := s1.SendMessage(ctx, h2.ID(), "echo", []byte("ping"))
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if string(resp) != "ok:ping" {
		t.Fatalf("unexpected response: %q", string(resp))
	}
}

func TestPubSubJoinTopicIdempotent(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h, err := NewHost(ctx, testP2PConfig())
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	ps, err := NewPubSub(ctx, h.RawHost())
	if err != nil {
		t.Fatalf("NewPubSub failed: %v", err)
	}
	defer ps.Close()

	topicA, err := ps.JoinTopic(ctx, "betar/test")
	if err != nil {
		t.Fatalf("first JoinTopic failed: %v", err)
	}
	topicB, err := ps.JoinTopic(ctx, "betar/test")
	if err != nil {
		t.Fatalf("second JoinTopic failed: %v", err)
	}

	if topicA != topicB {
		t.Fatalf("expected cached topic instance")
	}
}

func TestDiscoveryInvalidBootstrap(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h, err := NewHost(ctx, testP2PConfig())
	if err != nil {
		t.Fatalf("NewHost failed: %v", err)
	}
	defer h.Close()

	d, err := NewDiscovery(ctx, h.RawHost(), testP2PConfig())
	if err != nil {
		t.Fatalf("NewDiscovery failed: %v", err)
	}
	defer d.Close()

	err = d.DiscoverPeers(ctx, []string{"not-a-multiaddr"})
	if err == nil {
		t.Fatalf("expected bootstrap discovery error")
	}
	if !strings.Contains(err.Error(), "invalid bootstrap") {
		t.Fatalf("unexpected error: %v", err)
	}
}
