//go:build integration
// +build integration

package tests

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestCLI_WorkflowLifecycleEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	if os.Getenv("AGENTD_CLAUDE_CLI_INTEGRATION") != "1" {
		t.Skip("AGENTD_CLAUDE_CLI_INTEGRATION=1 not set - skipping Claude Workflow CLI integration test")
	}
	maxBudgetUSD := os.Getenv("CLAUDE_WORKFLOW_TEST_MAX_BUDGET_USD")
	if strings.TrimSpace(maxBudgetUSD) == "" {
		t.Skip("CLAUDE_WORKFLOW_TEST_MAX_BUDGET_USD not set - skipping Claude Workflow CLI integration test")
	}

	requireAuth(t)
	cliPath := requireClaude(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	workspace := t.TempDir()
	artifactPath := filepath.Join(t.TempDir(), "workflow_stream.ndjson")
	stdout, stderr, err := runWorkflowCLIProbe(ctx, cliPath, workspace, maxBudgetUSD)
	if writeErr := os.WriteFile(artifactPath, stdout, 0o600); writeErr != nil {
		t.Fatalf("write CLI artifact: %v", writeErr)
	}
	t.Logf("captured Claude Workflow stream: %s", artifactPath)
	if err != nil && !workflowStreamHasBudgetResult(stdout) {
		t.Fatalf("claude Workflow probe failed: %v\nstderr:\n%s\nstdout artifact: %s", err, string(stderr), artifactPath)
	}

	assertWorkflowCLIStream(t, stdout)
	if err != nil {
		t.Logf("claude exited non-zero after workflow terminal state due budget cap: %v", err)
	}
}

func runWorkflowCLIProbe(ctx context.Context, cliPath, workspace, maxBudgetUSD string) ([]byte, []byte, error) {
	prompt := strings.Join([]string{
		"Run the smallest possible dynamic Workflow that does not edit files.",
		"Use the Workflow tool exactly once.",
		"The workflow should have one phase and one agent.",
		"The agent should return exactly workflow-ok.",
		"Keep the workflow under the configured test budget of $" + maxBudgetUSD + ".",
	}, "\n")

	cmd := exec.CommandContext(ctx, cliPath,
		"-p",
		"--output-format", "stream-json",
		"--verbose",
		"--include-hook-events",
		"--permission-mode", "default",
		"--max-budget-usd", maxBudgetUSD,
		"--tools", "Workflow",
		"--allowedTools", "Workflow",
	)
	cmd.Dir = workspace
	cmd.Stdin = strings.NewReader(prompt)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func assertWorkflowCLIStream(t *testing.T, raw []byte) {
	t.Helper()

	var sawWorkflowTool bool
	var sawAssistantWorkflowToolUse bool
	var sawWorkflowStarted bool
	var sawAsyncLaunch bool
	var sawWorkflowPhaseProgress bool
	var sawWorkflowAgentProgress bool
	var sawTerminal bool
	var workflowTaskID string

	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		msg, err := types.UnmarshalMessage(line)
		if err != nil {
			t.Fatalf("UnmarshalMessage(%s) error = %v", string(line), err)
		}
		switch m := msg.(type) {
		case *types.SystemMessage:
			if m.Subtype == types.SystemSubtypeInit && containsString(m.Tools, "Workflow") {
				sawWorkflowTool = true
			}
		case *types.AssistantMessage:
			if assistantUsesWorkflow(m) {
				sawAssistantWorkflowToolUse = true
			}
		case *types.TaskStartedMessage:
			if m.TaskType != nil && *m.TaskType == "local_workflow" {
				sawWorkflowStarted = true
				workflowTaskID = m.TaskID
				if m.TaskID == "" || m.WorkflowName == "" {
					t.Fatalf("workflow task_started missing IDs/name: %+v", m)
				}
			}
		case *types.UserMessage:
			if m.ToolUseResult != nil && m.ToolUseResult.Status == "async_launched" {
				sawAsyncLaunch = true
				if m.ToolUseResult.TaskType != "local_workflow" {
					t.Fatalf("tool_use_result TaskType = %q, want local_workflow", m.ToolUseResult.TaskType)
				}
				if m.ToolUseResult.TaskID == "" || m.ToolUseResult.WorkflowName == "" || m.ToolUseResult.RunID == "" {
					t.Fatalf("tool_use_result missing workflow IDs: %+v", m.ToolUseResult)
				}
			}
		case *types.TaskProgressMessage:
			if len(m.WorkflowProgress) > 0 {
				for _, entry := range m.WorkflowProgress {
					sawWorkflowPhaseProgress = sawWorkflowPhaseProgress || entry.Type == "workflow_phase"
					sawWorkflowAgentProgress = sawWorkflowAgentProgress || entry.Type == "workflow_agent"
				}
			}
		case *types.TaskUpdatedMessage:
			if workflowTaskID == "" || m.TaskID == workflowTaskID {
				if m.Patch.Status == "completed" || m.Patch.Status == "failed" || m.Patch.Status == "canceled" || m.Patch.Status == "cancelled" {
					sawTerminal = true
				}
			}
		case *types.TaskNotificationMessage:
			if workflowTaskID == "" || m.TaskID == workflowTaskID {
				if m.Status != "" {
					sawTerminal = true
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan workflow stream: %v", err)
	}

	assertSawWorkflowCLIFrames(t, map[string]bool{
		"system.init.tools contains Workflow": sawWorkflowTool,
		"assistant Workflow tool_use":         sawAssistantWorkflowToolUse,
		"local_workflow task_started":         sawWorkflowStarted,
		"async tool_use_result":               sawAsyncLaunch,
		"workflow_progress phase":             sawWorkflowPhaseProgress,
		"workflow_progress agent":             sawWorkflowAgentProgress,
		"terminal task update/notification":   sawTerminal,
	})
}

func assistantUsesWorkflow(m *types.AssistantMessage) bool {
	for _, block := range m.Content {
		if toolUse, ok := block.(*types.ToolUseBlock); ok && toolUse.Name == "Workflow" {
			return true
		}
	}
	return false
}

func assertSawWorkflowCLIFrames(t *testing.T, checks map[string]bool) {
	t.Helper()
	for name, ok := range checks {
		if !ok {
			t.Fatalf("missing Workflow CLI frame: %s", name)
		}
	}
}

func workflowStreamHasBudgetResult(raw []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		msg, err := types.UnmarshalMessage(line)
		if err != nil {
			continue
		}
		result, ok := msg.(*types.ResultMessage)
		if ok && result.Subtype == types.ResultSubtypeErrorMaxBudget {
			return true
		}
	}
	return false
}

func TestWorkflowStreamHasBudgetResult(t *testing.T) {
	raw := []byte(strings.Join([]string{
		`{"type":"system","subtype":"task_updated","task_id":"w1","patch":{"status":"completed"}}`,
		`{"type":"result","subtype":"error_max_budget_usd","is_error":true,"session_id":"s","errors":["Reached maximum budget ($1)"]}`,
	}, "\n"))
	if !workflowStreamHasBudgetResult(raw) {
		t.Fatal("workflowStreamHasBudgetResult returned false for budget result stream")
	}
	if workflowStreamHasBudgetResult([]byte(`{"type":"result","subtype":"success","is_error":false,"session_id":"s"}`)) {
		t.Fatal("workflowStreamHasBudgetResult returned true for success result stream")
	}
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
