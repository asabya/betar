package marketplace

// ExecuteRequest is sent client → server to request resource execution.
type ExecuteRequest struct {
	Version       string `json:"version"`
	CorrelationID string `json:"correlation_id"`
	Resource      string `json:"resource"`
	Method        string `json:"method"`
	Body          []byte `json:"body"`
	CallerDID     string `json:"caller_did,omitempty"`
}
type ExecuteResponse struct {
	Version       string `json:"version"`
	CorrelationID string `json:"correlation_id"`
	PaymentID     string `json:"payment_id"`
	TxHash        string `json:"tx_hash"`
	Body          []byte `json:"body"`
	SellerDID     string `json:"seller_did,omitempty"`
	SellerTokenID string `json:"seller_token_id,omitempty"`
}

const (
	MsgTypeExecRequest         = "exec.request"
	MsgTypeExecPaymentRequired = "exec.payment_required"
	MsgTypeExecPaidRequest     = "exec.paid_request"
	MsgTypeExecResponse        = "exec.response"
	MsgTypeExecError           = "exec.error"
)

const ExecLibP2PVersion = "1.0"
