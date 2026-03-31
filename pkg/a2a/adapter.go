package a2a

import (
	"fmt"

	"github.com/asabya/betar/pkg/types"
)

// AgentListingToAgentCard converts a Betar AgentListing to an A2A AgentCard.
func AgentListingToAgentCard(listing *types.AgentListing) *AgentCard {
	url := listing.SellerID
	if len(listing.Addrs) > 0 {
		url = listing.Addrs[0]
	}

	var skills []Skill
	for i, proto := range listing.Protocols {
		skills = append(skills, Skill{
			ID:          fmt.Sprintf("protocol-%d", i),
			Name:        proto,
			Description: fmt.Sprintf("Supports the %s protocol.", proto),
			Examples: []string{
				fmt.Sprintf("Can you perform a task using the %s protocol?", proto),
				fmt.Sprintf("Demonstrate how to interact with the %s protocol.", proto),
			},
			Tags: []string{"coding", "generation", "x402"},
		})
	}

	var capabilities = Capabilities{
		Streaming: false, // Betar doesn't currently support streaming, but this could be enhanced in the future
		Extensions: []Extension{
			{
				URI:         "https://github.com/google-a2a/a2a-x402/v0.1",
				Description: "Supports payments using the x402 protocol.",
				Required:    true,
			},
		},
	}

	return &AgentCard{
		Name:               listing.Name,
		Description:        "Fast code generation with Gemini 2.5 Flash Lite",
		DefaultInputModes:  []string{"text", "text/plain"},
		DefaultOutputModes: []string{"text", "text/plain"},
		Version:            "0.0.1",
		ProtocolVersion:    "0.3.0",
		PrefferedTransport: "JSONRPC",
		URL:                url,
		Skills:             skills,
		Capabilities:       capabilities,
	}
}

// WorkflowStepToA2ATask converts a Betar WorkflowStep to an A2A Task.
func WorkflowStepToA2ATask(step *types.WorkflowStep) *Task {
	task := &Task{
		ID:     fmt.Sprintf("step-%d", step.Index),
		Status: A2ATaskStateFromStepStatus(step.Status),
	}

	if step.Output != "" {
		task.Artifacts = []Artifact{
			{
				Parts: []Part{
					{Type: "text", Content: step.Output},
				},
			},
		}
	}

	return task
}

// A2ATaskStateFromStepStatus maps Betar StepStatus to A2A TaskState.
func A2ATaskStateFromStepStatus(s types.StepStatus) TaskState {
	switch s {
	case types.StepStatusPending:
		return TaskStateSubmitted
	case types.StepStatusRunning:
		return TaskStateWorking
	case types.StepStatusCompleted:
		return TaskStateCompleted
	case types.StepStatusFailed:
		return TaskStateFailed
	case types.StepStatusSkipped:
		return TaskStateCanceled
	default:
		return TaskStateSubmitted
	}
}
