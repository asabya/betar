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

func TestMockRuntime_DeleteAgent(t *testing.T) {
	runtime := NewMockRuntime(map[string]string{})

	agentID, err := runtime.CreateAgent(context.Background(), AgentSpec{
		Name:        "ToDelete",
		Description: "Agent to delete",
	})
	if err != nil {
		t.Fatalf("CreateAgent failed: %v", err)
	}

	if err := runtime.DeleteAgent(context.Background(), agentID); err != nil {
		t.Fatalf("DeleteAgent failed: %v", err)
	}

	_, err = runtime.RunTask(context.Background(), types.TaskRequest{
		AgentID: agentID,
		Input:   "test",
	})
	if err == nil {
		t.Fatal("expected error for deleted agent")
	}
}

func TestMockRuntime_ErrorCases(t *testing.T) {
	runtime := NewMockRuntime(map[string]string{"test": "response"})

	t.Run("empty agent name", func(t *testing.T) {
		_, err := runtime.CreateAgent(context.Background(), AgentSpec{Name: ""})
		if err == nil {
			t.Fatal("expected error for empty name")
		}
	})

	t.Run("empty agent ID for delete", func(t *testing.T) {
		err := runtime.DeleteAgent(context.Background(), "")
		if err == nil {
			t.Fatal("expected error for empty agent ID")
		}
	})

	t.Run("delete non-existent agent", func(t *testing.T) {
		err := runtime.DeleteAgent(context.Background(), "non-existent")
		if err == nil {
			t.Fatal("expected error for non-existent agent")
		}
	})

	t.Run("run task with empty agent ID", func(t *testing.T) {
		_, err := runtime.RunTask(context.Background(), types.TaskRequest{
			AgentID: "",
			Input:   "test",
		})
		if err == nil {
			t.Fatal("expected error for empty agent ID")
		}
	})

	t.Run("run task with empty input", func(t *testing.T) {
		agentID, _ := runtime.CreateAgent(context.Background(), AgentSpec{Name: "TestAgent"})
		_, err := runtime.RunTask(context.Background(), types.TaskRequest{
			AgentID: agentID,
			Input:   "",
		})
		if err == nil {
			t.Fatal("expected error for empty input")
		}
	})

	t.Run("run task with non-existent agent", func(t *testing.T) {
		_, err := runtime.RunTask(context.Background(), types.TaskRequest{
			AgentID: "non-existent",
			Input:   "test",
		})
		if err == nil {
			t.Fatal("expected error for non-existent agent")
		}
	})
}

func TestMockRuntime_MultipleAgents(t *testing.T) {
	runtime := NewMockRuntime(map[string]string{
		"sum":  "15",
		"diff": "5",
	})

	agent1ID, err := runtime.CreateAgent(context.Background(), AgentSpec{
		Name:        "SumAgent",
		Description: "Sums numbers",
	})
	if err != nil {
		t.Fatalf("CreateAgent 1 failed: %v", err)
	}

	agent2ID, err := runtime.CreateAgent(context.Background(), AgentSpec{
		Name:        "DiffAgent",
		Description: "Diffs numbers",
	})
	if err != nil {
		t.Fatalf("CreateAgent 2 failed: %v", err)
	}

	result1, err := runtime.RunTask(context.Background(), types.TaskRequest{
		AgentID: agent1ID,
		Input:   "sum of 10 and 5",
	})
	if err != nil {
		t.Fatalf("RunTask 1 failed: %v", err)
	}
	if result1.Output != "15" {
		t.Errorf("agent1: expected '15', got %q", result1.Output)
	}

	result2, err := runtime.RunTask(context.Background(), types.TaskRequest{
		AgentID: agent2ID,
		Input:   "diff of 10 and 5",
	})
	if err != nil {
		t.Fatalf("RunTask 2 failed: %v", err)
	}
	if result2.Output != "5" {
		t.Errorf("agent2: expected '5', got %q", result2.Output)
	}
}
