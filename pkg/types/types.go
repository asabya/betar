package types

import (
	"encoding/json"
	"time"
)

// AgentRegistration represents an agent's on-chain registration (EIP-8004)
type AgentRegistration struct {
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Image       string    `json:"image,omitempty"`
	Services    []Service `json:"services"`
	X402Support bool      `json:"x402Support"`
	Active      bool      `json:"active"`
}

// Service represents a service offered by an agent
type Service struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// AgentListing represents an agent listed on the marketplace (off-chain)
type AgentListing struct {
	ID        string   `json:"id"`                  // Marketplace agent ID (peerID/agentID)
	Name      string   `json:"name"`                // Agent name
	Price     float64  `json:"price"`               // Price per task in ETH
	Metadata  string   `json:"metadata"`            // IPFS CID
	SellerID  string   `json:"sellerId"`            // Seller's peer ID
	Addrs     []string `json:"addrs,omitempty"`     // Multiaddrs to dial seller peer
	Protocols []string `json:"protocols,omitempty"` // Supported app protocols
	Timestamp int64    `json:"timestamp"`           // Unix timestamp
}

// Order represents a marketplace order
type Order struct {
	ID        string  `json:"orderId"`
	AgentID   string  `json:"agentId"`
	BuyerID   string  `json:"buyerId"`
	SellerID  string  `json:"sellerId,omitempty"`
	Price     float64 `json:"price"`
	Status    string  `json:"status"` // "pending", "accepted", "completed", "cancelled"
	Timestamp int64   `json:"timestamp"`
}

// AgentListingMessage represents a CRDT listing mutation payload.
type AgentListingMessage struct {
	Type      string   `json:"type"` // "list", "update", "delist"
	AgentID   string   `json:"agentId"`
	Name      string   `json:"name"`
	Price     float64  `json:"price"`
	Metadata  string   `json:"metadata"` // IPFS CID
	SellerID  string   `json:"sellerId"`
	Addrs     []string `json:"addrs,omitempty"`
	Protocols []string `json:"protocols,omitempty"`
	Timestamp int64    `json:"timestamp"`
}

// OrderMessage represents a pubsub message for order updates
type OrderMessage struct {
	Type      string  `json:"type"` // "new", "accept", "complete", "cancel"
	OrderID   string  `json:"orderId"`
	AgentID   string  `json:"agentId"`
	BuyerID   string  `json:"buyerId"`
	SellerID  string  `json:"sellerId,omitempty"`
	Price     float64 `json:"price"`
	Status    string  `json:"status"`
	Timestamp int64   `json:"timestamp"`
}

// TaskRequest represents a task execution request
type TaskRequest struct {
	AgentID   string `json:"agentId"`
	Input     string `json:"input"`
	Payment   string `json:"payment"` // Payment amount in wei
	RequestID string `json:"requestId"`
}

// TaskResponse represents a task execution response
type TaskResponse struct {
	RequestID string `json:"requestId"`
	Output    string `json:"output"`
	Error     string `json:"error,omitempty"`
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	RequestID string    `json:"requestId"`
	Output    string    `json:"output"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Reputation represents agent reputation data
type Reputation struct {
	AgentID       string  `json:"agentId"`
	TotalTasks    uint64  `json:"totalTasks"`
	SuccessRate   float64 `json:"successRate"`
	AverageRating float64 `json:"averageRating"`
	TotalEarnings string  `json:"totalEarnings"` // in wei
}

// ToJSON serializes a struct to JSON bytes
func ToJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// FromJSON deserializes JSON bytes to a struct
func FromJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
