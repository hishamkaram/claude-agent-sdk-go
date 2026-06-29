package types

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// Hook input types: JSON roundtrip tests for all 17 Phase C inputs
// ---------------------------------------------------------------------------

func TestPostToolUseFailureHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input PostToolUseFailureHookInput
	}{
		{
			name: "all fields populated",
			input: PostToolUseFailureHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-001",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user/project",
				},
				HookEventName: "PostToolUseFailure",
				ToolName:      "Bash",
				ToolInput:     map[string]interface{}{"command": "rm -rf /"},
				Error:         "permission denied",
			},
		},
		{
			name: "empty tool input",
			input: PostToolUseFailureHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-002",
					TranscriptPath: "/var/log/transcript",
					CWD:            "/",
				},
				HookEventName: "PostToolUseFailure",
				ToolName:      "Write",
				ToolInput:     map[string]interface{}{},
				Error:         "file not found",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded PostToolUseFailureHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.TranscriptPath != tt.input.TranscriptPath {
				t.Errorf("TranscriptPath = %q, want %q", decoded.TranscriptPath, tt.input.TranscriptPath)
			}
			if decoded.CWD != tt.input.CWD {
				t.Errorf("CWD = %q, want %q", decoded.CWD, tt.input.CWD)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
			if decoded.ToolName != tt.input.ToolName {
				t.Errorf("ToolName = %q, want %q", decoded.ToolName, tt.input.ToolName)
			}
			if decoded.Error != tt.input.Error {
				t.Errorf("Error = %q, want %q", decoded.Error, tt.input.Error)
			}
		})
	}
}

func TestNotificationHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input NotificationHookInput
	}{
		{
			name: "info level",
			input: NotificationHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-n01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "Notification",
				Message:       "Task completed successfully",
				Level:         "info",
			},
		},
		{
			name: "error level",
			input: NotificationHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-n02",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "Notification",
				Message:       "Something went wrong",
				Level:         "error",
			},
		},
		{
			name: "empty level",
			input: NotificationHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-n03",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "Notification",
				Message:       "Bare notification",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded NotificationHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
			if decoded.Message != tt.input.Message {
				t.Errorf("Message = %q, want %q", decoded.Message, tt.input.Message)
			}
			if decoded.Level != tt.input.Level {
				t.Errorf("Level = %q, want %q", decoded.Level, tt.input.Level)
			}
		})
	}
}

func TestSessionStartHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input SessionStartHookInput
	}{
		{
			name: "basic session start",
			input: SessionStartHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-ss01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user/project",
				},
				HookEventName: "SessionStart",
			},
		},
		{
			name: "with permission mode",
			input: SessionStartHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-ss02",
					TranscriptPath: "/var/log/t.json",
					CWD:            "/opt/app",
					PermissionMode: stringPtr("plan"),
				},
				HookEventName: "SessionStart",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SessionStartHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
			if decoded.TranscriptPath != tt.input.TranscriptPath {
				t.Errorf("TranscriptPath = %q, want %q", decoded.TranscriptPath, tt.input.TranscriptPath)
			}
			if decoded.CWD != tt.input.CWD {
				t.Errorf("CWD = %q, want %q", decoded.CWD, tt.input.CWD)
			}
			if (decoded.PermissionMode == nil) != (tt.input.PermissionMode == nil) {
				t.Errorf("PermissionMode nil mismatch: got nil=%v, want nil=%v", decoded.PermissionMode == nil, tt.input.PermissionMode == nil)
			}
			if decoded.PermissionMode != nil && *decoded.PermissionMode != *tt.input.PermissionMode {
				t.Errorf("PermissionMode = %q, want %q", *decoded.PermissionMode, *tt.input.PermissionMode)
			}
		})
	}
}

func TestSessionEndHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input SessionEndHookInput
	}{
		{
			name: "with reason",
			input: SessionEndHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-se01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "SessionEnd",
				Reason:        "user_requested",
			},
		},
		{
			name: "empty reason",
			input: SessionEndHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-se02",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "SessionEnd",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SessionEndHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
			if decoded.Reason != tt.input.Reason {
				t.Errorf("Reason = %q, want %q", decoded.Reason, tt.input.Reason)
			}
		})
	}
}

func TestStopFailureHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input StopFailureHookInput
	}{
		{
			name: "with error message",
			input: StopFailureHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-sf01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "StopFailure",
				Error:         "failed to terminate subprocess",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded StopFailureHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
			if decoded.Error != tt.input.Error {
				t.Errorf("Error = %q, want %q", decoded.Error, tt.input.Error)
			}
		})
	}
}

func TestSubagentStartHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input SubagentStartHookInput
	}{
		{
			name: "with agent type",
			input: SubagentStartHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-sa01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "SubagentStart",
				AgentName:     "go-test-writer",
				AgentType:     "opus",
			},
		},
		{
			name: "without agent type",
			input: SubagentStartHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-sa02",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "SubagentStart",
				AgentName:     "debugger",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SubagentStartHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
			if decoded.AgentName != tt.input.AgentName {
				t.Errorf("AgentName = %q, want %q", decoded.AgentName, tt.input.AgentName)
			}
			if decoded.AgentType != tt.input.AgentType {
				t.Errorf("AgentType = %q, want %q", decoded.AgentType, tt.input.AgentType)
			}
		})
	}
}

func TestPostCompactHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input PostCompactHookInput
	}{
		{
			name: "manual trigger",
			input: PostCompactHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-pc01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "PostCompact",
				Trigger:       "manual",
			},
		},
		{
			name: "auto trigger",
			input: PostCompactHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-pc02",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "PostCompact",
				Trigger:       "auto",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded PostCompactHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
			if decoded.Trigger != tt.input.Trigger {
				t.Errorf("Trigger = %q, want %q", decoded.Trigger, tt.input.Trigger)
			}
		})
	}
}
