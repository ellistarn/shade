package conversation

import (
	"encoding/json"
	"time"
)

// Provider is the interface for conversation sources. Each provider knows how to
// discover and normalize conversations from a specific agent or platform.
type Provider interface {
	// Name returns a human-readable name for this source (e.g. "OpenCode").
	Name() string
	// Conversations returns all conversations available from this source.
	// Returns (nil, nil) if the source data doesn't exist on this machine.
	Conversations() ([]Conversation, error)
}

// Providers returns the default set of conversation providers.
func Providers() []Provider {
	return []Provider{
		&OpenCode{},
		&ClaudeCode{},
		&Kiro{},
		&KiroCLI{},
	}
}

// Conversation is the normalized representation of a conversation from any agent.
type Conversation struct {
	SchemaVersion  int       `json:"schema_version"`
	Source         string    `json:"source"`
	ConversationID string    `json:"conversation_id"`
	Project        string    `json:"project"`
	Title          string    `json:"title"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ParentID       string    `json:"parent_id,omitempty"`
	SubagentIDs    []string  `json:"subagent_ids,omitempty"`
	Messages       []Message `json:"messages"`
}

type Message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	Timestamp time.Time  `json:"timestamp"`
	Model     string     `json:"model,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	Name   string          `json:"name"`
	Input  json.RawMessage `json:"input,omitempty"`
	Output json.RawMessage `json:"output,omitempty"`
}
