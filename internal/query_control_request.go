package internal

import (
	"context"
	"fmt"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// handleControlRequest handles an incoming control request from CLI.
func (q *Query) handleControlRequest(ctx context.Context, msg *types.SystemMessage) {
	defer q.handlerWg.Done() // Runs second — ensures WaitGroup completes even on panic
	defer func() {           // Runs first — catches panic before Done
		if r := recover(); r != nil {
			q.logger.Error("panic in handleControlRequest recovered",
				zap.String("panic_type", fmt.Sprintf("%T", r)),
				zap.Stack("stack"),
				zap.String("request_id", msg.RequestID),
			)
		}
	}()

	q.logger.Debug("handleControlRequest: entered",
		append([]zap.Field{zap.String("request_id", msg.RequestID)}, controlRequestLogFields(msg.Request)...)...)

	// Get request ID from top-level field (CLI sends it here)
	requestID := msg.RequestID

	// Get request data from Request field
	requestData := msg.Request

	q.logger.Debug("handleControlRequest: parsed fields",
		append([]zap.Field{zap.String("request_id", requestID)}, controlRequestLogFields(requestData)...)...)

	// For CLI-initiated requests (like can_use_tool), there might not be a request_id
	// Generate one if needed
	if requestID == "" {
		requestID = fmt.Sprintf("cli-request-%d", atomic.AddInt64(&q.nextRequestID, 1))
		q.logger.Debug("handleControlRequest: generated request ID", zap.String("request_id", requestID))
	}

	if requestData == nil {
		q.logger.Error("handleControlRequest: invalid control request format: requestData is nil")
		q.sendErrorResponse(requestID, "invalid control request format")
		return
	}

	subtype, _ := requestData["subtype"].(string)
	q.logger.Debug("handleControlRequest: dispatching", zap.String("subtype", subtype))

	var response map[string]interface{}
	var err error

	switch subtype {
	case "can_use_tool":
		response, err = q.handlePermissionRequest(ctx, requestData)
	case "hook_callback":
		response, err = q.handleHookCallback(requestData)
	case "mcp_message":
		response, err = q.handleMCPMessage(requestData)
	case "interrupt":
		// Handle interrupt - just acknowledge for now
		response = make(map[string]interface{})
	case "set_permission_mode":
		// Handle permission mode change - acknowledge for now
		response = make(map[string]interface{})
	default:
		err = types.NewControlProtocolError("unsupported control request subtype: " + subtype)
	}

	if err != nil {
		q.sendErrorResponse(requestID, err.Error())
		return
	}

	q.sendSuccessResponse(requestID, response)
}
