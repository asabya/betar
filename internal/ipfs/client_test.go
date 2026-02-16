package ipfs

import (
	"context"
	"testing"

	libp2p "github.com/libp2p/go-libp2p"
)

func TestClientAddGetAndPin(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	h, err := libp2p.New()
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	client, err := NewClient(ctx, h, nil, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	defer client.Close()

	cid, err := client.Add(ctx, []byte(`{"k":"v"}`))
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}

	data, err := client.Get(ctx, cid)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if string(data) != `{"k":"v"}` {
		t.Fatalf("unexpected get data: %q", string(data))
	}

	if err := client.Pin(ctx, cid); err != nil {
		t.Fatalf("Pin returned error: %v", err)
	}
}

func TestClientRejectsInvalidCID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	h, err := libp2p.New()
	if err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	defer h.Close()

	client, err := NewClient(ctx, h, nil, t.TempDir())
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	defer client.Close()

	if _, err := client.Get(ctx, "not-a-cid"); err == nil {
		t.Fatalf("Get should fail for invalid cid")
	}

	if err := client.Pin(ctx, "not-a-cid"); err == nil {
		t.Fatalf("Pin should fail for invalid cid")
	}
}
