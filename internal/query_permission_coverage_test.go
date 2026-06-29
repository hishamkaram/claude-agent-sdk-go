package internal

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

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
