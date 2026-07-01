package types

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
	SystemSubtypeCompactBoundary     = "compact_boundary"
	SystemSubtypeStatus              = "status"
	SystemSubtypeHookStarted         = "hook_started"
	SystemSubtypeHookProgress        = "hook_progress"
	SystemSubtypeHookResponse        = "hook_response"
	SystemSubtypeTaskNotification    = "task_notification"
	SystemSubtypeTaskStarted         = "task_started"
	SystemSubtypeTaskProgress        = "task_progress"
	SystemSubtypeTaskUpdated         = "task_updated"
	SystemSubtypeFilesPersisted      = "files_persisted"
	SystemSubtypeLocalCommandOutput  = "local_command_output"
	SystemSubtypeElicitationComplete = "elicitation_complete"
)

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
	Type         string  `json:"type"`
	Subtype      string  `json:"subtype"`
	TaskID       string  `json:"task_id"`
	ToolUseID    *string `json:"tool_use_id,omitempty"`
	Description  string  `json:"description"`
	TaskType     *string `json:"task_type,omitempty"`
	WorkflowName string  `json:"workflow_name,omitempty"`
	Prompt       string  `json:"prompt,omitempty"`
	UUID         string  `json:"uuid"`
	SessionID    string  `json:"session_id"`
}

func (m *TaskStartedMessage) GetMessageType() string    { return m.Type }
func (m *TaskStartedMessage) ShouldDisplayToUser() bool { return true }
func (m *TaskStartedMessage) isMessage()                {}

// TaskProgressMessage contains progress information for a background task.
type TaskProgressMessage struct {
	Type             string                  `json:"type"`
	Subtype          string                  `json:"subtype"`
	TaskID           string                  `json:"task_id"`
	ToolUseID        *string                 `json:"tool_use_id,omitempty"`
	Description      string                  `json:"description"`
	Summary          string                  `json:"summary,omitempty"`
	WorkflowProgress []WorkflowProgressEntry `json:"workflow_progress,omitempty"`
	Usage            TaskUsage               `json:"usage"`
	LastToolName     *string                 `json:"last_tool_name,omitempty"`
	UUID             string                  `json:"uuid"`
	SessionID        string                  `json:"session_id"`
}

func (m *TaskProgressMessage) GetMessageType() string    { return m.Type }
func (m *TaskProgressMessage) ShouldDisplayToUser() bool { return false }
func (m *TaskProgressMessage) isMessage()                {}

// TaskUpdatedMessage contains status updates for a background task patch.
type TaskUpdatedMessage struct {
	Type      string           `json:"type"`
	Subtype   string           `json:"subtype"`
	TaskID    string           `json:"task_id"`
	ToolUseID *string          `json:"tool_use_id,omitempty"`
	Patch     TaskUpdatedPatch `json:"patch"`
	UUID      string           `json:"uuid"`
	SessionID string           `json:"session_id"`
}

func (m *TaskUpdatedMessage) GetMessageType() string    { return m.Type }
func (m *TaskUpdatedMessage) ShouldDisplayToUser() bool { return false }
func (m *TaskUpdatedMessage) isMessage()                {}

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
