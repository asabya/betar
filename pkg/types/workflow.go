package types

import "time"

// WorkflowStatus represents the overall status of a workflow.
type WorkflowStatus string

const (
	// WorkflowStatusPending — workflow created but not yet started. Maps to A2A TaskState "submitted".
	WorkflowStatusPending WorkflowStatus = "pending"
	// WorkflowStatusRunning — workflow is actively executing steps. Maps to A2A TaskState "working".
	WorkflowStatusRunning WorkflowStatus = "running"
	// WorkflowStatusCompleted — all steps finished successfully. Maps to A2A TaskState "completed".
	WorkflowStatusCompleted WorkflowStatus = "completed"
	// WorkflowStatusFailed — workflow stopped due to a step failure. Maps to A2A TaskState "failed".
	WorkflowStatusFailed WorkflowStatus = "failed"
	// WorkflowStatusCanceled — workflow was canceled by the caller. Maps to A2A TaskState "canceled".
	WorkflowStatusCanceled WorkflowStatus = "canceled"
)

// StepStatus represents the status of an individual workflow step.
type StepStatus string

const (
	// StepStatusPending — step not yet started. Maps to A2A TaskState "submitted".
	StepStatusPending StepStatus = "pending"
	// StepStatusRunning — step is currently executing. Maps to A2A TaskState "working".
	StepStatusRunning StepStatus = "running"
	// StepStatusCompleted — step finished successfully. Maps to A2A TaskState "completed".
	StepStatusCompleted StepStatus = "completed"
	// StepStatusFailed — step encountered an error. Maps to A2A TaskState "failed".
	StepStatusFailed StepStatus = "failed"
	// StepStatusSkipped — step was skipped (e.g. due to earlier failure).
	StepStatusSkipped StepStatus = "skipped"
)

// Workflow represents a multi-agent workflow execution.
type Workflow struct {
	ID          string         `json:"id"`
	Status      WorkflowStatus `json:"status"`
	Steps       []WorkflowStep `json:"steps"`
	Input       string         `json:"input"`
	Output      string         `json:"output,omitempty"`
	TotalCost   string         `json:"totalCost"` // USDC micro-units
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	CompletedAt *time.Time     `json:"completedAt,omitempty"`
}

// WorkflowStep represents a single step in a workflow execution.
type WorkflowStep struct {
	Index       int            `json:"index"`
	AgentID     string         `json:"agentId"`
	Status      StepStatus     `json:"status"`
	Input       string         `json:"input"`
	Output      string         `json:"output,omitempty"`
	Error       string         `json:"error,omitempty"`
	Payment     *PaymentRecord `json:"payment,omitempty"`
	StartedAt   *time.Time     `json:"startedAt,omitempty"`
	CompletedAt *time.Time     `json:"completedAt,omitempty"`
}

// WorkflowDefinition is the user-provided specification for creating a workflow.
type WorkflowDefinition struct {
	AgentIDs []string `json:"agentIds"`
	Input    string   `json:"input"`
}
