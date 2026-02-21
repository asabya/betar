package agent

import (
	"context"
	"iter"
	"sort"
	"strings"
	"sync"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

type PatternResponse struct {
	Pattern  string
	Response string
}

type MockLLM struct {
	mu       sync.RWMutex
	patterns []PatternResponse
}

func NewMockLLM(responses map[string]string) *MockLLM {
	patterns := make([]PatternResponse, 0, len(responses))
	for pattern, resp := range responses {
		patterns = append(patterns, PatternResponse{Pattern: pattern, Response: resp})
	}
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Pattern < patterns[j].Pattern
	})
	return &MockLLM{
		patterns: patterns,
	}
}

func NewMockLLMOrdered(patterns []PatternResponse) *MockLLM {
	return &MockLLM{
		patterns: patterns,
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
		for _, pr := range m.patterns {
			if strings.Contains(inputText, pr.Pattern) {
				responseText = pr.Response
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
