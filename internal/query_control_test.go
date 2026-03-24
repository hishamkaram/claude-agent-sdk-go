package internal

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// sendControlResponse is a test helper that reads the first written request,
// extracts the request_id, and injects a success control_response.
func sendControlResponse(t *testing.T, transport *mockTransport, extraFields map[string]interface{}) {
	t.Helper()
	time.Sleep(50 * time.Millisecond)

	written := transport.getWrittenData()
	if len(written) == 0 {
		t.Fatal("no data written to transport")
	}

	var sentRequest map[string]interface{}
	if err := json.Unmarshal([]byte(written[len(written)-1]), &sentRequest); err != nil {
		t.Fatalf("failed to unmarshal sent request: %v", err)
	}

	requestID, _ := sentRequest["request_id"].(string)
	if requestID == "" {
		t.Fatal("request_id missing from sent request")
	}

	resp := map[string]interface{}{
		"subtype":    "success",
		"request_id": requestID,
	}
	for k, v := range extraFields {
		resp[k] = v
	}

	transport.sendMessage(&types.SystemMessage{
		Type:     "control_response",
		Subtype:  "control_response",
		Response: resp,
	})
}

// newTestQuery creates a Query for unit tests with an already-started loop.
func newTestQuery(t *testing.T, transport *mockTransport) *Query {
	t.Helper()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions()
	logger := log.NewLogger(false)
	q := NewQuery(ctx, transport, opts, logger, true)
	if err := q.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() {
		_ = q.Stop(ctx)
	})
	return q
}

// TestSendControlMessage_SetModel_Success verifies set_model round-trip via the
// public SendControlMessage method.
func TestSendControlMessage_SetModel_Success(t *testing.T) {
	t.Parallel()
	transport := newMockTransport()
	q := newTestQuery(t, transport)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		_, err := q.SendControlMessage(ctx, map[string]interface{}{
			"subtype": "set_model",
			"model":   "haiku",
		})
		errCh <- err
	}()

	// Verify the wire format and inject a success response.
	time.Sleep(50 * time.Millisecond)
	written := transport.getWrittenData()
	if len(written) == 0 {
		t.Fatal("no data written to transport")
	}

	var outer map[string]interface{}
	if err := json.Unmarshal([]byte(written[len(written)-1]), &outer); err != nil {
		t.Fatalf("unmarshal outer: %v", err)
	}
	if outer["type"] != "control_request" {
		t.Errorf("expected type 'control_request', got %q", outer["type"])
	}
	req, _ := outer["request"].(map[string]interface{})
	if req == nil {
		t.Fatal("request field missing")
	}
	if req["subtype"] != "set_model" {
		t.Errorf("expected subtype 'set_model', got %q", req["subtype"])
	}
	if req["model"] != "haiku" {
		t.Errorf("expected model 'haiku', got %q", req["model"])
	}

	sendControlResponse(t, transport, nil)

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}
}

// TestSendControlMessage_SetModel_DefaultModel verifies that an empty model string
// results in a set_model request WITHOUT a "model" key (omit = revert to default).
func TestSendControlMessage_SetModel_DefaultModel(t *testing.T) {
	t.Parallel()
	transport := newMockTransport()
	q := newTestQuery(t, transport)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// The caller (Client.SetModel) omits "model" when empty. We verify here that
	// if we send WITHOUT "model" the field is truly absent.
	req := map[string]interface{}{
		"subtype": "set_model",
		// "model" intentionally absent — reverting to default
	}

	errCh := make(chan error, 1)
	go func() {
		_, err := q.SendControlMessage(ctx, req)
		errCh <- err
	}()

	time.Sleep(50 * time.Millisecond)
	written := transport.getWrittenData()
	if len(written) == 0 {
		t.Fatal("no data written to transport")
	}

	var outer map[string]interface{}
	if err := json.Unmarshal([]byte(written[len(written)-1]), &outer); err != nil {
		t.Fatalf("unmarshal outer: %v", err)
	}
	inner, _ := outer["request"].(map[string]interface{})
	if inner == nil {
		t.Fatal("request field missing")
	}
	if _, exists := inner["model"]; exists {
		t.Errorf("expected 'model' key to be absent for default model, but it was present")
	}

	sendControlResponse(t, transport, nil)

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}
}

// TestSendControlMessage_SetPermissionMode_Plan verifies set_permission_mode round-trip.
func TestSendControlMessage_SetPermissionMode_Plan(t *testing.T) {
	t.Parallel()
	transport := newMockTransport()
	q := newTestQuery(t, transport)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		_, err := q.SendControlMessage(ctx, map[string]interface{}{
			"subtype": "set_permission_mode",
			"mode":    "plan",
		})
		errCh <- err
	}()

	time.Sleep(50 * time.Millisecond)
	written := transport.getWrittenData()
	if len(written) == 0 {
		t.Fatal("no data written to transport")
	}

	var outer map[string]interface{}
	if err := json.Unmarshal([]byte(written[len(written)-1]), &outer); err != nil {
		t.Fatalf("unmarshal outer: %v", err)
	}
	inner, _ := outer["request"].(map[string]interface{})
	if inner == nil {
		t.Fatal("request field missing")
	}
	if inner["subtype"] != "set_permission_mode" {
		t.Errorf("expected subtype 'set_permission_mode', got %q", inner["subtype"])
	}
	if inner["mode"] != "plan" {
		t.Errorf("expected mode 'plan', got %q", inner["mode"])
	}

	sendControlResponse(t, transport, nil)

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for response")
	}
}

// TestSendControlMessage_SetPermissionMode_AcceptEdits verifies acceptEdits mode.
func TestSendControlMessage_SetPermissionMode_AcceptEdits(t *testing.T) {
	t.Parallel()
	transport := newMockTransport()
	q := newTestQuery(t, transport)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		_, err := q.SendControlMessage(ctx, map[string]interface{}{
			"subtype": "set_permission_mode",
			"mode":    "acceptEdits",
		})
		errCh <- err
	}()

	time.Sleep(50 * time.Millisecond)
	written := transport.getWrittenData()
	if len(written) == 0 {
		t.Fatal("no data written")
	}

	sendControlResponse(t, transport, nil)

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

// TestSendControlMessage_ErrorResponse verifies error responses are propagated.
func TestSendControlMessage_ErrorResponse(t *testing.T) {
	t.Parallel()
	transport := newMockTransport()
	q := newTestQuery(t, transport)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		_, err := q.SendControlMessage(ctx, map[string]interface{}{
			"subtype": "set_permission_mode",
			"mode":    "invalid_mode",
		})
		errCh <- err
	}()

	time.Sleep(50 * time.Millisecond)
	written := transport.getWrittenData()
	if len(written) == 0 {
		t.Fatal("no data written")
	}

	var outer map[string]interface{}
	_ = json.Unmarshal([]byte(written[len(written)-1]), &outer)
	requestID, _ := outer["request_id"].(string)

	transport.sendMessage(&types.SystemMessage{
		Type:    "control_response",
		Subtype: "control_response",
		Response: map[string]interface{}{
			"subtype":    "error",
			"request_id": requestID,
			"error":      "invalid permission mode: invalid_mode",
		},
	})

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !types.IsControlProtocolError(err) {
			t.Errorf("expected ControlProtocolError, got %T: %v", err, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for error response")
	}
}

// TestSendControlMessage_ContextCancelled verifies context cancellation is respected.
func TestSendControlMessage_ContextCancelled(t *testing.T) {
	t.Parallel()
	transport := newMockTransport()
	q := newTestQuery(t, transport)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		_, err := q.SendControlMessage(ctx, map[string]interface{}{
			"subtype": "set_model",
			"model":   "haiku",
		})
		errCh <- err
	}()

	// Cancel before the response arrives.
	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error after context cancellation")
		}
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for cancellation error")
	}
}
