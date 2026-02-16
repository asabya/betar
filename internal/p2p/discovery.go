package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/asabya/betar/internal/config"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	corerouting "github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
)

// Discovery handles peer discovery
type Discovery struct {
	host      host.Host
	dhtClient *dht.IpfsDHT
	mdns      mdns.Service
	rw        *routing.RoutingDiscovery
	stopOnce  sync.Once
	stopFn    context.CancelFunc
}

const (
	defaultRendezvous = "betar/agent-marketplace"
	defaultMDNSName   = "betar-mdns"
)

// NewDiscovery creates a new discovery service
func NewDiscovery(ctx context.Context, h host.Host, cfg *config.P2PConfig) (*Discovery, error) {
	d := &Discovery{host: h}

	// Set up DHT for network-wide discovery
	if cfg.EnableDHT {
		var err error
		d.dhtClient, err = dht.New(ctx, h, dht.Mode(dht.ModeAuto))
		if err != nil {
			return nil, fmt.Errorf("failed to create DHT: %w", err)
		}
		if err := d.dhtClient.Bootstrap(ctx); err != nil {
			return nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
		}
		d.rw = routing.NewRoutingDiscovery(d.dhtClient)
	}

	// Set up mDNS for local discovery
	if cfg.EnableMDNS {
		d.mdns = mdns.NewMdnsService(h, defaultMDNSName, &mdnsPeerHandler{h: h})
		if err := d.mdns.Start(); err != nil {
			return nil, fmt.Errorf("failed to start mDNS: %w", err)
		}
	}

	return d, nil
}

// DiscoverPeers discovers peers on the network
func (d *Discovery) DiscoverPeers(ctx context.Context, bootstrapPeers []string) error {
	var firstErr error

	// Connect to bootstrap peers
	for _, addrStr := range bootstrapPeers {
		addr, err := parseMultiaddr(addrStr)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("invalid bootstrap multiaddr %q: %w", addrStr, err)
			}
			continue
		}
		pi, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("invalid bootstrap peer addr %q: %w", addrStr, err)
			}
			continue
		}
		if err := d.host.Connect(ctx, *pi); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("failed connecting bootstrap peer %q: %w", addrStr, err)
			}
			continue
		}
	}

	// Advertise our presence on DHT
	if d.dhtClient != nil {
		advCtx, cancel := context.WithCancel(ctx)
		d.stopFn = cancel

		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()

			for {
				util.Advertise(advCtx, d.rw, defaultRendezvous)
				select {
				case <-advCtx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
	}

	return firstErr
}

// FindPeers finds peers providing a specific service
func (d *Discovery) FindPeers(ctx context.Context, service string) []peer.AddrInfo {
	if d.dhtClient == nil {
		return nil
	}

	rendezvous := service
	if rendezvous == "" {
		rendezvous = defaultRendezvous
	}

	ch, err := d.rw.FindPeers(ctx, rendezvous)
	if err != nil {
		return nil
	}

	var peers []peer.AddrInfo
	for pi := range ch {
		if pi.ID == "" || pi.ID == d.host.ID() {
			continue
		}
		peers = append(peers, pi)
	}
	return peers
}

// Routing returns the routing implementation used by discovery.
func (d *Discovery) Routing() corerouting.Routing {
	if d == nil {
		return nil
	}
	return d.dhtClient
}

// Close closes the discovery service
func (d *Discovery) Close() error {
	d.stopOnce.Do(func() {
		if d.stopFn != nil {
			d.stopFn()
		}
	})

	if d.mdns != nil {
		d.mdns.Close()
	}
	if d.dhtClient != nil {
		return d.dhtClient.Close()
	}
	return nil
}

// mdnsPeerHandler handles mDNS peer discovery
type mdnsPeerHandler struct {
	h host.Host
}

func (m *mdnsPeerHandler) HandlePeerFound(p peer.AddrInfo) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := m.h.Connect(ctx, p); err != nil {
		return
	}
}

func parseMultiaddr(s string) (multiaddr.Multiaddr, error) {
	return multiaddr.NewMultiaddr(s)
}
