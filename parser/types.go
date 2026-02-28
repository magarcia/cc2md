package parser

import (
	"encoding/json"
)

// JsonlLine represents a single line from a Claude Code session JSONL file.
type JsonlLine struct {
	ParentUUID  *string         `json:"parentUuid"`
	IsSidechain bool            `json:"isSidechain"`
	UserType    string          `json:"userType"`
	Cwd         string          `json:"cwd"`
	SessionID   string          `json:"sessionId"`
	Version     string          `json:"version"`
	GitBranch   string          `json:"gitBranch,omitempty"`
	Type        string          `json:"type"`
	UUID        string          `json:"uuid"`
	Timestamp   string          `json:"timestamp"`
	IsMeta      bool            `json:"isMeta,omitempty"`
	Message     json.RawMessage `json:"message"`
}

// UserMessage represents a user's message.
type UserMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// ContentAsString returns the content as a string if it is a JSON string.
func (m *UserMessage) ContentAsString() (string, bool) {
	if len(m.Content) == 0 {
		return "", false
	}
	if m.Content[0] == '"' {
		var s string
		if err := json.Unmarshal(m.Content, &s); err != nil {
			return "", false
		}
		return s, true
	}
	return "", false
}

// ContentAsBlocks parses the content as an array of ContentBlock.
func (m *UserMessage) ContentAsBlocks() ([]ContentBlock, error) {
	if len(m.Content) == 0 {
		return nil, nil
	}
	if m.Content[0] == '[' {
		var blocks []ContentBlock
		if err := json.Unmarshal(m.Content, &blocks); err != nil {
			return nil, err
		}
		return blocks, nil
	}
	return nil, nil
}

// AssistantMessage represents an assistant's response.
type AssistantMessage struct {
	ID         string         `json:"id,omitempty"`
	Model      string         `json:"model,omitempty"`
	Role       string         `json:"role"`
	StopReason string         `json:"stop_reason,omitempty"`
	Type       string         `json:"type,omitempty"`
	Content    []ContentBlock `json:"content"`
}

// ContentBlock represents a polymorphic content block.
// The Type field discriminates between text, tool_use, tool_result, and thinking.
type ContentBlock struct {
	Type string `json:"type"`

	// text block
	Text string `json:"text,omitempty"`

	// thinking block
	Thinking string `json:"thinking,omitempty"`

	// tool_use block
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// tool_result block
	ToolUseID     string          `json:"tool_use_id,omitempty"`
	IsError       bool            `json:"is_error,omitempty"`
	ResultContent json.RawMessage `json:"-"` // handled by custom unmarshal
}

// UnmarshalJSON implements custom unmarshaling to handle the polymorphic "content" field
// in tool_result blocks (can be string or array of content blocks).
func (cb *ContentBlock) UnmarshalJSON(data []byte) error {
	// Use an alias to avoid infinite recursion
	type Alias ContentBlock
	aux := &struct {
		Content json.RawMessage `json:"content,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(cb),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Store the raw "content" field for tool_result blocks
	if cb.Type == "tool_result" {
		cb.ResultContent = aux.Content
	}

	return nil
}

// InputAsMap parses the Input json.RawMessage into a map.
func (cb *ContentBlock) InputAsMap() map[string]interface{} {
	if len(cb.Input) == 0 {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(cb.Input, &m); err != nil {
		return nil
	}
	return m
}

// ResultContentAsString returns the tool_result content as a string if it's a JSON string.
func (cb *ContentBlock) ResultContentAsString() (string, bool) {
	if len(cb.ResultContent) == 0 {
		return "", false
	}
	if cb.ResultContent[0] == '"' {
		var s string
		if err := json.Unmarshal(cb.ResultContent, &s); err != nil {
			return "", false
		}
		return s, true
	}
	return "", false
}

// ResultContentAsBlocks parses the tool_result content as an array of ContentBlock.
func (cb *ContentBlock) ResultContentAsBlocks() ([]ContentBlock, error) {
	if len(cb.ResultContent) == 0 {
		return nil, nil
	}
	if cb.ResultContent[0] == '[' {
		var blocks []ContentBlock
		if err := json.Unmarshal(cb.ResultContent, &blocks); err != nil {
			return nil, err
		}
		return blocks, nil
	}
	return nil, nil
}

// --- Processed types for the formatter ---

// SessionMetadata contains extracted session information.
type SessionMetadata struct {
	SessionID        string
	Date             string
	Model            string
	WorkingDirectory string
	Version          string
}

// ConversationTurn represents a processed conversation turn.
type ConversationTurn struct {
	Type         string // "user", "assistant", "teammate", "local-command"
	Text         []string
	ToolCalls    []LinkedToolCall
	Thinking     []string
	TeammateName string
	Timestamp    string
}

// LinkedToolCall represents a tool call linked with its result.
type LinkedToolCall struct {
	ID      string
	Name    string
	Input   map[string]interface{}
	Result  *string // nil means no result yet, empty string is valid
	IsError bool
}
