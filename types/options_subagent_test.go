package types

import (
	"testing"
)

// TestSubagentExecutionModeConstants tests the SubagentExecutionMode enum values.
func TestSubagentExecutionModeConstants(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TestAgentDefinitionWithExecutionControl keeps deprecated AgentDefinition
// execution-control fields source-compatible. Wire emission is covered by
// TestAgentDefinition_CurrentWireKeys.
func TestAgentDefinitionWithExecutionControl(t *testing.T) {
	t.Parallel()
	t.Run("agent with execution mode", func(t *testing.T) {
		mode := SubagentExecutionModeParallel
		agent := AgentDefinition{
			Description:   "Test agent",
			Prompt:        "Test prompt",
			ExecutionMode: &mode,
		}

		if agent.Description != "Test agent" {
			t.Errorf("expected Description %q, got %q", "Test agent", agent.Description)
		}
		if agent.Prompt != "Test prompt" {
			t.Errorf("expected Prompt %q, got %q", "Test prompt", agent.Prompt)
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

		if agent.Description != "Test agent" {
			t.Errorf("expected Description %q, got %q", "Test agent", agent.Description)
		}
		if agent.Prompt != "Test prompt" {
			t.Errorf("expected Prompt %q, got %q", "Test prompt", agent.Prompt)
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

		if agent.Description != "Test agent" {
			t.Errorf("expected Description %q, got %q", "Test agent", agent.Description)
		}
		if agent.Prompt != "Test prompt" {
			t.Errorf("expected Prompt %q, got %q", "Test prompt", agent.Prompt)
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

		if agent.Description != "Full agent" {
			t.Errorf("expected Description %q, got %q", "Full agent", agent.Description)
		}

		if agent.Prompt != "Full prompt" {
			t.Errorf("expected Prompt %q, got %q", "Full prompt", agent.Prompt)
		}

		if len(agent.Tools) != 2 || agent.Tools[0] != "Read" || agent.Tools[1] != "Write" {
			t.Errorf("expected Tools [Read Write], got %v", agent.Tools)
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
	t.Parallel()
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
