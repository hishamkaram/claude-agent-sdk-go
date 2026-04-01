package internal

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestMatchesToolName tests the matchesToolName function for various patterns.
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
func TestSendErrorResponse_NilRequest(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	_ = newTestQuery(t, transport)

	// Send a control_request with nil Request to trigger sendErrorResponse
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "test-req-1",
		Request:   nil, // This triggers the nil requestData path
	})

	// Wait briefly for the handler to process
	time.Sleep(100 * time.Millisecond)

	// Check that an error response was written
	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "invalid control request format") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error response for nil request, written data: %v", written)
	}
}

// TestHandleControlRequest_UnsupportedSubtype tests unknown subtype handling.
func TestHandleControlRequest_UnsupportedSubtype(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	_ = newTestQuery(t, transport)

	// Send a control_request with unknown subtype
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "test-req-2",
		Request: map[string]interface{}{
			"subtype": "unknown_subtype",
		},
	})

	time.Sleep(100 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "unsupported control request subtype") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error response for unknown subtype, written data: %v", written)
	}
}

// TestHandleControlRequest_InterruptSubtype tests interrupt request handling.
func TestHandleControlRequest_InterruptSubtype(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	_ = newTestQuery(t, transport)

	// Send interrupt request
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "test-req-3",
		Request: map[string]interface{}{
			"subtype": "interrupt",
		},
	})

	time.Sleep(100 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "success") && strings.Contains(data, "test-req-3") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected success response for interrupt, written data: %v", written)
	}
}

// TestHandleControlRequest_SetPermissionModeSubtype tests set_permission_mode request handling.
func TestHandleControlRequest_SetPermissionModeSubtype(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	_ = newTestQuery(t, transport)

	// Send set_permission_mode request
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "test-req-4",
		Request: map[string]interface{}{
			"subtype": "set_permission_mode",
			"mode":    "plan",
		},
	})

	time.Sleep(100 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "success") && strings.Contains(data, "test-req-4") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected success response for set_permission_mode, written data: %v", written)
	}
}

// TestHandleControlRequest_NoRequestID tests auto-generation of request ID.
func TestHandleControlRequest_NoRequestID(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	_ = newTestQuery(t, transport)

	// Send a control_request without request_id
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "", // empty request ID
		Request: map[string]interface{}{
			"subtype": "interrupt",
		},
	})

	time.Sleep(100 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "cli-request-") && strings.Contains(data, "success") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected success response with auto-generated request ID, written data: %v", written)
	}
}

// TestHandleControlRequest_CanUseTool_NoCallback tests permission request without callback.
func TestHandleControlRequest_CanUseTool_NoCallback(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	_ = newTestQuery(t, transport)

	// Send a can_use_tool request without setting up a canUseTool callback
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "test-req-5",
		Request: map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "Bash",
			"input":     map[string]interface{}{"command": "ls"},
		},
	})

	time.Sleep(100 * time.Millisecond)

	// Without a callback, it should send a success response with allow behavior
	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "test-req-5") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected response for can_use_tool, written data: %v", written)
	}
}

// TestSendSuccessResponse tests sendSuccessResponse directly.
func TestSendSuccessResponse_WritesJSON(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	q := newTestQuery(t, transport)

	q.sendSuccessResponse("req-123", map[string]interface{}{
		"status": "ok",
	})

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "req-123") && strings.Contains(data, "success") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected success response with req-123, written data: %v", written)
	}
}

// TestSendErrorResponse_WritesJSON tests sendErrorResponse directly.
func TestSendErrorResponse_WritesJSON(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	q := newTestQuery(t, transport)

	q.sendErrorResponse("req-456", "something went wrong")

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "req-456") && strings.Contains(data, "something went wrong") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error response with req-456, written data: %v", written)
	}
}

// TestHandleControlRequest_CanUseTool_WithCallback tests permission request with a callback.
func TestHandleControlRequest_CanUseTool_WithCallback(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	// Set up options with a canUseTool callback
	opts := types.NewClaudeAgentOptions().
		WithCanUseTool(func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
			return &types.PermissionResultAllow{
				Behavior: "allow",
			}, nil
		})

	logger := log.NewLogger(false)
	q := NewQuery(ctx, transport, opts, logger, true)
	if err := q.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { _ = q.Stop(ctx) })

	// Send a can_use_tool request
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "perm-req-1",
		Request: map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "Bash",
			"input":     map[string]interface{}{"command": "ls"},
		},
	})

	time.Sleep(200 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "perm-req-1") && strings.Contains(data, "success") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected success response for can_use_tool with callback, written: %v", written)
	}
}

// TestHandleControlRequest_CanUseTool_InvalidToolName tests non-string tool_name.
func TestHandleControlRequest_CanUseTool_InvalidToolName(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	opts := types.NewClaudeAgentOptions().
		WithCanUseTool(func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
			return &types.PermissionResultAllow{Behavior: "allow"}, nil
		})

	logger := log.NewLogger(false)
	q := NewQuery(ctx, transport, opts, logger, true)
	if err := q.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { _ = q.Stop(ctx) })

	// Send a can_use_tool with non-string tool_name
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "perm-req-2",
		Request: map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": 42, // not a string
			"input":     map[string]interface{}{},
		},
	})

	time.Sleep(200 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "perm-req-2") && strings.Contains(data, "error") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error response for invalid tool_name type, written: %v", written)
	}
}

// TestHandleControlRequest_CanUseTool_MissingToolName tests empty tool_name.
func TestHandleControlRequest_CanUseTool_MissingToolName(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	opts := types.NewClaudeAgentOptions().
		WithCanUseTool(func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
			return &types.PermissionResultAllow{Behavior: "allow"}, nil
		})

	logger := log.NewLogger(false)
	q := NewQuery(ctx, transport, opts, logger, true)
	if err := q.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { _ = q.Stop(ctx) })

	// Send a can_use_tool with empty tool_name
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "perm-req-3",
		Request: map[string]interface{}{
			"subtype": "can_use_tool",
			// tool_name missing
			"input": map[string]interface{}{},
		},
	})

	time.Sleep(200 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "perm-req-3") && strings.Contains(data, "missing tool_name") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error response for missing tool_name, written: %v", written)
	}
}

// TestHandleControlRequest_CanUseTool_NilInput tests permission request with nil input.
func TestHandleControlRequest_CanUseTool_NilInput(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	opts := types.NewClaudeAgentOptions().
		WithCanUseTool(func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
			// Should receive empty map, not nil
			if input == nil {
				t.Error("input should be normalized to empty map, not nil")
			}
			return &types.PermissionResultAllow{Behavior: "allow"}, nil
		})

	logger := log.NewLogger(false)
	q := NewQuery(ctx, transport, opts, logger, true)
	if err := q.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { _ = q.Stop(ctx) })

	// Send a can_use_tool with nil input (like ExitPlanMode)
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "perm-req-4",
		Request: map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "ExitPlanMode",
			// no "input" key
		},
	})

	time.Sleep(200 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "perm-req-4") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected response for can_use_tool with nil input, written: %v", written)
	}
}

// TestHandleControlRequest_CanUseTool_InvalidInput tests non-map input.
func TestHandleControlRequest_CanUseTool_InvalidInput(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	opts := types.NewClaudeAgentOptions().
		WithCanUseTool(func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
			return &types.PermissionResultAllow{Behavior: "allow"}, nil
		})

	logger := log.NewLogger(false)
	q := NewQuery(ctx, transport, opts, logger, true)
	if err := q.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { _ = q.Stop(ctx) })

	// Send a can_use_tool with non-map input
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "perm-req-5",
		Request: map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "Bash",
			"input":     "not a map",
		},
	})

	time.Sleep(200 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "perm-req-5") && strings.Contains(data, "input must be a map") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error response for invalid input type, written: %v", written)
	}
}

// TestHandleControlRequest_CanUseTool_WithSuggestions tests permission with suggestions.
func TestHandleControlRequest_CanUseTool_WithSuggestions(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	opts := types.NewClaudeAgentOptions().
		WithCanUseTool(func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
			return &types.PermissionResultAllow{Behavior: "allow"}, nil
		})

	logger := log.NewLogger(false)
	q := NewQuery(ctx, transport, opts, logger, true)
	if err := q.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { _ = q.Stop(ctx) })

	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "perm-req-6",
		Request: map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "Bash",
			"input":     map[string]interface{}{"command": "ls"},
			"permission_suggestions": []interface{}{
				map[string]interface{}{
					"type": "addRules",
					"rules": []interface{}{
						map[string]interface{}{
							"toolName": "Bash",
						},
					},
					"behavior": "allow",
				},
			},
		},
	})

	time.Sleep(200 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "perm-req-6") && strings.Contains(data, "success") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected success response with suggestions, written: %v", written)
	}
}

// failingMockTransport is a mock transport whose Write always fails.
type failingMockTransport struct {
	mockTransport
}

func newFailingMockTransport() *failingMockTransport {
	return &failingMockTransport{
		mockTransport: mockTransport{
			messagesChan: make(chan types.Message, 100),
			writtenData:  make([]string, 0),
			ready:        true,
		},
	}
}

func (f *failingMockTransport) Write(ctx context.Context, data string) error {
	return errors.New("write failed")
}

// TestSendSuccessResponse_WriteError tests that write errors in sendSuccessResponse are logged.
func TestSendSuccessResponse_WriteError(t *testing.T) {
	t.Parallel()

	transport := newFailingMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	logger := log.NewLogger(false)
	q := NewQuery(ctx, &transport.mockTransport, nil, logger, true)

	// Override transport to the failing one after Query captures the interface
	q.transport = transport

	if err := q.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { _ = q.Stop(ctx) })

	// This should not panic, just log the write error
	q.sendSuccessResponse("req-fail", map[string]interface{}{"status": "ok"})
}

// TestSendErrorResponse_WriteError tests that write errors in sendErrorResponse are logged.
func TestSendErrorResponse_WriteError(t *testing.T) {
	t.Parallel()

	transport := newFailingMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	logger := log.NewLogger(false)
	q := NewQuery(ctx, &transport.mockTransport, nil, logger, true)

	// Override transport to the failing one after Query captures the interface
	q.transport = transport

	if err := q.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { _ = q.Stop(ctx) })

	// This should not panic, just log the write error
	q.sendErrorResponse("req-fail", "something went wrong")
}

// TestHandleControlRequest_HookCallback tests hook_callback subtype.
func TestHandleControlRequest_HookCallback(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	hookCalledCh := make(chan struct{}, 1)
	hookCallback := func(ctx context.Context, input interface{}, toolUseID *string, hookCtx types.HookContext) (interface{}, error) {
		select {
		case hookCalledCh <- struct{}{}:
		default:
		}
		return map[string]interface{}{"continue": true}, nil
	}

	opts := types.NewClaudeAgentOptions()
	logger := log.NewLogger(false)
	q := NewQuery(ctx, transport, opts, logger, true)

	// Register a hook callback manually
	callbackID := q.registerHookCallback(hookCallback)

	if err := q.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { _ = q.Stop(ctx) })

	// Send a hook_callback control request
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "hook-req-1",
		Request: map[string]interface{}{
			"subtype":     "hook_callback",
			"callback_id": callbackID,
			"input":       map[string]interface{}{"tool_name": "Bash"},
		},
	})

	select {
	case <-hookCalledCh:
		// OK — hook was called
	case <-time.After(2 * time.Second):
		t.Error("hook callback was not invoked within 2s")
	}

	time.Sleep(100 * time.Millisecond) // Give time for response write

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "hook-req-1") && strings.Contains(data, "success") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected success response for hook_callback, written: %v", written)
	}
}

// TestHandleControlRequest_CanUseTool_InvalidSuggestions tests non-array suggestions.
func TestHandleControlRequest_CanUseTool_InvalidSuggestions(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	opts := types.NewClaudeAgentOptions().
		WithCanUseTool(func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
			return &types.PermissionResultAllow{Behavior: "allow"}, nil
		})

	logger := log.NewLogger(false)
	q := NewQuery(ctx, transport, opts, logger, true)
	if err := q.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { _ = q.Stop(ctx) })

	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "perm-req-7",
		Request: map[string]interface{}{
			"subtype":                "can_use_tool",
			"tool_name":              "Bash",
			"input":                  map[string]interface{}{},
			"permission_suggestions": "not an array",
		},
	})

	time.Sleep(200 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "perm-req-7") && strings.Contains(data, "permission_suggestions must be an array") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error response for invalid suggestions type, written: %v", written)
	}
}

// TestSendControlRequest_NonStreaming tests sendControlRequest returns error in non-streaming mode.
func TestSendControlRequest_NonStreaming(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	logger := log.NewLogger(false)
	// Create query in NON-streaming mode
	q := NewQuery(ctx, transport, nil, logger, false)
	if err := q.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { _ = q.Stop(ctx) })

	_, err := q.sendControlRequest(ctx, map[string]interface{}{"subtype": "test"})
	if err == nil {
		t.Fatal("expected error for non-streaming mode")
	}
	if !types.IsControlProtocolError(err) {
		t.Errorf("expected ControlProtocolError, got %T: %v", err, err)
	}
}

// TestSendControlRequest_WriteError tests sendControlRequest with a failing transport.
func TestSendControlRequest_WriteError(t *testing.T) {
	t.Parallel()

	transport := newFailingMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	logger := log.NewLogger(false)
	q := NewQuery(ctx, &transport.mockTransport, nil, logger, true)
	q.transport = transport

	if err := q.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { _ = q.Stop(ctx) })

	_, err := q.sendControlRequest(ctx, map[string]interface{}{"subtype": "test"})
	if err == nil {
		t.Fatal("expected error when write fails")
	}
	if !types.IsControlProtocolError(err) {
		t.Errorf("expected ControlProtocolError, got %T: %v", err, err)
	}
}

// TestHandleControlRequest_HookCallback_UnknownCallbackID tests unknown callback.
func TestHandleControlRequest_HookCallback_UnknownCallbackID(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	opts := types.NewClaudeAgentOptions()
	logger := log.NewLogger(false)
	q := NewQuery(ctx, transport, opts, logger, true)
	if err := q.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { _ = q.Stop(ctx) })

	// Send a hook_callback with unknown callback_id
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "hook-req-unknown",
		Request: map[string]interface{}{
			"subtype":     "hook_callback",
			"callback_id": "nonexistent-callback",
			"input":       map[string]interface{}{},
		},
	})

	time.Sleep(200 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "hook-req-unknown") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected response for unknown hook callback, written: %v", written)
	}
}

// TestParseHelpers tests the individual type-specific parse helper functions.
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
