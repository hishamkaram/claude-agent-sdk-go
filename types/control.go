package types

import "encoding/json"

// PermissionMode represents the permission mode for Claude.
type PermissionMode string

const (
	PermissionModeDefault           PermissionMode = "default"
	PermissionModeAcceptEdits       PermissionMode = "acceptEdits"
	PermissionModePlan              PermissionMode = "plan"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
	PermissionModeDontAsk           PermissionMode = "dontAsk"
)

// PermissionBehavior represents the behavior for a permission rule.
type PermissionBehavior string

const (
	PermissionBehaviorAllow PermissionBehavior = "allow"
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

// PermissionUpdateDestination represents where permission updates should be saved.
type PermissionUpdateDestination string

const (
	DestinationUserSettings    PermissionUpdateDestination = "userSettings"
	DestinationProjectSettings PermissionUpdateDestination = "projectSettings"
	DestinationLocalSettings   PermissionUpdateDestination = "localSettings"
	DestinationSession         PermissionUpdateDestination = "session"
)

// PermissionRuleValue represents a permission rule.
type PermissionRuleValue struct {
	ToolName    string  `json:"toolName"`
	RuleContent *string `json:"ruleContent,omitempty"`
}

// PermissionUpdate represents a permission update configuration.
type PermissionUpdate struct {
	Type        string                       `json:"type"` // addRules, replaceRules, removeRules, setMode, addDirectories, removeDirectories
	Rules       []PermissionRuleValue        `json:"rules,omitempty"`
	Behavior    *PermissionBehavior          `json:"behavior,omitempty"`
	Mode        *PermissionMode              `json:"mode,omitempty"`
	Directories []string                     `json:"directories,omitempty"`
	Destination *PermissionUpdateDestination `json:"destination,omitempty"`
}

// PermissionResultAllow represents an allow permission result.
type PermissionResultAllow struct {
	Behavior           string                  `json:"behavior"` // "allow"
	UpdatedInput       *map[string]interface{} `json:"updated_input,omitempty"`
	UpdatedPermissions []PermissionUpdate      `json:"updated_permissions,omitempty"`
}

// PermissionResultDeny represents a deny permission result.
type PermissionResultDeny struct {
	Behavior  string `json:"behavior"` // "deny"
	Message   string `json:"message,omitempty"`
	Interrupt bool   `json:"interrupt,omitempty"`
}

// ToolPermissionContext provides context for tool permission callbacks.
type ToolPermissionContext struct {
	Signal      interface{}        `json:"signal,omitempty"` // Future: abort signal support
	Suggestions []PermissionUpdate `json:"suggestions,omitempty"`
}

// HookEvent represents a hook event type.
type HookEvent string

const (
	HookEventPreToolUse       HookEvent = "PreToolUse"
	HookEventPostToolUse      HookEvent = "PostToolUse"
	HookEventUserPromptSubmit HookEvent = "UserPromptSubmit"
	HookEventStop             HookEvent = "Stop"
	HookEventSubagentStop     HookEvent = "SubagentStop"
	HookEventPreCompact       HookEvent = "PreCompact"

	// Phase C: 17 new hook events for TS SDK parity
	HookEventPostToolUseFailure HookEvent = "PostToolUseFailure"
	HookEventNotification       HookEvent = "Notification"
	HookEventSessionStart       HookEvent = "SessionStart"
	HookEventSessionEnd         HookEvent = "SessionEnd"
	HookEventStopFailure        HookEvent = "StopFailure"
	HookEventSubagentStart      HookEvent = "SubagentStart"
	HookEventPostCompact        HookEvent = "PostCompact"
	HookEventPermissionRequest  HookEvent = "PermissionRequest"
	HookEventSetup              HookEvent = "Setup"
	HookEventTeammateIdle       HookEvent = "TeammateIdle"
	HookEventTaskCompleted      HookEvent = "TaskCompleted"
	HookEventElicitation        HookEvent = "Elicitation"
	HookEventElicitationResult  HookEvent = "ElicitationResult"
	HookEventConfigChange       HookEvent = "ConfigChange"
	HookEventWorktreeCreate     HookEvent = "WorktreeCreate"
	HookEventWorktreeRemove     HookEvent = "WorktreeRemove"
	HookEventInstructionsLoaded HookEvent = "InstructionsLoaded"
)

// BaseHookInput contains common fields for all hook inputs.
type BaseHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	CWD            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
}

// PreToolUseHookInput represents input for PreToolUse hook events.
type PreToolUseHookInput struct {
	BaseHookInput
	HookEventName string                 `json:"hook_event_name"` // "PreToolUse"
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
}

// PostToolUseHookInput represents input for PostToolUse hook events.
type PostToolUseHookInput struct {
	BaseHookInput
	HookEventName string                 `json:"hook_event_name"` // "PostToolUse"
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	ToolResponse  interface{}            `json:"tool_response"`
}

// UserPromptSubmitHookInput represents input for UserPromptSubmit hook events.
type UserPromptSubmitHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "UserPromptSubmit"
	Prompt        string `json:"prompt"`
}

// StopHookInput represents input for Stop hook events.
type StopHookInput struct {
	BaseHookInput
	HookEventName  string `json:"hook_event_name"` // "Stop"
	StopHookActive bool   `json:"stop_hook_active"`
}

// SubagentStopHookInput represents input for SubagentStop hook events.
type SubagentStopHookInput struct {
	BaseHookInput
	HookEventName  string `json:"hook_event_name"` // "SubagentStop"
	StopHookActive bool   `json:"stop_hook_active"`
}

// PreCompactHookInput represents input for PreCompact hook events.
type PreCompactHookInput struct {
	BaseHookInput
	HookEventName      string  `json:"hook_event_name"` // "PreCompact"
	Trigger            string  `json:"trigger"`         // "manual" or "auto"
	CustomInstructions *string `json:"custom_instructions,omitempty"`
}

// PostToolUseFailureHookInput represents input for PostToolUseFailure hook events.
type PostToolUseFailureHookInput struct {
	BaseHookInput
	HookEventName string                 `json:"hook_event_name"` // "PostToolUseFailure"
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	Error         string                 `json:"error"`
}

// NotificationHookInput represents input for Notification hook events.
type NotificationHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "Notification"
	Message       string `json:"message"`
	Level         string `json:"level,omitempty"` // "info", "warn", "error"
}

// SessionStartHookInput represents input for SessionStart hook events.
type SessionStartHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "SessionStart"
}

// SessionEndHookInput represents input for SessionEnd hook events.
type SessionEndHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "SessionEnd"
	Reason        string `json:"reason,omitempty"`
}

// StopFailureHookInput represents input for StopFailure hook events.
type StopFailureHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "StopFailure"
	Error         string `json:"error"`
}

// SubagentStartHookInput represents input for SubagentStart hook events.
type SubagentStartHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "SubagentStart"
	AgentName     string `json:"agent_name"`
	AgentType     string `json:"agent_type,omitempty"`
}

// PostCompactHookInput represents input for PostCompact hook events.
type PostCompactHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "PostCompact"
	Trigger       string `json:"trigger"`         // "manual" or "auto"
}

// PermissionRequestHookInput represents input for PermissionRequest hook events.
type PermissionRequestHookInput struct {
	BaseHookInput
	HookEventName string                 `json:"hook_event_name"` // "PermissionRequest"
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
}

// SetupHookInput represents input for Setup hook events.
type SetupHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "Setup"
}

// TeammateIdleHookInput represents input for TeammateIdle hook events.
type TeammateIdleHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "TeammateIdle"
	AgentName     string `json:"agent_name"`
}

// TaskCompletedHookInput represents input for TaskCompleted hook events.
type TaskCompletedHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "TaskCompleted"
	TaskID        string `json:"task_id"`
}

// ElicitationHookInput represents input for Elicitation hook events.
type ElicitationHookInput struct {
	BaseHookInput
	HookEventName string                   `json:"hook_event_name"` // "Elicitation"
	Questions     []map[string]interface{} `json:"questions"`
}

// ElicitationResultHookInput represents input for ElicitationResult hook events.
type ElicitationResultHookInput struct {
	BaseHookInput
	HookEventName string                 `json:"hook_event_name"` // "ElicitationResult"
	Answers       map[string]interface{} `json:"answers"`
}

// ConfigChangeHookInput represents input for ConfigChange hook events.
type ConfigChangeHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "ConfigChange"
	ConfigPath    string `json:"config_path"`
}

// WorktreeCreateHookInput represents input for WorktreeCreate hook events.
type WorktreeCreateHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "WorktreeCreate"
	WorktreePath  string `json:"worktree_path"`
	BranchName    string `json:"branch_name,omitempty"`
}

// WorktreeRemoveHookInput represents input for WorktreeRemove hook events.
type WorktreeRemoveHookInput struct {
	BaseHookInput
	HookEventName string `json:"hook_event_name"` // "WorktreeRemove"
	WorktreePath  string `json:"worktree_path"`
}

// InstructionsLoadedHookInput represents input for InstructionsLoaded hook events.
type InstructionsLoadedHookInput struct {
	BaseHookInput
	HookEventName string   `json:"hook_event_name"` // "InstructionsLoaded"
	Sources       []string `json:"sources,omitempty"`
}

// --- Hook Output Types for events with Has Output Type = Yes ---

// PostToolUseFailureHookSpecificOutput represents hook-specific output for PostToolUseFailure events.
type PostToolUseFailureHookSpecificOutput struct {
	HookEventName     string  `json:"hookEventName"` // "PostToolUseFailure"
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// GetHookEventName returns the hook event name.
func (h *PostToolUseFailureHookSpecificOutput) GetHookEventName() string {
	return h.HookEventName
}

// NotificationHookSpecificOutput represents hook-specific output for Notification events.
type NotificationHookSpecificOutput struct {
	HookEventName     string  `json:"hookEventName"` // "Notification"
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// GetHookEventName returns the hook event name.
func (h *NotificationHookSpecificOutput) GetHookEventName() string {
	return h.HookEventName
}

// SessionStartHookSpecificOutput represents hook-specific output for SessionStart events.
type SessionStartHookSpecificOutput struct {
	HookEventName     string  `json:"hookEventName"` // "SessionStart"
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// GetHookEventName returns the hook event name.
func (h *SessionStartHookSpecificOutput) GetHookEventName() string {
	return h.HookEventName
}

// SubagentStartHookSpecificOutput represents hook-specific output for SubagentStart events.
type SubagentStartHookSpecificOutput struct {
	HookEventName     string  `json:"hookEventName"` // "SubagentStart"
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// GetHookEventName returns the hook event name.
func (h *SubagentStartHookSpecificOutput) GetHookEventName() string {
	return h.HookEventName
}

// PermissionRequestHookSpecificOutput represents hook-specific output for PermissionRequest events.
type PermissionRequestHookSpecificOutput struct {
	HookEventName            string  `json:"hookEventName"` // "PermissionRequest"
	PermissionDecision       *string `json:"permissionDecision,omitempty"`
	PermissionDecisionReason *string `json:"permissionDecisionReason,omitempty"`
}

// GetHookEventName returns the hook event name.
func (h *PermissionRequestHookSpecificOutput) GetHookEventName() string {
	return h.HookEventName
}

// SetupHookSpecificOutput represents hook-specific output for Setup events.
type SetupHookSpecificOutput struct {
	HookEventName     string  `json:"hookEventName"` // "Setup"
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// GetHookEventName returns the hook event name.
func (h *SetupHookSpecificOutput) GetHookEventName() string {
	return h.HookEventName
}

// ElicitationHookSpecificOutput represents hook-specific output for Elicitation events.
type ElicitationHookSpecificOutput struct {
	HookEventName string                 `json:"hookEventName"` // "Elicitation"
	Answers       map[string]interface{} `json:"answers,omitempty"`
}

// GetHookEventName returns the hook event name.
func (h *ElicitationHookSpecificOutput) GetHookEventName() string {
	return h.HookEventName
}

// ElicitationResultHookSpecificOutput represents hook-specific output for ElicitationResult events.
type ElicitationResultHookSpecificOutput struct {
	HookEventName     string  `json:"hookEventName"` // "ElicitationResult"
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// GetHookEventName returns the hook event name.
func (h *ElicitationResultHookSpecificOutput) GetHookEventName() string {
	return h.HookEventName
}

// HookSpecificOutput is an interface for all hook-specific outputs.
type HookSpecificOutput interface {
	GetHookEventName() string
}

// PreToolUseHookSpecificOutput represents hook-specific output for PreToolUse events.
type PreToolUseHookSpecificOutput struct {
	HookEventName            string                  `json:"hookEventName"`                // "PreToolUse"
	PermissionDecision       *string                 `json:"permissionDecision,omitempty"` // "allow", "deny", "ask"
	PermissionDecisionReason *string                 `json:"permissionDecisionReason,omitempty"`
	UpdatedInput             *map[string]interface{} `json:"updatedInput,omitempty"`
}

// GetHookEventName returns the hook event name.
func (h *PreToolUseHookSpecificOutput) GetHookEventName() string {
	return h.HookEventName
}

// PostToolUseHookSpecificOutput represents hook-specific output for PostToolUse events.
type PostToolUseHookSpecificOutput struct {
	HookEventName     string  `json:"hookEventName"` // "PostToolUse"
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// GetHookEventName returns the hook event name.
func (h *PostToolUseHookSpecificOutput) GetHookEventName() string {
	return h.HookEventName
}

// UserPromptSubmitHookSpecificOutput represents hook-specific output for UserPromptSubmit events.
type UserPromptSubmitHookSpecificOutput struct {
	HookEventName     string  `json:"hookEventName"` // "UserPromptSubmit"
	AdditionalContext *string `json:"additionalContext,omitempty"`
}

// GetHookEventName returns the hook event name.
func (h *UserPromptSubmitHookSpecificOutput) GetHookEventName() string {
	return h.HookEventName
}

// AsyncHookJSONOutput represents async hook output that defers hook execution.
type AsyncHookJSONOutput struct {
	Async        bool `json:"async"`
	AsyncTimeout *int `json:"asyncTimeout,omitempty"`
}

// SyncHookJSONOutput represents synchronous hook output with control and decision fields.
type SyncHookJSONOutput struct {
	// Common control fields
	Continue       *bool   `json:"continue,omitempty"`
	SuppressOutput *bool   `json:"suppressOutput,omitempty"`
	StopReason     *string `json:"stopReason,omitempty"`

	// Decision fields
	Decision      *string `json:"decision,omitempty"` // "block"
	SystemMessage *string `json:"systemMessage,omitempty"`
	Reason        *string `json:"reason,omitempty"`

	// Hook-specific outputs
	HookSpecificOutput interface{} `json:"hookSpecificOutput,omitempty"`
}

// HookContext provides context information for hook callbacks.
type HookContext struct {
	Signal interface{} `json:"signal,omitempty"` // Future: abort signal support
}

// SDKControlInterruptRequest represents an interrupt request.
type SDKControlInterruptRequest struct {
	Subtype string `json:"subtype"` // "interrupt"
}

// SDKControlPermissionRequest represents a permission request for tool use.
type SDKControlPermissionRequest struct {
	Subtype               string                 `json:"subtype"` // "can_use_tool"
	ToolName              string                 `json:"tool_name"`
	Input                 map[string]interface{} `json:"input"`
	PermissionSuggestions []PermissionUpdate     `json:"permission_suggestions,omitempty"`
	BlockedPath           *string                `json:"blocked_path,omitempty"`
}

// SDKControlInitializeRequest represents an initialization request.
type SDKControlInitializeRequest struct {
	Subtype string                 `json:"subtype"` // "initialize"
	Hooks   map[string]interface{} `json:"hooks,omitempty"`
}

// SDKControlSetPermissionModeRequest represents a request to set permission mode.
type SDKControlSetPermissionModeRequest struct {
	Subtype string `json:"subtype"` // "set_permission_mode"
	Mode    string `json:"mode"`
}

// SDKHookCallbackRequest represents a hook callback request.
type SDKHookCallbackRequest struct {
	Subtype    string      `json:"subtype"` // "hook_callback"
	CallbackID string      `json:"callback_id"`
	Input      interface{} `json:"input"`
	ToolUseID  *string     `json:"tool_use_id,omitempty"`
}

// SDKControlMcpMessageRequest represents an MCP message request.
type SDKControlMcpMessageRequest struct {
	Subtype    string      `json:"subtype"` // "mcp_message"
	ServerName string      `json:"server_name"`
	Message    interface{} `json:"message"`
}

// SDKControlStopTaskRequest represents a request to stop a background task.
type SDKControlStopTaskRequest struct {
	Subtype string `json:"subtype"` // "stop_task"
	TaskID  string `json:"task_id"`
}

// SDKControlRewindFilesRequest represents a request to rewind files to a checkpoint.
type SDKControlRewindFilesRequest struct {
	Subtype       string `json:"subtype"` // "rewind_files"
	UserMessageID string `json:"user_message_id"`
	DryRun        bool   `json:"dry_run"`
}

// SDKControlMcpStatusRequest represents a request for MCP server status.
type SDKControlMcpStatusRequest struct {
	Subtype string `json:"subtype"` // "mcp_status"
}

// SDKControlMcpReconnectRequest represents a request to reconnect an MCP server.
type SDKControlMcpReconnectRequest struct {
	Subtype    string `json:"subtype"` // "mcp_reconnect"
	ServerName string `json:"serverName"`
}

// SDKControlMcpToggleRequest represents a request to toggle an MCP server.
type SDKControlMcpToggleRequest struct {
	Subtype    string `json:"subtype"` // "mcp_toggle"
	ServerName string `json:"serverName"`
	Enabled    bool   `json:"enabled"`
}

// SDKControlMcpSetServersRequest represents a request to set MCP servers.
type SDKControlMcpSetServersRequest struct {
	Subtype string                 `json:"subtype"` // "mcp_set_servers"
	Servers map[string]interface{} `json:"servers"`
}

// SDKControlRequest represents a control request from the CLI.
type SDKControlRequest struct {
	Type      string          `json:"type"` // "control_request"
	RequestID string          `json:"request_id"`
	Request   json.RawMessage `json:"request"` // Union type - needs custom unmarshaling
}

// ControlResponse represents a successful control response.
type ControlResponse struct {
	Subtype   string                 `json:"subtype"` // "success"
	RequestID string                 `json:"request_id"`
	Response  map[string]interface{} `json:"response,omitempty"`
}

// ControlErrorResponse represents an error control response.
type ControlErrorResponse struct {
	Subtype   string `json:"subtype"` // "error"
	RequestID string `json:"request_id"`
	Error     string `json:"error"`
}

// SDKControlResponse represents a control response to the CLI.
type SDKControlResponse struct {
	Type     string          `json:"type"`     // "control_response"
	Response json.RawMessage `json:"response"` // Union type - needs custom unmarshaling
}

// MCPServer represents an MCP server interface for handling MCP messages.
// This is a minimal interface for routing MCP JSONRPC messages.
// Concrete implementations can use the MCP SDK or custom logic.
type MCPServer interface {
	// HandleMessage handles an incoming JSONRPC message and returns the response.
	HandleMessage(message map[string]interface{}) (map[string]interface{}, error)

	// Name returns the server name.
	Name() string

	// Version returns the server version.
	Version() string
}
