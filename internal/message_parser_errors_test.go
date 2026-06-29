package internal

import (
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestParseMessage_InvalidJSON tests error handling for invalid JSON.
func TestParseMessage_InvalidJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "malformed JSON",
			input:   invalidJSONMalformed,
			wantErr: true,
		},
		{
			name:    "missing type field",
			input:   invalidJSONMissingType,
			wantErr: true,
		},
		{
			name:    "unknown message type returns UnknownMessage",
			input:   invalidJSONUnknownType,
			wantErr: false,
		},
		{
			name:    "empty bytes",
			input:   invalidJSONEmptyBytes,
			wantErr: true,
		},
		{
			name:    "null type field",
			input:   invalidJSONNullType,
			wantErr: true,
		},
		{
			name:    "number type field",
			input:   invalidJSONNumberType,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg, err := ParseMessage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				// Verify error is one of the expected types
				if !types.IsMessageParseError(err) && !types.IsJSONDecodeError(err) {
					t.Errorf("expected MessageParseError or JSONDecodeError, got %T", err)
				}
			}
			if err == nil && msg != nil {
				// For unknown type, verify it returns *UnknownMessage
				if _, ok := msg.(*types.UnknownMessage); ok {
					if msg.GetMessageType() != "unknown_message_type" {
						t.Errorf("expected type 'unknown_message_type', got %s", msg.GetMessageType())
					}
				}
			}
		})
	}
}

// TestParseMessage_UnknownTypeReturnsUnknownMessage tests forward compatibility.
func TestParseMessage_UnknownTypeReturnsUnknownMessage(t *testing.T) {
	t.Parallel()
	msg, err := ParseMessage(unknownTypeMessage)
	if err != nil {
		t.Fatalf("ParseMessage() unexpected error: %v", err)
	}
	unknownMsg, ok := msg.(*types.UnknownMessage)
	if !ok {
		t.Fatalf("expected *types.UnknownMessage, got %T", msg)
	}
	if unknownMsg.Type != "future_feature_xyz" {
		t.Errorf("expected type 'future_feature_xyz', got %s", unknownMsg.Type)
	}
	if unknownMsg.RawJSON == nil {
		t.Error("expected RawJSON to be populated")
	}
	if unknownMsg.ShouldDisplayToUser() {
		t.Error("expected ShouldDisplayToUser() to return false")
	}
}

// TestExtractType tests the type extraction helper.
func TestExtractType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    []byte
		wantType string
		wantErr  bool
	}{
		{
			name:     "valid user type",
			input:    []byte(`{"type": "user", "content": "test"}`),
			wantType: "user",
			wantErr:  false,
		},
		{
			name:     "valid assistant type",
			input:    []byte(`{"type": "assistant", "content": []}`),
			wantType: "assistant",
			wantErr:  false,
		},
		{
			name:     "missing type",
			input:    []byte(`{"content": "test"}`),
			wantType: "",
			wantErr:  true,
		},
		{
			name:     "invalid JSON",
			input:    []byte(`{invalid`),
			wantType: "",
			wantErr:  true,
		},
		{
			name:     "type is number",
			input:    []byte(`{"type": 123}`),
			wantType: "",
			wantErr:  true,
		},
		{
			name:     "type is null",
			input:    []byte(`{"type": null}`),
			wantType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotType, err := extractType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotType != tt.wantType {
				t.Errorf("extractType() = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

// TestTruncateString tests the string truncation helper.
func TestTruncateString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "no truncation needed",
			input:  "short",
			maxLen: 10,
			want:   "short",
		},
		{
			name:   "exact length",
			input:  "exact",
			maxLen: 5,
			want:   "exact",
		},
		{
			name:   "truncation needed",
			input:  "this is a very long string",
			maxLen: 10,
			want:   "this is a ...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString() = %v, want %v", got, tt.want)
			}
		})
	}
}
