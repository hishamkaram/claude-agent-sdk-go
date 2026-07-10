package transport

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestBuildCommandArgs_DefaultPermissionModeDefersToCLI(t *testing.T) {
	t.Parallel()

	opts := types.NewClaudeAgentOptions().WithPermissionMode(types.PermissionModeDefault)
	args := newTestTransport(t, opts).buildCommandArgs()
	if hasFlag(args, "--permission-mode") {
		t.Fatalf("default permission mode must omit --permission-mode, got %v", args)
	}
}

func TestBuildCommandArgs_SessionAgent(t *testing.T) {
	t.Parallel()

	opts := types.NewClaudeAgentOptions().WithSessionAgent("reviewer")
	transport := newTestTransport(t, opts)
	args := transport.buildCommandArgs()

	val, found := flagValue(args, "--agent")
	if !found {
		t.Fatal("expected --agent flag")
	}
	if val != "reviewer" {
		t.Errorf("--agent = %q, want reviewer", val)
	}
}

// TestBuildCommandArgs_CombinedNewFlags verifies that multiple new flags can coexist.
func TestBuildCommandArgs_CombinedNewFlags(t *testing.T) {
	t.Parallel()

	budgetTokens := 4096
	opts := types.NewClaudeAgentOptions().
		WithEffort(types.EffortHigh).
		WithFallbackModel("claude-3-haiku").
		WithSessionID("abc-123").
		WithPersistSession(false).
		WithOutputFormat(types.OutputFormat{
			Type: "json_schema",
			Schema: map[string]interface{}{
				"type": "object",
			},
		}).
		WithThinking(types.ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: &budgetTokens,
		})

	transport := newTestTransport(t, opts)
	args := transport.buildCommandArgs()
	argsStr := strings.Join(args, " ")

	expectedFlags := []string{
		"--effort",
		"--fallback-model",
		"--session-id",
		"--no-session-persistence",
		"--json-schema",
		"--settings",
	}

	for _, flag := range expectedFlags {
		if !hasFlag(args, flag) {
			t.Errorf("flag %q not found in args: %s", flag, argsStr)
		}
	}

	// Verify specific values
	effortVal, _ := flagValue(args, "--effort")
	if effortVal != "high" {
		t.Errorf("--effort = %q, want %q", effortVal, "high")
	}

	fbVal, _ := flagValue(args, "--fallback-model")
	if fbVal != "claude-3-haiku" {
		t.Errorf("--fallback-model = %q, want %q", fbVal, "claude-3-haiku")
	}

	sidVal, _ := flagValue(args, "--session-id")
	if sidVal != "abc-123" {
		t.Errorf("--session-id = %q, want %q", sidVal, "abc-123")
	}
}

// ===== Phase D: US1 — Custom Process Spawner Transport Tests =====

// TestBuildCommandArgs_ResumeSessionAt tests --resume-session-at flag generation.
func TestBuildCommandArgs_ResumeSessionAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		msgID     *string
		wantFlag  bool
		wantValue string
	}{
		{
			name:      "set",
			msgID:     strPtr("msg-uuid-456"),
			wantFlag:  true,
			wantValue: "msg-uuid-456",
		},
		{
			name:     "nil",
			msgID:    nil,
			wantFlag: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := types.NewClaudeAgentOptions()
			opts.ResumeSessionAt = tt.msgID
			tr := newTestTransport(t, opts)
			args := tr.buildCommandArgs()

			if tt.wantFlag {
				val, found := flagValue(args, "--resume-session-at")
				if !found {
					t.Error("expected --resume-session-at flag")
				}
				if val != tt.wantValue {
					t.Errorf("--resume-session-at = %q, want %q", val, tt.wantValue)
				}
			} else {
				if hasFlag(args, "--resume-session-at") {
					t.Error("unexpected --resume-session-at flag")
				}
			}
		})
	}
}

// TestBuildCommandArgs_Tools tests --tools flag generation.
func TestBuildCommandArgs_Tools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tools     interface{}
		wantFlag  bool
		wantValue string
	}{
		{
			name:      "string slice",
			tools:     []string{"Bash", "Read", "Write"},
			wantFlag:  true,
			wantValue: "Bash,Read,Write",
		},
		{
			name:     "nil",
			tools:    nil,
			wantFlag: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := types.NewClaudeAgentOptions()
			opts.Tools = tt.tools
			tr := newTestTransport(t, opts)
			args := tr.buildCommandArgs()

			if tt.wantFlag {
				val, found := flagValue(args, "--tools")
				if !found {
					t.Error("expected --tools flag")
				}
				if val != tt.wantValue {
					t.Errorf("--tools = %q, want %q", val, tt.wantValue)
				}
			} else {
				if hasFlag(args, "--tools") {
					t.Error("unexpected --tools flag")
				}
			}
		})
	}
}

// TestBuildCommandArgs_DebugFile tests --debug-file flag generation.
func TestBuildCommandArgs_DebugFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		debugFile *string
		wantFlag  bool
		wantValue string
	}{
		{
			name:      "set",
			debugFile: strPtr("/tmp/debug.log"),
			wantFlag:  true,
			wantValue: "/tmp/debug.log",
		},
		{
			name:      "nil",
			debugFile: nil,
			wantFlag:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := types.NewClaudeAgentOptions()
			opts.DebugFile = tt.debugFile
			tr := newTestTransport(t, opts)
			args := tr.buildCommandArgs()

			if tt.wantFlag {
				val, found := flagValue(args, "--debug-file")
				if !found {
					t.Error("expected --debug-file flag")
				}
				if val != tt.wantValue {
					t.Errorf("--debug-file = %q, want %q", val, tt.wantValue)
				}
			} else {
				if hasFlag(args, "--debug-file") {
					t.Error("unexpected --debug-file flag")
				}
			}
		})
	}
}

// TestBuildCommandArgs_StrictMcpConfig tests --strict-mcp-config flag generation.
func TestBuildCommandArgs_StrictMcpConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		strict   bool
		wantFlag bool
	}{
		{
			name:     "enabled",
			strict:   true,
			wantFlag: true,
		},
		{
			name:     "disabled",
			strict:   false,
			wantFlag: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := types.NewClaudeAgentOptions()
			opts.StrictMcpConfig = tt.strict
			tr := newTestTransport(t, opts)
			args := tr.buildCommandArgs()

			if tt.wantFlag != hasFlag(args, "--strict-mcp-config") {
				if tt.wantFlag {
					t.Error("expected --strict-mcp-config flag")
				} else {
					t.Error("unexpected --strict-mcp-config flag")
				}
			}
		})
	}
}

// TestBuildSettingsJSON_ToolConfig tests toolConfig in settings JSON.
func TestBuildSettingsJSON_ToolConfig(t *testing.T) {
	t.Parallel()

	timeout := 30000
	cmd := "/bin/bash"
	display := 1

	tests := []struct {
		name       string
		toolConfig *types.ToolConfig
		wantKey    bool
	}{
		{
			name: "full tool config",
			toolConfig: &types.ToolConfig{
				Bash:     &types.BashToolConfig{Timeout: &timeout, Command: &cmd},
				Computer: &types.ComputerToolConfig{Display: &display},
			},
			wantKey: true,
		},
		{
			name:       "nil tool config",
			toolConfig: nil,
			wantKey:    false,
		},
		{
			name: "bash only",
			toolConfig: &types.ToolConfig{
				Bash: &types.BashToolConfig{Timeout: &timeout},
			},
			wantKey: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := types.NewClaudeAgentOptions()
			opts.ToolConfig = tt.toolConfig
			tr := newTestTransport(t, opts)
			settingsJSON := tr.buildSettingsJSON()

			if tt.wantKey {
				if settingsJSON == "" {
					t.Fatal("expected non-empty settings JSON")
				}
				var settings map[string]interface{}
				if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
					t.Fatalf("Failed to parse settings JSON: %v", err)
				}
				if _, exists := settings["toolConfig"]; !exists {
					t.Error("expected 'toolConfig' key in settings JSON")
				}
			} else {
				if settingsJSON != "" {
					var settings map[string]interface{}
					if err := json.Unmarshal([]byte(settingsJSON), &settings); err == nil {
						if _, exists := settings["toolConfig"]; exists {
							t.Error("unexpected 'toolConfig' key in settings JSON")
						}
					}
				}
			}
		})
	}
}

// TestBuildSettingsJSON_ToolConfig_NilSubStructs verifies nil sub-structs are omitted.
func TestBuildSettingsJSON_ToolConfig_NilSubStructs(t *testing.T) {
	t.Parallel()

	opts := types.NewClaudeAgentOptions()
	opts.ToolConfig = &types.ToolConfig{
		Bash:     nil,
		Computer: nil,
	}
	tr := newTestTransport(t, opts)
	settingsJSON := tr.buildSettingsJSON()

	if settingsJSON == "" {
		t.Fatal("expected non-empty settings JSON")
	}

	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		t.Fatalf("Failed to parse settings JSON: %v", err)
	}

	toolConfig, ok := settings["toolConfig"].(map[string]interface{})
	if !ok {
		t.Fatal("toolConfig should be a map")
	}

	if _, exists := toolConfig["bash"]; exists {
		t.Error("nil Bash should not appear in JSON")
	}
	if _, exists := toolConfig["computer"]; exists {
		t.Error("nil Computer should not appear in JSON")
	}
}

// TestBuildCommandArgs_ToolsAlongsideAllowedTools verifies both are independent.
func TestBuildCommandArgs_ToolsAlongsideAllowedTools(t *testing.T) {
	t.Parallel()

	opts := types.NewClaudeAgentOptions()
	opts.AllowedTools = []string{"Bash"}
	opts.Tools = []string{"Read", "Write"}
	tr := newTestTransport(t, opts)
	args := tr.buildCommandArgs()

	// --tools should be present regardless of AllowedTools being set
	toolsVal, toolsFound := flagValue(args, "--tools")
	if !toolsFound {
		t.Error("expected --tools flag")
	}
	if toolsVal != "Read,Write" {
		t.Errorf("--tools = %q, want %q", toolsVal, "Read,Write")
	}
}

// TestBuildCommandArgs_ResumeSessionAtWithoutResume verifies they work independently.
func TestBuildCommandArgs_ResumeSessionAtWithoutResume(t *testing.T) {
	t.Parallel()

	opts := types.NewClaudeAgentOptions()
	opts.ResumeSessionAt = strPtr("msg-uuid-789")
	// Resume is nil — only ResumeSessionAt is set
	tr := newTestTransport(t, opts)
	args := tr.buildCommandArgs()

	// ResumeSessionAt should be present
	val, found := flagValue(args, "--resume-session-at")
	if !found {
		t.Error("expected --resume-session-at flag")
	}
	if val != "msg-uuid-789" {
		t.Errorf("--resume-session-at = %q, want %q", val, "msg-uuid-789")
	}

	// --resume should NOT be present
	if hasFlag(args, "--resume") {
		t.Error("unexpected --resume flag when only ResumeSessionAt is set")
	}
}

// TestBuildCommandArgs_AgentDefinitionNewFields verifies Phase D fields are serialized.
func TestBuildCommandArgs_AgentDefinitionNewFields(t *testing.T) {
	t.Parallel()

	reminder := "critical reminder text"
	background := true
	effort := types.EffortXHigh
	permissionMode := types.PermissionModePlan
	memory := types.SettingSourceProject
	initialPrompt := "start here"
	isolation := types.AgentIsolationWorktree
	color := types.AgentColorCyan
	matcher := "Edit|Write"
	opts := types.NewClaudeAgentOptions()
	opts.Agents = map[string]types.AgentDefinition{
		"test-agent": {
			Description:     "test agent",
			Prompt:          "do stuff",
			DisallowedTools: []string{"Bash", "Write"},
			McpServers:      []interface{}{"server1", map[string]interface{}{"name": "server2"}},
			Hooks: map[types.HookEvent][]types.AgentHookMatcher{
				types.HookEventPostToolUse: {
					{
						Matcher: &matcher,
						Hooks: []types.AgentHookHandler{
							{"type": "command", "command": "./scripts/lint.sh"},
						},
					},
				},
			},
			Skills:                 []string{"skill-a", "skill-b"},
			InitialPrompt:          &initialPrompt,
			CriticalSystemReminder: &reminder,
			Background:             &background,
			Effort:                 &effort,
			PermissionMode:         &permissionMode,
			Memory:                 &memory,
			Isolation:              &isolation,
			Color:                  &color,
		},
	}
	tr := newTestTransport(t, opts)
	args := tr.buildCommandArgs()

	agentsVal, found := flagValue(args, "--agents")
	if !found {
		t.Fatal("expected --agents flag")
	}

	var agentsMap map[string]map[string]interface{}
	if err := json.Unmarshal([]byte(agentsVal), &agentsMap); err != nil {
		t.Fatalf("failed to parse --agents JSON: %v", err)
	}

	agent, ok := agentsMap["test-agent"]
	if !ok {
		t.Fatal("test-agent not found in agents JSON")
	}

	// Verify new fields are present
	if _, ok := agent["disallowedTools"]; !ok {
		t.Error("disallowedTools missing from agent JSON")
	}
	if _, ok := agent["mcpServers"]; !ok {
		t.Error("mcpServers missing from agent JSON")
	}
	if _, ok := agent["hooks"]; !ok {
		t.Error("hooks missing from agent JSON")
	}
	if _, ok := agent["skills"]; !ok {
		t.Error("skills missing from agent JSON")
	}
	if val := agent["initialPrompt"]; val != "start here" {
		t.Errorf("initialPrompt = %v, want start here", val)
	}
	if val := agent["background"]; val != true {
		t.Errorf("background = %v, want true", val)
	}
	if val := agent["effort"]; val != "xhigh" {
		t.Errorf("effort = %v, want xhigh", val)
	}
	if val := agent["permissionMode"]; val != "plan" {
		t.Errorf("permissionMode = %v, want plan", val)
	}
	if val := agent["memory"]; val != "project" {
		t.Errorf("memory = %v, want project", val)
	}
	if val := agent["isolation"]; val != "worktree" {
		t.Errorf("isolation = %v, want worktree", val)
	}
	if val := agent["color"]; val != "cyan" {
		t.Errorf("color = %v, want cyan", val)
	}
	if val, ok := agent["criticalSystemReminder_EXPERIMENTAL"]; !ok {
		t.Error("criticalSystemReminder_EXPERIMENTAL missing from agent JSON")
	} else if val != "critical reminder text" {
		t.Errorf("criticalSystemReminder_EXPERIMENTAL = %v, want %q", val, "critical reminder text")
	}
}

// ===== Phase D: Custom Spawner Integration Tests (T004-T, T005-T) =====

// mockSpawnedProcess implements types.SpawnedProcess for testing.
