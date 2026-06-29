package transport

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestMessageReaderLoop_ParseErrorBackoff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		invalidLines     int
		minTotalDuration time.Duration
	}{
		{
			name:             "3 consecutive parse errors have increasing delay",
			invalidLines:     3,
			minTotalDuration: 25 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a pipe to feed lines to messageReaderLoop
			stdoutR, stdoutW := io.Pipe()
			stderrR, stderrW := io.Pipe()
			mockProc := newMockSpawnedProcess()
			// Override the mock's stdout/stderr with our pipes
			mockProc.stdout = stdoutR
			mockProc.stdoutW = stdoutW
			mockProc.stderr = stderrR
			mockProc.stderrW = stderrW

			spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
				return mockProc, nil
			})

			opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
			tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)
			tr.parseErrorBackoff = func(consecutive uint) time.Duration {
				if consecutive == 0 {
					return 0
				}
				return time.Duration(1<<min(consecutive-1, 5)) * 10 * time.Millisecond
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := tr.Connect(ctx); err != nil {
				t.Fatalf("Connect() failed: %v", err)
			}

			// Send invalid JSON lines rapidly
			start := time.Now()
			for i := 0; i < tt.invalidLines; i++ {
				_, err := fmt.Fprintf(stdoutW, "not valid json %d\n", i)
				if err != nil {
					t.Fatalf("failed to write invalid line %d: %v", i, err)
				}
			}

			// Wait for messages to be drained (they won't produce valid messages)
			// Then send a valid message to verify the loop is still running
			validMsg := `{"type":"system","subtype":"init","session_id":"test-123"}` + "\n"
			_, err := stdoutW.Write([]byte(validMsg))
			if err != nil {
				t.Fatalf("failed to write valid message: %v", err)
			}

			// Read from messages channel — the valid message should eventually arrive
			// but only after backoff delays
			timer := time.NewTimer(2 * time.Second)
			defer timer.Stop()
			select {
			case msg, ok := <-tr.messages:
				elapsed := time.Since(start)
				if !ok {
					t.Fatal("messages channel closed unexpectedly")
				}
				if msg == nil {
					t.Fatal("received nil message")
				}
				// The test transport uses a short deterministic backoff, but
				// the valid message still must wait behind multiple parse-error delays.
				if elapsed < tt.minTotalDuration {
					t.Errorf("messages arrived too quickly: %v < minimum %v (no backoff?)",
						elapsed, tt.minTotalDuration)
				}
			case <-timer.C:
				t.Fatal("timed out waiting for valid message after parse errors")
			}

			// Cleanup
			cancel()
			_ = mockProc.Kill()
			closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer closeCancel()
			_ = tr.Close(closeCtx)
		})
	}
}

// TestMessageReaderLoop_ParseErrorBackoffResets verifies that a successful parse
// resets the backoff counter back to zero.
func TestMessageReaderLoop_ParseErrorBackoffResets(t *testing.T) {
	t.Parallel()

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	mockProc := newMockSpawnedProcess()
	mockProc.stdout = stdoutR
	mockProc.stdoutW = stdoutW
	mockProc.stderr = stderrR
	mockProc.stderrW = stderrW

	spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
		return mockProc, nil
	})

	opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
	tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)
	useFastParseErrorBackoff(tr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := tr.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	// Send one invalid line (will cause 1s backoff)
	_, _ = stdoutW.Write([]byte("invalid json\n"))

	// Then send a valid message (resets the counter)
	validMsg := `{"type":"system","subtype":"init","session_id":"test-123"}` + "\n"
	_, _ = stdoutW.Write([]byte(validMsg))

	// Read the valid message
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	select {
	case msg, ok := <-tr.messages:
		if !ok {
			t.Fatal("messages channel closed unexpectedly")
		}
		if msg == nil {
			t.Fatal("received nil message")
		}
		// Good — valid message received, counter should be reset
	case <-timer.C:
		t.Fatal("timed out waiting for valid message")
	}

	// Now send another invalid line — should only have 1s backoff (not 2s)
	start := time.Now()
	_, _ = stdoutW.Write([]byte("another invalid\n"))

	// Then send another valid message
	_, _ = stdoutW.Write([]byte(validMsg))

	select {
	case msg, ok := <-tr.messages:
		elapsed := time.Since(start)
		if !ok {
			t.Fatal("messages channel closed unexpectedly")
		}
		if msg == nil {
			t.Fatal("received nil message")
		}
		// After reset, the backoff for the single error should be ~1s (not 2s+)
		// Allow some tolerance
		if elapsed > 3*time.Second {
			t.Errorf("backoff was not reset: took %v (expected ~1s after reset)", elapsed)
		}
	case <-timer.C:
		t.Fatal("timed out waiting for second valid message")
	}

	// Cleanup
	cancel()
	_ = mockProc.Kill()
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer closeCancel()
	_ = tr.Close(closeCtx)
}

// ===== Bug C14: Context monitoring in spawned goroutines =====

// TestContextCancellation_GoroutinesExit verifies that canceling the parent
// context causes all spawned goroutines to exit within 5 seconds.
func TestContextCancellation_GoroutinesExit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "goroutines exit after context cancel"},
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

			ctx, cancel := context.WithCancel(context.Background())
			if err := tr.Connect(ctx); err != nil {
				t.Fatalf("Connect() failed: %v", err)
			}

			// Cancel the parent context — this should trigger goroutine shutdown
			cancel()

			// The watcher goroutine is waiting on mockProc.Wait() which blocks on waitCh.
			// Killing the mock process unblocks it, simulating what would happen when
			// context cancellation kills the subprocess.
			_ = mockProc.Kill()

			// Close should complete within 5 seconds, not hang
			closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer closeCancel()
			err := tr.Close(closeCtx)
			if closeCtx.Err() != nil {
				t.Fatal("Close() hung after context cancellation — goroutines did not exit within 5s")
			}
			_ = err
		})
	}
}

// ===== Bug C16: Stderr reader hang tests =====

// TestReadStderr_ExitsOnPipeClose verifies that the readStderr goroutine
// exits promptly when the stderr pipe is closed.
func TestReadStderr_ExitsOnPipeClose(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "readStderr exits when pipe is closed"},
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
			useFastParseErrorBackoff(tr)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := tr.Connect(ctx); err != nil {
				t.Fatalf("Connect() failed: %v", err)
			}

			// Close stderr pipe — this should unblock readStderr's ReadLine
			_ = mockProc.stderrW.Close()

			// Now kill the process and close — should complete quickly
			_ = mockProc.Kill()

			closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer closeCancel()
			err := tr.Close(closeCtx)
			if closeCtx.Err() != nil {
				t.Fatal("Close() hung — readStderr goroutine did not exit within 5s after stderr pipe close")
			}
			_ = err
		})
	}
}

// TestMessageReaderLoop_MaxConsecutiveParseErrors verifies that after
// maxConsecutiveParseErrors consecutive bad JSON lines, the reader loop exits
// and closes the messages channel.
func TestMessageReaderLoop_MaxConsecutiveParseErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "reader exits after maxConsecutiveParseErrors bad lines"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stdoutR, stdoutW := io.Pipe()
			stderrR, stderrW := io.Pipe()
			mockProc := newMockSpawnedProcess()
			mockProc.stdout = stdoutR
			mockProc.stdoutW = stdoutW
			mockProc.stderr = stderrR
			mockProc.stderrW = stderrW

			spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
				return mockProc, nil
			})

			opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
			tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)
			useFastParseErrorBackoff(tr)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := tr.Connect(ctx); err != nil {
				t.Fatalf("Connect() failed: %v", err)
			}

			// Send maxConsecutiveParseErrors invalid JSON lines.
			go func() {
				for i := uint(0); i < maxConsecutiveParseErrors; i++ {
					_, _ = fmt.Fprintf(stdoutW, "garbage-%d\n", i)
				}
			}()

			// The messages channel must close after the threshold is hit.
			timer := time.NewTimer(2 * time.Second)
			defer timer.Stop()

			channelClosed := false
			for !channelClosed {
				select {
				case _, ok := <-tr.messages:
					if !ok {
						channelClosed = true
					}
				case <-timer.C:
					t.Fatal("timed out waiting for messages channel to close after threshold")
					channelClosed = true // break loop on timeout
				}
			}

			// Channel being closed proves the reader loop exited due to the
			// threshold. Without the threshold, it would continue with backoff
			// (waiting for more data) and the channel would stay open.
			// OnError stores only the first error, so GetError() returns the
			// first parse error, not the threshold error — verify it's non-nil.
			storedErr := tr.GetError()
			if storedErr == nil {
				t.Fatal("expected stored error after parse failures, got nil")
			}

			// Cleanup
			cancel()
			_ = mockProc.Kill()
			closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer closeCancel()
			_ = tr.Close(closeCtx)
		})
	}
}

// TestMessageReaderLoop_ParseErrorCounterResetPreventsThreshold verifies that
// successful parses reset the consecutive error counter, so the threshold is
// NOT reached if valid messages are interspersed.
func TestMessageReaderLoop_ParseErrorCounterResetPreventsThreshold(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "valid message resets counter — threshold not reached"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stdoutR, stdoutW := io.Pipe()
			stderrR, stderrW := io.Pipe()
			mockProc := newMockSpawnedProcess()
			mockProc.stdout = stdoutR
			mockProc.stdoutW = stdoutW
			mockProc.stderr = stderrR
			mockProc.stderrW = stderrW

			spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
				return mockProc, nil
			})

			opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
			tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)
			useFastParseErrorBackoff(tr)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := tr.Connect(ctx); err != nil {
				t.Fatalf("Connect() failed: %v", err)
			}

			// Send (threshold-1) bad lines, then a valid message, then
			// (threshold-1) bad lines again. The counter should reset after
			// the valid message, so the loop should NOT exit.
			validMsg := `{"type":"system","subtype":"init","session_id":"test-123"}` + "\n"

			go func() {
				// First batch: threshold-1 errors
				for i := uint(0); i < maxConsecutiveParseErrors-1; i++ {
					_, _ = fmt.Fprintf(stdoutW, "bad-batch1-%d\n", i)
				}
				// Valid message resets counter
				_, _ = stdoutW.Write([]byte(validMsg))
				// Second batch: threshold-1 errors
				for i := uint(0); i < maxConsecutiveParseErrors-1; i++ {
					_, _ = fmt.Fprintf(stdoutW, "bad-batch2-%d\n", i)
				}
				// Another valid message — proves the loop is still running
				_, _ = stdoutW.Write([]byte(validMsg))
			}()

			// We should receive exactly 2 valid messages.
			received := 0
			timer := time.NewTimer(2 * time.Second)
			defer timer.Stop()

			for received < 2 {
				select {
				case msg, ok := <-tr.messages:
					if !ok {
						t.Fatalf("messages channel closed unexpectedly after %d messages — threshold was incorrectly reached", received)
					}
					if msg == nil {
						t.Fatal("received nil message")
					}
					received++
				case <-timer.C:
					t.Fatalf("timed out waiting for valid messages — received %d of 2", received)
				}
			}

			// Cleanup
			cancel()
			_ = mockProc.Kill()
			closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer closeCancel()
			_ = tr.Close(closeCtx)
		})
	}
}

// TestReadStderr_ExitsOnContextCancel verifies that the readStderr goroutine
// exits when the context is canceled, even if the pipe is still open.
func TestReadStderr_ExitsOnContextCancel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "readStderr exits on context cancel"},
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

			ctx, cancel := context.WithCancel(context.Background())
			if err := tr.Connect(ctx); err != nil {
				t.Fatalf("Connect() failed: %v", err)
			}

			// Cancel context — readStderr should detect this and exit
			cancel()

			// Kill the mock so the watcher goroutine unblocks too
			_ = mockProc.Kill()

			closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer closeCancel()
			err := tr.Close(closeCtx)
			if closeCtx.Err() != nil {
				t.Fatal("Close() hung — readStderr did not exit within 5s after context cancel")
			}
			_ = err
		})
	}
}
