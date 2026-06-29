package types

import (
	"encoding/json"
	"testing"
)

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
