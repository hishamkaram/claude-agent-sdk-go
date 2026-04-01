package types

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// TestFormatResult_AllBranches tests the formatResult method on SDKMCPServer
// covering string, map-with-text, map-as-json, array, and default branches.
func TestFormatResult_AllBranches(t *testing.T) {
	t.Parallel()

	dummyHandler := func(ctx context.Context, args map[string]any) (any, error) {
		return nil, nil
	}

	server, err := NewSDKMCPServer("test",
		Tool{Name: "t", Description: "d", Handler: dummyHandler},
	)
	if err != nil {
		t.Fatalf("NewSDKMCPServer: %v", err)
	}

	tests := []struct {
		name        string
		result      any
		wantType    string
		wantTextSub string
		wantLen     int // expected number of content blocks
	}{
		{
			name:        "string result",
			result:      "hello world",
			wantType:    "text",
			wantTextSub: "hello world",
			wantLen:     1,
		},
		{
			name:        "map with text key",
			result:      map[string]interface{}{"text": "from text key"},
			wantType:    "text",
			wantTextSub: "from text key",
			wantLen:     1,
		},
		{
			name:        "map without text key as JSON",
			result:      map[string]interface{}{"count": 42},
			wantType:    "text",
			wantTextSub: "42",
			wantLen:     1,
		},
		{
			name: "array of content blocks",
			result: []interface{}{
				map[string]interface{}{"type": "text", "text": "block1"},
				map[string]interface{}{"type": "text", "text": "block2"},
			},
			wantLen: 2,
		},
		{
			name:    "array with non-map items are skipped",
			result:  []interface{}{"not a map", 42},
			wantLen: 0,
		},
		{
			name:        "integer default path",
			result:      42,
			wantType:    "text",
			wantTextSub: "42",
			wantLen:     1,
		},
		{
			name:        "boolean default path",
			result:      true,
			wantType:    "text",
			wantTextSub: "true",
			wantLen:     1,
		},
		{
			name:        "nil default path",
			result:      nil,
			wantType:    "text",
			wantTextSub: "null",
			wantLen:     1,
		},
		{
			name:        "float default path",
			result:      3.14,
			wantType:    "text",
			wantTextSub: "3.14",
			wantLen:     1,
		},
		{
			name:        "slice of ints (not []interface)",
			result:      []int{1, 2, 3},
			wantType:    "text",
			wantTextSub: "[1,2,3]",
			wantLen:     1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content := server.formatResult(tt.result)

			if len(content) != tt.wantLen {
				t.Fatalf("formatResult() returned %d blocks, want %d", len(content), tt.wantLen)
			}

			if tt.wantLen > 0 && tt.wantType != "" {
				gotType, _ := content[0]["type"].(string)
				if gotType != tt.wantType {
					t.Errorf("content[0][type] = %q, want %q", gotType, tt.wantType)
				}
			}

			if tt.wantLen > 0 && tt.wantTextSub != "" {
				gotText, _ := content[0]["text"].(string)
				if !strings.Contains(gotText, tt.wantTextSub) {
					t.Errorf("content[0][text] = %q, want substring %q", gotText, tt.wantTextSub)
				}
			}
		})
	}
}

// TestFormatAsJSON tests the package-level formatAsJSON function.
func TestFormatAsJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   interface{}
		wantSub string
	}{
		{
			name:    "string value",
			input:   "hello",
			wantSub: `"hello"`,
		},
		{
			name:    "integer value",
			input:   42,
			wantSub: "42",
		},
		{
			name:    "nil value",
			input:   nil,
			wantSub: "null",
		},
		{
			name:    "map value",
			input:   map[string]interface{}{"key": "val"},
			wantSub: `"key"`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatAsJSON(tt.input)
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("formatAsJSON(%v) = %q, want substring %q", tt.input, got, tt.wantSub)
			}
		})
	}
}

// TestFormatMapAsJSON tests the package-level formatMapAsJSON function.
func TestFormatMapAsJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   map[string]interface{}
		wantSub string
	}{
		{
			name:    "simple map",
			input:   map[string]interface{}{"status": "ok"},
			wantSub: `"status"`,
		},
		{
			name:    "empty map",
			input:   map[string]interface{}{},
			wantSub: "{}",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatMapAsJSON(tt.input)
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("formatMapAsJSON() = %q, want substring %q", got, tt.wantSub)
			}
		})
	}
}

// TestSDKMCPServer_HandleCallTool_HandlerError tests that a handler returning an error
// produces an error response.
func TestSDKMCPServer_HandleCallTool_HandlerError(t *testing.T) {
	t.Parallel()

	server, _ := NewSDKMCPServer("test",
		Tool{
			Name:        "fail",
			Description: "Always fails",
			Handler: func(ctx context.Context, args map[string]any) (any, error) {
				return nil, errors.New("tool broke")
			},
		},
	)

	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "fail",
			"arguments": map[string]interface{}{},
		},
	}

	response, err := server.HandleMessage(message)
	if err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}

	errObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error response")
	}

	errMsg, _ := errObj["message"].(string)
	if !strings.Contains(errMsg, "tool broke") {
		t.Errorf("error message = %q, want 'tool broke'", errMsg)
	}
}

// TestSDKMCPServer_HandleMessage_MissingMethod tests missing method field.
func TestSDKMCPServer_HandleMessage_MissingMethod(t *testing.T) {
	t.Parallel()

	server, _ := NewSDKMCPServer("test",
		Tool{
			Name:        "t",
			Description: "d",
			Handler:     func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
		},
	)

	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		// no "method" field
	}

	response, err := server.HandleMessage(message)
	if err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}

	if response["error"] == nil {
		t.Fatal("expected error response for missing method")
	}

	errObj := response["error"].(map[string]interface{})
	errMsg, _ := errObj["message"].(string)
	if !strings.Contains(errMsg, "Invalid Request") {
		t.Errorf("error message = %q, want 'Invalid Request'", errMsg)
	}
}

// TestSDKMCPServer_HandleCallTool_MissingToolName tests missing tool name in params.
func TestSDKMCPServer_HandleCallTool_MissingToolName(t *testing.T) {
	t.Parallel()

	server, _ := NewSDKMCPServer("test",
		Tool{
			Name:        "t",
			Description: "d",
			Handler:     func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
		},
	)

	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			// missing "name"
			"arguments": map[string]interface{}{},
		},
	}

	response, _ := server.HandleMessage(message)

	if response["error"] == nil {
		t.Fatal("expected error response for missing tool name")
	}

	errObj := response["error"].(map[string]interface{})
	errMsg, _ := errObj["message"].(string)
	if !strings.Contains(errMsg, "missing tool name") {
		t.Errorf("error message = %q, want 'missing tool name'", errMsg)
	}
}
