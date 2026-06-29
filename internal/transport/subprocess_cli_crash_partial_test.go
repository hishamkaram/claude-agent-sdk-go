package transport

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestSubprocessCrash_ReadyFalse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "process killed externally"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockProc := newMockSpawnedProcess()
			spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
				return mockProc, nil
			})

			opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
			tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)

			ctx := context.Background()
			if err := tr.Connect(ctx); err != nil {
				t.Fatalf("Connect() failed: %v", err)
			}

			if !tr.IsReady() {
				t.Fatal("transport should be ready before crash")
			}

			// Simulate subprocess crash.
			_ = mockProc.Kill()

			// Watcher goroutine must set ready=false within 2s.
			if !waitForNotReady(tr, 2*time.Second) {
				t.Error("IsReady() should return false after subprocess crash, but it is still true")
			}

			closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = tr.Close(closeCtx)
		})
	}
}

// TestSubprocessCrash_WriteReturnsError verifies that Write() returns a non-nil
// error after the subprocess exits — without touching the closed pipe (no EPIPE).
func TestSubprocessCrash_WriteReturnsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "write after crash returns error"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockProc := newMockSpawnedProcess()
			spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
				return mockProc, nil
			})

			opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
			tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)

			ctx := context.Background()
			if err := tr.Connect(ctx); err != nil {
				t.Fatalf("Connect() failed: %v", err)
			}

			// Crash the subprocess.
			_ = mockProc.Kill()

			// Wait for watcher to mark transport not-ready.
			waitForNotReady(tr, 2*time.Second)

			// Write must return an error — must not attempt to write to the dead pipe.
			err := tr.Write(ctx, `{"type":"user","message":"hello"}`)
			if err == nil {
				t.Error("Write() should return a non-nil error after subprocess crash")
			}

			closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = tr.Close(closeCtx)
		})
	}
}

// TestSubprocessCrash_CloseDoesNotHang verifies that Close() completes within
// a 2s timeout after a spontaneous subprocess exit (no deadlock / double-Wait).
func TestSubprocessCrash_CloseDoesNotHang(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "close after spontaneous exit completes within timeout"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockProc := newMockSpawnedProcess()
			spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
				return mockProc, nil
			})

			opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
			tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)

			ctx := context.Background()
			if err := tr.Connect(ctx); err != nil {
				t.Fatalf("Connect() failed: %v", err)
			}

			// Simulate spontaneous subprocess exit.
			_ = mockProc.Kill()

			// Wait for watcher to detect exit (ensures procDone is closed before Close).
			waitForNotReady(tr, 2*time.Second)

			// Close() must complete within 2s — must not hang or deadlock.
			closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := tr.Close(closeCtx); closeCtx.Err() != nil {
				t.Errorf("Close() hung: context expired before Close() returned (err=%v)", err)
			}
		})
	}
}

// TestSubprocessCrash_PartialStdoutWithoutNewlineClosesMessages verifies that
// an exited process cannot leave the reader goroutine blocked forever on an
// incomplete stdout JSON line.
func TestSubprocessCrash_PartialStdoutWithoutNewlineClosesMessages(t *testing.T) {
	t.Parallel()

	mockProc := newMockSpawnedProcess()
	tr := newMockTransportWithProcess(t, mockProc, nil)

	t.Cleanup(func() {
		_ = mockProc.stdoutW.Close()
		_ = mockProc.stderrW.Close()
		closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tr.Close(closeCtx)
	})

	if _, err := mockProc.stdoutW.Write([]byte(`{"type":"system","subtype":"init"`)); err != nil {
		t.Fatalf("failed to write partial stdout JSON: %v", err)
	}

	mockProc.signalExitWithStdoutOpen()

	select {
	case _, ok := <-tr.messages:
		if ok {
			t.Fatal("received an unexpected message from partial stdout JSON")
		}
	case <-time.After(2 * time.Second):
		_ = mockProc.stdoutW.Close()
		t.Fatal("messages channel stayed open after process exit with partial stdout JSON")
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := tr.Close(closeCtx); closeCtx.Err() != nil {
		t.Fatalf("Close() hung after process exit with partial stdout JSON: %v", err)
	}
}

func TestMessageReaderLoop_ProcDoneClosesPartialStdout(t *testing.T) {
	t.Parallel()

	stdoutR, stdoutW := io.Pipe()
	procDone := make(chan struct{})
	tr := &SubprocessCLITransport{
		messages: make(chan types.Message, 1),
		logger:   log.NewLogger(false),
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		_ = stdoutW.Close()
		_ = stdoutR.Close()
	})

	loopDone := make(chan struct{})
	go func() {
		tr.messageReaderLoop(ctx, stdoutR, procDone)
		close(loopDone)
	}()

	writeDone := make(chan error, 1)
	go func() {
		_, err := stdoutW.Write([]byte(`{"type":"system","subtype":"init"`))
		writeDone <- err
	}()

	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("failed to write partial stdout JSON: %v", err)
		}
	case <-time.After(1 * time.Second):
		close(procDone)
		t.Fatal("timed out writing partial stdout JSON")
	}

	close(procDone)

	select {
	case <-loopDone:
	case <-time.After(2 * time.Second):
		t.Fatal("messageReaderLoop stayed blocked after procDone closed")
	}

	select {
	case _, ok := <-tr.messages:
		if ok {
			t.Fatal("received an unexpected message from partial stdout JSON")
		}
	default:
		t.Fatal("messages channel was not closed after messageReaderLoop returned")
	}
}

// TestSubprocessCrash_NoGoroutineLeak verifies that no goroutines are leaked
// after a subprocess crash followed by Close(). goleak.VerifyTestMain in
// TestMain catches any leaks across the entire test suite as well.
func TestSubprocessCrash_NoGoroutineLeak(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "crash then close leaks no goroutines"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockProc := newMockSpawnedProcess()
			spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
				return mockProc, nil
			})

			opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
			tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)

			ctx := context.Background()
			if err := tr.Connect(ctx); err != nil {
				t.Fatalf("Connect() failed: %v", err)
			}

			// Crash the subprocess.
			_ = mockProc.Kill()

			// Wait for watcher goroutine to complete.
			waitForNotReady(tr, 2*time.Second)

			// Close the transport — all goroutines must exit.
			closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = tr.Close(closeCtx)

			// goleak.VerifyTestMain catches any remaining goroutines for the whole suite.
			// This test ensures the specific crash+close lifecycle runs cleanly.
		})
	}
}

// ===== Bug C15: Parse error backoff tests =====
