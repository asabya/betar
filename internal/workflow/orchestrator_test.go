package workflow

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/asabya/betar/pkg/types"
)

// mockExecutor implements TaskExecutor with configurable per-agent behaviour.
type mockExecutor struct {
	mu       sync.Mutex
	handlers map[string]func(ctx context.Context, input string) (string, error)
	calls    []executorCall // records every call for inspection
}

type executorCall struct {
	AgentID string
	Input   string
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{
		handlers: make(map[string]func(ctx context.Context, input string) (string, error)),
	}
}

func (m *mockExecutor) on(agentID string, fn func(ctx context.Context, input string) (string, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[agentID] = fn
}

func (m *mockExecutor) ExecuteTask(ctx context.Context, agentID, input string) (string, error) {
	m.mu.Lock()
	m.calls = append(m.calls, executorCall{AgentID: agentID, Input: input})
	fn, ok := m.handlers[agentID]
	m.mu.Unlock()

	if !ok {
		return "", fmt.Errorf("no handler for agent %s", agentID)
	}
	return fn(ctx, input)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestThreeStepChainPipesOutput(t *testing.T) {
	exec := newMockExecutor()
	exec.on("agent-1", func(_ context.Context, input string) (string, error) {
		return input + "+out1", nil
	})
	exec.on("agent-2", func(_ context.Context, input string) (string, error) {
		return input + "+out2", nil
	})
	exec.on("agent-3", func(_ context.Context, input string) (string, error) {
		return input + "+out3", nil
	})

	orch := NewOrchestrator(exec)
	wf, err := orch.CreateWorkflow(context.Background(), types.WorkflowDefinition{
		AgentIDs: []string{"agent-1", "agent-2", "agent-3"},
		Input:    "seed",
	})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	result, err := orch.RunWorkflow(context.Background(), wf.ID)
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}

	if result.Status != types.WorkflowStatusCompleted {
		t.Fatalf("expected status completed, got %s", result.Status)
	}
	if result.Output != "seed+out1+out2+out3" {
		t.Fatalf("expected output 'seed+out1+out2+out3', got %q", result.Output)
	}

	// Verify piping: step 0 input = "seed", step 1 input = "seed+out1", step 2 input = "seed+out1+out2"
	expected := []struct{ input, output string }{
		{"seed", "seed+out1"},
		{"seed+out1", "seed+out1+out2"},
		{"seed+out1+out2", "seed+out1+out2+out3"},
	}
	for i, exp := range expected {
		s := result.Steps[i]
		if s.Input != exp.input {
			t.Errorf("step %d: expected input %q, got %q", i, exp.input, s.Input)
		}
		if s.Output != exp.output {
			t.Errorf("step %d: expected output %q, got %q", i, exp.output, s.Output)
		}
		if s.Status != types.StepStatusCompleted {
			t.Errorf("step %d: expected status completed, got %s", i, s.Status)
		}
		if s.StartedAt == nil || s.CompletedAt == nil {
			t.Errorf("step %d: timestamps not set", i)
		}
	}
}

func TestStepFailureSkipsRemaining(t *testing.T) {
	exec := newMockExecutor()
	exec.on("agent-1", func(_ context.Context, input string) (string, error) {
		return "ok", nil
	})
	exec.on("agent-2", func(_ context.Context, input string) (string, error) {
		return "", fmt.Errorf("boom")
	})
	exec.on("agent-3", func(_ context.Context, input string) (string, error) {
		return "should-not-run", nil
	})

	orch := NewOrchestrator(exec)
	wf, err := orch.CreateWorkflow(context.Background(), types.WorkflowDefinition{
		AgentIDs: []string{"agent-1", "agent-2", "agent-3"},
		Input:    "start",
	})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	result, err := orch.RunWorkflow(context.Background(), wf.ID)
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}

	if result.Status != types.WorkflowStatusFailed {
		t.Fatalf("expected workflow status failed, got %s", result.Status)
	}
	if result.Steps[0].Status != types.StepStatusCompleted {
		t.Errorf("step 0: expected completed, got %s", result.Steps[0].Status)
	}
	if result.Steps[1].Status != types.StepStatusFailed {
		t.Errorf("step 1: expected failed, got %s", result.Steps[1].Status)
	}
	if result.Steps[1].Error == "" {
		t.Error("step 1: expected error message")
	}
	if result.Steps[2].Status != types.StepStatusSkipped {
		t.Errorf("step 2: expected skipped, got %s", result.Steps[2].Status)
	}
}

func TestCancelMidWorkflow(t *testing.T) {
	exec := newMockExecutor()

	step1Done := make(chan struct{})

	exec.on("agent-1", func(_ context.Context, input string) (string, error) {
		return "out1", nil
	})
	exec.on("agent-2", func(ctx context.Context, input string) (string, error) {
		// Signal that step 2 is running, then block until context is canceled.
		close(step1Done)
		<-ctx.Done()
		return "", ctx.Err()
	})
	exec.on("agent-3", func(_ context.Context, input string) (string, error) {
		return "should-not-run", nil
	})

	orch := NewOrchestrator(exec)
	wf, err := orch.CreateWorkflow(context.Background(), types.WorkflowDefinition{
		AgentIDs: []string{"agent-1", "agent-2", "agent-3"},
		Input:    "go",
	})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	var result *types.Workflow
	done := make(chan struct{})
	go func() {
		result, _ = orch.RunWorkflow(context.Background(), wf.ID)
		close(done)
	}()

	// Wait until step 2 starts, then cancel.
	select {
	case <-step1Done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for step 2 to start")
	}

	if err := orch.CancelWorkflow(wf.ID); err != nil {
		t.Fatalf("CancelWorkflow: %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for RunWorkflow to return")
	}

	// Step 2 fails because its context was canceled; step 3 should be skipped.
	// The workflow status will be "failed" since step 2 returned an error.
	if result.Status != types.WorkflowStatusFailed && result.Status != types.WorkflowStatusCanceled {
		t.Fatalf("expected workflow status failed or canceled, got %s", result.Status)
	}
	if result.Steps[0].Status != types.StepStatusCompleted {
		t.Errorf("step 0: expected completed, got %s", result.Steps[0].Status)
	}
	if result.Steps[2].Status != types.StepStatusSkipped {
		t.Errorf("step 2: expected skipped, got %s", result.Steps[2].Status)
	}
}

func TestSingleStepWorkflow(t *testing.T) {
	exec := newMockExecutor()
	exec.on("solo", func(_ context.Context, input string) (string, error) {
		return "result-" + input, nil
	})

	orch := NewOrchestrator(exec)
	wf, err := orch.CreateWorkflow(context.Background(), types.WorkflowDefinition{
		AgentIDs: []string{"solo"},
		Input:    "ping",
	})
	if err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	result, err := orch.RunWorkflow(context.Background(), wf.ID)
	if err != nil {
		t.Fatalf("RunWorkflow: %v", err)
	}

	if result.Status != types.WorkflowStatusCompleted {
		t.Fatalf("expected completed, got %s", result.Status)
	}
	if result.Output != "result-ping" {
		t.Fatalf("expected output 'result-ping', got %q", result.Output)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(result.Steps))
	}
	if result.Steps[0].Status != types.StepStatusCompleted {
		t.Errorf("step 0: expected completed, got %s", result.Steps[0].Status)
	}
}

func TestEmptyChainRejected(t *testing.T) {
	exec := newMockExecutor()
	orch := NewOrchestrator(exec)

	_, err := orch.CreateWorkflow(context.Background(), types.WorkflowDefinition{
		AgentIDs: []string{},
		Input:    "something",
	})
	if err == nil {
		t.Fatal("expected error for empty agent list, got nil")
	}
}

func TestEmptyInputRejected(t *testing.T) {
	exec := newMockExecutor()
	orch := NewOrchestrator(exec)

	_, err := orch.CreateWorkflow(context.Background(), types.WorkflowDefinition{
		AgentIDs: []string{"agent-1"},
		Input:    "",
	})
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}
