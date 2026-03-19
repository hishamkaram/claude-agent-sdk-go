package types

import (
	"encoding/json"
)

// SystemMessageSubtype constants for common system message subtypes
const (
	SystemSubtypeInit        = "init"
	SystemSubtypeWarning     = "warning"
	SystemSubtypeError       = "error"
	SystemSubtypeInfo        = "info"
	SystemSubtypeDebug       = "debug"
	SystemSubtypeSessionEnd  = "session_end"
	SystemSubtypeSessionInfo = "session_info"

	// Typed system subtypes (016-sdk-message-types)
	SystemSubtypeCompactBoundary  = "compact_boundary"
	SystemSubtypeStatus           = "status"
	SystemSubtypeHookStarted      = "hook_started"
	SystemSubtypeHookProgress     = "hook_progress"
	SystemSubtypeHookResponse     = "hook_response"
	SystemSubtypeTaskNotification = "task_notification"
	SystemSubtypeTaskStarted      = "task_started"
	SystemSubtypeTaskProgress     = "task_progress"
	SystemSubtypeFilesPersisted   = "files_persisted"
)

// ResultMessage subtype constants
const (
	ResultSubtypeSuccess                         = "success"
	ResultSubtypeErrorMaxTurns                   = "error_max_turns"
	ResultSubtypeErrorMaxBudget                  = "error_max_budget_usd"
	ResultSubtypeErrorDuringExecution            = "error_during_execution"
	ResultSubtypeErrorMaxStructuredOutputRetries = "error_max_structured_output_retries"
)

// ContentBlock is an interface for all content block types.
// Content blocks can be text, thinking, tool use, or tool result blocks.
type ContentBlock interface {
	GetType() string
	isContentBlock()
}

// TextBlock represents a text content block from Claude.
type TextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// GetType returns the type of the content block.
func (t *TextBlock) GetType() string {
	return t.Type
}

func (t *TextBlock) isContentBlock() {}

// ThinkingBlock represents a thinking content block from Claude.
// This contains Claude's internal reasoning and signature.
type ThinkingBlock struct {
	Type      string `json:"type"`
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

// GetType returns the type of the content block.
func (t *ThinkingBlock) GetType() string {
	return t.Type
}

func (t *ThinkingBlock) isContentBlock() {}

// ToolUseBlock represents a tool use request from Claude.
type ToolUseBlock struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// GetType returns the type of the content block.
func (t *ToolUseBlock) GetType() string {
	return t.Type
}

func (t *ToolUseBlock) isContentBlock() {}

// ToolResultBlock represents the result of a tool execution.
type ToolResultBlock struct {
	Type      string      `json:"type"`
	ToolUseID string      `json:"tool_use_id"`
	Content   interface{} `json:"content,omitempty"`  // Can be string or []map[string]interface{}
	IsError   *bool       `json:"is_error,omitempty"` // Pointer to distinguish between false and not set
}

// GetType returns the type of the content block.
func (t *ToolResultBlock) GetType() string {
	return t.Type
}

func (t *ToolResultBlock) isContentBlock() {}

// UnmarshalContentBlock unmarshals a JSON content block into the appropriate type.
func UnmarshalContentBlock(data []byte) (ContentBlock, error) {
	var typeCheck struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, NewJSONDecodeErrorWithCause("failed to determine content block type", string(data), err)
	}

	switch typeCheck.Type {
	case "text":
		var block TextBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal text block", string(data), err)
		}
		return &block, nil
	case "thinking":
		var block ThinkingBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal thinking block", string(data), err)
		}
		return &block, nil
	case "tool_use":
		var block ToolUseBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal tool_use block", string(data), err)
		}
		return &block, nil
	case "tool_result":
		var block ToolResultBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal tool_result block", string(data), err)
		}
		return &block, nil
	default:
		return nil, NewMessageParseErrorWithType("unknown content block type", typeCheck.Type)
	}
}

// Message is an interface for all message types from Claude.
type Message interface {
	GetMessageType() string
	ShouldDisplayToUser() bool
	isMessage()
}

// ---------------------------------------------------------------------------
// Existing message types
// ---------------------------------------------------------------------------

// UserMessage represents a message from the user.
type UserMessage struct {
	Type            string      `json:"type"`
	Content         interface{} `json:"content"` // Can be string or []ContentBlock
	ParentToolUseID *string     `json:"parent_tool_use_id,omitempty"`
	IsReplay        bool        `json:"isReplay,omitempty"`
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
		Content json.RawMessage            `json:"content"`
		Message map[string]json.RawMessage `json:"message"` // Handle nested message format from CLI
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

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
				return err
			}
			blocks[i] = block
		}
		m.Content = blocks
		return nil
	}

	return NewMessageParseError("types.UserMessage.UnmarshalJSON: content must be string or array of content blocks")
}

// AssistantMessage represents a message from Claude assistant.
type AssistantMessage struct {
	Type            string         `json:"type"`
	Content         []ContentBlock `json:"content"`
	Model           string         `json:"model"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
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
		return err
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
			return err
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
	Type      string                 `json:"type"`
	Subtype   string                 `json:"subtype,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Response  map[string]interface{} `json:"response,omitempty"`   // For control_response messages
	Request   map[string]interface{} `json:"request,omitempty"`    // For control_request messages
	RequestID string                 `json:"request_id,omitempty"` // For control_request/control_response messages (top-level field)
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

// SlashCommand describes an available skill (invoked via /command syntax).
// This mirrors the official TypeScript SDK's SlashCommand type.
type SlashCommand struct {
	// Name is the skill name without the leading slash.
	Name string `json:"name"`
	// Description explains what the skill does.
	Description string `json:"description"`
	// ArgumentHint provides a hint for skill arguments (e.g., "<file>").
	ArgumentHint string `json:"argumentHint,omitempty"`
}

// ModelInfo describes a model available in the current session.
// It is populated from the CLI's initialize response.
type ModelInfo struct {
	// Value is the model identifier (e.g. "claude-3-5-haiku-latest").
	Value string `json:"value"`
	// DisplayName is the human-readable model name (e.g. "Claude 3.5 Haiku").
	DisplayName string `json:"displayName"`
	// Description is an optional short description of the model.
	Description string `json:"description,omitempty"`
}

// InitializeResult holds the parsed response from session initialization.
// It contains available commands, models, and session metadata returned
// by the Claude CLI control protocol initialize response.
type InitializeResult struct {
	// Commands is the list of available slash commands/skills.
	Commands []SlashCommand `json:"commands,omitempty"`
	// Models is the list of models available in this session.
	Models []ModelInfo `json:"models,omitempty"`
	// Raw holds the full untyped response for forward compatibility.
	Raw map[string]interface{} `json:"-"`
}

// ResultMessage represents a result message with cost and usage information.
type ResultMessage struct {
	Type              string                 `json:"type"`
	Subtype           string                 `json:"subtype"`
	DurationMs        int                    `json:"duration_ms"`
	DurationAPIMs     int                    `json:"duration_api_ms"`
	IsError           bool                   `json:"is_error"`
	NumTurns          int                    `json:"num_turns"`
	SessionID         string                 `json:"session_id"`
	TotalCostUSD      *float64               `json:"total_cost_usd,omitempty"`
	Usage             map[string]interface{} `json:"usage,omitempty"`
	Result            *string                `json:"result,omitempty"`
	Errors            []string               `json:"errors,omitempty"`
	StopReason        *string                `json:"stop_reason,omitempty"`
	PermissionDenials []PermissionDenial     `json:"permission_denials,omitempty"`
	ModelUsageMap     map[string]ModelUsage  `json:"modelUsage,omitempty"`
	UUID              string                 `json:"uuid,omitempty"`
}

// GetMessageType returns the type of the message.
func (m *ResultMessage) GetMessageType() string {
	return m.Type
}

// ShouldDisplayToUser returns false for result messages (metadata only).
func (m *ResultMessage) ShouldDisplayToUser() bool {
	return false
}

func (m *ResultMessage) isMessage() {}

// StreamEvent represents a stream event for partial message updates during streaming.
type StreamEvent struct {
	Type            string                 `json:"type"`
	UUID            string                 `json:"uuid"`
	SessionID       string                 `json:"session_id"`
	Event           map[string]interface{} `json:"event"` // The raw Anthropic API stream event
	ParentToolUseID *string                `json:"parent_tool_use_id,omitempty"`
}

// GetMessageType returns the type of the message.
func (m *StreamEvent) GetMessageType() string {
	return m.Type
}

// ShouldDisplayToUser returns false for stream events (internal only).
func (m *StreamEvent) ShouldDisplayToUser() bool {
	return false
}

func (m *StreamEvent) isMessage() {}

// ---------------------------------------------------------------------------
// Shared nested types
// ---------------------------------------------------------------------------

// RateLimitInfo contains rate limit status information.
type RateLimitInfo struct {
	Status      string   `json:"status"`
	ResetsAt    *float64 `json:"resetsAt,omitempty"`
	Utilization *float64 `json:"utilization,omitempty"`
}

// CompactMetadata contains context compaction metadata.
type CompactMetadata struct {
	Trigger   string `json:"trigger"`
	PreTokens int    `json:"pre_tokens"`
}

// TaskUsage contains task resource usage information.
type TaskUsage struct {
	TotalTokens int `json:"total_tokens"`
	ToolUses    int `json:"tool_uses"`
	DurationMs  int `json:"duration_ms"`
}

// PersistedFile represents a successfully persisted file.
type PersistedFile struct {
	Filename string `json:"filename"`
	FileID   string `json:"file_id"`
}

// FailedFile represents a file that failed to persist.
type FailedFile struct {
	Filename string `json:"filename"`
	Error    string `json:"error"`
}

// PermissionDenial represents a denied permission request.
type PermissionDenial struct {
	ToolName  string                 `json:"tool_name"`
	ToolUseID string                 `json:"tool_use_id"`
	ToolInput map[string]interface{} `json:"tool_input"`
}

// ModelUsage contains per-model token usage information.
type ModelUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// ---------------------------------------------------------------------------
// New top-level message types
// ---------------------------------------------------------------------------

// ToolProgressMessage is emitted periodically while a tool is executing.
type ToolProgressMessage struct {
	Type               string  `json:"type"`
	ToolUseID          string  `json:"tool_use_id"`
	ToolName           string  `json:"tool_name"`
	ParentToolUseID    *string `json:"parent_tool_use_id,omitempty"`
	ElapsedTimeSeconds float64 `json:"elapsed_time_seconds"`
	TaskID             *string `json:"task_id,omitempty"`
	UUID               string  `json:"uuid"`
	SessionID          string  `json:"session_id"`
}

func (m *ToolProgressMessage) GetMessageType() string    { return m.Type }
func (m *ToolProgressMessage) ShouldDisplayToUser() bool { return false }
func (m *ToolProgressMessage) isMessage()                {}

// AuthStatusMessage is emitted during authentication flows.
type AuthStatusMessage struct {
	Type             string   `json:"type"`
	IsAuthenticating bool     `json:"isAuthenticating"`
	Output           []string `json:"output"`
	Error            *string  `json:"error,omitempty"`
	UUID             string   `json:"uuid"`
	SessionID        string   `json:"session_id"`
}

func (m *AuthStatusMessage) GetMessageType() string    { return m.Type }
func (m *AuthStatusMessage) ShouldDisplayToUser() bool { return true }
func (m *AuthStatusMessage) isMessage()                {}

// ToolUseSummaryMessage contains a summary of tool usage.
type ToolUseSummaryMessage struct {
	Type                string   `json:"type"`
	Summary             string   `json:"summary"`
	PrecedingToolUseIDs []string `json:"preceding_tool_use_ids"`
	UUID                string   `json:"uuid"`
	SessionID           string   `json:"session_id"`
}

func (m *ToolUseSummaryMessage) GetMessageType() string    { return m.Type }
func (m *ToolUseSummaryMessage) ShouldDisplayToUser() bool { return false }
func (m *ToolUseSummaryMessage) isMessage()                {}

// RateLimitEvent is emitted when a rate limit is encountered.
type RateLimitEvent struct {
	Type          string        `json:"type"`
	RateLimitInfo RateLimitInfo `json:"rate_limit_info"`
	UUID          string        `json:"uuid"`
	SessionID     string        `json:"session_id"`
}

func (m *RateLimitEvent) GetMessageType() string    { return m.Type }
func (m *RateLimitEvent) ShouldDisplayToUser() bool { return true }
func (m *RateLimitEvent) isMessage()                {}

// PromptSuggestionMessage contains a predicted next user prompt.
type PromptSuggestionMessage struct {
	Type       string `json:"type"`
	Suggestion string `json:"suggestion"`
	UUID       string `json:"uuid"`
	SessionID  string `json:"session_id"`
}

func (m *PromptSuggestionMessage) GetMessageType() string    { return m.Type }
func (m *PromptSuggestionMessage) ShouldDisplayToUser() bool { return false }
func (m *PromptSuggestionMessage) isMessage()                {}

// ---------------------------------------------------------------------------
// Typed system message subtypes
// ---------------------------------------------------------------------------

// CompactBoundaryMessage indicates a context compaction boundary.
type CompactBoundaryMessage struct {
	Type            string          `json:"type"`
	Subtype         string          `json:"subtype"`
	CompactMetadata CompactMetadata `json:"compact_metadata"`
	UUID            string          `json:"uuid"`
	SessionID       string          `json:"session_id"`
}

func (m *CompactBoundaryMessage) GetMessageType() string    { return m.Type }
func (m *CompactBoundaryMessage) ShouldDisplayToUser() bool { return false }
func (m *CompactBoundaryMessage) isMessage()                {}

// StatusMessage indicates a system status change.
type StatusMessage struct {
	Type           string  `json:"type"`
	Subtype        string  `json:"subtype"`
	Status         *string `json:"status,omitempty"`
	PermissionMode *string `json:"permissionMode,omitempty"`
	UUID           string  `json:"uuid"`
	SessionID      string  `json:"session_id"`
}

func (m *StatusMessage) GetMessageType() string    { return m.Type }
func (m *StatusMessage) ShouldDisplayToUser() bool { return true }
func (m *StatusMessage) isMessage()                {}

// HookStartedMessage indicates a hook has started executing.
type HookStartedMessage struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	HookID    string `json:"hook_id"`
	HookName  string `json:"hook_name"`
	HookEvent string `json:"hook_event"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
}

func (m *HookStartedMessage) GetMessageType() string    { return m.Type }
func (m *HookStartedMessage) ShouldDisplayToUser() bool { return false }
func (m *HookStartedMessage) isMessage()                {}

// HookProgressMessage contains hook execution output.
type HookProgressMessage struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	HookID    string `json:"hook_id"`
	HookName  string `json:"hook_name"`
	HookEvent string `json:"hook_event"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	Output    string `json:"output"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
}

func (m *HookProgressMessage) GetMessageType() string    { return m.Type }
func (m *HookProgressMessage) ShouldDisplayToUser() bool { return false }
func (m *HookProgressMessage) isMessage()                {}

// HookResponseMessage indicates a hook has completed.
type HookResponseMessage struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	HookID    string `json:"hook_id"`
	HookName  string `json:"hook_name"`
	HookEvent string `json:"hook_event"`
	Output    string `json:"output"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ExitCode  *int   `json:"exit_code,omitempty"`
	Outcome   string `json:"outcome"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
}

func (m *HookResponseMessage) GetMessageType() string    { return m.Type }
func (m *HookResponseMessage) ShouldDisplayToUser() bool { return false }
func (m *HookResponseMessage) isMessage()                {}

// TaskNotificationMessage indicates a background task has completed.
type TaskNotificationMessage struct {
	Type       string     `json:"type"`
	Subtype    string     `json:"subtype"`
	TaskID     string     `json:"task_id"`
	ToolUseID  *string    `json:"tool_use_id,omitempty"`
	Status     string     `json:"status"`
	OutputFile string     `json:"output_file"`
	Summary    string     `json:"summary"`
	Usage      *TaskUsage `json:"usage,omitempty"`
	UUID       string     `json:"uuid"`
	SessionID  string     `json:"session_id"`
}

func (m *TaskNotificationMessage) GetMessageType() string    { return m.Type }
func (m *TaskNotificationMessage) ShouldDisplayToUser() bool { return true }
func (m *TaskNotificationMessage) isMessage()                {}

// TaskStartedMessage indicates a background task has started.
type TaskStartedMessage struct {
	Type        string  `json:"type"`
	Subtype     string  `json:"subtype"`
	TaskID      string  `json:"task_id"`
	ToolUseID   *string `json:"tool_use_id,omitempty"`
	Description string  `json:"description"`
	TaskType    *string `json:"task_type,omitempty"`
	UUID        string  `json:"uuid"`
	SessionID   string  `json:"session_id"`
}

func (m *TaskStartedMessage) GetMessageType() string    { return m.Type }
func (m *TaskStartedMessage) ShouldDisplayToUser() bool { return true }
func (m *TaskStartedMessage) isMessage()                {}

// TaskProgressMessage contains progress information for a background task.
type TaskProgressMessage struct {
	Type         string    `json:"type"`
	Subtype      string    `json:"subtype"`
	TaskID       string    `json:"task_id"`
	ToolUseID    *string   `json:"tool_use_id,omitempty"`
	Description  string    `json:"description"`
	Usage        TaskUsage `json:"usage"`
	LastToolName *string   `json:"last_tool_name,omitempty"`
	UUID         string    `json:"uuid"`
	SessionID    string    `json:"session_id"`
}

func (m *TaskProgressMessage) GetMessageType() string    { return m.Type }
func (m *TaskProgressMessage) ShouldDisplayToUser() bool { return false }
func (m *TaskProgressMessage) isMessage()                {}

// FilesPersistedEvent indicates files have been persisted to a checkpoint.
type FilesPersistedEvent struct {
	Type        string          `json:"type"`
	Subtype     string          `json:"subtype"`
	Files       []PersistedFile `json:"files"`
	Failed      []FailedFile    `json:"failed"`
	ProcessedAt string          `json:"processed_at"`
	UUID        string          `json:"uuid"`
	SessionID   string          `json:"session_id"`
}

func (m *FilesPersistedEvent) GetMessageType() string    { return m.Type }
func (m *FilesPersistedEvent) ShouldDisplayToUser() bool { return false }
func (m *FilesPersistedEvent) isMessage()                {}

// ---------------------------------------------------------------------------
// Forward compatibility
// ---------------------------------------------------------------------------

// UnknownMessage represents an unrecognized message type.
// It preserves the raw JSON for forward compatibility with future CLI versions.
type UnknownMessage struct {
	Type    string          `json:"type"`
	RawJSON json.RawMessage `json:"-"`
}

func (m *UnknownMessage) GetMessageType() string    { return m.Type }
func (m *UnknownMessage) ShouldDisplayToUser() bool { return false }
func (m *UnknownMessage) isMessage()                {}

// ---------------------------------------------------------------------------
// UnmarshalMessage — central message dispatch
// ---------------------------------------------------------------------------

// truncateRaw truncates raw JSON for safe inclusion in error messages.
// Prevents sensitive subprocess output from leaking into error structs.
func truncateRaw(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// UnmarshalMessage unmarshals a JSON message into the appropriate message type.
func UnmarshalMessage(data []byte) (Message, error) {
	var typeCheck struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, NewJSONDecodeErrorWithCause("failed to determine message type", truncateRaw(string(data), 200), err)
	}

	if typeCheck.Type == "" {
		return nil, NewMessageParseError("missing or empty type field")
	}

	switch typeCheck.Type {
	case "user":
		var msg UserMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal user message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case "assistant":
		var msg AssistantMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal assistant message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case "system":
		return unmarshalSystemMessage(data)
	case "control_request", "control_response":
		var msg SystemMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal system message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case "result":
		var msg ResultMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal result message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case "stream_event":
		var msg StreamEvent
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal stream event", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case "tool_progress":
		var msg ToolProgressMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal tool progress message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case "auth_status":
		var msg AuthStatusMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal auth status message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case "tool_use_summary":
		var msg ToolUseSummaryMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal tool use summary message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case "rate_limit_event":
		var msg RateLimitEvent
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal rate limit event", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case "prompt_suggestion":
		var msg PromptSuggestionMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal prompt suggestion message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	default:
		return &UnknownMessage{
			Type:    typeCheck.Type,
			RawJSON: append(json.RawMessage(nil), data...),
		}, nil
	}
}

// unmarshalSystemMessage handles system message subtype routing.
func unmarshalSystemMessage(data []byte) (Message, error) {
	var subtypeCheck struct {
		Subtype string `json:"subtype"`
	}
	if err := json.Unmarshal(data, &subtypeCheck); err != nil {
		return nil, NewJSONDecodeErrorWithCause("failed to extract system subtype", truncateRaw(string(data), 200), err)
	}

	switch subtypeCheck.Subtype {
	case SystemSubtypeCompactBoundary:
		var msg CompactBoundaryMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal compact boundary message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case SystemSubtypeStatus:
		var msg StatusMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal status message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case SystemSubtypeHookStarted:
		var msg HookStartedMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal hook started message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case SystemSubtypeHookProgress:
		var msg HookProgressMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal hook progress message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case SystemSubtypeHookResponse:
		var msg HookResponseMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal hook response message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case SystemSubtypeTaskNotification:
		var msg TaskNotificationMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal task notification message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case SystemSubtypeTaskStarted:
		var msg TaskStartedMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal task started message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case SystemSubtypeTaskProgress:
		var msg TaskProgressMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal task progress message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	case SystemSubtypeFilesPersisted:
		var msg FilesPersistedEvent
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal files persisted event", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	default:
		var msg SystemMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal system message", truncateRaw(string(data), 200), err)
		}
		return &msg, nil
	}
}
