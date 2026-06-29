package internal

import (
	"context"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestMessageLoop_PanicRecovery verifies that a panic inside routeMessage
// (triggered by a nil message) is caught by the deferred recover(). After the
// panic, readLoopDone must close (so Stop() doesn't hang) and messagesChan
// must close (so ReceiveResponse consumers unblock).
func TestMessageLoop_PanicRecovery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "nil message causes panic — recovered gracefully"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			mt := newMockTransport()
			opts := types.NewClaudeAgentOptions()
			logger := log.NewLogger(false)
			query := NewQuery(ctx, mt, opts, logger, true)

			if err := query.Start(ctx); err != nil {
				t.Fatalf("Start failed: %v", err)
			}

			// Send a nil message — causes nil pointer dereference in routeMessage
			// when it calls msg.GetMessageType(). This triggers the panic recovery.
			mt.messagesChan <- nil

			// readLoopDone should close after the panic is recovered.
			select {
			case <-query.readLoopDone:
				// Good — panic was recovered and readLoopDone closed.
			case <-time.After(5 * time.Second):
				t.Fatal("readLoopDone not closed after panic — recovery failed")
			}

			// messagesChan should be closed so consumers don't block forever.
			select {
			case _, ok := <-query.messagesChan:
				if ok {
					t.Error("messagesChan should be closed after panic recovery")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("messagesChan not closed after panic recovery")
			}

			// Stop should complete without hanging.
			stopDone := make(chan error, 1)
			go func() {
				stopDone <- query.Stop(ctx)
			}()

			select {
			case err := <-stopDone:
				if err != nil {
					t.Fatalf("Stop() returned error: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("Stop() hung after panic recovery")
			}
		})
	}
}

// TestHandleControlRequest_PanicRecovery verifies that a panicking canUseTool
// callback is converted into an error response and that handler cleanup still
// completes so Stop() doesn't hang.
func TestHandleControlRequest_PanicRecovery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "panicking canUseTool callback — recovered gracefully"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			mt := newMockTransport()

			// Create a canUseTool callback that panics.
			opts := types.NewClaudeAgentOptions().WithCanUseTool(
				func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
					panic("intentional panic in canUseTool callback")
				},
			)

			logger := log.NewLogger(false)
			query := NewQuery(ctx, mt, opts, logger, true)

			if err := query.Start(ctx); err != nil {
				t.Fatalf("Start failed: %v", err)
			}

			// Send a control_request that triggers canUseTool → panic
			controlRequest := &types.SystemMessage{
				Type:      "control_request",
				Subtype:   "control_request",
				RequestID: "test-panic-req-1",
				Request: map[string]interface{}{
					"subtype":   "can_use_tool",
					"tool_name": "Bash",
					"input":     map[string]interface{}{"command": "ls"},
				},
			}

			mt.sendMessage(controlRequest)

			assertPermissionPanicErrorResponse(t, mt, "test-panic-req-1")

			// Stop should complete without hanging — handlerWg.Done() was
			// called even though the handler panicked.
			stopDone := make(chan error, 1)
			go func() {
				stopDone <- query.Stop(ctx)
			}()

			select {
			case err := <-stopDone:
				if err != nil {
					t.Fatalf("Stop() returned error: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("Stop() hung — handlerWg.Done() was not called after panic")
			}
		})
	}
}

func TestHandleControlRequest_CanUseTool_PanicSendsErrorResponse(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mt := newMockTransport()
	opts := types.NewClaudeAgentOptions().WithCanUseTool(
		func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
			panic("permission callback exploded")
		},
	)

	logger := log.NewLogger(false)
	query := NewQuery(ctx, mt, opts, logger, true)

	if err := query.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	mt.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		Subtype:   "control_request",
		RequestID: "test-panic-req-1",
		Request: map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "Bash",
			"input":     map[string]interface{}{"command": "ls"},
		},
	})

	assertPermissionPanicErrorResponse(t, mt, "test-panic-req-1")

	stopDone := make(chan error, 1)
	go func() {
		stopDone <- query.Stop(ctx)
	}()

	select {
	case err := <-stopDone:
		if err != nil {
			t.Fatalf("Stop() returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() hung after permission callback panic")
	}
}

// TestHandleControlResponse_NonBlockingSend verifies that sending a response to
// a channel whose receiver has already exited (context canceled) does not block.
func TestHandleControlResponse_NonBlockingSend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sendFunc func(query *Query, mt *mockTransport, requestID string)
	}{
		{
			name: "success response dropped when receiver timed out",
			sendFunc: func(query *Query, mt *mockTransport, requestID string) {
				// Send a success response
				controlResponse := &types.SystemMessage{
					Type:    "control_response",
					Subtype: "control_response",
					Response: map[string]interface{}{
						"subtype":    "success",
						"request_id": requestID,
						"response":   map[string]interface{}{"ok": true},
					},
				}
				mt.sendMessage(controlResponse)
			},
		},
		{
			name: "error response dropped when receiver timed out",
			sendFunc: func(query *Query, mt *mockTransport, requestID string) {
				// Send an error response
				controlResponse := &types.SystemMessage{
					Type:    "control_response",
					Subtype: "control_response",
					Response: map[string]interface{}{
						"subtype":    "error",
						"request_id": requestID,
						"error":      "some error",
					},
				}
				mt.sendMessage(controlResponse)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			mt := newMockTransport()
			opts := types.NewClaudeAgentOptions()
			logger := log.NewLogger(false)
			query := NewQuery(ctx, mt, opts, logger, true)

			if err := query.Start(ctx); err != nil {
				t.Fatalf("Start failed: %v", err)
			}
			defer func() {
				if err := query.Stop(ctx); err != nil {
					t.Logf("error stopping query: %v", err)
				}
			}()

			// Manually register a response channel (buffer 1) and then fill it,
			// simulating a scenario where the receiver already consumed or a
			// duplicate response arrives.
			requestID := "test-non-blocking-req"
			responseChan := make(chan responseResult, 1)
			// Pre-fill the buffer to simulate a duplicate
			responseChan <- responseResult{response: map[string]interface{}{"first": true}}

			query.mu.Lock()
			query.requestMap[requestID] = responseChan
			query.mu.Unlock()

			// Now send another response — this should hit the `default` branch
			// and not block. We wrap in a goroutine with a timeout to detect blocking.
			done := make(chan struct{})
			go func() {
				tt.sendFunc(query, mt, requestID)
				close(done)
			}()

			// Give time for the message to be routed
			select {
			case <-done:
				// Message was sent successfully
			case <-time.After(2 * time.Second):
				t.Fatal("sendFunc blocked — should have returned immediately")
			}

			// Give time for the response to be processed by the messageLoop
			time.Sleep(200 * time.Millisecond)

			// The response channel should still have just the first entry.
			// The duplicate was dropped via the default branch.
			select {
			case result := <-responseChan:
				first, _ := result.response["first"].(bool)
				if !first {
					t.Error("expected the original response, not the duplicate")
				}
			default:
				t.Error("expected a response in the channel")
			}
		})
	}
}

// TestStopBeforeStart verifies that Stop() returns promptly when Start() was
// never called, instead of blocking forever on readLoopDone.
func TestStopBeforeStart(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	transport := newMockTransport()
	opts := types.NewClaudeAgentOptions()
	logger := log.NewLogger(false)
	query := NewQuery(ctx, transport, opts, logger, true)

	// Do NOT call Start(). Call Stop() directly.
	done := make(chan error, 1)
	go func() {
		done <- query.Stop(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Stop() returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() hung when Start() was never called")
	}
}
