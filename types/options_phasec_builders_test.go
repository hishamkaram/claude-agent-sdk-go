package types

import (
	"encoding/json"
	"reflect"
	"testing"
)

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
		{name: "xhigh", level: EffortXHigh},
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
