package transport

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// newTestTransport creates a SubprocessCLITransport for unit tests without
// actually starting a subprocess. This allows testing buildCommandArgs() and
// buildSettingsJSON() in isolation.
func newTestTransport(t *testing.T, opts *types.ClaudeAgentOptions) *SubprocessCLITransport {
	t.Helper()
	return NewSubprocessCLITransport(
		"/usr/local/bin/claude",
		"",
		nil,
		log.NewLogger(false),
		"",
		opts,
	)
}

// flagValue returns the value immediately following flag in args, or ("", false).
func flagValue(args []string, flag string) (string, bool) {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

// hasFlag checks whether flag appears anywhere in args.
func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

type mockSpawnedProcess struct {
	mu       sync.Mutex
	stdin    *mockWriteCloser
	stdout   *io.PipeReader
	stdoutW  *io.PipeWriter
	stderr   *io.PipeReader
	stderrW  *io.PipeWriter
	killed   bool
	exitCode int
	waitErr  error
	waitCh   chan struct{}
}

func newMockSpawnedProcess() *mockSpawnedProcess {
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	return &mockSpawnedProcess{
		stdin:   &mockWriteCloser{buf: &bytes.Buffer{}},
		stdout:  stdoutR,
		stdoutW: stdoutW,
		stderr:  stderrR,
		stderrW: stderrW,
		waitCh:  make(chan struct{}),
	}
}

func (m *mockSpawnedProcess) Stdin() io.WriteCloser { return m.stdin }
func (m *mockSpawnedProcess) Stdout() io.ReadCloser { return m.stdout }
func (m *mockSpawnedProcess) Stderr() io.ReadCloser { return m.stderr }

func (m *mockSpawnedProcess) Kill() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.killed = true
	// Close stdout/stderr to unblock readers
	_ = m.stdoutW.Close()
	_ = m.stderrW.Close()
	// Signal Wait() to return
	select {
	case <-m.waitCh:
	default:
		close(m.waitCh)
	}
	return nil
}

func (m *mockSpawnedProcess) Wait() error {
	<-m.waitCh
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.waitErr
}

func (m *mockSpawnedProcess) ExitCode() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.exitCode
}

func (m *mockSpawnedProcess) Killed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.killed
}

type mockWriteCloser struct {
	mu      sync.Mutex
	buf     *bytes.Buffer
	closed  bool
	onClose func()
}

func (m *mockWriteCloser) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, errors.New("write to closed writer")
	}
	return m.buf.Write(p)
}

func (m *mockWriteCloser) Close() error {
	m.mu.Lock()
	alreadyClosed := m.closed
	m.closed = true
	onClose := m.onClose
	m.mu.Unlock()
	if !alreadyClosed && onClose != nil {
		onClose()
	}
	return nil
}

func (m *mockSpawnedProcess) signalExit() {
	_ = m.stdoutW.Close()
	_ = m.stderrW.Close()
	select {
	case <-m.waitCh:
	default:
		close(m.waitCh)
	}
}

func (m *mockSpawnedProcess) signalExitWithStdoutOpen() {
	_ = m.stderrW.Close()
	select {
	case <-m.waitCh:
	default:
		close(m.waitCh)
	}
}

func waitForCondition(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was not satisfied before timeout")
}

func waitForStderr(t *testing.T, tr *SubprocessCLITransport, want string) {
	t.Helper()
	waitForCondition(t, 2*time.Second, func() bool {
		return tr.Stderr() == want
	})
}

func waitForStderrContains(t *testing.T, tr *SubprocessCLITransport, wants ...string) {
	t.Helper()
	waitForCondition(t, 2*time.Second, func() bool {
		got := tr.Stderr()
		for _, want := range wants {
			if !strings.Contains(got, want) {
				return false
			}
		}
		return true
	})
}

func newMockTransportWithProcess(t *testing.T, mockProc *mockSpawnedProcess, opts *types.ClaudeAgentOptions) *SubprocessCLITransport {
	t.Helper()
	spawner := types.ProcessSpawner(func(ctx context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
		return mockProc, nil
	})
	if opts == nil {
		opts = types.NewClaudeAgentOptions()
	}
	opts.WithSpawnProcess(spawner)
	tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)
	if err := tr.Connect(context.Background()); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	return tr
}

func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func effortPtr(e types.EffortLevel) *types.EffortLevel {
	return &e
}

func keysOf(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// ===== Phase E: Subprocess Crash Tests (T003-T) =====
//
// These tests use a mockSpawnedProcess (custom spawner) to simulate subprocess crashes.
// They verify the watcher goroutine behavior added in T007.
//
// RED before T007: IsReady() stays true after Kill() — no watcher to clear it.
// GREEN after T007: watcher sets ready=false; all assertions pass.

// waitForNotReady polls IsReady() until it returns false or the deadline is reached.
func waitForNotReady(tr *SubprocessCLITransport, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !tr.IsReady() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

// TestSubprocessCrash_ReadyFalse verifies that IsReady() returns false
// immediately after the subprocess exits spontaneously.

func useFastParseErrorBackoff(tr *SubprocessCLITransport) {
	tr.parseErrorBackoff = func(consecutive uint) time.Duration {
		if consecutive == 0 {
			return 0
		}
		return time.Duration(consecutive) * 5 * time.Millisecond
	}
}

// TestMessageReaderLoop_ParseErrorBackoff verifies that repeated parse errors
// trigger exponential backoff instead of spinning in a tight CPU loop.
