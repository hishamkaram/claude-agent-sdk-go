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

	t.Run("WorkflowTaskStartedMessage", func(t *testing.T) {
		t.Parallel()
		jsonData := `{
			"type":"system","subtype":"task_started",
			"task_id":"wtismrl5c","tool_use_id":"toolu_workflow",
			"description":"Smallest dynamic workflow",
			"task_type":"local_workflow",
			"workflow_name":"minimal-probe",
			"prompt":"export const meta = { name: 'minimal-probe' }",
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
		if m.TaskType == nil || *m.TaskType != "local_workflow" {
			t.Fatalf("TaskType = %v, want local_workflow", m.TaskType)
		}
		if m.WorkflowName != "minimal-probe" {
			t.Fatalf("WorkflowName = %q, want minimal-probe", m.WorkflowName)
		}
		if m.Prompt == "" {
			t.Fatal("Prompt was not parsed")
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

	t.Run("WorkflowTaskProgressMessage", func(t *testing.T) {
		t.Parallel()
		jsonData := `{
			"type":"system","subtype":"task_progress",
			"task_id":"wtismrl5c","summary":"1 phase, 1 agent",
			"usage":{"total_tokens":30574,"tool_uses":0,"duration_ms":3283},
			"workflow_progress":[
				{"type":"workflow_phase","index":1,"title":"Probe"},
				{"type":"workflow_agent","index":1,"label":"probe","phaseIndex":1,"phaseTitle":"Probe","model":"claude-opus-4-8","state":"done","tokens":30574,"toolCalls":0,"durationMs":3283,"resultPreview":"workflow-ok","promptPreview":"Return exactly workflow-ok"}
			],
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
		if m.Summary != "1 phase, 1 agent" {
			t.Fatalf("Summary = %q", m.Summary)
		}
		if len(m.WorkflowProgress) != 2 {
			t.Fatalf("WorkflowProgress len = %d, want 2", len(m.WorkflowProgress))
		}
		agent := m.WorkflowProgress[1]
		if agent.Type != "workflow_agent" || agent.Label != "probe" || agent.State != "done" {
			t.Fatalf("agent progress = %+v", agent)
		}
		if agent.Tokens != 30574 || agent.ResultPreview != "workflow-ok" || agent.PromptPreview == "" {
			t.Fatalf("agent progress details = %+v", agent)
		}
	})

	t.Run("TaskUpdatedMessage", func(t *testing.T) {
		t.Parallel()
		jsonData := `{
			"type":"system","subtype":"task_updated",
			"task_id":"wtismrl5c",
			"patch":{"status":"completed","end_time":1782904297561},
			"uuid":"u","session_id":"s"
		}`
		msg, err := UnmarshalMessage([]byte(jsonData))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		m, ok := msg.(*TaskUpdatedMessage)
		if !ok {
			t.Fatalf("expected *TaskUpdatedMessage, got %T", msg)
		}
		if m.Patch.Status != "completed" || m.Patch.EndTime != 1782904297561 {
			t.Fatalf("Patch = %+v", m.Patch)
		}
		if m.ShouldDisplayToUser() {
			t.Fatal("TaskUpdatedMessage should be internal-only")
		}
	})
}

func TestUnmarshalWorkflowToolUseResult(t *testing.T) {
	t.Parallel()
	jsonData := `{
		"type":"user",
		"message":{"role":"user","content":[{"tool_use_id":"toolu_workflow","type":"tool_result","content":"Workflow launched in background.","is_error":false}]},
		"session_id":"s","uuid":"u","timestamp":"2026-07-01T11:11:34.095Z",
		"tool_use_result":{
			"status":"async_launched",
			"taskId":"wtismrl5c",
			"taskType":"local_workflow",
			"workflowName":"minimal-probe",
			"runId":"wf_2c525e4f-988",
			"summary":"Smallest dynamic workflow",
			"transcriptDir":"/tmp/agentd/workflows/wf_2c525e4f-988",
			"scriptPath":"/tmp/agentd/workflows/minimal-probe.js"
		}
	}`
	msg, err := UnmarshalMessage([]byte(jsonData))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	m, ok := msg.(*UserMessage)
	if !ok {
		t.Fatalf("expected *UserMessage, got %T", msg)
	}
	if m.ToolUseResult == nil {
		t.Fatal("ToolUseResult was not parsed")
	}
	if m.ToolUseResult.Status != "async_launched" ||
		m.ToolUseResult.TaskID != "wtismrl5c" ||
		m.ToolUseResult.WorkflowName != "minimal-probe" ||
		m.ToolUseResult.RunID != "wf_2c525e4f-988" {
		t.Fatalf("ToolUseResult = %+v", m.ToolUseResult)
	}
	if m.Timestamp == "" {
		t.Fatal("Timestamp was not parsed")
	}
}

func TestUnmarshalUserMessageIgnoresNonObjectToolUseResult(t *testing.T) {
	t.Parallel()
	jsonData := `{
		"type":"user",
		"message":{"role":"user","content":[{"tool_use_id":"toolu_1","type":"tool_result","content":"InputValidationError: boom","is_error":true}]},
		"session_id":"s","uuid":"u",
		"tool_use_result":"InputValidationError: boom"
	}`
	msg, err := UnmarshalMessage([]byte(jsonData))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	m, ok := msg.(*UserMessage)
	if !ok {
		t.Fatalf("expected *UserMessage, got %T", msg)
	}
	if m.ToolUseResult != nil {
		t.Fatalf("ToolUseResult = %+v, want nil for non-object payload", m.ToolUseResult)
	}
	blocks, ok := m.Content.([]ContentBlock)
	if !ok || len(blocks) != 1 {
		t.Fatalf("Content = %#v, want one content block", m.Content)
	}
}

func TestUnmarshalSystemInitTools(t *testing.T) {
	t.Parallel()
	jsonData := `{
		"type":"system","subtype":"init",
		"tools":["Workflow","Bash","Read"],
		"cwd":"/tmp/workspace",
		"model":"claude-opus-4-8",
		"permissionMode":"default",
		"claude_code_version":"2.1.195",
		"session_id":"s"
	}`
	msg, err := UnmarshalMessage([]byte(jsonData))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	m, ok := msg.(*SystemMessage)
	if !ok {
		t.Fatalf("expected *SystemMessage, got %T", msg)
	}
	if len(m.Tools) != 3 || m.Tools[0] != "Workflow" {
		t.Fatalf("Tools = %#v", m.Tools)
	}
	if m.ClaudeCodeVersion != "2.1.195" || m.PermissionMode != "default" {
		t.Fatalf("init fields = %+v", m)
	}
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
