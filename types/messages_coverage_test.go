package types

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestTruncateRaw tests the truncateRaw helper function.
func TestTruncateRaw(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "shorter than max — no truncation",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exactly max — no truncation",
			input:  "1234567890",
			maxLen: 10,
			want:   "1234567890",
		},
		{
			name:   "longer than max — truncated with ellipsis",
			input:  "12345678901",
			maxLen: 10,
			want:   "1234567890...",
		},
		{
			name:   "empty string — no truncation",
			input:  "",
			maxLen: 5,
			want:   "",
		},
		{
			name:   "max zero — truncated to empty plus ellipsis",
			input:  "abc",
			maxLen: 0,
			want:   "...",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := truncateRaw(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateRaw(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// TestUnmarshalMessage_AllTypes tests UnmarshalMessage for every top-level message type.
func TestUnmarshalMessage_AllTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		json     string
		wantType string
		wantErr  bool
	}{
		{
			name:     "user message",
			json:     `{"type":"user","content":"hello"}`,
			wantType: "user",
		},
		{
			name:     "assistant message",
			json:     `{"type":"assistant","content":[{"type":"text","text":"hi"}]}`,
			wantType: "assistant",
		},
		{
			name:     "system init message",
			json:     `{"type":"system","subtype":"init"}`,
			wantType: "system",
		},
		{
			name:     "system warning message",
			json:     `{"type":"system","subtype":"warning"}`,
			wantType: "system",
		},
		{
			name:     "result message",
			json:     `{"type":"result","subtype":"success","duration_ms":100,"session_id":"s1"}`,
			wantType: "result",
		},
		{
			name:     "stream event",
			json:     `{"type":"stream_event","uuid":"u1","session_id":"s1","event":{}}`,
			wantType: "stream_event",
		},
		{
			name:     "control_request",
			json:     `{"type":"control_request","request_id":"r1"}`,
			wantType: "control_request",
		},
		{
			name:     "control_response",
			json:     `{"type":"control_response","request_id":"r1"}`,
			wantType: "control_response",
		},
		{
			name:     "tool_progress",
			json:     `{"type":"tool_progress","tool_use_id":"tu1","tool_name":"Bash","elapsed_time_seconds":1.5,"uuid":"u1","session_id":"s1"}`,
			wantType: "tool_progress",
		},
		{
			name:     "auth_status",
			json:     `{"type":"auth_status","isAuthenticating":true,"output":[],"uuid":"u1","session_id":"s1"}`,
			wantType: "auth_status",
		},
		{
			name:     "tool_use_summary",
			json:     `{"type":"tool_use_summary","summary":"s","preceding_tool_use_ids":[],"uuid":"u1","session_id":"s1"}`,
			wantType: "tool_use_summary",
		},
		{
			name:     "rate_limit_event",
			json:     `{"type":"rate_limit_event","rate_limit_info":{"status":"limited"},"uuid":"u1","session_id":"s1"}`,
			wantType: "rate_limit_event",
		},
		{
			name:     "prompt_suggestion",
			json:     `{"type":"prompt_suggestion","suggestion":"try this","uuid":"u1","session_id":"s1"}`,
			wantType: "prompt_suggestion",
		},
		{
			name:     "unknown type preserved",
			json:     `{"type":"future_type","data":"value"}`,
			wantType: "future_type",
		},
		{
			name:    "empty type field",
			json:    `{"type":""}`,
			wantErr: true,
		},
		{
			name:    "missing type field",
			json:    `{"data":"value"}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			json:    `{broken`,
			wantErr: true,
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
			if !tt.wantErr && msg.GetMessageType() != tt.wantType {
				t.Errorf("GetMessageType() = %q, want %q", msg.GetMessageType(), tt.wantType)
			}
		})
	}
}

// TestUnmarshalSystemMessage_Subtypes tests system message subtype routing.
func TestUnmarshalSystemMessage_Subtypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		json        string
		wantSubtype string
		wantMsgType interface{}
	}{
		{
			name:        "compact_boundary",
			json:        `{"type":"system","subtype":"compact_boundary","compact_metadata":{"trigger":"auto","pre_tokens":100},"uuid":"u","session_id":"s"}`,
			wantSubtype: "compact_boundary",
		},
		{
			name:        "status",
			json:        `{"type":"system","subtype":"status","uuid":"u","session_id":"s"}`,
			wantSubtype: "status",
		},
		{
			name:        "hook_started",
			json:        `{"type":"system","subtype":"hook_started","hook_id":"h1","hook_name":"n","hook_event":"e","uuid":"u","session_id":"s"}`,
			wantSubtype: "hook_started",
		},
		{
			name:        "hook_progress",
			json:        `{"type":"system","subtype":"hook_progress","hook_id":"h1","hook_name":"n","hook_event":"e","stdout":"","stderr":"","output":"","uuid":"u","session_id":"s"}`,
			wantSubtype: "hook_progress",
		},
		{
			name:        "hook_response",
			json:        `{"type":"system","subtype":"hook_response","hook_id":"h1","hook_name":"n","hook_event":"e","output":"","stdout":"","stderr":"","outcome":"ok","uuid":"u","session_id":"s"}`,
			wantSubtype: "hook_response",
		},
		{
			name:        "task_notification",
			json:        `{"type":"system","subtype":"task_notification","task_id":"t1","status":"done","output_file":"f","summary":"s","uuid":"u","session_id":"s"}`,
			wantSubtype: "task_notification",
		},
		{
			name:        "task_started",
			json:        `{"type":"system","subtype":"task_started","task_id":"t1","description":"d","uuid":"u","session_id":"s"}`,
			wantSubtype: "task_started",
		},
		{
			name:        "task_progress",
			json:        `{"type":"system","subtype":"task_progress","task_id":"t1","description":"d","usage":{"total_tokens":0,"tool_uses":0,"duration_ms":0},"uuid":"u","session_id":"s"}`,
			wantSubtype: "task_progress",
		},
		{
			name:        "files_persisted",
			json:        `{"type":"system","subtype":"files_persisted","files":[],"failed":[],"processed_at":"now","uuid":"u","session_id":"s"}`,
			wantSubtype: "files_persisted",
		},
		{
			name:        "generic info subtype",
			json:        `{"type":"system","subtype":"info"}`,
			wantSubtype: "info",
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
			if msg.GetMessageType() != "system" {
				t.Errorf("GetMessageType() = %q, want %q", msg.GetMessageType(), "system")
			}
		})
	}
}

// TestShouldDisplayToUser tests the ShouldDisplayToUser method for all message types.
func TestShouldDisplayToUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  Message
		want bool
	}{
		{"UserMessage", &UserMessage{Type: "user"}, true},
		{"AssistantMessage", &AssistantMessage{Type: "assistant"}, true},
		{"SystemMessage init", &SystemMessage{Type: "system", Subtype: "init"}, false},
		{"SystemMessage debug", &SystemMessage{Type: "system", Subtype: "debug"}, false},
		{"SystemMessage warning", &SystemMessage{Type: "system", Subtype: "warning"}, true},
		{"SystemMessage error", &SystemMessage{Type: "system", Subtype: "error"}, true},
		{"SystemMessage info", &SystemMessage{Type: "system", Subtype: "info"}, true},
		{"ResultMessage", &ResultMessage{Type: "result"}, false},
		{"StreamEvent", &StreamEvent{Type: "stream_event"}, false},
		{"ToolProgressMessage", &ToolProgressMessage{Type: "tool_progress"}, false},
		{"AuthStatusMessage", &AuthStatusMessage{Type: "auth_status"}, true},
		{"ToolUseSummaryMessage", &ToolUseSummaryMessage{Type: "tool_use_summary"}, false},
		{"RateLimitEvent", &RateLimitEvent{Type: "rate_limit_event"}, true},
		{"PromptSuggestionMessage", &PromptSuggestionMessage{Type: "prompt_suggestion"}, false},
		{"CompactBoundaryMessage", &CompactBoundaryMessage{Type: "system"}, false},
		{"StatusMessage", &StatusMessage{Type: "system"}, true},
		{"HookStartedMessage", &HookStartedMessage{Type: "system"}, false},
		{"HookProgressMessage", &HookProgressMessage{Type: "system"}, false},
		{"HookResponseMessage", &HookResponseMessage{Type: "system"}, false},
		{"TaskNotificationMessage", &TaskNotificationMessage{Type: "system"}, true},
		{"TaskStartedMessage", &TaskStartedMessage{Type: "system"}, true},
		{"TaskProgressMessage", &TaskProgressMessage{Type: "system"}, false},
		{"FilesPersistedEvent", &FilesPersistedEvent{Type: "system"}, false},
		{"UnknownMessage", &UnknownMessage{Type: "unknown"}, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.msg.ShouldDisplayToUser()
			if got != tt.want {
				t.Errorf("ShouldDisplayToUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSystemMessage_Helpers tests IsInit, IsWarning, IsError, IsInfo, IsDebug.
func TestSystemMessage_Helpers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		subtype string
		check   func(*SystemMessage) bool
		want    bool
	}{
		{"IsInit true", "init", (*SystemMessage).IsInit, true},
		{"IsInit false", "warning", (*SystemMessage).IsInit, false},
		{"IsWarning true", "warning", (*SystemMessage).IsWarning, true},
		{"IsWarning false", "error", (*SystemMessage).IsWarning, false},
		{"IsError true", "error", (*SystemMessage).IsError, true},
		{"IsError false", "info", (*SystemMessage).IsError, false},
		{"IsInfo true", "info", (*SystemMessage).IsInfo, true},
		{"IsInfo false", "debug", (*SystemMessage).IsInfo, false},
		{"IsDebug true", "debug", (*SystemMessage).IsDebug, true},
		{"IsDebug false", "init", (*SystemMessage).IsDebug, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg := &SystemMessage{Type: "system", Subtype: tt.subtype}
			if got := tt.check(msg); got != tt.want {
				t.Errorf("check(%q) = %v, want %v", tt.subtype, got, tt.want)
			}
		})
	}
}

// TestUnmarshalContentBlock_ThinkingBlock tests thinking block unmarshaling.
func TestUnmarshalContentBlock_ThinkingBlock(t *testing.T) {
	t.Parallel()
	data := `{"type":"thinking","thinking":"internal reasoning","signature":"sig123"}`
	block, err := UnmarshalContentBlock([]byte(data))
	if err != nil {
		t.Fatalf("UnmarshalContentBlock() error = %v", err)
	}
	tb, ok := block.(*ThinkingBlock)
	if !ok {
		t.Fatalf("expected *ThinkingBlock, got %T", block)
	}
	if tb.Thinking != "internal reasoning" {
		t.Errorf("Thinking = %q, want %q", tb.Thinking, "internal reasoning")
	}
	if tb.Signature != "sig123" {
		t.Errorf("Signature = %q, want %q", tb.Signature, "sig123")
	}
}

// TestUnmarshalContentBlock_Errors tests error paths in UnmarshalContentBlock.
func TestUnmarshalContentBlock_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "invalid JSON",
			json:    `{broken`,
			wantErr: true,
		},
		{
			name:    "unknown type",
			json:    `{"type":"future_block"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := UnmarshalContentBlock([]byte(tt.json))
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalContentBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAssistantMessage_NestedFormat tests AssistantMessage with nested message.content format.
func TestAssistantMessage_NestedFormat(t *testing.T) {
	t.Parallel()
	data := `{"type":"assistant","message":{"content":[{"type":"text","text":"hi"}],"model":"claude-3"}}`
	var msg AssistantMessage
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if msg.Model != "claude-3" {
		t.Errorf("Model = %q, want %q", msg.Model, "claude-3")
	}
	if len(msg.Content) != 1 {
		t.Fatalf("Content length = %d, want 1", len(msg.Content))
	}
}

// TestAssistantMessage_MarshalJSON tests that MarshalJSON roundtrips correctly.
func TestAssistantMessage_MarshalJSON(t *testing.T) {
	t.Parallel()
	msg := &AssistantMessage{
		Type:  "assistant",
		Model: "claude-3",
		Content: []ContentBlock{
			&TextBlock{Type: "text", Text: "hello"},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if !strings.Contains(string(data), "claude-3") {
		t.Errorf("marshaled JSON = %s, want to contain 'claude-3'", data)
	}
}

// TestUserMessage_NestedFormat tests UserMessage with nested message.content format.
func TestUserMessage_NestedFormat(t *testing.T) {
	t.Parallel()
	data := `{"type":"user","message":{"content":"hello from nested","parent_tool_use_id":"tu-1"}}`
	var msg UserMessage
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	content, ok := msg.Content.(string)
	if !ok {
		t.Fatalf("Content type = %T, want string", msg.Content)
	}
	if content != "hello from nested" {
		t.Errorf("Content = %q, want %q", content, "hello from nested")
	}
	if msg.ParentToolUseID == nil || *msg.ParentToolUseID != "tu-1" {
		t.Errorf("ParentToolUseID = %v, want 'tu-1'", msg.ParentToolUseID)
	}
}

// TestUserMessage_Errors tests UserMessage unmarshal error paths.
func TestUserMessage_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "invalid JSON",
			json:    `{broken`,
			wantErr: true,
		},
		{
			name:    "missing content field",
			json:    `{"type":"user"}`,
			wantErr: true,
		},
		{
			name:    "content is neither string nor array",
			json:    `{"type":"user","content":42}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var msg UserMessage
			err := json.Unmarshal([]byte(tt.json), &msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
