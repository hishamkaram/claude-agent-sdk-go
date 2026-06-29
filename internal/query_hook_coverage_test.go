package internal

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

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
