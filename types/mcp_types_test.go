package types

import (
	"encoding/json"
	"testing"
)

func TestMcpServerStatusInfo_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		info McpServerStatusInfo
	}{
		{
			name: "connected with server info and tools",
			info: McpServerStatusInfo{
				Name:   "my-mcp-server",
				Status: "connected",
				ServerInfo: &McpServerInfo{
					Name:    "MCP Server",
					Version: "1.2.3",
				},
				Scope: "project",
				Tools: []McpToolInfo{
					{Name: "read_file", Description: "Read a file"},
					{Name: "write_file", Description: "Write a file"},
				},
			},
		},
		{
			name: "disconnected with error",
			info: McpServerStatusInfo{
				Name:   "failing-server",
				Status: "disconnected",
				Error:  "connection refused",
			},
		},
		{
			name: "minimal fields",
			info: McpServerStatusInfo{
				Name:   "minimal",
				Status: "connecting",
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

			var decoded McpServerStatusInfo
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.Name != tt.info.Name {
				t.Errorf("Name = %q, want %q", decoded.Name, tt.info.Name)
			}
			if decoded.Status != tt.info.Status {
				t.Errorf("Status = %q, want %q", decoded.Status, tt.info.Status)
			}
			if decoded.Error != tt.info.Error {
				t.Errorf("Error = %q, want %q", decoded.Error, tt.info.Error)
			}
			if decoded.Scope != tt.info.Scope {
				t.Errorf("Scope = %q, want %q", decoded.Scope, tt.info.Scope)
			}

			if tt.info.ServerInfo != nil {
				if decoded.ServerInfo == nil {
					t.Fatal("ServerInfo should not be nil")
				}
				if decoded.ServerInfo.Name != tt.info.ServerInfo.Name {
					t.Errorf("ServerInfo.Name = %q, want %q", decoded.ServerInfo.Name, tt.info.ServerInfo.Name)
				}
				if decoded.ServerInfo.Version != tt.info.ServerInfo.Version {
					t.Errorf("ServerInfo.Version = %q, want %q", decoded.ServerInfo.Version, tt.info.ServerInfo.Version)
				}
			} else if decoded.ServerInfo != nil {
				t.Error("ServerInfo should be nil")
			}

			if len(decoded.Tools) != len(tt.info.Tools) {
				t.Fatalf("Tools len = %d, want %d", len(decoded.Tools), len(tt.info.Tools))
			}
			for i, tool := range tt.info.Tools {
				if decoded.Tools[i].Name != tool.Name {
					t.Errorf("Tools[%d].Name = %q, want %q", i, decoded.Tools[i].Name, tool.Name)
				}
				if decoded.Tools[i].Description != tool.Description {
					t.Errorf("Tools[%d].Description = %q, want %q", i, decoded.Tools[i].Description, tool.Description)
				}
			}
		})
	}
}

func TestMcpServerStatusInfo_OmitEmpty(t *testing.T) {
	t.Parallel()
	info := McpServerStatusInfo{
		Name:   "test",
		Status: "connected",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// Required fields present
	if _, ok := raw["name"]; !ok {
		t.Error("expected JSON key 'name'")
	}
	if _, ok := raw["status"]; !ok {
		t.Error("expected JSON key 'status'")
	}

	// Optional fields omitted when zero
	for _, key := range []string{"serverInfo", "error", "scope", "tools"} {
		if _, ok := raw[key]; ok {
			t.Errorf("expected JSON key %q to be omitted when zero", key)
		}
	}
}

func TestMcpServerStatusInfo_WireFormatKeys(t *testing.T) {
	t.Parallel()
	info := McpServerStatusInfo{
		Name:   "test",
		Status: "connected",
		ServerInfo: &McpServerInfo{
			Name:    "Server",
			Version: "1.0",
		},
		Error: "none",
		Scope: "project",
		Tools: []McpToolInfo{{Name: "tool1"}},
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	expectedKeys := []string{"name", "status", "serverInfo", "error", "scope", "tools"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected JSON key %q", key)
		}
	}
}

func TestMcpServerInfo_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		info McpServerInfo
	}{
		{
			name: "populated",
			info: McpServerInfo{
				Name:    "MCP File Server",
				Version: "2.1.0",
			},
		},
		{
			name: "empty strings",
			info: McpServerInfo{
				Name:    "",
				Version: "",
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

			var decoded McpServerInfo
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.Name != tt.info.Name {
				t.Errorf("Name = %q, want %q", decoded.Name, tt.info.Name)
			}
			if decoded.Version != tt.info.Version {
				t.Errorf("Version = %q, want %q", decoded.Version, tt.info.Version)
			}
		})
	}
}

func TestMcpToolInfo_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		info McpToolInfo
	}{
		{
			name: "with description",
			info: McpToolInfo{
				Name:        "read_file",
				Description: "Read the contents of a file",
			},
		},
		{
			name: "without description",
			info: McpToolInfo{
				Name: "list_files",
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

			var decoded McpToolInfo
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.Name != tt.info.Name {
				t.Errorf("Name = %q, want %q", decoded.Name, tt.info.Name)
			}
			if decoded.Description != tt.info.Description {
				t.Errorf("Description = %q, want %q", decoded.Description, tt.info.Description)
			}
		})
	}
}

func TestMcpToolInfo_OmitEmptyDescription(t *testing.T) {
	t.Parallel()
	info := McpToolInfo{Name: "tool1"}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["description"]; ok {
		t.Error("expected 'description' to be omitted when empty")
	}
}

func TestMcpSetServersResult_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		result McpSetServersResult
	}{
		{
			name: "added and removed with no errors",
			result: McpSetServersResult{
				Added:   []string{"server-a", "server-b"},
				Removed: []string{"server-c"},
				Errors:  map[string]string{},
			},
		},
		{
			name: "with errors",
			result: McpSetServersResult{
				Added:   []string{},
				Removed: []string{},
				Errors: map[string]string{
					"bad-server": "failed to connect",
				},
			},
		},
		{
			name: "nil slices and map",
			result: McpSetServersResult{
				Added:   nil,
				Removed: nil,
				Errors:  nil,
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

			var decoded McpSetServersResult
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			// Compare Added
			if tt.result.Added != nil {
				if len(decoded.Added) != len(tt.result.Added) {
					t.Fatalf("Added len = %d, want %d", len(decoded.Added), len(tt.result.Added))
				}
				for i, a := range tt.result.Added {
					if decoded.Added[i] != a {
						t.Errorf("Added[%d] = %q, want %q", i, decoded.Added[i], a)
					}
				}
			}

			// Compare Removed
			if tt.result.Removed != nil {
				if len(decoded.Removed) != len(tt.result.Removed) {
					t.Fatalf("Removed len = %d, want %d", len(decoded.Removed), len(tt.result.Removed))
				}
				for i, r := range tt.result.Removed {
					if decoded.Removed[i] != r {
						t.Errorf("Removed[%d] = %q, want %q", i, decoded.Removed[i], r)
					}
				}
			}

			// Compare Errors
			if tt.result.Errors != nil {
				if len(decoded.Errors) != len(tt.result.Errors) {
					t.Fatalf("Errors len = %d, want %d", len(decoded.Errors), len(tt.result.Errors))
				}
				for k, v := range tt.result.Errors {
					if decoded.Errors[k] != v {
						t.Errorf("Errors[%q] = %q, want %q", k, decoded.Errors[k], v)
					}
				}
			}
		})
	}
}

func TestMcpSetServersResult_WireFormatKeys(t *testing.T) {
	t.Parallel()
	result := McpSetServersResult{
		Added:   []string{"a"},
		Removed: []string{"b"},
		Errors:  map[string]string{"c": "err"},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	for _, key := range []string{"added", "removed", "errors"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected JSON key %q", key)
		}
	}
}

func FuzzMcpServerStatusInfo_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"name":"test","status":"connected"}`))
	f.Add([]byte(`{"name":"srv","status":"disconnected","error":"timeout","scope":"project","serverInfo":{"name":"S","version":"1.0"},"tools":[{"name":"t1","description":"d1"}]}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"name":"","status":"","tools":[]}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var info McpServerStatusInfo
		// Must not panic on any input
		_ = json.Unmarshal(data, &info)
	})
}
