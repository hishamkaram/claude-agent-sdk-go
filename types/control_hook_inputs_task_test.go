package types

import (
	"encoding/json"
	"testing"
)

func TestPermissionRequestHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input PermissionRequestHookInput
	}{
		{
			name: "with tool input",
			input: PermissionRequestHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-pr01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "PermissionRequest",
				ToolName:      "Bash",
				ToolInput:     map[string]interface{}{"command": "sudo apt update"},
			},
		},
		{
			name: "empty tool input",
			input: PermissionRequestHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-pr02",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "PermissionRequest",
				ToolName:      "Read",
				ToolInput:     map[string]interface{}{},
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

			var decoded PermissionRequestHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
			if decoded.ToolName != tt.input.ToolName {
				t.Errorf("ToolName = %q, want %q", decoded.ToolName, tt.input.ToolName)
			}
		})
	}
}

func TestSetupHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input SetupHookInput
	}{
		{
			name: "basic setup",
			input: SetupHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-su01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "Setup",
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

			var decoded SetupHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
		})
	}
}

func TestTeammateIdleHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input TeammateIdleHookInput
	}{
		{
			name: "idle teammate",
			input: TeammateIdleHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-ti01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "TeammateIdle",
				AgentName:     "go-implementer",
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

			var decoded TeammateIdleHookInput
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
		})
	}
}

func TestTaskCompletedHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input TaskCompletedHookInput
	}{
		{
			name: "completed task",
			input: TaskCompletedHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-tc01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "TaskCompleted",
				TaskID:        "task-abc-123",
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

			var decoded TaskCompletedHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
			if decoded.TaskID != tt.input.TaskID {
				t.Errorf("TaskID = %q, want %q", decoded.TaskID, tt.input.TaskID)
			}
		})
	}
}

func TestElicitationHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input ElicitationHookInput
	}{
		{
			name: "single question",
			input: ElicitationHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-el01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "Elicitation",
				Questions: []map[string]interface{}{
					{"id": "q1", "text": "What framework?", "type": "text"},
				},
			},
		},
		{
			name: "multiple questions",
			input: ElicitationHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-el02",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "Elicitation",
				Questions: []map[string]interface{}{
					{"id": "q1", "text": "Language?", "type": "select"},
					{"id": "q2", "text": "Version?", "type": "text"},
				},
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

			var decoded ElicitationHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
			if len(decoded.Questions) != len(tt.input.Questions) {
				t.Errorf("Questions length = %d, want %d", len(decoded.Questions), len(tt.input.Questions))
			}
		})
	}
}

func TestElicitationResultHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input ElicitationResultHookInput
	}{
		{
			name: "with answers",
			input: ElicitationResultHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-er01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "ElicitationResult",
				Answers:       map[string]interface{}{"q1": "Go", "q2": "1.24"},
			},
		},
		{
			name: "empty answers",
			input: ElicitationResultHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-er02",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "ElicitationResult",
				Answers:       map[string]interface{}{},
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

			var decoded ElicitationResultHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
			if len(decoded.Answers) != len(tt.input.Answers) {
				t.Errorf("Answers length = %d, want %d", len(decoded.Answers), len(tt.input.Answers))
			}
		})
	}
}

func TestConfigChangeHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input ConfigChangeHookInput
	}{
		{
			name: "config path set",
			input: ConfigChangeHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-cc01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "ConfigChange",
				ConfigPath:    "/home/user/.config/agentd/agentd.yaml",
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

			var decoded ConfigChangeHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
			if decoded.ConfigPath != tt.input.ConfigPath {
				t.Errorf("ConfigPath = %q, want %q", decoded.ConfigPath, tt.input.ConfigPath)
			}
		})
	}
}

func TestWorktreeCreateHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input WorktreeCreateHookInput
	}{
		{
			name: "with branch name",
			input: WorktreeCreateHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-wc01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "WorktreeCreate",
				WorktreePath:  "/home/user/project-wt",
				BranchName:    "feature/new-hooks",
			},
		},
		{
			name: "without branch name",
			input: WorktreeCreateHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-wc02",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "WorktreeCreate",
				WorktreePath:  "/home/user/project-wt",
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

			var decoded WorktreeCreateHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
			if decoded.WorktreePath != tt.input.WorktreePath {
				t.Errorf("WorktreePath = %q, want %q", decoded.WorktreePath, tt.input.WorktreePath)
			}
			if decoded.BranchName != tt.input.BranchName {
				t.Errorf("BranchName = %q, want %q", decoded.BranchName, tt.input.BranchName)
			}
		})
	}
}

func TestWorktreeRemoveHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input WorktreeRemoveHookInput
	}{
		{
			name: "remove worktree",
			input: WorktreeRemoveHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-wr01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "WorktreeRemove",
				WorktreePath:  "/home/user/project-wt",
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

			var decoded WorktreeRemoveHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
			if decoded.WorktreePath != tt.input.WorktreePath {
				t.Errorf("WorktreePath = %q, want %q", decoded.WorktreePath, tt.input.WorktreePath)
			}
		})
	}
}

func TestInstructionsLoadedHookInput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input InstructionsLoadedHookInput
	}{
		{
			name: "with sources",
			input: InstructionsLoadedHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-il01",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "InstructionsLoaded",
				Sources:       []string{"CLAUDE.md", ".claude/rules/go-tests.md"},
			},
		},
		{
			name: "empty sources",
			input: InstructionsLoadedHookInput{
				BaseHookInput: BaseHookInput{
					SessionID:      "sess-il02",
					TranscriptPath: "/tmp/transcript.json",
					CWD:            "/home/user",
				},
				HookEventName: "InstructionsLoaded",
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

			var decoded InstructionsLoadedHookInput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.input.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.input.SessionID)
			}
			if decoded.HookEventName != tt.input.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.input.HookEventName)
			}
			if len(decoded.Sources) != len(tt.input.Sources) {
				t.Errorf("Sources length = %d, want %d", len(decoded.Sources), len(tt.input.Sources))
			}
			for i := range decoded.Sources {
				if decoded.Sources[i] != tt.input.Sources[i] {
					t.Errorf("Sources[%d] = %q, want %q", i, decoded.Sources[i], tt.input.Sources[i])
				}
			}
		})
	}
}
