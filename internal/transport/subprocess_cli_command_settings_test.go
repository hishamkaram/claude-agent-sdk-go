package transport

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

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

func TestBuildCommandArgs_ThinkingDisplay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		thinking  types.ThinkingConfig
		wantFlag  bool
		wantValue string
	}{
		{
			name:      "adaptive summarized",
			thinking:  types.ThinkingConfig{Type: "adaptive", Display: "summarized"},
			wantFlag:  true,
			wantValue: "summarized",
		},
		{
			name: "enabled omitted",
			thinking: types.ThinkingConfig{
				Type:    "enabled",
				Display: "omitted",
			},
			wantFlag:  true,
			wantValue: "omitted",
		},
		{
			name:     "adaptive empty display",
			thinking: types.ThinkingConfig{Type: "adaptive"},
			wantFlag: false,
		},
		{
			name:     "disabled display ignored",
			thinking: types.ThinkingConfig{Type: "disabled", Display: "summarized"},
			wantFlag: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.NewClaudeAgentOptions().WithThinking(tt.thinking)
			transport := newTestTransport(t, opts)
			args := transport.buildCommandArgs()

			val, found := flagValue(args, "--thinking-display")
			if found != tt.wantFlag {
				t.Fatalf("--thinking-display found = %v, want %v; args: %v", found, tt.wantFlag, args)
			}
			if tt.wantFlag && val != tt.wantValue {
				t.Fatalf("--thinking-display = %q, want %q", val, tt.wantValue)
			}
		})
	}
}

func TestBuildCommandArgs_ThinkingDisplayUnsupportedCLI(t *testing.T) {
	t.Parallel()

	opts := types.NewClaudeAgentOptions().
		WithThinking(types.ThinkingConfig{Type: "adaptive", Display: "summarized"})
	transport := newTestTransport(t, opts)
	transport.thinkingDisplaySupported = false

	args := transport.buildCommandArgs()
	if hasFlag(args, "--thinking-display") {
		t.Fatalf("--thinking-display should be omitted when CLI support is not available; args: %v", args)
	}
}

func TestDetectThinkingDisplaySupportIgnoresSkipVersionCheck(t *testing.T) {
	t.Setenv("CLAUDE_AGENT_SDK_SKIP_VERSION_CHECK", "1")

	cliPath := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\necho '2.1.92 (Claude Code)'\n"), 0o755); err != nil {
		t.Fatalf("write fake cli: %v", err)
	}

	opts := types.NewClaudeAgentOptions().
		WithThinking(types.ThinkingConfig{Type: "adaptive", Display: "summarized"})
	transport := NewSubprocessCLITransport(cliPath, "", nil, log.NewLogger(false), "", opts)

	if transport.detectThinkingDisplaySupport(context.Background()) {
		t.Fatal("thinking display support should remain false for CLI 2.1.92 even when version checks are skipped")
	}
}

func TestConnectWithCustomSpawnerPreservesThinkingDisplay(t *testing.T) {
	t.Parallel()

	var receivedOpts types.SpawnOptions
	mockProc := newMockSpawnedProcess()
	spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
		receivedOpts = opts
		return mockProc, nil
	})

	opts := types.NewClaudeAgentOptions().
		WithThinking(types.ThinkingConfig{Type: "adaptive", Display: "summarized"}).
		WithSpawnProcess(spawner)
	transport := NewSubprocessCLITransport("/remote/only/claude", "", nil, log.NewLogger(false), "", opts)

	if err := transport.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	t.Cleanup(func() {
		_ = mockProc.Kill()
		closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = transport.Close(closeCtx)
	})

	val, found := flagValue(receivedOpts.Args, "--thinking-display")
	if !found {
		t.Fatalf("--thinking-display missing from custom SpawnOptions.Args: %v", receivedOpts.Args)
	}
	if val != "summarized" {
		t.Fatalf("--thinking-display = %q, want summarized", val)
	}
}

func TestConnectWithCustomSpawnerPreservesThinkingDisplayWithProbeableOldHostCLI(t *testing.T) {
	cliPath := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\necho '2.1.92 (Claude Code)'\n"), 0o755); err != nil {
		t.Fatalf("write fake cli: %v", err)
	}

	var spawnerCalled bool
	var receivedOpts types.SpawnOptions
	mockProc := newMockSpawnedProcess()
	spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
		spawnerCalled = true
		receivedOpts = opts
		return mockProc, nil
	})

	opts := types.NewClaudeAgentOptions().
		WithThinking(types.ThinkingConfig{Type: "adaptive", Display: "summarized"}).
		WithSpawnProcess(spawner)
	transport := NewSubprocessCLITransport(cliPath, "", nil, log.NewLogger(false), "", opts)

	if err := transport.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	t.Cleanup(func() {
		_ = mockProc.Kill()
		closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = transport.Close(closeCtx)
	})

	if !spawnerCalled {
		t.Fatal("custom spawner was not invoked")
	}
	val, found := flagValue(receivedOpts.Args, "--thinking-display")
	if !found {
		t.Fatalf("--thinking-display missing from custom SpawnOptions.Args: %v", receivedOpts.Args)
	}
	if val != "summarized" {
		t.Fatalf("--thinking-display = %q, want summarized", val)
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

// TestBuildCommandArgs_ReplayUserMessages tests --replay-user-messages is always present
// (needed for branch-at-message regardless of file checkpointing).
func TestBuildCommandArgs_ReplayUserMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		enableCheckpointing bool
		wantFlag            bool
	}{
		{
			name:                "present when file checkpointing enabled",
			enableCheckpointing: true,
			wantFlag:            true,
		},
		{
			name:                "absent when file checkpointing disabled",
			enableCheckpointing: false,
			wantFlag:            false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.NewClaudeAgentOptions()
			if tt.enableCheckpointing {
				opts = opts.WithEnableFileCheckpointing(true)
			}

			transport := newTestTransport(t, opts)
			args := transport.buildCommandArgs()

			got := hasFlag(args, "--replay-user-messages")
			if got != tt.wantFlag {
				t.Errorf("hasFlag(--replay-user-messages) = %v, want %v", got, tt.wantFlag)
			}
		})
	}
}

// TestBuildCommandArgs_SettingsMerge tests that typed settings fields override
// user-provided Settings string on conflict.
