package types

import (
	"context"
	"testing"
	"time"
)

// TestWithBuilders_AllMethods tests every With* builder method on ClaudeAgentOptions.
func TestWithBuilders_AllMethods(t *testing.T) {
	t.Parallel()

	t.Run("WithAllowedTools", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions()
		result := opts.WithAllowedTools("Bash", "Write")
		if result != opts {
			t.Error("should return same instance")
		}
		if len(opts.AllowedTools) != 2 || opts.AllowedTools[0] != "Bash" {
			t.Errorf("AllowedTools = %v, want [Bash Write]", opts.AllowedTools)
		}
	})

	t.Run("WithDisallowedTools", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions()
		result := opts.WithDisallowedTools("rm", "sudo")
		if result != opts {
			t.Error("should return same instance")
		}
		if len(opts.DisallowedTools) != 2 {
			t.Errorf("DisallowedTools = %v", opts.DisallowedTools)
		}
	})

	t.Run("WithSystemPrompt string", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithSystemPrompt("be helpful")
		if opts.SystemPrompt != "be helpful" {
			t.Errorf("SystemPrompt = %v", opts.SystemPrompt)
		}
	})

	t.Run("WithSystemPromptString", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithSystemPromptString("direct string")
		if opts.SystemPrompt != "direct string" {
			t.Errorf("SystemPrompt = %v", opts.SystemPrompt)
		}
	})

	t.Run("WithSystemPromptPreset", func(t *testing.T) {
		t.Parallel()
		preset := SystemPromptPreset{Type: "preset", Preset: "claude_code"}
		opts := NewClaudeAgentOptions().WithSystemPromptPreset(preset)
		p, ok := opts.SystemPrompt.(SystemPromptPreset)
		if !ok {
			t.Fatalf("SystemPrompt type = %T, want SystemPromptPreset", opts.SystemPrompt)
		}
		if p.Preset != "claude_code" {
			t.Errorf("Preset = %q", p.Preset)
		}
	})

	t.Run("WithMcpServers", func(t *testing.T) {
		t.Parallel()
		servers := map[string]interface{}{"server1": map[string]interface{}{"type": "stdio"}}
		opts := NewClaudeAgentOptions().WithMcpServers(servers)
		if opts.McpServers == nil {
			t.Error("McpServers should not be nil")
		}
	})

	t.Run("WithPermissionMode", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithPermissionMode(PermissionModePlan)
		if opts.PermissionMode == nil || *opts.PermissionMode != PermissionModePlan {
			t.Errorf("PermissionMode = %v", opts.PermissionMode)
		}
	})

	t.Run("WithPermissionPromptToolName", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithPermissionPromptToolName("custom_tool")
		if opts.PermissionPromptToolName == nil || *opts.PermissionPromptToolName != "custom_tool" {
			t.Error("PermissionPromptToolName not set correctly")
		}
	})

	t.Run("WithContinueConversation", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithContinueConversation(true)
		if !opts.ContinueConversation {
			t.Error("ContinueConversation should be true")
		}
	})

	t.Run("WithResume", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithResume("session-123")
		if opts.Resume == nil || *opts.Resume != "session-123" {
			t.Error("Resume not set correctly")
		}
	})

	t.Run("WithForkSession", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithForkSession(true)
		if !opts.ForkSession {
			t.Error("ForkSession should be true")
		}
	})

	t.Run("WithModel", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithModel("claude-3-opus")
		if opts.Model == nil || *opts.Model != "claude-3-opus" {
			t.Error("Model not set correctly")
		}
	})

	t.Run("WithMaxTurns", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithMaxTurns(5)
		if opts.MaxTurns == nil || *opts.MaxTurns != 5 {
			t.Error("MaxTurns not set correctly")
		}
	})

	t.Run("WithBaseURL", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithBaseURL("https://custom.api.com")
		if opts.BaseURL == nil || *opts.BaseURL != "https://custom.api.com" {
			t.Error("BaseURL not set correctly")
		}
	})

	t.Run("WithCWD", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithCWD("/tmp/work")
		if opts.CWD == nil || *opts.CWD != "/tmp/work" {
			t.Error("CWD not set correctly")
		}
	})

	t.Run("WithCLIPath", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithCLIPath("/usr/local/bin/claude")
		if opts.CLIPath == nil || *opts.CLIPath != "/usr/local/bin/claude" {
			t.Error("CLIPath not set correctly")
		}
	})

	t.Run("WithSettings", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithSettings("/path/to/settings.json")
		if opts.Settings == nil || *opts.Settings != "/path/to/settings.json" {
			t.Error("Settings not set correctly")
		}
	})

	t.Run("WithSettingSources", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithSettingSources(SettingSourceUser, SettingSourceProject)
		if len(opts.SettingSources) != 2 {
			t.Errorf("SettingSources length = %d, want 2", len(opts.SettingSources))
		}
	})

	t.Run("WithAddDirs", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithAddDirs("/dir1", "/dir2")
		if len(opts.AddDirs) != 2 {
			t.Errorf("AddDirs length = %d, want 2", len(opts.AddDirs))
		}
	})

	t.Run("WithEnv", func(t *testing.T) {
		t.Parallel()
		env := map[string]string{"KEY": "VALUE"}
		opts := NewClaudeAgentOptions().WithEnv(env)
		if opts.Env["KEY"] != "VALUE" {
			t.Errorf("Env[KEY] = %q", opts.Env["KEY"])
		}
	})

	t.Run("WithEnvVar", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithEnvVar("FOO", "BAR")
		if opts.Env["FOO"] != "BAR" {
			t.Errorf("Env[FOO] = %q", opts.Env["FOO"])
		}
	})

	t.Run("WithEnvVar on nil map", func(t *testing.T) {
		t.Parallel()
		opts := &ClaudeAgentOptions{}
		opts.WithEnvVar("X", "Y")
		if opts.Env["X"] != "Y" {
			t.Errorf("Env[X] = %q", opts.Env["X"])
		}
	})

	t.Run("WithExtraArgs", func(t *testing.T) {
		t.Parallel()
		val := "value"
		args := map[string]*string{"--flag": &val}
		opts := NewClaudeAgentOptions().WithExtraArgs(args)
		if opts.ExtraArgs["--flag"] == nil || *opts.ExtraArgs["--flag"] != "value" {
			t.Error("ExtraArgs not set correctly")
		}
	})

	t.Run("WithExtraArg", func(t *testing.T) {
		t.Parallel()
		val := "test"
		opts := NewClaudeAgentOptions().WithExtraArg("--custom", &val)
		if opts.ExtraArgs["--custom"] == nil || *opts.ExtraArgs["--custom"] != "test" {
			t.Error("ExtraArg not set correctly")
		}
	})

	t.Run("WithExtraArg on nil map", func(t *testing.T) {
		t.Parallel()
		opts := &ClaudeAgentOptions{}
		val := "x"
		opts.WithExtraArg("--key", &val)
		if opts.ExtraArgs["--key"] == nil || *opts.ExtraArgs["--key"] != "x" {
			t.Error("ExtraArg on nil map not set correctly")
		}
	})

	t.Run("WithMaxBufferSize", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithMaxBufferSize(2048)
		if opts.MaxBufferSize == nil || *opts.MaxBufferSize != 2048 {
			t.Error("MaxBufferSize not set correctly")
		}
	})

	t.Run("WithIncludePartialMessages", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithIncludePartialMessages(true)
		if !opts.IncludePartialMessages {
			t.Error("IncludePartialMessages should be true")
		}
	})

	t.Run("WithUser", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithUser("user-123")
		if opts.User == nil || *opts.User != "user-123" {
			t.Error("User not set correctly")
		}
	})

	t.Run("WithAgents", func(t *testing.T) {
		t.Parallel()
		agents := map[string]AgentDefinition{
			"reviewer": {Description: "reviews code", Prompt: "review"},
		}
		opts := NewClaudeAgentOptions().WithAgents(agents)
		if opts.Agents["reviewer"].Description != "reviews code" {
			t.Error("Agents not set correctly")
		}
	})

	t.Run("WithAgent", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithAgent("tester", AgentDefinition{Description: "tests", Prompt: "test"})
		if opts.Agents["tester"].Description != "tests" {
			t.Error("Agent not set correctly")
		}
	})

	t.Run("WithAgent on nil map", func(t *testing.T) {
		t.Parallel()
		opts := &ClaudeAgentOptions{}
		opts.WithAgent("x", AgentDefinition{Description: "d", Prompt: "p"})
		if opts.Agents["x"].Description != "d" {
			t.Error("Agent on nil map not set correctly")
		}
	})

	t.Run("WithSubagentExecution", func(t *testing.T) {
		t.Parallel()
		cfg := NewSubagentExecutionConfig()
		opts := NewClaudeAgentOptions().WithSubagentExecution(cfg)
		if opts.SubagentExecution != cfg {
			t.Error("SubagentExecution not set correctly")
		}
	})

	t.Run("WithCanUseTool", func(t *testing.T) {
		t.Parallel()
		called := false
		fn := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx ToolPermissionContext) (interface{}, error) {
			called = true
			return nil, nil
		}
		opts := NewClaudeAgentOptions().WithCanUseTool(fn)
		if opts.CanUseTool == nil {
			t.Fatal("CanUseTool should not be nil")
		}
		_, _ = opts.CanUseTool(context.Background(), "test", nil, ToolPermissionContext{})
		if !called {
			t.Error("CanUseTool callback not invoked")
		}
	})

	t.Run("WithToolCallbackTimeout", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithToolCallbackTimeout(30 * time.Second)
		if opts.ToolCallbackTimeout != 30*time.Second {
			t.Errorf("ToolCallbackTimeout = %v", opts.ToolCallbackTimeout)
		}
	})

	t.Run("WithHooks", func(t *testing.T) {
		t.Parallel()
		hooks := map[HookEvent][]HookMatcher{
			HookEventPreToolUse: {{Matcher: nil}},
		}
		opts := NewClaudeAgentOptions().WithHooks(hooks)
		if opts.Hooks[HookEventPreToolUse] == nil {
			t.Error("Hooks not set correctly")
		}
	})

	t.Run("WithHook", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithHook(HookEventStop, HookMatcher{})
		if len(opts.Hooks[HookEventStop]) != 1 {
			t.Error("Hook not added correctly")
		}
	})

	t.Run("WithHook on nil map", func(t *testing.T) {
		t.Parallel()
		opts := &ClaudeAgentOptions{}
		opts.WithHook(HookEventStop, HookMatcher{})
		if len(opts.Hooks[HookEventStop]) != 1 {
			t.Error("Hook on nil map not added correctly")
		}
	})

	t.Run("WithStderr", func(t *testing.T) {
		t.Parallel()
		called := false
		fn := func(line string) { called = true }
		opts := NewClaudeAgentOptions().WithStderr(fn)
		if opts.Stderr == nil {
			t.Fatal("Stderr should not be nil")
		}
		opts.Stderr("test line")
		if !called {
			t.Error("Stderr callback not invoked")
		}
	})

	t.Run("WithStderrLogFile", func(t *testing.T) {
		t.Parallel()
		path := "/var/log/test.log"
		opts := NewClaudeAgentOptions().WithStderrLogFile(&path)
		if opts.StderrLogFile == nil || *opts.StderrLogFile != path {
			t.Error("StderrLogFile not set correctly")
		}
	})

	t.Run("WithDefaultStderrLogFile", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithDefaultStderrLogFile()
		if opts.StderrLogFile == nil || *opts.StderrLogFile != "" {
			t.Error("DefaultStderrLogFile should set to empty string")
		}
	})

	t.Run("WithCustomStderrLogFile", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithCustomStderrLogFile("/custom/path.log")
		if opts.StderrLogFile == nil || *opts.StderrLogFile != "/custom/path.log" {
			t.Error("CustomStderrLogFile not set correctly")
		}
	})

	t.Run("WithVerbose", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithVerbose(true)
		if !opts.Verbose {
			t.Error("Verbose should be true")
		}
	})

	t.Run("WithDangerouslySkipPermissions", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithDangerouslySkipPermissions(true)
		if !opts.DangerouslySkipPermissions {
			t.Error("DangerouslySkipPermissions should be true")
		}
	})

	t.Run("WithAllowDangerouslySkipPermissions", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithAllowDangerouslySkipPermissions(true)
		if !opts.AllowDangerouslySkipPermissions {
			t.Error("AllowDangerouslySkipPermissions should be true")
		}
	})

	t.Run("WithEffort", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithEffort(EffortMax)
		if opts.Effort == nil || *opts.Effort != EffortMax {
			t.Error("Effort not set correctly")
		}
	})

	t.Run("WithThinking", func(t *testing.T) {
		t.Parallel()
		budget := 1000
		config := ThinkingConfig{Type: "enabled", BudgetTokens: &budget}
		opts := NewClaudeAgentOptions().WithThinking(config)
		if opts.Thinking == nil || opts.Thinking.Type != "enabled" {
			t.Error("Thinking not set correctly")
		}
	})

	t.Run("WithOutputFormat", func(t *testing.T) {
		t.Parallel()
		format := OutputFormat{Type: "json_schema", Schema: map[string]interface{}{"type": "object"}}
		opts := NewClaudeAgentOptions().WithOutputFormat(format)
		if opts.OutputFormat == nil || opts.OutputFormat.Type != "json_schema" {
			t.Error("OutputFormat not set correctly")
		}
	})

	t.Run("WithFallbackModel", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithFallbackModel("claude-3-haiku")
		if opts.FallbackModel == nil || *opts.FallbackModel != "claude-3-haiku" {
			t.Error("FallbackModel not set correctly")
		}
	})

	t.Run("WithEnableFileCheckpointing", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithEnableFileCheckpointing(true)
		if !opts.EnableFileCheckpointing {
			t.Error("EnableFileCheckpointing should be true")
		}
	})

	t.Run("WithSandbox", func(t *testing.T) {
		t.Parallel()
		enabled := true
		config := SandboxConfig{Enabled: &enabled}
		opts := NewClaudeAgentOptions().WithSandbox(config)
		if opts.Sandbox == nil || opts.Sandbox.Enabled == nil || !*opts.Sandbox.Enabled {
			t.Error("Sandbox not set correctly")
		}
	})

	t.Run("WithPersistSession", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithPersistSession(false)
		if opts.PersistSession == nil || *opts.PersistSession != false {
			t.Error("PersistSession not set correctly")
		}
	})

	t.Run("WithSessionID", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithSessionID("sess-abc")
		if opts.SessionID == nil || *opts.SessionID != "sess-abc" {
			t.Error("SessionID not set correctly")
		}
	})

	t.Run("WithPromptSuggestions", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithPromptSuggestions(true)
		if !opts.PromptSuggestions {
			t.Error("PromptSuggestions should be true")
		}
	})

	t.Run("WithSpawnProcess", func(t *testing.T) {
		t.Parallel()
		spawner := func(ctx context.Context, opts SpawnOptions) (SpawnedProcess, error) {
			return nil, nil
		}
		opts := NewClaudeAgentOptions().WithSpawnProcess(spawner)
		if opts.SpawnProcess == nil {
			t.Error("SpawnProcess should not be nil")
		}
	})

	t.Run("WithResumeSessionAt", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithResumeSessionAt("msg-uuid-123")
		if opts.ResumeSessionAt == nil || *opts.ResumeSessionAt != "msg-uuid-123" {
			t.Error("ResumeSessionAt not set correctly")
		}
	})

	t.Run("WithToolConfig", func(t *testing.T) {
		t.Parallel()
		timeout := 5000
		config := ToolConfig{Bash: &BashToolConfig{Timeout: &timeout}}
		opts := NewClaudeAgentOptions().WithToolConfig(config)
		if opts.ToolConfig == nil || opts.ToolConfig.Bash == nil || *opts.ToolConfig.Bash.Timeout != 5000 {
			t.Error("ToolConfig not set correctly")
		}
	})

	t.Run("WithTools", func(t *testing.T) {
		t.Parallel()
		tools := []string{"Bash", "Write"}
		opts := NewClaudeAgentOptions().WithTools(tools)
		if opts.Tools == nil {
			t.Error("Tools should not be nil")
		}
	})

	t.Run("WithDebugFile", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithDebugFile("/tmp/debug.log")
		if opts.DebugFile == nil || *opts.DebugFile != "/tmp/debug.log" {
			t.Error("DebugFile not set correctly")
		}
	})

	t.Run("WithStrictMcpConfig", func(t *testing.T) {
		t.Parallel()
		opts := NewClaudeAgentOptions().WithStrictMcpConfig(true)
		if !opts.StrictMcpConfig {
			t.Error("StrictMcpConfig should be true")
		}
	})
}

// TestSettingSourceConstants tests SettingSource constants.
func TestSettingSourceConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source SettingSource
		want   string
	}{
		{"user", SettingSourceUser, "user"},
		{"project", SettingSourceProject, "project"},
		{"local", SettingSourceLocal, "local"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if string(tt.source) != tt.want {
				t.Errorf("SettingSource = %q, want %q", tt.source, tt.want)
			}
		})
	}
}
