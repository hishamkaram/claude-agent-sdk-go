package internal

import (
	"strings"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestParseMessage_UserMessage tests parsing of user messages.
func TestParseMessage_UserMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		input       []byte
		wantErr     bool
		wantType    string
		checkResult func(t *testing.T, msg types.Message)
	}{
		{
			name:     "simple string content",
			input:    userMessageSimple,
			wantErr:  false,
			wantType: "user",
			checkResult: func(t *testing.T, msg types.Message) {
				userMsg, ok := msg.(*types.UserMessage)
				if !ok {
					t.Errorf("expected *types.UserMessage, got %T", msg)
					return
				}
				if userMsg.Type != "user" {
					t.Errorf("expected type 'user', got '%s'", userMsg.Type)
				}
				contentStr, ok := userMsg.Content.(string)
				if !ok {
					t.Errorf("expected content to be string, got %T", userMsg.Content)
					return
				}
				if contentStr != "Hello Claude" {
					t.Errorf("expected content 'Hello Claude', got '%s'", contentStr)
				}
			},
		},
		{
			name:     "content blocks with tool result",
			input:    userMessageComplex,
			wantErr:  false,
			wantType: "user",
			checkResult: func(t *testing.T, msg types.Message) {
				userMsg, ok := msg.(*types.UserMessage)
				if !ok {
					t.Errorf("expected *types.UserMessage, got %T", msg)
					return
				}
				blocks, ok := userMsg.Content.([]types.ContentBlock)
				if !ok {
					t.Errorf("expected content to be []ContentBlock, got %T", userMsg.Content)
					return
				}
				if len(blocks) != 2 {
					t.Errorf("expected 2 content blocks, got %d", len(blocks))
				}
				if userMsg.ParentToolUseID == nil || *userMsg.ParentToolUseID != "parent_456" {
					t.Errorf("expected parent_tool_use_id 'parent_456', got %v", userMsg.ParentToolUseID)
				}
			},
		},
		{
			name:     "only text content",
			input:    userMessageOnlyText,
			wantErr:  false,
			wantType: "user",
			checkResult: func(t *testing.T, msg types.Message) {
				userMsg, ok := msg.(*types.UserMessage)
				if !ok {
					t.Errorf("expected *types.UserMessage, got %T", msg)
					return
				}
				contentStr, ok := userMsg.Content.(string)
				if !ok {
					t.Errorf("expected content to be string, got %T", userMsg.Content)
					return
				}
				if contentStr != "Simple text content" {
					t.Errorf("expected content 'Simple text content', got '%s'", contentStr)
				}
			},
		},
		{
			name:     "content blocks array",
			input:    userMessageContentBlocks,
			wantErr:  false,
			wantType: "user",
			checkResult: func(t *testing.T, msg types.Message) {
				userMsg, ok := msg.(*types.UserMessage)
				if !ok {
					t.Errorf("expected *types.UserMessage, got %T", msg)
					return
				}
				blocks, ok := userMsg.Content.([]types.ContentBlock)
				if !ok {
					t.Errorf("expected content to be []ContentBlock, got %T", userMsg.Content)
					return
				}
				if len(blocks) != 2 {
					t.Errorf("expected 2 content blocks, got %d", len(blocks))
				}
			},
		},
		{
			name:     "extra fields ignored (forward compat)",
			input:    userMessageExtraFields,
			wantErr:  false,
			wantType: "user",
			checkResult: func(t *testing.T, msg types.Message) {
				userMsg, ok := msg.(*types.UserMessage)
				if !ok {
					t.Errorf("expected *types.UserMessage, got %T", msg)
					return
				}
				if userMsg.Type != "user" {
					t.Errorf("expected type 'user', got '%s'", userMsg.Type)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg, err := ParseMessage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if msg.GetMessageType() != tt.wantType {
					t.Errorf("expected message type %s, got %s", tt.wantType, msg.GetMessageType())
				}
				if tt.checkResult != nil {
					tt.checkResult(t, msg)
				}
			}
		})
	}
}

// TestParseMessage_AssistantMessage tests parsing of assistant messages.
func TestParseMessage_AssistantMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		input       []byte
		wantErr     bool
		wantType    string
		checkResult func(t *testing.T, msg types.Message)
	}{
		{
			name:     "simple text content",
			input:    assistantMessageText,
			wantErr:  false,
			wantType: "assistant",
			checkResult: func(t *testing.T, msg types.Message) {
				assistantMsg, ok := msg.(*types.AssistantMessage)
				if !ok {
					t.Errorf("expected *types.AssistantMessage, got %T", msg)
					return
				}
				if len(assistantMsg.Content) != 1 {
					t.Errorf("expected 1 content block, got %d", len(assistantMsg.Content))
					return
				}
				textBlock, ok := assistantMsg.Content[0].(*types.TextBlock)
				if !ok {
					t.Errorf("expected *types.TextBlock, got %T", assistantMsg.Content[0])
					return
				}
				if textBlock.Text != "Hi there! How can I help you today?" {
					t.Errorf("unexpected text content: %s", textBlock.Text)
				}
				if assistantMsg.Model != "claude-sonnet-4-5-20250929" {
					t.Errorf("expected model 'claude-sonnet-4-5-20250929', got '%s'", assistantMsg.Model)
				}
			},
		},
		{
			name:     "tool use content",
			input:    assistantMessageToolUse,
			wantErr:  false,
			wantType: "assistant",
			checkResult: func(t *testing.T, msg types.Message) {
				assistantMsg, ok := msg.(*types.AssistantMessage)
				if !ok {
					t.Errorf("expected *types.AssistantMessage, got %T", msg)
					return
				}
				if len(assistantMsg.Content) != 2 {
					t.Errorf("expected 2 content blocks, got %d", len(assistantMsg.Content))
					return
				}
				toolUseBlock, ok := assistantMsg.Content[1].(*types.ToolUseBlock)
				if !ok {
					t.Errorf("expected *types.ToolUseBlock, got %T", assistantMsg.Content[1])
					return
				}
				if toolUseBlock.Name != "calculator" {
					t.Errorf("expected tool name 'calculator', got '%s'", toolUseBlock.Name)
				}
			},
		},
		{
			name:     "thinking block content",
			input:    assistantMessageThinking,
			wantErr:  false,
			wantType: "assistant",
			checkResult: func(t *testing.T, msg types.Message) {
				assistantMsg, ok := msg.(*types.AssistantMessage)
				if !ok {
					t.Errorf("expected *types.AssistantMessage, got %T", msg)
					return
				}
				if len(assistantMsg.Content) < 1 {
					t.Errorf("expected at least 1 content block, got %d", len(assistantMsg.Content))
					return
				}
				thinkingBlock, ok := assistantMsg.Content[0].(*types.ThinkingBlock)
				if !ok {
					t.Errorf("expected *types.ThinkingBlock, got %T", assistantMsg.Content[0])
					return
				}
				if !strings.Contains(thinkingBlock.Thinking, "step by step") {
					t.Errorf("unexpected thinking content: %s", thinkingBlock.Thinking)
				}
			},
		},
		{
			name:     "mixed content blocks",
			input:    assistantMessageMixed,
			wantErr:  false,
			wantType: "assistant",
			checkResult: func(t *testing.T, msg types.Message) {
				assistantMsg, ok := msg.(*types.AssistantMessage)
				if !ok {
					t.Errorf("expected *types.AssistantMessage, got %T", msg)
					return
				}
				if len(assistantMsg.Content) != 3 {
					t.Errorf("expected 3 content blocks, got %d", len(assistantMsg.Content))
				}
			},
		},
		{
			name:     "all block types",
			input:    assistantMessageAllBlocks,
			wantErr:  false,
			wantType: "assistant",
			checkResult: func(t *testing.T, msg types.Message) {
				assistantMsg, ok := msg.(*types.AssistantMessage)
				if !ok {
					t.Errorf("expected *types.AssistantMessage, got %T", msg)
					return
				}
				if len(assistantMsg.Content) != 3 {
					t.Errorf("expected 3 content blocks, got %d", len(assistantMsg.Content))
				}
			},
		},
		{
			name:     "extra fields ignored (forward compat)",
			input:    assistantMessageExtraFields,
			wantErr:  false,
			wantType: "assistant",
			checkResult: func(t *testing.T, msg types.Message) {
				assistantMsg, ok := msg.(*types.AssistantMessage)
				if !ok {
					t.Errorf("expected *types.AssistantMessage, got %T", msg)
					return
				}
				if assistantMsg.Type != "assistant" {
					t.Errorf("expected type 'assistant', got '%s'", assistantMsg.Type)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg, err := ParseMessage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if msg.GetMessageType() != tt.wantType {
					t.Errorf("expected message type %s, got %s", tt.wantType, msg.GetMessageType())
				}
				if tt.checkResult != nil {
					tt.checkResult(t, msg)
				}
			}
		})
	}
}

// TestParseMessage_SystemMessage tests parsing of system messages.
func TestParseMessage_SystemMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		input       []byte
		wantErr     bool
		wantSubtype string
	}{
		{
			name:        "metadata system message",
			input:       systemMessageMetadata,
			wantErr:     false,
			wantSubtype: "metadata",
		},
		{
			name:        "warning system message",
			input:       systemMessageWarning,
			wantErr:     false,
			wantSubtype: "warning",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg, err := ParseMessage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				systemMsg, ok := msg.(*types.SystemMessage)
				if !ok {
					t.Errorf("expected *types.SystemMessage, got %T", msg)
					return
				}
				if systemMsg.Subtype != tt.wantSubtype {
					t.Errorf("expected subtype %s, got %s", tt.wantSubtype, systemMsg.Subtype)
				}
			}
		})
	}
}

// TestParseMessage_ResultMessage tests parsing of result messages.
func TestParseMessage_ResultMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    []byte
		wantErr  bool
		isError  bool
		hasUsage bool
	}{
		{
			name:     "success result",
			input:    resultMessageSuccess,
			wantErr:  false,
			isError:  false,
			hasUsage: true,
		},
		{
			name:     "error result",
			input:    resultMessageError,
			wantErr:  false,
			isError:  true,
			hasUsage: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg, err := ParseMessage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				resultMsg, ok := msg.(*types.ResultMessage)
				if !ok {
					t.Errorf("expected *types.ResultMessage, got %T", msg)
					return
				}
				if resultMsg.IsError != tt.isError {
					t.Errorf("expected is_error %v, got %v", tt.isError, resultMsg.IsError)
				}
				if tt.hasUsage && resultMsg.Usage == nil {
					t.Errorf("expected usage to be present")
				}
				if !tt.hasUsage && resultMsg.Usage != nil {
					t.Errorf("expected usage to be nil")
				}
			}
		})
	}
}

// TestParseMessage_StreamEvent tests parsing of stream events.
func TestParseMessage_StreamEvent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     []byte
		wantErr   bool
		eventType string
	}{
		{
			name:      "message_start event",
			input:     streamEventMessageStart,
			wantErr:   false,
			eventType: "message_start",
		},
		{
			name:      "content_block_delta event",
			input:     streamEventContentBlockDelta,
			wantErr:   false,
			eventType: "content_block_delta",
		},
		{
			name:      "message_delta event",
			input:     streamEventMessageDelta,
			wantErr:   false,
			eventType: "message_delta",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg, err := ParseMessage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				streamEvent, ok := msg.(*types.StreamEvent)
				if !ok {
					t.Errorf("expected *types.StreamEvent, got %T", msg)
					return
				}
				if streamEvent.Event == nil {
					t.Errorf("expected event to be present")
					return
				}
				evtType, ok := streamEvent.Event["type"].(string)
				if !ok || evtType != tt.eventType {
					t.Errorf("expected event type %s, got %v", tt.eventType, evtType)
				}
			}
		})
	}
}
