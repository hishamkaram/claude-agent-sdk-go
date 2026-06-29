package types

import (
	"encoding/json"
	"testing"
)

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
