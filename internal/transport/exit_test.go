package transport

import (
	"context"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

type exitObserver struct {
	types.NopObserver
	calls    atomic.Uint64
	lastCode atomic.Int64
}

func (e *exitObserver) OnSubprocessExit(code int, _ bool, _ error) {
	e.calls.Add(1)
	e.lastCode.Store(int64(code))
}

// TestTransportEmitsSubprocessExit proves the watcher emits OnSubprocessExit exactly
// once when the subprocess dies — the signal agentd needs for the subprocess-alive
// gauge and the restart counter.
func TestTransportEmitsSubprocessExit(t *testing.T) {
	t.Parallel()

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	mockProc := newMockSpawnedProcess()
	mockProc.stdout = stdoutR
	mockProc.stdoutW = stdoutW
	mockProc.stderr = stderrR
	mockProc.stderrW = stderrW

	spawner := types.ProcessSpawner(func(_ context.Context, _ types.SpawnOptions) (types.SpawnedProcess, error) {
		return mockProc, nil
	})

	obs := &exitObserver{}
	opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner).WithObserver(obs)
	tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := tr.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	// Kill the subprocess — the watcher observes Wait() returning and emits exit.
	_ = mockProc.Kill()

	// Poll with a deadline (allowed by anti-flaky rules) for the single emission.
	emitted := false
	for i := 0; i < 300 && !emitted; i++ {
		if obs.calls.Load() >= 1 {
			emitted = true
			break
		}
		time.Sleep(10 * time.Millisecond) // Sleep: bounded poll for async watcher emission
	}
	if !emitted {
		t.Fatal("OnSubprocessExit never fired after the subprocess was killed")
	}

	cancel()
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer closeCancel()
	_ = tr.Close(closeCtx)

	// Exactly one emission across the whole lifecycle (no double-emit from Close()).
	if got := obs.calls.Load(); got != 1 {
		t.Fatalf("OnSubprocessExit fired %d times, want exactly 1", got)
	}
}
