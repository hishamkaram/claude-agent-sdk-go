package types

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestWithMaxThinkingTokens tests the WithMaxThinkingTokens builder method.
func TestWithMaxThinkingTokens(t *testing.T) {
	opts := NewClaudeAgentOptions()

	// Test setting max thinking tokens
	result := opts.WithMaxThinkingTokens(5000)

	// Verify the method returns the same instance for chaining
	if result != opts {
		t.Error("WithMaxThinkingTokens should return the same instance for chaining")
	}

	// Verify the value is set correctly
	if opts.MaxThinkingTokens == nil {
		t.Fatal("MaxThinkingTokens should not be nil after setting")
	}

	if *opts.MaxThinkingTokens != 5000 {
		t.Errorf("Expected MaxThinkingTokens to be 5000, got %d", *opts.MaxThinkingTokens)
	}
}

// TestWithMaxBudgetUSD tests the WithMaxBudgetUSD builder method.
func TestWithMaxBudgetUSD(t *testing.T) {
	opts := NewClaudeAgentOptions()

	// Test setting max budget
	result := opts.WithMaxBudgetUSD(10.50)

	// Verify the method returns the same instance for chaining
	if result != opts {
		t.Error("WithMaxBudgetUSD should return the same instance for chaining")
	}

	// Verify the value is set correctly
	if opts.MaxBudgetUSD == nil {
		t.Fatal("MaxBudgetUSD should not be nil after setting")
	}

	if *opts.MaxBudgetUSD != 10.50 {
		t.Errorf("Expected MaxBudgetUSD to be 10.50, got %.2f", *opts.MaxBudgetUSD)
	}
}

// TestOptionsChaining tests that the builder methods can be chained together.
func TestOptionsChaining(t *testing.T) {
	opts := NewClaudeAgentOptions().
		WithMaxThinkingTokens(8000).
		WithMaxBudgetUSD(25.00).
		WithModel("claude-3-5-sonnet-20241022").
		WithMaxTurns(10)

	// Verify all values are set correctly
	if opts.MaxThinkingTokens == nil || *opts.MaxThinkingTokens != 8000 {
		t.Error("MaxThinkingTokens not set correctly in chain")
	}

	if opts.MaxBudgetUSD == nil || *opts.MaxBudgetUSD != 25.00 {
		t.Error("MaxBudgetUSD not set correctly in chain")
	}

	if opts.Model == nil || *opts.Model != "claude-3-5-sonnet-20241022" {
		t.Error("Model not set correctly in chain")
	}

	if opts.MaxTurns == nil || *opts.MaxTurns != 10 {
		t.Error("MaxTurns not set correctly in chain")
	}
}

// TestNewClaudeAgentOptions tests that the constructor creates a valid instance.
func TestNewClaudeAgentOptions(t *testing.T) {
	opts := NewClaudeAgentOptions()

	// Check that optional fields are nil by default
	if opts.MaxThinkingTokens != nil {
		t.Error("MaxThinkingTokens should be nil by default")
	}

	if opts.MaxBudgetUSD != nil {
		t.Error("MaxBudgetUSD should be nil by default")
	}

	// Check that maps are initialized
	if opts.Env == nil {
		t.Error("Env should be initialized")
	}

	if opts.ExtraArgs == nil {
		t.Error("ExtraArgs should be initialized")
	}
}

// TestWithMaxThinkingTokensZeroValue tests that zero values can be set.
func TestWithMaxThinkingTokensZeroValue(t *testing.T) {
	opts := NewClaudeAgentOptions().WithMaxThinkingTokens(0)

	if opts.MaxThinkingTokens == nil {
		t.Fatal("MaxThinkingTokens should not be nil")
	}

	if *opts.MaxThinkingTokens != 0 {
		t.Errorf("Expected MaxThinkingTokens to be 0, got %d", *opts.MaxThinkingTokens)
	}
}

// TestWithMaxBudgetUSDZeroValue tests that zero budget can be set.
func TestWithMaxBudgetUSDZeroValue(t *testing.T) {
	opts := NewClaudeAgentOptions().WithMaxBudgetUSD(0.0)

	if opts.MaxBudgetUSD == nil {
		t.Fatal("MaxBudgetUSD should not be nil")
	}

	if *opts.MaxBudgetUSD != 0.0 {
		t.Errorf("Expected MaxBudgetUSD to be 0.0, got %.2f", *opts.MaxBudgetUSD)
	}
}

// TestPluginConfig tests PluginConfig type and validation.
func TestPluginConfig(t *testing.T) {
	t.Run("NewLocalPluginConfig", func(t *testing.T) {
		plugin := NewLocalPluginConfig("/path/to/plugin")
		if plugin.Type != "local" {
			t.Errorf("expected Type 'local', got %s", plugin.Type)
		}
		if plugin.Path != "/path/to/plugin" {
			t.Errorf("expected Path '/path/to/plugin', got %s", plugin.Path)
		}
	})

	t.Run("NewPluginConfig with valid type", func(t *testing.T) {
		plugin, err := NewPluginConfig("local", "/path/to/plugin")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if plugin.Type != "local" {
			t.Errorf("expected Type 'local', got %s", plugin.Type)
		}
		if plugin.Path != "/path/to/plugin" {
			t.Errorf("expected Path '/path/to/plugin', got %s", plugin.Path)
		}
	})

	t.Run("NewPluginConfig with invalid type", func(t *testing.T) {
		_, err := NewPluginConfig("remote", "/path/to/plugin")
		if err == nil {
			t.Error("expected error for unsupported plugin type")
		}
	})

	t.Run("NewPluginConfig with empty path", func(t *testing.T) {
		_, err := NewPluginConfig("local", "")
		if err == nil {
			t.Error("expected error for empty path")
		}
	})
}

// TestClaudeAgentOptions_Plugins tests plugin builder methods.
func TestClaudeAgentOptions_Plugins(t *testing.T) {
	t.Run("WithPlugins", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		plugins := []PluginConfig{
			*NewLocalPluginConfig("/path/to/plugin1"),
			*NewLocalPluginConfig("/path/to/plugin2"),
		}
		opts.WithPlugins(plugins)

		if len(opts.Plugins) != 2 {
			t.Errorf("expected 2 plugins, got %d", len(opts.Plugins))
		}
	})

	t.Run("WithPlugin", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		plugin := *NewLocalPluginConfig("/path/to/plugin")
		opts.WithPlugin(plugin)

		if len(opts.Plugins) != 1 {
			t.Errorf("expected 1 plugin, got %d", len(opts.Plugins))
		}
		if opts.Plugins[0].Path != "/path/to/plugin" {
			t.Errorf("expected Path '/path/to/plugin', got %s", opts.Plugins[0].Path)
		}
	})

	t.Run("WithLocalPlugin", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		opts.WithLocalPlugin("/path/to/plugin")

		if len(opts.Plugins) != 1 {
			t.Errorf("expected 1 plugin, got %d", len(opts.Plugins))
		}
		if opts.Plugins[0].Type != "local" {
			t.Errorf("expected Type 'local', got %s", opts.Plugins[0].Type)
		}
		if opts.Plugins[0].Path != "/path/to/plugin" {
			t.Errorf("expected Path '/path/to/plugin', got %s", opts.Plugins[0].Path)
		}
	})

	t.Run("multiple plugins via WithPlugin", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		opts.WithPlugin(*NewLocalPluginConfig("/path/1")).
			WithPlugin(*NewLocalPluginConfig("/path/2")).
			WithPlugin(*NewLocalPluginConfig("/path/3"))

		if len(opts.Plugins) != 3 {
			t.Errorf("expected 3 plugins, got %d", len(opts.Plugins))
		}
	})

	t.Run("multiple plugins via WithLocalPlugin chaining", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		opts.WithLocalPlugin("/path/1").
			WithLocalPlugin("/path/2").
			WithLocalPlugin("/path/3")

		if len(opts.Plugins) != 3 {
			t.Errorf("expected 3 plugins, got %d", len(opts.Plugins))
		}

		// Verify paths
		expectedPaths := []string{"/path/1", "/path/2", "/path/3"}
		for i, plugin := range opts.Plugins {
			if plugin.Path != expectedPaths[i] {
				t.Errorf("plugin[%d].Path = %s, want %s", i, plugin.Path, expectedPaths[i])
			}
		}
	})

	t.Run("empty plugins by default", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		if opts.Plugins == nil {
			t.Error("Plugins should not be nil")
		}
		if len(opts.Plugins) != 0 {
			t.Errorf("expected 0 plugins by default, got %d", len(opts.Plugins))
		}
	})
}

// TestWithBetas tests the WithBetas builder method.
func TestWithBetas(t *testing.T) {
	t.Run("WithBetas sets multiple beta flags", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		betas := []string{"context-1m-2025-08-07"}

		result := opts.WithBetas(betas)

		// Verify the method returns the same instance for chaining
		if result != opts {
			t.Error("WithBetas should return the same instance for chaining")
		}

		// Verify the values are set correctly
		if len(opts.Betas) != 1 {
			t.Errorf("expected 1 beta, got %d", len(opts.Betas))
		}

		if opts.Betas[0] != "context-1m-2025-08-07" {
			t.Errorf("expected beta 'context-1m-2025-08-07', got %s", opts.Betas[0])
		}
	})

	t.Run("WithBetas empty list", func(t *testing.T) {
		opts := NewClaudeAgentOptions().WithBetas([]string{})

		if len(opts.Betas) != 0 {
			t.Errorf("expected 0 betas, got %d", len(opts.Betas))
		}
	})

	t.Run("WithBetas replaces existing betas", func(t *testing.T) {
		opts := NewClaudeAgentOptions().
			WithBeta("beta-1").
			WithBeta("beta-2").
			WithBetas([]string{"beta-3", "beta-4"})

		if len(opts.Betas) != 2 {
			t.Errorf("expected 2 betas after WithBetas, got %d", len(opts.Betas))
		}

		if opts.Betas[0] != "beta-3" || opts.Betas[1] != "beta-4" {
			t.Errorf("expected betas [beta-3, beta-4], got %v", opts.Betas)
		}
	})
}

// TestWithBeta tests the WithBeta builder method.
func TestWithBeta(t *testing.T) {
	t.Run("WithBeta adds single beta flag", func(t *testing.T) {
		opts := NewClaudeAgentOptions()

		result := opts.WithBeta("context-1m-2025-08-07")

		// Verify the method returns the same instance for chaining
		if result != opts {
			t.Error("WithBeta should return the same instance for chaining")
		}

		// Verify the value is set correctly
		if len(opts.Betas) != 1 {
			t.Errorf("expected 1 beta, got %d", len(opts.Betas))
		}

		if opts.Betas[0] != "context-1m-2025-08-07" {
			t.Errorf("expected beta 'context-1m-2025-08-07', got %s", opts.Betas[0])
		}
	})

	t.Run("WithBeta multiple calls accumulate", func(t *testing.T) {
		opts := NewClaudeAgentOptions().
			WithBeta("beta-1").
			WithBeta("beta-2").
			WithBeta("beta-3")

		if len(opts.Betas) != 3 {
			t.Errorf("expected 3 betas, got %d", len(opts.Betas))
		}

		expectedBetas := []string{"beta-1", "beta-2", "beta-3"}
		for i, beta := range opts.Betas {
			if beta != expectedBetas[i] {
				t.Errorf("beta[%d] = %s, expected %s", i, beta, expectedBetas[i])
			}
		}
	})

	t.Run("empty betas by default", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		if opts.Betas == nil {
			t.Error("Betas should not be nil")
		}
		if len(opts.Betas) != 0 {
			t.Errorf("expected 0 betas by default, got %d", len(opts.Betas))
		}
	})
}

// TestSubagentExecutionModeConstants tests the SubagentExecutionMode enum values.
func TestSubagentExecutionModeConstants(t *testing.T) {
	tests := []struct {
		mode     SubagentExecutionMode
		expected string
	}{
		{SubagentExecutionModeSequential, "sequential"},
		{SubagentExecutionModeParallel, "parallel"},
		{SubagentExecutionModeAuto, "auto"},
	}

	for _, tt := range tests {
		if string(tt.mode) != tt.expected {
			t.Errorf("SubagentExecutionMode = %q, expected %q", string(tt.mode), tt.expected)
		}
	}
}

// TestMultiInvocationModeConstants tests the MultiInvocationMode enum values.
func TestMultiInvocationModeConstants(t *testing.T) {
	tests := []struct {
		mode     MultiInvocationMode
		expected string
	}{
		{MultiInvocationModeSequential, "sequential"},
		{MultiInvocationModeParallel, "parallel"},
		{MultiInvocationModeError, "error"},
	}

	for _, tt := range tests {
		if string(tt.mode) != tt.expected {
			t.Errorf("MultiInvocationMode = %q, expected %q", string(tt.mode), tt.expected)
		}
	}
}

// TestSubagentErrorHandlingConstants tests the SubagentErrorHandling enum values.
func TestSubagentErrorHandlingConstants(t *testing.T) {
	tests := []struct {
		mode     SubagentErrorHandling
		expected string
	}{
		{SubagentErrorHandlingFailFast, "fail_fast"},
		{SubagentErrorHandlingContinue, "continue"},
	}

	for _, tt := range tests {
		if string(tt.mode) != tt.expected {
			t.Errorf("SubagentErrorHandling = %q, expected %q", string(tt.mode), tt.expected)
		}
	}
}

// TestNewSubagentExecutionConfig tests the NewSubagentExecutionConfig constructor.
func TestNewSubagentExecutionConfig(t *testing.T) {
	t.Run("creates config with sensible defaults", func(t *testing.T) {
		config := NewSubagentExecutionConfig()

		if config.MultiInvocation != MultiInvocationModeSequential {
			t.Errorf("expected MultiInvocation to be sequential, got %s", config.MultiInvocation)
		}

		if config.MaxConcurrent != 3 {
			t.Errorf("expected MaxConcurrent to be 3, got %d", config.MaxConcurrent)
		}

		if config.ErrorHandling != SubagentErrorHandlingContinue {
			t.Errorf("expected ErrorHandling to be continue, got %s", config.ErrorHandling)
		}
	})

	t.Run("can be customized after creation", func(t *testing.T) {
		config := NewSubagentExecutionConfig()
		config.MultiInvocation = MultiInvocationModeParallel
		config.MaxConcurrent = 5
		config.ErrorHandling = SubagentErrorHandlingFailFast

		if config.MultiInvocation != MultiInvocationModeParallel {
			t.Errorf("expected MultiInvocation to be parallel, got %s", config.MultiInvocation)
		}

		if config.MaxConcurrent != 5 {
			t.Errorf("expected MaxConcurrent to be 5, got %d", config.MaxConcurrent)
		}

		if config.ErrorHandling != SubagentErrorHandlingFailFast {
			t.Errorf("expected ErrorHandling to be fail_fast, got %s", config.ErrorHandling)
		}
	})
}

// TestAgentDefinitionWithExecutionControl tests AgentDefinition with new execution control fields.
func TestAgentDefinitionWithExecutionControl(t *testing.T) {
	t.Run("agent with execution mode", func(t *testing.T) {
		mode := SubagentExecutionModeParallel
		agent := AgentDefinition{
			Description:   "Test agent",
			Prompt:        "Test prompt",
			ExecutionMode: &mode,
		}

		if agent.ExecutionMode == nil {
			t.Fatal("ExecutionMode should not be nil")
		}

		if *agent.ExecutionMode != SubagentExecutionModeParallel {
			t.Errorf("expected ExecutionMode to be parallel, got %s", *agent.ExecutionMode)
		}
	})

	t.Run("agent with timeout", func(t *testing.T) {
		timeout := 30.5
		agent := AgentDefinition{
			Description: "Test agent",
			Prompt:      "Test prompt",
			Timeout:     &timeout,
		}

		if agent.Timeout == nil {
			t.Fatal("Timeout should not be nil")
		}

		if *agent.Timeout != 30.5 {
			t.Errorf("expected Timeout to be 30.5, got %f", *agent.Timeout)
		}
	})

	t.Run("agent with max turns", func(t *testing.T) {
		maxTurns := 5
		agent := AgentDefinition{
			Description: "Test agent",
			Prompt:      "Test prompt",
			MaxTurns:    &maxTurns,
		}

		if agent.MaxTurns == nil {
			t.Fatal("MaxTurns should not be nil")
		}

		if *agent.MaxTurns != 5 {
			t.Errorf("expected MaxTurns to be 5, got %d", *agent.MaxTurns)
		}
	})

	t.Run("agent with all execution control fields", func(t *testing.T) {
		mode := SubagentExecutionModeSequential
		timeout := 60.0
		maxTurns := 10
		agent := AgentDefinition{
			Description:   "Full agent",
			Prompt:        "Full prompt",
			Tools:         []string{"Read", "Write"},
			ExecutionMode: &mode,
			Timeout:       &timeout,
			MaxTurns:      &maxTurns,
		}

		if agent.ExecutionMode == nil || *agent.ExecutionMode != SubagentExecutionModeSequential {
			t.Errorf("ExecutionMode mismatch")
		}

		if agent.Timeout == nil || *agent.Timeout != 60.0 {
			t.Errorf("Timeout mismatch")
		}

		if agent.MaxTurns == nil || *agent.MaxTurns != 10 {
			t.Errorf("MaxTurns mismatch")
		}
	})
}

// TestWithSubagentExecution tests the WithSubagentExecution builder method.
func TestWithSubagentExecution(t *testing.T) {
	t.Run("sets subagent execution config", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		config := NewSubagentExecutionConfig()
		config.MaxConcurrent = 5

		result := opts.WithSubagentExecution(config)

		// Verify the method returns the same instance for chaining
		if result != opts {
			t.Error("WithSubagentExecution should return the same instance for chaining")
		}

		// Verify the value is set
		if opts.SubagentExecution == nil {
			t.Fatal("SubagentExecution should not be nil after setting")
		}

		if opts.SubagentExecution.MaxConcurrent != 5 {
			t.Errorf("expected MaxConcurrent to be 5, got %d", opts.SubagentExecution.MaxConcurrent)
		}
	})

	t.Run("replaces existing config", func(t *testing.T) {
		opts := NewClaudeAgentOptions()

		config1 := NewSubagentExecutionConfig()
		config1.MaxConcurrent = 2
		opts.WithSubagentExecution(config1)

		config2 := NewSubagentExecutionConfig()
		config2.MaxConcurrent = 8
		opts.WithSubagentExecution(config2)

		if opts.SubagentExecution.MaxConcurrent != 8 {
			t.Errorf("expected MaxConcurrent to be 8 after replacement, got %d", opts.SubagentExecution.MaxConcurrent)
		}
	})

	t.Run("method chaining works", func(t *testing.T) {
		config := NewSubagentExecutionConfig()
		config.MultiInvocation = MultiInvocationModeParallel

		opts := NewClaudeAgentOptions().
			WithModel("claude-opus-4-5-latest").
			WithSubagentExecution(config).
			WithAgent("test", AgentDefinition{
				Description: "Test",
				Prompt:      "Test",
			})

		if opts.SubagentExecution == nil {
			t.Fatal("SubagentExecution should be set")
		}

		if opts.SubagentExecution.MultiInvocation != MultiInvocationModeParallel {
			t.Errorf("expected MultiInvocation to be parallel")
		}

		if opts.Model == nil || *opts.Model != "claude-opus-4-5-latest" {
			t.Errorf("Model should be set to claude-opus-4-5-latest")
		}

		if _, ok := opts.Agents["test"]; !ok {
			t.Errorf("Agent 'test' should be set")
		}
	})
}

// ---------------------------------------------------------------------------
// Phase C: Configuration Parity — Type Constants, JSON Roundtrip, Builders
// ---------------------------------------------------------------------------

// TestEffortLevelConstants verifies each EffortLevel constant maps to the
// correct string value expected by the Claude Code CLI.
func TestEffortLevelConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level EffortLevel
		want  string
	}{
		{name: "low", level: EffortLow, want: "low"},
		{name: "medium", level: EffortMedium, want: "medium"},
		{name: "high", level: EffortHigh, want: "high"},
		{name: "max", level: EffortMax, want: "max"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := string(tt.level); got != tt.want {
				t.Errorf("EffortLevel(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

// TestThinkingConfig_JSONRoundtrip verifies ThinkingConfig can be marshaled
// and unmarshaled without data loss for each variant (adaptive, enabled, disabled).
func TestThinkingConfig_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	budgetTokens := 10000

	tests := []struct {
		name   string
		config ThinkingConfig
		// checkJSON validates the intermediate JSON bytes if non-nil.
		checkJSON func(t *testing.T, data []byte)
	}{
		{
			name:   "adaptive type without budget",
			config: ThinkingConfig{Type: "adaptive"},
			checkJSON: func(t *testing.T, data []byte) {
				t.Helper()
				var m map[string]interface{}
				if err := json.Unmarshal(data, &m); err != nil {
					t.Fatalf("unmarshal to map: %v", err)
				}
				if _, ok := m["budgetTokens"]; ok {
					t.Error("budgetTokens should be omitted for adaptive config")
				}
			},
		},
		{
			name:   "enabled type with budget",
			config: ThinkingConfig{Type: "enabled", BudgetTokens: &budgetTokens},
			checkJSON: func(t *testing.T, data []byte) {
				t.Helper()
				var m map[string]interface{}
				if err := json.Unmarshal(data, &m); err != nil {
					t.Fatalf("unmarshal to map: %v", err)
				}
				v, ok := m["budgetTokens"]
				if !ok {
					t.Fatal("budgetTokens should be present for enabled config")
				}
				if int(v.(float64)) != 10000 {
					t.Errorf("budgetTokens = %v, want 10000", v)
				}
			},
		},
		{
			name:   "disabled type",
			config: ThinkingConfig{Type: "disabled"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			if tt.checkJSON != nil {
				tt.checkJSON(t, data)
			}

			var got ThinkingConfig
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			if got.Type != tt.config.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.config.Type)
			}

			if tt.config.BudgetTokens == nil {
				if got.BudgetTokens != nil {
					t.Errorf("BudgetTokens should be nil, got %d", *got.BudgetTokens)
				}
			} else {
				if got.BudgetTokens == nil {
					t.Fatal("BudgetTokens should not be nil")
				}
				if *got.BudgetTokens != *tt.config.BudgetTokens {
					t.Errorf("BudgetTokens = %d, want %d", *got.BudgetTokens, *tt.config.BudgetTokens)
				}
			}
		})
	}
}

// TestOutputFormat_JSONRoundtrip verifies OutputFormat marshal/unmarshal
// with and without schema and name.
func TestOutputFormat_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	schemaName := "my_schema"

	tests := []struct {
		name   string
		format OutputFormat
	}{
		{
			name: "json_schema with schema and name",
			format: OutputFormat{
				Type: "json_schema",
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"result": map[string]interface{}{
							"type": "string",
						},
					},
				},
				Name: &schemaName,
			},
		},
		{
			name: "json_schema without schema or name",
			format: OutputFormat{
				Type: "json_schema",
			},
		},
		{
			name: "json_schema with schema only",
			format: OutputFormat{
				Type: "json_schema",
				Schema: map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.format)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var got OutputFormat
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			if got.Type != tt.format.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.format.Type)
			}

			if !reflect.DeepEqual(got.Schema, tt.format.Schema) {
				t.Errorf("Schema mismatch: got %v, want %v", got.Schema, tt.format.Schema)
			}

			if tt.format.Name == nil {
				if got.Name != nil {
					t.Errorf("Name should be nil, got %q", *got.Name)
				}
			} else {
				if got.Name == nil {
					t.Fatal("Name should not be nil")
				}
				if *got.Name != *tt.format.Name {
					t.Errorf("Name = %q, want %q", *got.Name, *tt.format.Name)
				}
			}
		})
	}
}

// TestSandboxConfig_JSONRoundtrip verifies SandboxConfig marshal/unmarshal
// including nested network and filesystem configs.
func TestSandboxConfig_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	boolTrue := true
	boolFalse := false
	proxyPort := 8080

	tests := []struct {
		name   string
		config SandboxConfig
	}{
		{
			name: "full config with nested network and filesystem",
			config: SandboxConfig{
				Enabled:                  &boolTrue,
				AutoAllowBashIfSandboxed: &boolTrue,
				AllowUnsandboxedCommands: &boolFalse,
				Network: &SandboxNetworkConfig{
					AllowedDomains:          []string{"example.com", "api.example.com"},
					AllowManagedDomainsOnly: &boolTrue,
					AllowUnixSockets:        []string{"/var/run/docker.sock"},
					AllowAllUnixSockets:     &boolFalse,
					AllowLocalBinding:       &boolTrue,
					HttpProxyPort:           &proxyPort,
				},
				Filesystem: &SandboxFilesystemConfig{
					AllowWrite:                []string{"/tmp", "/home/user/project"},
					DenyWrite:                 []string{"/etc"},
					DenyRead:                  []string{"/root"},
					AllowRead:                 []string{"/usr/local"},
					AllowManagedReadPathsOnly: &boolFalse,
				},
				IgnoreViolations: map[string][]string{
					"network": {"dns"},
				},
				EnableWeakerNestedSandbox:    &boolFalse,
				EnableWeakerNetworkIsolation: &boolFalse,
				ExcludedCommands:             []string{"rm", "dd"},
			},
		},
		{
			name: "minimal config with enabled only",
			config: SandboxConfig{
				Enabled: &boolTrue,
			},
		},
		{
			name:   "empty config",
			config: SandboxConfig{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var got SandboxConfig
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			// Compare using JSON round-trip: marshal both and compare bytes.
			// This avoids deep comparison complexity with pointers.
			wantData, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("re-Marshal original error = %v", err)
			}
			gotData, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("re-Marshal roundtripped error = %v", err)
			}
			if string(gotData) != string(wantData) {
				t.Errorf("JSON roundtrip mismatch:\n  got:  %s\n  want: %s", gotData, wantData)
			}
		})
	}
}

// TestSandboxNetworkConfig_JSONRoundtrip verifies SandboxNetworkConfig in isolation.
func TestSandboxNetworkConfig_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	boolTrue := true
	boolFalse := false
	httpPort := 3128
	socksPort := 1080

	tests := []struct {
		name   string
		config SandboxNetworkConfig
	}{
		{
			name: "all fields populated",
			config: SandboxNetworkConfig{
				AllowedDomains:          []string{"*.example.com"},
				AllowManagedDomainsOnly: &boolTrue,
				AllowUnixSockets:        []string{"/run/app.sock"},
				AllowAllUnixSockets:     &boolFalse,
				AllowLocalBinding:       &boolTrue,
				HttpProxyPort:           &httpPort,
				SocksProxyPort:          &socksPort,
			},
		},
		{
			name:   "empty config",
			config: SandboxNetworkConfig{},
		},
		{
			name: "domains only",
			config: SandboxNetworkConfig{
				AllowedDomains: []string{"api.github.com"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var got SandboxNetworkConfig
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			wantData, _ := json.Marshal(tt.config)
			gotData, _ := json.Marshal(got)
			if string(gotData) != string(wantData) {
				t.Errorf("JSON roundtrip mismatch:\n  got:  %s\n  want: %s", gotData, wantData)
			}
		})
	}
}

// TestSandboxFilesystemConfig_JSONRoundtrip verifies SandboxFilesystemConfig in isolation.
func TestSandboxFilesystemConfig_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	boolTrue := true

	tests := []struct {
		name   string
		config SandboxFilesystemConfig
	}{
		{
			name: "all fields populated",
			config: SandboxFilesystemConfig{
				AllowWrite:                []string{"/tmp", "/var/data"},
				DenyWrite:                 []string{"/etc", "/usr"},
				DenyRead:                  []string{"/root/.ssh"},
				AllowRead:                 []string{"/opt/app"},
				AllowManagedReadPathsOnly: &boolTrue,
			},
		},
		{
			name:   "empty config",
			config: SandboxFilesystemConfig{},
		},
		{
			name: "write paths only",
			config: SandboxFilesystemConfig{
				AllowWrite: []string{"/tmp"},
				DenyWrite:  []string{"/etc"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var got SandboxFilesystemConfig
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			wantData, _ := json.Marshal(tt.config)
			gotData, _ := json.Marshal(got)
			if string(gotData) != string(wantData) {
				t.Errorf("JSON roundtrip mismatch:\n  got:  %s\n  want: %s", gotData, wantData)
			}
		})
	}
}

// TestWithEffort verifies the WithEffort builder sets the Effort pointer correctly.
func TestWithEffort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level EffortLevel
	}{
		{name: "low", level: EffortLow},
		{name: "medium", level: EffortMedium},
		{name: "high", level: EffortHigh},
		{name: "max", level: EffortMax},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := NewClaudeAgentOptions()
			result := opts.WithEffort(tt.level)

			if result != opts {
				t.Error("WithEffort should return the same instance for chaining")
			}
			if opts.Effort == nil {
				t.Fatal("Effort should not be nil after setting")
			}
			if *opts.Effort != tt.level {
				t.Errorf("Effort = %q, want %q", *opts.Effort, tt.level)
			}
		})
	}
}

// TestWithThinking verifies the WithThinking builder sets the Thinking pointer.
func TestWithThinking(t *testing.T) {
	t.Parallel()

	budgetTokens := 5000

	tests := []struct {
		name   string
		config ThinkingConfig
	}{
		{
			name:   "adaptive",
			config: ThinkingConfig{Type: "adaptive"},
		},
		{
			name:   "enabled with budget",
			config: ThinkingConfig{Type: "enabled", BudgetTokens: &budgetTokens},
		},
		{
			name:   "disabled",
			config: ThinkingConfig{Type: "disabled"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := NewClaudeAgentOptions()
			result := opts.WithThinking(tt.config)

			if result != opts {
				t.Error("WithThinking should return the same instance for chaining")
			}
			if opts.Thinking == nil {
				t.Fatal("Thinking should not be nil after setting")
			}
			if opts.Thinking.Type != tt.config.Type {
				t.Errorf("Thinking.Type = %q, want %q", opts.Thinking.Type, tt.config.Type)
			}
			if tt.config.BudgetTokens != nil {
				if opts.Thinking.BudgetTokens == nil {
					t.Fatal("Thinking.BudgetTokens should not be nil")
				}
				if *opts.Thinking.BudgetTokens != *tt.config.BudgetTokens {
					t.Errorf("Thinking.BudgetTokens = %d, want %d", *opts.Thinking.BudgetTokens, *tt.config.BudgetTokens)
				}
			}
		})
	}
}

// TestWithOutputFormat verifies the WithOutputFormat builder sets the pointer.
func TestWithOutputFormat(t *testing.T) {
	t.Parallel()

	schemaName := "test_schema"

	tests := []struct {
		name   string
		format OutputFormat
	}{
		{
			name: "json_schema with schema and name",
			format: OutputFormat{
				Type:   "json_schema",
				Schema: map[string]interface{}{"type": "object"},
				Name:   &schemaName,
			},
		},
		{
			name: "json_schema minimal",
			format: OutputFormat{
				Type: "json_schema",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := NewClaudeAgentOptions()
			result := opts.WithOutputFormat(tt.format)

			if result != opts {
				t.Error("WithOutputFormat should return the same instance for chaining")
			}
			if opts.OutputFormat == nil {
				t.Fatal("OutputFormat should not be nil after setting")
			}
			if opts.OutputFormat.Type != tt.format.Type {
				t.Errorf("OutputFormat.Type = %q, want %q", opts.OutputFormat.Type, tt.format.Type)
			}
			if !reflect.DeepEqual(opts.OutputFormat.Schema, tt.format.Schema) {
				t.Errorf("OutputFormat.Schema mismatch: got %v, want %v", opts.OutputFormat.Schema, tt.format.Schema)
			}
			if tt.format.Name != nil {
				if opts.OutputFormat.Name == nil {
					t.Fatal("OutputFormat.Name should not be nil")
				}
				if *opts.OutputFormat.Name != *tt.format.Name {
					t.Errorf("OutputFormat.Name = %q, want %q", *opts.OutputFormat.Name, *tt.format.Name)
				}
			}
		})
	}
}

// TestWithFallbackModel verifies the WithFallbackModel builder.
func TestWithFallbackModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		model string
	}{
		{name: "haiku model", model: "claude-3-haiku"},
		{name: "sonnet model", model: "claude-3-5-sonnet-20241022"},
		{name: "empty string", model: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := NewClaudeAgentOptions()
			result := opts.WithFallbackModel(tt.model)

			if result != opts {
				t.Error("WithFallbackModel should return the same instance for chaining")
			}
			if opts.FallbackModel == nil {
				t.Fatal("FallbackModel should not be nil after setting")
			}
			if *opts.FallbackModel != tt.model {
				t.Errorf("FallbackModel = %q, want %q", *opts.FallbackModel, tt.model)
			}
		})
	}
}

// TestWithEnableFileCheckpointing verifies the WithEnableFileCheckpointing builder.
func TestWithEnableFileCheckpointing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		enabled bool
	}{
		{name: "enabled", enabled: true},
		{name: "disabled", enabled: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := NewClaudeAgentOptions()
			result := opts.WithEnableFileCheckpointing(tt.enabled)

			if result != opts {
				t.Error("WithEnableFileCheckpointing should return the same instance for chaining")
			}
			if opts.EnableFileCheckpointing != tt.enabled {
				t.Errorf("EnableFileCheckpointing = %v, want %v", opts.EnableFileCheckpointing, tt.enabled)
			}
		})
	}
}

// TestWithSandbox verifies the WithSandbox builder sets the pointer correctly.
func TestWithSandbox(t *testing.T) {
	t.Parallel()

	boolTrue := true
	boolFalse := false

	tests := []struct {
		name   string
		config SandboxConfig
	}{
		{
			name: "full sandbox config",
			config: SandboxConfig{
				Enabled:                  &boolTrue,
				AutoAllowBashIfSandboxed: &boolTrue,
				Network: &SandboxNetworkConfig{
					AllowedDomains: []string{"example.com"},
				},
				Filesystem: &SandboxFilesystemConfig{
					AllowWrite: []string{"/tmp"},
				},
				ExcludedCommands: []string{"rm"},
			},
		},
		{
			name: "minimal sandbox",
			config: SandboxConfig{
				Enabled: &boolFalse,
			},
		},
		{
			name:   "empty sandbox",
			config: SandboxConfig{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := NewClaudeAgentOptions()
			result := opts.WithSandbox(tt.config)

			if result != opts {
				t.Error("WithSandbox should return the same instance for chaining")
			}
			if opts.Sandbox == nil {
				t.Fatal("Sandbox should not be nil after setting")
			}

			// Compare via JSON roundtrip for deep equality of pointer-heavy structs.
			wantData, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Marshal want: %v", err)
			}
			gotData, err := json.Marshal(*opts.Sandbox)
			if err != nil {
				t.Fatalf("Marshal got: %v", err)
			}
			if string(gotData) != string(wantData) {
				t.Errorf("Sandbox mismatch:\n  got:  %s\n  want: %s", gotData, wantData)
			}
		})
	}
}

// TestWithPersistSession verifies the WithPersistSession builder sets a bool pointer.
func TestWithPersistSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		persist bool
	}{
		{name: "persist true", persist: true},
		{name: "persist false", persist: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := NewClaudeAgentOptions()
			result := opts.WithPersistSession(tt.persist)

			if result != opts {
				t.Error("WithPersistSession should return the same instance for chaining")
			}
			if opts.PersistSession == nil {
				t.Fatal("PersistSession should not be nil after setting")
			}
			if *opts.PersistSession != tt.persist {
				t.Errorf("PersistSession = %v, want %v", *opts.PersistSession, tt.persist)
			}
		})
	}
}

// TestWithSessionID verifies the WithSessionID builder sets a string pointer.
func TestWithSessionID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
	}{
		{name: "uuid style", id: "abc-123-def-456"},
		{name: "simple id", id: "session-1"},
		{name: "empty string", id: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := NewClaudeAgentOptions()
			result := opts.WithSessionID(tt.id)

			if result != opts {
				t.Error("WithSessionID should return the same instance for chaining")
			}
			if opts.SessionID == nil {
				t.Fatal("SessionID should not be nil after setting")
			}
			if *opts.SessionID != tt.id {
				t.Errorf("SessionID = %q, want %q", *opts.SessionID, tt.id)
			}
		})
	}
}

// TestWithPromptSuggestions verifies the WithPromptSuggestions builder.
func TestWithPromptSuggestions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		enabled bool
	}{
		{name: "enabled", enabled: true},
		{name: "disabled", enabled: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := NewClaudeAgentOptions()
			result := opts.WithPromptSuggestions(tt.enabled)

			if result != opts {
				t.Error("WithPromptSuggestions should return the same instance for chaining")
			}
			if opts.PromptSuggestions != tt.enabled {
				t.Errorf("PromptSuggestions = %v, want %v", opts.PromptSuggestions, tt.enabled)
			}
		})
	}
}

// TestPhaseCBuilderChaining verifies all 9 Phase C builders chain correctly
// in a single fluent expression together with existing builders.
func TestPhaseCBuilderChaining(t *testing.T) {
	t.Parallel()

	boolTrue := true
	budgetTokens := 8000

	opts := NewClaudeAgentOptions().
		WithModel("claude-opus-4-5-latest").
		WithMaxTurns(20).
		WithEffort(EffortHigh).
		WithThinking(ThinkingConfig{Type: "enabled", BudgetTokens: &budgetTokens}).
		WithOutputFormat(OutputFormat{Type: "json_schema", Schema: map[string]interface{}{"type": "object"}}).
		WithFallbackModel("claude-3-haiku").
		WithEnableFileCheckpointing(true).
		WithSandbox(SandboxConfig{Enabled: &boolTrue}).
		WithPersistSession(false).
		WithSessionID("chain-test-123").
		WithPromptSuggestions(true)

	// Verify all Phase C fields
	if opts.Effort == nil || *opts.Effort != EffortHigh {
		t.Error("Effort not set correctly in chain")
	}
	if opts.Thinking == nil || opts.Thinking.Type != "enabled" {
		t.Error("Thinking not set correctly in chain")
	}
	if opts.Thinking.BudgetTokens == nil || *opts.Thinking.BudgetTokens != 8000 {
		t.Error("Thinking.BudgetTokens not set correctly in chain")
	}
	if opts.OutputFormat == nil || opts.OutputFormat.Type != "json_schema" {
		t.Error("OutputFormat not set correctly in chain")
	}
	if opts.FallbackModel == nil || *opts.FallbackModel != "claude-3-haiku" {
		t.Error("FallbackModel not set correctly in chain")
	}
	if !opts.EnableFileCheckpointing {
		t.Error("EnableFileCheckpointing not set correctly in chain")
	}
	if opts.Sandbox == nil || opts.Sandbox.Enabled == nil || !*opts.Sandbox.Enabled {
		t.Error("Sandbox not set correctly in chain")
	}
	if opts.PersistSession == nil || *opts.PersistSession != false {
		t.Error("PersistSession not set correctly in chain")
	}
	if opts.SessionID == nil || *opts.SessionID != "chain-test-123" {
		t.Error("SessionID not set correctly in chain")
	}
	if !opts.PromptSuggestions {
		t.Error("PromptSuggestions not set correctly in chain")
	}

	// Verify pre-existing builders still work alongside Phase C builders
	if opts.Model == nil || *opts.Model != "claude-opus-4-5-latest" {
		t.Error("Model not set correctly in chain")
	}
	if opts.MaxTurns == nil || *opts.MaxTurns != 20 {
		t.Error("MaxTurns not set correctly in chain")
	}
}

// TestNewClaudeAgentOptions_PhaseCDefaults verifies that all Phase C fields
// are nil/zero by default in a freshly-created options instance.
func TestNewClaudeAgentOptions_PhaseCDefaults(t *testing.T) {
	t.Parallel()

	opts := NewClaudeAgentOptions()

	if opts.Effort != nil {
		t.Errorf("Effort should be nil by default, got %q", *opts.Effort)
	}
	if opts.Thinking != nil {
		t.Error("Thinking should be nil by default")
	}
	if opts.OutputFormat != nil {
		t.Error("OutputFormat should be nil by default")
	}
	if opts.FallbackModel != nil {
		t.Errorf("FallbackModel should be nil by default, got %q", *opts.FallbackModel)
	}
	if opts.EnableFileCheckpointing {
		t.Error("EnableFileCheckpointing should be false by default")
	}
	if opts.Sandbox != nil {
		t.Error("Sandbox should be nil by default")
	}
	if opts.PersistSession != nil {
		t.Error("PersistSession should be nil by default")
	}
	if opts.SessionID != nil {
		t.Errorf("SessionID should be nil by default, got %q", *opts.SessionID)
	}
	if opts.PromptSuggestions {
		t.Error("PromptSuggestions should be false by default")
	}
}

// --- Fuzz Tests (Phase C) ---

func FuzzThinkingConfig_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"type":"adaptive"}`))
	f.Add([]byte(`{"type":"enabled","budgetTokens":10000}`))
	f.Add([]byte(`{"type":"disabled"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"type":"","budgetTokens":null}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var tc ThinkingConfig
		_ = json.Unmarshal(data, &tc)
	})
}

func FuzzOutputFormat_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"type":"json_schema","schema":{"type":"object"},"name":"test"}`))
	f.Add([]byte(`{"type":"json_schema"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"type":"","schema":null}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var of OutputFormat
		_ = json.Unmarshal(data, &of)
	})
}

func FuzzSandboxConfig_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"enabled":true,"network":{"allowedDomains":["example.com"]}}`))
	f.Add([]byte(`{"enabled":false,"filesystem":{"allowWrite":["/tmp"]}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"autoAllowBashIfSandboxed":true,"excludedCommands":["rm"]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var sc SandboxConfig
		_ = json.Unmarshal(data, &sc)
	})
}
