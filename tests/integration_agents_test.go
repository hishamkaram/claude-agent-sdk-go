//go:build integration
// +build integration

package tests

import (
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestCLI_AgentDefinitionsAdvertisedInInit(t *testing.T) {
	background := true
	maxTurns := 1
	effort := types.EffortLow
	permissionMode := types.PermissionModePlan
	color := types.AgentColorCyan

	client, _ := setupClient(t, func(opts *types.ClaudeAgentOptions) {
		opts.WithAgent("sdk-parity-reviewer", types.AgentDefinition{
			Description:     "SDK parity integration reviewer",
			Prompt:          "Review SDK CLI parity and report only factual drift.",
			Tools:           []string{"Read", "Grep"},
			DisallowedTools: []string{"Bash", "Write", "Edit"},
			MaxTurns:        &maxTurns,
			Background:      &background,
			Effort:          &effort,
			PermissionMode:  &permissionMode,
			Color:           &color,
		})
	})

	agent, ok := findAgent(client.SupportedAgents(), "sdk-parity-reviewer")
	if !ok {
		t.Fatalf("dynamic --agents definition was not advertised by real CLI init; agents=%v", client.SupportedAgents())
	}
	if agent.Description != "SDK parity integration reviewer" {
		t.Errorf("agent description = %q, want %q", agent.Description, "SDK parity integration reviewer")
	}
}

func TestCLI_SessionAgentFlagConnects(t *testing.T) {
	client, _ := setupClient(t, func(opts *types.ClaudeAgentOptions) {
		opts.WithAgent("sdk-session-agent", types.AgentDefinition{
			Description: "SDK session agent",
			Prompt:      "You are a session-scoped SDK integration test agent.",
		})
		opts.WithSessionAgent("sdk-session-agent")
	})

	if _, ok := findAgent(client.SupportedAgents(), "sdk-session-agent"); !ok {
		t.Fatalf("--agent target was not advertised by real CLI init; agents=%v", client.SupportedAgents())
	}
}

func findAgent(agents []types.AgentInfo, name string) (types.AgentInfo, bool) {
	for _, agent := range agents {
		if agent.Name == name {
			return agent, true
		}
	}
	return types.AgentInfo{}, false
}
