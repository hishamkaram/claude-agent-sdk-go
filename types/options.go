package types

import "time"

// ClaudeAgentOptions represents configuration options for the Claude SDK.
type ClaudeAgentOptions struct {
	// Tool configuration
	AllowedTools    []string `json:"allowed_tools,omitempty"`
	DisallowedTools []string `json:"disallowed_tools,omitempty"`

	// System prompt - can be string or SystemPromptPreset
	SystemPrompt interface{} `json:"system_prompt,omitempty"`

	// MCP servers - can be map[string]interface{} (config), string (path), or actual path
	McpServers interface{} `json:"mcp_servers,omitempty"`

	// Permission configuration
	PermissionMode           *PermissionMode `json:"permission_mode,omitempty"`
	PermissionPromptToolName *string         `json:"permission_prompt_tool_name,omitempty"`

	// Permission bypass configuration (use with caution - only for sandboxed environments)
	// These flags disable ALL permission checks, allowing Claude to execute any tool without approval.
	//
	// DangerouslySkipPermissions: Actually bypass all permissions (requires AllowDangerouslySkipPermissions)
	// AllowDangerouslySkipPermissions: Enable permission bypass as an option
	//
	// Security Warning: Only use in isolated environments with no internet access.
	DangerouslySkipPermissions      bool `json:"dangerously_skip_permissions,omitempty"`
	AllowDangerouslySkipPermissions bool `json:"allow_dangerously_skip_permissions,omitempty"`

	// Session configuration
	ContinueConversation bool    `json:"continue_conversation,omitempty"`
	Resume               *string `json:"resume,omitempty"`
	ForkSession          bool    `json:"fork_session,omitempty"`

	// Model and execution limits
	Model             *string  `json:"model,omitempty"`
	MaxTurns          *int     `json:"max_turns,omitempty"`
	MaxThinkingTokens *int     `json:"max_thinking_tokens,omitempty"` // Deprecated: use Thinking with BudgetTokens instead.
	MaxBudgetUSD      *float64 `json:"max_budget_usd,omitempty"`      // Maximum budget in USD for this query

	// Beta features
	Betas []string `json:"betas,omitempty"` // List of beta feature flags (e.g., "context-1m-2025-08-07")

	// API configuration
	BaseURL *string `json:"base_url,omitempty"` // Custom Anthropic API base URL (ANTHROPIC_BASE_URL)

	// Working directory and CLI path
	CWD     *string `json:"cwd,omitempty"`
	CLIPath *string `json:"cli_path,omitempty"`

	// Settings
	Settings       *string         `json:"settings,omitempty"`
	SettingSources []SettingSource `json:"setting_sources,omitempty"`
	AddDirs        []string        `json:"add_dirs,omitempty"`

	// Environment and extra arguments
	Env       map[string]string  `json:"env,omitempty"`
	ExtraArgs map[string]*string `json:"extra_args,omitempty"` // Pass arbitrary CLI flags

	// Buffer configuration
	MaxBufferSize *int `json:"max_buffer_size,omitempty"` // Max bytes when buffering CLI stdout

	// Observability — Observer receives SDK lifecycle and health telemetry.
	// A nil Observer drops all telemetry (NopObserver semantics). Never serialized.
	Observer Observer `json:"-"`

	// MaxConsecutiveParseErrors caps how many consecutive CLI JSON parse failures
	// the transport tolerates before it terminates the subprocess as unrecoverable.
	// nil or non-positive → the transport default. The terminated subprocess is
	// reaped (not left as a zombie) and the error is surfaced to the consumer.
	MaxConsecutiveParseErrors *uint `json:"max_consecutive_parse_errors,omitempty"`

	// Streaming configuration
	IncludePartialMessages bool `json:"include_partial_messages,omitempty"`

	// User identifier
	User *string `json:"user,omitempty"`

	// Agent definitions
	Agents map[string]AgentDefinition `json:"agents,omitempty"`

	// Session agent name — run the whole session as this configured subagent.
	Agent *string `json:"agent,omitempty"`

	// Subagent execution configuration
	SubagentExecution *SubagentExecutionConfig `json:"subagent_execution,omitempty"`

	// Plugin configurations for custom plugins
	Plugins []PluginConfig `json:"plugins,omitempty"`

	// Reasoning effort control
	Effort *EffortLevel `json:"effort,omitempty"` // "low", "medium", "high", "xhigh", "max" → --effort flag

	// Thinking configuration
	Thinking *ThinkingConfig `json:"thinking,omitempty"` // adaptive/enabled/disabled → --settings JSON

	// Structured output
	OutputFormat *OutputFormat `json:"output_format,omitempty"` // JSON schema output → --json-schema flag

	// Model fallback
	FallbackModel *string `json:"fallback_model,omitempty"` // → --fallback-model flag

	// File checkpointing
	EnableFileCheckpointing bool `json:"enable_file_checkpointing,omitempty"` // → --settings JSON

	// Sandbox configuration
	Sandbox *SandboxConfig `json:"sandbox,omitempty"` // → --settings JSON

	// Session persistence control
	PersistSession *bool `json:"persist_session,omitempty"` // false → --no-session-persistence flag

	// Session ID
	SessionID *string `json:"session_id,omitempty"` // → --session-id flag

	// Prompt suggestions
	PromptSuggestions bool `json:"prompt_suggestions,omitempty"` // → init control protocol

	// Custom process spawner (not marshaled — consumer provides implementation)
	SpawnProcess ProcessSpawner `json:"-"`

	// Resume session at a specific message UUID
	ResumeSessionAt *string `json:"resume_session_at,omitempty"` // → --resume-session-at CLI flag

	// Built-in tool configuration (delivered via settings JSON)
	ToolConfig *ToolConfig `json:"tool_config,omitempty"` // → settings JSON "toolConfig" key

	// Tool loading preset — []string of tool names or preset struct → --tools CLI flag
	// This is different from AllowedTools which auto-approves tools.
	Tools interface{} `json:"tools,omitempty"`

	// Debug file path — implicitly enables debug mode
	DebugFile *string `json:"debug_file,omitempty"` // → --debug-file CLI flag

	// Strict MCP config validation
	StrictMcpConfig bool `json:"strict_mcp_config,omitempty"` // → --strict-mcp-config CLI flag

	// Task budget in USD — limits spending for a single task invocation
	TaskBudget *float64 `json:"taskBudget,omitempty"` // → --task-budget CLI flag

	// Agent progress summaries — experimental flag retained for forward-compatible callers.
	// Current Claude Code CLI releases may reject this option with an unknown-flag error.
	AgentProgressSummaries bool `json:"agentProgressSummaries,omitempty"` // → --agent-progress-summaries CLI flag

	// Include hook events — receive hook lifecycle events in the message stream
	IncludeHookEvents bool `json:"includeHookEvents,omitempty"` // → settings JSON "includeHookEvents" key

	// Debug and diagnostics
	Verbose bool `json:"-"` // Enable verbose debug logging

	// Callbacks (not marshaled to JSON)
	CanUseTool             CanUseToolFunc              `json:"-"`
	ToolCallbackTimeout    time.Duration               `json:"-"` // Timeout for CanUseTool callback; defaults to 5m if zero
	ControlResponseTimeout time.Duration               `json:"-"` // Timeout for SDK control responses; defaults to 30s if zero
	Hooks                  map[HookEvent][]HookMatcher `json:"-"`
	Stderr                 StderrCallbackFunc          `json:"-"`

	// Stderr file logging (SDK-managed, configuration-time only)
	// - nil (default): No file logging
	// - &"": Use default location (~/.claude/agents_server/cli_stderr.log)
	// - &"path": Use custom path
	// For runtime control, use the Stderr callback instead
	StderrLogFile *string `json:"-"`

	// SessionStore mirrors Claude Code transcript entries for SDK-managed
	// runtimes. Resume loads are materialized into an isolated
	// CLAUDE_CONFIG_DIR; runtime mirror failures are surfaced as system
	// messages and do not interrupt Claude's local transcript durability.
	SessionStore SessionStore `json:"-"`

	// SessionStoreKey overrides the default key derived from CWD + Resume.
	SessionStoreKey *SessionKey `json:"-"`
}

// NewClaudeAgentOptions creates a new ClaudeAgentOptions with sensible defaults.
func NewClaudeAgentOptions() *ClaudeAgentOptions {
	return &ClaudeAgentOptions{
		AllowedTools:           []string{},
		DisallowedTools:        []string{},
		Env:                    make(map[string]string),
		ExtraArgs:              make(map[string]*string),
		ContinueConversation:   false,
		ForkSession:            false,
		IncludePartialMessages: false,
		Plugins:                []PluginConfig{},
		Betas:                  []string{},
	}
}
