package types

import (
	"encoding/json"
	"testing"
)

func TestSDKSessionInfo_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		info SDKSessionInfo
	}{
		{
			name: "all fields populated",
			info: SDKSessionInfo{
				SessionID:    "sess-abc-123",
				Summary:      "Fix authentication bug",
				LastModified: 1711036800,
				FileSize:     2048,
				CustomTitle:  "Auth Fix",
				FirstPrompt:  "Fix the login flow",
				GitBranch:    "fix/auth-bug",
				CWD:          "/home/user/project",
				Tag:          "v1.0",
				CreatedAt:    1711000000,
			},
		},
		{
			name: "only required fields",
			info: SDKSessionInfo{
				SessionID:    "sess-minimal",
				Summary:      "Minimal session",
				LastModified: 1711036800,
			},
		},
		{
			name: "empty session ID",
			info: SDKSessionInfo{
				SessionID:    "",
				Summary:      "",
				LastModified: 0,
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

			var decoded SDKSessionInfo
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.info.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.info.SessionID)
			}
			if decoded.Summary != tt.info.Summary {
				t.Errorf("Summary = %q, want %q", decoded.Summary, tt.info.Summary)
			}
			if decoded.LastModified != tt.info.LastModified {
				t.Errorf("LastModified = %d, want %d", decoded.LastModified, tt.info.LastModified)
			}
			if decoded.FileSize != tt.info.FileSize {
				t.Errorf("FileSize = %d, want %d", decoded.FileSize, tt.info.FileSize)
			}
			if decoded.CustomTitle != tt.info.CustomTitle {
				t.Errorf("CustomTitle = %q, want %q", decoded.CustomTitle, tt.info.CustomTitle)
			}
			if decoded.FirstPrompt != tt.info.FirstPrompt {
				t.Errorf("FirstPrompt = %q, want %q", decoded.FirstPrompt, tt.info.FirstPrompt)
			}
			if decoded.GitBranch != tt.info.GitBranch {
				t.Errorf("GitBranch = %q, want %q", decoded.GitBranch, tt.info.GitBranch)
			}
			if decoded.CWD != tt.info.CWD {
				t.Errorf("CWD = %q, want %q", decoded.CWD, tt.info.CWD)
			}
			if decoded.Tag != tt.info.Tag {
				t.Errorf("Tag = %q, want %q", decoded.Tag, tt.info.Tag)
			}
			if decoded.CreatedAt != tt.info.CreatedAt {
				t.Errorf("CreatedAt = %d, want %d", decoded.CreatedAt, tt.info.CreatedAt)
			}
		})
	}
}

func TestSDKSessionInfo_OmitEmpty(t *testing.T) {
	t.Parallel()
	info := SDKSessionInfo{
		SessionID:    "sess-001",
		Summary:      "Test",
		LastModified: 1711036800,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// Required fields must be present
	for _, key := range []string{"sessionId", "summary", "lastModified"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected JSON key %q to be present", key)
		}
	}

	// Optional fields should be omitted when zero
	for _, key := range []string{"fileSize", "customTitle", "firstPrompt", "gitBranch", "cwd", "tag", "createdAt"} {
		if _, ok := raw[key]; ok {
			t.Errorf("expected JSON key %q to be omitted when zero, but it was present", key)
		}
	}
}

func TestSDKSessionInfo_WireFormatKeys(t *testing.T) {
	t.Parallel()
	info := SDKSessionInfo{
		SessionID:    "sess-001",
		Summary:      "Test",
		LastModified: 1711036800,
		FileSize:     1024,
		CustomTitle:  "My Session",
		FirstPrompt:  "Hello",
		GitBranch:    "main",
		CWD:          "/tmp",
		Tag:          "v1",
		CreatedAt:    1711000000,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	expectedKeys := []string{
		"sessionId", "summary", "lastModified", "fileSize",
		"customTitle", "firstPrompt", "gitBranch", "cwd", "tag", "createdAt",
	}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected camelCase JSON key %q", key)
		}
	}
}

func TestSessionMessage_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	parentID := "toolu_parent_001"
	tests := []struct {
		name string
		msg  SessionMessage
	}{
		{
			name: "with parent tool use ID",
			msg: SessionMessage{
				Type:            "assistant",
				UUID:            "uuid-abc",
				SessionID:       "sess-001",
				Message:         json.RawMessage(`{"role":"assistant","content":[{"type":"text","text":"Hello"}]}`),
				ParentToolUseID: &parentID,
			},
		},
		{
			name: "without parent tool use ID",
			msg: SessionMessage{
				Type:      "user",
				UUID:      "uuid-def",
				SessionID: "sess-002",
				Message:   json.RawMessage(`{"role":"user","content":"Hi"}`),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.msg)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SessionMessage
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.Type != tt.msg.Type {
				t.Errorf("Type = %q, want %q", decoded.Type, tt.msg.Type)
			}
			if decoded.UUID != tt.msg.UUID {
				t.Errorf("UUID = %q, want %q", decoded.UUID, tt.msg.UUID)
			}
			if decoded.SessionID != tt.msg.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.msg.SessionID)
			}
			if string(decoded.Message) != string(tt.msg.Message) {
				t.Errorf("Message = %s, want %s", decoded.Message, tt.msg.Message)
			}

			if tt.msg.ParentToolUseID != nil {
				if decoded.ParentToolUseID == nil {
					t.Fatal("ParentToolUseID should not be nil")
				}
				if *decoded.ParentToolUseID != *tt.msg.ParentToolUseID {
					t.Errorf("ParentToolUseID = %q, want %q", *decoded.ParentToolUseID, *tt.msg.ParentToolUseID)
				}
			} else {
				if decoded.ParentToolUseID != nil {
					t.Errorf("ParentToolUseID should be nil, got %q", *decoded.ParentToolUseID)
				}
			}
		})
	}
}

func TestSessionMessage_OmitEmptyParentToolUseID(t *testing.T) {
	t.Parallel()
	msg := SessionMessage{
		Type:      "user",
		UUID:      "uuid-001",
		SessionID: "sess-001",
		Message:   json.RawMessage(`{}`),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["parentToolUseId"]; ok {
		t.Error("expected 'parentToolUseId' to be omitted when nil")
	}
}

func TestSessionMessage_WireFormatKeys(t *testing.T) {
	t.Parallel()
	parentID := "parent-001"
	msg := SessionMessage{
		Type:            "assistant",
		UUID:            "uuid-001",
		SessionID:       "sess-001",
		Message:         json.RawMessage(`{}`),
		ParentToolUseID: &parentID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	expectedKeys := []string{"type", "uuid", "sessionId", "message", "parentToolUseId"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected JSON key %q (camelCase wire format)", key)
		}
	}
}

func TestListSessionsOptions_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		opts ListSessionsOptions
	}{
		{
			name: "all fields set",
			opts: ListSessionsOptions{
				Dir:              "/home/user/projects",
				Limit:            50,
				Offset:           10,
				IncludeWorktrees: true,
			},
		},
		{
			name: "defaults only",
			opts: ListSessionsOptions{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.opts)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded ListSessionsOptions
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.Dir != tt.opts.Dir {
				t.Errorf("Dir = %q, want %q", decoded.Dir, tt.opts.Dir)
			}
			if decoded.Limit != tt.opts.Limit {
				t.Errorf("Limit = %d, want %d", decoded.Limit, tt.opts.Limit)
			}
			if decoded.Offset != tt.opts.Offset {
				t.Errorf("Offset = %d, want %d", decoded.Offset, tt.opts.Offset)
			}
			if decoded.IncludeWorktrees != tt.opts.IncludeWorktrees {
				t.Errorf("IncludeWorktrees = %v, want %v", decoded.IncludeWorktrees, tt.opts.IncludeWorktrees)
			}
		})
	}
}

func TestListSessionsOptions_OmitEmpty(t *testing.T) {
	t.Parallel()
	opts := ListSessionsOptions{}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	for _, key := range []string{"dir", "limit", "offset", "includeWorktrees"} {
		if _, ok := raw[key]; ok {
			t.Errorf("expected JSON key %q to be omitted when zero value", key)
		}
	}
}

func TestGetSessionMessagesOptions_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		opts GetSessionMessagesOptions
	}{
		{
			name: "all fields set",
			opts: GetSessionMessagesOptions{
				Dir:    "/home/user/projects",
				Limit:  100,
				Offset: 25,
			},
		},
		{
			name: "defaults only",
			opts: GetSessionMessagesOptions{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.opts)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded GetSessionMessagesOptions
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.Dir != tt.opts.Dir {
				t.Errorf("Dir = %q, want %q", decoded.Dir, tt.opts.Dir)
			}
			if decoded.Limit != tt.opts.Limit {
				t.Errorf("Limit = %d, want %d", decoded.Limit, tt.opts.Limit)
			}
			if decoded.Offset != tt.opts.Offset {
				t.Errorf("Offset = %d, want %d", decoded.Offset, tt.opts.Offset)
			}
		})
	}
}

func TestGetSessionMessagesOptions_OmitEmpty(t *testing.T) {
	t.Parallel()
	opts := GetSessionMessagesOptions{}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	for _, key := range []string{"dir", "limit", "offset", "includeSystemMessages"} {
		if _, ok := raw[key]; ok {
			t.Errorf("expected JSON key %q to be omitted when zero value", key)
		}
	}
}

func TestForkSessionResult_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		result ForkSessionResult
	}{
		{
			name: "all fields populated",
			result: ForkSessionResult{
				SessionID: "sess-fork-abc",
				Summary:   "Forked from original session",
			},
		},
		{
			name: "only session ID",
			result: ForkSessionResult{
				SessionID: "sess-fork-minimal",
			},
		},
		{
			name:   "empty",
			result: ForkSessionResult{},
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

			var decoded ForkSessionResult
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.SessionID != tt.result.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tt.result.SessionID)
			}
			if decoded.Summary != tt.result.Summary {
				t.Errorf("Summary = %q, want %q", decoded.Summary, tt.result.Summary)
			}
		})
	}
}

func TestForkSessionResult_WireFormatKeys(t *testing.T) {
	t.Parallel()
	result := ForkSessionResult{
		SessionID: "sess-001",
		Summary:   "Test fork",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	expectedKeys := []string{"sessionId"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected camelCase JSON key %q", key)
		}
	}
}

func TestSubagentInfo_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		agent SubagentInfo
	}{
		{
			name: "all fields populated",
			agent: SubagentInfo{
				ID:              "agent-abc-123",
				Name:            "code-reviewer",
				ParentSessionID: "sess-parent-001",
				Model:           "claude-sonnet-4-6",
			},
		},
		{
			name: "required fields only",
			agent: SubagentInfo{
				ID:              "agent-minimal",
				Name:            "helper",
				ParentSessionID: "sess-001",
			},
		},
		{
			name:  "empty",
			agent: SubagentInfo{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.agent)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SubagentInfo
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.ID != tt.agent.ID {
				t.Errorf("ID = %q, want %q", decoded.ID, tt.agent.ID)
			}
			if decoded.Name != tt.agent.Name {
				t.Errorf("Name = %q, want %q", decoded.Name, tt.agent.Name)
			}
			if decoded.ParentSessionID != tt.agent.ParentSessionID {
				t.Errorf("ParentSessionID = %q, want %q", decoded.ParentSessionID, tt.agent.ParentSessionID)
			}
			if decoded.Model != tt.agent.Model {
				t.Errorf("Model = %q, want %q", decoded.Model, tt.agent.Model)
			}
		})
	}
}

func TestSubagentInfo_WireFormatKeys(t *testing.T) {
	t.Parallel()
	agent := SubagentInfo{
		ID:              "agent-001",
		Name:            "reviewer",
		ParentSessionID: "sess-001",
		Model:           "claude-opus-4-6",
	}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	expectedKeys := []string{"id", "name", "parentSessionId"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected camelCase JSON key %q", key)
		}
	}
}

func TestGetSessionMessagesOptions_IncludeSystemMessages(t *testing.T) {
	t.Parallel()

	// Test that IncludeSystemMessages is included in JSON when true
	opts := GetSessionMessagesOptions{
		Dir:                   "/tmp/test",
		Limit:                 50,
		IncludeSystemMessages: true,
	}

	data, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["includeSystemMessages"]; !ok {
		t.Error("expected 'includeSystemMessages' key when set to true")
	}

	// Test omitted when false (default)
	optsDefault := GetSessionMessagesOptions{Dir: "/tmp"}
	data2, err := json.Marshal(optsDefault)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw2 map[string]interface{}
	if err := json.Unmarshal(data2, &raw2); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw2["includeSystemMessages"]; ok {
		t.Error("expected 'includeSystemMessages' to be omitted when false")
	}
}

func FuzzForkSessionResult_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"sessionId":"s1","summary":"test"}`))
	f.Add([]byte(`{"sessionId":"","summary":""}`))
	f.Add([]byte(`{}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var result ForkSessionResult
		_ = json.Unmarshal(data, &result)
	})
}

func FuzzSubagentInfo_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"id":"a1","name":"agent","parentSessionId":"s1","model":"claude"}`))
	f.Add([]byte(`{"id":"","name":"","parentSessionId":"","model":""}`))
	f.Add([]byte(`{}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var info SubagentInfo
		_ = json.Unmarshal(data, &info)
	})
}

func FuzzSessionMessage_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"type":"user","uuid":"abc","sessionId":"s1","message":{}}`))
	f.Add([]byte(`{"type":"assistant","uuid":"","sessionId":"","message":null}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"type":"user","uuid":"x","sessionId":"y","message":{"role":"user"},"parentToolUseId":"tool1"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var msg SessionMessage
		// Must not panic on any input
		_ = json.Unmarshal(data, &msg)
	})
}

func FuzzSDKSessionInfo_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"sessionId":"s1","summary":"test","lastModified":1711036800}`))
	f.Add([]byte(`{"sessionId":"","summary":"","lastModified":0}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"sessionId":"s1","summary":"test","lastModified":1711036800,"fileSize":2048,"customTitle":"t","firstPrompt":"p","gitBranch":"main","cwd":"/tmp","tag":"v1","createdAt":1711000000}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var info SDKSessionInfo
		// Must not panic on any input
		_ = json.Unmarshal(data, &info)
	})
}
