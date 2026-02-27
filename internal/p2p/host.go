package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/asabya/betar/internal/config"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

// Host wraps the libp2p host
type Host struct {
	host host.Host
	cfg  *config.P2PConfig
}

// NewHost creates a new libp2p host
func NewHost(ctx context.Context, cfg *config.P2PConfig) (*Host, error) {
	// Parse bootstrap peers
	var bootstrapPeers []peer.AddrInfo

	if len(cfg.BootstrapPeers) > 0 {
		for _, addrStr := range cfg.BootstrapPeers {
			addr, err := multiaddr.NewMultiaddr(addrStr)
			if err != nil {
				continue
			}
			pi, err := peer.AddrInfoFromP2pAddr(addr)
			if err != nil {
				continue
			}
			bootstrapPeers = append(bootstrapPeers, *pi)
		}
	} else {
		bootstrapAddrs := dht.DefaultBootstrapPeers
		for _, addr := range bootstrapAddrs {
			pi, _ := peer.AddrInfoFromP2pAddr(addr)
			bootstrapPeers = append(bootstrapPeers, *pi)
		}
	}

	// Build libp2p options
	opts := []libp2p.Option{
		// Listen addresses
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", cfg.Port),
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", cfg.Port),
		),

		// Enable PING protocol
		libp2p.Ping(true),
	}

	if cfg.EnableRelay {
		opts = append(opts, libp2p.EnableRelay())
	}

	// Add private key if provided
	if cfg.PrivKey != nil {
		opts = append(opts, libp2p.Identity(cfg.PrivKey))
	}

	// Create the host
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Add bootstrap peers to peerstore
	for _, pi := range bootstrapPeers {
		h.Peerstore().AddAddrs(pi.ID, pi.Addrs, peerstore.PermanentAddrTTL)
	}

	// Connect to bootstrap peers in background
	if len(bootstrapPeers) > 0 {
		go func() {
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			for _, pi := range bootstrapPeers {
				if err := h.Connect(ctx, pi); err != nil {
					continue
				}
			}
		}()
	}

	return &Host{host: h, cfg: cfg}, nil
}

// ID returns the peer ID
func (h *Host) ID() peer.ID {
	return h.host.ID()
}

// Addrs returns the listen addresses
func (h *Host) Addrs() []multiaddr.Multiaddr {
	return h.host.Addrs()
}

// AddrStrings returns the listen addresses as strings
func (h *Host) AddrStrings() []string {
	var addrs []string
	for _, addr := range h.host.Addrs() {
		addrs = append(addrs, addr.String())
	}
	return addrs
}

// Connect connects to a peer
func (h *Host) Connect(ctx context.Context, pi peer.AddrInfo) error {
	return h.host.Connect(ctx, pi)
}

// Disconnect disconnects from a peer
func (h *Host) Disconnect(p peer.ID) error {
	return h.host.Network().ClosePeer(p)
}

// Peerstore returns the peer store
func (h *Host) Peerstore() peerstore.Peerstore {
	return h.host.Peerstore()
}

// NewStream creates a new stream to a peer
func (h *Host) NewStream(ctx context.Context, p peer.ID, protocols ...string) error {
	ids := make([]protocol.ID, 0, len(protocols))
	for _, proto := range protocols {
		ids = append(ids, protocol.ID(proto))
	}

	_, err := h.host.NewStream(ctx, p, ids...)
	return err
}

// Close closes the host
func (h *Host) Close() error {
	return h.host.Close()
}

// RawHost returns the underlying libp2p host
func (h *Host) RawHost() host.Host {
	return h.host
}
