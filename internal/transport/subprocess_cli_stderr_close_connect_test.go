package transport

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestSubprocessCLITransportStderrNoNewlineDrainsBeyondScannerLimit(t *testing.T) {
	t.Parallel()

	mockProc := newMockSpawnedProcess()
	tr := newMockTransportWithProcess(t, mockProc, nil)
	t.Cleanup(func() {
		_ = mockProc.stderr.Close()
		_ = mockProc.Kill()
		closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tr.Close(closeCtx)
	})

	payload := strings.Repeat("N", DefaultMaxBufferSize+StderrRingSize+1024)
	want := payload[len(payload)-StderrRingSize:]
	writeDone := make(chan error, 1)
	go func() {
		_, err := mockProc.stderrW.Write([]byte(payload))
		writeDone <- err
	}()

	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("write stderr: %v", err)
		}
	case <-time.After(2 * time.Second):
		_ = mockProc.stderr.Close()
		t.Fatal("stderr write without newline blocked; readStderr stopped draining")
	}

	waitForStderr(t, tr, want)
	mockProc.signalExit()

	closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := tr.Close(closeCtx); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}
	if closeCtx.Err() != nil {
		t.Fatal("Close() timed out after draining large stderr without newline")
	}
	if got := tr.Stderr(); got != want {
		t.Fatalf("Stderr() after close = len %d, want len %d", len(got), len(want))
	}
}

func TestSubprocessCLITransportStderrTailRetainsLast64KiB(t *testing.T) {
	t.Parallel()

	mockProc := newMockSpawnedProcess()
	tr := newMockTransportWithProcess(t, mockProc, nil)

	payload := strings.Repeat("A", StderrRingSize+17)
	want := payload[len(payload)-StderrRingSize:]
	if _, err := mockProc.stderrW.Write([]byte(payload + "\n")); err != nil {
		t.Fatalf("write stderr: %v", err)
	}

	waitForStderr(t, tr, want)
	mockProc.signalExit()
	waitForNotReady(tr, 2*time.Second)

	closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := tr.Close(closeCtx); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}
	if got := tr.Stderr(); got != want {
		t.Fatalf("Stderr() after close changed: len=%d want len=%d", len(got), len(want))
	}
}

func TestSubprocessCLITransportStderrLinesPreserveCallbackLogDiagnosticsAndParsing(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	logPath := tempDir + "/stderr.log"
	var (
		mu       sync.Mutex
		callback []string
	)
	opts := types.NewClaudeAgentOptions().
		WithCustomStderrLogFile(logPath).
		WithStderr(func(line string) {
			mu.Lock()
			defer mu.Unlock()
			callback = append(callback, line)
		})
	mockProc := newMockSpawnedProcess()
	tr := newMockTransportWithProcess(t, mockProc, opts)

	lines := []string{
		"first diagnostic line",
		"No conversation found with session ID: session-123",
	}
	if _, err := mockProc.stderrW.Write([]byte(strings.Join(lines, "\n") + "\n")); err != nil {
		t.Fatalf("write stderr: %v", err)
	}
	waitForStderrContains(t, tr, lines...)
	waitForCondition(t, 2*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(callback) == len(lines) && callback[0] == lines[0] && callback[1] == lines[1]
	})
	waitForCondition(t, 2*time.Second, func() bool {
		data, err := os.ReadFile(logPath)
		if err != nil {
			return false
		}
		logText := string(data)
		return strings.Contains(logText, lines[0]) && strings.Contains(logText, lines[1])
	})
	waitForCondition(t, 2*time.Second, func() bool {
		err := tr.GetError()
		var sessionErr *types.SessionNotFoundError
		return errors.As(err, &sessionErr) && sessionErr.SessionID == "session-123"
	})

	mockProc.signalExit()
	closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := tr.Close(closeCtx); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}
}

func TestSubprocessCLITransportUnexpectedExitIncludesStderrTail(t *testing.T) {
	t.Parallel()

	mockProc := newMockSpawnedProcess()
	mockProc.exitCode = 7
	mockProc.waitErr = errors.New("exit status 7")
	tr := newMockTransportWithProcess(t, mockProc, nil)

	const stderrTail = "deterministic stderr tail"
	if _, err := mockProc.stderrW.Write([]byte(stderrTail + "\n")); err != nil {
		t.Fatalf("write stderr: %v", err)
	}
	waitForStderr(t, tr, stderrTail)

	mockProc.signalExit()
	if !waitForNotReady(tr, 2*time.Second) {
		t.Fatal("transport did not observe subprocess exit")
	}

	err := tr.GetError()
	var procErr *types.ProcessError
	if !errors.As(err, &procErr) {
		t.Fatalf("GetError() = %T %v, want *types.ProcessError", err, err)
	}
	if procErr.ExitCode != 7 {
		t.Fatalf("ProcessError.ExitCode = %d, want 7", procErr.ExitCode)
	}
	if got := err.Error(); !strings.Contains(got, stderrTail) || !strings.Contains(got, "exit code: 7") {
		t.Fatalf("ProcessError message = %q, want exit code and stderr tail", got)
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = tr.Close(closeCtx)
}

func TestSubprocessCLITransportStderrCallbackFileAndDiagnostics(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	logPath := tempDir + "/stderr.log"
	var (
		mu       sync.Mutex
		callback []string
	)
	opts := types.NewClaudeAgentOptions().
		WithCustomStderrLogFile(logPath).
		WithStderr(func(line string) {
			mu.Lock()
			defer mu.Unlock()
			callback = append(callback, line)
		})
	mockProc := newMockSpawnedProcess()
	tr := newMockTransportWithProcess(t, mockProc, opts)

	const line = "callback file diagnostic line"
	if _, err := mockProc.stderrW.Write([]byte(line + "\n")); err != nil {
		t.Fatalf("write stderr: %v", err)
	}
	waitForStderr(t, tr, line)
	waitForCondition(t, 2*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(callback) == 1 && callback[0] == line
	})
	waitForCondition(t, 2*time.Second, func() bool {
		data, err := os.ReadFile(logPath)
		return err == nil && strings.Contains(string(data), line)
	})

	mockProc.signalExit()
	closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := tr.Close(closeCtx); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}
}

func TestSubprocessCLITransportCloseGracefulEOF(t *testing.T) {
	t.Parallel()

	mockProc := newMockSpawnedProcess()
	mockProc.stdin.onClose = mockProc.signalExit
	tr := newMockTransportWithProcess(t, mockProc, nil)

	closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := tr.Close(closeCtx); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}
	if mockProc.Killed() {
		t.Fatal("Close() killed process that exited after stdin EOF")
	}
}

func TestSubprocessCLITransportCloseIdempotent(t *testing.T) {
	t.Parallel()

	mockProc := newMockSpawnedProcess()
	mockProc.stdin.onClose = mockProc.signalExit
	tr := newMockTransportWithProcess(t, mockProc, nil)

	closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := tr.Close(closeCtx); err != nil {
		t.Fatalf("first Close() failed: %v", err)
	}
	if err := tr.Close(closeCtx); err != nil {
		t.Fatalf("second Close() failed: %v", err)
	}
	if mockProc.Killed() {
		t.Fatal("idempotent Close() should not kill after graceful exit")
	}
}

func TestSubprocessCLITransportWriteAfterCloseReturnsTypedError(t *testing.T) {
	t.Parallel()

	mockProc := newMockSpawnedProcess()
	mockProc.stdin.onClose = mockProc.signalExit
	tr := newMockTransportWithProcess(t, mockProc, nil)

	closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := tr.Close(closeCtx); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	err := tr.Write(context.Background(), `{"type":"user","message":"after close"}`)
	var cliErr *types.CLIConnectionError
	if !errors.As(err, &cliErr) {
		t.Fatalf("Write() after close = %T %v, want *types.CLIConnectionError", err, err)
	}
}

func TestSubprocessCLITransportCloseContextCancellationEscalates(t *testing.T) {
	t.Parallel()

	mockProc := newMockSpawnedProcess()
	tr := newMockTransportWithProcess(t, mockProc, nil)

	closeCtx, cancel := context.WithCancel(context.Background())
	cancel()
	err := tr.Close(closeCtx)
	var procErr *types.ProcessError
	if !errors.As(err, &procErr) {
		t.Fatalf("Close(canceled ctx) = %T %v, want *types.ProcessError", err, err)
	}
	if !mockProc.Killed() {
		t.Fatal("Close(canceled ctx) did not escalate to Kill")
	}
}

// TestConnectWithCustomSpawner verifies that Connect() uses the custom spawner
// when SpawnProcess is set and that it receives correct SpawnOptions.
func TestConnectWithCustomSpawner(t *testing.T) {
	t.Parallel()

	var receivedOpts types.SpawnOptions
	var receivedCtx context.Context
	mockProc := newMockSpawnedProcess()

	spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
		receivedCtx = ctx
		receivedOpts = opts
		return mockProc, nil
	})

	opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
	tr := NewSubprocessCLITransport("/usr/bin/claude", "/tmp/test", map[string]string{"MY_VAR": "my_val"}, log.NewLogger(true), "", opts)

	ctx := context.Background()
	err := tr.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	// Verify spawner received correct options
	if receivedOpts.Command != "/usr/bin/claude" {
		t.Errorf("SpawnOptions.Command = %q, want %q", receivedOpts.Command, "/usr/bin/claude")
	}
	if receivedOpts.CWD != "/tmp/test" {
		t.Errorf("SpawnOptions.CWD = %q, want %q", receivedOpts.CWD, "/tmp/test")
	}
	if receivedCtx == nil {
		t.Error("spawner received nil context")
	}
	// Verify env vars contain both SDK vars and custom vars
	if receivedOpts.Env["MY_VAR"] != "my_val" {
		t.Error("custom env var not passed to spawner")
	}
	if receivedOpts.Env["CLAUDE_CODE_ENTRYPOINT"] != "agent" {
		t.Errorf("CLAUDE_CODE_ENTRYPOINT = %q, want %q", receivedOpts.Env["CLAUDE_CODE_ENTRYPOINT"], "agent")
	}

	// Verify transport state
	if tr.customProcess == nil {
		t.Error("customProcess should be set after Connect()")
	}
	if tr.cmd != nil {
		t.Error("cmd should be nil when using custom spawner")
	}

	// Cleanup
	_ = mockProc.Kill()
	ctx2, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = tr.Close(ctx2)
}

// TestConnectWithCustomSpawner_Error verifies Connect() propagates spawner errors.
func TestConnectWithCustomSpawner_Error(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("spawner failed: VM not available")
	spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
		return nil, expectedErr
	})

	opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
	tr := NewSubprocessCLITransport("/usr/bin/claude", "", nil, log.NewLogger(true), "", opts)

	err := tr.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect() should have failed")
	}
	if !strings.Contains(err.Error(), "spawner failed") {
		t.Errorf("error should contain spawner message, got: %v", err)
	}
}

// TestConnectWithCustomSpawner_NilPipes verifies Connect() rejects a process with nil pipes.
func TestConnectWithCustomSpawner_NilPipes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		process types.SpawnedProcess
	}{
		{
			name:    "nil stdin",
			process: &mockSpawnedProcessNilPipe{nilStdin: true},
		},
		{
			name:    "nil stdout",
			process: &mockSpawnedProcessNilPipe{nilStdout: true},
		},
		{
			name:    "nil stderr",
			process: &mockSpawnedProcessNilPipe{nilStderr: true},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
				return tt.process, nil
			})

			opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner)
			tr := NewSubprocessCLITransport("/usr/bin/claude", "", nil, log.NewLogger(true), "", opts)

			err := tr.Connect(context.Background())
			if err == nil {
				t.Fatal("Connect() should fail with nil pipe")
			}
			if !strings.Contains(err.Error(), "nil") {
				t.Errorf("error should mention nil pipe, got: %v", err)
			}
		})
	}
}

// mockSpawnedProcessNilPipe returns nil for specified pipes.
type mockSpawnedProcessNilPipe struct {
	nilStdin  bool
	nilStdout bool
	nilStderr bool
}

func (m *mockSpawnedProcessNilPipe) Stdin() io.WriteCloser {
	if m.nilStdin {
		return nil
	}
	return &mockWriteCloser{buf: &bytes.Buffer{}}
}

func (m *mockSpawnedProcessNilPipe) Stdout() io.ReadCloser {
	if m.nilStdout {
		return nil
	}
	r, _ := io.Pipe()
	return r
}

func (m *mockSpawnedProcessNilPipe) Stderr() io.ReadCloser {
	if m.nilStderr {
		return nil
	}
	r, _ := io.Pipe()
	return r
}
func (m *mockSpawnedProcessNilPipe) Kill() error   { return nil }
func (m *mockSpawnedProcessNilPipe) Wait() error   { return nil }
func (m *mockSpawnedProcessNilPipe) ExitCode() int { return 0 }
func (m *mockSpawnedProcessNilPipe) Killed() bool  { return false }

// TestCloseCustomProcess verifies Close() calls Wait() on custom process.
func TestCloseCustomProcess(t *testing.T) {
	t.Parallel()

	mockProc := newMockSpawnedProcess()
	tr := NewSubprocessCLITransport("/usr/bin/claude", "", nil, log.NewLogger(true), "", nil)

	// Manually set transport state as if connectWithCustomSpawner ran
	tr.customProcess = mockProc
	tr.stdin = mockProc.Stdin()
	tr.stdout = mockProc.Stdout()
	tr.stderr = mockProc.Stderr()
	tr.ready = true
	ctx, cancel := context.WithCancel(context.Background())
	tr.ctx = ctx
	tr.cancel = cancel

	// Initialize procDone and launch watcher goroutine (mirrors connectWithCustomSpawner)
	tr.procDone = make(chan struct{})
	go func() {
		_ = mockProc.Wait()
		close(tr.procDone)
	}()

	// Simulate process exiting cleanly after stdin is closed
	go func() {
		select {
		case <-mockProc.waitCh:
		default:
			close(mockProc.waitCh)
		}
	}()

	err := tr.Close(context.Background())
	if err != nil {
		t.Fatalf("Close() failed: %v", err)
	}

	if tr.customProcess != nil {
		t.Error("customProcess should be nil after Close()")
	}
}

// TestCloseCustomProcess_NotConnected verifies Close() is a no-op when not connected.
func TestCloseCustomProcess_NotConnected(t *testing.T) {
	t.Parallel()

	tr := NewSubprocessCLITransport("/usr/bin/claude", "", nil, log.NewLogger(true), "", nil)

	err := tr.Close(context.Background())
	if err != nil {
		t.Fatalf("Close() should succeed when not connected, got: %v", err)
	}
}

// --- helpers ---
