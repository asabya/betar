package agent

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"time"

	didkeyctl "github.com/MetaMask/go-did-it/controller/did-key"
	"github.com/MetaMask/go-did-it/crypto/ed25519"
	"github.com/asabya/betar/pkg/types"
	"github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// ADKConfig configures the runtime bridge.
type ADKConfig struct {
	AppName   string
	ModelName string
	APIKey    string
	PrivKey   p2pcrypto.PrivKey // libp2p private key for DID generation
}

// Runtime defines required agent runtime capabilities.
type Runtime interface {
	CreateAgent(ctx context.Context, spec AgentSpec) (string, error)
	DeleteAgent(ctx context.Context, runtimeAgentID string) error
	RunTask(ctx context.Context, req types.TaskRequest) (*types.TaskResult, error)
}

// ADKRuntime is the runtime wrapper used by the manager and worker.
type ADKRuntime struct {
	appName   string
	modelName string
	apiKey    string
	privKey   p2pcrypto.PrivKey

	mu     sync.RWMutex
	agents map[string]*runtimeAgent
}

type runtimeAgent struct {
	agent          agent.Agent
	runner         *runner.Runner
	sessionService session.Service
	sessionID      string
	userID         string
}

// NewADKRuntime creates a runtime wrapper.
func NewADKRuntime(cfg ADKConfig) (*ADKRuntime, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY is required for ADK runtime")
	}
	if strings.TrimSpace(cfg.ModelName) == "" {
		cfg.ModelName = "gemini-2.5-flash"
	}
	if strings.TrimSpace(cfg.AppName) == "" {
		cfg.AppName = "betar"
	}

	return &ADKRuntime{
		appName:   cfg.AppName,
		modelName: cfg.ModelName,
		apiKey:    cfg.APIKey,
		privKey:   cfg.PrivKey,
		agents:    make(map[string]*runtimeAgent),
	}, nil
}

// CreateAgent creates a runtime agent and returns runtime agent ID.
func (r *ADKRuntime) CreateAgent(ctx context.Context, spec AgentSpec) (string, error) {
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return "", fmt.Errorf("agent name is required")
	}

	modelName := r.modelName
	if strings.TrimSpace(spec.Model) != "" {
		modelName = strings.TrimSpace(spec.Model)
	}

	llm, err := gemini.NewModel(ctx, modelName, &genai.ClientConfig{APIKey: r.apiKey})
	if err != nil {
		return "", fmt.Errorf("failed to initialize gemini model: %w", err)
	}

	instruction := strings.TrimSpace(spec.Description)
	if instruction == "" {
		instruction = "Process marketplace tasks and return concise outputs."
	}

	a, err := llmagent.New(llmagent.Config{
		Name:        sanitizeAgentName(name),
		Description: spec.Description,
		Instruction: instruction,
		Model:       llm,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create adk llm agent: %w", err)
	}

	sessionSvc := session.InMemoryService()
	agentID := r.generateAgentDID(name)
	userID := "marketplace"
	sessionID := "session-" + agentID

	if _, err := sessionSvc.Create(ctx, &session.CreateRequest{
		AppName:   r.appName,
		UserID:    userID,
		SessionID: sessionID,
	}); err != nil {
		return "", fmt.Errorf("failed to create adk session: %w", err)
	}

	runnerInstance, err := runner.New(runner.Config{
		AppName:        r.appName,
		Agent:          a,
		SessionService: sessionSvc,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create adk runner: %w", err)
	}

	r.mu.Lock()
	r.agents[agentID] = &runtimeAgent{
		agent:          a,
		runner:         runnerInstance,
		sessionService: sessionSvc,
		sessionID:      sessionID,
		userID:         userID,
	}
	r.mu.Unlock()

	return agentID, nil
}

// DeleteAgent removes runtime agent.
func (r *ADKRuntime) DeleteAgent(ctx context.Context, runtimeAgentID string) error {
	_ = ctx
	if runtimeAgentID == "" {
		return fmt.Errorf("runtime agent ID is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.agents[runtimeAgentID]; !ok {
		return fmt.Errorf("runtime agent not found: %s", runtimeAgentID)
	}

	delete(r.agents, runtimeAgentID)
	return nil
}

// RunTask executes a task via runtime.
func (r *ADKRuntime) RunTask(ctx context.Context, req types.TaskRequest) (*types.TaskResult, error) {
	if req.AgentID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}
	if strings.TrimSpace(req.Input) == "" {
		return nil, fmt.Errorf("task input is required")
	}

	r.mu.RLock()
	ra, ok := r.agents[req.AgentID]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("runtime agent not found: %s", req.AgentID)
	}

	msg := genai.NewContentFromText(req.Input, genai.RoleUser)

	var outputParts []string
	for event, err := range ra.runner.Run(ctx, ra.userID, ra.sessionID, msg, agent.RunConfig{StreamingMode: agent.StreamingModeNone}) {
		if err != nil {
			return nil, fmt.Errorf("adk run failed: %w", err)
		}
		if event == nil || event.Content == nil || !event.IsFinalResponse() {
			continue
		}
		for _, part := range event.Content.Parts {
			if part == nil || part.Text == "" {
				continue
			}
			outputParts = append(outputParts, part.Text)
		}
	}

	output := strings.TrimSpace(strings.Join(outputParts, "\n"))
	if output == "" {
		output = ""
	}

	requestID := req.RequestID
	if requestID == "" {
		requestID = uuid.NewString()
	}

	return &types.TaskResult{
		RequestID: requestID,
		Output:    output,
		Timestamp: time.Now(),
	}, nil
}

func sanitizeAgentName(name string) string {
	clean := strings.TrimSpace(strings.ToLower(name))
	if clean == "" {
		return "betar_agent"
	}

	b := strings.Builder{}
	for _, r := range clean {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}

	result := strings.Trim(b.String(), "_")
	if result == "" {
		return "betar_agent"
	}
	if result == "user" {
		return "betar_user_agent"
	}
	return result
}

// generateAgentDID generates a deterministic DID from the agent name and libp2p private key.
// Uses Ed25519 key derivation to create a consistent agent ID on every run.
func (r *ADKRuntime) generateAgentDID(agentName string) string {
	if r.privKey == nil {
		// Fallback to UUID if no private key available
		return uuid.NewString()
	}

	// Get the public key from the private key
	pubKey := r.privKey.GetPublic()
	pubBytes, err := pubKey.Raw()
	if err != nil {
		// Fallback to UUID if we can't get raw public key
		return uuid.NewString()
	}

	// Derive a seed from: appName + agentName + public key
	derivationInput := r.appName + "/" + agentName + "/" + string(pubBytes)
	hash := sha256.Sum256([]byte(derivationInput))
	seed := hash[:32]

	// Create Ed25519 private key from seed
	edPrivKey, err := ed25519.PrivateKeyFromSeed(seed)
	if err != nil {
		// Fallback to UUID if derivation fails
		return uuid.NewString()
	}

	// Generate DID from the derived Ed25519 key
	did := didkeyctl.FromPrivateKey(edPrivKey)
	return did.String()
}
