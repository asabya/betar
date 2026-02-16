package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// PubSub handles pubsub messaging
type PubSub struct {
	ps     *pubsub.PubSub
	host   host.Host
	topics map[string]*pubsub.Topic
	mu     sync.RWMutex
}

// Raw returns the underlying libp2p pubsub instance.
func (ps *PubSub) Raw() *pubsub.PubSub {
	if ps == nil {
		return nil
	}
	return ps.ps
}

// NewPubSub creates a new pubsub instance
func NewPubSub(ctx context.Context, h host.Host) (*PubSub, error) {
	// Create pubsub with GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, h,
		pubsub.WithMessageSigning(true),
		pubsub.WithStrictSignatureVerification(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	return &PubSub{
		ps:     ps,
		host:   h,
		topics: make(map[string]*pubsub.Topic),
	}, nil
}

// JoinTopic joins a topic
func (ps *PubSub) JoinTopic(ctx context.Context, topic string) (*pubsub.Topic, error) {
	_ = ctx
	ps.mu.RLock()
	if t, ok := ps.topics[topic]; ok {
		ps.mu.RUnlock()
		return t, nil
	}
	ps.mu.RUnlock()

	t, err := ps.ps.Join(topic)
	if err != nil {
		return nil, fmt.Errorf("failed to join topic %s: %w", topic, err)
	}

	ps.mu.Lock()
	ps.topics[topic] = t
	ps.mu.Unlock()
	return t, nil
}

// Subscribe subscribes to a topic
func (ps *PubSub) Subscribe(ctx context.Context, topic string) (*pubsub.Subscription, error) {
	t, err := ps.JoinTopic(ctx, topic)
	if err != nil {
		return nil, err
	}

	return t.Subscribe()
}

// Publish publishes a message to a topic
func (ps *PubSub) Publish(ctx context.Context, topic string, data []byte) error {
	t, err := ps.JoinTopic(ctx, topic)
	if err != nil {
		return err
	}

	return t.Publish(ctx, data)
}

// RegisterTopicValidator registers a validator for a topic
func (ps *PubSub) RegisterTopicValidator(topic string, val func(context.Context, peer.ID, *pubsub.Message) pubsub.ValidationResult) error {
	return ps.ps.RegisterTopicValidator(topic, val, pubsub.WithValidatorTimeout(100*time.Millisecond))
}

// GetTopics returns list of subscribed topics
func (ps *PubSub) GetTopics() []string {
	return ps.ps.GetTopics()
}

// ListPeers returns peers subscribed to a topic
func (ps *PubSub) ListPeers(topic string) []peer.ID {
	return ps.ps.ListPeers(topic)
}

// Close closes the pubsub
func (ps *PubSub) Close() error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	for _, t := range ps.topics {
		t.Close()
	}
	return nil
}
