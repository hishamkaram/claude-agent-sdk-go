package types

import (
	"encoding/json"
	"testing"
)

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
