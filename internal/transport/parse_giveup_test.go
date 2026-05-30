package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// recordingTransportObserver is a concurrency-safe Observer for asserting that the
// transport emits give-up telemetry. It embeds NopObserver so it only overrides the
// events under test.
type recordingTransportObserver struct {
	types.NopObserver
	parseErrs atomic.Uint64
	giveUp    atomic.Uint64
}

func (r *recordingTransportObserver) OnParseError(n uint, _ error) { r.parseErrs.Store(uint64(n)) }
func (r *recordingTransportObserver) OnParseGiveUp(n uint)         { r.giveUp.Store(uint64(n)) }

// TestMessageReaderLoop_GiveUpTerminatesSubprocess proves the B2 fix: when the
// configurable consecutive-parse-error threshold is crossed, the transport
// (1) honors the configured threshold, (2) emits OnParseGiveUp, and (3) terminates
// the subprocess (reaps it) rather than leaving a zombie — the exact gap that
// previously left an orphaned process alive after the reader loop gave up.
func TestMessageReaderLoop_GiveUpTerminatesSubprocess(t *testing.T) {
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

	obs := &recordingTransportObserver{}
	const threshold uint = 2
	opts := types.NewClaudeAgentOptions().
		WithSpawnProcess(spawner).
		WithObserver(obs).
		WithMaxConsecutiveParseErrors(threshold)
	tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)
	useFastParseErrorBackoff(tr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := tr.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	// Feed exactly `threshold` invalid JSON lines to trip the give-up branch.
	go func() {
		for i := uint(0); i < threshold; i++ {
			_, _ = fmt.Fprintf(stdoutW, "garbage-%d\n", i)
		}
	}()

	// Reader loop exit closes the messages channel.
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	closed := false
	for !closed {
		select {
		case _, ok := <-tr.messages:
			if !ok {
				closed = true
			}
		case <-timer.C:
			t.Fatal("timed out waiting for reader loop to give up and close the channel")
		}
	}

	// (1) configurable threshold honored + (2) give-up telemetry emitted.
	if got := obs.giveUp.Load(); got != uint64(threshold) {
		t.Fatalf("OnParseGiveUp count = %d, want %d", got, threshold)
	}
	if got := obs.parseErrs.Load(); got != uint64(threshold) {
		t.Fatalf("OnParseError last count = %d, want %d", got, threshold)
	}

	// (3) the subprocess was terminated, not left as a zombie.
	if !mockProc.Killed() {
		t.Fatal("subprocess was NOT terminated after parse give-up (B2 regression: zombie left alive)")
	}

	// The error is surfaced to the consumer and is detectable as the
	// ErrParseGiveUp sentinel via errors.Is (so consumers can branch on it).
	gotErr := tr.GetError()
	if gotErr == nil {
		t.Fatal("expected a stored transport error after give-up, got nil")
	}
	if !errors.Is(gotErr, ErrParseGiveUp) {
		t.Fatalf("give-up error %v is not errors.Is(ErrParseGiveUp)", gotErr)
	}

	cancel()
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer closeCancel()
	_ = tr.Close(closeCtx)
}
