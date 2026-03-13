package eip8004

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"

	"github.com/asabya/betar/pkg/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Client handles EIP-8004 on-chain contract interactions using the official
// ERC-8004 contracts deployed on Base Sepolia.
type Client struct {
	ethClient  *ethclient.Client
	auth       *bind.TransactOpts
	identity   *IdentityRegistry   // nil if address not configured
	reputation *ReputationRegistry // nil if address not configured
	validation *ValidationRegistry // nil if address not configured
	txMu       sync.Mutex          // serializes write transactions to prevent nonce races
}

// NewClient dials the RPC endpoint, parses the private key, and instantiates
// bindings for whichever registry addresses are non-zero.
func NewClient(ctx context.Context, rpcURL, privKey string,
	identityAddr, reputationAddr, validationAddr common.Address,
	chainID int64) (*Client, error) {

	ec, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("eip8004: dial %s: %w", rpcURL, err)
	}

	key, err := crypto.HexToECDSA(privKey)
	if err != nil {
		return nil, fmt.Errorf("eip8004: parse private key: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(chainID))
	if err != nil {
		return nil, fmt.Errorf("eip8004: create transactor: %w", err)
	}

	c := &Client{
		ethClient: ec,
		auth:      auth,
	}

	zero := common.Address{}

	if identityAddr != zero {
		c.identity, err = NewIdentityRegistry(identityAddr, ec)
		if err != nil {
			return nil, fmt.Errorf("eip8004: bind IdentityRegistry: %w", err)
		}
	}

	if reputationAddr != zero {
		c.reputation, err = NewReputationRegistry(reputationAddr, ec)
		if err != nil {
			return nil, fmt.Errorf("eip8004: bind ReputationRegistry: %w", err)
		}
	}

	if validationAddr != zero {
		c.validation, err = NewValidationRegistry(validationAddr, ec)
		if err != nil {
			return nil, fmt.Errorf("eip8004: bind ValidationRegistry: %w", err)
		}
	}

	return c, nil
}

// ─── Identity ────────────────────────────────────────────────────────────────

// RegisterIdentity mints a new agent NFT and returns the on-chain agentId.
// Returns nil, nil gracefully when the identity binding is not configured.
func (c *Client) RegisterIdentity(ctx context.Context, agentURI string) (*big.Int, error) {
	if c == nil || c.identity == nil {
		return nil, nil
	}

	// Serialize write txs to prevent nonce-too-low errors from concurrent calls.
	c.txMu.Lock()
	defer c.txMu.Unlock()

	auth := *c.auth
	auth.Context = ctx

	// Register1 is the ABI overload that takes (string agentURI).
	tx, err := c.identity.Register1(&auth, agentURI)
	if err != nil {
		return nil, fmt.Errorf("eip8004: RegisterIdentity tx: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, c.ethClient, tx)
	if err != nil {
		return nil, fmt.Errorf("eip8004: RegisterIdentity wait: %w", err)
	}

	// Parse the Registered event from receipt logs using the embedded filterer.
	for _, log := range receipt.Logs {
		ev, parseErr := c.identity.ParseRegistered(*log)
		if parseErr != nil {
			continue
		}
		return ev.AgentId, nil
	}

	return nil, fmt.Errorf("eip8004: RegisterIdentity: Registered event not found in receipt")
}

// SetAgentURI updates the IPFS metadata URI for an agent.
func (c *Client) SetAgentURI(ctx context.Context, agentID *big.Int, newURI string) error {
	if c == nil || c.identity == nil {
		return nil
	}
	c.txMu.Lock()
	defer c.txMu.Unlock()
	auth := *c.auth
	auth.Context = ctx
	_, err := c.identity.SetAgentURI(&auth, agentID, newURI)
	if err != nil {
		return fmt.Errorf("eip8004: SetAgentURI: %w", err)
	}
	return nil
}

// LinkWallet links the agent's Ethereum payment wallet via an EIP-712 signed
// proof. deadline is a Unix timestamp; sig is the 65-byte ECDSA signature.
func (c *Client) LinkWallet(ctx context.Context, agentID *big.Int, wallet common.Address, deadline *big.Int, sig []byte) error {
	if c == nil || c.identity == nil {
		return nil
	}
	c.txMu.Lock()
	defer c.txMu.Unlock()
	auth := *c.auth
	auth.Context = ctx
	_, err := c.identity.SetAgentWallet(&auth, agentID, wallet, deadline, sig)
	if err != nil {
		return fmt.Errorf("eip8004: LinkWallet: %w", err)
	}
	return nil
}

// ─── Reputation ──────────────────────────────────────────────────────────────

// GiveFeedback submits a scored feedback record from the calling address.
// value is a fixed-point score (e.g. 100 with valueDecimals=0 means 100%).
// tag1 is a category tag (e.g. "execution").
func (c *Client) GiveFeedback(ctx context.Context, agentID *big.Int,
	value int64, valueDecimals uint8,
	tag1, tag2, endpoint, feedbackURI string,
	feedbackHash [32]byte) error {

	if c == nil || c.reputation == nil {
		return nil
	}
	c.txMu.Lock()
	defer c.txMu.Unlock()
	auth := *c.auth
	auth.Context = ctx
	_, err := c.reputation.GiveFeedback(&auth,
		agentID,
		big.NewInt(value),
		valueDecimals,
		tag1, tag2, endpoint, feedbackURI,
		feedbackHash,
	)
	if err != nil {
		return fmt.Errorf("eip8004: GiveFeedback: %w", err)
	}
	return nil
}

// GetReputationSummary retrieves aggregated feedback for an agent.
// Pass empty tag1/tag2 to aggregate across all tags.
func (c *Client) GetReputationSummary(ctx context.Context, agentID *big.Int,
	tag1, tag2 string) (count uint64, summaryValue int64, decimals uint8, err error) {

	if c == nil || c.reputation == nil {
		return 0, 0, 0, nil
	}
	opts := &bind.CallOpts{Context: ctx}
	out, callErr := c.reputation.GetSummary(opts, agentID, nil, tag1, tag2)
	if callErr != nil {
		return 0, 0, 0, fmt.Errorf("eip8004: GetReputationSummary: %w", callErr)
	}
	sv := int64(0)
	if out.SummaryValue != nil {
		sv = out.SummaryValue.Int64()
	}
	return out.Count, sv, out.SummaryValueDecimals, nil
}

// ─── Validation ──────────────────────────────────────────────────────────────

// RequestValidation asks a specific validator to validate an agent.
func (c *Client) RequestValidation(ctx context.Context,
	validatorAddr common.Address,
	agentID *big.Int,
	requestURI string,
	requestHash [32]byte) error {

	if c == nil || c.validation == nil {
		return nil
	}
	c.txMu.Lock()
	defer c.txMu.Unlock()
	auth := *c.auth
	auth.Context = ctx
	_, err := c.validation.ValidationRequest(&auth, validatorAddr, agentID, requestURI, requestHash)
	if err != nil {
		return fmt.Errorf("eip8004: RequestValidation: %w", err)
	}
	return nil
}

// GetAgentValidations returns all requestHashes recorded for a given agent.
func (c *Client) GetAgentValidations(ctx context.Context, agentID *big.Int) ([][32]byte, error) {
	if c == nil || c.validation == nil {
		return nil, nil
	}
	opts := &bind.CallOpts{Context: ctx}
	return c.validation.GetAgentValidations(opts, agentID)
}

// ─── Metadata helpers ────────────────────────────────────────────────────────

// RegistrationData represents on-chain registration data.
type RegistrationData struct {
	TokenID     *big.Int
	Name        string
	Description string
	MetadataURI string
	Services    []string
	X402Support bool
	Active      bool
}

// ReputationData represents reputation information.
type ReputationData struct {
	TotalTasks    *big.Int
	SuccessTasks  *big.Int
	AverageRating *big.Int
	RatingCount   *big.Int
	TotalEarnings *big.Int
}

// MetadataSchema defines the EIP-8004 metadata schema.
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

// Service represents a service offered by an agent.
type Service struct {
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	Version  string `json:"version,omitempty"`
}

// Attribute represents metadata attributes.
type Attribute struct {
	TraitType string `json:"trait_type"`
	Value     any    `json:"value"`
}

// ToJSON serializes metadata to JSON.
func (m *MetadataSchema) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON deserializes metadata from JSON.
func FromJSON(data []byte) (*MetadataSchema, error) {
	var m MetadataSchema
	err := json.Unmarshal(data, &m)
	return &m, err
}

// CreateMetadataSchema creates a metadata schema from a registration payload.
// If httpEndpoint is non-empty, it is populated as the endpoint for each service.
func CreateMetadataSchema(reg *types.AgentRegistration, httpEndpoint ...string) *MetadataSchema {
	endpoint := ""
	if len(httpEndpoint) > 0 {
		endpoint = httpEndpoint[0]
	}
	services := make([]Service, len(reg.Services))
	for i, s := range reg.Services {
		services[i] = Service{
			Name:     s.Name,
			Endpoint: endpoint,
			Version:  s.Version,
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
