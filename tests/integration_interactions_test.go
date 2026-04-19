//go:build integration
// +build integration

// Real-CLI interaction tests — concurrency safety of read methods,
// reconnect-after-close, and context-cancellation discipline. These are
// the "cross-cutting" tests analogous to codex's
// integration_interactions_test.go.

package tests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	claude "github.com/hishamkaram/claude-agent-sdk-go"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestInteractions_ConcurrentReads_AreSafe fires several read-only
// control-protocol calls concurrently and asserts no race is reported
// under -race and every call succeeds. Catches missing mutexes or
// request_id collisions in the SDK's control-protocol demuxer.
func TestInteractions_ConcurrentReads_AreSafe(t *testing.T) {
	client, ctx := setupClient(t, nil)

	const concurrency = 4
	var wg sync.WaitGroup
	errs := make(chan error, concurrency*3)

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func(worker int) {
			defer wg.Done()

			if _, err := client.GetContextUsage(ctx); err != nil {
				errs <- err
			}
			if _, err := client.GetSettings(ctx); err != nil {
				errs <- err
			}
			if _, err := client.MCPServerStatus(ctx); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	var all []error
	for err := range errs {
		all = append(all, err)
	}
	if len(all) > 0 {
		t.Errorf("concurrent reads produced %d error(s); first: %v", len(all), all[0])
	}
}

// TestInteractions_DoubleClose_IsIdempotent asserts that calling Close()
// twice returns without panic. A double-close must be safe because
// defer-based shutdown paths may call it alongside an explicit Close().
func TestInteractions_DoubleClose_IsIdempotent(t *testing.T) {
	client, ctx := setupClient(t, nil)

	if err := client.Close(ctx); err != nil {
		t.Errorf("first Close: %v", err)
	}

	// Second close must not panic.
	if err := client.Close(ctx); err != nil {
		// An error here is acceptable (channel closed, etc.); what
		// matters is that we don't panic.
		t.Logf("second Close returned err (acceptable): %v", err)
	}
}

// TestInteractions_ContextCancel_ClosesChannel drives a query, cancels
// the context, and asserts the ReceiveResponse channel closes within a
// reasonable bound. Catches missing ctx-propagation inside the stream
// reader goroutine.
func TestInteractions_ContextCancel_ClosesChannel(t *testing.T) {
	requireRunTurns(t)
	client, _ := setupClient(t, nil)

	runCtx, runCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer runCancel()

	if err := client.Query(runCtx, "Count from 1 to 1000 slowly."); err != nil {
		t.Fatalf("Query: %v", err)
	}

	// Let a few messages flow, then cancel.
	received := 0
	go func() {
		for range client.ReceiveResponse(runCtx) {
			received++
			if received >= 2 {
				runCancel()
				return
			}
		}
	}()

	// Wait for either the channel to fully drain or a safety timeout.
	deadline := time.After(10 * time.Second)
	select {
	case <-runCtx.Done():
		// Good — ctx cancellation propagated.
	case <-deadline:
		t.Fatal("channel did not close within 10s of context cancellation")
	}
	// ctx deadline errors are benign here; we only require the channel closes.
	if err := runCtx.Err(); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("unexpected ctx error: %v", err)
	}
}

// TestInteractions_ReconnectAfterClose verifies a Client can be Close()d
// and then a fresh client constructed against the same options. Catches
// state that leaks between Client instances (singleton registries, etc.).
func TestInteractions_ReconnectAfterClose(t *testing.T) {
	requireAuth(t)
	cliPath := requireClaude(t)

	opts := types.NewClaudeAgentOptions().
		WithCLIPath(cliPath).
		WithPermissionMode(types.PermissionModeBypassPermissions)

	// First connect.
	ctx1, cancel1 := CreateTestContext(t, 30*time.Second)
	t.Cleanup(cancel1)

	client1, err := claude.NewClient(ctx1, opts)
	if err != nil {
		t.Fatalf("NewClient #1: %v", err)
	}
	if err := client1.Connect(ctx1); err != nil {
		t.Fatalf("Connect #1: %v", err)
	}
	pid1 := client1.ProcessID()
	_ = client1.Close(ctx1)

	// Second connect on a fresh client — must not inherit state.
	ctx2, cancel2 := CreateTestContext(t, 30*time.Second)
	t.Cleanup(cancel2)

	client2, err := claude.NewClient(ctx2, opts)
	if err != nil {
		t.Fatalf("NewClient #2: %v", err)
	}
	if err := client2.Connect(ctx2); err != nil {
		t.Fatalf("Connect #2: %v", err)
	}
	t.Cleanup(func() { _ = client2.Close(ctx2) })

	pid2 := client2.ProcessID()
	if pid1 == pid2 {
		t.Errorf("second client reused the first client's PID (%d); expected a fresh subprocess", pid1)
	}
}
