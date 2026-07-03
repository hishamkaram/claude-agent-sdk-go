//go:build integration
// +build integration

package tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

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

func TestCLI_NamedAgentFullCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	requireRunTurns(t)

	const agentName = "agentd-live-echo"
	proofToken := fmt.Sprintf("AGENTD_NAMED_AGENT_LIVE_OK_%d", time.Now().UnixNano())
	finalText := "MAIN_RECEIVED: " + proofToken
	background := false

	client, ctx := setupClient(t, func(opts *types.ClaudeAgentOptions) {
		opts.WithTools([]string{"Agent"}).
			WithAllowedTools("Agent").
			WithAgent(agentName, types.AgentDefinition{
				Description: "Use this agent when asked to provide AgentD named-agent live proof.",
				Prompt:      "You are agentd-live-echo. Reply with exactly " + proofToken + " and no other text.",
				Background:  &background,
			})
	})
	if _, ok := findAgent(client.SupportedAgents(), agentName); !ok {
		t.Fatalf("dynamic named agent %q was not advertised by real CLI init; agents=%v", agentName, client.SupportedAgents())
	}

	prompt := strings.Join([]string{
		"Use the Agent tool exactly once.",
		"Invoke subagent_type " + agentName + " with description 'Return your configured proof token exactly'.",
		"After launching the agent, do not guess or restate its output unless it is already available.",
		"Do not use any tool except Agent.",
	}, "\n")
	if err := client.Query(ctx, prompt); err != nil {
		t.Fatalf("Query named-agent prompt: %v", err)
	}

	var (
		agentToolUseID      string
		sawAgentToolUse     bool
		sawChildWithParent  bool
		sawTaskNotification bool
		sawFinalMain        bool
		sawFinalResult      bool
		trace               []string
	)
	for msg := range client.ReceiveResponse(ctx) {
		trace = append(trace, "turn1: "+namedAgentTrace(msg))
		switch m := msg.(type) {
		case *types.AssistantMessage:
			parent := ""
			if m.ParentToolUseID != nil {
				parent = *m.ParentToolUseID
			}
			for _, block := range m.Content {
				switch b := block.(type) {
				case *types.ToolUseBlock:
					if b.Name == "Agent" {
						sawAgentToolUse = true
						agentToolUseID = b.ID
						if got := agentInputString(b.Input, "subagent_type"); got != agentName {
							t.Fatalf("Agent tool subagent_type = %q, want %q", got, agentName)
						}
					}
				case *types.TextBlock:
					if parent == agentToolUseID && strings.Contains(b.Text, proofToken) {
						sawChildWithParent = true
					}
					if parent == "" && strings.Contains(b.Text, finalText) {
						sawFinalMain = true
					}
				}
			}
		case *types.ResultMessage:
			if m.Result != nil && strings.Contains(*m.Result, finalText) {
				sawFinalResult = true
			}
		case *types.TaskNotificationMessage:
			if m.ToolUseID != nil && *m.ToolUseID == agentToolUseID && m.Status == "completed" {
				sawTaskNotification = true
			}
		}
	}

	if !sawAgentToolUse {
		t.Fatalf("Claude did not invoke the Agent tool\ntrace:\n%s", strings.Join(trace, "\n"))
	}
	if agentToolUseID == "" {
		t.Fatalf("Agent tool_use id was empty\ntrace:\n%s", strings.Join(trace, "\n"))
	}
	if !sawChildWithParent {
		t.Fatalf("no child assistant text with parent_tool_use_id=%q and token %q\ntrace:\n%s",
			agentToolUseID, proofToken, strings.Join(trace, "\n"))
	}
	if !sawTaskNotification {
		t.Fatalf("named agent task did not complete with task_notification for tool_use_id=%q\ntrace:\n%s",
			agentToolUseID, strings.Join(trace, "\n"))
	}
	if !sawFinalMain || !sawFinalResult {
		followup := strings.Join([]string{
			"The named agent has completed.",
			"Read the completed child assistant output already in this conversation.",
			"Reply exactly with MAIN_RECEIVED: followed by that child output.",
			"Do not call tools.",
		}, "\n")
		if err := client.Query(ctx, followup); err != nil {
			t.Fatalf("Query named-agent follow-up: %v\ntrace:\n%s", err, strings.Join(trace, "\n"))
		}
		for msg := range client.ReceiveResponse(ctx) {
			trace = append(trace, "turn2: "+namedAgentTrace(msg))
			switch m := msg.(type) {
			case *types.AssistantMessage:
				parent := ""
				if m.ParentToolUseID != nil {
					parent = *m.ParentToolUseID
				}
				for _, block := range m.Content {
					switch b := block.(type) {
					case *types.ToolUseBlock:
						if b.Name == "Agent" {
							t.Fatalf("follow-up unexpectedly invoked Agent tool\ntrace:\n%s", strings.Join(trace, "\n"))
						}
					case *types.TextBlock:
						if parent == "" && strings.Contains(b.Text, finalText) {
							sawFinalMain = true
						}
					}
				}
			case *types.ResultMessage:
				if m.Result != nil && strings.Contains(*m.Result, finalText) {
					sawFinalResult = true
				}
			}
		}
	}
	if !sawFinalMain {
		t.Fatalf("main assistant did not return final text %q after child completion\ntrace:\n%s",
			finalText, strings.Join(trace, "\n"))
	}
	if !sawFinalResult {
		t.Fatalf("final ResultMessage did not contain %q\ntrace:\n%s", finalText, strings.Join(trace, "\n"))
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

func agentInputString(input map[string]interface{}, key string) string {
	if input == nil {
		return ""
	}
	value, _ := input[key].(string)
	return value
}

func namedAgentTrace(msg types.Message) string {
	switch m := msg.(type) {
	case *types.AssistantMessage:
		parent := ""
		if m.ParentToolUseID != nil {
			parent = *m.ParentToolUseID
		}
		parts := make([]string, 0, len(m.Content))
		for _, block := range m.Content {
			switch b := block.(type) {
			case *types.ToolUseBlock:
				parts = append(parts, fmt.Sprintf("tool=%s id=%s", b.Name, b.ID))
			case *types.TextBlock:
				parts = append(parts, fmt.Sprintf("text=%q", b.Text))
			default:
				parts = append(parts, fmt.Sprintf("block=%T", block))
			}
		}
		return fmt.Sprintf("assistant parent=%q %s", parent, strings.Join(parts, " | "))
	case *types.ResultMessage:
		result := ""
		if m.Result != nil {
			result = *m.Result
		}
		return fmt.Sprintf("result subtype=%q result=%q", m.Subtype, result)
	case *types.TaskStartedMessage:
		taskType := ""
		if m.TaskType != nil {
			taskType = *m.TaskType
		}
		toolUseID := ""
		if m.ToolUseID != nil {
			toolUseID = *m.ToolUseID
		}
		return fmt.Sprintf("task_started id=%q tool_use_id=%q task_type=%q", m.TaskID, toolUseID, taskType)
	case *types.TaskUpdatedMessage:
		toolUseID := ""
		if m.ToolUseID != nil {
			toolUseID = *m.ToolUseID
		}
		return fmt.Sprintf("task_updated id=%q tool_use_id=%q status=%q", m.TaskID, toolUseID, m.Patch.Status)
	case *types.TaskNotificationMessage:
		toolUseID := ""
		if m.ToolUseID != nil {
			toolUseID = *m.ToolUseID
		}
		return fmt.Sprintf("task_notification id=%q tool_use_id=%q status=%q", m.TaskID, toolUseID, m.Status)
	default:
		return fmt.Sprintf("%T type=%q", msg, msg.GetMessageType())
	}
}
