package transport

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// emissionObserver records connect / first-message / unknown-message telemetry.
type emissionObserver struct {
	types.NopObserver
	connectCalls  atomic.Uint64
	connectHadErr atomic.Bool
	firstMsgCount atomic.Uint64
	mu            sync.Mutex
	unknownTypes  []string
}

func (e *emissionObserver) OnConnect(_ time.Duration, err error) {
	e.connectCalls.Add(1)
	if err != nil {
		e.connectHadErr.Store(true)
	}
}

func (e *emissionObserver) OnFirstMessage(time.Duration) { e.firstMsgCount.Add(1) }

func (e *emissionObserver) OnUnknownMessage(discriminator string) {
	e.mu.Lock()
	e.unknownTypes = append(e.unknownTypes, discriminator)
	e.mu.Unlock()
}

// TestTransportEmitsConnectFirstMessageAndUnknown proves the connect, first-message,
// and schema-drift (unknown-message) telemetry fire at the right sites: OnConnect once
// on a successful connect, OnFirstMessage once on the first decoded message, and
// OnUnknownMessage with the drifted discriminator for an unrecognized message type.
func TestTransportEmitsConnectFirstMessageAndUnknown(t *testing.T) {
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

	obs := &emissionObserver{}
	opts := types.NewClaudeAgentOptions().WithSpawnProcess(spawner).WithObserver(obs)
	tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := tr.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	// OnConnect fired exactly once, success (no error).
	if got := obs.connectCalls.Load(); got != 1 {
		t.Fatalf("OnConnect calls = %d, want 1", got)
	}
	if obs.connectHadErr.Load() {
		t.Fatal("OnConnect reported an error on a successful connect")
	}

	const unknownType = "totally_unknown_xyz_2026"
	go func() {
		_, _ = fmt.Fprintf(stdoutW, "%s\n", `{"type":"`+unknownType+`"}`)
	}()

	select {
	case msg, ok := <-tr.messages:
		if !ok {
			t.Fatal("messages channel closed before delivering the message")
		}
		if msg.GetMessageType() != unknownType {
			t.Fatalf("delivered message type = %q, want %q", msg.GetMessageType(), unknownType)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for the message to be delivered")
	}

	if got := obs.firstMsgCount.Load(); got != 1 {
		t.Fatalf("OnFirstMessage count = %d, want 1", got)
	}
	obs.mu.Lock()
	unknown := append([]string(nil), obs.unknownTypes...)
	obs.mu.Unlock()
	if len(unknown) != 1 || unknown[0] != unknownType {
		t.Fatalf("OnUnknownMessage = %v, want [%s]", unknown, unknownType)
	}

	cancel()
	_ = mockProc.Kill()
	closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer closeCancel()
	_ = tr.Close(closeCtx)
}
