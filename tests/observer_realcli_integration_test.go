//go:build integration

package tests

import (
	"sync/atomic"
	"testing"
	"time"

	claude "github.com/hishamkaram/claude-agent-sdk-go"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// realObserver records connect telemetry from a live CLI connection. Concurrency-safe
// because the SDK calls Observer methods from transport goroutines.
type realObserver struct {
	types.NopObserver
	connects   atomic.Uint64
	connectErr atomic.Bool
}

func (o *realObserver) OnConnect(_ time.Duration, err error) {
	o.connects.Add(1)
	if err != nil {
		o.connectErr.Store(true)
	}
}

// TestObserver_RealCLI_OnConnect connects to the REAL Claude CLI (no model turn, so
// no tokens spent) and asserts the Observer's OnConnect fires exactly once with no
// error, and that Health() reports the live subprocess. This is a real-peer test:
// real producer (the claude binary), real transport (subprocess stdio).
func TestObserver_RealCLI_OnConnect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	requireAuth(t)
	cliPath := requireClaude(t)

	checkGoroutines := AssertNoGoroutineLeaks(t)
	defer checkGoroutines()

	ctx, cancel := CreateTestContext(t, 30*time.Second)
	defer cancel()

	obs := &realObserver{}
	opts := types.NewClaudeAgentOptions().
		WithCLIPath(cliPath).
		WithObserver(obs)

	client, err := claude.NewClient(ctx, opts)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	defer func() { _ = client.Close(ctx) }()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	if got := obs.connects.Load(); got != 1 {
		t.Fatalf("OnConnect fired %d times against the real CLI, want exactly 1", got)
	}
	if obs.connectErr.Load() {
		t.Fatal("OnConnect reported an error on a successful real-CLI connect")
	}

	if h := client.Health(); !h.Connected {
		t.Fatalf("Health().Connected=false after a successful real connect: %+v", h)
	}
}

// TestCapabilityContract_RealCLI_NoFlagCrash proves the B3 capability gate against the
// real CLI: --agent-progress-summaries is absent from the installed binary, so
// requesting it must be skipped (Connect succeeds) rather than emitted (Connect
// crashes with "unknown option"). Real producer + real transport, no model turn.
func TestCapabilityContract_RealCLI_NoFlagCrash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	requireAuth(t)
	cliPath := requireClaude(t)

	checkGoroutines := AssertNoGoroutineLeaks(t)
	defer checkGoroutines()

	ctx, cancel := CreateTestContext(t, 30*time.Second)
	defer cancel()

	opts := types.NewClaudeAgentOptions().
		WithCLIPath(cliPath).
		WithAgentProgressSummaries(true)

	client, err := claude.NewClient(ctx, opts)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	defer func() { _ = client.Close(ctx) }()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed with an unsupported experimental flag requested — the "+
			"capability gate should have skipped it, not crashed: %v", err)
	}
}
