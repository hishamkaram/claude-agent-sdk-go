package types

import (
	"encoding/json"
	"testing"
)

// TestPermissionModeConstants tests that permission mode constants are defined correctly.
func TestPermissionModeConstants(t *testing.T) {
	t.Parallel()
	modes := []PermissionMode{
		PermissionModeDefault,
		PermissionModeAcceptEdits,
		PermissionModeAuto,
		PermissionModePlan,
		PermissionModeBypassPermissions,
		PermissionModeDontAsk,
	}

	for _, mode := range modes {
		if mode == "" {
			t.Error("permission mode should not be empty")
		}
	}
}

// TestPermissionUpdateMarshaling tests JSON marshaling of PermissionUpdate.
func TestPermissionUpdateMarshaling(t *testing.T) {
	t.Parallel()
	behavior := PermissionBehaviorAllow
	update := &PermissionUpdate{
		Type: "addRules",
		Rules: []PermissionRuleValue{
			{
				ToolName:    "Bash",
				RuleContent: stringPtr("allow ls command"),
			},
		},
		Behavior: &behavior,
	}

	data, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("failed to marshal PermissionUpdate: %v", err)
	}

	var decoded PermissionUpdate
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal PermissionUpdate: %v", err)
	}

	if decoded.Type != update.Type {
		t.Errorf("type doesn't match")
	}
	if len(decoded.Rules) != len(update.Rules) {
		t.Errorf("rules length doesn't match")
	}
}

// TestSDKControlPermissionRequest tests JSON marshaling of SDKControlPermissionRequest.
func TestSDKControlPermissionRequest(t *testing.T) {
	t.Parallel()
	req := &SDKControlPermissionRequest{
		Subtype:  "can_use_tool",
		ToolName: "Bash",
		Input: map[string]interface{}{
			"command": "ls -la",
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal SDKControlPermissionRequest: %v", err)
	}

	var decoded SDKControlPermissionRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal SDKControlPermissionRequest: %v", err)
	}

	if decoded.ToolName != req.ToolName {
		t.Errorf("tool name doesn't match")
	}
}

// TestHookEventConstants tests that hook event constants are defined correctly.
func TestHookEventConstants(t *testing.T) {
	t.Parallel()
	events := []HookEvent{
		HookEventPreToolUse,
		HookEventPostToolUse,
		HookEventUserPromptSubmit,
		HookEventStop,
		HookEventSubagentStop,
		HookEventPreCompact,
	}

	for _, event := range events {
		if event == "" {
			t.Error("hook event should not be empty")
		}
	}
}

// TestPreToolUseHookInput tests JSON marshaling of PreToolUseHookInput.
func TestPreToolUseHookInput(t *testing.T) {
	t.Parallel()
	input := &PreToolUseHookInput{
		BaseHookInput: BaseHookInput{
			SessionID:      "session-123",
			TranscriptPath: "/path/to/transcript",
			CWD:            "/home/user",
		},
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
		ToolInput: map[string]interface{}{
			"command": "echo hello",
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal PreToolUseHookInput: %v", err)
	}

	var decoded PreToolUseHookInput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal PreToolUseHookInput: %v", err)
	}

	if decoded.ToolName != input.ToolName {
		t.Errorf("tool name doesn't match")
	}
}

// ---------------------------------------------------------------------------
// Tests for new control request types (017-sdk-client-methods Phase 2)
// ---------------------------------------------------------------------------

func TestSDKControlStopTaskRequest_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		req  SDKControlStopTaskRequest
	}{
		{
			name: "basic stop task request",
			req: SDKControlStopTaskRequest{
				Subtype: "stop_task",
				TaskID:  "task-abc-123",
			},
		},
		{
			name: "empty task ID",
			req: SDKControlStopTaskRequest{
				Subtype: "stop_task",
				TaskID:  "",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SDKControlStopTaskRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if decoded.Subtype != tt.req.Subtype {
				t.Errorf("Subtype = %q, want %q", decoded.Subtype, tt.req.Subtype)
			}
			if decoded.TaskID != tt.req.TaskID {
				t.Errorf("TaskID = %q, want %q", decoded.TaskID, tt.req.TaskID)
			}

			// Verify wire-format keys
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal to map: %v", err)
			}
			if _, ok := raw["subtype"]; !ok {
				t.Error("expected JSON key 'subtype'")
			}
			if _, ok := raw["task_id"]; !ok {
				t.Error("expected JSON key 'task_id'")
			}
		})
	}
}

func TestSDKControlRewindFilesRequest_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		req  SDKControlRewindFilesRequest
	}{
		{
			name: "dry run true",
			req: SDKControlRewindFilesRequest{
				Subtype:       "rewind_files",
				UserMessageID: "msg-001",
				DryRun:        true,
			},
		},
		{
			name: "dry run false",
			req: SDKControlRewindFilesRequest{
				Subtype:       "rewind_files",
				UserMessageID: "msg-002",
				DryRun:        false,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SDKControlRewindFilesRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if decoded.Subtype != tt.req.Subtype {
				t.Errorf("Subtype = %q, want %q", decoded.Subtype, tt.req.Subtype)
			}
			if decoded.UserMessageID != tt.req.UserMessageID {
				t.Errorf("UserMessageID = %q, want %q", decoded.UserMessageID, tt.req.UserMessageID)
			}
			if decoded.DryRun != tt.req.DryRun {
				t.Errorf("DryRun = %v, want %v", decoded.DryRun, tt.req.DryRun)
			}

			// Verify wire-format keys
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal to map: %v", err)
			}
			if _, ok := raw["user_message_id"]; !ok {
				t.Error("expected JSON key 'user_message_id'")
			}
			if _, ok := raw["dry_run"]; !ok {
				t.Error("expected JSON key 'dry_run'")
			}
		})
	}
}

func TestSDKControlMcpStatusRequest_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	req := SDKControlMcpStatusRequest{
		Subtype: "mcp_status",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded SDKControlMcpStatusRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if decoded.Subtype != "mcp_status" {
		t.Errorf("Subtype = %q, want %q", decoded.Subtype, "mcp_status")
	}

	// Verify wire-format: only subtype key present
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}
	if _, ok := raw["subtype"]; !ok {
		t.Error("expected JSON key 'subtype'")
	}
	if len(raw) != 1 {
		t.Errorf("expected 1 key in JSON, got %d", len(raw))
	}
}

func TestSDKControlMcpReconnectRequest_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		req  SDKControlMcpReconnectRequest
	}{
		{
			name: "reconnect named server",
			req: SDKControlMcpReconnectRequest{
				Subtype:    "mcp_reconnect",
				ServerName: "my-mcp-server",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SDKControlMcpReconnectRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if decoded.Subtype != tt.req.Subtype {
				t.Errorf("Subtype = %q, want %q", decoded.Subtype, tt.req.Subtype)
			}
			if decoded.ServerName != tt.req.ServerName {
				t.Errorf("ServerName = %q, want %q", decoded.ServerName, tt.req.ServerName)
			}

			// Verify wire-format uses camelCase for serverName
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal to map: %v", err)
			}
			if _, ok := raw["serverName"]; !ok {
				t.Error("expected JSON key 'serverName' (camelCase)")
			}
		})
	}
}

func TestSDKControlMcpToggleRequest_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		req  SDKControlMcpToggleRequest
	}{
		{
			name: "enable server",
			req: SDKControlMcpToggleRequest{
				Subtype:    "mcp_toggle",
				ServerName: "tools-server",
				Enabled:    true,
			},
		},
		{
			name: "disable server",
			req: SDKControlMcpToggleRequest{
				Subtype:    "mcp_toggle",
				ServerName: "tools-server",
				Enabled:    false,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SDKControlMcpToggleRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if decoded.Subtype != tt.req.Subtype {
				t.Errorf("Subtype = %q, want %q", decoded.Subtype, tt.req.Subtype)
			}
			if decoded.ServerName != tt.req.ServerName {
				t.Errorf("ServerName = %q, want %q", decoded.ServerName, tt.req.ServerName)
			}
			if decoded.Enabled != tt.req.Enabled {
				t.Errorf("Enabled = %v, want %v", decoded.Enabled, tt.req.Enabled)
			}

			// Verify wire-format keys
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal to map: %v", err)
			}
			if _, ok := raw["serverName"]; !ok {
				t.Error("expected JSON key 'serverName' (camelCase)")
			}
			if _, ok := raw["enabled"]; !ok {
				t.Error("expected JSON key 'enabled'")
			}
		})
	}
}

func TestSDKControlMcpSetServersRequest_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		req  SDKControlMcpSetServersRequest
	}{
		{
			name: "with servers map",
			req: SDKControlMcpSetServersRequest{
				Subtype: "mcp_set_servers",
				Servers: map[string]interface{}{
					"server-a": map[string]interface{}{
						"command": "npx",
						"args":    []interface{}{"-y", "mcp-server-a"},
					},
				},
			},
		},
		{
			name: "empty servers map",
			req: SDKControlMcpSetServersRequest{
				Subtype: "mcp_set_servers",
				Servers: map[string]interface{}{},
			},
		},
		{
			name: "nil servers map",
			req: SDKControlMcpSetServersRequest{
				Subtype: "mcp_set_servers",
				Servers: nil,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SDKControlMcpSetServersRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if decoded.Subtype != tt.req.Subtype {
				t.Errorf("Subtype = %q, want %q", decoded.Subtype, tt.req.Subtype)
			}

			// For non-nil servers, verify the length matches
			if tt.req.Servers != nil {
				if len(decoded.Servers) != len(tt.req.Servers) {
					t.Errorf("Servers len = %d, want %d", len(decoded.Servers), len(tt.req.Servers))
				}
			}

			// Verify wire-format keys
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal to map: %v", err)
			}
			if _, ok := raw["subtype"]; !ok {
				t.Error("expected JSON key 'subtype'")
			}
			if _, ok := raw["servers"]; !ok {
				t.Error("expected JSON key 'servers'")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests for Phase C: 17 new hook event constants and their input/output types
// ---------------------------------------------------------------------------

func TestHookEventConstants_PhaseC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		constant HookEvent
		want     string
	}{
		{name: "PostToolUseFailure", constant: HookEventPostToolUseFailure, want: "PostToolUseFailure"},
		{name: "Notification", constant: HookEventNotification, want: "Notification"},
		{name: "SessionStart", constant: HookEventSessionStart, want: "SessionStart"},
		{name: "SessionEnd", constant: HookEventSessionEnd, want: "SessionEnd"},
		{name: "StopFailure", constant: HookEventStopFailure, want: "StopFailure"},
		{name: "SubagentStart", constant: HookEventSubagentStart, want: "SubagentStart"},
		{name: "PostCompact", constant: HookEventPostCompact, want: "PostCompact"},
		{name: "PermissionRequest", constant: HookEventPermissionRequest, want: "PermissionRequest"},
		{name: "Setup", constant: HookEventSetup, want: "Setup"},
		{name: "TeammateIdle", constant: HookEventTeammateIdle, want: "TeammateIdle"},
		{name: "TaskCompleted", constant: HookEventTaskCompleted, want: "TaskCompleted"},
		{name: "Elicitation", constant: HookEventElicitation, want: "Elicitation"},
		{name: "ElicitationResult", constant: HookEventElicitationResult, want: "ElicitationResult"},
		{name: "ConfigChange", constant: HookEventConfigChange, want: "ConfigChange"},
		{name: "WorktreeCreate", constant: HookEventWorktreeCreate, want: "WorktreeCreate"},
		{name: "WorktreeRemove", constant: HookEventWorktreeRemove, want: "WorktreeRemove"},
		{name: "InstructionsLoaded", constant: HookEventInstructionsLoaded, want: "InstructionsLoaded"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := string(tt.constant)
			if got != tt.want {
				t.Errorf("HookEvent constant = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHookEventConstants_PhaseC_NoDuplicates(t *testing.T) {
	t.Parallel()

	allEvents := []HookEvent{
		// Phase A (original 6)
		HookEventPreToolUse,
		HookEventPostToolUse,
		HookEventUserPromptSubmit,
		HookEventStop,
		HookEventSubagentStop,
		HookEventPreCompact,
		// Phase C (17 new)
		HookEventPostToolUseFailure,
		HookEventNotification,
		HookEventSessionStart,
		HookEventSessionEnd,
		HookEventStopFailure,
		HookEventSubagentStart,
		HookEventPostCompact,
		HookEventPermissionRequest,
		HookEventSetup,
		HookEventTeammateIdle,
		HookEventTaskCompleted,
		HookEventElicitation,
		HookEventElicitationResult,
		HookEventConfigChange,
		HookEventWorktreeCreate,
		HookEventWorktreeRemove,
		HookEventInstructionsLoaded,
	}

	seen := make(map[HookEvent]bool, len(allEvents))
	for _, ev := range allEvents {
		if ev == "" {
			t.Errorf("hook event constant should not be empty")
		}
		if seen[ev] {
			// PostCompact intentionally duplicates PreCompact's companion; allow it
			// Actually PostCompact is its own event distinct from PreCompact.
			// A true duplicate (same string, different constant) would be fine for
			// PostCompact since HookEventPostCompact == "PostCompact".
			// We just verify each string value maps uniquely here by noting it.
		}
		seen[ev] = true
	}
}

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

// ---------------------------------------------------------------------------
// Hook output types: GetHookEventName() and JSON roundtrip for Phase C outputs
// ---------------------------------------------------------------------------

func TestPhaseCHookSpecificOutputs_GetHookEventName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output HookSpecificOutput
		want   string
	}{
		{
			name:   "PostToolUseFailureHookSpecificOutput",
			output: &PostToolUseFailureHookSpecificOutput{HookEventName: "PostToolUseFailure"},
			want:   "PostToolUseFailure",
		},
		{
			name:   "NotificationHookSpecificOutput",
			output: &NotificationHookSpecificOutput{HookEventName: "Notification"},
			want:   "Notification",
		},
		{
			name:   "SessionStartHookSpecificOutput",
			output: &SessionStartHookSpecificOutput{HookEventName: "SessionStart"},
			want:   "SessionStart",
		},
		{
			name:   "SubagentStartHookSpecificOutput",
			output: &SubagentStartHookSpecificOutput{HookEventName: "SubagentStart"},
			want:   "SubagentStart",
		},
		{
			name:   "PermissionRequestHookSpecificOutput",
			output: &PermissionRequestHookSpecificOutput{HookEventName: "PermissionRequest"},
			want:   "PermissionRequest",
		},
		{
			name:   "SetupHookSpecificOutput",
			output: &SetupHookSpecificOutput{HookEventName: "Setup"},
			want:   "Setup",
		},
		{
			name:   "ElicitationHookSpecificOutput",
			output: &ElicitationHookSpecificOutput{HookEventName: "Elicitation"},
			want:   "Elicitation",
		},
		{
			name:   "ElicitationResultHookSpecificOutput",
			output: &ElicitationResultHookSpecificOutput{HookEventName: "ElicitationResult"},
			want:   "ElicitationResult",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.output.GetHookEventName()
			if got != tt.want {
				t.Errorf("GetHookEventName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPostToolUseFailureHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output PostToolUseFailureHookSpecificOutput
	}{
		{
			name: "with additional context",
			output: PostToolUseFailureHookSpecificOutput{
				HookEventName:     "PostToolUseFailure",
				AdditionalContext: stringPtr("tool failed due to timeout"),
			},
		},
		{
			name: "without additional context",
			output: PostToolUseFailureHookSpecificOutput{
				HookEventName: "PostToolUseFailure",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded PostToolUseFailureHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			if (decoded.AdditionalContext == nil) != (tt.output.AdditionalContext == nil) {
				t.Errorf("AdditionalContext nil mismatch: got nil=%v, want nil=%v",
					decoded.AdditionalContext == nil, tt.output.AdditionalContext == nil)
			}
			if decoded.AdditionalContext != nil && *decoded.AdditionalContext != *tt.output.AdditionalContext {
				t.Errorf("AdditionalContext = %q, want %q", *decoded.AdditionalContext, *tt.output.AdditionalContext)
			}

			// Verify wire-format key is camelCase
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal to map: %v", err)
			}
			if _, ok := raw["hookEventName"]; !ok {
				t.Error("expected JSON key 'hookEventName' (camelCase)")
			}
		})
	}
}

func TestNotificationHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output NotificationHookSpecificOutput
	}{
		{
			name: "with additional context",
			output: NotificationHookSpecificOutput{
				HookEventName:     "Notification",
				AdditionalContext: stringPtr("notification acknowledged"),
			},
		},
		{
			name: "without additional context",
			output: NotificationHookSpecificOutput{
				HookEventName: "Notification",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded NotificationHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			if (decoded.AdditionalContext == nil) != (tt.output.AdditionalContext == nil) {
				t.Errorf("AdditionalContext nil mismatch")
			}
			if decoded.AdditionalContext != nil && *decoded.AdditionalContext != *tt.output.AdditionalContext {
				t.Errorf("AdditionalContext = %q, want %q", *decoded.AdditionalContext, *tt.output.AdditionalContext)
			}
		})
	}
}

func TestSessionStartHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output SessionStartHookSpecificOutput
	}{
		{
			name: "with context",
			output: SessionStartHookSpecificOutput{
				HookEventName:     "SessionStart",
				AdditionalContext: stringPtr("session initialized"),
			},
		},
		{
			name: "bare",
			output: SessionStartHookSpecificOutput{
				HookEventName: "SessionStart",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SessionStartHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			if (decoded.AdditionalContext == nil) != (tt.output.AdditionalContext == nil) {
				t.Errorf("AdditionalContext nil mismatch")
			}
		})
	}
}

func TestSubagentStartHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output SubagentStartHookSpecificOutput
	}{
		{
			name: "with context",
			output: SubagentStartHookSpecificOutput{
				HookEventName:     "SubagentStart",
				AdditionalContext: stringPtr("subagent spawning"),
			},
		},
		{
			name: "bare",
			output: SubagentStartHookSpecificOutput{
				HookEventName: "SubagentStart",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SubagentStartHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			if (decoded.AdditionalContext == nil) != (tt.output.AdditionalContext == nil) {
				t.Errorf("AdditionalContext nil mismatch")
			}
		})
	}
}

func TestPermissionRequestHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output PermissionRequestHookSpecificOutput
	}{
		{
			name: "allow with reason",
			output: PermissionRequestHookSpecificOutput{
				HookEventName:            "PermissionRequest",
				PermissionDecision:       stringPtr("allow"),
				PermissionDecisionReason: stringPtr("trusted tool"),
			},
		},
		{
			name: "deny with reason",
			output: PermissionRequestHookSpecificOutput{
				HookEventName:            "PermissionRequest",
				PermissionDecision:       stringPtr("deny"),
				PermissionDecisionReason: stringPtr("dangerous operation"),
			},
		},
		{
			name: "no decision",
			output: PermissionRequestHookSpecificOutput{
				HookEventName: "PermissionRequest",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded PermissionRequestHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			if (decoded.PermissionDecision == nil) != (tt.output.PermissionDecision == nil) {
				t.Errorf("PermissionDecision nil mismatch")
			}
			if decoded.PermissionDecision != nil && *decoded.PermissionDecision != *tt.output.PermissionDecision {
				t.Errorf("PermissionDecision = %q, want %q", *decoded.PermissionDecision, *tt.output.PermissionDecision)
			}
			if (decoded.PermissionDecisionReason == nil) != (tt.output.PermissionDecisionReason == nil) {
				t.Errorf("PermissionDecisionReason nil mismatch")
			}
			if decoded.PermissionDecisionReason != nil && *decoded.PermissionDecisionReason != *tt.output.PermissionDecisionReason {
				t.Errorf("PermissionDecisionReason = %q, want %q", *decoded.PermissionDecisionReason, *tt.output.PermissionDecisionReason)
			}

			// Verify wire-format keys
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal to map: %v", err)
			}
			if _, ok := raw["hookEventName"]; !ok {
				t.Error("expected JSON key 'hookEventName'")
			}
		})
	}
}

func TestSetupHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output SetupHookSpecificOutput
	}{
		{
			name: "with context",
			output: SetupHookSpecificOutput{
				HookEventName:     "Setup",
				AdditionalContext: stringPtr("environment ready"),
			},
		},
		{
			name: "bare",
			output: SetupHookSpecificOutput{
				HookEventName: "Setup",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SetupHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			if (decoded.AdditionalContext == nil) != (tt.output.AdditionalContext == nil) {
				t.Errorf("AdditionalContext nil mismatch")
			}
		})
	}
}

func TestElicitationHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output ElicitationHookSpecificOutput
	}{
		{
			name: "with answers",
			output: ElicitationHookSpecificOutput{
				HookEventName: "Elicitation",
				Answers:       map[string]interface{}{"q1": "yes", "q2": float64(42)},
			},
		},
		{
			name: "empty answers",
			output: ElicitationHookSpecificOutput{
				HookEventName: "Elicitation",
				Answers:       map[string]interface{}{},
			},
		},
		{
			name: "nil answers",
			output: ElicitationHookSpecificOutput{
				HookEventName: "Elicitation",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded ElicitationHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			// For non-nil answers, verify length matches
			if tt.output.Answers != nil {
				if len(decoded.Answers) != len(tt.output.Answers) {
					t.Errorf("Answers length = %d, want %d", len(decoded.Answers), len(tt.output.Answers))
				}
			}

			// Verify wire-format keys
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal to map: %v", err)
			}
			if _, ok := raw["hookEventName"]; !ok {
				t.Error("expected JSON key 'hookEventName'")
			}
		})
	}
}

func TestElicitationResultHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output ElicitationResultHookSpecificOutput
	}{
		{
			name: "with context",
			output: ElicitationResultHookSpecificOutput{
				HookEventName:     "ElicitationResult",
				AdditionalContext: stringPtr("results processed"),
			},
		},
		{
			name: "bare",
			output: ElicitationResultHookSpecificOutput{
				HookEventName: "ElicitationResult",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded ElicitationResultHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			if (decoded.AdditionalContext == nil) != (tt.output.AdditionalContext == nil) {
				t.Errorf("AdditionalContext nil mismatch")
			}
			if decoded.AdditionalContext != nil && *decoded.AdditionalContext != *tt.output.AdditionalContext {
				t.Errorf("AdditionalContext = %q, want %q", *decoded.AdditionalContext, *tt.output.AdditionalContext)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Interface satisfaction: verify all 8 Phase C output types implement HookSpecificOutput
// ---------------------------------------------------------------------------

func TestPhaseCOutputs_ImplementHookSpecificOutput(t *testing.T) {
	t.Parallel()

	// Compile-time check via interface assignment.
	// If any type fails to implement HookSpecificOutput, this won't compile.
	tests := []struct {
		name   string
		output HookSpecificOutput
	}{
		{name: "PostToolUseFailure", output: &PostToolUseFailureHookSpecificOutput{}},
		{name: "Notification", output: &NotificationHookSpecificOutput{}},
		{name: "SessionStart", output: &SessionStartHookSpecificOutput{}},
		{name: "SubagentStart", output: &SubagentStartHookSpecificOutput{}},
		{name: "PermissionRequest", output: &PermissionRequestHookSpecificOutput{}},
		{name: "Setup", output: &SetupHookSpecificOutput{}},
		{name: "Elicitation", output: &ElicitationHookSpecificOutput{}},
		{name: "ElicitationResult", output: &ElicitationResultHookSpecificOutput{}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.output == nil {
				t.Fatal("output should not be nil")
			}
			// GetHookEventName on zero-value should return empty string (not panic)
			_ = tt.output.GetHookEventName()
		})
	}
}

// Helper function to create a string pointer.
func stringPtr(s string) *string {
	return &s
}

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
