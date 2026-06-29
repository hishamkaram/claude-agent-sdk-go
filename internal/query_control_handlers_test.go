package internal

import (
	"context"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestHandlePermissionRequest tests permission callback handling.
func TestHandlePermissionRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		requestData    map[string]interface{}
		callbackResult interface{}
		callbackError  error
		expectedError  bool
		expectedResult map[string]interface{}
	}{
		{
			name: "allow permission",
			requestData: map[string]interface{}{
				"subtype":   "can_use_tool",
				"tool_name": "Bash",
				"input":     map[string]interface{}{"command": "ls"},
			},
			callbackResult: types.PermissionResultAllow{
				Behavior: "allow",
			},
			expectedResult: map[string]interface{}{
				"behavior":     "allow",
				"updatedInput": map[string]interface{}{"command": "ls"},
			},
		},
		{
			name: "deny permission",
			requestData: map[string]interface{}{
				"subtype":   "can_use_tool",
				"tool_name": "Write",
				"input":     map[string]interface{}{"file_path": "/etc/passwd"},
			},
			callbackResult: types.PermissionResultDeny{
				Behavior: "deny",
				Message:  "Access denied",
			},
			expectedResult: map[string]interface{}{
				"behavior": "deny",
				"message":  "Access denied",
			},
		},
		{
			name: "allow with updated input",
			requestData: map[string]interface{}{
				"subtype":   "can_use_tool",
				"tool_name": "Write",
				"input":     map[string]interface{}{"file_path": "/tmp/test.txt"},
			},
			callbackResult: types.PermissionResultAllow{
				Behavior: "allow",
				UpdatedInput: &map[string]interface{}{
					"file_path": "/tmp/sanitized.txt",
				},
			},
			expectedResult: map[string]interface{}{
				"behavior": "allow",
				"updatedInput": map[string]interface{}{
					"file_path": "/tmp/sanitized.txt",
				},
			},
		},
		{
			// ExitPlanMode sends input: null from the Claude CLI.
			// The SDK must normalize nil to {} and call CanUseTool rather than
			// returning an error. This was the root cause of the plan approval
			// prompt silently disappearing.
			name: "nil input normalized to empty map — ExitPlanMode",
			requestData: map[string]interface{}{
				"subtype":   "can_use_tool",
				"tool_name": "ExitPlanMode",
				"input":     nil,
			},
			callbackResult: types.PermissionResultAllow{
				Behavior: "allow",
			},
			expectedResult: map[string]interface{}{
				"behavior":     "allow",
				"updatedInput": map[string]interface{}{},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			transport := newMockTransport()

			opts := types.NewClaudeAgentOptions().WithCanUseTool(
				func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
					if tt.callbackError != nil {
						return nil, tt.callbackError
					}
					return tt.callbackResult, nil
				},
			)

			logger := log.NewLogger(false) // Non-verbose for tests
			query := NewQuery(ctx, transport, opts, logger, true)

			result, err := query.handlePermissionRequest(ctx, tt.requestData)
			if tt.expectedError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.expectedResult != nil {
				// Check behavior
				if behavior, ok := result["behavior"].(string); ok {
					if expectedBehavior, ok := tt.expectedResult["behavior"].(string); ok {
						if behavior != expectedBehavior {
							t.Errorf("behavior mismatch: got %s, want %s", behavior, expectedBehavior)
						}
					}
				}

				// Check message for deny
				if message, ok := result["message"].(string); ok {
					if expectedMessage, ok := tt.expectedResult["message"].(string); ok {
						if message != expectedMessage {
							t.Errorf("message mismatch: got %s, want %s", message, expectedMessage)
						}
					}
				}
			}
		})
	}
}

// TestHandleHookCallback tests hook callback handling.
func TestHandleHookCallback(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	transport := newMockTransport()

	hookCalled := false
	hookOutput := map[string]interface{}{
		"continue":       true,
		"suppressOutput": false,
	}

	opts := types.NewClaudeAgentOptions()
	logger := log.NewLogger(false) // Non-verbose for tests
	query := NewQuery(ctx, transport, opts, logger, true)

	// Register a hook callback
	callback := func(ctx context.Context, input interface{}, toolUseID *string, hookCtx types.HookContext) (interface{}, error) {
		hookCalled = true
		return hookOutput, nil
	}

	callbackID := query.registerHookCallback(callback)

	// Create hook callback request
	requestData := map[string]interface{}{
		"subtype":     "hook_callback",
		"callback_id": callbackID,
		"input": map[string]interface{}{
			"tool_name":  "Bash",
			"tool_input": map[string]interface{}{"command": "echo test"},
		},
	}

	result, err := query.handleHookCallback(requestData)
	if err != nil {
		t.Fatalf("handleHookCallback failed: %v", err)
	}

	if !hookCalled {
		t.Error("hook callback was not called")
	}

	if continueVal, ok := result["continue"].(bool); !ok || !continueVal {
		t.Error("expected continue to be true")
	}
}

// TestHandleMCPMessage tests MCP message routing.
func TestHandleMCPMessage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	transport := newMockTransport()
	opts := types.NewClaudeAgentOptions()

	logger := log.NewLogger(false) // Non-verbose for tests
	query := NewQuery(ctx, transport, opts, logger, true)

	// Add a mock MCP server
	mockServer := &mockMCPServer{
		name:    "test-server",
		version: "1.0.0",
	}
	query.AddMCPServer("test-server", mockServer)

	// Test successful MCP message
	requestData := map[string]interface{}{
		"subtype":     "mcp_message",
		"server_name": "test-server",
		"message": map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "test/method",
			"params":  map[string]interface{}{},
		},
	}

	result, err := query.handleMCPMessage(requestData)
	if err != nil {
		t.Fatalf("handleMCPMessage failed: %v", err)
	}

	mcpResponse, ok := result["mcp_response"].(map[string]interface{})
	if !ok {
		t.Fatal("expected mcp_response in result")
	}

	if mcpResponse["jsonrpc"] != "2.0" {
		t.Error("expected jsonrpc 2.0")
	}

	// Test server not found
	requestData["server_name"] = "nonexistent"
	result, err = query.handleMCPMessage(requestData)
	if err != nil {
		t.Fatalf("handleMCPMessage failed: %v", err)
	}

	mcpResponse, ok = result["mcp_response"].(map[string]interface{})
	if !ok {
		t.Fatal("expected mcp_response in result")
	}

	errorData, ok := mcpResponse["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected error in mcp_response")
	}

	if code, ok := errorData["code"].(int); !ok || code != -32601 {
		t.Errorf("expected error code -32601, got %v", code)
	}
}

// TestRequestResponseCorrelation tests request-response pairing.
