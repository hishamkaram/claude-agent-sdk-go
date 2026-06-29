package types

import (
	"testing"
)

func TestUnmarshalCompactBoundaryMessage(t *testing.T) {
	t.Parallel()
	jsonData := `{
		"type": "system",
		"subtype": "compact_boundary",
		"compact_metadata": {"trigger": "auto", "pre_tokens": 50000},
		"uuid": "u1",
		"session_id": "s1"
	}`

	msg, err := UnmarshalMessage([]byte(jsonData))
	if err != nil {
		t.Fatalf("UnmarshalMessage() error = %v", err)
	}

	m, ok := msg.(*CompactBoundaryMessage)
	if !ok {
		t.Fatalf("expected *CompactBoundaryMessage, got %T", msg)
	}
	if m.CompactMetadata.Trigger != "auto" {
		t.Errorf("Trigger = %q, want %q", m.CompactMetadata.Trigger, "auto")
	}
	if m.CompactMetadata.PreTokens != 50000 {
		t.Errorf("PreTokens = %d, want %d", m.CompactMetadata.PreTokens, 50000)
	}
	if m.ShouldDisplayToUser() {
		t.Error("ShouldDisplayToUser() should return false")
	}
}

func TestUnmarshalStatusMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		json        string
		checkResult func(t *testing.T, msg Message)
	}{
		{
			name: "compacting with permission mode",
			json: `{
				"type": "system", "subtype": "status",
				"status": "compacting", "permissionMode": "default",
				"uuid": "u", "session_id": "s"
			}`,
			checkResult: func(t *testing.T, msg Message) {
				m := msg.(*StatusMessage)
				if m.Status == nil || *m.Status != "compacting" {
					t.Errorf("Status = %v, want %q", m.Status, "compacting")
				}
				if m.PermissionMode == nil || *m.PermissionMode != "default" {
					t.Errorf("PermissionMode = %v, want %q", m.PermissionMode, "default")
				}
			},
		},
		{
			name: "nil status",
			json: `{"type": "system", "subtype": "status", "uuid": "u", "session_id": "s"}`,
			checkResult: func(t *testing.T, msg Message) {
				m := msg.(*StatusMessage)
				if m.Status != nil {
					t.Errorf("Status should be nil, got %v", m.Status)
				}
			},
		},
		{
			name: "ShouldDisplayToUser returns true",
			json: `{"type":"system","subtype":"status","uuid":"u","session_id":"s"}`,
			checkResult: func(t *testing.T, msg Message) {
				if !msg.ShouldDisplayToUser() {
					t.Error("ShouldDisplayToUser() should return true")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg, err := UnmarshalMessage([]byte(tt.json))
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if tt.checkResult != nil {
				tt.checkResult(t, msg)
			}
		})
	}
}

func TestUnmarshalHookMessages(t *testing.T) {
	t.Parallel()

	t.Run("HookStartedMessage", func(t *testing.T) {
		t.Parallel()
		jsonData := `{
			"type":"system","subtype":"hook_started",
			"hook_id":"h1","hook_name":"pre-commit","hook_event":"PostToolUse",
			"uuid":"u","session_id":"s"
		}`
		msg, err := UnmarshalMessage([]byte(jsonData))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		m, ok := msg.(*HookStartedMessage)
		if !ok {
			t.Fatalf("expected *HookStartedMessage, got %T", msg)
		}
		if m.HookID != "h1" || m.HookName != "pre-commit" || m.HookEvent != "PostToolUse" {
			t.Errorf("fields = %+v", m)
		}
		if m.ShouldDisplayToUser() {
			t.Error("ShouldDisplayToUser() should return false")
		}
	})

	t.Run("HookProgressMessage", func(t *testing.T) {
		t.Parallel()
		jsonData := `{
			"type":"system","subtype":"hook_progress",
			"hook_id":"h2","hook_name":"lint","hook_event":"PostToolUse",
			"stdout":"Running...\n","stderr":"","output":"Running...\n",
			"uuid":"u","session_id":"s"
		}`
		msg, err := UnmarshalMessage([]byte(jsonData))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		m, ok := msg.(*HookProgressMessage)
		if !ok {
			t.Fatalf("expected *HookProgressMessage, got %T", msg)
		}
		if m.Stdout != "Running...\n" {
			t.Errorf("Stdout = %q", m.Stdout)
		}
		if m.ShouldDisplayToUser() {
			t.Error("ShouldDisplayToUser() should return false")
		}
	})

	t.Run("HookResponseMessage", func(t *testing.T) {
		t.Parallel()
		jsonData := `{
			"type":"system","subtype":"hook_response",
			"hook_id":"h3","hook_name":"pre-commit","hook_event":"PostToolUse",
			"output":"Done","stdout":"All passed","stderr":"",
			"exit_code":0,"outcome":"success",
			"uuid":"u","session_id":"s"
		}`
		msg, err := UnmarshalMessage([]byte(jsonData))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		m, ok := msg.(*HookResponseMessage)
		if !ok {
			t.Fatalf("expected *HookResponseMessage, got %T", msg)
		}
		if m.Outcome != "success" {
			t.Errorf("Outcome = %q", m.Outcome)
		}
		if m.ExitCode == nil || *m.ExitCode != 0 {
			t.Errorf("ExitCode = %v", m.ExitCode)
		}
		if m.ShouldDisplayToUser() {
			t.Error("ShouldDisplayToUser() should return false")
		}
	})
}

func TestUnmarshalTaskMessages(t *testing.T) {
	t.Parallel()

	t.Run("TaskNotificationMessage", func(t *testing.T) {
		t.Parallel()
		jsonData := `{
			"type":"system","subtype":"task_notification",
			"task_id":"t1","status":"completed",
			"output_file":"/tmp/out.txt","summary":"Done",
			"usage":{"total_tokens":5000,"tool_uses":12,"duration_ms":30000},
			"uuid":"u","session_id":"s"
		}`
		msg, err := UnmarshalMessage([]byte(jsonData))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		m, ok := msg.(*TaskNotificationMessage)
		if !ok {
			t.Fatalf("expected *TaskNotificationMessage, got %T", msg)
		}
		if m.TaskID != "t1" || m.Status != "completed" || m.Summary != "Done" {
			t.Errorf("fields = %+v", m)
		}
		if m.Usage == nil || m.Usage.TotalTokens != 5000 {
			t.Errorf("Usage = %v", m.Usage)
		}
		if !m.ShouldDisplayToUser() {
			t.Error("ShouldDisplayToUser() should return true")
		}
	})

	t.Run("TaskStartedMessage", func(t *testing.T) {
		t.Parallel()
		jsonData := `{
			"type":"system","subtype":"task_started",
			"task_id":"t2","description":"Building feature",
			"task_type":"agent",
			"uuid":"u","session_id":"s"
		}`
		msg, err := UnmarshalMessage([]byte(jsonData))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		m, ok := msg.(*TaskStartedMessage)
		if !ok {
			t.Fatalf("expected *TaskStartedMessage, got %T", msg)
		}
		if m.Description != "Building feature" {
			t.Errorf("Description = %q", m.Description)
		}
		if m.TaskType == nil || *m.TaskType != "agent" {
			t.Errorf("TaskType = %v", m.TaskType)
		}
		if !m.ShouldDisplayToUser() {
			t.Error("ShouldDisplayToUser() should return true")
		}
	})

	t.Run("TaskProgressMessage", func(t *testing.T) {
		t.Parallel()
		jsonData := `{
			"type":"system","subtype":"task_progress",
			"task_id":"t3","description":"Running tests",
			"usage":{"total_tokens":2500,"tool_uses":5,"duration_ms":15000},
			"last_tool_name":"Bash",
			"uuid":"u","session_id":"s"
		}`
		msg, err := UnmarshalMessage([]byte(jsonData))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		m, ok := msg.(*TaskProgressMessage)
		if !ok {
			t.Fatalf("expected *TaskProgressMessage, got %T", msg)
		}
		if m.Usage.ToolUses != 5 {
			t.Errorf("Usage.ToolUses = %d", m.Usage.ToolUses)
		}
		if m.LastToolName == nil || *m.LastToolName != "Bash" {
			t.Errorf("LastToolName = %v", m.LastToolName)
		}
		if m.ShouldDisplayToUser() {
			t.Error("ShouldDisplayToUser() should return false")
		}
	})
}

func TestUnmarshalFilesPersistedEvent(t *testing.T) {
	t.Parallel()
	jsonData := `{
		"type":"system","subtype":"files_persisted",
		"files":[
			{"filename":"main.go","file_id":"f1"},
			{"filename":"test.go","file_id":"f2"}
		],
		"failed":[{"filename":"big.bin","error":"too large"}],
		"processed_at":"2026-03-19T10:30:00Z",
		"uuid":"u","session_id":"s"
	}`

	msg, err := UnmarshalMessage([]byte(jsonData))
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	m, ok := msg.(*FilesPersistedEvent)
	if !ok {
		t.Fatalf("expected *FilesPersistedEvent, got %T", msg)
	}
	if len(m.Files) != 2 {
		t.Errorf("Files len = %d, want 2", len(m.Files))
	}
	if m.Files[0].Filename != "main.go" || m.Files[0].FileID != "f1" {
		t.Errorf("Files[0] = %+v", m.Files[0])
	}
	if len(m.Failed) != 1 || m.Failed[0].Error != "too large" {
		t.Errorf("Failed = %+v", m.Failed)
	}
	if m.ShouldDisplayToUser() {
		t.Error("ShouldDisplayToUser() should return false")
	}
}

func TestUnmarshalSystemUnknownSubtype(t *testing.T) {
	t.Parallel()
	jsonData := `{
		"type": "system",
		"subtype": "future_system_event",
		"data": {"key": "value"}
	}`

	msg, err := UnmarshalMessage([]byte(jsonData))
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	m, ok := msg.(*SystemMessage)
	if !ok {
		t.Fatalf("expected *SystemMessage for unknown subtype, got %T", msg)
	}
	if m.Subtype != "future_system_event" {
		t.Errorf("Subtype = %q", m.Subtype)
	}
}

// ---------------------------------------------------------------------------
// Tests for enhanced ResultMessage & UserMessage (US3)
// ---------------------------------------------------------------------------

func TestExistingSystemSubtypesReturnSystemMessage(t *testing.T) {
	t.Parallel()
	subtypes := []string{"init", "warning", "error", "info", "debug", "session_end", "session_info"}
	for _, sub := range subtypes {
		sub := sub
		t.Run(sub, func(t *testing.T) {
			t.Parallel()
			jsonData := `{"type":"system","subtype":"` + sub + `","data":{}}`
			msg, err := UnmarshalMessage([]byte(jsonData))
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if _, ok := msg.(*SystemMessage); !ok {
				t.Errorf("subtype %q should return *SystemMessage, got %T", sub, msg)
			}
		})
	}
}
