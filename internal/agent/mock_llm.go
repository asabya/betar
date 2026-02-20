package agent

import (
	"context"
	"iter"
	"strings"
	"sync"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

type MockLLM struct {
	mu        sync.RWMutex
	responses map[string]string
}

func NewMockLLM(responses map[string]string) *MockLLM {
	return &MockLLM{
		responses: responses,
	}
}

func (m *MockLLM) Name() string {
	return "mock-llm"
}

func (m *MockLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		m.mu.RLock()
		defer m.mu.RUnlock()

		var inputText string
		if len(req.Contents) > 0 && len(req.Contents[0].Parts) > 0 {
			inputText = req.Contents[0].Parts[0].Text
		}

		responseText := inputText
		for pattern, resp := range m.responses {
			if strings.Contains(inputText, pattern) {
				responseText = resp
				break
			}
		}

		yield(&model.LLMResponse{
			Content: &genai.Content{
				Role:  "model",
				Parts: []*genai.Part{{Text: responseText}},
			},
		}, nil)
	}
}
