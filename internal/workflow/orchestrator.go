package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/asabya/betar/pkg/types"
	"github.com/google/uuid"
)

// TaskExecutor abstracts agent task execution so the orchestrator can be tested
// with mocks. agent.Manager satisfies this interface directly.
type TaskExecutor interface {
	ExecuteTask(ctx context.Context, agentID, input string) (string, error)
}

// Orchestrator manages workflow creation, execution, and lifecycle.
type Orchestrator struct {
	executor  TaskExecutor
	mu        sync.RWMutex
	workflows map[string]*types.Workflow
	cancels   map[string]context.CancelFunc
}

// NewOrchestrator creates an Orchestrator backed by the given TaskExecutor.
func NewOrchestrator(executor TaskExecutor) *Orchestrator {
	return &Orchestrator{
		executor:  executor,
		workflows: make(map[string]*types.Workflow),
		cancels:   make(map[string]context.CancelFunc),
	}
}

// CreateWorkflow validates a definition and creates a pending workflow.
func (o *Orchestrator) CreateWorkflow(ctx context.Context, def types.WorkflowDefinition) (*types.Workflow, error) {
	if len(def.AgentIDs) == 0 {
		return nil, fmt.Errorf("workflow definition must have at least one agent")
	}
	if def.Input == "" {
		return nil, fmt.Errorf("workflow definition must have non-empty input")
	}

	now := time.Now()
	wf := &types.Workflow{
		ID:        uuid.New().String(),
		Status:    types.WorkflowStatusPending,
		Input:     def.Input,
		CreatedAt: now,
		UpdatedAt: now,
		Steps:     make([]types.WorkflowStep, len(def.AgentIDs)),
	}

	for i, agentID := range def.AgentIDs {
		wf.Steps[i] = types.WorkflowStep{
			Index:   i,
			AgentID: agentID,
			Status:  types.StepStatusPending,
		}
	}

	o.mu.Lock()
	o.workflows[wf.ID] = wf
	o.mu.Unlock()

	return copyWorkflow(wf), nil
}

// RunWorkflow executes all steps sequentially, piping each step's output as
// input to the next. It respects context cancellation and the CancelWorkflow
// flag between steps.
func (o *Orchestrator) RunWorkflow(ctx context.Context, workflowID string) (*types.Workflow, error) {
	o.mu.Lock()
	wf, ok := o.workflows[workflowID]
	if !ok {
		o.mu.Unlock()
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}
	if wf.Status != types.WorkflowStatusPending {
		o.mu.Unlock()
		return nil, fmt.Errorf("workflow %s is not pending (status: %s)", workflowID, wf.Status)
	}

	// Set up a cancellable context so CancelWorkflow can stop execution.
	ctx, cancel := context.WithCancel(ctx)
	o.cancels[workflowID] = cancel

	wf.Status = types.WorkflowStatusRunning
	wf.UpdatedAt = time.Now()
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		delete(o.cancels, workflowID)
		o.mu.Unlock()
		cancel()
	}()

	input := wf.Input

	for i := range wf.Steps {
		// Check for cancellation before starting each step.
		select {
		case <-ctx.Done():
			o.mu.Lock()
			o.skipRemaining(wf, i)
			wf.Status = types.WorkflowStatusCanceled
			now := time.Now()
			wf.CompletedAt = &now
			wf.UpdatedAt = now
			o.mu.Unlock()
			return copyWorkflow(wf), nil
		default:
		}

		now := time.Now()
		o.mu.Lock()
		wf.Steps[i].Status = types.StepStatusRunning
		wf.Steps[i].Input = input
		wf.Steps[i].StartedAt = &now
		wf.UpdatedAt = now
		o.mu.Unlock()

		output, err := o.executor.ExecuteTask(ctx, wf.Steps[i].AgentID, input)

		now = time.Now()
		o.mu.Lock()
		if err != nil {
			wf.Steps[i].Status = types.StepStatusFailed
			wf.Steps[i].Error = err.Error()
			wf.Steps[i].CompletedAt = &now
			o.skipRemaining(wf, i+1)
			wf.Status = types.WorkflowStatusFailed
			wf.CompletedAt = &now
			wf.UpdatedAt = now
			o.mu.Unlock()
			return copyWorkflow(wf), nil
		}

		wf.Steps[i].Status = types.StepStatusCompleted
		wf.Steps[i].Output = output
		wf.Steps[i].CompletedAt = &now
		wf.UpdatedAt = now
		o.mu.Unlock()

		// Pipe output to the next step.
		input = output
	}

	o.mu.Lock()
	now := time.Now()
	wf.Output = input
	wf.Status = types.WorkflowStatusCompleted
	wf.CompletedAt = &now
	wf.UpdatedAt = now
	o.mu.Unlock()

	return copyWorkflow(wf), nil
}

// RunWorkflowAsync starts RunWorkflow in a background goroutine.
func (o *Orchestrator) RunWorkflowAsync(ctx context.Context, workflowID string) error {
	o.mu.RLock()
	wf, ok := o.workflows[workflowID]
	if !ok {
		o.mu.RUnlock()
		return fmt.Errorf("workflow %s not found", workflowID)
	}
	if wf.Status != types.WorkflowStatusPending {
		o.mu.RUnlock()
		return fmt.Errorf("workflow %s is not pending (status: %s)", workflowID, wf.Status)
	}
	o.mu.RUnlock()

	go func() {
		_, _ = o.RunWorkflow(ctx, workflowID)
	}()
	return nil
}

// GetWorkflow returns a snapshot copy of the workflow.
func (o *Orchestrator) GetWorkflow(workflowID string) (*types.Workflow, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	wf, ok := o.workflows[workflowID]
	if !ok {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}
	return copyWorkflow(wf), nil
}

// ListWorkflows returns snapshot copies of all workflows.
func (o *Orchestrator) ListWorkflows() []*types.Workflow {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make([]*types.Workflow, 0, len(o.workflows))
	for _, wf := range o.workflows {
		result = append(result, copyWorkflow(wf))
	}
	return result
}

// CancelWorkflow signals the running workflow to stop between steps.
func (o *Orchestrator) CancelWorkflow(workflowID string) error {
	o.mu.RLock()
	defer o.mu.RUnlock()

	wf, ok := o.workflows[workflowID]
	if !ok {
		return fmt.Errorf("workflow %s not found", workflowID)
	}
	if wf.Status != types.WorkflowStatusRunning {
		return fmt.Errorf("workflow %s is not running (status: %s)", workflowID, wf.Status)
	}

	cancel, ok := o.cancels[workflowID]
	if ok {
		cancel()
	}
	return nil
}

// skipRemaining marks all steps from index start onwards as skipped.
func (o *Orchestrator) skipRemaining(wf *types.Workflow, start int) {
	for j := start; j < len(wf.Steps); j++ {
		wf.Steps[j].Status = types.StepStatusSkipped
	}
}

// copyWorkflow returns a deep-enough copy of a workflow for safe reading.
func copyWorkflow(wf *types.Workflow) *types.Workflow {
	cp := *wf
	cp.Steps = make([]types.WorkflowStep, len(wf.Steps))
	copy(cp.Steps, wf.Steps)
	return &cp
}
