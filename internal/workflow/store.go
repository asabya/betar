package workflow

import (
	"context"
	"fmt"
	"sync"

	"github.com/asabya/betar/pkg/types"
)

// WorkflowStore persists workflows. MemoryStore provides an in-memory
// implementation; a LevelDB-backed store can be added later.
type WorkflowStore interface {
	Save(ctx context.Context, wf *types.Workflow) error
	Get(ctx context.Context, id string) (*types.Workflow, error)
	List(ctx context.Context) ([]*types.Workflow, error)
}

// MemoryStore is an in-memory WorkflowStore implementation.
type MemoryStore struct {
	mu        sync.RWMutex
	workflows map[string]*types.Workflow
}

// NewMemoryStore creates an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		workflows: make(map[string]*types.Workflow),
	}
}

// Save stores a workflow, overwriting any previous version with the same ID.
func (s *MemoryStore) Save(_ context.Context, wf *types.Workflow) error {
	if wf == nil {
		return fmt.Errorf("cannot save nil workflow")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workflows[wf.ID] = copyWorkflow(wf)
	return nil
}

// Get retrieves a workflow by ID, returning a copy.
func (s *MemoryStore) Get(_ context.Context, id string) (*types.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wf, ok := s.workflows[id]
	if !ok {
		return nil, fmt.Errorf("workflow %s not found", id)
	}
	return copyWorkflow(wf), nil
}

// List returns copies of all stored workflows.
func (s *MemoryStore) List(_ context.Context) ([]*types.Workflow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*types.Workflow, 0, len(s.workflows))
	for _, wf := range s.workflows {
		result = append(result, copyWorkflow(wf))
	}
	return result, nil
}
