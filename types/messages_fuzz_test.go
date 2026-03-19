package types

import (
	"testing"
)

// FuzzUnmarshalMessage fuzzes UnmarshalMessage with random bytes to verify no panics.
func FuzzUnmarshalMessage(f *testing.F) {
	// Seed corpus with representative valid messages
	f.Add([]byte(`{"type": "user", "content": "hello"}`))
	f.Add([]byte(`{"type": "assistant", "content": [{"type": "text", "text": "hi"}], "model": "claude"}`))
	f.Add([]byte(`{"type": "system", "subtype": "init", "data": {}}`))
	f.Add([]byte(`{"type": "result", "subtype": "success", "is_error": false, "num_turns": 1, "session_id": "s1"}`))
	f.Add([]byte(`{"type": "stream_event", "uuid": "u1", "session_id": "s1", "event": {"type": "message_start"}}`))
	f.Add([]byte(`{"type": "tool_progress", "tool_use_id": "t1", "tool_name": "Bash", "elapsed_time_seconds": 1.0, "uuid": "u1", "session_id": "s1"}`))
	f.Add([]byte(`{"type": "auth_status", "isAuthenticating": true, "output": [], "uuid": "u1", "session_id": "s1"}`))
	f.Add([]byte(`{"type": "tool_use_summary", "summary": "test", "preceding_tool_use_ids": [], "uuid": "u1", "session_id": "s1"}`))
	f.Add([]byte(`{"type": "rate_limit_event", "rate_limit_info": {"status": "allowed"}, "uuid": "u1", "session_id": "s1"}`))
	f.Add([]byte(`{"type": "prompt_suggestion", "suggestion": "test", "uuid": "u1", "session_id": "s1"}`))
	f.Add([]byte(`{"type": "system", "subtype": "compact_boundary", "compact_metadata": {"trigger": "auto", "pre_tokens": 50000}}`))
	f.Add([]byte(`{"type": "system", "subtype": "status", "status": "compacting"}`))
	f.Add([]byte(`{"type": "system", "subtype": "hook_started", "hook_id": "h1", "hook_name": "test", "hook_event": "PostToolUse"}`))
	f.Add([]byte(`{"type": "system", "subtype": "hook_progress", "hook_id": "h2", "hook_name": "lint", "hook_event": "PostToolUse", "stdout": "ok", "stderr": "", "output": "ok"}`))
	f.Add([]byte(`{"type": "system", "subtype": "hook_response", "hook_id": "h3", "hook_name": "pre-commit", "hook_event": "PostToolUse", "exit_code": 0, "outcome": "success"}`))
	f.Add([]byte(`{"type": "system", "subtype": "task_notification", "task_id": "t1", "tool_use_id": "tu1", "status": "completed", "summary": "done"}`))
	f.Add([]byte(`{"type": "system", "subtype": "task_started", "task_id": "t2", "tool_use_id": "tu2", "description": "test", "task_type": "agent"}`))
	f.Add([]byte(`{"type": "system", "subtype": "task_progress", "task_id": "t3", "description": "running"}`))
	f.Add([]byte(`{"type": "system", "subtype": "files_persisted", "files": [{"filename": "a.go", "file_id": "f1"}], "failed": []}`))
	f.Add([]byte(`{"type": "control_request", "subtype": "test"}`))
	f.Add([]byte(`{"type": "control_response", "subtype": "test"}`))
	f.Add([]byte(`{"type": "system", "subtype": "unknown_future_subtype", "data": {}}`))
	f.Add([]byte(`{"type": "future_unknown_type", "data": {}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"type": ""}`))
	f.Add([]byte(`{"type": null}`))
	f.Add([]byte(`{"type": 123}`))
	f.Add([]byte(`not json`))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic — errors are acceptable
		msg, err := UnmarshalMessage(data)
		if err == nil && msg == nil {
			t.Error("UnmarshalMessage returned nil message with nil error")
		}
		if err != nil && msg != nil {
			t.Error("UnmarshalMessage returned non-nil message with non-nil error")
		}
	})
}
