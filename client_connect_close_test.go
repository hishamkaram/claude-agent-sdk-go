package claude

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal"
	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// ===== Bug C19: Connect lock scope tests =====

// TestClient_ConnectDoesNotBlockIsConnected verifies that IsConnected() is not
// blocked by a concurrent Connect() call. With the old broad lock scope,
// IsConnected() would block waiting for the lock while Connect() held it
// during blocking transport operations.
func TestClient_ConnectDoesNotBlockIsConnected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "IsConnected returns immediately during Connect"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

			client, err := NewClient(ctx, opts)
			if err != nil {
				t.Skip("Could not create client")
			}
			defer func() { _ = client.Close(ctx) }()

			// Start Connect in background — it will block on initialization
			connectDone := make(chan error, 1)
			go func() { connectDone <- client.Connect(ctx) }()

			// Give Connect a moment to start
			time.Sleep(20 * time.Millisecond)

			// IsConnected() should return immediately, not block on the lock.
			// We test this with a tight deadline.
			isConnectedDone := make(chan bool, 1)
			go func() {
				isConnectedDone <- client.IsConnected()
			}()

			select {
			case connected := <-isConnectedDone:
				// Good — IsConnected returned without blocking.
				// It should be false since Connect hasn't completed.
				if connected {
					t.Error("expected IsConnected() = false during Connect()")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("IsConnected() blocked for 2s — lock scope too broad during Connect()")
			}

			// Wait for Connect to finish (will fail because /bin/echo is not Claude CLI)
			select {
			case <-connectDone:
			case <-time.After(10 * time.Second):
				t.Fatal("Connect() hung for 10s")
			}
		})
	}
}

// TestClient_ConnectRejectsDoubleConnecting verifies that concurrent Connect()
// calls are rejected with a clear error rather than blocking or deadlocking.
func TestClient_ConnectRejectsDoubleConnecting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "second concurrent Connect returns error"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

			client, err := NewClient(ctx, opts)
			if err != nil {
				t.Skip("Could not create client")
			}
			defer func() { _ = client.Close(ctx) }()

			// Manually set the connecting flag to simulate an in-progress Connect
			client.mu.Lock()
			client.connecting = true
			client.mu.Unlock()

			// A second Connect call should return immediately with an error
			err = client.Connect(ctx)
			if err == nil {
				t.Fatal("expected error for concurrent Connect()")
			}
			if !types.IsControlProtocolError(err) {
				t.Errorf("expected ControlProtocolError, got: %T - %v", err, err)
			}

			// Reset the flag so Close() doesn't see stale state
			client.mu.Lock()
			client.connecting = false
			client.mu.Unlock()
		})
	}
}

// TestClient_CloseDuringConnect verifies that calling Close() while Connect()
// is in progress (Phase 2) sets closePending so Connect() cleans up instead
// of completing the connection.
func TestClient_CloseDuringConnect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "Close during Connect sets closePending flag"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

			client, err := NewClient(ctx, opts)
			if err != nil {
				t.Skip("Could not create client")
			}

			// Simulate in-progress Connect by setting the connecting flag.
			client.mu.Lock()
			client.connecting = true
			client.mu.Unlock()

			// Close() should not return an error — it sets closePending instead.
			err = client.Close(ctx)
			if err != nil {
				t.Fatalf("Close() during connecting returned error: %v", err)
			}

			// Verify closePending was set.
			client.mu.Lock()
			pending := client.closePending
			client.mu.Unlock()

			if !pending {
				t.Error("expected closePending=true after Close() during connecting")
			}

			// Verify connected is still false (Close didn't try to clean up
			// a non-existent connection).
			if client.IsConnected() {
				t.Error("expected IsConnected()=false after Close() during connecting")
			}

			// Reset flags so cleanup doesn't see stale state.
			client.mu.Lock()
			client.connecting = false
			client.closePending = false
			client.mu.Unlock()
		})
	}
}

func TestClient_ConnectTransportClosedClientReturnsClientCause(t *testing.T) {
	t.Parallel()

	parentCtx := context.Background()
	clientCtx, cancel := context.WithCancel(parentCtx)
	mockTransport := newClientTestTransport()
	client := &Client{
		transport: mockTransport,
		logger:    log.NewLogger(false),
		ctx:       clientCtx,
		cancel:    cancel,
	}
	cancel()

	err := client.connectTransport(parentCtx)
	if !types.IsControlProtocolError(err) {
		t.Fatalf("connectTransport error = %T %v, want ControlProtocolError", err, err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("connectTransport error = %v, want context.Canceled cause", err)
	}
}

// TestClient_CloseNotConnectingNoop verifies that Close() on a client that is
// neither connected nor connecting is a no-op.
func TestClient_CloseNotConnectingNoop(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}

	// Close on a fresh (not connected, not connecting) client should be a no-op.
	err = client.Close(ctx)
	if err != nil {
		t.Fatalf("Close() on fresh client returned error: %v", err)
	}

	// closePending should NOT be set.
	client.mu.Lock()
	pending := client.closePending
	client.mu.Unlock()

	if pending {
		t.Error("expected closePending=false when Close() called on idle client")
	}
}

func TestClientCloseReturnsFirstShutdownError(t *testing.T) {
	t.Parallel()

	handlerStarted := make(chan struct{})
	handlerDone := make(chan struct{})
	releaseHandler := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseHandler) }) }
	opts := types.NewClaudeAgentOptions().
		WithCanUseTool(func(context.Context, string, map[string]interface{}, types.ToolPermissionContext) (interface{}, error) {
			defer close(handlerDone)
			close(handlerStarted)
			<-releaseHandler
			return types.PermissionResultDeny{Message: "closing"}, nil
		})
	t.Cleanup(release)

	transportErr := errors.New("transport close failed")
	transport := newClientTestTransport()
	transport.closeErr = transportErr
	logger := log.NewLogger(false)
	query := internal.NewQuery(context.Background(), transport, opts, logger, true)
	if err := query.Start(context.Background()); err != nil {
		t.Fatalf("query.Start: %v", err)
	}
	transport.sendMessage(&types.SystemMessage{
		Type:      "control_request",
		Subtype:   "control_request",
		RequestID: "permission-1",
		Request: map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "Bash",
			"input":     map[string]interface{}{"command": "pwd"},
		},
	})

	select {
	case <-handlerStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("permission handler did not start")
	}

	clientCtx, clientCancel := context.WithCancel(context.Background())
	client := &Client{
		options:   opts,
		transport: transport,
		query:     query,
		logger:    logger,
		connected: true,
		ctx:       clientCtx,
		cancel:    clientCancel,
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := client.Close(closeCtx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Close error = %v, want context deadline component", err)
	}
	if errors.Is(err, transportErr) {
		t.Fatalf("Close error = %v, want legacy first-error contract without transport component", err)
	}
	release()
	select {
	case <-handlerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("permission handler did not finish after release")
	}
}

// TestClient_ReceiveResponse_GoroutineTracked verifies that the recvWg field on
// Client is incremented when ReceiveResponse spawns a goroutine and decremented
// when the goroutine exits. This ensures Close() can wait for in-flight goroutines.
func TestClient_ReceiveResponse_GoroutineTracked(t *testing.T) {
	t.Parallel()

	client, mockTransport := makeConnectedClient(t)
	ctx := context.Background()

	// Call ReceiveResponse — spawns a goroutine internally.
	respCh := client.ReceiveResponse(ctx)

	// The goroutine is now running. Send a result message so it finishes.
	mockTransport.sendMessage(&types.ResultMessage{Type: "result"})

	// Drain the output channel.
	for range respCh {
	}

	// After the goroutine exits, recvWg should be at zero.
	// We verify by calling recvWg.Wait() which should return immediately.
	done := make(chan struct{})
	go func() {
		client.recvWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// recvWg is at zero — goroutine properly tracked.
	case <-time.After(2 * time.Second):
		t.Fatal("recvWg.Wait() blocked — goroutine was not tracked with recvWg.Add/Done")
	}

	// Close should complete quickly.
	if err := client.Close(ctx); err != nil {
		t.Logf("Close returned error (acceptable): %v", err)
	}
}

func TestClient_ReceiveResponseClosesAfterPostResultWorkflowNotification(t *testing.T) {
	t.Parallel()

	client, mockTransport := makeConnectedClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	respCh := client.ReceiveResponse(ctx)

	taskID := "task-agent-1"
	toolUseID := "toolu_agent_1"
	taskType := "local_workflow"
	intermediate := "launched child agent"
	mockTransport.sendMessage(&types.TaskStartedMessage{
		Type:      "system",
		TaskID:    taskID,
		ToolUseID: &toolUseID,
		TaskType:  &taskType,
	})
	mockTransport.sendMessage(&types.TaskUpdatedMessage{
		Type:   "system",
		TaskID: taskID,
		Patch:  types.TaskUpdatedPatch{Status: "completed"},
	})
	mockTransport.sendMessage(&types.ResultMessage{
		Type:   "result",
		Result: &intermediate,
	})

	if msg := receiveTestMessage(t, respCh); msg.GetMessageType() != "system" {
		t.Fatalf("first message type = %q, want system", msg.GetMessageType())
	}
	if msg := receiveTestMessage(t, respCh); msg.GetMessageType() != "system" {
		t.Fatalf("second message type = %q, want system", msg.GetMessageType())
	}
	if msg := receiveTestMessage(t, respCh); msg.GetMessageType() != "result" {
		t.Fatalf("third message type = %q, want result", msg.GetMessageType())
	}
	select {
	case _, ok := <-respCh:
		if !ok {
			t.Fatal("ReceiveResponse closed on intermediate ResultMessage before task notification")
		}
		t.Fatal("unexpected extra message before task completion")
	case <-time.After(20 * time.Millisecond):
	}

	mockTransport.sendMessage(&types.TaskNotificationMessage{
		Type:      "system",
		TaskID:    taskID,
		ToolUseID: &toolUseID,
		Status:    "completed",
	})

	if msg := receiveTestMessage(t, respCh); msg.GetMessageType() != "system" {
		t.Fatalf("fourth message type = %q, want system", msg.GetMessageType())
	}
	select {
	case _, ok := <-respCh:
		if ok {
			t.Fatal("ReceiveResponse stayed open after post-result task notification")
		}
	case <-time.After(time.Second):
		t.Fatal("ReceiveResponse did not close after post-result task notification")
	}

	if err := client.Close(ctx); err != nil {
		t.Logf("Close returned error (acceptable): %v", err)
	}
}

func TestClient_ReceiveResponseClosesAfterPostResultWorkflowTerminalUpdate(t *testing.T) {
	t.Parallel()

	client, mockTransport := makeConnectedClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	respCh := client.ReceiveResponse(ctx)

	taskID := "task-workflow-1"
	toolUseID := "toolu_workflow_1"
	taskType := taskTypeLocalWorkflow
	final := "workflow completed"
	mockTransport.sendMessage(&types.TaskStartedMessage{
		Type:      "system",
		TaskID:    taskID,
		ToolUseID: &toolUseID,
		TaskType:  &taskType,
	})
	mockTransport.sendMessage(&types.ResultMessage{
		Type:   "result",
		Result: &final,
	})

	if msg := receiveTestMessage(t, respCh); msg.GetMessageType() != "system" {
		t.Fatalf("first message type = %q, want system", msg.GetMessageType())
	}
	if msg := receiveTestMessage(t, respCh); msg.GetMessageType() != "result" {
		t.Fatalf("second message type = %q, want result", msg.GetMessageType())
	}
	select {
	case _, ok := <-respCh:
		if !ok {
			t.Fatal("ReceiveResponse closed before terminal workflow update")
		}
		t.Fatal("unexpected extra message before terminal workflow update")
	case <-time.After(20 * time.Millisecond):
	}

	mockTransport.sendMessage(&types.TaskUpdatedMessage{
		Type:      "system",
		TaskID:    taskID,
		ToolUseID: &toolUseID,
		Patch:     types.TaskUpdatedPatch{Status: "completed"},
	})

	if msg := receiveTestMessage(t, respCh); msg.GetMessageType() != "system" {
		t.Fatalf("third message type = %q, want system", msg.GetMessageType())
	}
	select {
	case _, ok := <-respCh:
		if ok {
			t.Fatal("ReceiveResponse stayed open after post-result terminal workflow update")
		}
	case <-time.After(time.Second):
		t.Fatal("ReceiveResponse did not close after post-result terminal workflow update")
	}

	if err := client.Close(ctx); err != nil {
		t.Logf("Close returned error (acceptable): %v", err)
	}
}

func TestClient_ReceiveResponseKeepsGraceCloseAfterUnrelatedFrame(t *testing.T) {
	t.Parallel()

	client, mockTransport := makeConnectedClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	respCh := client.ReceiveResponse(ctx)

	taskID := "task-workflow-1"
	toolUseID := "toolu_workflow_1"
	taskType := taskTypeLocalWorkflow
	final := "workflow completed"
	mockTransport.sendMessage(&types.TaskStartedMessage{
		Type:      "system",
		TaskID:    taskID,
		ToolUseID: &toolUseID,
		TaskType:  &taskType,
	})
	mockTransport.sendMessage(&types.TaskUpdatedMessage{
		Type:      "system",
		TaskID:    taskID,
		ToolUseID: &toolUseID,
		Patch:     types.TaskUpdatedPatch{Status: "completed"},
	})
	mockTransport.sendMessage(&types.ResultMessage{
		Type:   "result",
		Result: &final,
	})

	for i := 0; i < 3; i++ {
		receiveTestMessage(t, respCh)
	}
	time.Sleep(20 * time.Millisecond)
	mockTransport.sendMessage(&types.SystemMessage{
		Type:    "system",
		Subtype: "status",
	})

	if msg := receiveTestMessage(t, respCh); msg.GetMessageType() != "system" {
		t.Fatalf("fourth message type = %q, want system", msg.GetMessageType())
	}
	select {
	case _, ok := <-respCh:
		if ok {
			t.Fatal("ReceiveResponse stayed open after unrelated frame followed terminal result")
		}
	case <-time.After(time.Second):
		t.Fatal("ReceiveResponse did not close after unrelated frame followed terminal result")
	}

	if err := client.Close(ctx); err != nil {
		t.Logf("Close returned error (acceptable): %v", err)
	}
}

func TestClient_ReceiveResponseClosesOnResultWithLegacyTaskType(t *testing.T) {
	t.Parallel()

	client, mockTransport := makeConnectedClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	respCh := client.ReceiveResponse(ctx)

	taskType := "agent"
	final := "legacy task completed"
	mockTransport.sendMessage(&types.TaskStartedMessage{
		Type:     "system",
		TaskID:   "task-legacy-1",
		TaskType: &taskType,
	})
	mockTransport.sendMessage(&types.ResultMessage{
		Type:   "result",
		Result: &final,
	})

	if msg := receiveTestMessage(t, respCh); msg.GetMessageType() != "system" {
		t.Fatalf("first message type = %q, want system", msg.GetMessageType())
	}
	result, ok := receiveTestMessage(t, respCh).(*types.ResultMessage)
	if !ok {
		t.Fatal("second message was not ResultMessage")
	}
	if result.Result == nil || *result.Result != final {
		t.Fatalf("final result = %v, want %q", result.Result, final)
	}
	select {
	case _, ok := <-respCh:
		if ok {
			t.Fatal("ReceiveResponse stayed open after result with legacy task type")
		}
	case <-time.After(time.Second):
		t.Fatal("ReceiveResponse did not close after result with legacy task type")
	}

	if err := client.Close(ctx); err != nil {
		t.Logf("Close returned error (acceptable): %v", err)
	}
}

func TestClient_ReceiveResponseKeepsAgentOpenAfterTerminalChildResultUntilFinalResult(t *testing.T) {
	t.Parallel()

	client, mockTransport := makeConnectedClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	respCh := client.ReceiveResponse(ctx)

	taskID := "task-agent-1"
	toolUseID := "toolu_agent_1"
	taskType := "local_agent"
	intermediate := "child agent completed"
	final := "MAIN_RECEIVED: AGENTD_NAMED_AGENT_LIVE_OK"
	mockTransport.sendMessage(&types.TaskStartedMessage{
		Type:      "system",
		TaskID:    taskID,
		ToolUseID: &toolUseID,
		TaskType:  &taskType,
	})
	mockTransport.sendMessage(&types.TaskUpdatedMessage{
		Type:      "system",
		TaskID:    taskID,
		ToolUseID: &toolUseID,
		Patch:     types.TaskUpdatedPatch{Status: "completed"},
	})
	mockTransport.sendMessage(&types.ResultMessage{
		Type:   "result",
		Result: &intermediate,
	})

	if msg := receiveTestMessage(t, respCh); msg.GetMessageType() != "system" {
		t.Fatalf("first message type = %q, want system", msg.GetMessageType())
	}
	if msg := receiveTestMessage(t, respCh); msg.GetMessageType() != "system" {
		t.Fatalf("second message type = %q, want system", msg.GetMessageType())
	}
	if msg := receiveTestMessage(t, respCh); msg.GetMessageType() != "result" {
		t.Fatalf("third message type = %q, want result", msg.GetMessageType())
	}
	select {
	case _, ok := <-respCh:
		if !ok {
			t.Fatal("ReceiveResponse closed on local-agent child result before parent resumed")
		}
		t.Fatal("unexpected extra message before local-agent final result")
	case <-time.After(terminalTaskUpdateCloseGrace + 50*time.Millisecond):
	}

	mockTransport.sendMessage(&types.TaskNotificationMessage{
		Type:      "system",
		TaskID:    taskID,
		ToolUseID: &toolUseID,
		Status:    "completed",
	})
	mockTransport.sendMessage(&types.AssistantMessage{
		Type: "assistant",
		Content: []types.ContentBlock{
			&types.TextBlock{Type: "text", Text: final},
		},
	})
	mockTransport.sendMessage(&types.ResultMessage{
		Type:   "result",
		Result: &final,
	})

	if msg := receiveTestMessage(t, respCh); msg.GetMessageType() != "system" {
		t.Fatalf("fourth message type = %q, want assistant", msg.GetMessageType())
	}
	if msg := receiveTestMessage(t, respCh); msg.GetMessageType() != "assistant" {
		t.Fatalf("fifth message type = %q, want assistant", msg.GetMessageType())
	}
	result, ok := receiveTestMessage(t, respCh).(*types.ResultMessage)
	if !ok {
		t.Fatal("sixth message was not ResultMessage")
	}
	if result.Result == nil || *result.Result != final {
		t.Fatalf("final result = %v, want %q", result.Result, final)
	}
	select {
	case _, ok := <-respCh:
		if ok {
			t.Fatal("ReceiveResponse stayed open after local-agent final result")
		}
	case <-time.After(time.Second):
		t.Fatal("ReceiveResponse did not close after local-agent final result")
	}

	if err := client.Close(ctx); err != nil {
		t.Logf("Close returned error (acceptable): %v", err)
	}
}

func TestClient_ReceiveResponseDoesNotReuseAgentResultForWorkflowClose(t *testing.T) {
	t.Parallel()

	client, mockTransport := makeConnectedClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	respCh := client.ReceiveResponse(ctx)

	agentTaskID := "task-agent-1"
	agentToolUseID := "toolu_agent_1"
	agentTaskType := "local_agent"
	workflowTaskID := "task-workflow-1"
	workflowToolUseID := "toolu_workflow_1"
	workflowTaskType := taskTypeLocalWorkflow
	intermediate := "child agent completed"
	final := "parent final result"
	mockTransport.sendMessage(&types.TaskStartedMessage{
		Type:      "system",
		TaskID:    agentTaskID,
		ToolUseID: &agentToolUseID,
		TaskType:  &agentTaskType,
	})
	mockTransport.sendMessage(&types.ResultMessage{
		Type:   "result",
		Result: &intermediate,
	})
	mockTransport.sendMessage(&types.TaskNotificationMessage{
		Type:      "system",
		TaskID:    agentTaskID,
		ToolUseID: &agentToolUseID,
		Status:    "completed",
	})
	mockTransport.sendMessage(&types.TaskStartedMessage{
		Type:      "system",
		TaskID:    workflowTaskID,
		ToolUseID: &workflowToolUseID,
		TaskType:  &workflowTaskType,
	})
	mockTransport.sendMessage(&types.TaskNotificationMessage{
		Type:      "system",
		TaskID:    workflowTaskID,
		ToolUseID: &workflowToolUseID,
		Status:    "completed",
	})

	for i := 0; i < 5; i++ {
		receiveTestMessage(t, respCh)
	}
	select {
	case _, ok := <-respCh:
		if !ok {
			t.Fatal("ReceiveResponse closed by stale local-agent result after workflow completion")
		}
		t.Fatal("unexpected extra message before parent final result")
	case <-time.After(20 * time.Millisecond):
	}

	mockTransport.sendMessage(&types.ResultMessage{
		Type:   "result",
		Result: &final,
	})
	result, ok := receiveTestMessage(t, respCh).(*types.ResultMessage)
	if !ok {
		t.Fatal("final message was not ResultMessage")
	}
	if result.Result == nil || *result.Result != final {
		t.Fatalf("final result = %v, want %q", result.Result, final)
	}
	select {
	case _, ok := <-respCh:
		if ok {
			t.Fatal("ReceiveResponse stayed open after parent final result")
		}
	case <-time.After(time.Second):
		t.Fatal("ReceiveResponse did not close after parent final result")
	}

	if err := client.Close(ctx); err != nil {
		t.Logf("Close returned error (acceptable): %v", err)
	}
}

func receiveTestMessage(t *testing.T, ch <-chan types.Message) types.Message {
	t.Helper()
	select {
	case msg, ok := <-ch:
		if !ok {
			t.Fatal("response channel closed before expected message")
		}
		return msg
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for response message")
		return nil
	}
}

// TestClient_Close_WaitsForReceiveGoroutines verifies that Close() waits for
// all in-flight ReceiveResponse goroutines to finish before returning.
func TestClient_Close_WaitsForReceiveGoroutines(t *testing.T) {
	t.Parallel()

	client, _ := makeConnectedClient(t)
	ctx := context.Background()

	// Start ReceiveResponse — goroutine blocks on messages channel.
	_ = client.ReceiveResponse(ctx)

	// Close cancels context and waits for goroutines via recvWg.
	closeDone := make(chan error, 1)
	go func() {
		closeDone <- client.Close(ctx)
	}()

	select {
	case err := <-closeDone:
		if err != nil {
			t.Logf("Close returned error (acceptable): %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Close() deadlocked — ReceiveResponse goroutine not canceled or not tracked")
	}
}
