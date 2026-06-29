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
