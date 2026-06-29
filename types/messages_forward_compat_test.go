package types

import (
	"testing"
)

func TestUnmarshalUnknownMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		json        string
		checkResult func(t *testing.T, msg Message)
	}{
		{
			name: "unknown type returns UnknownMessage",
			json: `{"type":"future_feature_xyz","data":{"key":"value"}}`,
			checkResult: func(t *testing.T, msg Message) {
				m, ok := msg.(*UnknownMessage)
				if !ok {
					t.Fatalf("expected *UnknownMessage, got %T", msg)
				}
				if m.Type != "future_feature_xyz" {
					t.Errorf("Type = %q, want %q", m.Type, "future_feature_xyz")
				}
				if len(m.RawJSON) == 0 {
					t.Error("RawJSON should be populated")
				}
			},
		},
		{
			name: "ShouldDisplayToUser returns false",
			json: `{"type":"something_new"}`,
			checkResult: func(t *testing.T, msg Message) {
				if msg.ShouldDisplayToUser() {
					t.Error("ShouldDisplayToUser() should return false")
				}
			},
		},
		{
			name: "GetMessageType returns unknown type string",
			json: `{"type":"abc_123"}`,
			checkResult: func(t *testing.T, msg Message) {
				if msg.GetMessageType() != "abc_123" {
					t.Errorf("GetMessageType() = %q, want %q", msg.GetMessageType(), "abc_123")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			msg, err := UnmarshalMessage([]byte(tt.json))
			if err != nil {
				t.Fatalf("UnmarshalMessage() should not error on unknown types, got %v", err)
			}
			if tt.checkResult != nil {
				tt.checkResult(t, msg)
			}
		})
	}
}

func TestUnmarshalMessageNoErrorOnUnknownType(t *testing.T) {
	t.Parallel()
	msg, err := UnmarshalMessage([]byte(`{"type":"never_seen_before","x":1}`))
	if err != nil {
		t.Fatalf("expected no error for unknown type, got %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
}

// ---------------------------------------------------------------------------
// Tests for empty/missing type still errors
// ---------------------------------------------------------------------------

func TestUnmarshalMessageMissingType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		json string
	}{
		{name: "missing type", json: `{"content":"hello"}`},
		{name: "null type", json: `{"type":null}`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := UnmarshalMessage([]byte(tt.json))
			if err == nil {
				t.Error("expected error for missing/null type")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Backward compatibility: existing subtypes still return *SystemMessage
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Tests for AgentInfo and RewindFilesResult (017-sdk-client-methods Phase 2)
// ---------------------------------------------------------------------------
