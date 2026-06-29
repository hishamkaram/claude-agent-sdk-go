package types

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// Hook input types: wire-format key verification for Phase C inputs
// ---------------------------------------------------------------------------

func TestPhaseCHookInputs_WireFormatKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        interface{}
		expectedKeys []string
	}{
		{
			name: "PostToolUseFailureHookInput",
			input: PostToolUseFailureHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "PostToolUseFailure",
				ToolName:      "Bash",
				ToolInput:     map[string]interface{}{},
				Error:         "err",
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name", "tool_name", "tool_input", "error"},
		},
		{
			name: "NotificationHookInput",
			input: NotificationHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "Notification",
				Message:       "msg",
				Level:         "info",
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name", "message", "level"},
		},
		{
			name: "SessionStartHookInput",
			input: SessionStartHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "SessionStart",
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name"},
		},
		{
			name: "SessionEndHookInput",
			input: SessionEndHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "SessionEnd",
				Reason:        "done",
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name", "reason"},
		},
		{
			name: "StopFailureHookInput",
			input: StopFailureHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "StopFailure",
				Error:         "err",
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name", "error"},
		},
		{
			name: "SubagentStartHookInput",
			input: SubagentStartHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "SubagentStart",
				AgentName:     "agent",
				AgentType:     "opus",
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name", "agent_name", "agent_type"},
		},
		{
			name: "PostCompactHookInput",
			input: PostCompactHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "PostCompact",
				Trigger:       "manual",
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name", "trigger"},
		},
		{
			name: "PermissionRequestHookInput",
			input: PermissionRequestHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "PermissionRequest",
				ToolName:      "Bash",
				ToolInput:     map[string]interface{}{},
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name", "tool_name", "tool_input"},
		},
		{
			name: "SetupHookInput",
			input: SetupHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "Setup",
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name"},
		},
		{
			name: "TeammateIdleHookInput",
			input: TeammateIdleHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "TeammateIdle",
				AgentName:     "agent",
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name", "agent_name"},
		},
		{
			name: "TaskCompletedHookInput",
			input: TaskCompletedHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "TaskCompleted",
				TaskID:        "task-1",
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name", "task_id"},
		},
		{
			name: "ElicitationHookInput",
			input: ElicitationHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "Elicitation",
				Questions:     []map[string]interface{}{{"id": "q1"}},
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name", "questions"},
		},
		{
			name: "ElicitationResultHookInput",
			input: ElicitationResultHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "ElicitationResult",
				Answers:       map[string]interface{}{"a": "b"},
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name", "answers"},
		},
		{
			name: "ConfigChangeHookInput",
			input: ConfigChangeHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "ConfigChange",
				ConfigPath:    "/path",
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name", "config_path"},
		},
		{
			name: "WorktreeCreateHookInput",
			input: WorktreeCreateHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "WorktreeCreate",
				WorktreePath:  "/wt",
				BranchName:    "main",
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name", "worktree_path", "branch_name"},
		},
		{
			name: "WorktreeRemoveHookInput",
			input: WorktreeRemoveHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "WorktreeRemove",
				WorktreePath:  "/wt",
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name", "worktree_path"},
		},
		{
			name: "InstructionsLoadedHookInput",
			input: InstructionsLoadedHookInput{
				BaseHookInput: BaseHookInput{SessionID: "s", TranscriptPath: "t", CWD: "c"},
				HookEventName: "InstructionsLoaded",
				Sources:       []string{"CLAUDE.md"},
			},
			expectedKeys: []string{"session_id", "transcript_path", "cwd", "hook_event_name", "sources"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal %s: %v", tt.name, err)
			}

			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal to map: %v", err)
			}

			for _, key := range tt.expectedKeys {
				if _, ok := raw[key]; !ok {
					t.Errorf("expected JSON key %q in %s wire format", key, tt.name)
				}
			}
		})
	}
}
