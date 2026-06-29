package types

import (
	"testing"
)

func TestUnmarshalToolProgressMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		json        string
		wantErr     bool
		checkResult func(t *testing.T, msg Message)
	}{
		{
			name: "valid with all fields",
			json: `{
				"type": "tool_progress",
				"tool_use_id": "toolu_001",
				"tool_name": "Bash",
				"parent_tool_use_id": "parent_001",
				"elapsed_time_seconds": 5.3,
				"task_id": "task_001",
				"uuid": "uuid_001",
				"session_id": "sess_001"
			}`,
			wantErr: false,
			checkResult: func(t *testing.T, msg Message) {
				m, ok := msg.(*ToolProgressMessage)
				if !ok {
					t.Fatalf("expected *ToolProgressMessage, got %T", msg)
				}
				if m.ToolUseID != "toolu_001" {
					t.Errorf("ToolUseID = %q, want %q", m.ToolUseID, "toolu_001")
				}
				if m.ToolName != "Bash" {
					t.Errorf("ToolName = %q, want %q", m.ToolName, "Bash")
				}
				if m.ParentToolUseID == nil || *m.ParentToolUseID != "parent_001" {
					t.Errorf("ParentToolUseID = %v, want %q", m.ParentToolUseID, "parent_001")
				}
				if m.ElapsedTimeSeconds != 5.3 {
					t.Errorf("ElapsedTimeSeconds = %v, want %v", m.ElapsedTimeSeconds, 5.3)
				}
				if m.TaskID == nil || *m.TaskID != "task_001" {
					t.Errorf("TaskID = %v, want %q", m.TaskID, "task_001")
				}
			},
		},
		{
			name: "missing optional task_id",
			json: `{
				"type": "tool_progress",
				"tool_use_id": "toolu_002",
				"tool_name": "Read",
				"elapsed_time_seconds": 1.0,
				"uuid": "uuid_002",
				"session_id": "sess_002"
			}`,
			wantErr: false,
			checkResult: func(t *testing.T, msg Message) {
				m := msg.(*ToolProgressMessage)
				if m.TaskID != nil {
					t.Errorf("TaskID should be nil, got %v", m.TaskID)
				}
				if m.ParentToolUseID != nil {
					t.Errorf("ParentToolUseID should be nil, got %v", m.ParentToolUseID)
				}
			},
		},
		{
			name: "ShouldDisplayToUser returns false",
			json: `{"type":"tool_progress","tool_use_id":"t","tool_name":"x","elapsed_time_seconds":0,"uuid":"u","session_id":"s"}`,
			checkResult: func(t *testing.T, msg Message) {
				if msg.ShouldDisplayToUser() {
					t.Error("ShouldDisplayToUser() should return false")
				}
			},
		},
		{
			name: "GetMessageType returns tool_progress",
			json: `{"type":"tool_progress","tool_use_id":"t","tool_name":"x","elapsed_time_seconds":0,"uuid":"u","session_id":"s"}`,
			checkResult: func(t *testing.T, msg Message) {
				if msg.GetMessageType() != "tool_progress" {
					t.Errorf("GetMessageType() = %q, want %q", msg.GetMessageType(), "tool_progress")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg, err := UnmarshalMessage([]byte(tt.json))
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.checkResult != nil && err == nil {
				tt.checkResult(t, msg)
			}
		})
	}
}

func TestUnmarshalAuthStatusMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		json        string
		wantErr     bool
		checkResult func(t *testing.T, msg Message)
	}{
		{
			name: "valid with error",
			json: `{
				"type": "auth_status",
				"isAuthenticating": true,
				"output": ["Checking...", "Contacting server..."],
				"error": "token expired",
				"uuid": "uuid_001",
				"session_id": "sess_001"
			}`,
			checkResult: func(t *testing.T, msg Message) {
				m := msg.(*AuthStatusMessage)
				if !m.IsAuthenticating {
					t.Error("IsAuthenticating should be true")
				}
				if len(m.Output) != 2 {
					t.Errorf("Output len = %d, want 2", len(m.Output))
				}
				if m.Error == nil || *m.Error != "token expired" {
					t.Errorf("Error = %v, want %q", m.Error, "token expired")
				}
			},
		},
		{
			name: "optional error absent",
			json: `{
				"type": "auth_status",
				"isAuthenticating": false,
				"output": ["Success"],
				"uuid": "uuid_002",
				"session_id": "sess_002"
			}`,
			checkResult: func(t *testing.T, msg Message) {
				m := msg.(*AuthStatusMessage)
				if m.Error != nil {
					t.Errorf("Error should be nil, got %v", m.Error)
				}
				if m.IsAuthenticating {
					t.Error("IsAuthenticating should be false")
				}
			},
		},
		{
			name: "ShouldDisplayToUser returns true",
			json: `{"type":"auth_status","isAuthenticating":false,"output":[],"uuid":"u","session_id":"s"}`,
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
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.checkResult != nil && err == nil {
				tt.checkResult(t, msg)
			}
		})
	}
}

func TestUnmarshalToolUseSummaryMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		json        string
		checkResult func(t *testing.T, msg Message)
	}{
		{
			name: "valid",
			json: `{
				"type": "tool_use_summary",
				"summary": "Used Bash to run tests",
				"preceding_tool_use_ids": ["t1", "t2"],
				"uuid": "u1",
				"session_id": "s1"
			}`,
			checkResult: func(t *testing.T, msg Message) {
				m := msg.(*ToolUseSummaryMessage)
				if m.Summary != "Used Bash to run tests" {
					t.Errorf("Summary = %q", m.Summary)
				}
				if len(m.PrecedingToolUseIDs) != 2 {
					t.Errorf("PrecedingToolUseIDs len = %d, want 2", len(m.PrecedingToolUseIDs))
				}
				if m.PrecedingToolUseIDs[0] != "t1" || m.PrecedingToolUseIDs[1] != "t2" {
					t.Errorf("PrecedingToolUseIDs = %v", m.PrecedingToolUseIDs)
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
				t.Fatalf("UnmarshalMessage() error = %v", err)
			}
			if tt.checkResult != nil {
				tt.checkResult(t, msg)
			}
		})
	}
}

func TestUnmarshalRateLimitEvent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		json        string
		checkResult func(t *testing.T, msg Message)
	}{
		{
			name: "allowed",
			json: `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed"},"uuid":"u","session_id":"s"}`,
			checkResult: func(t *testing.T, msg Message) {
				m := msg.(*RateLimitEvent)
				if m.RateLimitInfo.Status != "allowed" {
					t.Errorf("Status = %q, want %q", m.RateLimitInfo.Status, "allowed")
				}
				if m.RateLimitInfo.ResetsAt != nil {
					t.Error("ResetsAt should be nil")
				}
				if m.RateLimitInfo.Utilization != nil {
					t.Error("Utilization should be nil")
				}
			},
		},
		{
			name: "allowed_warning with utilization",
			json: `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed_warning","utilization":0.85},"uuid":"u","session_id":"s"}`,
			checkResult: func(t *testing.T, msg Message) {
				m := msg.(*RateLimitEvent)
				if m.RateLimitInfo.Status != "allowed_warning" {
					t.Errorf("Status = %q", m.RateLimitInfo.Status)
				}
				if m.RateLimitInfo.Utilization == nil || *m.RateLimitInfo.Utilization != 0.85 {
					t.Errorf("Utilization = %v, want 0.85", m.RateLimitInfo.Utilization)
				}
			},
		},
		{
			name: "rejected with resetsAt",
			json: `{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":1711036800.0,"utilization":1.0},"uuid":"u","session_id":"s"}`,
			checkResult: func(t *testing.T, msg Message) {
				m := msg.(*RateLimitEvent)
				if m.RateLimitInfo.Status != "rejected" {
					t.Errorf("Status = %q", m.RateLimitInfo.Status)
				}
				if m.RateLimitInfo.ResetsAt == nil || *m.RateLimitInfo.ResetsAt != 1711036800.0 {
					t.Errorf("ResetsAt = %v", m.RateLimitInfo.ResetsAt)
				}
			},
		},
		{
			name: "ShouldDisplayToUser returns true",
			json: `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed"},"uuid":"u","session_id":"s"}`,
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
				t.Fatalf("UnmarshalMessage() error = %v", err)
			}
			if tt.checkResult != nil {
				tt.checkResult(t, msg)
			}
		})
	}
}

func TestUnmarshalPromptSuggestionMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		json        string
		checkResult func(t *testing.T, msg Message)
	}{
		{
			name: "valid",
			json: `{"type":"prompt_suggestion","suggestion":"Add tests?","uuid":"u","session_id":"s"}`,
			checkResult: func(t *testing.T, msg Message) {
				m := msg.(*PromptSuggestionMessage)
				if m.Suggestion != "Add tests?" {
					t.Errorf("Suggestion = %q", m.Suggestion)
				}
				if msg.ShouldDisplayToUser() {
					t.Error("ShouldDisplayToUser() should return false")
				}
				if msg.GetMessageType() != "prompt_suggestion" {
					t.Errorf("GetMessageType() = %q", msg.GetMessageType())
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
				t.Fatalf("UnmarshalMessage() error = %v", err)
			}
			if tt.checkResult != nil {
				tt.checkResult(t, msg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests for typed system message subtypes (US2)
// ---------------------------------------------------------------------------
