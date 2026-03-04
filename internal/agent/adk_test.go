package agent

import (
	"strings"
	"testing"
)

func TestNewADKRuntimeRequiresProvider(t *testing.T) {
	t.Parallel()
	_, err := NewADKRuntime(ADKConfig{
		ModelName:     "some-model",
		APIKey:        "",
		OpenAIAPIKey:  "",
		OpenAIBaseURL: "",
		Provider:      "",
	})
	if err == nil {
		t.Fatal("expected error when no provider credentials are configured")
	}
	if !strings.Contains(err.Error(), "llm provider not available") {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestNewADKRuntimeAutoDetectsGoogle(t *testing.T) {
	t.Parallel()
	rt, err := NewADKRuntime(ADKConfig{
		ModelName: "gemini-2.5-flash",
		APIKey:    "fake-google-key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.provider != "google" {
		t.Fatalf("expected provider 'google', got %q", rt.provider)
	}
}

func TestNewADKRuntimeAutoDetectsOpenAI(t *testing.T) {
	t.Parallel()
	rt, err := NewADKRuntime(ADKConfig{
		ModelName:    "gpt-4o",
		OpenAIAPIKey: "fake-openai-key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.provider != "openai" {
		t.Fatalf("expected provider 'openai', got %q", rt.provider)
	}
}

func TestNewADKRuntimeOpenAIWithBaseURLOnly(t *testing.T) {
	t.Parallel()
	// Ollama: no API key needed, just base URL
	rt, err := NewADKRuntime(ADKConfig{
		ModelName:     "llama3",
		OpenAIBaseURL: "http://localhost:11434/v1/",
		Provider:      "openai",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.provider != "openai" {
		t.Fatalf("expected provider 'openai', got %q", rt.provider)
	}
	if rt.openAIBaseURL != "http://localhost:11434/v1/" {
		t.Fatalf("base URL not stored: %q", rt.openAIBaseURL)
	}
}

func TestNewADKRuntimeExplicitGoogleOverrides(t *testing.T) {
	t.Parallel()
	rt, err := NewADKRuntime(ADKConfig{
		ModelName:    "gemini-2.5-flash",
		APIKey:       "google-key",
		OpenAIAPIKey: "also-present",
		Provider:     "google",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rt.provider != "google" {
		t.Fatalf("expected provider 'google', got %q", rt.provider)
	}
}

func TestSanitizeAgentName(t *testing.T) {
	t.Parallel()
	if got := sanitizeAgentName("My Agent 1"); got != "my_agent_1" {
		t.Fatalf("unexpected sanitized name: %s", got)
	}
	if got := sanitizeAgentName("user"); got != "betar_user_agent" {
		t.Fatalf("unexpected reserved-name conversion: %s", got)
	}
}
