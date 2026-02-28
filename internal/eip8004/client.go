package eip8004

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/asabya/betar/pkg/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// Client handles EIP-8004 contract interactions
type Client struct {
	registryAddr   common.Address
	reputationAddr common.Address
	validationAddr common.Address
}

// NewClient creates a new EIP-8004 client
func NewClient(registryAddr, reputationAddr, validationAddr common.Address) *Client {
	return &Client{
		registryAddr:   registryAddr,
		reputationAddr: reputationAddr,
		validationAddr: validationAddr,
	}
}

// RegistrationData represents on-chain registration data
type RegistrationData struct {
	TokenID     *big.Int
	Name        string
	Description string
	MetadataURI string
	Services    []string
	X402Support bool
	Active      bool
}

// RegisterAgent registers an agent on-chain
func (c *Client) RegisterAgent(ctx context.Context, auth *bind.TransactOpts, reg *types.AgentRegistration) (*types.AgentRegistration, error) {
	// In production, this would call the actual contract
	// For now, we return the registration data
	return reg, nil
}

// GetAgent retrieves agent data from the registry
func (c *Client) GetAgent(ctx context.Context, opts *bind.CallOpts, tokenID *big.Int) (*RegistrationData, error) {
	// In production, this would call the actual contract
	// Return mock data for now
	return &RegistrationData{
		TokenID:     tokenID,
		Name:        "Agent",
		Description: "AI Agent",
		MetadataURI: "",
		Services:    []string{},
		X402Support: true,
		Active:      true,
	}, nil
}

// UpdateAgent updates agent metadata on-chain
func (c *Client) UpdateAgent(ctx context.Context, auth *bind.TransactOpts, tokenID *big.Int, metadataURI string, active bool) error {
	// In production, this would call the actual contract
	return nil
}

// ReputationData represents reputation information
type ReputationData struct {
	TotalTasks    *big.Int
	SuccessTasks  *big.Int
	AverageRating *big.Int
	RatingCount   *big.Int
	TotalEarnings *big.Int
}

// GetReputation retrieves reputation data for an agent
func (c *Client) GetReputation(ctx context.Context, opts *bind.CallOpts, agentID *big.Int) (*ReputationData, error) {
	// In production, this would call the actual contract
	return &ReputationData{
		TotalTasks:    big.NewInt(0),
		SuccessTasks:  big.NewInt(0),
		AverageRating: big.NewInt(0),
		RatingCount:   big.NewInt(0),
		TotalEarnings: big.NewInt(0),
	}, nil
}

// RecordTaskCompletion records task completion on-chain
func (c *Client) RecordTaskCompletion(ctx context.Context, auth *bind.TransactOpts, agentID *big.Int, success bool, earnings *big.Int) error {
	// In production, this would call the actual contract
	return nil
}

// SubmitFeedback submits feedback for an agent
func (c *Client) SubmitFeedback(ctx context.Context, auth *bind.TransactOpts, agentID *big.Int, rating *big.Int, comment string) error {
	// In production, this would call the actual contract
	return nil
}

// MetadataSchema defines the EIP-8004 metadata schema
type MetadataSchema struct {
	Type        string      `json:"type"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Image       string      `json:"image,omitempty"`
	Services    []Service   `json:"services"`
	X402Support bool        `json:"x402Support"`
	Active      bool        `json:"active"`
	ExternalURL string      `json:"external_url,omitempty"`
	Attributes  []Attribute `json:"attributes,omitempty"`
}

// Service represents a service offered by an agent
type Service struct {
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	Version  string `json:"version,omitempty"`
}

// Attribute represents metadata attributes
type Attribute struct {
	TraitType string `json:"trait_type"`
	Value     any    `json:"value"`
}

// ToJSON serializes metadata to JSON
func (m *MetadataSchema) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON deserializes metadata from JSON
func FromJSON(data []byte) (*MetadataSchema, error) {
	var m MetadataSchema
	err := json.Unmarshal(data, &m)
	return &m, err
}

// CreateMetadataSchema creates a metadata schema from registration
func CreateMetadataSchema(reg *types.AgentRegistration) *MetadataSchema {
	services := make([]Service, len(reg.Services))
	for i, s := range reg.Services {
		services[i] = Service{
			Name:    s.Name,
			Version: s.Version,
		}
	}

	return &MetadataSchema{
		Type:        "Agent",
		Name:        reg.Name,
		Description: reg.Description,
		Image:       reg.Image,
		Services:    services,
		X402Support: reg.X402Support,
		Active:      reg.Active,
	}
}
