package internal

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

type blockingReadTransport struct {
	messages chan types.Message
	release  chan struct{}
	once     sync.Once
}

func newBlockingReadTransport() *blockingReadTransport {
	return &blockingReadTransport{
		messages: make(chan types.Message),
		release:  make(chan struct{}),
	}
}

func (t *blockingReadTransport) Connect(ctx context.Context) error { return nil }
func (t *blockingReadTransport) Close(ctx context.Context) error {
	t.releaseRead()
	return nil
}
func (t *blockingReadTransport) Write(ctx context.Context, data string) error { return nil }
func (t *blockingReadTransport) ReadMessages(ctx context.Context) <-chan types.Message {
	<-t.release
	return t.messages
}
func (t *blockingReadTransport) OnError(err error) {}
func (t *blockingReadTransport) IsReady() bool     { return true }
func (t *blockingReadTransport) GetError() error   { return nil }
func (t *blockingReadTransport) releaseRead()      { t.once.Do(func() { close(t.release) }) }

// TestNewQuery tests Query construction.
func TestNewQuery(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	transport := newMockTransport()
	opts := types.NewClaudeAgentOptions()

	logger := log.NewLogger(false) // Non-verbose for tests
	query := NewQuery(ctx, transport, opts, logger, true)

	if query.transport != transport {
		t.Error("transport not set correctly")
	}
	if !query.isStreamingMode {
		t.Error("expected streaming mode to be true")
	}
	if query.requestMap == nil {
		t.Error("requestMap not initialized")
	}
	if query.hookCallbacks == nil {
		t.Error("hookCallbacks not initialized")
	}
}

// TestInitialize tests Query initialization with hooks.
func TestInitialize(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	transport := newMockTransport()

	// Create hook callback
	hookCallback := func(ctx context.Context, input interface{}, toolUseID *string, hookCtx types.HookContext) (interface{}, error) {
		return map[string]interface{}{
			"continue": true,
		}, nil
	}

	// Create options with hooks
	bashMatcher := "Bash"
	opts := types.NewClaudeAgentOptions().WithHook(
		types.HookEventPreToolUse,
		types.HookMatcher{
			Matcher: &bashMatcher,
			Hooks:   []types.HookCallbackFunc{hookCallback},
		},
	)

	logger := log.NewLogger(false) // Non-verbose for tests
	query := NewQuery(ctx, transport, opts, logger, true)

	if err := query.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() {
		if err := query.Stop(ctx); err != nil {
			t.Logf("error stopping query: %v", err)
		}
	}()

	// Start goroutine to respond to initialize request
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
		t.Fatal("expected non-nil result")
	}

	// Verify hooks were registered
	query.mu.Lock()
	hookCount := len(query.hookCallbacks)
	query.mu.Unlock()

	if hookCount != 1 {
		t.Errorf("expected 1 hook callback, got %d", hookCount)
	}

	// Test non-streaming mode
	logger = log.NewLogger(false) // Non-verbose for tests
	nonStreamingQuery := NewQuery(ctx, transport, opts, logger, false)
	result, err = nonStreamingQuery.Initialize(ctx)
	if err != nil {
		t.Errorf("unexpected error for non-streaming mode: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for non-streaming mode")
	}
}

func TestQueryStop_ClearsHookCallbacks(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	transport := newMockTransport()
	hookCallback := func(ctx context.Context, input interface{}, toolUseID *string, hookCtx types.HookContext) (interface{}, error) {
		return map[string]interface{}{"continue": true}, nil
	}

	opts := types.NewClaudeAgentOptions().WithHook(
		types.HookEventPreToolUse,
		types.HookMatcher{
			Hooks: []types.HookCallbackFunc{hookCallback, hookCallback},
		},
	)
	logger := log.NewLogger(false)
	query := NewQuery(ctx, transport, opts, logger, true)

	if err := query.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	initDone := make(chan error, 1)
	go func() {
		_, err := query.Initialize(ctx)
		initDone <- err
	}()

	requestID := waitForWrittenControlRequest(t, transport, "initialize")
	transport.sendMessage(&types.SystemMessage{
		Type:    "control_response",
		Subtype: "control_response",
		Response: map[string]interface{}{
			"subtype":    "success",
			"request_id": requestID,
			"response": map[string]interface{}{
				"capabilities": []interface{}{"hooks"},
			},
		},
	})

	if err := <-initDone; err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if got := hookCallbackCount(query); got != 2 {
		t.Fatalf("expected 2 hook callbacks before Stop, got %d", got)
	}

	if err := query.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if got := hookCallbackCount(query); got != 0 {
		t.Fatalf("expected hook callbacks to be cleared after Stop, got %d", got)
	}

	if err := query.Stop(ctx); err != nil {
		t.Fatalf("second Stop failed: %v", err)
	}
	if got := hookCallbackCount(query); got != 0 {
		t.Fatalf("expected hook callbacks to stay cleared after second Stop, got %d", got)
	}
}

func TestQueryStopBeforeStart_ClearsHookCallbacks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	transport := newMockTransport()
	opts := types.NewClaudeAgentOptions()
	logger := log.NewLogger(false)
	query := NewQuery(ctx, transport, opts, logger, true)

	query.registerHookCallback(func(ctx context.Context, input interface{}, toolUseID *string, hookCtx types.HookContext) (interface{}, error) {
		return map[string]interface{}{"continue": true}, nil
	})

	if got := hookCallbackCount(query); got != 1 {
		t.Fatalf("expected 1 hook callback before Stop, got %d", got)
	}

	if err := query.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if got := hookCallbackCount(query); got != 0 {
		t.Fatalf("expected hook callbacks to be cleared after Stop before Start, got %d", got)
	}
}

func TestQueryStopHonorsContextWhenControlHandlerBlocks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	transport := newMockTransport()
	query := NewQuery(ctx, transport, types.NewClaudeAgentOptions(), log.NewLogger(false), true)
	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	callbackID := query.registerHookCallback(func(context.Context, interface{}, *string, types.HookContext) (interface{}, error) {
		close(handlerStarted)
		<-releaseHandler
		return map[string]interface{}{"continue": true}, nil
	})
	t.Cleanup(func() { close(releaseHandler) })

	if err := query.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		Subtype:   "control_request",
		RequestID: "hook-1",
		Request: map[string]interface{}{
			"subtype":     "hook_callback",
			"callback_id": callbackID,
			"input":       map[string]interface{}{"tool_name": "Bash"},
		},
	})

	select {
	case <-handlerStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("control handler did not start")
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := query.Stop(stopCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Stop error = %v, want context deadline", err)
	}
}

func TestInitializeFailure_ClearsRegisteredHookCallbacks(t *testing.T) {
	t.Parallel()

	parentCtx := context.Background()
	transport := newMockTransport()
	hookCallback := func(ctx context.Context, input interface{}, toolUseID *string, hookCtx types.HookContext) (interface{}, error) {
		return map[string]interface{}{"continue": true}, nil
	}

	opts := types.NewClaudeAgentOptions().WithHook(
		types.HookEventPreToolUse,
		types.HookMatcher{
			Hooks: []types.HookCallbackFunc{hookCallback},
		},
	)
	logger := log.NewLogger(false)
	query := NewQuery(parentCtx, transport, opts, logger, true)

	initCtx, cancel := context.WithTimeout(parentCtx, 50*time.Millisecond)
	t.Cleanup(cancel)

	if _, err := query.Initialize(initCtx); err == nil {
		t.Fatal("expected Initialize to fail")
	}
	if got := hookCallbackCount(query); got != 0 {
		t.Fatalf("expected hook callbacks to be cleared after Initialize failure, got %d", got)
	}
}

// TestErrorResponse tests error response handling.
func TestErrorResponse(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	transport := newMockTransport()
	opts := types.NewClaudeAgentOptions()

	logger := log.NewLogger(false) // Non-verbose for tests
	query := NewQuery(ctx, transport, opts, logger, true)

	if err := query.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() {
		if err := query.Stop(ctx); err != nil {
			t.Logf("error stopping query: %v", err)
		}
	}()

	// Send a control request
	responseChan := make(chan error, 1)

	go func() {
		request := map[string]interface{}{
			"subtype": "set_permission_mode",
			"mode":    "invalid",
		}
		_, err := query.sendControlRequest(ctx, request)
		responseChan <- err
	}()

	// Wait a bit for request to be sent
	time.Sleep(50 * time.Millisecond)

	// Get the written data to extract request ID
	written := transport.getWrittenData()
	if len(written) == 0 {
		t.Fatal("no data written to transport")
	}

	var sentRequest map[string]interface{}
	if err := json.Unmarshal([]byte(written[0]), &sentRequest); err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}

	requestID, _ := sentRequest["request_id"].(string)

	// Send an error response
	controlResponse := &types.SystemMessage{
		Type:    "control_response",
		Subtype: "control_response",
		Response: map[string]interface{}{
			"subtype":    "error",
			"request_id": requestID,
			"error":      "invalid permission mode",
		},
	}

	transport.sendMessage(controlResponse)

	// Wait for error
	select {
	case err := <-responseChan:
		if err == nil {
			t.Fatal("expected error response")
		}
		if !types.IsControlProtocolError(err) {
			t.Errorf("expected ControlProtocolError, got %T", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for error response")
	}
}

// TestQueryStartStop tests lifecycle management.
func TestQueryStartStop(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	transport := newMockTransport()
	opts := types.NewClaudeAgentOptions()

	logger := log.NewLogger(false) // Non-verbose for tests
	query := NewQuery(ctx, transport, opts, logger, true)

	// Start the query
	if err := query.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Starting again should fail
	if err := query.Start(ctx); err == nil {
		t.Error("expected error when starting already started query")
	}

	// Stop the query
	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := query.Stop(stopCtx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Messages channel should be closed
	select {
	case _, ok := <-query.messagesChan:
		if ok {
			t.Error("messages channel should be closed")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for messages channel to close")
	}
}

func TestQueryStopTimeoutDoesNotCloseMessagesBeforeProducerExits(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	transport := newBlockingReadTransport()
	t.Cleanup(transport.releaseRead)
	query := NewQuery(ctx, transport, types.NewClaudeAgentOptions(), log.NewLogger(false), true)

	if err := query.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	stopCtx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	if err := query.Stop(stopCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Stop error = %v, want context deadline", err)
	}

	select {
	case _, ok := <-query.messagesChan:
		if !ok {
			t.Fatal("messages channel closed before producer exited")
		}
	default:
	}

	transport.releaseRead()
	select {
	case <-query.readLoopDone:
	case <-time.After(2 * time.Second):
		t.Fatal("message loop did not exit after releasing blocked transport")
	}

	drainCtx, drainCancel := context.WithTimeout(ctx, 2*time.Second)
	defer drainCancel()
	if err := query.Stop(drainCtx); err != nil {
		t.Fatalf("second Stop after producer exit failed: %v", err)
	}
	select {
	case _, ok := <-query.messagesChan:
		if ok {
			t.Fatal("messages channel still open after producer exit")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for messages channel to close")
	}
}

// TestHandlePermissionRequest tests permission callback handling.
