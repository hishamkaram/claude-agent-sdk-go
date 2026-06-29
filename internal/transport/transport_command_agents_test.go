package transport

import (
	"encoding/json"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestBuildCommandArgs_Agents tests agent configuration JSON serialization
func TestBuildCommandArgs_Agents(t *testing.T) {
	t.Parallel()
	t.Run("single agent with all fields", func(t *testing.T) {
		maxTurns := 5
		modelStr := "claude-opus-4-5-latest"
		background := true
		effort := types.EffortHigh
		permissionMode := types.PermissionModePlan
		memory := types.SettingSourceProject
		initialPrompt := "Start by reading README.md"
		isolation := types.AgentIsolationWorktree
		color := types.AgentColorGreen
		matcher := "Bash"

		opts := types.NewClaudeAgentOptions().
			WithAgent("search", types.AgentDefinition{
				Description:     "Search agent",
				Prompt:          "Search for information",
				Tools:           []string{"Read", "Glob"},
				DisallowedTools: []string{"Bash"},
				Model:           &modelStr,
				MaxTurns:        &maxTurns,
				McpServers:      []interface{}{"scanner"},
				Hooks: map[types.HookEvent][]types.AgentHookMatcher{
					types.HookEventPreToolUse: {
						{
							Matcher: &matcher,
							Hooks: []types.AgentHookHandler{
								{"type": "command", "command": "./scripts/validate-bash.sh"},
							},
						},
					},
				},
				Skills:         []string{"search-skill"},
				InitialPrompt:  &initialPrompt,
				Background:     &background,
				Effort:         &effort,
				PermissionMode: &permissionMode,
				Memory:         &memory,
				Isolation:      &isolation,
				Color:          &color,
			})

		transport := NewSubprocessCLITransport(
			"claude",
			"",
			nil,
			log.NewLogger(false),
			"",
			opts,
		)

		args := transport.buildCommandArgs()

		// Verify --agents flag is present
		agentsIdx := -1
		for i, arg := range args {
			if arg == "--agents" {
				agentsIdx = i
				break
			}
		}

		if agentsIdx == -1 {
			t.Fatal("--agents flag not found in command arguments")
		}

		if agentsIdx+1 >= len(args) {
			t.Fatal("--agents flag has no value")
		}

		agentsJSON := args[agentsIdx+1]

		// Verify JSON can be unmarshaled
		var agentsData map[string]map[string]interface{}
		if err := json.Unmarshal([]byte(agentsJSON), &agentsData); err != nil {
			t.Fatalf("Failed to unmarshal agents JSON: %v", err)
		}

		// Verify agent exists
		searchAgent, ok := agentsData["search"]
		if !ok {
			t.Fatal("Agent 'search' not found in JSON")
		}

		// Verify fields
		if searchAgent["description"] != "Search agent" {
			t.Errorf("Expected description 'Search agent', got %v", searchAgent["description"])
		}

		if searchAgent["prompt"] != "Search for information" {
			t.Errorf("Expected prompt 'Search for information', got %v", searchAgent["prompt"])
		}

		if disallowed, ok := searchAgent["disallowedTools"].([]interface{}); !ok || len(disallowed) != 1 || disallowed[0] != "Bash" {
			t.Errorf("Expected disallowedTools [Bash], got %v", searchAgent["disallowedTools"])
		}

		if servers, ok := searchAgent["mcpServers"].([]interface{}); !ok || len(servers) != 1 || servers[0] != "scanner" {
			t.Errorf("Expected mcpServers [scanner], got %v", searchAgent["mcpServers"])
		}

		if hooks, ok := searchAgent["hooks"].(map[string]interface{}); !ok || len(hooks) != 1 {
			t.Errorf("Expected hooks with one event, got %v", searchAgent["hooks"])
		}

		if maxTurnsVal, ok := searchAgent["maxTurns"].(float64); !ok || maxTurnsVal != 5 {
			t.Errorf("Expected maxTurns 5, got %v", searchAgent["maxTurns"])
		}

		if searchAgent["initialPrompt"] != initialPrompt {
			t.Errorf("Expected initialPrompt %q, got %v", initialPrompt, searchAgent["initialPrompt"])
		}

		if searchAgent["background"] != true {
			t.Errorf("Expected background true, got %v", searchAgent["background"])
		}

		if searchAgent["effort"] != "high" {
			t.Errorf("Expected effort high, got %v", searchAgent["effort"])
		}

		if searchAgent["permissionMode"] != "plan" {
			t.Errorf("Expected permissionMode plan, got %v", searchAgent["permissionMode"])
		}

		if searchAgent["memory"] != "project" {
			t.Errorf("Expected memory project, got %v", searchAgent["memory"])
		}

		if searchAgent["isolation"] != "worktree" {
			t.Errorf("Expected isolation worktree, got %v", searchAgent["isolation"])
		}

		if searchAgent["color"] != "green" {
			t.Errorf("Expected color green, got %v", searchAgent["color"])
		}

		if _, ok := searchAgent["execution_mode"]; ok {
			t.Error("legacy execution_mode should not be emitted")
		}
		if _, ok := searchAgent["timeout"]; ok {
			t.Error("legacy timeout should not be emitted")
		}
		if _, ok := searchAgent["max_turns"]; ok {
			t.Error("legacy max_turns should not be emitted")
		}
	})

	t.Run("multiple agents with different configs", func(t *testing.T) {
		background := true
		effort := types.EffortMedium

		opts := types.NewClaudeAgentOptions().
			WithAgent("agent1", types.AgentDefinition{
				Description: "First agent",
				Prompt:      "First prompt",
				Background:  &background,
			}).
			WithAgent("agent2", types.AgentDefinition{
				Description: "Second agent",
				Prompt:      "Second prompt",
				Effort:      &effort,
			})

		transport := NewSubprocessCLITransport(
			"claude",
			"",
			nil,
			log.NewLogger(false),
			"",
			opts,
		)

		args := transport.buildCommandArgs()

		agentsIdx := -1
		for i, arg := range args {
			if arg == "--agents" {
				agentsIdx = i
				break
			}
		}

		if agentsIdx == -1 {
			t.Fatal("--agents flag not found")
		}

		agentsJSON := args[agentsIdx+1]
		var agentsData map[string]map[string]interface{}
		if err := json.Unmarshal([]byte(agentsJSON), &agentsData); err != nil {
			t.Fatalf("Failed to unmarshal agents JSON: %v", err)
		}

		if len(agentsData) != 2 {
			t.Errorf("Expected 2 agents, got %d", len(agentsData))
		}

		if agentsData["agent1"]["background"] != true {
			t.Error("agent1 should have background true")
		}

		if agentsData["agent2"]["effort"] != "medium" {
			t.Error("agent2 should have medium effort")
		}
	})

	t.Run("agent with only required fields", func(t *testing.T) {
		opts := types.NewClaudeAgentOptions().
			WithAgent("simple", types.AgentDefinition{
				Description: "Simple agent",
				Prompt:      "Simple prompt",
			})

		transport := NewSubprocessCLITransport(
			"claude",
			"",
			nil,
			log.NewLogger(false),
			"",
			opts,
		)

		args := transport.buildCommandArgs()

		agentsIdx := -1
		for i, arg := range args {
			if arg == "--agents" {
				agentsIdx = i
				break
			}
		}

		if agentsIdx == -1 {
			t.Fatal("--agents flag not found")
		}

		agentsJSON := args[agentsIdx+1]
		var agentsData map[string]map[string]interface{}
		if err := json.Unmarshal([]byte(agentsJSON), &agentsData); err != nil {
			t.Fatalf("Failed to unmarshal agents JSON: %v", err)
		}

		simpleAgent := agentsData["simple"]

		// Verify required fields are present
		if simpleAgent["description"] != "Simple agent" {
			t.Error("description should be present")
		}
		if simpleAgent["prompt"] != "Simple prompt" {
			t.Error("prompt should be present")
		}

		// Verify optional fields are absent (not in JSON)
		if _, ok := simpleAgent["execution_mode"]; ok {
			t.Error("execution_mode should not be in JSON when not set")
		}
		if _, ok := simpleAgent["timeout"]; ok {
			t.Error("timeout should not be in JSON when not set")
		}
		if _, ok := simpleAgent["maxTurns"]; ok {
			t.Error("maxTurns should not be in JSON when not set")
		}
	})

	t.Run("no agents when not specified", func(t *testing.T) {
		opts := types.NewClaudeAgentOptions()

		transport := NewSubprocessCLITransport(
			"claude",
			"",
			nil,
			log.NewLogger(false),
			"",
			opts,
		)

		args := transport.buildCommandArgs()

		// Verify --agents flag is not present
		for _, arg := range args {
			if arg == "--agents" {
				t.Fatal("--agents flag should not be present when no agents are configured")
			}
		}
	})
}

// TestBuildCommandArgs_SubagentExecution tests subagent execution config JSON serialization
func TestBuildCommandArgs_SubagentExecution(t *testing.T) {
	t.Parallel()
	t.Run("subagent execution with all fields", func(t *testing.T) {
		config := types.NewSubagentExecutionConfig()
		config.MultiInvocation = types.MultiInvocationModeParallel
		config.MaxConcurrent = 5
		config.ErrorHandling = types.SubagentErrorHandlingFailFast

		opts := types.NewClaudeAgentOptions().
			WithSubagentExecution(config)

		transport := NewSubprocessCLITransport(
			"claude",
			"",
			nil,
			log.NewLogger(false),
			"",
			opts,
		)

		args := transport.buildCommandArgs()

		// Verify --subagent-execution flag is present
		subagentIdx := -1
		for i, arg := range args {
			if arg == "--subagent-execution" {
				subagentIdx = i
				break
			}
		}

		if subagentIdx == -1 {
			t.Fatal("--subagent-execution flag not found")
		}

		subagentJSON := args[subagentIdx+1]
		var subagentData map[string]interface{}
		if err := json.Unmarshal([]byte(subagentJSON), &subagentData); err != nil {
			t.Fatalf("Failed to unmarshal subagent JSON: %v", err)
		}

		if subagentData["multi_invocation"] != "parallel" {
			t.Errorf("Expected multi_invocation 'parallel', got %v", subagentData["multi_invocation"])
		}

		if subagentData["max_concurrent"] != float64(5) {
			t.Errorf("Expected max_concurrent 5, got %v", subagentData["max_concurrent"])
		}

		if subagentData["error_handling"] != "fail_fast" {
			t.Errorf("Expected error_handling 'fail_fast', got %v", subagentData["error_handling"])
		}
	})

	t.Run("subagent execution with defaults", func(t *testing.T) {
		config := types.NewSubagentExecutionConfig()

		opts := types.NewClaudeAgentOptions().
			WithSubagentExecution(config)

		transport := NewSubprocessCLITransport(
			"claude",
			"",
			nil,
			log.NewLogger(false),
			"",
			opts,
		)

		args := transport.buildCommandArgs()

		subagentIdx := -1
		for i, arg := range args {
			if arg == "--subagent-execution" {
				subagentIdx = i
				break
			}
		}

		if subagentIdx == -1 {
			t.Fatal("--subagent-execution flag not found")
		}

		subagentJSON := args[subagentIdx+1]
		var subagentData map[string]interface{}
		if err := json.Unmarshal([]byte(subagentJSON), &subagentData); err != nil {
			t.Fatalf("Failed to unmarshal subagent JSON: %v", err)
		}

		// Verify defaults are serialized
		if subagentData["multi_invocation"] != "sequential" {
			t.Error("Default multi_invocation should be sequential")
		}

		if subagentData["max_concurrent"] != float64(3) {
			t.Error("Default max_concurrent should be 3")
		}

		if subagentData["error_handling"] != "continue" {
			t.Error("Default error_handling should be continue")
		}
	})

	t.Run("no subagent execution when not specified", func(t *testing.T) {
		opts := types.NewClaudeAgentOptions()

		transport := NewSubprocessCLITransport(
			"claude",
			"",
			nil,
			log.NewLogger(false),
			"",
			opts,
		)

		args := transport.buildCommandArgs()

		// Verify --subagent-execution flag is not present
		for _, arg := range args {
			if arg == "--subagent-execution" {
				t.Fatal("--subagent-execution flag should not be present when config not set")
			}
		}
	})
}

// TestBuildCommandArgs_AgentsWithSubagentExecution tests agents and subagent config together
func TestBuildCommandArgs_AgentsWithSubagentExecution(t *testing.T) {
	t.Parallel()
	subagentConfig := types.NewSubagentExecutionConfig()
	subagentConfig.MaxConcurrent = 4
	background := true

	opts := types.NewClaudeAgentOptions().
		WithAgent("agent1", types.AgentDefinition{
			Description: "Agent 1",
			Prompt:      "Prompt 1",
			Background:  &background,
		}).
		WithSubagentExecution(subagentConfig)

	transport := NewSubprocessCLITransport(
		"claude",
		"",
		nil,
		log.NewLogger(false),
		"",
		opts,
	)

	args := transport.buildCommandArgs()

	// Verify both flags are present
	hasAgents := false
	hasSubagentExecution := false

	for _, arg := range args {
		if arg == "--agents" {
			hasAgents = true
		}
		if arg == "--subagent-execution" {
			hasSubagentExecution = true
		}
	}

	if !hasAgents {
		t.Error("--agents flag should be present")
	}

	if !hasSubagentExecution {
		t.Error("--subagent-execution flag should be present")
	}
}
