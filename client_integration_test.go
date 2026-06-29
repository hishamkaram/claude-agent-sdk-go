package claude

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestClient_Integration is an integration test that requires Claude CLI to be installed.
func TestClient_Integration(t *testing.T) {
	t.Parallel()
	// This test requires actual Claude CLI and API key
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration test (set RUN_INTEGRATION_TESTS=1 to run)")
	}

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := types.NewClaudeAgentOptions().
		WithModel("claude-3-5-sonnet-latest").
		WithPermissionMode(types.PermissionModeBypassPermissions)

	client, err := NewClient(ctx, opts)
	if err != nil {
		if types.IsCLINotFoundError(err) {
			t.Skip("Claude CLI not installed")
		}
		t.Fatal(err)
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	// Connect
	if err := client.Connect(ctx); err != nil {
		if types.IsCLIConnectionError(err) {
			t.Skip("Could not connect to Claude CLI")
		}
		t.Fatal(err)
	}

	// First query
	if err := client.Query(ctx, "What is 2+2? Reply with just the number."); err != nil {
		t.Fatal(err)
	}

	// Receive response
	var messageCount int
	for msg := range client.ReceiveResponse(ctx) {
		messageCount++
		if _, ok := msg.(*types.ResultMessage); ok {
			break
		}
	}

	if messageCount == 0 {
		t.Fatal("expected at least one message")
	}

	t.Logf("First query received %d messages", messageCount)

	// Second query in same session
	if err := client.Query(ctx, "What is 3+3? Reply with just the number."); err != nil {
		t.Fatal(err)
	}

	// Receive second response
	messageCount = 0
	for msg := range client.ReceiveResponse(ctx) {
		messageCount++
		if _, ok := msg.(*types.ResultMessage); ok {
			break
		}
	}

	if messageCount == 0 {
		t.Fatal("expected at least one message in second query")
	}

	t.Logf("Second query received %d messages", messageCount)
}

// TestClient_MultipleQueries tests multiple query/response cycles
func TestClient_MultipleQueries(t *testing.T) {
	t.Parallel()
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration test (set RUN_INTEGRATION_TESTS=1 to run)")
	}
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Long-lived connection context covers NewClient + Connect + Close. Each
	// query gets its OWN timeout below: the previous single 60s budget shared
	// across three sequential live queries flaked when the real API was slow on
	// a later query (an early slow response starved the last one). Per-query
	// budgets remove that cross-query coupling.
	connCtx, connCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer connCancel()

	opts := types.NewClaudeAgentOptions().
		WithModel("claude-3-5-sonnet-latest").
		WithPermissionMode(types.PermissionModeBypassPermissions)

	client, err := NewClient(connCtx, opts)
	if err != nil {
		if types.IsCLINotFoundError(err) {
			t.Skip("Claude CLI not installed")
		}
		t.Fatal(err)
	}
	defer func() {
		_ = client.Close(connCtx)
	}()

	if err := client.Connect(connCtx); err != nil {
		if types.IsCLIConnectionError(err) {
			t.Skip("Could not connect to Claude CLI")
		}
		t.Fatal(err)
	}

	// Send 3 queries in sequence
	queries := []string{
		"Say 'first'",
		"Say 'second'",
		"Say 'third'",
	}

	for i, prompt := range queries {
		// Per-query deadline so a slow earlier query cannot starve a later one.
		qCtx, qCancel := context.WithTimeout(connCtx, 90*time.Second)

		if err := client.Query(qCtx, prompt); err != nil {
			qCancel()
			t.Fatalf("Query %d failed: %v", i+1, err)
		}

		// Receive response
		gotResult := false
		for msg := range client.ReceiveResponse(qCtx) {
			if _, ok := msg.(*types.ResultMessage); ok {
				gotResult = true
				break
			}
		}
		// Cancel after the cycle to release the ReceiveResponse forwarding
		// goroutine (we break early on the ResultMessage).
		qCancel()

		if !gotResult {
			t.Fatalf("Query %d did not receive ResultMessage", i+1)
		}

		t.Logf("Query %d completed", i+1)
	}
}
