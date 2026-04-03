package transport

import (
	"io"
	"os/exec"
	"sync/atomic"
)

// WarmProcess holds a pre-spawned CLI subprocess for faster first connection.
type WarmProcess struct {
	Cmd    *exec.Cmd
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
	Stderr io.ReadCloser
	Done   chan struct{} // closed when the process exits
}

// Kill terminates the warm process and waits for it to exit.
// Safe to call multiple times. Waits on the Done channel to ensure
// the background goroutine has completed cmd.Wait().
func (w *WarmProcess) Kill() {
	if w.Cmd != nil && w.Cmd.Process != nil {
		_ = w.Cmd.Process.Kill()
	}
	// Wait for the background goroutine to finish cmd.Wait()
	<-w.Done
}

// IsAlive returns true if the process has not yet exited.
func (w *WarmProcess) IsAlive() bool {
	select {
	case <-w.Done:
		return false
	default:
		return true
	}
}

// warmPool holds at most one pre-warmed subprocess.
var warmPool atomic.Pointer[WarmProcess]

// StoreWarmProcess atomically stores a warm process in the pool.
// If the pool already contains a process, the old one is replaced.
func StoreWarmProcess(wp *WarmProcess) {
	old := warmPool.Swap(wp)
	if old != nil {
		old.Kill()
	}
}

// ConsumeWarmProcess atomically takes the warm process from the pool.
// Returns nil if the pool is empty. The process is removed from the pool
// on return (exactly-once consumption via Swap).
func ConsumeWarmProcess() *WarmProcess {
	return warmPool.Swap(nil)
}
