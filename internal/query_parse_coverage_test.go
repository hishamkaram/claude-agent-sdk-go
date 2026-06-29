package internal

import (
	"encoding/json"
	"testing"
)

func TestMatchesToolName(t *testing.T) {
	t.Parallel()

	bashPattern := "Bash"
	writeEditPattern := "Write|Edit"
	prefixPattern := "^Bash"
	emptyPattern := ""
	invalidPattern := "[invalid"

	tests := []struct {
		name     string
		toolName string
		pattern  *string
		want     bool
	}{
		{
			name:     "nil pattern matches all",
			toolName: "Bash",
			pattern:  nil,
			want:     true,
		},
		{
			name:     "empty pattern matches all",
			toolName: "Bash",
			pattern:  &emptyPattern,
			want:     true,
		},
		{
			name:     "exact match",
			toolName: "Bash",
			pattern:  &bashPattern,
			want:     true,
		},
		{
			name:     "exact no match",
			toolName: "Write",
			pattern:  &bashPattern,
			want:     false,
		},
		{
			name:     "alternation match first",
			toolName: "Write",
			pattern:  &writeEditPattern,
			want:     true,
		},
		{
			name:     "alternation match second",
			toolName: "Edit",
			pattern:  &writeEditPattern,
			want:     true,
		},
		{
			name:     "alternation no match",
			toolName: "Bash",
			pattern:  &writeEditPattern,
			want:     false,
		},
		{
			name:     "prefix pattern match",
			toolName: "BashTool",
			pattern:  &prefixPattern,
			want:     true,
		},
		{
			name:     "prefix pattern no match",
			toolName: "NotBash",
			pattern:  &prefixPattern,
			want:     false,
		},
		{
			name:     "invalid regex returns false",
			toolName: "anything",
			pattern:  &invalidPattern,
			want:     false,
		},
		{
			name:     "substring match (Bash in LongBash)",
			toolName: "LongBash",
			pattern:  &bashPattern,
			want:     true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := matchesToolName(tt.toolName, tt.pattern)
			if got != tt.want {
				t.Errorf("matchesToolName(%q, %v) = %v, want %v", tt.toolName, tt.pattern, got, tt.want)
			}
		})
	}
}

// TestTruncateString_AllBranches tests the truncateString helper function for all boundary conditions.

func TestTruncateString_AllBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "shorter than max",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exactly max",
			input:  "1234567890",
			maxLen: 10,
			want:   "1234567890",
		},
		{
			name:   "longer than max",
			input:  "12345678901",
			maxLen: 10,
			want:   "1234567890...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 5,
			want:   "",
		},
		{
			name:   "max zero truncates to ellipsis",
			input:  "abc",
			maxLen: 0,
			want:   "...",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// TestExtractType_AllBranches tests the extractType helper function for all branches.

func TestExtractType_AllBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "valid type field",
			input: `{"type":"user"}`,
			want:  "user",
		},
		{
			name:  "type among other fields",
			input: `{"type":"assistant","model":"claude-3"}`,
			want:  "assistant",
		},
		{
			name:    "missing type field",
			input:   `{"data":"value"}`,
			wantErr: true,
		},
		{
			name:    "type is not string",
			input:   `{"type":42}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   `{broken`,
			wantErr: true,
		},
		{
			name:    "empty JSON",
			input:   `{}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := extractType([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("extractType() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("extractType() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestParseMessage_Errors tests error paths in ParseMessage.

func TestParseMessage_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "empty data",
			input:   []byte{},
			wantErr: true,
		},
		{
			name:    "nil data",
			input:   nil,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   []byte(`{broken`),
			wantErr: true,
		},
		{
			name:    "valid user message",
			input:   []byte(`{"type":"user","content":"hi"}`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseMessage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestParseContentBlock_Errors tests error paths in ParseContentBlock.

func TestParseContentBlock_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "empty data",
			input:   []byte{},
			wantErr: true,
		},
		{
			name:    "nil data",
			input:   nil,
			wantErr: true,
		},
		{
			name:    "valid text block",
			input:   []byte(`{"type":"text","text":"hello"}`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseContentBlock(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseContentBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSendErrorResponse tests sendErrorResponse via handleControlRequest with nil request.

func TestParseHelpers(t *testing.T) {
	t.Parallel()

	t.Run("parseUserMessage/valid", func(t *testing.T) {
		t.Parallel()
		data := []byte(`{"type":"user","content":"hello"}`)
		msg, err := parseUserMessage(data)
		if err != nil {
			t.Fatalf("parseUserMessage() error = %v", err)
		}
		if msg.Type != "user" {
			t.Errorf("Type = %q, want %q", msg.Type, "user")
		}
	})

	t.Run("parseUserMessage/invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseUserMessage([]byte(`{broken`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("parseAssistantMessage/valid", func(t *testing.T) {
		t.Parallel()
		data := []byte(`{"type":"assistant","content":[{"type":"text","text":"hi"}]}`)
		msg, err := parseAssistantMessage(data)
		if err != nil {
			t.Fatalf("parseAssistantMessage() error = %v", err)
		}
		if msg.Type != "assistant" {
			t.Errorf("Type = %q, want %q", msg.Type, "assistant")
		}
	})

	t.Run("parseAssistantMessage/invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseAssistantMessage([]byte(`{broken`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("parseSystemMessage/valid", func(t *testing.T) {
		t.Parallel()
		data := []byte(`{"type":"system","subtype":"info"}`)
		msg, err := parseSystemMessage(data)
		if err != nil {
			t.Fatalf("parseSystemMessage() error = %v", err)
		}
		if msg.Subtype != "info" {
			t.Errorf("Subtype = %q, want %q", msg.Subtype, "info")
		}
	})

	t.Run("parseSystemMessage/invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseSystemMessage([]byte(`{broken`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("parseResultMessage/valid", func(t *testing.T) {
		t.Parallel()
		data := []byte(`{"type":"result","subtype":"success","duration_ms":100,"session_id":"s1"}`)
		msg, err := parseResultMessage(data)
		if err != nil {
			t.Fatalf("parseResultMessage() error = %v", err)
		}
		if msg.Subtype != "success" {
			t.Errorf("Subtype = %q, want %q", msg.Subtype, "success")
		}
	})

	t.Run("parseResultMessage/invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseResultMessage([]byte(`{broken`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("parseStreamEvent/valid", func(t *testing.T) {
		t.Parallel()
		data := []byte(`{"type":"stream_event","uuid":"u1","session_id":"s1","event":{}}`)
		msg, err := parseStreamEvent(data)
		if err != nil {
			t.Fatalf("parseStreamEvent() error = %v", err)
		}
		if msg.UUID != "u1" {
			t.Errorf("UUID = %q, want %q", msg.UUID, "u1")
		}
	})

	t.Run("parseStreamEvent/invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseStreamEvent([]byte(`{broken`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("parseTextBlock/valid", func(t *testing.T) {
		t.Parallel()
		data := []byte(`{"type":"text","text":"hello"}`)
		block, err := parseTextBlock(data)
		if err != nil {
			t.Fatalf("parseTextBlock() error = %v", err)
		}
		if block.Text != "hello" {
			t.Errorf("Text = %q, want %q", block.Text, "hello")
		}
	})

	t.Run("parseTextBlock/invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseTextBlock([]byte(`{broken`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("parseToolUseBlock/valid", func(t *testing.T) {
		t.Parallel()
		data := []byte(`{"type":"tool_use","id":"tu1","name":"Bash","input":{}}`)
		block, err := parseToolUseBlock(data)
		if err != nil {
			t.Fatalf("parseToolUseBlock() error = %v", err)
		}
		if block.Name != "Bash" {
			t.Errorf("Name = %q, want %q", block.Name, "Bash")
		}
	})

	t.Run("parseToolUseBlock/invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseToolUseBlock([]byte(`{broken`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("parseToolResultBlock/valid", func(t *testing.T) {
		t.Parallel()
		data := []byte(`{"type":"tool_result","tool_use_id":"tu1"}`)
		block, err := parseToolResultBlock(data)
		if err != nil {
			t.Fatalf("parseToolResultBlock() error = %v", err)
		}
		if block.ToolUseID != "tu1" {
			t.Errorf("ToolUseID = %q, want %q", block.ToolUseID, "tu1")
		}
	})

	t.Run("parseToolResultBlock/invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseToolResultBlock([]byte(`{broken`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("parseThinkingBlock/valid", func(t *testing.T) {
		t.Parallel()
		data := []byte(`{"type":"thinking","thinking":"internal","signature":"sig"}`)
		block, err := parseThinkingBlock(data)
		if err != nil {
			t.Fatalf("parseThinkingBlock() error = %v", err)
		}
		if block.Thinking != "internal" {
			t.Errorf("Thinking = %q, want %q", block.Thinking, "internal")
		}
	})

	t.Run("parseThinkingBlock/invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseThinkingBlock([]byte(`{broken`))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// TestParseContentBlocks tests ParseContentBlocks with various inputs.

func TestParseContentBlocks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{
			name:    "empty array",
			input:   "[]",
			wantLen: 0,
		},
		{
			name:    "single text block",
			input:   `[{"type":"text","text":"hello"}]`,
			wantLen: 1,
		},
		{
			name:    "multiple blocks",
			input:   `[{"type":"text","text":"a"},{"type":"text","text":"b"}]`,
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var blocks []json.RawMessage
			if err := json.Unmarshal([]byte(tt.input), &blocks); err != nil {
				t.Fatalf("failed to unmarshal test input: %v", err)
			}
			result, err := ParseContentBlocks(blocks)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseContentBlocks() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(result) != tt.wantLen {
				t.Errorf("ParseContentBlocks() len = %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}
