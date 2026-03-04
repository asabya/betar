package types

import "time"

// Session records all exchanges between a caller and a specific agent.
type Session struct {
	ID        string     `json:"id"`
	AgentID   string     `json:"agentId"`
	CallerID  string     `json:"callerId"` // peer.ID string or "local"
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	Exchanges []Exchange `json:"exchanges"`
}

// Exchange is a single task request-response pair within a session.
type Exchange struct {
	RequestID string         `json:"requestId"`
	Input     string         `json:"input"`
	Output    string         `json:"output"`
	Error     string         `json:"error,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Payment   *PaymentRecord `json:"payment,omitempty"`
}

// PaymentRecord holds settlement info for a paid exchange.
type PaymentRecord struct {
	PaymentID string    `json:"paymentId"`
	TxHash    string    `json:"txHash"`
	Amount    string    `json:"amount"` // USDC micro-units (e.g. "1000000" = 1 USDC)
	Payer     string    `json:"payer"`  // buyer wallet address
	PaidAt    time.Time `json:"paidAt"`
}
