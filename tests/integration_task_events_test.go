//go:build integration
// +build integration

package tests

import (
	"testing"
	"time"

	claude "github.com/hishamkaram/claude-agent-sdk-go"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestRealCLI_TaskLifecycleEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	requireAuth(t)
	requireRunTurns(t)
	cliPath := requireClaude(t)

	ctx, cancel := CreateTestContext(t, 120*time.Second)
	defer cancel()

	opts := types.NewClaudeAgentOptions().
		WithCLIPath(cliPath).
		WithModel("sonnet").
		WithPermissionMode(types.PermissionModeDontAsk).
		WithSystemPromptPreset(types.SystemPromptPreset{Type: "preset", Preset: "claude_code"})

	msgCh, err := claude.Query(ctx, "Use the Task tool with subagent_type Explore to inspect /tmp and report only the directory path it saw. Then reply with one short sentence. Do not edit files.", opts)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	msgs := CollectMessages(ctx, t, msgCh, 120*time.Second)

	var started *types.TaskStartedMessage
	var progress *types.TaskProgressMessage
	var notification *types.TaskNotificationMessage
	for _, msg := range msgs {
		switch m := msg.(type) {
		case *types.TaskStartedMessage:
			if started == nil {
				started = m
			}
		case *types.TaskProgressMessage:
			if started == nil || m.TaskID == started.TaskID {
				progress = m
			}
		case *types.TaskNotificationMessage:
			if started == nil || m.TaskID == started.TaskID {
				notification = m
			}
		}
	}

	if started == nil {
		t.Fatalf("no TaskStartedMessage received; saw %d messages", len(msgs))
	}
	if started.TaskID == "" {
		t.Fatal("TaskStartedMessage.TaskID is empty")
	}
	if started.Description == "" {
		t.Fatal("TaskStartedMessage.Description is empty")
	}
	if started.TaskType == nil || *started.TaskType != "local_agent" {
		t.Fatalf("TaskStartedMessage.TaskType = %v, want local_agent from current Claude Code CLI", started.TaskType)
	}
	if progress == nil {
		t.Fatalf("no TaskProgressMessage received for task %q", started.TaskID)
	}
	if progress.TaskID != started.TaskID {
		t.Fatalf("TaskProgressMessage.TaskID = %q, want %q", progress.TaskID, started.TaskID)
	}
	if notification == nil {
		t.Fatalf("no TaskNotificationMessage received for task %q", started.TaskID)
	}
	if notification.TaskID != started.TaskID {
		t.Fatalf("TaskNotificationMessage.TaskID = %q, want %q", notification.TaskID, started.TaskID)
	}
	if notification.Status == "" {
		t.Fatal("TaskNotificationMessage.Status is empty")
	}
	if notification.Summary == "" {
		t.Fatal("TaskNotificationMessage.Summary is empty")
	}
}
