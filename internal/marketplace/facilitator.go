package marketplace

import "context"

// Facilitator abstracts payment verification and settlement.
// Implementations handle specific payment schemes (EIP-712 ECDSA, SAFE 4337, etc.).
type Facilitator interface {
	// Scheme returns the payment scheme this facilitator handles (e.g. "exact", "safe-4337").
	Scheme() string

	// Verify checks that the payment payload is valid against the requirements.
	Verify(ctx context.Context, payload *FacilitatorPayload) (*FacilitatorVerifyResult, error)

	// Settle executes the payment settlement and returns a transaction hash.
	Settle(ctx context.Context, payload *FacilitatorPayload) (*FacilitatorSettleResult, error)
}

// FacilitatorPayload is the input to Verify/Settle.
type FacilitatorPayload struct {
	PaymentPayload      interface{}         // scheme-specific payload (EVMPayload or map for UserOp)
	PaymentRequirements PaymentRequirements // x402 payment requirements
	Payer               string              // payer's Ethereum address
}

// FacilitatorVerifyResult is the output of Verify.
type FacilitatorVerifyResult struct {
	IsValid       bool
	InvalidReason string
	Payer         string
}

// FacilitatorSettleResult is the output of Settle.
type FacilitatorSettleResult struct {
	Success     bool
	TxHash      string
	ErrorReason string
}

// FacilitatorRegistry holds facilitators keyed by scheme name.
type FacilitatorRegistry struct {
	facilitators map[string]Facilitator
}

// NewFacilitatorRegistry creates a new empty facilitator registry.
func NewFacilitatorRegistry() *FacilitatorRegistry {
	return &FacilitatorRegistry{facilitators: make(map[string]Facilitator)}
}

// Register adds a facilitator to the registry, keyed by its scheme.
func (r *FacilitatorRegistry) Register(f Facilitator) {
	r.facilitators[f.Scheme()] = f
}

// Get returns the facilitator for the given scheme, if registered.
func (r *FacilitatorRegistry) Get(scheme string) (Facilitator, bool) {
	f, ok := r.facilitators[scheme]
	return f, ok
}

// Schemes returns a list of all registered scheme names.
func (r *FacilitatorRegistry) Schemes() []string {
	schemes := make([]string, 0, len(r.facilitators))
	for s := range r.facilitators {
		schemes = append(schemes, s)
	}
	return schemes
}
