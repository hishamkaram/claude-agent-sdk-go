package transport

import (
	"encoding/json"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

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

func TestBuildSettingsJSON_ThinkingDisplay(t *testing.T) {
	t.Parallel()

	opts := types.NewClaudeAgentOptions().
		WithThinking(types.ThinkingConfig{Type: "adaptive", Display: "summarized"})

	transport := newTestTransport(t, opts)
	result := transport.buildSettingsJSON()

	var parsed struct {
		Thinking types.ThinkingConfig `json:"thinking"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("buildSettingsJSON() returned invalid JSON: %v\nresult: %q", err, result)
	}
	if parsed.Thinking.Type != "adaptive" {
		t.Fatalf("thinking.type = %q, want adaptive", parsed.Thinking.Type)
	}
	if parsed.Thinking.Display != "summarized" {
		t.Fatalf("thinking.display = %q, want summarized", parsed.Thinking.Display)
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

// TestBuildCommandArgs_SessionAgent verifies --agent flag generation.
