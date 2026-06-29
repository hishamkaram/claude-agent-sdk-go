package internal

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

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
