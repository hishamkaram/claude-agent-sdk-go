package types

import (
	"encoding/json"
	"testing"
)

// --- Fuzz Tests for Hook Input Types (Phase C) ---

func FuzzPostToolUseFailureHookInput_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"session_id":"s1","transcript_path":"/t","cwd":"/c","hook_event_name":"PostToolUseFailure","tool_name":"Bash","tool_input":{},"error":"fail"}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var v PostToolUseFailureHookInput
		_ = json.Unmarshal(data, &v)
	})
}

func FuzzNotificationHookInput_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"session_id":"s1","transcript_path":"/t","cwd":"/c","hook_event_name":"Notification","message":"hello","level":"info"}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var v NotificationHookInput
		_ = json.Unmarshal(data, &v)
	})
}

func FuzzSessionStartHookInput_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"session_id":"s1","transcript_path":"/t","cwd":"/c","hook_event_name":"SessionStart"}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var v SessionStartHookInput
		_ = json.Unmarshal(data, &v)
	})
}

func FuzzElicitationHookInput_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"session_id":"s1","transcript_path":"/t","cwd":"/c","hook_event_name":"Elicitation","questions":[{"q":"test"}]}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var v ElicitationHookInput
		_ = json.Unmarshal(data, &v)
	})
}
