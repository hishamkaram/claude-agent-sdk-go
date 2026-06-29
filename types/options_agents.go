package types

import "encoding/json"

// SubagentExecutionMode represents how a subagent executes relative to others.
type SubagentExecutionMode string

const (
	SubagentExecutionModeSequential SubagentExecutionMode = "sequential"
	SubagentExecutionModeParallel   SubagentExecutionMode = "parallel"
	SubagentExecutionModeAuto       SubagentExecutionMode = "auto"
)

// MultiInvocationMode represents how multiple subagent invocations are handled.
type MultiInvocationMode string

const (
	MultiInvocationModeSequential MultiInvocationMode = "sequential"
	MultiInvocationModeParallel   MultiInvocationMode = "parallel"
	MultiInvocationModeError      MultiInvocationMode = "error"
)

// SubagentErrorHandling represents how errors in subagent execution are handled.
type SubagentErrorHandling string

const (
	SubagentErrorHandlingFailFast SubagentErrorHandling = "fail_fast"
	SubagentErrorHandlingContinue SubagentErrorHandling = "continue"
)

// SubagentExecutionConfig represents global configuration for subagent execution.
type SubagentExecutionConfig struct {
	// MultiInvocation specifies how multiple subagent invocations are handled
	MultiInvocation MultiInvocationMode `json:"multi_invocation,omitempty"`
	// MaxConcurrent specifies the maximum number of concurrent subagent executions (default: 3)
	MaxConcurrent int `json:"max_concurrent,omitempty"`
	// ErrorHandling specifies how errors in subagent execution are handled
	ErrorHandling SubagentErrorHandling `json:"error_handling,omitempty"`
}

// NewSubagentExecutionConfig creates a new SubagentExecutionConfig with sensible defaults.
// Default values:
// - MultiInvocation: sequential
// - MaxConcurrent: 3
// - ErrorHandling: continue
func NewSubagentExecutionConfig() *SubagentExecutionConfig {
	return &SubagentExecutionConfig{
		MultiInvocation: MultiInvocationModeSequential,
		MaxConcurrent:   3,
		ErrorHandling:   SubagentErrorHandlingContinue,
	}
}

// AgentIsolation controls whether a subagent runs in the current worktree or
// an isolated temporary worktree.
type AgentIsolation string

const (
	AgentIsolationWorktree AgentIsolation = "worktree"
)

// AgentColor controls the display color for a subagent in Claude Code UI.
type AgentColor string

const (
	AgentColorRed    AgentColor = "red"
	AgentColorBlue   AgentColor = "blue"
	AgentColorGreen  AgentColor = "green"
	AgentColorYellow AgentColor = "yellow"
	AgentColorPurple AgentColor = "purple"
	AgentColorOrange AgentColor = "orange"
	AgentColorPink   AgentColor = "pink"
	AgentColorCyan   AgentColor = "cyan"
)

// AgentHookHandler is a raw Claude Code hook handler entry for subagent
// frontmatter hooks. It supports command, HTTP, MCP tool, prompt, and agent
// hook handler fields.
type AgentHookHandler map[string]interface{}

// AgentHookMatcher represents a subagent-scoped hook matcher group.
type AgentHookMatcher struct {
	Matcher *string            `json:"matcher,omitempty"`
	Hooks   []AgentHookHandler `json:"hooks,omitempty"`
}

// AgentDefinition represents a custom agent definition.
type AgentDefinition struct {
	Description            string                           `json:"description"`
	Prompt                 string                           `json:"prompt"`
	Tools                  []string                         `json:"tools,omitempty"`
	DisallowedTools        []string                         `json:"disallowedTools,omitempty"`                     // Tools explicitly disallowed for this agent
	Model                  *string                          `json:"model,omitempty"`                               // "sonnet", "opus", "haiku", "inherit", or full model ID
	MaxTurns               *int                             `json:"maxTurns,omitempty"`                            // Maximum conversation turns for this agent
	McpServers             []interface{}                    `json:"mcpServers,omitempty"`                          // MCP server specs (string refs or inline configs)
	Hooks                  map[HookEvent][]AgentHookMatcher `json:"hooks,omitempty"`                               // Subagent-scoped lifecycle hooks
	Skills                 []string                         `json:"skills,omitempty"`                              // Skill names to preload
	InitialPrompt          *string                          `json:"initialPrompt,omitempty"`                       // First user turn when the agent runs as the main session agent
	Memory                 *SettingSource                   `json:"memory,omitempty"`                              // Memory source: user, project, or local
	Background             *bool                            `json:"background,omitempty"`                          // Run as a non-blocking background task when invoked
	Effort                 *EffortLevel                     `json:"effort,omitempty"`                              // Per-agent reasoning effort; numeric effort is not modeled by the Go enum
	PermissionMode         *PermissionMode                  `json:"permissionMode,omitempty"`                      // Permission mode for tool execution in this agent
	Isolation              *AgentIsolation                  `json:"isolation,omitempty"`                           // Set to worktree for an isolated temporary worktree
	Color                  *AgentColor                      `json:"color,omitempty"`                               // Display color in Claude Code UI
	CriticalSystemReminder *string                          `json:"criticalSystemReminder_EXPERIMENTAL,omitempty"` // Experimental critical system reminder

	// Deprecated: ExecutionMode is retained for source compatibility with older
	// SDK releases. Current Claude Code AgentDefinition JSON does not document
	// execution_mode, so the SDK no longer emits this field in --agents payloads.
	ExecutionMode *SubagentExecutionMode `json:"-"`

	// Deprecated: Timeout is retained for source compatibility with older SDK
	// releases. Current Claude Code AgentDefinition JSON does not document
	// timeout, so the SDK no longer emits this field in --agents payloads.
	Timeout *float64 `json:"-"`
}

// UnmarshalJSON accepts both the current documented camelCase AgentDefinition
// keys and the older snake_case keys emitted by previous Go SDK releases.
func (a *AgentDefinition) UnmarshalJSON(data []byte) error {
	type agentDefinitionAlias AgentDefinition
	aux := struct {
		*agentDefinitionAlias

		DisallowedToolsSnake []string               `json:"disallowed_tools"`
		MaxTurnsSnake        *int                   `json:"max_turns"`
		McpServersSnake      []interface{}          `json:"mcp_servers"`
		ExecutionModeSnake   *SubagentExecutionMode `json:"execution_mode"`
		TimeoutValue         *float64               `json:"timeout"`
	}{
		agentDefinitionAlias: (*agentDefinitionAlias)(a),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if len(a.DisallowedTools) == 0 && len(aux.DisallowedToolsSnake) > 0 {
		a.DisallowedTools = aux.DisallowedToolsSnake
	}
	if a.MaxTurns == nil && aux.MaxTurnsSnake != nil {
		a.MaxTurns = aux.MaxTurnsSnake
	}
	if len(a.McpServers) == 0 && len(aux.McpServersSnake) > 0 {
		a.McpServers = aux.McpServersSnake
	}
	if aux.ExecutionModeSnake != nil {
		a.ExecutionMode = aux.ExecutionModeSnake
	}
	if aux.TimeoutValue != nil {
		a.Timeout = aux.TimeoutValue
	}

	return nil
}
