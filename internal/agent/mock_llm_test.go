package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

func TestMockLLM_GenerateContent(t *testing.T) {
	responses := map[string]string{
		"What is 2+2?": "4",
		"Hello":        "Hi there!",
	}
	llm := NewMockLLM(responses)

	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromText("What is 2+2?", genai.RoleUser),
		},
	}

	var gotResponse *model.LLMResponse
	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			t.Fatalf("GenerateContent failed: %v", err)
		}
		gotResponse = resp
	}

	if gotResponse == nil || gotResponse.Content == nil {
		t.Fatal("expected response content")
	}
	if len(gotResponse.Content.Parts) == 0 {
		t.Fatal("expected response parts")
	}
	if gotResponse.Content.Parts[0].Text != "4" {
		t.Errorf("expected '4', got %q", gotResponse.Content.Parts[0].Text)
	}
}

func TestMockLLM_EchoesInputOnNoMatch(t *testing.T) {
	responses := map[string]string{
		"pattern": "matched response",
	}
	llm := NewMockLLM(responses)

	inputText := "This is unmatched input text"
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromText(inputText, genai.RoleUser),
		},
	}

	var gotResponse *model.LLMResponse
	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			t.Fatalf("GenerateContent failed: %v", err)
		}
		gotResponse = resp
	}

	if gotResponse.Content.Parts[0].Text != inputText {
		t.Errorf("expected input text %q to be echoed, got %q", inputText, gotResponse.Content.Parts[0].Text)
	}
}

func TestMockLLM_MultiplePatternLookups(t *testing.T) {
	responses := map[string]string{
		"foo": "foo response",
		"bar": "bar response",
		"baz": "baz response",
	}
	llm := NewMockLLM(responses)

	tests := []struct {
		input    string
		expected string
	}{
		{"something foo here", "foo response"},
		{"bar is present", "bar response"},
		{"contains baz", "baz response"},
		{"no match here", "no match here"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			req := &model.LLMRequest{
				Contents: []*genai.Content{
					genai.NewContentFromText(tt.input, genai.RoleUser),
				},
			}

			var gotResponse *model.LLMResponse
			for resp, err := range llm.GenerateContent(context.Background(), req, false) {
				if err != nil {
					t.Fatalf("GenerateContent failed: %v", err)
				}
				gotResponse = resp
			}

			if gotResponse.Content.Parts[0].Text != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, gotResponse.Content.Parts[0].Text)
			}
		})
	}
}

func TestMockLLM_ConcurrentAccess(t *testing.T) {
	responses := map[string]string{
		"concurrent": "safe response",
	}
	llm := NewMockLLM(responses)

	var wg sync.WaitGroup
	numGoroutines := 100
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			input := "concurrent test"
			req := &model.LLMRequest{
				Contents: []*genai.Content{
					genai.NewContentFromText(input, genai.RoleUser),
				},
			}

			for resp, err := range llm.GenerateContent(context.Background(), req, false) {
				if err != nil {
					errChan <- err
					return
				}
				if resp.Content.Parts[0].Text != "safe response" {
					errChan <- fmt.Errorf("unexpected response: got %q, want %q", resp.Content.Parts[0].Text, "safe response")
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			t.Fatalf("concurrent access failed: %v", err)
		}
	}
}
