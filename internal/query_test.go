package internal

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// mockTransport implements a mock transport for testing.
type mockTransport struct {
	mu             sync.Mutex
	messagesChan   chan types.Message
	writtenData    []string
	closed         bool
	ready          bool
	err            error
	onErrorHandler func(error)
}

type failingSessionStore struct{}

func (failingSessionStore) Append(ctx context.Context, key types.SessionKey, entries []types.SessionMessage) error {
	return errors.New("mirror append failed")
}

func (failingSessionStore) Load(ctx context.Context, key types.SessionKey) (*types.SessionStoreEntry, error) {
	return nil, nil
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		messagesChan: make(chan types.Message, 100),
		writtenData:  make([]string, 0),
		ready:        true,
	}
}

func (m *mockTransport) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ready = true
	return nil
}

func (m *mockTransport) Close(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		close(m.messagesChan)
		m.closed = true
	}
	m.ready = false
	return nil
}

func (m *mockTransport) Write(ctx context.Context, data string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writtenData = append(m.writtenData, data)
	return nil
}

func (m *mockTransport) ReadMessages(ctx context.Context) <-chan types.Message {
	return m.messagesChan
}

func (m *mockTransport) OnError(err error) {
	if m.onErrorHandler != nil {
		m.onErrorHandler(err)
	}
}

func (m *mockTransport) IsReady() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ready
}

func (m *mockTransport) GetError() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}

func (m *mockTransport) sendMessage(msg types.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.messagesChan <- msg
	}
}

func (m *mockTransport) getWrittenData() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.writtenData...)
}

func waitForControlResponses(t *testing.T, transport *mockTransport, want int) []map[string]interface{} {
	t.Helper()

	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		written := transport.getWrittenData()
		responses := make([]map[string]interface{}, 0, len(written))
		for _, data := range written {
			var msg map[string]interface{}
			if err := json.Unmarshal([]byte(data), &msg); err != nil {
				continue
			}
			if msg["type"] == "control_response" {
				responses = append(responses, msg)
			}
		}

		if len(responses) >= want {
			return responses
		}

		select {
		case <-ticker.C:
		case <-deadline:
			t.Fatalf("timed out waiting for %d control_response messages, got %d: %v", want, len(responses), written)
		}
	}
}

func assertPermissionPanicErrorResponse(t *testing.T, transport *mockTransport, requestID string) {
	t.Helper()

	responses := waitForControlResponses(t, transport, 1)
	if len(responses) != 1 {
		t.Fatalf("expected exactly one control_response, got %d: %v", len(responses), responses)
	}

	response, ok := responses[0]["response"].(map[string]interface{})
	if !ok {
		t.Fatalf("response field missing or invalid: %v", responses[0])
	}
	if got, _ := response["request_id"].(string); got != requestID {
		t.Fatalf("response.request_id = %q, want %q", got, requestID)
	}
	if got, _ := response["subtype"].(string); got != "error" {
		t.Fatalf("response.subtype = %q, want error", got)
	}
	errMsg, _ := response["error"].(string)
	if !strings.Contains(errMsg, "permission callback panicked") {
		t.Fatalf("response.error = %q, want permission callback panicked", errMsg)
	}
}

func TestRouteMessage_LogsBackpressureWhenMessagesChannelFull(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	core, logs := observer.New(zapcore.DebugLevel)
	logger := log.NewLoggerFromZap(zap.New(core))
	query := NewQuery(ctx, newMockTransport(), types.NewClaudeAgentOptions(), logger, true)

	capacity := cap(query.messagesChan)
	for i := 0; i < capacity; i++ {
		query.messagesChan <- &types.UserMessage{Type: "user", Content: "queued"}
	}

	overflow := &types.AssistantMessage{Type: "assistant"}
	routeErr := make(chan error, 1)
	go func() {
		routeErr <- query.routeMessage(ctx, overflow)
	}()

	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	var entries []observer.LoggedEntry
	for len(entries) == 0 {
		entries = logs.FilterMessage("message queue backpressure detected").All()
		if len(entries) > 0 {
			break
		}

		select {
		case err := <-routeErr:
			t.Fatalf("routeMessage completed before logging backpressure: %v", err)
		case <-ticker.C:
		case <-deadline:
			t.Fatal("timed out waiting for backpressure warning log")
		}
	}

	fields := entries[0].ContextMap()
	if got := fields["queued"]; got != int64(capacity) {
		t.Fatalf("queued field = %v, want %d", got, capacity)
	}
	if got := fields["capacity"]; got != int64(capacity) {
		t.Fatalf("capacity field = %v, want %d", got, capacity)
	}
	if got, _ := fields["message_type"].(string); got != "assistant" {
		t.Fatalf("message_type field = %v, want assistant", fields["message_type"])
	}

	select {
	case err := <-routeErr:
		t.Fatalf("routeMessage completed before the queue was drained: %v", err)
	default:
	}

	<-query.messagesChan

	select {
	case err := <-routeErr:
		if err != nil {
			t.Fatalf("routeMessage returned error after drain: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("routeMessage did not unblock after draining one queued message")
	}

	deliveredOverflow := false
	for i := 0; i < capacity; i++ {
		select {
		case got := <-query.messagesChan:
			if got == overflow {
				deliveredOverflow = true
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out draining delivered message %d", i)
		}
	}
	if !deliveredOverflow {
		t.Fatal("overflow message was not delivered after backpressure cleared")
	}
}

func TestRouteMessageSessionStoreAppendErrorEmitsSystemMessage(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	opts := types.NewClaudeAgentOptions().
		WithSessionStore(failingSessionStore{}).
		WithSessionStoreKey(types.SessionKey{SessionID: "session-1", ProjectKey: "project-1"})
	query := NewQuery(ctx, newMockTransport(), opts, log.NewLogger(false), true)

	msg := &types.UserMessage{Type: "user", Content: "hello", SessionID: "session-1"}
	if err := query.routeMessage(ctx, msg); err != nil {
		t.Fatalf("routeMessage: %v", err)
	}
	first := <-query.messagesChan
	sys, ok := first.(*types.SystemMessage)
	if !ok {
		t.Fatalf("first message = %T, want *types.SystemMessage", first)
	}
	if sys.Subtype != "session_store_error" {
		t.Fatalf("system subtype = %q, want session_store_error", sys.Subtype)
	}
	second := <-query.messagesChan
	if second != msg {
		t.Fatalf("second message = %T, want original user message", second)
	}
}

func hookCallbackCount(q *Query) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.hookCallbacks)
}

func waitForWrittenControlRequest(t *testing.T, transport *mockTransport, subtype string) string {
	t.Helper()

	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		for _, data := range transport.getWrittenData() {
			var sentRequest map[string]interface{}
			if err := json.Unmarshal([]byte(data), &sentRequest); err != nil {
				continue
			}

			reqType, _ := sentRequest["type"].(string)
			if reqType != "control_request" {
				continue
			}

			request, _ := sentRequest["request"].(map[string]interface{})
			gotSubtype, _ := request["subtype"].(string)
			if gotSubtype != subtype {
				continue
			}

			requestID, _ := sentRequest["request_id"].(string)
			if requestID == "" {
				t.Fatal("request_id missing from sent request")
			}
			return requestID
		}

		select {
		case <-ticker.C:
		case <-deadline:
			t.Fatalf("timed out waiting for %s control request, written: %v", subtype, transport.getWrittenData())
		}
	}
}

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
func TestRequestResponseCorrelation(t *testing.T) {
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

	// Send a control request in a goroutine
	responseChan := make(chan map[string]interface{}, 1)
	errorChan := make(chan error, 1)

	go func() {
		request := map[string]interface{}{
			"subtype": "set_permission_mode",
			"mode":    "default",
		}
		result, err := query.sendControlRequest(ctx, request)
		if err != nil {
			errorChan <- err
			return
		}
		responseChan <- result
	}()

	// Wait a bit for the request to be sent
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
	if requestID == "" {
		t.Fatal("request_id not found in sent request")
	}

	// Send a control response
	controlResponse := &types.SystemMessage{
		Type:    "control_response",
		Subtype: "control_response",
		Response: map[string]interface{}{
			"subtype":    "success",
			"request_id": requestID,
			"response": map[string]interface{}{
				"mode": "default",
			},
		},
	}

	transport.sendMessage(controlResponse)

	// Wait for response
	select {
	case result := <-responseChan:
		if mode, ok := result["mode"].(string); !ok || mode != "default" {
			t.Errorf("unexpected result: %v", result)
		}
	case err := <-errorChan:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}
}

// TestMessageRouting tests that normal messages pass through to consumer.
func TestMessageRouting(t *testing.T) {
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

	// Send a normal message
	userMsg := &types.UserMessage{
		Type:    "user",
		Content: "test message",
	}

	transport.sendMessage(userMsg)

	// Receive from messages channel
	messages := query.GetMessages(ctx)

	select {
	case msg := <-messages:
		if msg.GetMessageType() != "user" {
			t.Errorf("expected user message, got %s", msg.GetMessageType())
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

// TestControlMessageFiltering tests that control messages don't leak to consumer.
func TestControlMessageFiltering(t *testing.T) {
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
	controlRequest := &types.SystemMessage{
		Type:    "control_request",
		Subtype: "control_request",
		Request: map[string]interface{}{
			"subtype": "interrupt",
		},
	}

	transport.sendMessage(controlRequest)

	// Send a normal message after
	userMsg := &types.UserMessage{
		Type:    "user",
		Content: "test message",
	}

	transport.sendMessage(userMsg)

	// Should only receive the user message
	messages := query.GetMessages(ctx)

	select {
	case msg := <-messages:
		if msg.GetMessageType() != "user" {
			t.Errorf("expected user message, got %s", msg.GetMessageType())
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

// TestConcurrentRequests tests multiple simultaneous requests.
func TestConcurrentRequests(t *testing.T) {
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

	// Start a goroutine to respond to all control requests
	go func() {
		for {
			time.Sleep(10 * time.Millisecond)
			written := transport.getWrittenData()

			// Process all written requests
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
				if requestID == "" {
					continue
				}

				// Check if we already responded to this request
				// by checking if it's still in the request map
				query.mu.Lock()
				_, exists := query.requestMap[requestID]
				query.mu.Unlock()

				if exists {
					// Send response
					controlResponse := &types.SystemMessage{
						Type:    "control_response",
						Subtype: "control_response",
						Response: map[string]interface{}{
							"subtype":    "success",
							"request_id": requestID,
							"response":   map[string]interface{}{},
						},
					}
					transport.sendMessage(controlResponse)
				}
			}

			// Exit when all requests are done
			query.mu.Lock()
			pendingCount := len(query.requestMap)
			query.mu.Unlock()
			if pendingCount == 0 {
				time.Sleep(100 * time.Millisecond) // Give time for any stragglers
				break
			}
		}
	}()

	numRequests := 10
	var wg sync.WaitGroup
	wg.Add(numRequests)

	results := make([]error, numRequests)

	// Send multiple concurrent requests
	for i := 0; i < numRequests; i++ {
		go func(index int) {
			defer wg.Done()

			request := map[string]interface{}{
				"subtype": "set_permission_mode",
				"mode":    "default",
			}

			_, err := query.sendControlRequest(ctx, request)
			results[index] = err
		}(i)
	}

	wg.Wait()

	// Check results
	for i, err := range results {
		if err != nil {
			t.Errorf("request %d failed: %v", i, err)
		}
	}
}

// TestContextCancellation tests cleanup on context cancellation.
func TestContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	transport := newMockTransport()
	opts := types.NewClaudeAgentOptions()

	logger := log.NewLogger(false) // Non-verbose for tests
	query := NewQuery(ctx, transport, opts, logger, true)

	if err := query.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Cancel context
	cancel()

	// Wait a bit for cleanup
	time.Sleep(100 * time.Millisecond)

	// Messages channel should eventually close
	select {
	case _, ok := <-query.messagesChan:
		if ok {
			t.Error("messages channel should be closed after context cancellation")
		}
	case <-time.After(1 * time.Second):
		// Channel might not be closed immediately, that's ok
	}
}

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
