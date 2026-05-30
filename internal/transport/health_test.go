package transport

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestTransportHealth_ReflectsLifecycle proves Health() is an authoritative snapshot
// of subprocess state: not connected before Connect, connected+ready after, and the
// last error surfaced after the subprocess dies unexpectedly.
func TestTransportHealth_ReflectsLifecycle(t *testing.T) {
	t.Parallel()

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	mockProc := newMockSpawnedProcess()
	mockProc.stdout = stdoutR
	mockProc.stdoutW = stdoutW
	mockProc.stderr = stderrR
	mockProc.stderrW = stderrW
	// Simulate an unexpected crash exit (set before Connect to avoid racing the
	// watcher's Wait()) so the watcher records a transport error.
	mockProc.exitCode = 1
	mockProc.waitErr = errors.New("simulated crash: exit status 1")

	spawner := types.ProcessSpawner(func(_ context.Context, _ types.SpawnOptions) (types.SpawnedProcess, error) {
		return mockProc, nil
	})
	opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
	tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)

	// Before Connect: not connected, not ready, no error.
	if h := tr.Health(); h.Connected || h.Ready || h.LastError != nil {
		t.Fatalf("pre-connect Health = %+v, want zero", h)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := tr.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	// After Connect: connected and ready.
	if h := tr.Health(); !h.Connected || !h.Ready {
		t.Fatalf("post-connect Health = %+v, want Connected && Ready", h)
	}

	// Kill the subprocess unexpectedly → the watcher records the error.
	_ = mockProc.Kill()
	got := false
	for i := 0; i < 300 && !got; i++ {
		if h := tr.Health(); !h.Ready && h.LastError != nil {
			got = true
			break
		}
		time.Sleep(10 * time.Millisecond) // Sleep: bounded poll for async watcher
	}
	if !got {
		t.Fatalf("after kill, Health never reported not-ready with an error: %+v", tr.Health())
	}

	cancel()
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer closeCancel()
	_ = tr.Close(closeCtx)
}
