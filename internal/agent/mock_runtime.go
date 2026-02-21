package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/asabya/betar/pkg/types"
	"github.com/google/uuid"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type MockRuntime struct {
	mu        sync.RWMutex
	agents    map[string]*mockRuntimeAgent
	responses map[string]string
	appName   string
}

type mockRuntimeAgent struct {
	agent          agent.Agent
	runner         *runner.Runner
	sessionService session.Service
	sessionID      string
	userID         string
}

func NewMockRuntime(responses map[string]string) *MockRuntime {
	return &MockRuntime{
		agents:    make(map[string]*mockRuntimeAgent),
		responses: responses,
		appName:   "betar-test",
	}
}

func (r *MockRuntime) CreateAgent(ctx context.Context, spec AgentSpec) (string, error) {
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return "", fmt.Errorf("agent name is required")
	}

	llm := NewMockLLM(r.responses)

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
		return "", fmt.Errorf("failed to create mock llm agent: %w", err)
	}

	sessionSvc := session.InMemoryService()
	agentID := "mock-" + uuid.NewString()[:8]
	userID := "test-user"
	sessionID := "session-" + agentID

	if _, err := sessionSvc.Create(ctx, &session.CreateRequest{
		AppName:   r.appName,
		UserID:    userID,
		SessionID: sessionID,
	}); err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	runnerInstance, err := runner.New(runner.Config{
		AppName:        r.appName,
		Agent:          a,
		SessionService: sessionSvc,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create runner: %w", err)
	}

	r.mu.Lock()
	r.agents[agentID] = &mockRuntimeAgent{
		agent:          a,
		runner:         runnerInstance,
		sessionService: sessionSvc,
		sessionID:      sessionID,
		userID:         userID,
	}
	r.mu.Unlock()

	return agentID, nil
}

func (r *MockRuntime) DeleteAgent(ctx context.Context, runtimeAgentID string) error {
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

func (r *MockRuntime) RunTask(ctx context.Context, req types.TaskRequest) (*types.TaskResult, error) {
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
			return nil, fmt.Errorf("mock run failed: %w", err)
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
