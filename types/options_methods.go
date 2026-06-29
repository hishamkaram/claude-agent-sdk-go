package types

import "time"

// WithAllowedTools sets the allowed tools.
func (o *ClaudeAgentOptions) WithAllowedTools(tools ...string) *ClaudeAgentOptions {
	o.AllowedTools = tools
	return o
}

// WithDisallowedTools sets the disallowed tools.
func (o *ClaudeAgentOptions) WithDisallowedTools(tools ...string) *ClaudeAgentOptions {
	o.DisallowedTools = tools
	return o
}

// WithSystemPrompt sets the system prompt (can be string or SystemPromptPreset).
func (o *ClaudeAgentOptions) WithSystemPrompt(prompt interface{}) *ClaudeAgentOptions {
	o.SystemPrompt = prompt
	return o
}

// WithSystemPromptString sets the system prompt as a string.
func (o *ClaudeAgentOptions) WithSystemPromptString(prompt string) *ClaudeAgentOptions {
	o.SystemPrompt = prompt
	return o
}

// WithSystemPromptPreset sets the system prompt as a preset.
func (o *ClaudeAgentOptions) WithSystemPromptPreset(preset SystemPromptPreset) *ClaudeAgentOptions {
	o.SystemPrompt = preset
	return o
}

// WithMcpServers sets the MCP servers configuration.
func (o *ClaudeAgentOptions) WithMcpServers(servers interface{}) *ClaudeAgentOptions {
	o.McpServers = servers
	return o
}

// WithPermissionMode sets the permission mode.
func (o *ClaudeAgentOptions) WithPermissionMode(mode PermissionMode) *ClaudeAgentOptions {
	o.PermissionMode = &mode
	return o
}

// WithPermissionPromptToolName sets the permission prompt tool name.
func (o *ClaudeAgentOptions) WithPermissionPromptToolName(toolName string) *ClaudeAgentOptions {
	o.PermissionPromptToolName = &toolName
	return o
}

// WithContinueConversation sets whether to continue the conversation.
func (o *ClaudeAgentOptions) WithContinueConversation(continue_ bool) *ClaudeAgentOptions {
	o.ContinueConversation = continue_
	return o
}

// WithResume sets the session ID to resume.
func (o *ClaudeAgentOptions) WithResume(sessionID string) *ClaudeAgentOptions {
	o.Resume = &sessionID
	return o
}

// WithSessionStore sets the mirrored SessionStore used for resume hydration
// and runtime transcript appends.
func (o *ClaudeAgentOptions) WithSessionStore(store SessionStore) *ClaudeAgentOptions {
	o.SessionStore = store
	return o
}

// WithSessionStoreKey overrides the key used with WithSessionStore.
func (o *ClaudeAgentOptions) WithSessionStoreKey(key SessionKey) *ClaudeAgentOptions {
	o.SessionStoreKey = &key
	return o
}

// WithForkSession sets whether to fork the session.
func (o *ClaudeAgentOptions) WithForkSession(fork bool) *ClaudeAgentOptions {
	o.ForkSession = fork
	return o
}

// WithModel sets the model to use.
func (o *ClaudeAgentOptions) WithModel(model string) *ClaudeAgentOptions {
	o.Model = &model
	return o
}

// WithMaxTurns sets the maximum number of turns.
func (o *ClaudeAgentOptions) WithMaxTurns(maxTurns int) *ClaudeAgentOptions {
	o.MaxTurns = &maxTurns
	return o
}

// WithMaxThinkingTokens sets the maximum tokens for extended thinking.
//
// Deprecated: prefer WithThinking(ThinkingConfig{Type: "enabled", BudgetTokens: &n}).
func (o *ClaudeAgentOptions) WithMaxThinkingTokens(maxTokens int) *ClaudeAgentOptions {
	o.MaxThinkingTokens = &maxTokens
	return o
}

// WithMaxBudgetUSD sets the maximum budget in USD for this query.
// This helps prevent unexpectedly high API costs by stopping execution when the limit is reached.
func (o *ClaudeAgentOptions) WithMaxBudgetUSD(maxBudget float64) *ClaudeAgentOptions {
	o.MaxBudgetUSD = &maxBudget
	return o
}

// WithBetas sets the beta feature flags to opt into Anthropic beta APIs.
// Example: []string{"context-1m-2025-08-07"} for extended context window support.
func (o *ClaudeAgentOptions) WithBetas(betas []string) *ClaudeAgentOptions {
	o.Betas = betas
	return o
}

// WithBeta adds a single beta feature flag.
// This is useful for adding beta features without replacing the existing list.
func (o *ClaudeAgentOptions) WithBeta(beta string) *ClaudeAgentOptions {
	o.Betas = append(o.Betas, beta)
	return o
}

// WithBaseURL sets the custom Anthropic API base URL.
func (o *ClaudeAgentOptions) WithBaseURL(baseURL string) *ClaudeAgentOptions {
	o.BaseURL = &baseURL
	return o
}

// WithCWD sets the working directory.
func (o *ClaudeAgentOptions) WithCWD(cwd string) *ClaudeAgentOptions {
	o.CWD = &cwd
	return o
}

// WithCLIPath sets the CLI binary path.
func (o *ClaudeAgentOptions) WithCLIPath(cliPath string) *ClaudeAgentOptions {
	o.CLIPath = &cliPath
	return o
}

// WithSettings sets the settings file path.
func (o *ClaudeAgentOptions) WithSettings(settings string) *ClaudeAgentOptions {
	o.Settings = &settings
	return o
}

// WithSettingSources sets the setting sources to load.
func (o *ClaudeAgentOptions) WithSettingSources(sources ...SettingSource) *ClaudeAgentOptions {
	o.SettingSources = sources
	return o
}

// WithAddDirs sets the directories to add.
func (o *ClaudeAgentOptions) WithAddDirs(dirs ...string) *ClaudeAgentOptions {
	o.AddDirs = dirs
	return o
}

// WithEnv sets environment variables.
func (o *ClaudeAgentOptions) WithEnv(env map[string]string) *ClaudeAgentOptions {
	o.Env = env
	return o
}

// WithEnvVar sets a single environment variable.
func (o *ClaudeAgentOptions) WithEnvVar(key, value string) *ClaudeAgentOptions {
	if o.Env == nil {
		o.Env = make(map[string]string)
	}
	o.Env[key] = value
	return o
}

// WithExtraArgs sets extra CLI arguments.
func (o *ClaudeAgentOptions) WithExtraArgs(args map[string]*string) *ClaudeAgentOptions {
	o.ExtraArgs = args
	return o
}

// WithExtraArg sets a single extra CLI argument.
func (o *ClaudeAgentOptions) WithExtraArg(key string, value *string) *ClaudeAgentOptions {
	if o.ExtraArgs == nil {
		o.ExtraArgs = make(map[string]*string)
	}
	o.ExtraArgs[key] = value
	return o
}

// WithMaxBufferSize sets the maximum buffer size.
func (o *ClaudeAgentOptions) WithMaxBufferSize(size int) *ClaudeAgentOptions {
	o.MaxBufferSize = &size
	return o
}

// WithObserver sets the telemetry Observer for SDK lifecycle and health events.
func (o *ClaudeAgentOptions) WithObserver(obs Observer) *ClaudeAgentOptions {
	o.Observer = obs
	return o
}

// ObserverOrNop returns the configured Observer, or NopObserver when none is set
// (including when the receiver itself is nil). This is the single source of the
// "default to no-op" rule — callers in the transport and query layers use it so
// telemetry call sites never need their own nil guards.
func (o *ClaudeAgentOptions) ObserverOrNop() Observer {
	if o == nil || o.Observer == nil {
		return NopObserver{}
	}
	return o.Observer
}

// WithMaxConsecutiveParseErrors sets how many consecutive CLI JSON parse failures
// the transport tolerates before terminating the subprocess. Non-positive is ignored.
func (o *ClaudeAgentOptions) WithMaxConsecutiveParseErrors(n uint) *ClaudeAgentOptions {
	o.MaxConsecutiveParseErrors = &n
	return o
}

// WithIncludePartialMessages sets whether to include partial messages.
func (o *ClaudeAgentOptions) WithIncludePartialMessages(include bool) *ClaudeAgentOptions {
	o.IncludePartialMessages = include
	return o
}

// WithUser sets the user identifier.
func (o *ClaudeAgentOptions) WithUser(user string) *ClaudeAgentOptions {
	o.User = &user
	return o
}

// WithAgents sets the agent definitions.
func (o *ClaudeAgentOptions) WithAgents(agents map[string]AgentDefinition) *ClaudeAgentOptions {
	o.Agents = agents
	return o
}

// WithAgent sets a single agent definition.
func (o *ClaudeAgentOptions) WithAgent(name string, agent AgentDefinition) *ClaudeAgentOptions {
	if o.Agents == nil {
		o.Agents = make(map[string]AgentDefinition)
	}
	o.Agents[name] = agent
	return o
}

// WithSessionAgent sets the configured subagent to use as the main session agent.
func (o *ClaudeAgentOptions) WithSessionAgent(name string) *ClaudeAgentOptions {
	o.Agent = &name
	return o
}

// WithSubagentExecution sets the subagent execution configuration.
// This emits an experimental CLI flag retained for forward-compatible callers.
// Current Claude Code CLI releases may reject it with an unknown-flag error.
func (o *ClaudeAgentOptions) WithSubagentExecution(config *SubagentExecutionConfig) *ClaudeAgentOptions {
	o.SubagentExecution = config
	return o
}

// WithPlugins sets the plugin configurations.
func (o *ClaudeAgentOptions) WithPlugins(plugins []PluginConfig) *ClaudeAgentOptions {
	o.Plugins = plugins
	return o
}

// WithPlugin adds a single plugin configuration.
func (o *ClaudeAgentOptions) WithPlugin(plugin PluginConfig) *ClaudeAgentOptions {
	o.Plugins = append(o.Plugins, plugin)
	return o
}

// WithLocalPlugin adds a local plugin by path (convenience method).
// This is the most common way to add plugins.
func (o *ClaudeAgentOptions) WithLocalPlugin(path string) *ClaudeAgentOptions {
	o.Plugins = append(o.Plugins, *NewLocalPluginConfig(path))
	return o
}

// WithCanUseTool sets the tool permission callback.
func (o *ClaudeAgentOptions) WithCanUseTool(callback CanUseToolFunc) *ClaudeAgentOptions {
	o.CanUseTool = callback
	return o
}

// WithToolCallbackTimeout sets the timeout for the CanUseTool callback.
// If zero, defaults to 5 minutes.
func (o *ClaudeAgentOptions) WithToolCallbackTimeout(d time.Duration) *ClaudeAgentOptions {
	o.ToolCallbackTimeout = d
	return o
}

// WithHooks sets the hook configurations.
func (o *ClaudeAgentOptions) WithHooks(hooks map[HookEvent][]HookMatcher) *ClaudeAgentOptions {
	o.Hooks = hooks
	return o
}

// WithHook adds a hook matcher for a specific event.
func (o *ClaudeAgentOptions) WithHook(event HookEvent, matcher HookMatcher) *ClaudeAgentOptions {
	if o.Hooks == nil {
		o.Hooks = make(map[HookEvent][]HookMatcher)
	}
	o.Hooks[event] = append(o.Hooks[event], matcher)
	return o
}

// WithStderr sets the stderr callback.
func (o *ClaudeAgentOptions) WithStderr(callback StderrCallbackFunc) *ClaudeAgentOptions {
	o.Stderr = callback
	return o
}

// WithStderrLogFile enables SDK-managed stderr file logging.
// Pass nil to disable (default), empty string for default location, or custom path.
func (o *ClaudeAgentOptions) WithStderrLogFile(path *string) *ClaudeAgentOptions {
	o.StderrLogFile = path
	return o
}

// WithDefaultStderrLogFile enables stderr logging to default location.
// Default: ~/.claude/agents_server/cli_stderr.log
func (o *ClaudeAgentOptions) WithDefaultStderrLogFile() *ClaudeAgentOptions {
	empty := ""
	o.StderrLogFile = &empty
	return o
}

// WithCustomStderrLogFile enables stderr logging to a custom file path.
func (o *ClaudeAgentOptions) WithCustomStderrLogFile(path string) *ClaudeAgentOptions {
	o.StderrLogFile = &path
	return o
}

// WithVerbose enables or disables verbose debug logging.
func (o *ClaudeAgentOptions) WithVerbose(enabled bool) *ClaudeAgentOptions {
	o.Verbose = enabled
	return o
}

// WithDangerouslySkipPermissions bypasses all permission checks.
// This is DANGEROUS and should only be used in sandboxed environments.
// Requires AllowDangerouslySkipPermissions to be enabled first.
//
// Security Warning: This disables ALL safety checks. Only use in isolated environments
// with no internet access and no sensitive data.
func (o *ClaudeAgentOptions) WithDangerouslySkipPermissions(skip bool) *ClaudeAgentOptions {
	o.DangerouslySkipPermissions = skip
	return o
}

// WithAllowDangerouslySkipPermissions enables the option to bypass permissions.
// This must be set to true before DangerouslySkipPermissions can be used.
//
// This is the "safety switch" that allows the dangerous flag to work.
func (o *ClaudeAgentOptions) WithAllowDangerouslySkipPermissions(allow bool) *ClaudeAgentOptions {
	o.AllowDangerouslySkipPermissions = allow
	return o
}

// WithEffort sets the reasoning effort level (low/medium/high/xhigh/max).
func (o *ClaudeAgentOptions) WithEffort(level EffortLevel) *ClaudeAgentOptions {
	o.Effort = &level
	return o
}

// WithThinking sets the thinking configuration (adaptive/enabled/disabled).
func (o *ClaudeAgentOptions) WithThinking(config ThinkingConfig) *ClaudeAgentOptions {
	o.Thinking = &config
	return o
}

// WithOutputFormat sets the structured output format configuration.
func (o *ClaudeAgentOptions) WithOutputFormat(format OutputFormat) *ClaudeAgentOptions {
	o.OutputFormat = &format
	return o
}

// WithFallbackModel sets the fallback model name.
func (o *ClaudeAgentOptions) WithFallbackModel(model string) *ClaudeAgentOptions {
	o.FallbackModel = &model
	return o
}

// WithEnableFileCheckpointing enables or disables file checkpointing.
func (o *ClaudeAgentOptions) WithEnableFileCheckpointing(enabled bool) *ClaudeAgentOptions {
	o.EnableFileCheckpointing = enabled
	return o
}

// WithSandbox sets the sandbox configuration.
func (o *ClaudeAgentOptions) WithSandbox(config SandboxConfig) *ClaudeAgentOptions {
	o.Sandbox = &config
	return o
}

// WithPersistSession controls session persistence. Pass false to disable.
func (o *ClaudeAgentOptions) WithPersistSession(persist bool) *ClaudeAgentOptions {
	o.PersistSession = &persist
	return o
}

// WithSessionID sets a specific session ID instead of auto-generating.
func (o *ClaudeAgentOptions) WithSessionID(id string) *ClaudeAgentOptions {
	o.SessionID = &id
	return o
}

// WithPromptSuggestions enables or disables prompt suggestions.
func (o *ClaudeAgentOptions) WithPromptSuggestions(enabled bool) *ClaudeAgentOptions {
	o.PromptSuggestions = enabled
	return o
}

// WithSpawnProcess sets a custom process spawner for running Claude Code
// in non-local environments (Docker, VMs, SSH, etc.).
// When set, the SDK calls this function instead of exec.Command.
func (o *ClaudeAgentOptions) WithSpawnProcess(spawner ProcessSpawner) *ClaudeAgentOptions {
	o.SpawnProcess = spawner
	return o
}

// WithResumeSessionAt sets the message UUID to resume a session at.
// This allows branching from a specific point in the conversation.
func (o *ClaudeAgentOptions) WithResumeSessionAt(messageID string) *ClaudeAgentOptions {
	o.ResumeSessionAt = &messageID
	return o
}

// WithToolConfig sets the built-in tool configuration (bash timeout, computer display, etc.).
func (o *ClaudeAgentOptions) WithToolConfig(config ToolConfig) *ClaudeAgentOptions {
	o.ToolConfig = &config
	return o
}

// WithTools sets the tool loading preset.
// Accepts []string of tool names or a preset struct (e.g., map[string]string{"type": "preset", "preset": "claude_code"}).
// This is different from AllowedTools which auto-approves tools.
func (o *ClaudeAgentOptions) WithTools(tools interface{}) *ClaudeAgentOptions {
	o.Tools = tools
	return o
}

// WithDebugFile sets the debug log file path. Implicitly enables debug mode.
func (o *ClaudeAgentOptions) WithDebugFile(path string) *ClaudeAgentOptions {
	o.DebugFile = &path
	return o
}

// WithStrictMcpConfig enables or disables strict MCP configuration validation.
func (o *ClaudeAgentOptions) WithStrictMcpConfig(strict bool) *ClaudeAgentOptions {
	o.StrictMcpConfig = strict
	return o
}

// WithTaskBudget sets the dollar budget for a single task invocation.
func (o *ClaudeAgentOptions) WithTaskBudget(budget float64) *ClaudeAgentOptions {
	o.TaskBudget = &budget
	return o
}

// WithAgentProgressSummaries enables or disables agent progress summary messages.
// This emits an experimental CLI flag retained for forward-compatible callers.
// Current Claude Code CLI releases may reject it with an unknown-flag error.
func (o *ClaudeAgentOptions) WithAgentProgressSummaries(enabled bool) *ClaudeAgentOptions {
	o.AgentProgressSummaries = enabled
	return o
}

// WithIncludeHookEvents enables or disables hook lifecycle events in the message stream.
func (o *ClaudeAgentOptions) WithIncludeHookEvents(enabled bool) *ClaudeAgentOptions {
	o.IncludeHookEvents = enabled
	return o
}
