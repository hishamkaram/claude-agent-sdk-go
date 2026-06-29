package types

import "encoding/json"

// ContentBlock is an interface for all content block types.
// Content blocks can be text, thinking, tool use, or tool result blocks.
type ContentBlock interface {
	GetType() string
	isContentBlock()
}

// TextBlock represents a text content block from Claude.
type TextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// GetType returns the type of the content block.
func (t *TextBlock) GetType() string {
	return t.Type
}

func (t *TextBlock) isContentBlock() {}

// ThinkingBlock represents a thinking content block from Claude.
// This contains Claude's internal reasoning and signature.
type ThinkingBlock struct {
	Type      string `json:"type"`
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

// GetType returns the type of the content block.
func (t *ThinkingBlock) GetType() string {
	return t.Type
}

func (t *ThinkingBlock) isContentBlock() {}

// ToolUseBlock represents a tool use request from Claude.
type ToolUseBlock struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// GetType returns the type of the content block.
func (t *ToolUseBlock) GetType() string {
	return t.Type
}

func (t *ToolUseBlock) isContentBlock() {}

// ToolResultBlock represents the result of a tool execution.
type ToolResultBlock struct {
	Type      string      `json:"type"`
	ToolUseID string      `json:"tool_use_id"`
	Content   interface{} `json:"content,omitempty"`  // Can be string or []map[string]interface{}
	IsError   *bool       `json:"is_error,omitempty"` // Pointer to distinguish between false and not set
}

// GetType returns the type of the content block.
func (t *ToolResultBlock) GetType() string {
	return t.Type
}

func (t *ToolResultBlock) isContentBlock() {}

// UnmarshalContentBlock unmarshals a JSON content block into the appropriate type.
func UnmarshalContentBlock(data []byte) (ContentBlock, error) {
	var typeCheck struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, NewJSONDecodeErrorWithCause("failed to determine content block type", string(data), err)
	}

	switch typeCheck.Type {
	case "text":
		var block TextBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal text block", string(data), err)
		}
		return &block, nil
	case "thinking":
		var block ThinkingBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal thinking block", string(data), err)
		}
		return &block, nil
	case "tool_use":
		var block ToolUseBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal tool_use block", string(data), err)
		}
		return &block, nil
	case "tool_result":
		var block ToolResultBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil, NewJSONDecodeErrorWithCause("failed to unmarshal tool_result block", string(data), err)
		}
		return &block, nil
	default:
		return nil, NewMessageParseErrorWithType("unknown content block type", typeCheck.Type)
	}
}
