package internal

import (
	"fmt"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestParseMessage_NewTopLevelTypes tests routing for 5 new top-level message types.
func TestParseMessage_NewTopLevelTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    []byte
		wantType string
		goType   string
	}{
		{
			name:     "tool_progress",
			input:    toolProgressMessage,
			wantType: "tool_progress",
			goType:   "*types.ToolProgressMessage",
		},
		{
			name:     "tool_progress minimal",
			input:    toolProgressMessageMinimal,
			wantType: "tool_progress",
			goType:   "*types.ToolProgressMessage",
		},
		{
			name:     "auth_status",
			input:    authStatusMessage,
			wantType: "auth_status",
			goType:   "*types.AuthStatusMessage",
		},
		{
			name:     "auth_status no error",
			input:    authStatusMessageNoError,
			wantType: "auth_status",
			goType:   "*types.AuthStatusMessage",
		},
		{
			name:     "tool_use_summary",
			input:    toolUseSummaryMessage,
			wantType: "tool_use_summary",
			goType:   "*types.ToolUseSummaryMessage",
		},
		{
			name:     "rate_limit_event allowed",
			input:    rateLimitEventAllowed,
			wantType: "rate_limit_event",
			goType:   "*types.RateLimitEvent",
		},
		{
			name:     "rate_limit_event warning",
			input:    rateLimitEventWarning,
			wantType: "rate_limit_event",
			goType:   "*types.RateLimitEvent",
		},
		{
			name:     "rate_limit_event rejected",
			input:    rateLimitEventRejected,
			wantType: "rate_limit_event",
			goType:   "*types.RateLimitEvent",
		},
		{
			name:     "prompt_suggestion",
			input:    promptSuggestionMessage,
			wantType: "prompt_suggestion",
			goType:   "*types.PromptSuggestionMessage",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg, err := ParseMessage(tt.input)
			if err != nil {
				t.Fatalf("ParseMessage() unexpected error: %v", err)
			}
			if msg.GetMessageType() != tt.wantType {
				t.Errorf("expected type %s, got %s", tt.wantType, msg.GetMessageType())
			}
			gotType := fmt.Sprintf("%T", msg)
			if gotType != tt.goType {
				t.Errorf("expected Go type %s, got %s", tt.goType, gotType)
			}
		})
	}
}

// TestParseMessage_SystemSubtypes tests routing for typed system subtypes.
func TestParseMessage_SystemSubtypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  []byte
		goType string
	}{
		{
			name:   "compact_boundary",
			input:  systemCompactBoundary,
			goType: "*types.CompactBoundaryMessage",
		},
		{
			name:   "status",
			input:  systemStatus,
			goType: "*types.StatusMessage",
		},
		{
			name:   "status nil status field",
			input:  systemStatusNilStatus,
			goType: "*types.StatusMessage",
		},
		{
			name:   "hook_started",
			input:  systemHookStarted,
			goType: "*types.HookStartedMessage",
		},
		{
			name:   "hook_progress",
			input:  systemHookProgress,
			goType: "*types.HookProgressMessage",
		},
		{
			name:   "hook_response",
			input:  systemHookResponse,
			goType: "*types.HookResponseMessage",
		},
		{
			name:   "task_notification",
			input:  systemTaskNotification,
			goType: "*types.TaskNotificationMessage",
		},
		{
			name:   "task_started",
			input:  systemTaskStarted,
			goType: "*types.TaskStartedMessage",
		},
		{
			name:   "task_progress",
			input:  systemTaskProgress,
			goType: "*types.TaskProgressMessage",
		},
		{
			name:   "task_updated",
			input:  systemTaskUpdated,
			goType: "*types.TaskUpdatedMessage",
		},
		{
			name:   "files_persisted",
			input:  systemFilesPersisted,
			goType: "*types.FilesPersistedEvent",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg, err := ParseMessage(tt.input)
			if err != nil {
				t.Fatalf("ParseMessage() unexpected error: %v", err)
			}
			if msg.GetMessageType() != "system" {
				t.Errorf("expected message type 'system', got %s", msg.GetMessageType())
			}
			gotType := fmt.Sprintf("%T", msg)
			if gotType != tt.goType {
				t.Errorf("expected Go type %s, got %s", tt.goType, gotType)
			}
		})
	}
}

// TestParseMessage_SystemUnknownSubtypeReturnsGeneric verifies unknown system subtypes
// still return *SystemMessage (not error, not typed struct).
func TestParseMessage_SystemUnknownSubtypeReturnsGeneric(t *testing.T) {
	t.Parallel()
	// metadata and warning are existing subtypes that should still return *SystemMessage
	for _, tt := range []struct {
		name  string
		input []byte
	}{
		{"metadata", systemMessageMetadata},
		{"warning", systemMessageWarning},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg, err := ParseMessage(tt.input)
			if err != nil {
				t.Fatalf("ParseMessage() unexpected error: %v", err)
			}
			if _, ok := msg.(*types.SystemMessage); !ok {
				t.Errorf("expected *types.SystemMessage, got %T", msg)
			}
		})
	}
}

// TestParseMessage_EnhancedResultMessage tests enhanced ResultMessage fields via parser.
func TestParseMessage_EnhancedResultMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		input       []byte
		checkResult func(t *testing.T, msg *types.ResultMessage)
	}{
		{
			name:  "max turns error with errors array",
			input: resultMessageMaxTurns,
			checkResult: func(t *testing.T, msg *types.ResultMessage) {
				if msg.Subtype != "error_max_turns" {
					t.Errorf("expected subtype 'error_max_turns', got %s", msg.Subtype)
				}
				if len(msg.Errors) != 2 {
					t.Errorf("expected 2 errors, got %d", len(msg.Errors))
				}
				if !msg.IsError {
					t.Error("expected IsError true")
				}
			},
		},
		{
			name:  "max budget error with cost",
			input: resultMessageMaxBudget,
			checkResult: func(t *testing.T, msg *types.ResultMessage) {
				if msg.Subtype != "error_max_budget_usd" {
					t.Errorf("expected subtype 'error_max_budget_usd', got %s", msg.Subtype)
				}
				if msg.TotalCostUSD == nil || *msg.TotalCostUSD != 1.05 {
					t.Errorf("expected total_cost_usd 1.05, got %v", msg.TotalCostUSD)
				}
			},
		},
		{
			name:  "success with permission denials",
			input: resultMessageWithPermissionDenials,
			checkResult: func(t *testing.T, msg *types.ResultMessage) {
				if len(msg.PermissionDenials) != 1 {
					t.Fatalf("expected 1 permission denial, got %d", len(msg.PermissionDenials))
				}
				if msg.PermissionDenials[0].ToolName != "Bash" {
					t.Errorf("expected tool_name 'Bash', got %s", msg.PermissionDenials[0].ToolName)
				}
			},
		},
		{
			name:  "success with model usage",
			input: resultMessageSuccessWithModelUsage,
			checkResult: func(t *testing.T, msg *types.ResultMessage) {
				if msg.ModelUsageMap == nil {
					t.Fatal("expected modelUsage to be present")
				}
				usage, ok := msg.ModelUsageMap["claude-sonnet-4-5-20250929"]
				if !ok {
					t.Fatal("expected model usage for claude-sonnet-4-5-20250929")
				}
				if usage.InputTokens != 1000 {
					t.Errorf("expected 1000 input tokens, got %d", usage.InputTokens)
				}
				if msg.StopReason == nil || *msg.StopReason != "end_turn" {
					t.Errorf("expected stop_reason 'end_turn', got %v", msg.StopReason)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg, err := ParseMessage(tt.input)
			if err != nil {
				t.Fatalf("ParseMessage() unexpected error: %v", err)
			}
			resultMsg, ok := msg.(*types.ResultMessage)
			if !ok {
				t.Fatalf("expected *types.ResultMessage, got %T", msg)
			}
			tt.checkResult(t, resultMsg)
		})
	}
}

// TestParseMessage_UserMessageReplay tests IsReplay field via parser.
func TestParseMessage_UserMessageReplay(t *testing.T) {
	t.Parallel()
	msg, err := ParseMessage(userMessageReplay)
	if err != nil {
		t.Fatalf("ParseMessage() unexpected error: %v", err)
	}
	userMsg, ok := msg.(*types.UserMessage)
	if !ok {
		t.Fatalf("expected *types.UserMessage, got %T", msg)
	}
	if !userMsg.IsReplay {
		t.Error("expected IsReplay to be true")
	}
}
