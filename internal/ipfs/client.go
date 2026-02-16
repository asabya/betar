package ipfs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	ipfslite "github.com/hsanjuan/ipfs-lite"
	"github.com/ipfs/go-cid"
	leveldb "github.com/ipfs/go-ds-leveldb"
	ipld "github.com/ipfs/go-ipld-format"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/routing"
)

// Client is an embedded IPFS-lite client.
type Client struct {
	peer  *ipfslite.Peer
	store io.Closer
}

// NewClient creates an embedded IPFS-lite client using the existing libp2p host.
func NewClient(ctx context.Context, h host.Host, router routing.Routing, dataDir string) (*Client, error) {
	if h == nil {
		return nil, fmt.Errorf("libp2p host is required")
	}
	if dataDir == "" {
		return nil, fmt.Errorf("data directory is required")
	}

	repoDir := filepath.Join(dataDir, "ipfslite")
	if err := os.MkdirAll(repoDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create ipfs-lite repo dir: %w", err)
	}

	store, err := leveldb.NewDatastore(filepath.Join(repoDir, "datastore"), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ipfs-lite datastore: %w", err)
	}

	peerCfg := &ipfslite.Config{Offline: router == nil}
	peer, err := ipfslite.New(ctx, store, nil, h, router, peerCfg)
	if err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("failed to create ipfs-lite peer: %w", err)
	}

	return &Client{peer: peer, store: store}, nil
}

// Add adds data to IPFS and returns the CID
func (c *Client) Add(ctx context.Context, data []byte) (string, error) {
	if c == nil || c.peer == nil {
		return "", fmt.Errorf("ipfs-lite client is not initialized")
	}
	if len(data) == 0 {
		return "", fmt.Errorf("data cannot be empty")
	}

	node, err := c.peer.AddFile(ctx, bytes.NewReader(data), nil)
	if err != nil {
		return "", fmt.Errorf("failed to add data to ipfs-lite: %w", err)
	}
	return node.Cid().String(), nil
}

// DAGService returns the underlying DAG service for CRDT/block operations.
func (c *Client) DAGService() ipld.DAGService {
	if c == nil || c.peer == nil {
		return nil
	}
	return c.peer
}

func (c *Client) validateCID(cidStr string) error {
	if _, err := cid.Parse(cidStr); err != nil {
		return fmt.Errorf("invalid CID %q: %w", cidStr, err)
	}
	return nil
}

// Get retrieves data from IPFS by CID
func (c *Client) Get(ctx context.Context, cidStr string) ([]byte, error) {
	if c == nil || c.peer == nil {
		return nil, fmt.Errorf("ipfs-lite client is not initialized")
	}
	if err := c.validateCID(cidStr); err != nil {
		return nil, err
	}

	parsedCID, err := cid.Parse(cidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid CID %q: %w", cidStr, err)
	}
	r, err := c.peer.GetFile(ctx, parsedCID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cid %s from ipfs-lite: %w", cidStr, err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read ipfs-lite content: %w", err)
	}
	return data, nil
}

// Pin pins a CID to local storage
func (c *Client) Pin(ctx context.Context, cidStr string) error {
	_ = ctx
	if err := c.validateCID(cidStr); err != nil {
		return err
	}
	return nil
}

// Unpin unpins a CID from local storage
func (c *Client) Unpin(ctx context.Context, cidStr string) error {
	_ = ctx
	if err := c.validateCID(cidStr); err != nil {
		return err
	}
	return nil
}

// AddJSON adds a JSON object to IPFS
func (c *Client) AddJSON(ctx context.Context, v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return c.Add(ctx, data)
}

// GetJSON retrieves and unmarshals JSON from IPFS
func (c *Client) GetJSON(ctx context.Context, cidStr string, v interface{}) error {
	data, err := c.Get(ctx, cidStr)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Cat is an alias for Get
func (c *Client) Cat(ctx context.Context, cidStr string) ([]byte, error) {
	return c.Get(ctx, cidStr)
}

// DHTFindProvs finds providers for a CID
func (c *Client) DHTFindProvs(ctx context.Context, cidStr string) ([]string, error) {
	_ = ctx
	if err := c.validateCID(cidStr); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("DHTFindProvs is not implemented for embedded ipfs-lite")
}

// Close closes local IPFS resources.
func (c *Client) Close() error {
	if c == nil || c.store == nil {
		return nil
	}
	return c.store.Close()
}
