package workflow

import (
	"context"
	"testing"
	"time"

	"github.com/asabya/betar/pkg/types"
)

func TestLevelDBStore_SaveAndGet(t *testing.T) {
	store, err := NewLevelDBStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLevelDBStore: %v", err)
	}
	defer store.Close()

	now := time.Now()
	wf := &types.Workflow{
		ID:        "wf-1",
		Status:    types.WorkflowStatusPending,
		Input:     "hello",
		CreatedAt: now,
		UpdatedAt: now,
		Steps: []types.WorkflowStep{
			{Index: 0, AgentID: "agent-1", Status: types.StepStatusPending},
		},
	}

	ctx := context.Background()
	if err := store.Save(ctx, wf); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Get(ctx, "wf-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "wf-1" || got.Input != "hello" || got.Status != types.WorkflowStatusPending {
		t.Fatalf("unexpected workflow: %+v", got)
	}
	if len(got.Steps) != 1 || got.Steps[0].AgentID != "agent-1" {
		t.Fatalf("unexpected steps: %+v", got.Steps)
	}
}

func TestLevelDBStore_List(t *testing.T) {
	store, err := NewLevelDBStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLevelDBStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now()

	for _, id := range []string{"wf-a", "wf-b", "wf-c"} {
		if err := store.Save(ctx, &types.Workflow{
			ID:        id,
			Status:    types.WorkflowStatusPending,
			Input:     "input-" + id,
			CreatedAt: now,
			UpdatedAt: now,
		}); err != nil {
			t.Fatalf("Save %s: %v", id, err)
		}
	}

	workflows, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(workflows) != 3 {
		t.Fatalf("expected 3 workflows, got %d", len(workflows))
	}
}

func TestLevelDBStore_GetNotFound(t *testing.T) {
	store, err := NewLevelDBStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLevelDBStore: %v", err)
	}
	defer store.Close()

	_, err = store.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent workflow, got nil")
	}
}

func TestLevelDBStore_Persistence(t *testing.T) {
	dir := t.TempDir()

	store, err := NewLevelDBStore(dir)
	if err != nil {
		t.Fatalf("NewLevelDBStore: %v", err)
	}

	ctx := context.Background()
	now := time.Now()
	if err := store.Save(ctx, &types.Workflow{
		ID:        "persist-1",
		Status:    types.WorkflowStatusCompleted,
		Input:     "data",
		Output:    "result",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	store.Close()

	// Reopen
	store2, err := NewLevelDBStore(dir)
	if err != nil {
		t.Fatalf("NewLevelDBStore (reopen): %v", err)
	}
	defer store2.Close()

	got, err := store2.Get(ctx, "persist-1")
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}
	if got.Output != "result" || got.Status != types.WorkflowStatusCompleted {
		t.Fatalf("unexpected workflow after reopen: %+v", got)
	}
}
