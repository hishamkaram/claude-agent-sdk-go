package transport

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// buildCommandArgs builds the command line arguments for the CLI subprocess.
// This is extracted into a separate method to allow for testing.
func (t *SubprocessCLITransport) buildCommandArgs() []string {
	args := []string{
		"--input-format=stream-json",
		"--output-format=stream-json",
		"--verbose",
	}

	// Each helper appends its flag group in declaration order; the call order
	// below is the canonical CLI argument order (pinned by the args oracle).
	args = t.appendPermissionArgs(args)
	args = t.appendSystemPromptArgs(args)
	args = t.appendModelAndResumeArgs(args)
	args = t.appendBudgetArgs(args)
	args = t.appendCollectionArgs(args)
	args = t.appendAgentAndEffortArgs(args)
	args = t.appendSessionArgs(args)
	args = t.appendOutputArgs(args)
	args = t.appendSubagentArgs(args)
	args = t.appendToolArgs(args)
	args = t.appendMiscArgs(args)

	return args
}

// appendPermissionArgs adds the permission-prompt-tool and permission-mode flags.
func (t *SubprocessCLITransport) appendPermissionArgs(args []string) []string {
	// Add permission prompt tool if specified
	if t.options != nil && t.options.PermissionPromptToolName != nil {
		args = append(args, "--permission-prompt-tool", *t.options.PermissionPromptToolName)
		t.logger.Debug("setting permission prompt tool", zap.String("tool", *t.options.PermissionPromptToolName))
	}

	// The canonical default means "use this CLI's default". Omitting the flag
	// avoids coupling the SDK to provider spellings such as "manual".
	if t.options != nil && t.options.PermissionMode != nil &&
		*t.options.PermissionMode != types.PermissionModeDefault {
		args = append(args, "--permission-mode", string(*t.options.PermissionMode))
		t.logger.Debug("setting permission mode", zap.String("mode", string(*t.options.PermissionMode)))
	}
	return args
}

// appendSystemPromptArgs adds the --system-prompt / --append-system-prompt flag.
func (t *SubprocessCLITransport) appendSystemPromptArgs(args []string) []string {
	// Add system prompt - always pass the flag to match Python SDK behavior
	// When nil, pass empty string to prevent unintended Claude Code defaults
	if t.options != nil {
		switch prompt := t.options.SystemPrompt.(type) {
		case nil:
			// Default to empty system prompt when not specified
			args = append(args, "--system-prompt", "")
			t.logger.Debug("Setting empty system prompt (default)")
		case string:
			// Handle string prompt
			args = append(args, "--system-prompt", prompt)
			t.logger.Debug("setting system prompt", zap.String("prompt", prompt))
		case types.SystemPromptPreset:
			// Handle preset case - append to default Claude Code prompt
			if prompt.Append != nil {
				args = append(args, "--append-system-prompt", *prompt.Append)
				t.logger.Debug("appending to system prompt preset", zap.String("append", *prompt.Append))
			}
		}
	} else {
		// No options provided, use empty system prompt
		args = append(args, "--system-prompt", "")
		t.logger.Debug("Setting empty system prompt (no options)")
	}
	return args
}

// appendModelAndResumeArgs adds model, resume, fork-session, and the permission
// bypass safety flags.
func (t *SubprocessCLITransport) appendModelAndResumeArgs(args []string) []string {
	// Add model if specified
	if t.options != nil && t.options.Model != nil {
		args = append(args, "--model", *t.options.Model)
		t.logger.Debug("setting model", zap.String("model", *t.options.Model))
	}

	// Add --resume flag if resuming a conversation
	if t.resumeSessionID != "" {
		args = append(args, "--resume", t.resumeSessionID)
		t.logger.Debug("resuming Claude CLI conversation", zap.String("session_id", t.resumeSessionID))
	}

	// Add --fork-session flag if forking a resumed session
	if t.options != nil && t.options.ForkSession {
		args = append(args, "--fork-session")
		t.logger.Debug("Forking resumed session to new session ID")
	}

	// Add permission bypass flags if enabled
	if t.options != nil {
		// Must set allow flag first (acts as safety switch)
		if t.options.AllowDangerouslySkipPermissions {
			args = append(args, "--allow-dangerously-skip-permissions")
			t.logger.Debug("Allowing permission bypass (safety switch enabled)")

			// Only add skip flag if allow flag is also set
			if t.options.DangerouslySkipPermissions {
				args = append(args, "--dangerously-skip-permissions")
				t.logger.Debug("DANGER: Bypassing all permissions - use only in sandboxed environments!")
			}
		}
	}
	return args
}

// appendBudgetArgs adds the thinking-token and budget limit flags.
func (t *SubprocessCLITransport) appendBudgetArgs(args []string) []string {
	// Add extended thinking token limit if specified
	if t.options != nil && t.options.MaxThinkingTokens != nil {
		args = append(args, "--max-thinking-tokens", fmt.Sprintf("%d", *t.options.MaxThinkingTokens))
		t.logger.Debug("setting max thinking tokens", zap.Int("max_thinking_tokens", *t.options.MaxThinkingTokens))
	}

	// Add budget limit if specified
	if t.options != nil && t.options.MaxBudgetUSD != nil {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", *t.options.MaxBudgetUSD))
		t.logger.Debug("setting max budget", zap.Float64("max_budget_usd", *t.options.MaxBudgetUSD))
	}
	return args
}

// appendCollectionArgs adds the multi-value option flags: betas, plugin dirs,
// and setting sources.
func (t *SubprocessCLITransport) appendCollectionArgs(args []string) []string {
	// Add beta feature flags if specified
	if t.options != nil && len(t.options.Betas) > 0 {
		for _, beta := range t.options.Betas {
			args = append(args, "--betas", beta)
			t.logger.Debug("adding beta feature flag", zap.String("beta", beta))
		}
	}

	// Add plugin directories
	if t.options != nil && len(t.options.Plugins) > 0 {
		for _, plugin := range t.options.Plugins {
			if plugin.Type == "local" {
				args = append(args, "--plugin-dir", plugin.Path)
				t.logger.Debug("adding plugin directory", zap.String("path", plugin.Path))
			} else {
				// This shouldn't happen if NewPluginConfig is used, but handle it anyway
				t.logger.Warn("skipping unsupported plugin type", zap.String("type", plugin.Type))
			}
		}
	}

	// Add setting sources if specified (enables local slash commands, CLAUDE.md, etc.)
	if t.options != nil && len(t.options.SettingSources) > 0 {
		sources := make([]string, len(t.options.SettingSources))
		for i, src := range t.options.SettingSources {
			sources[i] = string(src)
		}
		args = append(args, "--setting-sources", joinStrings(sources, ","))
		t.logger.Debug("setting sources", zap.String("sources", joinStrings(sources, ",")))
	}
	return args
}

// appendAgentAndEffortArgs adds the agents JSON, session agent, and effort flags.
func (t *SubprocessCLITransport) appendAgentAndEffortArgs(args []string) []string {
	// Add agents if specified
	if t.options != nil && len(t.options.Agents) > 0 {
		agentsJSONBytes, err := json.Marshal(t.options.Agents)
		if err != nil {
			t.logger.Warn("failed to marshal agents to JSON", zap.Error(err))
		} else {
			args = append(args, "--agents", string(agentsJSONBytes))
			t.logger.Debug("agents configuration", zap.String("agents_json", string(agentsJSONBytes)))
		}
	}

	// Add session agent if specified
	if t.options != nil && t.options.Agent != nil {
		args = append(args, "--agent", *t.options.Agent)
		t.logger.Debug("setting session agent", zap.String("agent", *t.options.Agent))
	}

	// Add effort level if specified
	if t.options != nil && t.options.Effort != nil {
		args = append(args, "--effort", string(*t.options.Effort))
		t.logger.Debug("setting effort level", zap.String("effort", string(*t.options.Effort)))
	}
	return args
}

// appendSessionArgs adds thinking-display, fallback model, session id, and
// session persistence.
func (t *SubprocessCLITransport) appendSessionArgs(args []string) []string {
	// Claude Code consumes thinking display as a CLI flag. Keep the typed
	// settings JSON for thinking mode/budget, and pass display explicitly so
	// summarized/omitted behavior is honored by the subprocess.
	if t.thinkingDisplaySupported && t.wantsThinkingDisplay() {
		args = append(args, "--thinking-display", t.options.Thinking.Display)
		t.logger.Debug("setting thinking display", zap.String("display", t.options.Thinking.Display))
	}

	// Add fallback model if specified
	if t.options != nil && t.options.FallbackModel != nil {
		args = append(args, "--fallback-model", *t.options.FallbackModel)
		t.logger.Debug("setting fallback model", zap.String("fallback_model", *t.options.FallbackModel))
	}

	// Add session ID if specified
	if t.options != nil && t.options.SessionID != nil {
		args = append(args, "--session-id", *t.options.SessionID)
		t.logger.Debug("setting session ID", zap.String("session_id", *t.options.SessionID))
	}

	// Add no-session-persistence flag if PersistSession is explicitly false
	if t.options != nil && t.options.PersistSession != nil && !*t.options.PersistSession {
		args = append(args, "--no-session-persistence")
		t.logger.Debug("Disabling session persistence")
	}
	return args
}

// appendOutputArgs adds the output-format / json-schema, settings JSON, and
// replay-user-messages flags.
func (t *SubprocessCLITransport) appendOutputArgs(args []string) []string {
	// Add JSON schema output format if specified
	if t.options != nil && t.options.OutputFormat != nil {
		schemaJSON, err := json.Marshal(t.options.OutputFormat)
		if err != nil {
			t.logger.Warn("failed to marshal output format to JSON", zap.Error(err))
		} else {
			args = append(args, "--json-schema", string(schemaJSON))
			t.logger.Debug("setting JSON schema output format", zap.String("schema", string(schemaJSON)))
		}
	}

	// Build and add settings JSON if needed (thinking, sandbox, file checkpointing)
	if t.options != nil {
		settingsJSON := t.buildSettingsJSON()
		if settingsJSON != "" {
			args = append(args, "--settings", settingsJSON)
			t.logger.Debug("setting settings JSON", zap.String("settings", settingsJSON))
		}
	}

	// When file checkpointing is enabled, also request user message UUIDs
	// for checkpoint targeting (rewind, branch-at-message).
	if t.options != nil && t.options.EnableFileCheckpointing {
		args = append(args, "--replay-user-messages")
	}
	return args
}

// appendSubagentArgs adds the subagent-execution configuration flag.
func (t *SubprocessCLITransport) appendSubagentArgs(args []string) []string {
	// Add subagent execution configuration if specified
	if t.options != nil && t.options.SubagentExecution != nil {
		subagentJSON := make(map[string]interface{})

		if t.options.SubagentExecution.MultiInvocation != "" {
			subagentJSON["multi_invocation"] = string(t.options.SubagentExecution.MultiInvocation)
		}
		if t.options.SubagentExecution.MaxConcurrent > 0 {
			subagentJSON["max_concurrent"] = t.options.SubagentExecution.MaxConcurrent
		}
		if t.options.SubagentExecution.ErrorHandling != "" {
			subagentJSON["error_handling"] = string(t.options.SubagentExecution.ErrorHandling)
		}

		if len(subagentJSON) > 0 {
			subagentJSONBytes, err := json.Marshal(subagentJSON)
			switch {
			case err != nil:
				t.logger.Warn("failed to marshal subagent execution config to JSON", zap.Error(err))
			case t.subagentExecutionSupported:
				args = append(args, "--subagent-execution", string(subagentJSONBytes))
				t.logger.Debug("subagent execution configuration", zap.String("config", string(subagentJSONBytes)))
			default:
				t.logger.Warn("Claude CLI does not support --subagent-execution; skipping to avoid Connect failure")
			}
		}
	}
	return args
}

// appendToolArgs adds the resume-session-at and tools flags.
func (t *SubprocessCLITransport) appendToolArgs(args []string) []string {
	// Add --resume-session-at flag
	if t.options != nil && t.options.ResumeSessionAt != nil {
		args = append(args, "--resume-session-at", *t.options.ResumeSessionAt)
		t.logger.Debug("setting resume session at", zap.String("resume_at", *t.options.ResumeSessionAt))
	}

	// Add --tools flag ([]string joined by comma, or JSON for preset)
	if t.options != nil && t.options.Tools != nil {
		switch v := t.options.Tools.(type) {
		case []string:
			if len(v) > 0 {
				args = append(args, "--tools", joinStrings(v, ","))
				t.logger.Debug("setting tools", zap.Strings("tools", v))
			}
		default:
			// Serialize non-string-slice values as JSON (e.g., preset objects)
			toolsJSON, err := json.Marshal(v)
			if err != nil {
				t.logger.Warn("failed to marshal tools to JSON", zap.Error(err))
			} else {
				args = append(args, "--tools", string(toolsJSON))
				t.logger.Debug("setting tools (JSON)", zap.String("tools_json", string(toolsJSON)))
			}
		}
	}
	return args
}

// appendMiscArgs adds debug-file, mcp-config, strict-mcp-config, task-budget,
// and the capability-gated agent-progress-summaries flag.
func (t *SubprocessCLITransport) appendMiscArgs(args []string) []string {
	// Add --debug-file flag
	if t.options != nil && t.options.DebugFile != nil {
		args = append(args, "--debug-file", *t.options.DebugFile)
		t.logger.Debug("setting debug file", zap.String("debug_file", *t.options.DebugFile))
	}

	// Add --mcp-config flag with inline JSON when McpServers is a non-empty
	// name->config map. The Claude CLI's --mcp-config accepts an inline config
	// value of the form {"mcpServers": {<name>: <config>, ...}} (same envelope as
	// .mcp.json). McpServers holds the INNER name->config map (per its option doc
	// comment and the SDK mcpServers convention), so it is wrapped here. Inline
	// JSON only — no temp file is written, preserving non-persistence. nil/empty
	// or non-map values emit nothing (backward compatible). --strict-mcp-config
	// emission below is unchanged; the two flags compose (config + strict modifier).
	args = t.appendMcpConfigArg(args)

	// Add --strict-mcp-config flag
	if t.options != nil && t.options.StrictMcpConfig {
		args = append(args, "--strict-mcp-config")
		t.logger.Debug("Enabling strict MCP config")
	}

	// Add --task-budget flag if specified
	if t.options != nil && t.options.TaskBudget != nil {
		args = append(args, "--task-budget", fmt.Sprintf("%.2f", *t.options.TaskBudget))
		t.logger.Debug("setting task budget", zap.Float64("task_budget", *t.options.TaskBudget))
	}

	// Add --agent-progress-summaries flag if enabled AND the CLI supports it.
	if t.options != nil && t.options.AgentProgressSummaries {
		if t.agentProgressSummariesSupported {
			args = append(args, "--agent-progress-summaries")
			t.logger.Debug("Enabling agent progress summaries")
		} else {
			t.logger.Warn("Claude CLI does not support --agent-progress-summaries; skipping to avoid Connect failure")
		}
	}
	return args
}

// appendMcpConfigArg appends the --mcp-config flag with an inline JSON envelope
// derived from options.McpServers, when it is a non-empty map[string]interface{}
// (name->config). The CLI envelope is {"mcpServers": <map>}. This is a generic
// SDK behavior: the WithMcpServers option already promises to deliver MCP servers
// to the CLI, and inline --mcp-config is the documented delivery mechanism. No
// temp file is written — the JSON is passed inline, preserving non-persistence.
func (t *SubprocessCLITransport) appendMcpConfigArg(args []string) []string {
	if t.options == nil || t.options.McpServers == nil {
		return args
	}
	// Only a name->config map is emittable. String path refs and other shapes
	// fall through to the CLI's own config discovery, unchanged.
	mcpMap, ok := t.options.McpServers.(map[string]interface{})
	if !ok || len(mcpMap) == 0 {
		return args
	}
	envelope := map[string]interface{}{"mcpServers": mcpMap}
	configJSON, err := json.Marshal(envelope)
	if err != nil {
		// Effectively unreachable: a map[string]interface{} built from
		// JSON-decodable config marshals cleanly. Log and skip rather than
		// emit a malformed flag — never silently drop without a record.
		t.logger.Warn("appendMcpConfigArg: failed to marshal MCP config envelope; skipping --mcp-config",
			zap.Error(err))
		return args
	}
	args = append(args, "--mcp-config", string(configJSON))
	t.logger.Debug("setting MCP servers via --mcp-config", zap.Int("server_count", len(mcpMap)))
	return args
}

// joinStrings joins strings with a separator (avoiding strings import)
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// buildSettingsJSON constructs the --settings JSON string from typed option fields.
// It merges typed fields (Thinking, Sandbox, EnableFileCheckpointing) on top of
// any user-provided Settings string. Typed fields take precedence on conflict.
func (t *SubprocessCLITransport) buildSettingsJSON() string {
	// hasTypedSettings is the OR of every typed settings field; by De Morgan
	// !hasTypedSettings equals the original "none of the typed fields set" guard.
	hasTyped := t.hasTypedSettings()

	if !hasTyped && t.options.Settings == nil {
		return ""
	}

	// Start with user-provided settings as base (if any)
	settings := make(map[string]interface{})
	if t.options.Settings != nil && *t.options.Settings != "" {
		if err := json.Unmarshal([]byte(*t.options.Settings), &settings); err != nil {
			t.logger.Warn("failed to parse user settings JSON, using typed fields only", zap.Error(err))
		}
	}

	// If no typed fields are set, just return the original settings string
	if !hasTyped {
		if t.options.Settings != nil {
			return *t.options.Settings
		}
		return ""
	}

	// Typed fields override user-provided settings
	t.applyTypedSettings(settings)

	result, err := json.Marshal(settings)
	if err != nil {
		t.logger.Warn("failed to marshal settings JSON", zap.Error(err))
		return ""
	}
	return string(result)
}

// hasTypedSettings reports whether any typed settings field (thinking, sandbox,
// checkpointing, tool config, include-hook-events) is set.
func (t *SubprocessCLITransport) hasTypedSettings() bool {
	return t.options.Thinking != nil ||
		t.options.Sandbox != nil ||
		t.options.EnableFileCheckpointing ||
		t.options.ToolConfig != nil ||
		t.options.IncludeHookEvents
}

// applyTypedSettings writes the typed settings fields onto the settings map,
// overriding any user-provided values for the same keys.
func (t *SubprocessCLITransport) applyTypedSettings(settings map[string]interface{}) {
	if t.options.Thinking != nil {
		settings["thinking"] = t.options.Thinking
	}
	if t.options.Sandbox != nil {
		settings["sandbox"] = t.options.Sandbox
	}
	if t.options.EnableFileCheckpointing {
		settings["enableFileCheckpointing"] = true
	}
	if t.options.ToolConfig != nil {
		settings["toolConfig"] = t.options.ToolConfig
	}
	if t.options.IncludeHookEvents {
		settings["includeHookEvents"] = true
	}
}
