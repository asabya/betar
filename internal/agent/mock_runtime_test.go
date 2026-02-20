package agent

import (
	"context"
	"testing"

	"github.com/asabya/betar/pkg/types"
)

func TestMockRuntime_CreateAndRunAgent(t *testing.T) {
	responses := map[string]string{
		"calculate": "42",
	}
	runtime := NewMockRuntime(responses)

	agentID, err := runtime.CreateAgent(context.Background(), AgentSpec{
		Name:        "TestAgent",
		Description: "A test agent",
	})
	if err != nil {
		t.Fatalf("CreateAgent failed: %v", err)
	}
	if agentID == "" {
		t.Fatal("expected non-empty agent ID")
	}

	result, err := runtime.RunTask(context.Background(), types.TaskRequest{
		AgentID: agentID,
		Input:   "calculate something",
	})
	if err != nil {
		t.Fatalf("RunTask failed: %v", err)
	}
	if result.Output != "42" {
		t.Errorf("expected '42', got %q", result.Output)
	}
}
