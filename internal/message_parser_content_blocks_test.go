package internal

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestParseContentBlock_TextBlock tests parsing text blocks.
func TestParseContentBlock_TextBlock(t *testing.T) {
	t.Parallel()
	block, err := ParseContentBlock(textBlockJSON)
	if err != nil {
		t.Fatalf("ParseContentBlock() error = %v", err)
	}

	textBlock, ok := block.(*types.TextBlock)
	if !ok {
		t.Fatalf("expected *types.TextBlock, got %T", block)
	}

	if textBlock.Type != "text" {
		t.Errorf("expected type 'text', got '%s'", textBlock.Type)
	}
	if textBlock.Text != "This is a text block" {
		t.Errorf("unexpected text: %s", textBlock.Text)
	}
}

// TestParseContentBlock_ToolUseBlock tests parsing tool use blocks.
func TestParseContentBlock_ToolUseBlock(t *testing.T) {
	t.Parallel()
	block, err := ParseContentBlock(toolUseBlockJSON)
	if err != nil {
		t.Fatalf("ParseContentBlock() error = %v", err)
	}

	toolUseBlock, ok := block.(*types.ToolUseBlock)
	if !ok {
		t.Fatalf("expected *types.ToolUseBlock, got %T", block)
	}

	if toolUseBlock.Type != "tool_use" {
		t.Errorf("expected type 'tool_use', got '%s'", toolUseBlock.Type)
	}
	if toolUseBlock.Name != "calculator" {
		t.Errorf("expected name 'calculator', got '%s'", toolUseBlock.Name)
	}
	if toolUseBlock.ID != "toolu_block_123" {
		t.Errorf("expected id 'toolu_block_123', got '%s'", toolUseBlock.ID)
	}
}

// TestParseContentBlock_ToolResultBlock tests parsing tool result blocks.
func TestParseContentBlock_ToolResultBlock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		input       []byte
		wantErr     bool
		wantIsError *bool
	}{
		{
			name:        "simple result",
			input:       toolResultBlockJSON,
			wantErr:     false,
			wantIsError: nil,
		},
		{
			name:        "error result",
			input:       toolResultBlockJSONWithError,
			wantErr:     false,
			wantIsError: boolPtr(true),
		},
		{
			name:        "complex content",
			input:       toolResultBlockJSONComplex,
			wantErr:     false,
			wantIsError: boolPtr(false),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			block, err := ParseContentBlock(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseContentBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				toolResultBlock, ok := block.(*types.ToolResultBlock)
				if !ok {
					t.Errorf("expected *types.ToolResultBlock, got %T", block)
					return
				}
				if toolResultBlock.Type != "tool_result" {
					t.Errorf("expected type 'tool_result', got '%s'", toolResultBlock.Type)
				}
				if tt.wantIsError != nil {
					if toolResultBlock.IsError == nil {
						t.Errorf("expected is_error to be %v, got nil", *tt.wantIsError)
					} else if *toolResultBlock.IsError != *tt.wantIsError {
						t.Errorf("expected is_error %v, got %v", *tt.wantIsError, *toolResultBlock.IsError)
					}
				}
			}
		})
	}
}

// TestParseContentBlock_ThinkingBlock tests parsing thinking blocks.
func TestParseContentBlock_ThinkingBlock(t *testing.T) {
	t.Parallel()
	block, err := ParseContentBlock(thinkingBlockJSON)
	if err != nil {
		t.Fatalf("ParseContentBlock() error = %v", err)
	}

	thinkingBlock, ok := block.(*types.ThinkingBlock)
	if !ok {
		t.Fatalf("expected *types.ThinkingBlock, got %T", block)
	}

	if thinkingBlock.Type != "thinking" {
		t.Errorf("expected type 'thinking', got '%s'", thinkingBlock.Type)
	}
	if !strings.Contains(thinkingBlock.Thinking, "think about this") {
		t.Errorf("unexpected thinking content: %s", thinkingBlock.Thinking)
	}
}

// TestParseContentBlock_InvalidBlocks tests error handling for invalid content blocks.
func TestParseContentBlock_InvalidBlocks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "empty bytes",
			input:   []byte{},
			wantErr: true,
		},
		{
			name:    "missing type field",
			input:   contentBlockMissingType,
			wantErr: true,
		},
		{
			name:    "unknown type",
			input:   contentBlockUnknownType,
			wantErr: true,
		},
		{
			name:    "malformed JSON",
			input:   contentBlockMalformed,
			wantErr: true,
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

// TestParseContentBlocks_Multiple tests parsing multiple content blocks.
func TestParseContentBlocks_Multiple(t *testing.T) {
	t.Parallel()
	rawBlocks := make([]json.RawMessage, len(multipleContentBlocks))
	for i, block := range multipleContentBlocks {
		rawBlocks[i] = json.RawMessage(block)
	}

	blocks, err := ParseContentBlocks(rawBlocks)
	if err != nil {
		t.Fatalf("ParseContentBlocks() error = %v", err)
	}

	if len(blocks) != 3 {
		t.Errorf("expected 3 blocks, got %d", len(blocks))
	}

	// Verify types
	if _, ok := blocks[0].(*types.TextBlock); !ok {
		t.Errorf("expected block 0 to be *types.TextBlock, got %T", blocks[0])
	}
	if _, ok := blocks[1].(*types.ThinkingBlock); !ok {
		t.Errorf("expected block 1 to be *types.ThinkingBlock, got %T", blocks[1])
	}
	if _, ok := blocks[2].(*types.ToolUseBlock); !ok {
		t.Errorf("expected block 2 to be *types.ToolUseBlock, got %T", blocks[2])
	}
}

// TestParseContentBlocks_Empty tests parsing empty content blocks array.
func TestParseContentBlocks_Empty(t *testing.T) {
	t.Parallel()
	blocks, err := ParseContentBlocks([]json.RawMessage{})
	if err != nil {
		t.Fatalf("ParseContentBlocks() error = %v", err)
	}

	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks, got %d", len(blocks))
	}
}

// TestParseContentBlocks_WithError tests error handling in batch parsing.
func TestParseContentBlocks_WithError(t *testing.T) {
	t.Parallel()
	rawBlocks := []json.RawMessage{
		json.RawMessage(`{"type": "text", "text": "Valid"}`),
		json.RawMessage(`{"type": "invalid_type"}`),
		json.RawMessage(`{"type": "text", "text": "Also valid"}`),
	}

	_, err := ParseContentBlocks(rawBlocks)
	if err == nil {
		t.Error("expected error for invalid block, got nil")
	}

	// Error should mention the index
	if !strings.Contains(err.Error(), "index 1") {
		t.Errorf("error should mention index 1, got: %v", err)
	}
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}
