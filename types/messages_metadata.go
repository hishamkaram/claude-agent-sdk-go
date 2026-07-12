package types

// ResultMessage subtype constants
const (
	ResultSubtypeSuccess                         = "success"
	ResultSubtypeErrorMaxTurns                   = "error_max_turns"
	ResultSubtypeErrorMaxBudget                  = "error_max_budget_usd"
	ResultSubtypeErrorDuringExecution            = "error_during_execution"
	ResultSubtypeErrorMaxStructuredOutputRetries = "error_max_structured_output_retries"
)

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
	// Value is the model identifier reported by the connected Claude CLI.
	Value string `json:"value"`
	// DisplayName is the human-readable model name reported by the CLI.
	DisplayName string `json:"displayName"`
	// Description is an optional short description of the model.
	Description string `json:"description,omitempty"`
	// ResolvedModel is the concrete provider model selected by an alias.
	ResolvedModel string `json:"resolvedModel,omitempty"`
	// SupportsEffort reports whether the model accepts an effort setting.
	SupportsEffort bool `json:"supportsEffort,omitempty"`
	// SupportedEffortLevels is the provider-reported effort set for this model.
	SupportedEffortLevels []EffortLevel `json:"supportedEffortLevels,omitempty"`
	// SupportsAdaptiveThinking reports adaptive-thinking support.
	SupportsAdaptiveThinking bool `json:"supportsAdaptiveThinking,omitempty"`
	// SupportsFastMode reports fast-mode support.
	SupportsFastMode bool `json:"supportsFastMode,omitempty"`
	// SupportsAutoMode reports auto-mode support.
	SupportsAutoMode bool `json:"supportsAutoMode,omitempty"`
	// Disabled reports that the provider exposed the model but made it unavailable.
	Disabled bool `json:"disabled,omitempty"`
	// Raw preserves the full CLI model row for forward compatibility.
	Raw map[string]interface{} `json:"-"`
}

// AgentInfo describes a supported agent type from the initialization response.
type AgentInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Model       string `json:"model,omitempty"`
}

// InitializeResult holds the parsed response from session initialization.
// It contains available commands, models, and session metadata returned
// by the Claude CLI control protocol initialize response.
type InitializeResult struct {
	// Commands is the list of available slash commands/skills.
	Commands []SlashCommand `json:"commands,omitempty"`
	// Models is the list of models available in this session.
	Models []ModelInfo `json:"models,omitempty"`
	// Agents is the list of agent types available in this session.
	Agents []AgentInfo `json:"agents,omitempty"`
	// Raw holds the full untyped response for forward compatibility.
	Raw map[string]interface{} `json:"-"`
}

// RewindFilesResult represents the result of a RewindFiles operation.
type RewindFilesResult struct {
	CanRewind    bool     `json:"canRewind"`
	Error        string   `json:"error,omitempty"`
	FilesChanged []string `json:"filesChanged,omitempty"`
	Insertions   int      `json:"insertions,omitempty"`
	Deletions    int      `json:"deletions,omitempty"`
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

// ToolUseResult describes top-level metadata emitted with CLI tool_result user
// messages. Workflow launches populate the task and run identifiers here.
type ToolUseResult struct {
	Status        string `json:"status,omitempty"`
	TaskID        string `json:"taskId,omitempty"`
	TaskType      string `json:"taskType,omitempty"`
	WorkflowName  string `json:"workflowName,omitempty"`
	RunID         string `json:"runId,omitempty"`
	Summary       string `json:"summary,omitempty"`
	TranscriptDir string `json:"transcriptDir,omitempty"`
	ScriptPath    string `json:"scriptPath,omitempty"`
}

// WorkflowProgressEntry describes one workflow_progress item from a
// task_progress system frame. PromptPreview is parsed for completeness but
// downstream AgentD payloads must not forward it to clients.
type WorkflowProgressEntry struct {
	Type           string `json:"type"`
	Index          int    `json:"index,omitempty"`
	Title          string `json:"title,omitempty"`
	Label          string `json:"label,omitempty"`
	AgentID        string `json:"agentId,omitempty"`
	PhaseIndex     int    `json:"phaseIndex,omitempty"`
	PhaseTitle     string `json:"phaseTitle,omitempty"`
	Model          string `json:"model,omitempty"`
	State          string `json:"state,omitempty"`
	QueuedAt       int64  `json:"queuedAt,omitempty"`
	StartedAt      int64  `json:"startedAt,omitempty"`
	LastProgressAt int64  `json:"lastProgressAt,omitempty"`
	Attempt        int    `json:"attempt,omitempty"`
	Tokens         int    `json:"tokens,omitempty"`
	ToolCalls      int    `json:"toolCalls,omitempty"`
	DurationMs     int    `json:"durationMs,omitempty"`
	ResultPreview  string `json:"resultPreview,omitempty"`
	PromptPreview  string `json:"promptPreview,omitempty"`
}

// TaskUpdatedPatch contains the sparse patch object in task_updated frames.
type TaskUpdatedPatch struct {
	Status    string `json:"status,omitempty"`
	StartTime int64  `json:"start_time,omitempty"`
	EndTime   int64  `json:"end_time,omitempty"`
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
