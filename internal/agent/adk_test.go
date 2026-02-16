package agent

import "testing"

func TestNewADKRuntimeRequiresAPIKey(t *testing.T) {
	t.Parallel()

	_, err := NewADKRuntime(ADKConfig{ModelName: "gemini-2.5-flash", APIKey: ""})
	if err == nil {
		t.Fatalf("expected error when API key is missing")
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
