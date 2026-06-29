package types

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

// ===== Phase D: US1 — Custom Process Spawner Tests =====

// TestSpawnOptions_JSONRoundtrip tests SpawnOptions JSON serialization.
func TestSpawnOptions_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input SpawnOptions
	}{
		{
			name: "full options",
			input: SpawnOptions{
				Command: "/usr/bin/docker",
				Args:    []string{"run", "-i", "--rm", "claude:latest"},
				CWD:     "/workspace",
				Env:     map[string]string{"HOME": "/root", "PATH": "/usr/bin"},
			},
		},
		{
			name: "minimal options",
			input: SpawnOptions{
				Command: "claude",
			},
		},
		{
			name: "empty env",
			input: SpawnOptions{
				Command: "claude",
				Args:    []string{"--help"},
				Env:     map[string]string{},
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
			var got SpawnOptions
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if got.Command != tt.input.Command {
				t.Errorf("Command = %q, want %q", got.Command, tt.input.Command)
			}
			if !reflect.DeepEqual(got.Args, tt.input.Args) {
				t.Errorf("Args = %v, want %v", got.Args, tt.input.Args)
			}
			if got.CWD != tt.input.CWD {
				t.Errorf("CWD = %q, want %q", got.CWD, tt.input.CWD)
			}
		})
	}
}

// TestWithSpawnProcess tests the WithSpawnProcess builder.
func TestWithSpawnProcess(t *testing.T) {
	t.Parallel()

	called := false
	spawner := ProcessSpawner(func(ctx context.Context, opts SpawnOptions) (SpawnedProcess, error) {
		called = true
		return nil, nil
	})

	opts := NewClaudeAgentOptions().WithSpawnProcess(spawner)
	if opts.SpawnProcess == nil {
		t.Fatal("SpawnProcess should not be nil after setting")
	}

	// Verify it's callable
	_, _ = opts.SpawnProcess(context.Background(), SpawnOptions{Command: "test"})
	if !called {
		t.Error("SpawnProcess function was not called")
	}
}

// TestSpawnProcess_NotMarshaledToJSON verifies SpawnProcess is excluded from JSON.
func TestSpawnProcess_NotMarshaledToJSON(t *testing.T) {
	t.Parallel()

	opts := NewClaudeAgentOptions().WithSpawnProcess(func(ctx context.Context, opts SpawnOptions) (SpawnedProcess, error) {
		return nil, nil
	})

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if _, exists := raw["spawn_process"]; exists {
		t.Error("SpawnProcess should not appear in JSON output")
	}
}

// ===== Phase D: US2 — Missing Options Tests =====

// TestToolConfig_JSONRoundtrip tests ToolConfig JSON serialization.
func TestToolConfig_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	timeout := 30000
	cmd := "/bin/bash"
	display := 1
	width := 1920
	height := 1080

	tests := []struct {
		name  string
		input ToolConfig
	}{
		{
			name: "full config",
			input: ToolConfig{
				Bash:     &BashToolConfig{Timeout: &timeout, Command: &cmd},
				Computer: &ComputerToolConfig{Display: &display, Width: &width, Height: &height},
			},
		},
		{
			name:  "empty config",
			input: ToolConfig{},
		},
		{
			name: "bash only",
			input: ToolConfig{
				Bash: &BashToolConfig{Timeout: &timeout},
			},
		},
		{
			name: "computer only",
			input: ToolConfig{
				Computer: &ComputerToolConfig{Width: &width},
			},
		},
		{
			name: "nil sub-structs",
			input: ToolConfig{
				Bash:     nil,
				Computer: nil,
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
			var got ToolConfig
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			// Re-marshal and compare
			data2, _ := json.Marshal(got)
			if string(data) != string(data2) {
				t.Errorf("Roundtrip mismatch:\n  got:  %s\n  want: %s", data2, data)
			}
		})
	}
}

// TestWithResumeSessionAt tests the WithResumeSessionAt builder.
func TestWithResumeSessionAt(t *testing.T) {
	t.Parallel()
	opts := NewClaudeAgentOptions().WithResumeSessionAt("msg-uuid-123")
	if opts.ResumeSessionAt == nil || *opts.ResumeSessionAt != "msg-uuid-123" {
		t.Errorf("ResumeSessionAt = %v, want %q", opts.ResumeSessionAt, "msg-uuid-123")
	}
}

// TestWithToolConfig tests the WithToolConfig builder.
func TestWithToolConfig(t *testing.T) {
	t.Parallel()
	timeout := 5000
	opts := NewClaudeAgentOptions().WithToolConfig(ToolConfig{
		Bash: &BashToolConfig{Timeout: &timeout},
	})
	if opts.ToolConfig == nil {
		t.Fatal("ToolConfig should not be nil")
	}
	if opts.ToolConfig.Bash == nil || opts.ToolConfig.Bash.Timeout == nil || *opts.ToolConfig.Bash.Timeout != 5000 {
		t.Error("ToolConfig.Bash.Timeout not set correctly")
	}
}

// TestWithTools tests the WithTools builder with different input types.
func TestWithTools(t *testing.T) {
	t.Parallel()

	t.Run("string slice", func(t *testing.T) {
		t.Parallel()
		tools := []string{"Bash", "Read", "Write"}
		opts := NewClaudeAgentOptions().WithTools(tools)
		got, ok := opts.Tools.([]string)
		if !ok {
			t.Fatalf("Tools should be []string, got %T", opts.Tools)
		}
		if !reflect.DeepEqual(got, tools) {
			t.Errorf("Tools = %v, want %v", got, tools)
		}
	})

	t.Run("preset map", func(t *testing.T) {
		t.Parallel()
		preset := map[string]string{"type": "preset", "preset": "claude_code"}
		opts := NewClaudeAgentOptions().WithTools(preset)
		got, ok := opts.Tools.(map[string]string)
		if !ok {
			t.Fatalf("Tools should be map[string]string, got %T", opts.Tools)
		}
		if got["preset"] != "claude_code" {
			t.Errorf("Tools preset = %q, want %q", got["preset"], "claude_code")
		}
	})
}

// TestWithDebugFile tests the WithDebugFile builder.
func TestWithDebugFile(t *testing.T) {
	t.Parallel()
	opts := NewClaudeAgentOptions().WithDebugFile("/tmp/debug.log")
	if opts.DebugFile == nil || *opts.DebugFile != "/tmp/debug.log" {
		t.Errorf("DebugFile = %v, want %q", opts.DebugFile, "/tmp/debug.log")
	}
}

// TestWithStrictMcpConfig tests the WithStrictMcpConfig builder.
func TestWithStrictMcpConfig(t *testing.T) {
	t.Parallel()
	opts := NewClaudeAgentOptions().WithStrictMcpConfig(true)
	if !opts.StrictMcpConfig {
		t.Error("StrictMcpConfig should be true")
	}
}
