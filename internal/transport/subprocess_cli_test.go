package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

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
		name      string
		effort    *types.EffortLevel
		wantFlag  bool
		wantValue string
	}{
		{
			name:      "effort high",
			effort:    effortPtr(types.EffortHigh),
			wantFlag:  true,
			wantValue: "high",
		},
		{
			name:      "effort low",
			effort:    effortPtr(types.EffortLow),
			wantFlag:  true,
			wantValue: "low",
		},
		{
			name:      "effort medium",
			effort:    effortPtr(types.EffortMedium),
			wantFlag:  true,
			wantValue: "medium",
		},
		{
			name:      "effort max",
			effort:    effortPtr(types.EffortMax),
			wantFlag:  true,
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
			name:      "session ID nil — flag absent",
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
	opts := types.NewClaudeAgentOptions()
	opts.Agents = map[string]types.AgentDefinition{
		"test-agent": {
			Description:            "test agent",
			Prompt:                 "do stuff",
			DisallowedTools:        []string{"Bash", "Write"},
			McpServers:             []interface{}{"server1", map[string]interface{}{"name": "server2"}},
			Skills:                 []string{"skill-a", "skill-b"},
			CriticalSystemReminder: &reminder,
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
	if _, ok := agent["disallowed_tools"]; !ok {
		t.Error("disallowed_tools missing from agent JSON")
	}
	if _, ok := agent["mcp_servers"]; !ok {
		t.Error("mcp_servers missing from agent JSON")
	}
	if _, ok := agent["skills"]; !ok {
		t.Error("skills missing from agent JSON")
	}
	if val, ok := agent["criticalSystemReminder_EXPERIMENTAL"]; !ok {
		t.Error("criticalSystemReminder_EXPERIMENTAL missing from agent JSON")
	} else if val != "critical reminder text" {
		t.Errorf("criticalSystemReminder_EXPERIMENTAL = %v, want %q", val, "critical reminder text")
	}
}

// ===== Phase D: Custom Spawner Integration Tests (T004-T, T005-T) =====

// mockSpawnedProcess implements types.SpawnedProcess for testing.
type mockSpawnedProcess struct {
	mu       sync.Mutex
	stdin    *mockWriteCloser
	stdout   *io.PipeReader
	stdoutW  *io.PipeWriter
	stderr   *io.PipeReader
	stderrW  *io.PipeWriter
	killed   bool
	exitCode int
	waitErr  error
	waitCh   chan struct{}
}

func newMockSpawnedProcess() *mockSpawnedProcess {
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	return &mockSpawnedProcess{
		stdin:   &mockWriteCloser{buf: &bytes.Buffer{}},
		stdout:  stdoutR,
		stdoutW: stdoutW,
		stderr:  stderrR,
		stderrW: stderrW,
		waitCh:  make(chan struct{}),
	}
}

func (m *mockSpawnedProcess) Stdin() io.WriteCloser { return m.stdin }
func (m *mockSpawnedProcess) Stdout() io.ReadCloser { return m.stdout }
func (m *mockSpawnedProcess) Stderr() io.ReadCloser { return m.stderr }

func (m *mockSpawnedProcess) Kill() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.killed = true
	// Close stdout/stderr to unblock readers
	_ = m.stdoutW.Close()
	_ = m.stderrW.Close()
	// Signal Wait() to return
	select {
	case <-m.waitCh:
	default:
		close(m.waitCh)
	}
	return nil
}

func (m *mockSpawnedProcess) Wait() error {
	<-m.waitCh
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.waitErr
}

func (m *mockSpawnedProcess) ExitCode() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.exitCode
}

func (m *mockSpawnedProcess) Killed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.killed
}

type mockWriteCloser struct {
	buf    *bytes.Buffer
	closed bool
}

func (m *mockWriteCloser) Write(p []byte) (int, error) {
	if m.closed {
		return 0, errors.New("write to closed writer")
	}
	return m.buf.Write(p)
}

func (m *mockWriteCloser) Close() error {
	m.closed = true
	return nil
}

// TestConnectWithCustomSpawner verifies that Connect() uses the custom spawner
// when SpawnProcess is set and that it receives correct SpawnOptions.
func TestConnectWithCustomSpawner(t *testing.T) {
	t.Parallel()

	var receivedOpts types.SpawnOptions
	var receivedCtx context.Context
	mockProc := newMockSpawnedProcess()

	spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
		receivedCtx = ctx
		receivedOpts = opts
		return mockProc, nil
	})

	opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
	tr := NewSubprocessCLITransport("/usr/bin/claude", "/tmp/test", map[string]string{"MY_VAR": "my_val"}, log.NewLogger(true), "", opts)

	ctx := context.Background()
	err := tr.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	// Verify spawner received correct options
	if receivedOpts.Command != "/usr/bin/claude" {
		t.Errorf("SpawnOptions.Command = %q, want %q", receivedOpts.Command, "/usr/bin/claude")
	}
	if receivedOpts.CWD != "/tmp/test" {
		t.Errorf("SpawnOptions.CWD = %q, want %q", receivedOpts.CWD, "/tmp/test")
	}
	if receivedCtx == nil {
		t.Error("spawner received nil context")
	}
	// Verify env vars contain both SDK vars and custom vars
	if receivedOpts.Env["MY_VAR"] != "my_val" {
		t.Error("custom env var not passed to spawner")
	}
	if receivedOpts.Env["CLAUDE_CODE_ENTRYPOINT"] != "agent" {
		t.Errorf("CLAUDE_CODE_ENTRYPOINT = %q, want %q", receivedOpts.Env["CLAUDE_CODE_ENTRYPOINT"], "agent")
	}

	// Verify transport state
	if tr.customProcess == nil {
		t.Error("customProcess should be set after Connect()")
	}
	if tr.cmd != nil {
		t.Error("cmd should be nil when using custom spawner")
	}

	// Cleanup
	_ = mockProc.Kill()
	ctx2, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = tr.Close(ctx2)
}

// TestConnectWithCustomSpawner_Error verifies Connect() propagates spawner errors.
func TestConnectWithCustomSpawner_Error(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("spawner failed: VM not available")
	spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
		return nil, expectedErr
	})

	opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
	tr := NewSubprocessCLITransport("/usr/bin/claude", "", nil, log.NewLogger(true), "", opts)

	err := tr.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should have failed")
	}
	if !strings.Contains(err.Error(), "spawner failed") {
		t.Errorf("error should contain spawner message, got: %v", err)
	}
}

// TestConnectWithCustomSpawner_NilPipes verifies Connect() rejects a process with nil pipes.
func TestConnectWithCustomSpawner_NilPipes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		process types.SpawnedProcess
	}{
		{
			name:    "nil stdin",
			process: &mockSpawnedProcessNilPipe{nilStdin: true},
		},
		{
			name:    "nil stdout",
			process: &mockSpawnedProcessNilPipe{nilStdout: true},
		},
		{
			name:    "nil stderr",
			process: &mockSpawnedProcessNilPipe{nilStderr: true},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
				return tt.process, nil
			})

			opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
			tr := NewSubprocessCLITransport("/usr/bin/claude", "", nil, log.NewLogger(true), "", opts)

			err := tr.Connect(context.Background())
			if err == nil {
				t.Fatal("Connect() should fail with nil pipe")
			}
			if !strings.Contains(err.Error(), "nil") {
				t.Errorf("error should mention nil pipe, got: %v", err)
			}
		})
	}
}

// mockSpawnedProcessNilPipe returns nil for specified pipes.
type mockSpawnedProcessNilPipe struct {
	nilStdin  bool
	nilStdout bool
	nilStderr bool
}

func (m *mockSpawnedProcessNilPipe) Stdin() io.WriteCloser {
	if m.nilStdin {
		return nil
	}
	return &mockWriteCloser{buf: &bytes.Buffer{}}
}
func (m *mockSpawnedProcessNilPipe) Stdout() io.ReadCloser {
	if m.nilStdout {
		return nil
	}
	r, _ := io.Pipe()
	return r
}
func (m *mockSpawnedProcessNilPipe) Stderr() io.ReadCloser {
	if m.nilStderr {
		return nil
	}
	r, _ := io.Pipe()
	return r
}
func (m *mockSpawnedProcessNilPipe) Kill() error   { return nil }
func (m *mockSpawnedProcessNilPipe) Wait() error   { return nil }
func (m *mockSpawnedProcessNilPipe) ExitCode() int { return 0 }
func (m *mockSpawnedProcessNilPipe) Killed() bool  { return false }

// TestCloseCustomProcess verifies Close() calls Wait() on custom process.
func TestCloseCustomProcess(t *testing.T) {
	t.Parallel()

	mockProc := newMockSpawnedProcess()
	tr := NewSubprocessCLITransport("/usr/bin/claude", "", nil, log.NewLogger(true), "", nil)

	// Manually set transport state as if connectWithCustomSpawner ran
	tr.customProcess = mockProc
	tr.stdin = mockProc.Stdin()
	tr.stdout = mockProc.Stdout()
	tr.stderr = mockProc.Stderr()
	tr.ready = true
	ctx, cancel := context.WithCancel(context.Background())
	tr.ctx = ctx
	tr.cancel = cancel

	// Initialize procDone and launch watcher goroutine (mirrors connectWithCustomSpawner)
	tr.procDone = make(chan struct{})
	go func() {
		_ = mockProc.Wait()
		close(tr.procDone)
	}()

	// Simulate process exiting cleanly after stdin is closed
	go func() {
		select {
		case <-mockProc.waitCh:
		default:
			close(mockProc.waitCh)
		}
	}()

	err := tr.Close(context.Background())
	if err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	if tr.customProcess != nil {
		t.Error("customProcess should be nil after Close()")
	}
}

// TestCloseCustomProcess_NotConnected verifies Close() is a no-op when not connected.
func TestCloseCustomProcess_NotConnected(t *testing.T) {
	t.Parallel()

	tr := NewSubprocessCLITransport("/usr/bin/claude", "", nil, log.NewLogger(true), "", nil)

	err := tr.Close(context.Background())
	if err != nil {
		t.Fatalf("Close() should succeed when not connected, got: %v", err)
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

// ===== Phase E: Subprocess Crash Tests (T003-T) =====
//
// These tests use a mockSpawnedProcess (custom spawner) to simulate subprocess crashes.
// They verify the watcher goroutine behaviour added in T007.
//
// RED before T007: IsReady() stays true after Kill() — no watcher to clear it.
// GREEN after T007: watcher sets ready=false; all assertions pass.

// waitForNotReady polls IsReady() until it returns false or the deadline is reached.
func waitForNotReady(tr *SubprocessCLITransport, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !tr.IsReady() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

// TestSubprocessCrash_ReadyFalse verifies that IsReady() returns false
// immediately after the subprocess exits spontaneously.
func TestSubprocessCrash_ReadyFalse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "process killed externally"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockProc := newMockSpawnedProcess()
			spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
				return mockProc, nil
			})

			opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
			tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)

			ctx := context.Background()
			if err := tr.Connect(ctx); err != nil {
				t.Fatalf("Connect() failed: %v", err)
			}

			if !tr.IsReady() {
				t.Fatal("transport should be ready before crash")
			}

			// Simulate subprocess crash.
			_ = mockProc.Kill()

			// Watcher goroutine must set ready=false within 2s.
			if !waitForNotReady(tr, 2*time.Second) {
				t.Error("IsReady() should return false after subprocess crash, but it is still true")
			}

			closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = tr.Close(closeCtx)
		})
	}
}

// TestSubprocessCrash_WriteReturnsError verifies that Write() returns a non-nil
// error after the subprocess exits — without touching the closed pipe (no EPIPE).
func TestSubprocessCrash_WriteReturnsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "write after crash returns error"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockProc := newMockSpawnedProcess()
			spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
				return mockProc, nil
			})

			opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
			tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)

			ctx := context.Background()
			if err := tr.Connect(ctx); err != nil {
				t.Fatalf("Connect() failed: %v", err)
			}

			// Crash the subprocess.
			_ = mockProc.Kill()

			// Wait for watcher to mark transport not-ready.
			waitForNotReady(tr, 2*time.Second)

			// Write must return an error — must not attempt to write to the dead pipe.
			err := tr.Write(ctx, `{"type":"user","message":"hello"}`)
			if err == nil {
				t.Error("Write() should return a non-nil error after subprocess crash")
			}

			closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = tr.Close(closeCtx)
		})
	}
}

// TestSubprocessCrash_CloseDoesNotHang verifies that Close() completes within
// a 2s timeout after a spontaneous subprocess exit (no deadlock / double-Wait).
func TestSubprocessCrash_CloseDoesNotHang(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "close after spontaneous exit completes within timeout"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockProc := newMockSpawnedProcess()
			spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
				return mockProc, nil
			})

			opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
			tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)

			ctx := context.Background()
			if err := tr.Connect(ctx); err != nil {
				t.Fatalf("Connect() failed: %v", err)
			}

			// Simulate spontaneous subprocess exit.
			_ = mockProc.Kill()

			// Wait for watcher to detect exit (ensures procDone is closed before Close).
			waitForNotReady(tr, 2*time.Second)

			// Close() must complete within 2s — must not hang or deadlock.
			closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := tr.Close(closeCtx); closeCtx.Err() != nil {
				t.Errorf("Close() hung: context expired before Close() returned (err=%v)", err)
			}
		})
	}
}

// TestSubprocessCrash_NoGoroutineLeak verifies that no goroutines are leaked
// after a subprocess crash followed by Close(). goleak.VerifyTestMain in
// TestMain catches any leaks across the entire test suite as well.
func TestSubprocessCrash_NoGoroutineLeak(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "crash then close leaks no goroutines"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockProc := newMockSpawnedProcess()
			spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
				return mockProc, nil
			})

			opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
			tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)

			ctx := context.Background()
			if err := tr.Connect(ctx); err != nil {
				t.Fatalf("Connect() failed: %v", err)
			}

			// Crash the subprocess.
			_ = mockProc.Kill()

			// Wait for watcher goroutine to complete.
			waitForNotReady(tr, 2*time.Second)

			// Close the transport — all goroutines must exit.
			closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = tr.Close(closeCtx)

			// goleak.VerifyTestMain catches any remaining goroutines for the whole suite.
			// This test ensures the specific crash+close lifecycle runs cleanly.
		})
	}
}
