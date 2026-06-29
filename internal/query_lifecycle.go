package internal

import (
	"context"

	"go.uber.org/zap"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// Start begins the control message handling loop.
func (q *Query) Start(ctx context.Context) error {
	q.mu.Lock()
	if q.started {
		q.mu.Unlock()
		return types.NewControlProtocolError("query already started")
	}
	q.started = true
	q.mu.Unlock()

	// Start message reading loop. Pass the query lifecycle context (q.ctx, not
	// the caller's Start ctx) so the loop and its control-request handler chain
	// observe the same cancellation as Stop(), while threading it as a parameter
	// down to the permission callback's timeout.
	//
	// q.ctx is created via context.WithCancel in NewQuery and is canceled by
	// Stop(); the caller's Start ctx has different cancellation semantics (it does
	// not fire on Stop's q.cancel()). contextcheck cannot see that this struct
	// field originated from context.WithCancel, so the field read at this single
	// entry point is annotated rather than restructured into a behavior change.
	go q.messageLoop(q.ctx) //nolint:contextcheck // q.ctx is the query lifecycle context (context.WithCancel in NewQuery, canceled by Stop); the caller's Start ctx has different cancellation semantics.

	return nil
}

// Stop gracefully stops the query handler.
func (q *Query) Stop(ctx context.Context) error {
	defer q.clearHookCallbacks()

	// Signal stop
	q.stopOnce.Do(func() { close(q.stopChan) })

	// Cancel context to stop all operations
	q.cancel()

	// If Start() was never called, readLoopDone will never be closed by
	// messageLoop. Close channels directly and return.
	q.mu.Lock()
	wasStarted := q.started
	q.mu.Unlock()
	if !wasStarted {
		q.closeMessagesOnce.Do(func() { close(q.messagesChan) })
		return nil
	}

	// Wait for read loop to complete
	select {
	case <-q.readLoopDone:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Wait for any in-flight handleControlRequest goroutines to finish.
	// This is safe after readLoopDone because no new handlers will be dispatched
	// once the message loop exits.
	if err := q.waitForHandlers(ctx); err != nil {
		return err
	}

	// Close message channel (safe even if messageLoop already closed it on transport EOF).
	q.closeMessagesOnce.Do(func() { close(q.messagesChan) })

	return nil
}

func (q *Query) waitForHandlers(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		q.handlerWg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetMessages returns a channel for consuming normal (non-control) messages.
func (q *Query) GetMessages(ctx context.Context) <-chan types.Message {
	return q.messagesChan
}

// messageLoop reads messages from transport and routes them. ctx is the query
// lifecycle context (q.ctx), threaded as a parameter so the control-request
// handler chain inherits it rather than reaching for the struct field.
func (q *Query) messageLoop(ctx context.Context) {
	defer close(q.readLoopDone) // Always close, even on panic (runs second — LIFO)
	defer func() {              // Runs first (LIFO) — catches panic before readLoopDone closes
		if r := recover(); r != nil {
			q.logger.Error("panic in messageLoop recovered",
				zap.Any("panic", r),
				zap.Stack("stack"),
			)
			// Close messagesChan so ReceiveResponse goroutines don't block forever.
			q.closeMessagesOnce.Do(func() { close(q.messagesChan) })
		}
	}()

	messages := q.transport.ReadMessages(ctx)
	q.logger.Debug("Message routing loop started")

	for {
		select {
		case <-ctx.Done():
			q.logger.Debug("Message loop stopped: context canceled")
			return
		case <-q.stopChan:
			q.logger.Debug("Message loop stopped: stop signal received")
			return
		case msg, ok := <-messages:
			if !ok {
				q.logger.Debug("Message loop stopped: transport channel closed")
				// Transport channel closed (subprocess EOF or crash). Close
				// messagesChan so that any ReceiveResponse() goroutines blocking
				// on it can exit immediately rather than waiting forever.
				// sync.Once prevents a double-close panic if Stop() is also called.
				q.closeMessagesOnce.Do(func() { close(q.messagesChan) })
				return
			}

			// Route message based on type
			if err := q.routeMessage(ctx, msg); err != nil {
				q.logger.Warn("message routing error", zap.Error(err))
				// Log error but continue processing
				// In a production system, we might want to report this via an error channel
				continue
			}
		}
	}
}

// routeMessage routes a message to the appropriate handler. ctx is threaded to
// the control-request handler goroutine so the permission callback inherits it.
func (q *Query) routeMessage(ctx context.Context, msg types.Message) error {
	// Check message type
	msgType := msg.GetMessageType()
	q.logger.Debug("routing message", zap.String("type", msgType))

	// Handle control responses
	if msgType == "control_response" {
		if sysMsg, ok := msg.(*types.SystemMessage); ok {
			return q.handleControlResponse(sysMsg)
		}
		return types.NewControlProtocolError("invalid control_response message type")
	}

	// Handle control requests
	if msgType == "control_request" {
		q.logger.Debug("Handling control request from CLI")
		if sysMsg, ok := msg.(*types.SystemMessage); ok {
			q.handlerWg.Add(1)
			go q.handleControlRequest(ctx, sysMsg)
			return nil
		}
		return types.NewControlProtocolError("invalid control_request message type")
	}

	if err := q.appendSessionStoreMessage(msg); err != nil {
		q.logger.Warn("session store append failed", zap.Error(err))
		if enqueueErr := q.enqueueMessage(&types.SystemMessage{
			Type:    "system",
			Subtype: "session_store_error",
			Data: map[string]interface{}{
				"operation": "append",
				"error":     err.Error(),
			},
		}); enqueueErr != nil {
			return enqueueErr
		}
	}
	return q.enqueueMessage(msg)
}

func (q *Query) enqueueMessage(msg types.Message) error {
	msgType := msg.GetMessageType()
	// The second select preserves the existing blocking backpressure behavior
	// when the internal queue is full.
	select {
	case q.messagesChan <- msg:
		if len(q.messagesChan) < cap(q.messagesChan) {
			q.messagesBackpressureWarned.Store(false)
		}
		return nil
	case <-q.ctx.Done():
		return q.ctx.Err()
	default:
	}

	// Queue is full — the producer must block until the consumer drains. Surface
	// each occurrence as a backpressure event so slow consumers are observable.
	q.options.ObserverOrNop().OnBackpressure()

	if q.messagesBackpressureWarned.CompareAndSwap(false, true) {
		q.logger.Warn("message queue backpressure detected",
			zap.Int("queued", len(q.messagesChan)),
			zap.Int("capacity", cap(q.messagesChan)),
			zap.String("message_type", msgType),
		)
	}

	select {
	case q.messagesChan <- msg:
		if len(q.messagesChan) < cap(q.messagesChan) {
			q.messagesBackpressureWarned.Store(false)
		}
		return nil
	case <-q.ctx.Done():
		return q.ctx.Err()
	}
}
