package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/asabya/betar/pkg/types"
	datastore "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	leveldb "github.com/ipfs/go-ds-leveldb"
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

// LevelDBStore is a LevelDB-backed WorkflowStore.
type LevelDBStore struct {
	ds datastore.Datastore
}

// NewLevelDBStore opens (or creates) a LevelDB datastore at dataDir/workflows.
func NewLevelDBStore(dataDir string) (*LevelDBStore, error) {
	ds, err := leveldb.NewDatastore(dataDir+"/workflows", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open workflow store at %s/workflows: %w", dataDir, err)
	}
	return &LevelDBStore{ds: ds}, nil
}

// Close releases the underlying datastore.
func (s *LevelDBStore) Close() error {
	return s.ds.Close()
}

func workflowKey(id string) datastore.Key {
	return datastore.NewKey("/workflows/" + id)
}

func (s *LevelDBStore) Save(ctx context.Context, wf *types.Workflow) error {
	if wf == nil {
		return fmt.Errorf("cannot save nil workflow")
	}
	data, err := json.Marshal(wf)
	if err != nil {
		return fmt.Errorf("workflow marshal: %w", err)
	}
	return s.ds.Put(ctx, workflowKey(wf.ID), data)
}

func (s *LevelDBStore) Get(ctx context.Context, id string) (*types.Workflow, error) {
	data, err := s.ds.Get(ctx, workflowKey(id))
	if err == datastore.ErrNotFound {
		return nil, fmt.Errorf("workflow %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("workflow store get: %w", err)
	}
	var wf types.Workflow
	if err := json.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("workflow unmarshal: %w", err)
	}
	return &wf, nil
}

func (s *LevelDBStore) List(ctx context.Context) ([]*types.Workflow, error) {
	results, err := s.ds.Query(ctx, query.Query{
		Prefix: "/workflows/",
	})
	if err != nil {
		return nil, fmt.Errorf("workflow store query: %w", err)
	}
	defer results.Close()

	var workflows []*types.Workflow
	for result := range results.Next() {
		if result.Error != nil {
			return nil, result.Error
		}
		var wf types.Workflow
		if err := json.Unmarshal(result.Value, &wf); err != nil {
			continue
		}
		workflows = append(workflows, &wf)
	}
	if workflows == nil {
		workflows = []*types.Workflow{}
	}
	return workflows, nil
}
