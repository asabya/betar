package agent

import (
	"context"
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
