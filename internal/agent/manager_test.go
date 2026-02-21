package agent

import (
	"context"
	"testing"

	"github.com/asabya/betar/pkg/types"
)

func TestNewManagerWithRuntime(t *testing.T) {
	responses := map[string]string{
		"test": "mock response",
	}
	runtime := NewMockRuntime(responses)

	mgr := NewManagerWithRuntime(runtime, nil, nil, nil, nil, "", nil)
	if mgr == nil {
		t.Fatal("expected manager")
	}

	agentID, err := mgr.runtime.CreateAgent(context.Background(), AgentSpec{
		Name:        "TestAgent",
		Description: "Test",
	})
	if err != nil {
		t.Fatalf("CreateAgent failed: %v", err)
	}

	result, err := mgr.runtime.RunTask(context.Background(), types.TaskRequest{
		AgentID: agentID,
		Input:   "test query",
	})
	if err != nil {
		t.Fatalf("RunTask failed: %v", err)
	}
	if result.Output != "mock response" {
		t.Errorf("expected 'mock response', got %q", result.Output)
	}
}
