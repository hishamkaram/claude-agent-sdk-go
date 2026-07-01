package types

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkflowStreamFixture_ClaudeCode2195(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join("testdata", "workflow_stream.ndjson"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var sawWorkflowTool bool
	var sawAssistantWorkflowToolUse bool
	var sawStarted bool
	var sawAsyncLaunch bool
	var sawProgress bool
	var sawTaskUpdated bool
	var sawBudgetResult bool
	var sawNotification bool

	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		msg, err := UnmarshalMessage(line)
		if err != nil {
			t.Fatalf("UnmarshalMessage(%s) error = %v", string(line), err)
		}
		switch m := msg.(type) {
		case *SystemMessage:
			if m.Subtype == SystemSubtypeInit && stringSliceContains(m.Tools, "Workflow") {
				sawWorkflowTool = true
			}
		case *AssistantMessage:
			sawAssistantWorkflowToolUse = sawAssistantWorkflowToolUse || messageHasWorkflowToolUse(m)
		case *TaskStartedMessage:
			sawStarted = true
			if m.TaskID != "wtismrl5c" || m.WorkflowName != "minimal-probe" {
				t.Fatalf("workflow task_started = %+v", m)
			}
			if m.TaskType == nil || *m.TaskType != "local_workflow" {
				t.Fatalf("workflow task_type = %v, want local_workflow", m.TaskType)
			}
			if m.Prompt == "" {
				t.Fatal("workflow prompt was not parsed")
			}
		case *UserMessage:
			if m.ToolUseResult != nil && m.ToolUseResult.Status == "async_launched" {
				sawAsyncLaunch = true
				if m.ToolUseResult.TaskID != "wtismrl5c" ||
					m.ToolUseResult.WorkflowName != "minimal-probe" ||
					m.ToolUseResult.RunID != "wf_2c525e4f-988" {
					t.Fatalf("tool_use_result = %+v", m.ToolUseResult)
				}
			}
		case *TaskProgressMessage:
			if len(m.WorkflowProgress) > 0 {
				sawProgress = true
				if !workflowFixtureProgressHasPhaseAndAgent(m.WorkflowProgress) {
					t.Fatalf("workflow_progress = %+v", m.WorkflowProgress)
				}
			}
		case *TaskUpdatedMessage:
			sawTaskUpdated = true
			if m.Patch.Status != "completed" {
				t.Fatalf("task_updated patch = %+v", m.Patch)
			}
		case *ResultMessage:
			sawBudgetResult = m.Subtype == ResultSubtypeErrorMaxBudget
		case *TaskNotificationMessage:
			sawNotification = true
			if m.Status != "completed" || m.OutputFile == "" || m.Usage == nil {
				t.Fatalf("task_notification = %+v", m)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan fixture: %v", err)
	}

	for name, ok := range map[string]bool{
		"init Workflow tool":           sawWorkflowTool,
		"assistant Workflow tool_use":  sawAssistantWorkflowToolUse,
		"task_started local_workflow":  sawStarted,
		"async tool_use_result":        sawAsyncLaunch,
		"workflow_progress":            sawProgress,
		"task_updated terminal":        sawTaskUpdated,
		"budget result after workflow": sawBudgetResult,
		"task_notification terminal":   sawNotification,
	} {
		if !ok {
			t.Fatalf("fixture missing expected frame: %s", name)
		}
	}
}

func messageHasWorkflowToolUse(m *AssistantMessage) bool {
	for _, block := range m.Content {
		if tool, ok := block.(*ToolUseBlock); ok && tool.Name == "Workflow" {
			return true
		}
	}
	return false
}

func workflowFixtureProgressHasPhaseAndAgent(entries []WorkflowProgressEntry) bool {
	var hasPhase bool
	var hasAgent bool
	for _, entry := range entries {
		hasPhase = hasPhase || entry.Type == "workflow_phase"
		hasAgent = hasAgent || entry.Type == "workflow_agent"
	}
	return hasPhase && hasAgent
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
