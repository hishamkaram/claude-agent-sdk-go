package types

import (
	"encoding/json"
	"testing"
)

// TestTextBlockMarshaling tests JSON marshaling/unmarshaling of TextBlock.
func TestTextBlockMarshaling(t *testing.T) {
	t.Parallel()
	original := &TextBlock{
		Type: "text",
		Text: "Hello, world!",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal TextBlock: %v", err)
	}

	var decoded TextBlock
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal TextBlock: %v", err)
	}

	if decoded.Type != original.Type || decoded.Text != original.Text {
		t.Errorf("unmarshaled TextBlock doesn't match original")
	}
}

// TestToolUseBlockMarshaling tests JSON marshaling/unmarshaling of ToolUseBlock.
func TestToolUseBlockMarshaling(t *testing.T) {
	t.Parallel()
	original := &ToolUseBlock{
		Type: "tool_use",
		ID:   "test-123",
		Name: "Bash",
		Input: map[string]interface{}{
			"command": "ls -la",
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal ToolUseBlock: %v", err)
	}

	var decoded ToolUseBlock
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ToolUseBlock: %v", err)
	}

	if decoded.Type != original.Type || decoded.ID != original.ID || decoded.Name != original.Name {
		t.Errorf("unmarshaled ToolUseBlock doesn't match original")
	}
}

// TestUnmarshalContentBlock tests unmarshaling of different content block types.
func TestUnmarshalContentBlock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		json     string
		wantType string
	}{
		{
			name:     "text block",
			json:     `{"type":"text","text":"Hello"}`,
			wantType: "text",
		},
		{
			name:     "tool_use block",
			json:     `{"type":"tool_use","id":"123","name":"Bash","input":{}}`,
			wantType: "tool_use",
		},
		{
			name:     "tool_result block",
			json:     `{"type":"tool_result","tool_use_id":"123"}`,
			wantType: "tool_result",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			block, err := UnmarshalContentBlock([]byte(tt.json))
			if err != nil {
				t.Fatalf("UnmarshalContentBlock failed: %v", err)
			}
			if block.GetType() != tt.wantType {
				t.Errorf("expected type %s, got %s", tt.wantType, block.GetType())
			}
		})
	}
}

// TestUserMessageMarshaling tests JSON marshaling/unmarshaling of UserMessage.
func TestUserMessageMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("string content", func(t *testing.T) {
		t.Parallel()
		original := &UserMessage{
			Type:    "user",
			Content: "Hello, Claude!",
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("failed to marshal UserMessage: %v", err)
		}

		var decoded UserMessage
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal UserMessage: %v", err)
		}

		if str, ok := decoded.Content.(string); !ok || str != "Hello, Claude!" {
			t.Errorf("content doesn't match: got %v", decoded.Content)
		}
	})

	t.Run("nested message format with tool_result", func(t *testing.T) {
		t.Parallel()
		jsonData := `{
			"type": "user",
			"message": {
				"role": "user",
				"content": [
					{
						"type": "tool_result",
						"tool_use_id": "toolu_018VGbrw1cvCFai5w3ofrJC6",
						"content": "Command output here"
					}
				]
			}
		}`

		var decoded UserMessage
		if err := json.Unmarshal([]byte(jsonData), &decoded); err != nil {
			t.Fatalf("failed to unmarshal nested UserMessage: %v", err)
		}

		if decoded.Type != "user" {
			t.Errorf("expected type 'user', got %s", decoded.Type)
		}

		blocks, ok := decoded.Content.([]ContentBlock)
		if !ok {
			t.Fatalf("expected content to be []ContentBlock, got %T", decoded.Content)
		}

		if len(blocks) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(blocks))
		}

		toolResult, ok := blocks[0].(*ToolResultBlock)
		if !ok {
			t.Fatalf("expected ToolResultBlock, got %T", blocks[0])
		}

		if toolResult.ToolUseID != "toolu_018VGbrw1cvCFai5w3ofrJC6" {
			t.Errorf("expected tool_use_id 'toolu_018VGbrw1cvCFai5w3ofrJC6', got %s", toolResult.ToolUseID)
		}
	})

	t.Run("top-level content array", func(t *testing.T) {
		t.Parallel()
		jsonData := `{
			"type": "user",
			"content": [
				{
					"type": "text",
					"text": "Hello"
				}
			]
		}`

		var decoded UserMessage
		if err := json.Unmarshal([]byte(jsonData), &decoded); err != nil {
			t.Fatalf("failed to unmarshal UserMessage with content array: %v", err)
		}

		blocks, ok := decoded.Content.([]ContentBlock)
		if !ok {
			t.Fatalf("expected content to be []ContentBlock, got %T", decoded.Content)
		}

		if len(blocks) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(blocks))
		}
	})
}

// TestUserMessageUUID tests JSON marshaling/unmarshaling of the UserMessage UUID field.
func TestUserMessageUUID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		json     string
		wantUUID string
	}{
		{
			name:     "uuid present",
			json:     `{"type":"user","content":"hello","uuid":"msg-uuid-123"}`,
			wantUUID: "msg-uuid-123",
		},
		{
			name:     "uuid absent",
			json:     `{"type":"user","content":"hello"}`,
			wantUUID: "",
		},
		{
			name:     "uuid empty string",
			json:     `{"type":"user","content":"hello","uuid":""}`,
			wantUUID: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var msg UserMessage
			if err := json.Unmarshal([]byte(tt.json), &msg); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if msg.UUID != tt.wantUUID {
				t.Errorf("UUID = %q, want %q", msg.UUID, tt.wantUUID)
			}
		})
	}

	// Round-trip: marshal then unmarshal preserves UUID.
	t.Run("round-trip preserves UUID", func(t *testing.T) {
		t.Parallel()

		original := &UserMessage{
			Type:    "user",
			Content: "test message",
			UUID:    "roundtrip-uuid-456",
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		var decoded UserMessage
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if decoded.UUID != original.UUID {
			t.Errorf("UUID after round-trip = %q, want %q", decoded.UUID, original.UUID)
		}
	})

	// Verify omitempty: empty UUID is not included in marshaled JSON.
	t.Run("omitempty excludes empty UUID from JSON", func(t *testing.T) {
		t.Parallel()

		msg := &UserMessage{
			Type:    "user",
			Content: "no uuid",
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("Unmarshal to map failed: %v", err)
		}

		if _, exists := raw["uuid"]; exists {
			t.Error("expected 'uuid' to be omitted from JSON when empty")
		}
	})
}

// TestResultMessageMarshaling tests JSON marshaling/unmarshaling of ResultMessage.
func TestResultMessageMarshaling(t *testing.T) {
	t.Parallel()
	costUSD := 0.05
	result := "Success"
	original := &ResultMessage{
		Type:          "result",
		Subtype:       "query_complete",
		DurationMs:    5000,
		DurationAPIMs: 4500,
		IsError:       false,
		NumTurns:      3,
		SessionID:     "session-123",
		TotalCostUSD:  &costUSD,
		Result:        &result,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal ResultMessage: %v", err)
	}

	var decoded ResultMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ResultMessage: %v", err)
	}

	if decoded.SessionID != original.SessionID {
		t.Errorf("session ID doesn't match")
	}
	if decoded.TotalCostUSD == nil || *decoded.TotalCostUSD != *original.TotalCostUSD {
		t.Errorf("total cost doesn't match")
	}
}

// TestModelInfo_JSONRoundtrip verifies ModelInfo marshals and unmarshals correctly.
func TestModelInfo_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	original := ModelInfo{
		Value:       "claude-3-5-haiku-latest",
		DisplayName: "Claude 3.5 Haiku",
		Description: "Fast and affordable model",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal ModelInfo: %v", err)
	}

	var decoded ModelInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ModelInfo: %v", err)
	}

	if decoded.Value != original.Value {
		t.Errorf("Value mismatch: got %q, want %q", decoded.Value, original.Value)
	}
	if decoded.DisplayName != original.DisplayName {
		t.Errorf("DisplayName mismatch: got %q, want %q", decoded.DisplayName, original.DisplayName)
	}
	if decoded.Description != original.Description {
		t.Errorf("Description mismatch: got %q, want %q", decoded.Description, original.Description)
	}
}

// TestModelInfo_OptionalDescription verifies that omitting Description works correctly.
func TestModelInfo_OptionalDescription(t *testing.T) {
	t.Parallel()
	original := ModelInfo{
		Value:       "claude-3-opus-latest",
		DisplayName: "Claude 3 Opus",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal ModelInfo: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}
	if _, exists := raw["description"]; exists {
		t.Error("expected 'description' to be omitted from JSON when empty")
	}

	var decoded ModelInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ModelInfo: %v", err)
	}
	if decoded.Description != "" {
		t.Errorf("expected empty description, got %q", decoded.Description)
	}
}

// ---------------------------------------------------------------------------
// Tests for new top-level message types (US1)
// ---------------------------------------------------------------------------

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

func TestResultMessageErrorSubtypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		json       string
		wantSub    string
		wantErrors []string
		wantIsErr  bool
	}{
		{
			name: "error_max_turns with errors",
			json: `{
				"type":"result","subtype":"error_max_turns",
				"duration_ms":120000,"duration_api_ms":115000,
				"is_error":true,"num_turns":25,"session_id":"s",
				"errors":["Max turns reached","Task incomplete"]
			}`,
			wantSub:    "error_max_turns",
			wantErrors: []string{"Max turns reached", "Task incomplete"},
			wantIsErr:  true,
		},
		{
			name: "error_max_budget_usd",
			json: `{
				"type":"result","subtype":"error_max_budget_usd",
				"duration_ms":60000,"duration_api_ms":55000,
				"is_error":true,"num_turns":10,"session_id":"s",
				"errors":["Budget exceeded"]
			}`,
			wantSub:    "error_max_budget_usd",
			wantErrors: []string{"Budget exceeded"},
			wantIsErr:  true,
		},
		{
			name: "error_during_execution",
			json: `{
				"type":"result","subtype":"error_during_execution",
				"duration_ms":5000,"duration_api_ms":4500,
				"is_error":true,"num_turns":1,"session_id":"s",
				"errors":["Process crashed"]
			}`,
			wantSub:    "error_during_execution",
			wantErrors: []string{"Process crashed"},
			wantIsErr:  true,
		},
		{
			name: "error_max_structured_output_retries",
			json: `{
				"type":"result","subtype":"error_max_structured_output_retries",
				"duration_ms":10000,"duration_api_ms":9000,
				"is_error":true,"num_turns":3,"session_id":"s",
				"errors":["Schema validation failed"]
			}`,
			wantSub:    "error_max_structured_output_retries",
			wantErrors: []string{"Schema validation failed"},
			wantIsErr:  true,
		},
		{
			name: "success with nil errors",
			json: `{
				"type":"result","subtype":"success",
				"duration_ms":5000,"duration_api_ms":4500,
				"is_error":false,"num_turns":3,"session_id":"s",
				"result":"All done"
			}`,
			wantSub:   "success",
			wantIsErr: false,
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
			m := msg.(*ResultMessage)
			if m.Subtype != tt.wantSub {
				t.Errorf("Subtype = %q, want %q", m.Subtype, tt.wantSub)
			}
			if m.IsError != tt.wantIsErr {
				t.Errorf("IsError = %v, want %v", m.IsError, tt.wantIsErr)
			}
			if tt.wantErrors != nil {
				if len(m.Errors) != len(tt.wantErrors) {
					t.Fatalf("Errors len = %d, want %d", len(m.Errors), len(tt.wantErrors))
				}
				for i, e := range tt.wantErrors {
					if m.Errors[i] != e {
						t.Errorf("Errors[%d] = %q, want %q", i, m.Errors[i], e)
					}
				}
			} else if m.Errors != nil {
				t.Errorf("Errors should be nil, got %v", m.Errors)
			}
		})
	}
}

func TestResultMessageNewFields(t *testing.T) {
	t.Parallel()
	jsonData := `{
		"type":"result","subtype":"success",
		"duration_ms":8000,"duration_api_ms":7500,
		"is_error":false,"num_turns":5,"session_id":"s",
		"stop_reason":"end_turn",
		"permission_denials":[{
			"tool_name":"Bash",
			"tool_use_id":"toolu_pd_001",
			"tool_input":{"command":"rm -rf /"}
		}],
		"modelUsage":{
			"claude-sonnet-4-5-20250929":{
				"input_tokens":1000,
				"output_tokens":500,
				"cache_creation_input_tokens":200,
				"cache_read_input_tokens":100
			}
		},
		"uuid":"uuid_test"
	}`

	msg, err := UnmarshalMessage([]byte(jsonData))
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	m := msg.(*ResultMessage)

	if m.StopReason == nil || *m.StopReason != "end_turn" {
		t.Errorf("StopReason = %v, want %q", m.StopReason, "end_turn")
	}
	if len(m.PermissionDenials) != 1 {
		t.Fatalf("PermissionDenials len = %d, want 1", len(m.PermissionDenials))
	}
	if m.PermissionDenials[0].ToolName != "Bash" {
		t.Errorf("PermissionDenials[0].ToolName = %q", m.PermissionDenials[0].ToolName)
	}
	if m.PermissionDenials[0].ToolUseID != "toolu_pd_001" {
		t.Errorf("PermissionDenials[0].ToolUseID = %q", m.PermissionDenials[0].ToolUseID)
	}

	usage, ok := m.ModelUsageMap["claude-sonnet-4-5-20250929"]
	if !ok {
		t.Fatal("ModelUsageMap missing key claude-sonnet-4-5-20250929")
	}
	if usage.InputTokens != 1000 || usage.OutputTokens != 500 {
		t.Errorf("ModelUsage = %+v", usage)
	}
	if usage.CacheCreationInputTokens != 200 || usage.CacheReadInputTokens != 100 {
		t.Errorf("cache tokens = %d/%d", usage.CacheCreationInputTokens, usage.CacheReadInputTokens)
	}

	if m.UUID != "uuid_test" {
		t.Errorf("UUID = %q", m.UUID)
	}
}

func TestUserMessageIsReplay(t *testing.T) {
	t.Parallel()

	t.Run("isReplay true", func(t *testing.T) {
		t.Parallel()
		jsonData := `{"type":"user","content":"Hello","isReplay":true}`
		msg, err := UnmarshalMessage([]byte(jsonData))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		m := msg.(*UserMessage)
		if !m.IsReplay {
			t.Error("IsReplay should be true")
		}
	})

	t.Run("isReplay absent defaults to false", func(t *testing.T) {
		t.Parallel()
		jsonData := `{"type":"user","content":"Hello"}`
		msg, err := UnmarshalMessage([]byte(jsonData))
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		m := msg.(*UserMessage)
		if m.IsReplay {
			t.Error("IsReplay should default to false")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests for forward compatibility (US4)
// ---------------------------------------------------------------------------

func TestUnmarshalUnknownMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		json        string
		checkResult func(t *testing.T, msg Message)
	}{
		{
			name: "unknown type returns UnknownMessage",
			json: `{"type":"future_feature_xyz","data":{"key":"value"}}`,
			checkResult: func(t *testing.T, msg Message) {
				m, ok := msg.(*UnknownMessage)
				if !ok {
					t.Fatalf("expected *UnknownMessage, got %T", msg)
				}
				if m.Type != "future_feature_xyz" {
					t.Errorf("Type = %q, want %q", m.Type, "future_feature_xyz")
				}
				if len(m.RawJSON) == 0 {
					t.Error("RawJSON should be populated")
				}
			},
		},
		{
			name: "ShouldDisplayToUser returns false",
			json: `{"type":"something_new"}`,
			checkResult: func(t *testing.T, msg Message) {
				if msg.ShouldDisplayToUser() {
					t.Error("ShouldDisplayToUser() should return false")
				}
			},
		},
		{
			name: "GetMessageType returns unknown type string",
			json: `{"type":"abc_123"}`,
			checkResult: func(t *testing.T, msg Message) {
				if msg.GetMessageType() != "abc_123" {
					t.Errorf("GetMessageType() = %q, want %q", msg.GetMessageType(), "abc_123")
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
				t.Fatalf("UnmarshalMessage() should not error on unknown types, got %v", err)
			}
			if tt.checkResult != nil {
				tt.checkResult(t, msg)
			}
		})
	}
}

func TestUnmarshalMessageNoErrorOnUnknownType(t *testing.T) {
	t.Parallel()
	msg, err := UnmarshalMessage([]byte(`{"type":"never_seen_before","x":1}`))
	if err != nil {
		t.Fatalf("expected no error for unknown type, got %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
}

// ---------------------------------------------------------------------------
// Tests for empty/missing type still errors
// ---------------------------------------------------------------------------

func TestUnmarshalMessageMissingType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		json string
	}{
		{name: "missing type", json: `{"content":"hello"}`},
		{name: "null type", json: `{"type":null}`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := UnmarshalMessage([]byte(tt.json))
			if err == nil {
				t.Error("expected error for missing/null type")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Backward compatibility: existing subtypes still return *SystemMessage
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Tests for AgentInfo and RewindFilesResult (017-sdk-client-methods Phase 2)
// ---------------------------------------------------------------------------

func TestAgentInfo_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		info AgentInfo
	}{
		{
			name: "all fields populated",
			info: AgentInfo{
				Name:        "code-reviewer",
				Description: "Reviews code for style and correctness",
				Model:       "claude-sonnet-4-5-20250929",
			},
		},
		{
			name: "without optional model",
			info: AgentInfo{
				Name:        "researcher",
				Description: "Performs web research",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.info)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded AgentInfo
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.Name != tt.info.Name {
				t.Errorf("Name = %q, want %q", decoded.Name, tt.info.Name)
			}
			if decoded.Description != tt.info.Description {
				t.Errorf("Description = %q, want %q", decoded.Description, tt.info.Description)
			}
			if decoded.Model != tt.info.Model {
				t.Errorf("Model = %q, want %q", decoded.Model, tt.info.Model)
			}
		})
	}
}

func TestAgentInfo_OmitEmptyModel(t *testing.T) {
	t.Parallel()
	info := AgentInfo{
		Name:        "test-agent",
		Description: "A test agent",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["model"]; ok {
		t.Error("expected 'model' to be omitted when empty")
	}
	if _, ok := raw["name"]; !ok {
		t.Error("expected 'name' to be present")
	}
	if _, ok := raw["description"]; !ok {
		t.Error("expected 'description' to be present")
	}
}

func TestRewindFilesResult_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		result RewindFilesResult
	}{
		{
			name: "can rewind with changes",
			result: RewindFilesResult{
				CanRewind:    true,
				FilesChanged: []string{"main.go", "util.go"},
				Insertions:   15,
				Deletions:    8,
			},
		},
		{
			name: "cannot rewind with error",
			result: RewindFilesResult{
				CanRewind: false,
				Error:     "no checkpoint found",
			},
		},
		{
			name: "can rewind no changes",
			result: RewindFilesResult{
				CanRewind: true,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded RewindFilesResult
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.CanRewind != tt.result.CanRewind {
				t.Errorf("CanRewind = %v, want %v", decoded.CanRewind, tt.result.CanRewind)
			}
			if decoded.Error != tt.result.Error {
				t.Errorf("Error = %q, want %q", decoded.Error, tt.result.Error)
			}
			if len(decoded.FilesChanged) != len(tt.result.FilesChanged) {
				t.Fatalf("FilesChanged len = %d, want %d", len(decoded.FilesChanged), len(tt.result.FilesChanged))
			}
			for i, f := range tt.result.FilesChanged {
				if decoded.FilesChanged[i] != f {
					t.Errorf("FilesChanged[%d] = %q, want %q", i, decoded.FilesChanged[i], f)
				}
			}
			if decoded.Insertions != tt.result.Insertions {
				t.Errorf("Insertions = %d, want %d", decoded.Insertions, tt.result.Insertions)
			}
			if decoded.Deletions != tt.result.Deletions {
				t.Errorf("Deletions = %d, want %d", decoded.Deletions, tt.result.Deletions)
			}
		})
	}
}

func TestRewindFilesResult_OmitEmpty(t *testing.T) {
	t.Parallel()
	result := RewindFilesResult{
		CanRewind: true,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// canRewind is required (not omitempty), so it should be present
	if _, ok := raw["canRewind"]; !ok {
		t.Error("expected 'canRewind' to be present")
	}

	// Optional fields should be omitted when zero
	for _, key := range []string{"error", "filesChanged", "insertions", "deletions"} {
		if _, ok := raw[key]; ok {
			t.Errorf("expected JSON key %q to be omitted when zero", key)
		}
	}
}

func TestRewindFilesResult_WireFormatKeys(t *testing.T) {
	t.Parallel()
	result := RewindFilesResult{
		CanRewind:    true,
		Error:        "test",
		FilesChanged: []string{"a.go"},
		Insertions:   1,
		Deletions:    2,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	expectedKeys := []string{"canRewind", "error", "filesChanged", "insertions", "deletions"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected camelCase JSON key %q", key)
		}
	}
}

func TestInitializeResult_AgentsField(t *testing.T) {
	t.Parallel()
	result := InitializeResult{
		Commands: []SlashCommand{{Name: "help", Description: "Show help"}},
		Models:   []ModelInfo{{Value: "claude-3-opus", DisplayName: "Opus"}},
		Agents: []AgentInfo{
			{Name: "coder", Description: "Writes code", Model: "claude-sonnet-4-5-20250929"},
			{Name: "reviewer", Description: "Reviews code"},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded InitializeResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(decoded.Agents) != 2 {
		t.Fatalf("Agents len = %d, want 2", len(decoded.Agents))
	}
	if decoded.Agents[0].Name != "coder" {
		t.Errorf("Agents[0].Name = %q, want %q", decoded.Agents[0].Name, "coder")
	}
	if decoded.Agents[1].Model != "" {
		t.Errorf("Agents[1].Model = %q, want empty", decoded.Agents[1].Model)
	}
}

func TestInitializeResult_AgentsOmitEmpty(t *testing.T) {
	t.Parallel()
	result := InitializeResult{
		Commands: []SlashCommand{{Name: "help", Description: "Show help"}},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["agents"]; ok {
		t.Error("expected 'agents' to be omitted when nil/empty")
	}
}

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
