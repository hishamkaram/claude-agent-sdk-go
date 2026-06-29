package types

import (
	"testing"
)

// TestWithTaskBudget verifies the WithTaskBudget builder sets the TaskBudget pointer correctly.
func TestWithTaskBudget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		budget float64
	}{
		{name: "positive budget", budget: 5.0},
		{name: "zero budget", budget: 0.0},
		{name: "large budget", budget: 100.50},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := NewClaudeAgentOptions()
			result := opts.WithTaskBudget(tt.budget)

			if result != opts {
				t.Error("WithTaskBudget should return the same options instance for chaining")
			}
			if opts.TaskBudget == nil {
				t.Fatal("TaskBudget should not be nil after WithTaskBudget")
			}
			if *opts.TaskBudget != tt.budget {
				t.Errorf("TaskBudget = %f, want %f", *opts.TaskBudget, tt.budget)
			}
		})
	}
}

// TestWithTaskBudget_NilByDefault verifies TaskBudget is nil when not set.
func TestWithTaskBudget_NilByDefault(t *testing.T) {
	t.Parallel()
	opts := NewClaudeAgentOptions()
	if opts.TaskBudget != nil {
		t.Errorf("TaskBudget should be nil by default, got %v", *opts.TaskBudget)
	}
}

// TestWithAgentProgressSummaries verifies the WithAgentProgressSummaries builder.
func TestWithAgentProgressSummaries(t *testing.T) {
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
			result := opts.WithAgentProgressSummaries(tt.enabled)

			if result != opts {
				t.Error("WithAgentProgressSummaries should return the same options instance for chaining")
			}
			if opts.AgentProgressSummaries != tt.enabled {
				t.Errorf("AgentProgressSummaries = %v, want %v", opts.AgentProgressSummaries, tt.enabled)
			}
		})
	}
}

// TestWithIncludeHookEvents verifies the WithIncludeHookEvents builder.
func TestWithIncludeHookEvents(t *testing.T) {
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
			result := opts.WithIncludeHookEvents(tt.enabled)

			if result != opts {
				t.Error("WithIncludeHookEvents should return the same options instance for chaining")
			}
			if opts.IncludeHookEvents != tt.enabled {
				t.Errorf("IncludeHookEvents = %v, want %v", opts.IncludeHookEvents, tt.enabled)
			}
		})
	}
}
