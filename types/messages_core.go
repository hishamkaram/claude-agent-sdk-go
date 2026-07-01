package types

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// UserMessage represents a message from the user.
type UserMessage struct {
	Type            string         `json:"type"`
	Content         interface{}    `json:"content"` // Can be string or []ContentBlock
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
	IsReplay        bool           `json:"isReplay,omitempty"`
	UUID            string         `json:"uuid,omitempty"` // User message identifier for checkpoint targeting
	SessionID       string         `json:"session_id,omitempty"`
	Timestamp       string         `json:"timestamp,omitempty"`
	ToolUseResult   *ToolUseResult `json:"tool_use_result,omitempty"`
}

// GetMessageType returns the type of the message.
func (m *UserMessage) GetMessageType() string {
	return m.Type
}

// ShouldDisplayToUser returns true for user messages (always display).
func (m *UserMessage) ShouldDisplayToUser() bool {
	return true
}

func (m *UserMessage) isMessage() {}

// UnmarshalJSON implements custom unmarshaling for UserMessage to handle content union type.
func (m *UserMessage) UnmarshalJSON(data []byte) error {
	type Alias UserMessage
	aux := &struct {
		Content          json.RawMessage            `json:"content"`
		Message          map[string]json.RawMessage `json:"message"` // Handle nested message format from CLI
		ToolUseResultRaw json.RawMessage            `json:"tool_use_result"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("types.UserMessage.UnmarshalJSON: %w", err)
	}
	m.ToolUseResult = parseWorkflowToolUseResult(aux.ToolUseResultRaw)

	var contentRaw json.RawMessage

	// Check if content is in nested message.content (Claude CLI format)
	if aux.Message != nil {
		if content, ok := aux.Message["content"]; ok {
			contentRaw = content
		}
		// Also extract parent_tool_use_id from nested message if present
		if parentToolUseID, ok := aux.Message["parent_tool_use_id"]; ok {
			var id string
			if err := json.Unmarshal(parentToolUseID, &id); err == nil {
				m.ParentToolUseID = &id
			}
		}
	}

	// Fall back to top-level content if nested not found
	if contentRaw == nil && aux.Content != nil {
		contentRaw = aux.Content
	}

	// If we still don't have content, that's an error
	if contentRaw == nil {
		return NewMessageParseError("types.UserMessage.UnmarshalJSON: missing content field")
	}

	// Try to unmarshal as string first
	var contentStr string
	if err := json.Unmarshal(contentRaw, &contentStr); err == nil {
		m.Content = contentStr
		return nil
	}

	// Try to unmarshal as array of content blocks
	var contentArr []json.RawMessage
	if err := json.Unmarshal(contentRaw, &contentArr); err == nil {
		blocks := make([]ContentBlock, len(contentArr))
		for i, rawBlock := range contentArr {
			block, err := UnmarshalContentBlock(rawBlock)
			if err != nil {
				return fmt.Errorf("types.UserMessage.UnmarshalJSON: unmarshal content block %d: %w", i, err)
			}
			blocks[i] = block
		}
		m.Content = blocks
		return nil
	}

	return NewMessageParseError("types.UserMessage.UnmarshalJSON: content must be string or array of content blocks")
}

func parseWorkflowToolUseResult(raw json.RawMessage) *ToolUseResult {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) || raw[0] != '{' {
		return nil
	}
	var result ToolUseResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return &result
}

// AssistantMessage represents a message from Claude assistant.
type AssistantMessage struct {
	Type            string         `json:"type"`
	Content         []ContentBlock `json:"content"`
	Model           string         `json:"model"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
	UUID            string         `json:"uuid,omitempty"`
	SessionID       string         `json:"session_id,omitempty"`
}

// GetMessageType returns the type of the message.
func (m *AssistantMessage) GetMessageType() string {
	return m.Type
}

// ShouldDisplayToUser returns true for assistant messages (always display).
func (m *AssistantMessage) ShouldDisplayToUser() bool {
	return true
}

func (m *AssistantMessage) isMessage() {}

// UnmarshalJSON implements custom unmarshaling for AssistantMessage to handle content blocks.
func (m *AssistantMessage) UnmarshalJSON(data []byte) error {
	type Alias AssistantMessage
	aux := &struct {
		Content []json.RawMessage          `json:"content"`
		Message map[string]json.RawMessage `json:"message"` // Handle nested message format from CLI
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("types.AssistantMessage.UnmarshalJSON: %w", err)
	}

	var contentBlocks []json.RawMessage

	// Check if content is in nested message.content (Claude CLI format)
	if aux.Message != nil {
		if contentRaw, ok := aux.Message["content"]; ok {
			var nested []json.RawMessage
			if err := json.Unmarshal(contentRaw, &nested); err == nil {
				contentBlocks = nested
			}
		}
		// Also extract model from nested message if present
		if modelRaw, ok := aux.Message["model"]; ok {
			var model string
			if err := json.Unmarshal(modelRaw, &model); err == nil {
				m.Model = model
			}
		}
	}

	// Fall back to top-level content if nested not found
	if contentBlocks == nil && aux.Content != nil {
		contentBlocks = aux.Content
	}

	// Unmarshal content blocks
	m.Content = make([]ContentBlock, len(contentBlocks))
	for i, rawBlock := range contentBlocks {
		block, err := UnmarshalContentBlock(rawBlock)
		if err != nil {
			return fmt.Errorf("types.AssistantMessage.UnmarshalJSON: unmarshal content block %d: %w", i, err)
		}
		m.Content[i] = block
	}

	return nil
}

// MarshalJSON implements custom marshaling for AssistantMessage to handle content blocks.
func (m *AssistantMessage) MarshalJSON() ([]byte, error) {
	type Alias AssistantMessage
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}

// SystemMessage represents a system message with metadata.
type SystemMessage struct {
	Type              string                 `json:"type"`
	Subtype           string                 `json:"subtype,omitempty"`
	Data              map[string]interface{} `json:"data,omitempty"`
	Response          map[string]interface{} `json:"response,omitempty"`   // For control_response messages
	Request           map[string]interface{} `json:"request,omitempty"`    // For control_request messages
	RequestID         string                 `json:"request_id,omitempty"` // For control_request/control_response messages (top-level field)
	Tools             []string               `json:"tools,omitempty"`
	CWD               string                 `json:"cwd,omitempty"`
	Model             string                 `json:"model,omitempty"`
	PermissionMode    string                 `json:"permissionMode,omitempty"`
	ClaudeCodeVersion string                 `json:"claude_code_version,omitempty"`
	SessionID         string                 `json:"session_id,omitempty"`
}

// GetMessageType returns the type of the message.
func (m *SystemMessage) GetMessageType() string {
	return m.Type
}

func (m *SystemMessage) isMessage() {}

// IsInit returns true if this is a system init message.
func (m *SystemMessage) IsInit() bool {
	return m.Subtype == SystemSubtypeInit
}

// IsWarning returns true if this is a system warning message.
func (m *SystemMessage) IsWarning() bool {
	return m.Subtype == SystemSubtypeWarning
}

// IsError returns true if this is a system error message.
func (m *SystemMessage) IsError() bool {
	return m.Subtype == SystemSubtypeError
}

// IsInfo returns true if this is a system info message.
func (m *SystemMessage) IsInfo() bool {
	return m.Subtype == SystemSubtypeInfo
}

// IsDebug returns true if this is a system debug message.
func (m *SystemMessage) IsDebug() bool {
	return m.Subtype == SystemSubtypeDebug
}

// ShouldDisplayToUser returns true if this system message should be shown to the user.
// By default, init and debug messages are not shown to users.
func (m *SystemMessage) ShouldDisplayToUser() bool {
	return m.Subtype != SystemSubtypeInit && m.Subtype != SystemSubtypeDebug
}
