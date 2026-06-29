package claude

import (
	"context"
	"errors"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// --- Phase B: New method tests ---

// TestInterrupt_BeforeConnect ensures Interrupt returns connection error when not connected.
func TestInterrupt_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.Interrupt(ctx)
	if err == nil {
		t.Fatal("expected error when calling Interrupt before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestStreamInput_BeforeConnect ensures StreamInput returns connection error when not connected.
func TestStreamInput_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.StreamInput(ctx, "hello")
	if err == nil {
		t.Fatal("expected error when calling StreamInput before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestStreamInput_EmptyContent ensures StreamInput rejects empty content.
func TestStreamInput_EmptyContent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.StreamInput(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

// TestStopTask_BeforeConnect ensures StopTask returns connection error when not connected.
func TestStopTask_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.StopTask(ctx, "task-123")
	if err == nil {
		t.Fatal("expected error when calling StopTask before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestStopTask_EmptyTaskID ensures StopTask rejects empty task ID.
func TestStopTask_EmptyTaskID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.StopTask(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty taskID")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

// TestMCPServerStatus_BeforeConnect ensures MCPServerStatus returns connection error when not connected.
func TestMCPServerStatus_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	_, err = client.MCPServerStatus(ctx)
	if err == nil {
		t.Fatal("expected error when calling MCPServerStatus before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestReconnectMCPServer_BeforeConnect ensures ReconnectMCPServer returns connection error when not connected.
func TestReconnectMCPServer_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.ReconnectMCPServer(ctx, "my-server")
	if err == nil {
		t.Fatal("expected error when calling ReconnectMCPServer before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestReconnectMCPServer_EmptyName ensures ReconnectMCPServer rejects empty server name.
func TestReconnectMCPServer_EmptyName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.ReconnectMCPServer(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty serverName")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

// TestToggleMCPServer_BeforeConnect ensures ToggleMCPServer returns connection error when not connected.
func TestToggleMCPServer_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.ToggleMCPServer(ctx, "my-server", true)
	if err == nil {
		t.Fatal("expected error when calling ToggleMCPServer before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestToggleMCPServer_EmptyName ensures ToggleMCPServer rejects empty server name.
func TestToggleMCPServer_EmptyName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.ToggleMCPServer(ctx, "", false)
	if err == nil {
		t.Fatal("expected error for empty serverName")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

// TestSetMCPServers_BeforeConnect ensures SetMCPServers returns connection error when not connected.
func TestSetMCPServers_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	_, err = client.SetMCPServers(ctx, map[string]interface{}{"server1": map[string]interface{}{}})
	if err == nil {
		t.Fatal("expected error when calling SetMCPServers before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestSetMCPServers_NilConfig ensures SetMCPServers rejects nil servers config.
func TestSetMCPServers_NilConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	_, err = client.SetMCPServers(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil servers")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

// TestRewindFiles_BeforeConnect ensures RewindFiles returns connection error when not connected.
func TestRewindFiles_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	_, err = client.RewindFiles(ctx, "msg-123", false)
	if err == nil {
		t.Fatal("expected error when calling RewindFiles before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestRewindFiles_EmptyUserMessageID ensures RewindFiles rejects empty user message ID.
func TestRewindFiles_EmptyUserMessageID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	_, err = client.RewindFiles(ctx, "", false)
	if err == nil {
		t.Fatal("expected error for empty userMessageID")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

// TestSupportedAgents_BeforeConnect ensures SupportedAgents returns nil when not connected.
func TestSupportedAgents_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	agents := client.SupportedAgents()
	if agents != nil {
		t.Errorf("expected nil before Connect(), got %v", agents)
	}
}

// TestSupportedAgents_ReturnsFromInitResult verifies SupportedAgents uses the stored init result.
func TestSupportedAgents_ReturnsFromInitResult(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	client.mu.Lock()
	client.initResult = &types.InitializeResult{
		Agents: []types.AgentInfo{
			{Name: "Explore", Description: "Fast agent for exploring codebases", Model: "sonnet"},
			{Name: "Plan", Description: "Software architect agent"},
		},
	}
	client.mu.Unlock()

	agents := client.SupportedAgents()
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	if agents[0].Name != "Explore" {
		t.Errorf("expected 'Explore', got %q", agents[0].Name)
	}
	if agents[0].Model != "sonnet" {
		t.Errorf("expected model 'sonnet', got %q", agents[0].Model)
	}
	if agents[1].Name != "Plan" {
		t.Errorf("expected 'Plan', got %q", agents[1].Name)
	}
	if agents[1].Model != "" {
		t.Errorf("expected empty model, got %q", agents[1].Model)
	}
}

// TestParseInitResult_WithAgents verifies that agents are parsed from the init result.
func TestParseInitResult_WithAgents(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"agents": []interface{}{
			map[string]interface{}{
				"name":        "Explore",
				"description": "Fast agent for exploring codebases",
				"model":       "sonnet",
			},
			map[string]interface{}{
				"name":        "Plan",
				"description": "Software architect agent",
			},
		},
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(result.Agents))
	}
	if result.Agents[0].Name != "Explore" {
		t.Errorf("expected 'Explore', got %q", result.Agents[0].Name)
	}
	if result.Agents[0].Description != "Fast agent for exploring codebases" {
		t.Errorf("unexpected description: %q", result.Agents[0].Description)
	}
	if result.Agents[0].Model != "sonnet" {
		t.Errorf("expected model 'sonnet', got %q", result.Agents[0].Model)
	}
	if result.Agents[1].Name != "Plan" {
		t.Errorf("expected 'Plan', got %q", result.Agents[1].Name)
	}
	if result.Agents[1].Model != "" {
		t.Errorf("expected empty model for Plan, got %q", result.Agents[1].Model)
	}
}

// TestParseInitResult_AgentsSkipsEmptyName verifies agents with empty names are skipped.
func TestParseInitResult_AgentsSkipsEmptyName(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"agents": []interface{}{
			map[string]interface{}{
				"name":        "Valid",
				"description": "A valid agent",
			},
			map[string]interface{}{
				"description": "Missing name",
			},
			map[string]interface{}{
				"name":        "",
				"description": "Empty name",
			},
		},
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Agents) != 1 {
		t.Fatalf("expected 1 agent (skipping empty names), got %d", len(result.Agents))
	}
	if result.Agents[0].Name != "Valid" {
		t.Errorf("expected 'Valid', got %q", result.Agents[0].Name)
	}
}

// TestParseInitResult_AgentsInvalidType verifies graceful handling when agents is not an array.
func TestParseInitResult_AgentsInvalidType(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"agents": "not-an-array",
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Agents) != 0 {
		t.Errorf("expected 0 agents for invalid type, got %d", len(result.Agents))
	}
}

// TestParseInitResult_AllFieldsTogether verifies commands, models, and agents together.
func TestParseInitResult_AllFieldsTogether(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"commands": []interface{}{
			map[string]interface{}{
				"name":        "compact",
				"description": "Compact context",
			},
		},
		"models": []interface{}{
			map[string]interface{}{
				"value":       "haiku",
				"displayName": "Haiku",
			},
		},
		"agents": []interface{}{
			map[string]interface{}{
				"name":        "Explore",
				"description": "Explorer",
			},
		},
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(result.Commands))
	}
	if len(result.Models) != 1 {
		t.Errorf("expected 1 model, got %d", len(result.Models))
	}
	if len(result.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(result.Agents))
	}
}
