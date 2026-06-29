package internal

import (
	"strings"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestSendErrorResponse_NilRequest(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	_ = newTestQuery(t, transport)

	// Send a control_request with nil Request to trigger sendErrorResponse
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "test-req-1",
		Request:   nil, // This triggers the nil requestData path
	})

	// Wait briefly for the handler to process
	time.Sleep(100 * time.Millisecond)

	// Check that an error response was written
	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "invalid control request format") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error response for nil request, written data: %v", written)
	}
}

// TestHandleControlRequest_UnsupportedSubtype tests unknown subtype handling.

func TestHandleControlRequest_UnsupportedSubtype(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	_ = newTestQuery(t, transport)

	// Send a control_request with unknown subtype
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "test-req-2",
		Request: map[string]interface{}{
			"subtype": "unknown_subtype",
		},
	})

	time.Sleep(100 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "unsupported control request subtype") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error response for unknown subtype, written data: %v", written)
	}
}

// TestHandleControlRequest_InterruptSubtype tests interrupt request handling.

func TestHandleControlRequest_InterruptSubtype(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	_ = newTestQuery(t, transport)

	// Send interrupt request
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "test-req-3",
		Request: map[string]interface{}{
			"subtype": "interrupt",
		},
	})

	time.Sleep(100 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "success") && strings.Contains(data, "test-req-3") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected success response for interrupt, written data: %v", written)
	}
}

// TestHandleControlRequest_SetPermissionModeSubtype tests set_permission_mode request handling.

func TestHandleControlRequest_SetPermissionModeSubtype(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	_ = newTestQuery(t, transport)

	// Send set_permission_mode request
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "test-req-4",
		Request: map[string]interface{}{
			"subtype": "set_permission_mode",
			"mode":    "plan",
		},
	})

	time.Sleep(100 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "success") && strings.Contains(data, "test-req-4") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected success response for set_permission_mode, written data: %v", written)
	}
}

// TestHandleControlRequest_NoRequestID tests auto-generation of request ID.

func TestHandleControlRequest_NoRequestID(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	_ = newTestQuery(t, transport)

	// Send a control_request without request_id
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "", // empty request ID
		Request: map[string]interface{}{
			"subtype": "interrupt",
		},
	})

	time.Sleep(100 * time.Millisecond)

	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "cli-request-") && strings.Contains(data, "success") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected success response with auto-generated request ID, written data: %v", written)
	}
}

// TestHandleControlRequest_CanUseTool_NoCallback tests permission request without callback.

func TestHandleControlRequest_CanUseTool_NoCallback(t *testing.T) {
	t.Parallel()

	transport := newMockTransport()
	_ = newTestQuery(t, transport)

	// Send a can_use_tool request without setting up a canUseTool callback
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		RequestID: "test-req-5",
		Request: map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "Bash",
			"input":     map[string]interface{}{"command": "ls"},
		},
	})

	time.Sleep(100 * time.Millisecond)

	// Without a callback, it should send a success response with allow behavior
	written := transport.getWrittenData()
	found := false
	for _, data := range written {
		if strings.Contains(data, "test-req-5") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected response for can_use_tool, written data: %v", written)
	}
}

// TestSendSuccessResponse tests sendSuccessResponse directly.
