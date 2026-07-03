package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

const defaultControlResponseTimeout = 30 * time.Second

// handleControlResponse handles a control response message.
func (q *Query) handleControlResponse(msg *types.SystemMessage) error {
	// Parse response - use msg.Response for control_response messages
	responseData := msg.Response
	if responseData == nil {
		return types.NewControlProtocolError("invalid control response format: response field is nil")
	}

	requestID, ok := responseData["request_id"].(string)
	if !ok {
		return types.NewControlProtocolError("missing request_id in control response")
	}

	// Find pending request
	q.mu.Lock()
	responseChan, exists := q.requestMap[requestID]
	if exists {
		delete(q.requestMap, requestID)
	}
	q.mu.Unlock()

	if !exists {
		// Orphaned response - might be a timeout or duplicate
		return nil
	}

	// Check for error response
	subtype, _ := responseData["subtype"].(string)
	if subtype == "error" {
		errMsg, _ := responseData["error"].(string)
		if errMsg == "" {
			errMsg = "unknown control protocol error"
		}
		select {
		case responseChan <- responseResult{err: types.NewControlProtocolError(errMsg)}:
		case <-q.ctx.Done():
		default:
			q.logger.Debug("dropping late control error response (receiver timed out)",
				zap.String("request_id", requestID),
			)
		}
		return nil
	}

	// Success response
	response, _ := responseData["response"].(map[string]interface{})
	select {
	case responseChan <- responseResult{response: response}:
	case <-q.ctx.Done():
	default:
		q.logger.Debug("dropping late control success response (receiver timed out)",
			zap.String("request_id", requestID),
		)
	}

	return nil
}

// SendControlMessage is the exported wrapper around sendControlRequest.
// It allows callers outside the internal package (e.g. Client in the top-level
// package) to send arbitrary control protocol messages without exposing the
// full internal query machinery.
func (q *Query) SendControlMessage(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	return q.sendControlRequest(ctx, request)
}

// sendControlRequest sends a control request to CLI and waits for response.
func (q *Query) sendControlRequest(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	if !q.isStreamingMode {
		return nil, types.NewControlProtocolError("control requests require streaming mode")
	}
	waitCtx, cancel := q.controlResponseContext(ctx)
	defer cancel()

	// Generate unique request ID
	requestID := q.generateRequestID()

	// Create response channel
	responseChan := make(chan responseResult, 1)
	q.mu.Lock()
	q.requestMap[requestID] = responseChan
	q.mu.Unlock()

	// Build control request
	controlRequest := map[string]interface{}{
		"type":       "control_request",
		"request_id": requestID,
		"request":    request,
	}

	// Marshal and send
	data, err := json.Marshal(controlRequest)
	if err != nil {
		q.mu.Lock()
		delete(q.requestMap, requestID)
		q.mu.Unlock()
		return nil, types.NewControlProtocolErrorWithCause("failed to marshal control request", err)
	}

	if err := q.transport.Write(waitCtx, string(data)); err != nil {
		q.mu.Lock()
		delete(q.requestMap, requestID)
		q.mu.Unlock()
		return nil, types.NewControlProtocolErrorWithCause("failed to send control request", err)
	}

	// Wait for response with timeout
	select {
	case result := <-responseChan:
		if result.err != nil {
			return nil, result.err
		}
		return result.response, nil
	case <-waitCtx.Done():
		q.mu.Lock()
		delete(q.requestMap, requestID)
		q.mu.Unlock()
		return nil, waitCtx.Err()
	case <-q.ctx.Done():
		q.mu.Lock()
		delete(q.requestMap, requestID)
		q.mu.Unlock()
		return nil, types.NewControlProtocolError("query stopped")
	}
}

func (q *Query) controlResponseContext(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := defaultControlResponseTimeout
	if q.options != nil && q.options.ControlResponseTimeout > 0 {
		timeout = q.options.ControlResponseTimeout
	}
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= timeout {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

// sendSuccessResponse sends a success control response.
func (q *Query) sendSuccessResponse(requestID string, response map[string]interface{}) {
	controlResponse := map[string]interface{}{
		"type": "control_response",
		"response": map[string]interface{}{
			"subtype":    "success",
			"request_id": requestID,
			"response":   response,
		},
	}

	data, err := json.Marshal(controlResponse)
	if err != nil {
		q.logger.Error("sendSuccessResponse: failed to marshal response", zap.Error(err))
		return
	}

	q.logger.Debug("sendSuccessResponse: sending control_response",
		zap.String("request_id", requestID),
		zap.Int("response_key_count", len(response)),
	)
	if err := q.transport.Write(q.ctx, string(data)); err != nil {
		q.logger.Error("sendSuccessResponse: failed to write", zap.Error(err))
	}
}

// sendErrorResponse sends an error control response.
func (q *Query) sendErrorResponse(requestID, errorMsg string) {
	controlResponse := map[string]interface{}{
		"type": "control_response",
		"response": map[string]interface{}{
			"subtype":    "error",
			"request_id": requestID,
			"error":      errorMsg,
		},
	}

	data, err := json.Marshal(controlResponse)
	if err != nil {
		q.logger.Error("sendErrorResponse: failed to marshal error response", zap.Error(err))
		return
	}

	if err := q.transport.Write(q.ctx, string(data)); err != nil {
		q.logger.Error("sendErrorResponse: failed to write", zap.Error(err))
	}
}

// generateRequestID generates a unique request ID.
func (q *Query) generateRequestID() string {
	id := atomic.AddInt64(&q.nextRequestID, 1)
	return fmt.Sprintf("req_%d", id)
}
