package internal

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

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
