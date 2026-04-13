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
	adkopenai "github.com/byebyebruce/adk-go-openai"
	"github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	go_openai "github.com/sashabaranov/go-openai"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/agent/remoteagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// ADKConfig configures the runtime bridge.
type ADKConfig struct {
	AppName   string
	ModelName string
	APIKey    string            // Google API key
	PrivKey   p2pcrypto.PrivKey // libp2p private key for DID generation

	// Provider fields
	Provider      string // "google", "openai", or "" for auto-detect
	OpenAIAPIKey  string // OpenAI-compatible API key
	OpenAIBaseURL string // OpenAI-compatible base URL (empty = api.openai.com)
	AgentAPI      string // URL for custom agent hosting (must implement /execute endpoint)
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
	apiKey    string // Google API key
	privKey   p2pcrypto.PrivKey

	// Provider fields
	provider      string // resolved: "google" or "openai"
	openAIAPIKey  string
	openAIBaseURL string

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
	provider, err := resolveProvider(cfg)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.ModelName) == "" {
		cfg.ModelName = defaultModelName(provider)
	}
	if strings.TrimSpace(cfg.AppName) == "" {
		cfg.AppName = "betar"
	}
	return &ADKRuntime{
		appName:       cfg.AppName,
		modelName:     cfg.ModelName,
		apiKey:        cfg.APIKey,
		privKey:       cfg.PrivKey,
		provider:      provider,
		openAIAPIKey:  cfg.OpenAIAPIKey,
		openAIBaseURL: cfg.OpenAIBaseURL,
		agents:        make(map[string]*runtimeAgent),
	}, nil
}

// resolveProvider determines which LLM provider to use.
func resolveProvider(cfg ADKConfig) (string, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "google":
		if strings.TrimSpace(cfg.APIKey) == "" {
			return "", fmt.Errorf("provider %q requires APIKey (GOOGLE_API_KEY)", cfg.Provider)
		}
		return "google", nil
	case "openai":
		if strings.TrimSpace(cfg.OpenAIAPIKey) == "" && strings.TrimSpace(cfg.OpenAIBaseURL) == "" {
			return "", fmt.Errorf("provider %q requires OpenAIAPIKey (OPENAI_API_KEY) or OpenAIBaseURL (OPENAI_BASE_URL)", cfg.Provider)
		}
		return "openai", nil
	case "":
		// Auto-detect: google wins if APIKey present
		if strings.TrimSpace(cfg.APIKey) != "" {
			return "google", nil
		}
		if strings.TrimSpace(cfg.OpenAIAPIKey) != "" || strings.TrimSpace(cfg.OpenAIBaseURL) != "" {
			return "openai", nil
		}
		return "", fmt.Errorf("llm provider not available: set GOOGLE_API_KEY for Google, or OPENAI_API_KEY/OPENAI_BASE_URL for OpenAI-compatible providers")
	default:
		return "", fmt.Errorf("unknown provider %q: must be \"google\", \"openai\", or empty for auto-detect", cfg.Provider)
	}
}

// defaultModelName returns a sensible default model for the given provider.
func defaultModelName(provider string) string {
	if provider == "openai" {
		return "gpt-4o-mini"
	}
	return "gemini-2.5-flash"
}

// selectLLM creates the LLM for the resolved provider.
func (r *ADKRuntime) selectLLM(ctx context.Context, modelName string) (model.LLM, error) {
	switch r.provider {
	case "google":
		return gemini.NewModel(ctx, modelName, &genai.ClientConfig{APIKey: r.apiKey})
	case "openai":
		cfg := go_openai.DefaultConfig(r.openAIAPIKey)
		if strings.TrimSpace(r.openAIBaseURL) != "" {
			cfg.BaseURL = r.openAIBaseURL
		}
		return adkopenai.NewOpenAIModel(modelName, cfg), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", r.provider)
	}
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

	llm, err := r.selectLLM(ctx, modelName)
	if err != nil {
		return "", fmt.Errorf("failed to initialize llm model: %w", err)
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

func (r *ADKRuntime) CreateHTTPAgent(ctx context.Context, spec AgentSpec) (string, error) {
	if strings.TrimSpace(spec.AgentAPI) == "" {
		return "", fmt.Errorf("Agent API URL is required")
	}
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return "", fmt.Errorf("agent name is required")
	}

	a, err := remoteagent.NewA2A(remoteagent.A2AConfig{
		Name: name,
		// AgentCardSource: "http://localhost:8424/coding-agent/.well-known/agent-card.json",
		AgentCardSource: spec.AgentCardSource,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create remote agent: %w", err)
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

// GenerateDID generates a deterministic did:key from a libp2p private key and a name.
// Uses SHA256(appName/name/pubBytes) as an Ed25519 seed to derive the DID.
// Returns "" on failure.
func GenerateDID(privKey p2pcrypto.PrivKey, appName, name string) string {
	if privKey == nil {
		return ""
	}

	pubKey := privKey.GetPublic()
	pubBytes, err := pubKey.Raw()
	if err != nil {
		return ""
	}

	derivationInput := appName + "/" + name + "/" + string(pubBytes)
	hash := sha256.Sum256([]byte(derivationInput))
	seed := hash[:32]

	edPrivKey, err := ed25519.PrivateKeyFromSeed(seed)
	if err != nil {
		return ""
	}

	did := didkeyctl.FromPrivateKey(edPrivKey)
	return did.String()
}

// generateAgentDID generates a deterministic DID from the agent name and libp2p private key.
func (r *ADKRuntime) generateAgentDID(agentName string) string {
	did := GenerateDID(r.privKey, r.appName, agentName)
	if did == "" {
		return uuid.NewString()
	}
	return did
}
