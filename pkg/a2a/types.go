package a2a

// AgentCard describes an agent's capabilities for A2A discovery.
// Maps from types.AgentListing.
type AgentCard struct {
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	URL          string   `json:"url"`                    // libp2p multiaddr
	Version      string   `json:"version,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Skills       []Skill  `json:"skills,omitempty"`
}

// Skill represents a specific capability an agent advertises.
type Skill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// TaskState represents the A2A task lifecycle state.
type TaskState string

const (
	TaskStateSubmitted     TaskState = "submitted"
	TaskStateWorking       TaskState = "working"
	TaskStateInputRequired TaskState = "input-required"
	TaskStateCompleted     TaskState = "completed"
	TaskStateFailed        TaskState = "failed"
	TaskStateCanceled      TaskState = "canceled"
)

// Task represents an A2A protocol task.
type Task struct {
	ID        string     `json:"id"`
	Status    TaskState  `json:"status"`
	Artifacts []Artifact `json:"artifacts,omitempty"`
}

// Artifact represents a piece of output produced by a task.
type Artifact struct {
	Name  string `json:"name,omitempty"`
	Parts []Part `json:"parts"`
}

// Part represents a single content element within an artifact.
type Part struct {
	Type    string `json:"type"`    // "text", "data", etc.
	Content string `json:"content"`
}
