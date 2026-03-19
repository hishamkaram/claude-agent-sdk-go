package transport

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// newTestTransport creates a SubprocessCLITransport for unit tests without
// actually starting a subprocess. This allows testing buildCommandArgs() and
// buildSettingsJSON() in isolation.
func newTestTransport(t *testing.T, opts *types.ClaudeAgentOptions) *SubprocessCLITransport {
	t.Helper()
	return NewSubprocessCLITransport(
		"/usr/local/bin/claude",
		"",
		nil,
		log.NewLogger(false),
		"",
		opts,
	)
}

// flagValue returns the value immediately following flag in args, or ("", false).
func flagValue(args []string, flag string) (string, bool) {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

// hasFlag checks whether flag appears anywhere in args.
func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

// TestBuildCommandArgs_Effort tests --effort flag generation.
func TestBuildCommandArgs_Effort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		effort     *types.EffortLevel
		wantFlag   bool
		wantValue  string
	}{
		{
			name:     "effort high",
			effort:   effortPtr(types.EffortHigh),
			wantFlag: true,
			wantValue: "high",
		},
		{
			name:     "effort low",
			effort:   effortPtr(types.EffortLow),
			wantFlag: true,
			wantValue: "low",
		},
		{
			name:     "effort medium",
			effort:   effortPtr(types.EffortMedium),
			wantFlag: true,
			wantValue: "medium",
		},
		{
			name:     "effort max",
			effort:   effortPtr(types.EffortMax),
			wantFlag: true,
			wantValue: "max",
		},
		{
			name:     "effort nil — flag absent",
			effort:   nil,
			wantFlag: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.NewClaudeAgentOptions()
			if tt.effort != nil {
				opts.WithEffort(*tt.effort)
			}

			transport := newTestTransport(t, opts)
			args := transport.buildCommandArgs()

			if got := hasFlag(args, "--effort"); got != tt.wantFlag {
				t.Errorf("--effort flag present = %v, want %v", got, tt.wantFlag)
			}

			if tt.wantFlag {
				val, ok := flagValue(args, "--effort")
				if !ok {
					t.Fatal("--effort flag present but no value follows")
				}
				if val != tt.wantValue {
					t.Errorf("--effort value = %q, want %q", val, tt.wantValue)
				}
			}
		})
	}
}

// TestBuildCommandArgs_FallbackModel tests --fallback-model flag generation.
func TestBuildCommandArgs_FallbackModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		model     *string
		wantFlag  bool
		wantValue string
	}{
		{
			name:      "fallback model set",
			model:     strPtr("claude-3-haiku"),
			wantFlag:  true,
			wantValue: "claude-3-haiku",
		},
		{
			name:     "fallback model nil — flag absent",
			model:    nil,
			wantFlag: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.NewClaudeAgentOptions()
			if tt.model != nil {
				opts.WithFallbackModel(*tt.model)
			}

			transport := newTestTransport(t, opts)
			args := transport.buildCommandArgs()

			if got := hasFlag(args, "--fallback-model"); got != tt.wantFlag {
				t.Errorf("--fallback-model flag present = %v, want %v", got, tt.wantFlag)
			}

			if tt.wantFlag {
				val, ok := flagValue(args, "--fallback-model")
				if !ok {
					t.Fatal("--fallback-model flag present but no value follows")
				}
				if val != tt.wantValue {
					t.Errorf("--fallback-model value = %q, want %q", val, tt.wantValue)
				}
			}
		})
	}
}

// TestBuildCommandArgs_SessionID tests --session-id flag generation.
func TestBuildCommandArgs_SessionID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sessionID *string
		wantFlag  bool
		wantValue string
	}{
		{
			name:      "session ID set",
			sessionID: strPtr("abc-123"),
			wantFlag:  true,
			wantValue: "abc-123",
		},
		{
			name:      "session ID with UUID",
			sessionID: strPtr("8587b432-e504-42c8-b9a7-e3fd0b4b2c60"),
			wantFlag:  true,
			wantValue: "8587b432-e504-42c8-b9a7-e3fd0b4b2c60",
		},
		{
			name:     "session ID nil — flag absent",
			sessionID: nil,
			wantFlag:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.NewClaudeAgentOptions()
			if tt.sessionID != nil {
				opts.WithSessionID(*tt.sessionID)
			}

			transport := newTestTransport(t, opts)
			args := transport.buildCommandArgs()

			if got := hasFlag(args, "--session-id"); got != tt.wantFlag {
				t.Errorf("--session-id flag present = %v, want %v", got, tt.wantFlag)
			}

			if tt.wantFlag {
				val, ok := flagValue(args, "--session-id")
				if !ok {
					t.Fatal("--session-id flag present but no value follows")
				}
				if val != tt.wantValue {
					t.Errorf("--session-id value = %q, want %q", val, tt.wantValue)
				}
			}
		})
	}
}

// TestBuildCommandArgs_PersistSession tests --no-session-persistence flag generation.
func TestBuildCommandArgs_PersistSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		persist  *bool
		wantFlag bool
	}{
		{
			name:     "PersistSession false — flag present",
			persist:  boolPtr(false),
			wantFlag: true,
		},
		{
			name:     "PersistSession true — flag absent",
			persist:  boolPtr(true),
			wantFlag: false,
		},
		{
			name:     "PersistSession nil — flag absent",
			persist:  nil,
			wantFlag: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.NewClaudeAgentOptions()
			if tt.persist != nil {
				opts.WithPersistSession(*tt.persist)
			}

			transport := newTestTransport(t, opts)
			args := transport.buildCommandArgs()

			if got := hasFlag(args, "--no-session-persistence"); got != tt.wantFlag {
				t.Errorf("--no-session-persistence flag present = %v, want %v", got, tt.wantFlag)
			}
		})
	}
}

// TestBuildCommandArgs_OutputFormat tests --json-schema flag generation.
func TestBuildCommandArgs_OutputFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		format   *types.OutputFormat
		wantFlag bool
	}{
		{
			name: "output format with schema",
			format: &types.OutputFormat{
				Type: "json_schema",
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"answer": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
			wantFlag: true,
		},
		{
			name:     "output format nil — flag absent",
			format:   nil,
			wantFlag: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.NewClaudeAgentOptions()
			if tt.format != nil {
				opts.WithOutputFormat(*tt.format)
			}

			transport := newTestTransport(t, opts)
			args := transport.buildCommandArgs()

			if got := hasFlag(args, "--json-schema"); got != tt.wantFlag {
				t.Errorf("--json-schema flag present = %v, want %v", got, tt.wantFlag)
			}

			if tt.wantFlag {
				val, ok := flagValue(args, "--json-schema")
				if !ok {
					t.Fatal("--json-schema flag present but no value follows")
				}

				// Verify it is valid JSON
				var parsed map[string]interface{}
				if err := json.Unmarshal([]byte(val), &parsed); err != nil {
					t.Fatalf("--json-schema value is not valid JSON: %v", err)
				}

				// Verify type field is present
				if parsed["type"] != "json_schema" {
					t.Errorf("parsed type = %v, want json_schema", parsed["type"])
				}
			}
		})
	}
}

// TestBuildCommandArgs_SettingsThinking tests --settings flag with thinking config.
func TestBuildCommandArgs_SettingsThinking(t *testing.T) {
	t.Parallel()

	budgetTokens := 8192
	opts := types.NewClaudeAgentOptions().
		WithThinking(types.ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: &budgetTokens,
		})

	transport := newTestTransport(t, opts)
	args := transport.buildCommandArgs()

	if !hasFlag(args, "--settings") {
		t.Fatal("--settings flag not found when Thinking is set")
	}

	val, ok := flagValue(args, "--settings")
	if !ok {
		t.Fatal("--settings flag present but no value follows")
	}

	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(val), &settings); err != nil {
		t.Fatalf("--settings value is not valid JSON: %v", err)
	}

	thinkingRaw, ok := settings["thinking"]
	if !ok {
		t.Fatal("thinking key not found in settings JSON")
	}

	// Re-marshal and unmarshal to get typed access
	thinkingJSON, err := json.Marshal(thinkingRaw)
	if err != nil {
		t.Fatalf("failed to re-marshal thinking: %v", err)
	}

	var thinking types.ThinkingConfig
	if err := json.Unmarshal(thinkingJSON, &thinking); err != nil {
		t.Fatalf("failed to unmarshal ThinkingConfig: %v", err)
	}

	if thinking.Type != "enabled" {
		t.Errorf("thinking.Type = %q, want %q", thinking.Type, "enabled")
	}
	if thinking.BudgetTokens == nil || *thinking.BudgetTokens != 8192 {
		t.Errorf("thinking.BudgetTokens = %v, want 8192", thinking.BudgetTokens)
	}
}

// TestBuildCommandArgs_SettingsSandbox tests --settings flag with sandbox config.
func TestBuildCommandArgs_SettingsSandbox(t *testing.T) {
	t.Parallel()

	enabled := true
	opts := types.NewClaudeAgentOptions().
		WithSandbox(types.SandboxConfig{
			Enabled: &enabled,
		})

	transport := newTestTransport(t, opts)
	args := transport.buildCommandArgs()

	if !hasFlag(args, "--settings") {
		t.Fatal("--settings flag not found when Sandbox is set")
	}

	val, ok := flagValue(args, "--settings")
	if !ok {
		t.Fatal("--settings flag present but no value follows")
	}

	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(val), &settings); err != nil {
		t.Fatalf("--settings value is not valid JSON: %v", err)
	}

	sandboxRaw, ok := settings["sandbox"]
	if !ok {
		t.Fatal("sandbox key not found in settings JSON")
	}

	sandboxMap, ok := sandboxRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("sandbox is not a map, got %T", sandboxRaw)
	}

	if sandboxMap["enabled"] != true {
		t.Errorf("sandbox.enabled = %v, want true", sandboxMap["enabled"])
	}
}

// TestBuildCommandArgs_SettingsFileCheckpointing tests --settings flag with file checkpointing.
func TestBuildCommandArgs_SettingsFileCheckpointing(t *testing.T) {
	t.Parallel()

	opts := types.NewClaudeAgentOptions().
		WithEnableFileCheckpointing(true)

	transport := newTestTransport(t, opts)
	args := transport.buildCommandArgs()

	if !hasFlag(args, "--settings") {
		t.Fatal("--settings flag not found when EnableFileCheckpointing is true")
	}

	val, ok := flagValue(args, "--settings")
	if !ok {
		t.Fatal("--settings flag present but no value follows")
	}

	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(val), &settings); err != nil {
		t.Fatalf("--settings value is not valid JSON: %v", err)
	}

	checkpointing, ok := settings["enableFileCheckpointing"]
	if !ok {
		t.Fatal("enableFileCheckpointing key not found in settings JSON")
	}

	if checkpointing != true {
		t.Errorf("enableFileCheckpointing = %v, want true", checkpointing)
	}
}

// TestBuildCommandArgs_SettingsMerge tests that typed settings fields override
// user-provided Settings string on conflict.
func TestBuildCommandArgs_SettingsMerge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		userSettings        *string
		thinking            *types.ThinkingConfig
		enableCheckpointing bool
		wantThinkingType    string
		wantCheckpointing   bool
		wantUserField       bool // Whether user-provided "customField" survives
	}{
		{
			name:         "typed fields override user settings",
			userSettings: strPtr(`{"thinking":{"type":"disabled"},"customField":"keep-me"}`),
			thinking: &types.ThinkingConfig{
				Type: "adaptive",
			},
			wantThinkingType: "adaptive",
			wantUserField:    true,
		},
		{
			name:                "typed checkpointing added to user settings",
			userSettings:        strPtr(`{"customField":"keep-me"}`),
			enableCheckpointing: true,
			wantCheckpointing:   true,
			wantUserField:       true,
		},
		{
			name:         "only typed fields — no user settings",
			userSettings: nil,
			thinking: &types.ThinkingConfig{
				Type: "enabled",
			},
			wantThinkingType: "enabled",
			wantUserField:    false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.NewClaudeAgentOptions()
			if tt.userSettings != nil {
				opts.WithSettings(*tt.userSettings)
			}
			if tt.thinking != nil {
				opts.WithThinking(*tt.thinking)
			}
			if tt.enableCheckpointing {
				opts.WithEnableFileCheckpointing(true)
			}

			transport := newTestTransport(t, opts)
			args := transport.buildCommandArgs()

			val, ok := flagValue(args, "--settings")
			if !ok {
				t.Fatal("--settings flag not found")
			}

			var settings map[string]interface{}
			if err := json.Unmarshal([]byte(val), &settings); err != nil {
				t.Fatalf("--settings value is not valid JSON: %v", err)
			}

			// Check thinking override
			if tt.wantThinkingType != "" {
				thinkingRaw, ok := settings["thinking"]
				if !ok {
					t.Fatal("thinking key not found in merged settings")
				}
				thinkingJSON, _ := json.Marshal(thinkingRaw)
				var tc types.ThinkingConfig
				if err := json.Unmarshal(thinkingJSON, &tc); err != nil {
					t.Fatalf("failed to unmarshal ThinkingConfig: %v", err)
				}
				if tc.Type != tt.wantThinkingType {
					t.Errorf("thinking.Type = %q, want %q", tc.Type, tt.wantThinkingType)
				}
			}

			// Check checkpointing
			if tt.wantCheckpointing {
				if settings["enableFileCheckpointing"] != true {
					t.Error("enableFileCheckpointing should be true")
				}
			}

			// Check user field preservation
			if tt.wantUserField {
				if settings["customField"] != "keep-me" {
					t.Error("customField from user settings should be preserved")
				}
			}
		})
	}
}

// TestBuildSettingsJSON tests the buildSettingsJSON method directly.
func TestBuildSettingsJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		userSettings        *string
		thinking            *types.ThinkingConfig
		sandbox             *types.SandboxConfig
		enableCheckpointing bool
		wantEmpty           bool
		wantKeys            []string // Keys expected in resulting JSON
	}{
		{
			name:      "no settings at all — returns empty",
			wantEmpty: true,
		},
		{
			name:         "user settings only — returns user string",
			userSettings: strPtr(`{"foo":"bar"}`),
			wantKeys:     []string{"foo"},
		},
		{
			name: "thinking only",
			thinking: &types.ThinkingConfig{
				Type: "adaptive",
			},
			wantKeys: []string{"thinking"},
		},
		{
			name: "sandbox only",
			sandbox: &types.SandboxConfig{
				Enabled: boolPtr(true),
			},
			wantKeys: []string{"sandbox"},
		},
		{
			name:                "checkpointing only",
			enableCheckpointing: true,
			wantKeys:            []string{"enableFileCheckpointing"},
		},
		{
			name:         "user settings + thinking — typed wins",
			userSettings: strPtr(`{"thinking":{"type":"disabled"},"extra":"yes"}`),
			thinking: &types.ThinkingConfig{
				Type: "enabled",
			},
			wantKeys: []string{"thinking", "extra"},
		},
		{
			name:                "all typed fields set",
			thinking:            &types.ThinkingConfig{Type: "adaptive"},
			sandbox:             &types.SandboxConfig{Enabled: boolPtr(true)},
			enableCheckpointing: true,
			wantKeys:            []string{"thinking", "sandbox", "enableFileCheckpointing"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.NewClaudeAgentOptions()
			if tt.userSettings != nil {
				opts.WithSettings(*tt.userSettings)
			}
			if tt.thinking != nil {
				opts.WithThinking(*tt.thinking)
			}
			if tt.sandbox != nil {
				opts.WithSandbox(*tt.sandbox)
			}
			if tt.enableCheckpointing {
				opts.WithEnableFileCheckpointing(true)
			}

			transport := newTestTransport(t, opts)
			result := transport.buildSettingsJSON()

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("buildSettingsJSON() = %q, want empty string", result)
				}
				return
			}

			if result == "" {
				t.Fatal("buildSettingsJSON() returned empty string, want non-empty")
			}

			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(result), &parsed); err != nil {
				t.Fatalf("buildSettingsJSON() returned invalid JSON: %v\nresult: %q", err, result)
			}

			for _, key := range tt.wantKeys {
				if _, ok := parsed[key]; !ok {
					t.Errorf("buildSettingsJSON() result missing key %q, got keys: %v", key, keysOf(parsed))
				}
			}
		})
	}
}

// TestBuildSettingsJSON_UserSettingsReturnedVerbatim verifies that when only user
// Settings is set (no typed fields), the original string is returned without
// re-serialization (preserving formatting).
func TestBuildSettingsJSON_UserSettingsReturnedVerbatim(t *testing.T) {
	t.Parallel()

	userJSON := `{"custom":"value","number":42}`
	opts := types.NewClaudeAgentOptions().
		WithSettings(userJSON)

	transport := newTestTransport(t, opts)
	result := transport.buildSettingsJSON()

	if result != userJSON {
		t.Errorf("buildSettingsJSON() = %q, want original string %q (verbatim)", result, userJSON)
	}
}

// TestBuildSettingsJSON_InvalidUserSettingsIgnored verifies that invalid user JSON
// falls back to typed fields only.
func TestBuildSettingsJSON_InvalidUserSettingsIgnored(t *testing.T) {
	t.Parallel()

	opts := types.NewClaudeAgentOptions().
		WithSettings("not valid json{{{").
		WithThinking(types.ThinkingConfig{Type: "adaptive"})

	transport := newTestTransport(t, opts)
	result := transport.buildSettingsJSON()

	if result == "" {
		t.Fatal("buildSettingsJSON() returned empty, expected thinking config")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("buildSettingsJSON() returned invalid JSON: %v", err)
	}

	if _, ok := parsed["thinking"]; !ok {
		t.Error("thinking key missing from result")
	}
}

// TestBuildCommandArgs_NoSettingsWhenNothingSet verifies --settings is absent when
// no settings-related options are configured.
func TestBuildCommandArgs_NoSettingsWhenNothingSet(t *testing.T) {
	t.Parallel()

	opts := types.NewClaudeAgentOptions()
	transport := newTestTransport(t, opts)
	args := transport.buildCommandArgs()

	if hasFlag(args, "--settings") {
		t.Error("--settings flag should not be present when no settings are configured")
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

// --- helpers ---

func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func effortPtr(e types.EffortLevel) *types.EffortLevel {
	return &e
}

func keysOf(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
