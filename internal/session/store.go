package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	datastore "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	leveldb "github.com/ipfs/go-ds-leveldb"

	"github.com/asabya/betar/pkg/types"
)

// Store is a LevelDB-backed store for agent sessions.
type Store struct {
	ds datastore.Datastore
}

// NewStore opens (or creates) a LevelDB datastore at dataDir/sessions.
func NewStore(dataDir string) (*Store, error) {
	ds, err := leveldb.NewDatastore(dataDir+"/sessions", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open session store at %s/sessions: %w", dataDir, err)
	}
	return &Store{ds: ds}, nil
}

// Close releases the underlying datastore.
func (s *Store) Close() error {
	return s.ds.Close()
}

// sessionKey returns the datastore key for a (agentID, callerID) pair.
func sessionKey(agentID, callerID string) datastore.Key {
	a := base64.RawURLEncoding.EncodeToString([]byte(agentID))
	c := base64.RawURLEncoding.EncodeToString([]byte(callerID))
	return datastore.NewKey("/sessions/" + a + "/" + c)
}

// agentPrefix returns the datastore key prefix for all sessions of an agent.
func agentPrefix(agentID string) string {
	a := base64.RawURLEncoding.EncodeToString([]byte(agentID))
	return "/sessions/" + a + "/"
}

// Get returns the session for (agentID, callerID). Returns nil, nil if not found.
func (s *Store) Get(ctx context.Context, agentID, callerID string) (*types.Session, error) {
	data, err := s.ds.Get(ctx, sessionKey(agentID, callerID))
	if err == datastore.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("session store get: %w", err)
	}
	var sess types.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("session unmarshal: %w", err)
	}
	return &sess, nil
}

// save persists a session.
func (s *Store) save(ctx context.Context, sess *types.Session) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("session marshal: %w", err)
	}
	return s.ds.Put(ctx, sessionKey(sess.AgentID, sess.CallerID), data)
}

// AddExchange appends an exchange to the session for (agentID, callerID),
// creating a new session if none exists.
func (s *Store) AddExchange(ctx context.Context, agentID, callerID string, ex types.Exchange) error {
	sess, err := s.Get(ctx, agentID, callerID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if sess == nil {
		sess = &types.Session{
			ID:        uuid.NewString(),
			AgentID:   agentID,
			CallerID:  callerID,
			CreatedAt: now,
		}
	}
	sess.Exchanges = append(sess.Exchanges, ex)
	sess.UpdatedAt = now
	return s.save(ctx, sess)
}

// ListByAgent returns all sessions for a given agentID.
func (s *Store) ListByAgent(ctx context.Context, agentID string) ([]*types.Session, error) {
	results, err := s.ds.Query(ctx, query.Query{
		Prefix: agentPrefix(agentID),
	})
	if err != nil {
		return nil, fmt.Errorf("session store query: %w", err)
	}
	defer results.Close()

	var sessions []*types.Session
	for result := range results.Next() {
		if result.Error != nil {
			return nil, result.Error
		}
		var sess types.Session
		if err := json.Unmarshal(result.Value, &sess); err != nil {
			continue
		}
		sessions = append(sessions, &sess)
	}
	return sessions, nil
}
