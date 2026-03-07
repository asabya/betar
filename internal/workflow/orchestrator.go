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
	executor TaskExecutor
	store    WorkflowStore
	mu       sync.RWMutex
	cancels  map[string]context.CancelFunc
}

// NewOrchestrator creates an Orchestrator backed by the given TaskExecutor and WorkflowStore.
func NewOrchestrator(executor TaskExecutor, store WorkflowStore) *Orchestrator {
	return &Orchestrator{
		executor: executor,
		store:    store,
		cancels:  make(map[string]context.CancelFunc),
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

	if err := o.store.Save(ctx, wf); err != nil {
		return nil, fmt.Errorf("save workflow: %w", err)
	}

	return copyWorkflow(wf), nil
}

// RunWorkflow executes all steps sequentially, piping each step's output as
// input to the next. It respects context cancellation and the CancelWorkflow
// flag between steps.
func (o *Orchestrator) RunWorkflow(ctx context.Context, workflowID string) (*types.Workflow, error) {
	wf, err := o.store.Get(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if wf.Status != types.WorkflowStatusPending {
		return nil, fmt.Errorf("workflow %s is not pending (status: %s)", workflowID, wf.Status)
	}

	// Set up a cancellable context so CancelWorkflow can stop execution.
	ctx, cancel := context.WithCancel(ctx)
	o.mu.Lock()
	o.cancels[workflowID] = cancel
	o.mu.Unlock()

	wf.Status = types.WorkflowStatusRunning
	wf.UpdatedAt = time.Now()
	_ = o.store.Save(ctx, wf)

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
			o.skipRemaining(wf, i)
			wf.Status = types.WorkflowStatusCanceled
			now := time.Now()
			wf.CompletedAt = &now
			wf.UpdatedAt = now
			_ = o.store.Save(context.Background(), wf)
			return copyWorkflow(wf), nil
		default:
		}

		now := time.Now()
		wf.Steps[i].Status = types.StepStatusRunning
		wf.Steps[i].Input = input
		wf.Steps[i].StartedAt = &now
		wf.UpdatedAt = now
		_ = o.store.Save(ctx, wf)

		output, err := o.executor.ExecuteTask(ctx, wf.Steps[i].AgentID, input)

		now = time.Now()
		if err != nil {
			wf.Steps[i].Status = types.StepStatusFailed
			wf.Steps[i].Error = err.Error()
			wf.Steps[i].CompletedAt = &now
			o.skipRemaining(wf, i+1)
			wf.Status = types.WorkflowStatusFailed
			wf.CompletedAt = &now
			wf.UpdatedAt = now
			_ = o.store.Save(context.Background(), wf)
			return copyWorkflow(wf), nil
		}

		wf.Steps[i].Status = types.StepStatusCompleted
		wf.Steps[i].Output = output
		wf.Steps[i].CompletedAt = &now
		wf.UpdatedAt = now
		_ = o.store.Save(ctx, wf)

		// Pipe output to the next step.
		input = output
	}

	now := time.Now()
	wf.Output = input
	wf.Status = types.WorkflowStatusCompleted
	wf.CompletedAt = &now
	wf.UpdatedAt = now
	_ = o.store.Save(ctx, wf)

	return copyWorkflow(wf), nil
}

// RunWorkflowAsync starts RunWorkflow in a background goroutine.
func (o *Orchestrator) RunWorkflowAsync(ctx context.Context, workflowID string) error {
	wf, err := o.store.Get(ctx, workflowID)
	if err != nil {
		return err
	}
	if wf.Status != types.WorkflowStatusPending {
		return fmt.Errorf("workflow %s is not pending (status: %s)", workflowID, wf.Status)
	}

	go func() {
		_, _ = o.RunWorkflow(ctx, workflowID)
	}()
	return nil
}

// GetWorkflow returns a snapshot copy of the workflow.
func (o *Orchestrator) GetWorkflow(ctx context.Context, workflowID string) (*types.Workflow, error) {
	return o.store.Get(ctx, workflowID)
}

// ListWorkflows returns snapshot copies of all workflows.
func (o *Orchestrator) ListWorkflows(ctx context.Context) ([]*types.Workflow, error) {
	return o.store.List(ctx)
}

// CancelWorkflow signals the running workflow to stop between steps.
func (o *Orchestrator) CancelWorkflow(ctx context.Context, workflowID string) error {
	wf, err := o.store.Get(ctx, workflowID)
	if err != nil {
		return err
	}
	if wf.Status != types.WorkflowStatusRunning {
		return fmt.Errorf("workflow %s is not running (status: %s)", workflowID, wf.Status)
	}

	o.mu.RLock()
	cancel, ok := o.cancels[workflowID]
	o.mu.RUnlock()
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
