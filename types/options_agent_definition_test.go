package types

import (
	"encoding/json"
	"reflect"
	"testing"
)

// ===== Phase D: US3 — AgentDefinition Parity Tests =====

// TestAgentDefinition_NewFields_JSONRoundtrip tests new AgentDefinition fields.
func TestAgentDefinition_NewFields_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	reminder := "Always verify before destructive operations"
	background := true
	effort := EffortHigh
	permissionMode := PermissionModePlan
	memory := SettingSourceProject
	initialPrompt := "Start by reading README.md"
	isolation := AgentIsolationWorktree
	color := AgentColorPurple
	matcher := "Bash"

	tests := []struct {
		name  string
		input AgentDefinition
		check func(t *testing.T, got AgentDefinition)
	}{
		{
			name: "disallowed tools",
			input: AgentDefinition{
				Description:     "test",
				Prompt:          "test",
				DisallowedTools: []string{"Bash", "Write"},
			},
			check: func(t *testing.T, got AgentDefinition) {
				if !reflect.DeepEqual(got.DisallowedTools, []string{"Bash", "Write"}) {
					t.Errorf("DisallowedTools = %v, want [Bash Write]", got.DisallowedTools)
				}
			},
		},
		{
			name: "memory",
			input: AgentDefinition{
				Description: "test",
				Prompt:      "test",
				Memory:      &memory,
			},
			check: func(t *testing.T, got AgentDefinition) {
				if got.Memory == nil {
					t.Fatal("Memory should not be nil")
				}
				if *got.Memory != SettingSourceProject {
					t.Errorf("Memory = %v, want project", *got.Memory)
				}
			},
		},
		{
			name: "mcp servers string refs",
			input: AgentDefinition{
				Description: "test",
				Prompt:      "test",
				McpServers:  []interface{}{"server-a", "server-b"},
			},
			check: func(t *testing.T, got AgentDefinition) {
				if len(got.McpServers) != 2 {
					t.Fatalf("McpServers len = %d, want 2", len(got.McpServers))
				}
				if got.McpServers[0] != "server-a" {
					t.Errorf("McpServers[0] = %v, want server-a", got.McpServers[0])
				}
			},
		},
		{
			name: "mcp servers mixed",
			input: AgentDefinition{
				Description: "test",
				Prompt:      "test",
				McpServers: []interface{}{
					"existing-server",
					map[string]interface{}{"command": "npx", "args": []interface{}{"-y", "@mcp/server"}},
				},
			},
			check: func(t *testing.T, got AgentDefinition) {
				if len(got.McpServers) != 2 {
					t.Fatalf("McpServers len = %d, want 2", len(got.McpServers))
				}
				if got.McpServers[0] != "existing-server" {
					t.Errorf("McpServers[0] = %v, want existing-server", got.McpServers[0])
				}
				inline, ok := got.McpServers[1].(map[string]interface{})
				if !ok {
					t.Fatalf("McpServers[1] should be map, got %T", got.McpServers[1])
				}
				if inline["command"] != "npx" {
					t.Errorf("McpServers[1].command = %v, want npx", inline["command"])
				}
			},
		},
		{
			name: "skills",
			input: AgentDefinition{
				Description: "test",
				Prompt:      "test",
				Skills:      []string{"security-audit", "code-review"},
			},
			check: func(t *testing.T, got AgentDefinition) {
				if !reflect.DeepEqual(got.Skills, []string{"security-audit", "code-review"}) {
					t.Errorf("Skills = %v, want [security-audit code-review]", got.Skills)
				}
			},
		},
		{
			name: "hooks",
			input: AgentDefinition{
				Description: "test",
				Prompt:      "test",
				Hooks: map[HookEvent][]AgentHookMatcher{
					HookEventPreToolUse: {
						{
							Matcher: &matcher,
							Hooks: []AgentHookHandler{
								{
									"type":    "command",
									"command": "./scripts/validate-bash.sh",
								},
							},
						},
					},
				},
			},
			check: func(t *testing.T, got AgentDefinition) {
				groups := got.Hooks[HookEventPreToolUse]
				if len(groups) != 1 {
					t.Fatalf("PreToolUse hook groups len = %d, want 1", len(groups))
				}
				if groups[0].Matcher == nil || *groups[0].Matcher != "Bash" {
					t.Errorf("Matcher = %v, want Bash", groups[0].Matcher)
				}
				if len(groups[0].Hooks) != 1 {
					t.Fatalf("hook handlers len = %d, want 1", len(groups[0].Hooks))
				}
				if groups[0].Hooks[0]["type"] != "command" {
					t.Errorf("hook type = %v, want command", groups[0].Hooks[0]["type"])
				}
			},
		},
		{
			name: "initial prompt isolation and color",
			input: AgentDefinition{
				Description:   "test",
				Prompt:        "test",
				InitialPrompt: &initialPrompt,
				Isolation:     &isolation,
				Color:         &color,
			},
			check: func(t *testing.T, got AgentDefinition) {
				if got.InitialPrompt == nil || *got.InitialPrompt != initialPrompt {
					t.Errorf("InitialPrompt = %v, want %q", got.InitialPrompt, initialPrompt)
				}
				if got.Isolation == nil || *got.Isolation != AgentIsolationWorktree {
					t.Errorf("Isolation = %v, want worktree", got.Isolation)
				}
				if got.Color == nil || *got.Color != AgentColorPurple {
					t.Errorf("Color = %v, want purple", got.Color)
				}
			},
		},
		{
			name: "critical system reminder",
			input: AgentDefinition{
				Description:            "test",
				Prompt:                 "test",
				CriticalSystemReminder: &reminder,
			},
			check: func(t *testing.T, got AgentDefinition) {
				if got.CriticalSystemReminder == nil {
					t.Fatal("CriticalSystemReminder should not be nil")
				}
				if *got.CriticalSystemReminder != reminder {
					t.Errorf("CriticalSystemReminder = %q, want %q", *got.CriticalSystemReminder, reminder)
				}
			},
		},
		{
			name: "background effort and permission mode",
			input: AgentDefinition{
				Description:    "test",
				Prompt:         "test",
				Background:     &background,
				Effort:         &effort,
				PermissionMode: &permissionMode,
			},
			check: func(t *testing.T, got AgentDefinition) {
				if got.Background == nil || *got.Background != true {
					t.Errorf("Background = %v, want true", got.Background)
				}
				if got.Effort == nil || *got.Effort != EffortHigh {
					t.Errorf("Effort = %v, want high", got.Effort)
				}
				if got.PermissionMode == nil || *got.PermissionMode != PermissionModePlan {
					t.Errorf("PermissionMode = %v, want plan", got.PermissionMode)
				}
			},
		},
		{
			name: "all new fields together",
			input: AgentDefinition{
				Description:     "security reviewer",
				Prompt:          "audit for vulnerabilities",
				Tools:           []string{"Read", "Grep"},
				DisallowedTools: []string{"Bash", "Write", "Edit"},
				McpServers:      []interface{}{"scanner"},
				Hooks: map[HookEvent][]AgentHookMatcher{
					HookEventPostToolUse: {
						{
							Matcher: &matcher,
							Hooks: []AgentHookHandler{
								{"type": "command", "command": "./scripts/post-tool.sh"},
							},
						},
					},
				},
				Skills:                 []string{"security-audit"},
				InitialPrompt:          &initialPrompt,
				CriticalSystemReminder: &reminder,
				Background:             &background,
				Effort:                 &effort,
				PermissionMode:         &permissionMode,
				Memory:                 &memory,
				Isolation:              &isolation,
				Color:                  &color,
			},
			check: func(t *testing.T, got AgentDefinition) {
				if len(got.DisallowedTools) != 3 {
					t.Errorf("DisallowedTools len = %d, want 3", len(got.DisallowedTools))
				}
				if len(got.McpServers) != 1 {
					t.Errorf("McpServers len = %d, want 1", len(got.McpServers))
				}
				if len(got.Skills) != 1 {
					t.Errorf("Skills len = %d, want 1", len(got.Skills))
				}
				if len(got.Hooks[HookEventPostToolUse]) != 1 {
					t.Errorf("PostToolUse hooks len = %d, want 1", len(got.Hooks[HookEventPostToolUse]))
				}
				if got.InitialPrompt == nil || *got.InitialPrompt != initialPrompt {
					t.Errorf("InitialPrompt = %v, want %q", got.InitialPrompt, initialPrompt)
				}
				if got.CriticalSystemReminder == nil {
					t.Error("CriticalSystemReminder should not be nil")
				}
				if got.Background == nil || *got.Background != true {
					t.Errorf("Background = %v, want true", got.Background)
				}
				if got.Effort == nil || *got.Effort != EffortHigh {
					t.Errorf("Effort = %v, want high", got.Effort)
				}
				if got.PermissionMode == nil || *got.PermissionMode != PermissionModePlan {
					t.Errorf("PermissionMode = %v, want plan", got.PermissionMode)
				}
				if got.Memory == nil || *got.Memory != SettingSourceProject {
					t.Errorf("Memory = %v, want project", got.Memory)
				}
				if got.Isolation == nil || *got.Isolation != AgentIsolationWorktree {
					t.Errorf("Isolation = %v, want worktree", got.Isolation)
				}
				if got.Color == nil || *got.Color != AgentColorPurple {
					t.Errorf("Color = %v, want purple", got.Color)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			var got AgentDefinition
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			tt.check(t, got)
		})
	}
}

// TestAgentDefinition_CurrentWireKeys verifies that AgentDefinition marshals
// the current documented camelCase keys for the --agents CLI payload.
func TestAgentDefinition_CurrentWireKeys(t *testing.T) {
	t.Parallel()

	reminder := "remember"
	background := true
	maxTurns := 4
	effort := EffortXHigh
	permissionMode := PermissionModePlan
	memory := SettingSourceLocal
	initialPrompt := "Start here"
	isolation := AgentIsolationWorktree
	color := AgentColorCyan
	matcher := "Edit|Write"
	legacyMode := SubagentExecutionModeParallel
	legacyTimeout := 30.0

	agent := AgentDefinition{
		Description:     "reviewer",
		Prompt:          "review code",
		DisallowedTools: []string{"Bash"},
		MaxTurns:        &maxTurns,
		McpServers:      []interface{}{"scanner"},
		Hooks: map[HookEvent][]AgentHookMatcher{
			HookEventPostToolUse: {
				{
					Matcher: &matcher,
					Hooks: []AgentHookHandler{
						{"type": "command", "command": "./scripts/lint.sh"},
					},
				},
			},
		},
		InitialPrompt:          &initialPrompt,
		CriticalSystemReminder: &reminder,
		Background:             &background,
		Effort:                 &effort,
		PermissionMode:         &permissionMode,
		Memory:                 &memory,
		Isolation:              &isolation,
		Color:                  &color,
		ExecutionMode:          &legacyMode,
		Timeout:                &legacyTimeout,
	}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal raw failed: %v", err)
	}

	for _, key := range []string{
		"disallowedTools",
		"maxTurns",
		"mcpServers",
		"hooks",
		"initialPrompt",
		"criticalSystemReminder_EXPERIMENTAL",
		"background",
		"effort",
		"permissionMode",
		"memory",
		"isolation",
		"color",
	} {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing current AgentDefinition key %q in %s", key, data)
		}
	}

	for _, legacyKey := range []string{"disallowed_tools", "max_turns", "mcp_servers", "execution_mode", "timeout"} {
		if _, ok := raw[legacyKey]; ok {
			t.Errorf("legacy key %q should not be emitted in %s", legacyKey, data)
		}
	}
}

// TestAgentDefinition_LegacySnakeCaseUnmarshal preserves compatibility with
// persisted/configured JSON written with the SDK's older snake_case tags.
func TestAgentDefinition_LegacySnakeCaseUnmarshal(t *testing.T) {
	t.Parallel()

	data := []byte(`{
		"description":"legacy",
		"prompt":"legacy prompt",
		"disallowed_tools":["Bash"],
		"mcp_servers":["scanner"],
		"max_turns":7,
		"execution_mode":"parallel",
		"timeout":12.5
	}`)

	var got AgentDefinition
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(got.DisallowedTools, []string{"Bash"}) {
		t.Errorf("DisallowedTools = %v, want [Bash]", got.DisallowedTools)
	}
	if len(got.McpServers) != 1 || got.McpServers[0] != "scanner" {
		t.Errorf("McpServers = %v, want [scanner]", got.McpServers)
	}
	if got.MaxTurns == nil || *got.MaxTurns != 7 {
		t.Errorf("MaxTurns = %v, want 7", got.MaxTurns)
	}
	if got.ExecutionMode == nil || *got.ExecutionMode != SubagentExecutionModeParallel {
		t.Errorf("ExecutionMode = %v, want parallel", got.ExecutionMode)
	}
	if got.Timeout == nil || *got.Timeout != 12.5 {
		t.Errorf("Timeout = %v, want 12.5", got.Timeout)
	}
}

// TestAgentDefinition_CriticalSystemReminder_JSONKey verifies the experimental JSON key.
func TestAgentDefinition_CriticalSystemReminder_JSONKey(t *testing.T) {
	t.Parallel()

	reminder := "test reminder"
	agent := AgentDefinition{
		Description:            "test",
		Prompt:                 "test",
		CriticalSystemReminder: &reminder,
	}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if _, exists := raw["criticalSystemReminder_EXPERIMENTAL"]; !exists {
		t.Error("Expected JSON key 'criticalSystemReminder_EXPERIMENTAL', not found in output")
	}
}
