package types

import (
	"encoding/json"
	"time"

	"github.com/mark3labs/x402-go"
	"google.golang.org/genai"
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

// OnChainReputation holds EIP-8004 reputation summary for an agent.
type OnChainReputation struct {
	Count    uint64 `json:"count"`
	Score    int64  `json:"score"`
	Decimals uint8  `json:"decimals"`
}

// AgentListing represents an agent listed on the marketplace (off-chain)
type AgentListing struct {
	ID                string             `json:"id"`                          // Marketplace agent ID (peerID/agentID)
	Name              string             `json:"name"`                        // Agent name
	Price             float64            `json:"price"`                       // Price per task in ETH
	Metadata          string             `json:"metadata"`                    // IPFS CID
	SellerID          string             `json:"sellerId"`                    // Seller's peer ID
	Addrs             []string           `json:"addrs,omitempty"`             // Multiaddrs to dial seller peer
	Protocols         []string           `json:"protocols,omitempty"`         // Supported app protocols
	Timestamp         int64              `json:"timestamp"`                   // Unix timestamp
	TokenID           string             `json:"tokenId,omitempty"`           // EIP-8004 on-chain token ID
	OnChainReputation *OnChainReputation `json:"onChainReputation,omitempty"` // On-chain reputation (when available)
	AgentAPI          string             `json:"agentApi,omitempty"`          // Optional API endpoint for the agent
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
	TokenID   string   `json:"tokenId,omitempty"`
	AgentAPI  string   `json:"agentApi,omitempty"`
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

type X402PaymentPayload struct {
	x402.PaymentPayload
	Payload struct {
		UserOpHash string `json:"userOpHash"`
	} `json:"payload"`
	Accepted struct {
		Scheme            string `json:"scheme"`
		Network           string `json:"network"`
		Asset             string `json:"asset"`
		Amount            string `json:"amount"`
		PayTo             string `json:"payTo"`
		MaxTimeoutSeconds int    `json:"maxTimeoutSeconds"`
		Extra             struct {
			Model      string `json:"model"`
			Capability string `json:"capability"`
			Product    struct {
				Name string `json:"name"`
			} `json:"product"`
		} `json:"extra"`
	} `json:"accepted"`
	Extensions any `json:"extensions"`
}

type Params struct {
	Message struct {
		ContextID string `json:"contextId"`
		Kind      string `json:"kind"`
		MessageID string `json:"messageId"`
		Metadata  struct {
			X402PaymentPayload X402PaymentPayload `json:"x402.payment.payload"`
		} `json:"metadata"`
		Parts  []genai.Part `json:"parts"`
		Role   string       `json:"role"`
		TaskID string       `json:"taskId"`
	} `json:"message"`
}

// Raw request body to agent
//
//	{
//	  "id": "ac1dad7d-bf4c-43f0-8de6-6bc08824c859",
//	  "jsonrpc": "2.0",
//	  "method": "message/send",
//	  "params": {
//	    "message": {
//	      "contextId": "5c218235-d137-4817-9766-ba8321b49fa5",
//	      "kind": "message",
//	      "messageId": "0fa01610-4a85-4e0d-a006-b4aa96069af1",
//	      "metadata": {
//	        "x402.payment.payload": {
//	          "x402Version": 2,
//	          "payload": {
//	            "userOpHash": "0x111953B276CA476F5203CABED83DC4CF58055DEE4C10BEBF33748046B9AC9FF1"
//	          },
//	          "accepted": {
//	            "scheme": "exact",
//	            "network": "eip155:11155111",
//	            "asset": "0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238",
//	            "amount": "500000",
//	            "payTo": "0xb870F81337AFEC08308062FEC9667698f324eAf6",
//	            "maxTimeoutSeconds": 600,
//	            "extra": {
//	              "model": "gemini-2.5-flash-lite",
//	              "capability": "chat",
//	              "product": {
//	                "name": "gemini-2.5-flash-lite"
//	              }
//	            }
//	          },
//	          "resource": null,
//	          "extensions": null
//	        },
//	        "x402.payment.status": "payment-submitted"
//	      },
//	      "parts": [
//	        {
//	          "kind": "text",
//	          "text": "send_signed_payment_payload"
//	        }
//	      ],
//	      "role": "user",
//	      "taskId": "188ac63f-63ac-44e5-b6c2-8bfd0201154f"
//	    }
//	  }
//	}
type AgentRequest struct {
	ID       string `json:"id,omitempty"`
	Jsonrpc  string `json:"jsonrpc,omitempty"`
	Method   string `json:"method,omitempty"`
	Params   Params `json:"params"`
	Input    string `json:"input,omitempty"`
	Resource string `json:"resource,omitempty"`
}

type AgentResponse struct {
	ID      string `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		ContextID string `json:"contextId"`
		ID        string `json:"id"`
		Kind      string `json:"kind"`
		Metadata  struct {
		} `json:"metadata"`
		Status struct {
			Message struct {
				Kind      string `json:"kind"`
				MessageID string `json:"messageId"`
				Metadata  struct {
					X402PaymentStatus   string `json:"x402.payment.status"`
					X402PaymentRequired struct {
						X402Version int    `json:"x402Version"`
						Error       string `json:"error"`
						Resource    struct {
							URL         string `json:"url"`
							Description string `json:"description"`
							MimeType    string `json:"mimeType"`
						} `json:"resource"`
						Accepts []struct {
							Scheme            string `json:"scheme"`
							Network           string `json:"network"`
							Asset             string `json:"asset"`
							Amount            string `json:"amount"`
							PayTo             string `json:"payTo"`
							MaxTimeoutSeconds int    `json:"maxTimeoutSeconds"`
							Extra             struct {
								Model      string `json:"model"`
								Capability string `json:"capability"`
								Product    struct {
									Name string `json:"name"`
								} `json:"product"`
							} `json:"extra"`
						} `json:"accepts"`
						Extensions any `json:"extensions"`
					} `json:"x402.payment.required"`
				} `json:"metadata"`
				Parts []genai.Part `json:"parts"`
				Role  string       `json:"role"`
			} `json:"message"`
			State string `json:"state"`
		} `json:"status"`
	} `json:"result"`
}
