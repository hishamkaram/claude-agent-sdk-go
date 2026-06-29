package internal

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestCallbackTimeouts tests timeout handling for callbacks.
func TestCallbackTimeouts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	transport := newMockTransport()

	// Create a callback that times out
	opts := types.NewClaudeAgentOptions().WithCanUseTool(
		func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
			// Simulate slow callback
			select {
			case <-time.After(5 * time.Second):
				return types.PermissionResultAllow{Behavior: "allow"}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	)

	logger := log.NewLogger(false) // Non-verbose for tests
	query := NewQuery(ctx, transport, opts, logger, true)

	requestData := map[string]interface{}{
		"subtype":   "can_use_tool",
		"tool_name": "Bash",
		"input":     map[string]interface{}{"command": "ls"},
	}

	// Use a short timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	// This should timeout: the 100ms deadline is carried by the parent context
	// passed to handlePermissionRequest (the callback timeout defaults to 5m,
	// so the parent's shorter deadline is what fires).
	_, err := query.handlePermissionRequest(timeoutCtx, requestData)
	if err == nil {
		t.Error("expected timeout error")
	}
}

// mockMCPServer implements a mock MCP server for testing.
type mockMCPServer struct {
	name    string
	version string
}

func (m *mockMCPServer) HandleMessage(message map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      message["id"],
		"result":  map[string]interface{}{},
	}, nil
}

func (m *mockMCPServer) Name() string {
	return m.name
}

func (m *mockMCPServer) Version() string {
	return m.version
}

// TestQuery_CanUseTool_Timeout verifies that the canUseTool callback is called with
// a context that has a timeout derived from ToolCallbackTimeout, and that a slow
// callback is canceled when the timeout expires.
func TestQuery_CanUseTool_Timeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		callbackDelay time.Duration
		configTimeout time.Duration
		expectTimeout bool
	}{
		{
			name:          "callback completes before timeout",
			callbackDelay: 10 * time.Millisecond,
			configTimeout: 5 * time.Second,
			expectTimeout: false,
		},
		{
			name:          "callback exceeds configured timeout",
			callbackDelay: 5 * time.Second,
			configTimeout: 50 * time.Millisecond,
			expectTimeout: true,
		},
		{
			name:          "zero timeout uses default 5m — callback completes",
			callbackDelay: 10 * time.Millisecond,
			configTimeout: 0, // should use default
			expectTimeout: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			transport := newMockTransport()

			opts := types.NewClaudeAgentOptions()
			opts.ToolCallbackTimeout = tt.configTimeout
			opts.CanUseTool = func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
				select {
				case <-time.After(tt.callbackDelay):
					return types.PermissionResultAllow{Behavior: "allow"}, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}

			logger := log.NewLogger(false)
			query := NewQuery(ctx, transport, opts, logger, true)

			requestData := map[string]interface{}{
				"subtype":   "can_use_tool",
				"tool_name": "Bash",
				"input":     map[string]interface{}{"command": "ls"},
			}

			result, err := query.handlePermissionRequest(ctx, requestData)

			if tt.expectTimeout {
				if err == nil {
					t.Fatal("expected timeout error but got nil")
				}
				// The error should be a context deadline exceeded
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Errorf("expected context.DeadlineExceeded, got: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result == nil {
					t.Fatal("expected non-nil result")
				}
				if result["behavior"] != "allow" {
					t.Errorf("expected behavior 'allow', got %v", result["behavior"])
				}
			}
		})
	}
}

// TestQuery_HandlePermissionRequest_TypeAssertionOk verifies that handlePermissionRequest
// returns an error when request data fields have incorrect types instead of silently
// using zero values.
func TestQuery_HandlePermissionRequest_TypeAssertionOk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		requestData map[string]interface{}
		expectError bool
		errContains string
	}{
		{
			name: "tool_name is not a string",
			requestData: map[string]interface{}{
				"subtype":   "can_use_tool",
				"tool_name": 12345,
				"input":     map[string]interface{}{"command": "ls"},
			},
			expectError: true,
			errContains: "tool_name",
		},
		{
			name: "input is not a map",
			requestData: map[string]interface{}{
				"subtype":   "can_use_tool",
				"tool_name": "Bash",
				"input":     "not-a-map",
			},
			expectError: true,
			errContains: "input",
		},
		{
			name: "permission_suggestions is not a slice",
			requestData: map[string]interface{}{
				"subtype":                "can_use_tool",
				"tool_name":              "Bash",
				"input":                  map[string]interface{}{"command": "ls"},
				"permission_suggestions": "not-a-slice",
			},
			expectError: true,
			errContains: "permission_suggestions",
		},
		{
			name: "valid request — all types correct",
			requestData: map[string]interface{}{
				"subtype":   "can_use_tool",
				"tool_name": "Bash",
				"input":     map[string]interface{}{"command": "ls"},
			},
			expectError: false,
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
					return types.PermissionResultAllow{Behavior: "allow"}, nil
				},
			)

			logger := log.NewLogger(false)
			query := NewQuery(ctx, transport, opts, logger, true)

			_, err := query.handlePermissionRequest(ctx, tt.requestData)
			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestQuery_HandleMCPMessage_TypeAssertionOk verifies that handleMCPMessage
// returns an error when request data fields have incorrect types.
func TestQuery_HandleMCPMessage_TypeAssertionOk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		requestData map[string]interface{}
		expectError bool
	}{
		{
			name: "server_name is not a string",
			requestData: map[string]interface{}{
				"subtype":     "mcp_message",
				"server_name": 12345,
				"message": map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      1,
				},
			},
			expectError: true,
		},
		{
			name: "message is not a map",
			requestData: map[string]interface{}{
				"subtype":     "mcp_message",
				"server_name": "test-server",
				"message":     "not-a-map",
			},
			expectError: true,
		},
		{
			name: "both server_name and message have wrong types",
			requestData: map[string]interface{}{
				"subtype":     "mcp_message",
				"server_name": 42,
				"message":     42,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			transport := newMockTransport()
			opts := types.NewClaudeAgentOptions()

			logger := log.NewLogger(false)
			query := NewQuery(ctx, transport, opts, logger, true)

			_, err := query.handleMCPMessage(tt.requestData)
			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestInitialize_PromptSuggestionsAndJsonSchema tests that Initialize includes
// promptSuggestions and jsonSchema in the init control request when configured.
func TestInitialize_PromptSuggestionsAndJsonSchema(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		promptSuggestions     bool
		outputFormat          *types.OutputFormat
		wantPromptSuggestions bool
		wantJsonSchema        bool
	}{
		{
			name:                  "promptSuggestions enabled",
			promptSuggestions:     true,
			outputFormat:          nil,
			wantPromptSuggestions: true,
			wantJsonSchema:        false,
		},
		{
			name:              "jsonSchema from OutputFormat with schema",
			promptSuggestions: false,
			outputFormat: &types.OutputFormat{
				Type: "json_schema",
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"answer": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
			wantPromptSuggestions: false,
			wantJsonSchema:        true,
		},
		{
			name:              "both promptSuggestions and jsonSchema",
			promptSuggestions: true,
			outputFormat: &types.OutputFormat{
				Type: "json_schema",
				Schema: map[string]interface{}{
					"type": "object",
				},
			},
			wantPromptSuggestions: true,
			wantJsonSchema:        true,
		},
		{
			name:                  "neither set — baseline behavior",
			promptSuggestions:     false,
			outputFormat:          nil,
			wantPromptSuggestions: false,
			wantJsonSchema:        false,
		},
		{
			name:              "outputFormat without schema — no jsonSchema in init",
			promptSuggestions: false,
			outputFormat: &types.OutputFormat{
				Type:   "json_schema",
				Schema: nil,
			},
			wantPromptSuggestions: false,
			wantJsonSchema:        false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			transport := newMockTransport()

			opts := types.NewClaudeAgentOptions()
			if tt.promptSuggestions {
				opts.WithPromptSuggestions(true)
			}
			if tt.outputFormat != nil {
				opts.WithOutputFormat(*tt.outputFormat)
			}

			logger := log.NewLogger(false)
			query := NewQuery(ctx, transport, opts, logger, true)

			if err := query.Start(ctx); err != nil {
				t.Fatalf("Start failed: %v", err)
			}
			defer func() {
				if err := query.Stop(ctx); err != nil {
					t.Logf("error stopping query: %v", err)
				}
			}()

			// Goroutine to respond to the initialize request and capture its payload.
			requestCaptured := make(chan map[string]interface{}, 1)
			go func() {
				time.Sleep(50 * time.Millisecond)
				written := transport.getWrittenData()

				for _, data := range written {
					var sentRequest map[string]interface{}
					if err := json.Unmarshal([]byte(data), &sentRequest); err != nil {
						continue
					}

					reqType, _ := sentRequest["type"].(string)
					if reqType != "control_request" {
						continue
					}

					requestID, _ := sentRequest["request_id"].(string)
					request, _ := sentRequest["request"].(map[string]interface{})
					subtype, _ := request["subtype"].(string)

					if subtype == "initialize" {
						// Capture the inner request for assertions
						requestCaptured <- request

						// Send success response
						controlResponse := &types.SystemMessage{
							Type:    "control_response",
							Subtype: "control_response",
							Response: map[string]interface{}{
								"subtype":    "success",
								"request_id": requestID,
								"response": map[string]interface{}{
									"capabilities": []string{"hooks", "permissions"},
								},
							},
						}
						transport.sendMessage(controlResponse)
						return
					}
				}
			}()

			// Initialize
			result, err := query.Initialize(ctx)
			if err != nil {
				t.Fatalf("Initialize failed: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result from Initialize")
			}

			// Get the captured request
			select {
			case req := <-requestCaptured:
				// Check promptSuggestions
				ps, psExists := req["promptSuggestions"]
				if tt.wantPromptSuggestions {
					if !psExists {
						t.Error("expected promptSuggestions in init request, but not found")
					} else if ps != true {
						t.Errorf("promptSuggestions = %v, want true", ps)
					}
				} else {
					if psExists {
						t.Errorf("promptSuggestions should not be present in init request, but got %v", ps)
					}
				}

				// Check jsonSchema
				js, jsExists := req["jsonSchema"]
				if tt.wantJsonSchema {
					if !jsExists {
						t.Error("expected jsonSchema in init request, but not found")
					}
					// Verify it is a map (the schema object)
					if _, ok := js.(map[string]interface{}); !ok {
						t.Errorf("jsonSchema should be a map, got %T", js)
					}
				} else {
					if jsExists {
						t.Errorf("jsonSchema should not be present in init request, but got %v", js)
					}
				}

			case <-time.After(2 * time.Second):
				t.Fatal("timeout waiting for captured init request")
			}
		})
	}
}
