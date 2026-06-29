package transport

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// Close terminates the subprocess and cleans up all resources.
//
// Shutdown is staged to match Codex SDK semantics:
//  1. mark shutdown requested and close stdin to signal EOF;
//  2. wait up to ShutdownGrace for the watcher-owned Wait() result;
//  3. interrupt/cancel and wait up to TerminateGrace;
//  4. kill as a last resort.
//
// cmd.Wait() / customProcess.Wait() is owned by the watcher goroutine. Close
// drains procDone and must not call Wait() directly.
func (t *SubprocessCLITransport) Close(ctx context.Context) error {
	t.mu.Lock()

	if t.cmd == nil && t.customProcess == nil {
		t.mu.Unlock()
		return nil // Not connected
	}

	t.logger.Debug("Closing CLI subprocess...")
	t.ready = false
	t.shutdownRequested = true

	// 1. Close stdin FIRST — signals subprocess to exit gracefully.
	if t.stdin != nil {
		_ = t.stdin.Close()
		t.stdin = nil
	}

	// 2. Wait for process to exit.
	var err error
	if t.customProcess != nil {
		err = t.closeCustomProcess(ctx)
	} else {
		err = t.closeExecCommand(ctx)
	}
	t.mu.Unlock()

	// 3. Wait for readStderr goroutine to finish.
	// This MUST happen after releasing t.mu because readStderr may call
	// OnError which acquires t.mu — waiting under lock would deadlock.
	t.wg.Wait()

	return err
}

// closeCustomProcess handles cleanup for a custom-spawned process.
// customProcess.Wait() is owned by the watcher goroutine (launched in connectWithCustomSpawner).
// This method drains procDone — it must NOT call customProcess.Wait() again.
func (t *SubprocessCLITransport) closeCustomProcess(ctx context.Context) error {
	if waitForProcDone(ctx, t.procDone, ShutdownGrace) {
		if t.cancel != nil {
			t.cancel()
			t.cancel = nil
		}
		t.customProcess = nil
		return nil
	}

	// Custom processes do not expose a portable interrupt. Cancel the transport
	// context first; the context monitor goroutine will call Kill().
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
	if waitForProcDone(ctx, t.procDone, TerminateGrace) {
		t.customProcess = nil
		return nil
	}

	_ = t.customProcess.Kill()
	<-t.procDone
	t.customProcess = nil
	return t.newProcessErrorWithDiagnostics("subprocess did not exit gracefully, killed", -1)
}

// closeExecCommand handles cleanup for exec.Command-spawned process.
// cmd.Wait() is owned by the watcher goroutine (launched in connectWithExecCommand).
// This method drains procDone — it must NOT call cmd.Wait() again.
func (t *SubprocessCLITransport) closeExecCommand(ctx context.Context) error {
	if waitForProcDone(ctx, t.procDone, ShutdownGrace) {
		if t.cancel != nil {
			t.cancel()
			t.cancel = nil
		}
		t.cmd = nil
		return nil
	}

	if t.cmd != nil && t.cmd.Process != nil {
		_ = t.cmd.Process.Signal(os.Interrupt)
	}
	if waitForProcDone(ctx, t.procDone, TerminateGrace) {
		if t.cancel != nil {
			t.cancel()
			t.cancel = nil
		}
		t.cmd = nil
		return nil
	}

	if t.cmd != nil && t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
	}
	<-t.procDone
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
	t.cmd = nil
	return t.newProcessErrorWithDiagnostics("subprocess did not exit gracefully, killed", -1)
}

// Stderr returns the captured stderr diagnostic tail. It is stable after Close.
func (t *SubprocessCLITransport) Stderr() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stderrStringLocked()
}

// OnError stores an error that occurred during transport operation.
// This allows errors from the reading loop to be retrieved later.
func (t *SubprocessCLITransport) OnError(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.err == nil {
		t.err = err
	}
}

// IsReady returns true if the transport is ready for communication.
func (t *SubprocessCLITransport) IsReady() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.ready
}

// GetError returns any error that occurred during transport operation.
// This is useful for checking if an error occurred in the reading loop.
func (t *SubprocessCLITransport) GetError() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.err
}

// ProcessID returns the OS process ID of the Claude Code subprocess.
// Returns 0 if the subprocess has not been started or cmd.Process is nil.
func (t *SubprocessCLITransport) ProcessID() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cmd != nil && t.cmd.Process != nil {
		return t.cmd.Process.Pid
	}
	return 0
}

// Health returns a point-in-time snapshot of subprocess/transport health. The
// transport owns this truth (liveness, readiness, last error); callers read it for
// health endpoints rather than tracking subprocess state separately.
func (t *SubprocessCLITransport) Health() types.TransportHealth {
	t.mu.Lock()
	defer t.mu.Unlock()

	h := types.TransportHealth{
		Connected: t.cmd != nil || t.customProcess != nil,
		Ready:     t.ready,
		LastError: t.err,
	}
	if t.cmd != nil && t.cmd.Process != nil {
		h.PID = t.cmd.Process.Pid
	}
	return h
}

func waitForProcDone(ctx context.Context, procDone <-chan struct{}, timeout time.Duration) bool {
	// Prioritize an already-canceled context: the caller asked to stop waiting,
	// so escalate deterministically. Without this, a procDone that closes
	// concurrently (e.g. the context-monitor goroutine's Kill already reaped the
	// process) races the ctx.Done() case in the select below — Go would pick a
	// ready case at random, so Close() would non-deterministically report a
	// graceful exit during a forced shutdown.
	if ctx.Err() != nil {
		return false
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-procDone:
		return true
	case <-timer.C:
		return false
	case <-ctx.Done():
		return false
	}
}

func procDoneClosed(procDone <-chan struct{}) bool {
	if procDone == nil {
		return false
	}
	select {
	case <-procDone:
		return true
	default:
		return false
	}
}

func (t *SubprocessCLITransport) drainStderr(done <-chan struct{}) {
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
	}
}

func (t *SubprocessCLITransport) stderrStringLocked() string {
	if t.stderrTail == nil {
		return ""
	}
	return t.stderrTail.String()
}

func captureStderrTail(tail *ringBuffer, line []byte) {
	if tail == nil || len(line) == 0 {
		return
	}
	_, _ = tail.Write(line)
}

func (t *SubprocessCLITransport) newProcessErrorWithDiagnostics(message string, exitCode int) *types.ProcessError {
	if tail := t.stderrStringLocked(); tail != "" {
		message = fmt.Sprintf("%s: stderr tail: %q", message, tail)
	}
	return types.NewProcessErrorWithCode(message, exitCode)
}
