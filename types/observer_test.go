package types

import (
	"errors"
	"testing"
	"time"
)

// recordingObserver embeds NopObserver and records which events fired, proving the
// embedding pattern lets implementations override only the methods they care about.
type recordingObserver struct {
	NopObserver
	connectCalls    int
	lastConnectErr  error
	firstMsg        time.Duration
	exitCode        int
	exitRequested   bool
	parseErr        uint
	parseGiveUp     uint
	backpressure    int
	unknownMessages []string
}

func (r *recordingObserver) OnConnect(_ time.Duration, err error) {
	r.connectCalls++
	r.lastConnectErr = err
}
func (r *recordingObserver) OnFirstMessage(d time.Duration) { r.firstMsg = d }
func (r *recordingObserver) OnSubprocessExit(c int, req bool, _ error) {
	r.exitCode, r.exitRequested = c, req
}
func (r *recordingObserver) OnParseError(n uint, _ error) { r.parseErr = n }
func (r *recordingObserver) OnParseGiveUp(n uint)         { r.parseGiveUp = n }
func (r *recordingObserver) OnBackpressure()              { r.backpressure++ }
func (r *recordingObserver) OnUnknownMessage(d string) {
	r.unknownMessages = append(r.unknownMessages, d)
}

func TestNopObserver_SatisfiesInterfaceAndIsInert(t *testing.T) {
	t.Parallel()

	var obs Observer = NopObserver{}
	// All methods must be safe to call and do nothing — no panic.
	obs.OnConnect(time.Second, errors.New("x"))
	obs.OnFirstMessage(time.Second)
	obs.OnSubprocessExit(1, true, errors.New("x"))
	obs.OnParseError(3, errors.New("x"))
	obs.OnParseGiveUp(6)
	obs.OnBackpressure()
	obs.OnUnknownMessage("mystery")
}

func TestObserverOrNop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    *ClaudeAgentOptions
		wantNop bool
	}{
		{name: "nil options", opts: nil, wantNop: true},
		{name: "nil observer", opts: &ClaudeAgentOptions{}, wantNop: true},
		{name: "set observer", opts: &ClaudeAgentOptions{Observer: &recordingObserver{}}, wantNop: false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.opts.ObserverOrNop()
			if got == nil {
				t.Fatal("ObserverOrNop returned nil — must never be nil")
			}
			_, isNop := got.(NopObserver)
			if isNop != tt.wantNop {
				t.Fatalf("ObserverOrNop() nop=%v, want %v", isNop, tt.wantNop)
			}
		})
	}
}

func TestWithObserver_SetsField(t *testing.T) {
	t.Parallel()

	rec := &recordingObserver{}
	opts := (&ClaudeAgentOptions{}).WithObserver(rec)
	if opts.Observer != rec {
		t.Fatal("WithObserver did not set the Observer field")
	}
	// The accessor returns the exact instance, and events route to it.
	opts.ObserverOrNop().OnBackpressure()
	if rec.backpressure != 1 {
		t.Fatalf("backpressure routed=%d, want 1", rec.backpressure)
	}
}

func TestRecordingObserver_EmbeddingOverridesSubset(t *testing.T) {
	t.Parallel()

	rec := &recordingObserver{}
	var obs Observer = rec // embedding NopObserver satisfies the full interface

	obs.OnConnect(10*time.Millisecond, nil)
	obs.OnFirstMessage(20 * time.Millisecond)
	obs.OnSubprocessExit(0, true, nil)
	obs.OnParseError(2, errors.New("bad json"))
	obs.OnParseGiveUp(6)
	obs.OnBackpressure()
	obs.OnUnknownMessage("future_block")

	if rec.connectCalls != 1 || rec.lastConnectErr != nil {
		t.Fatalf("OnConnect: calls=%d err=%v", rec.connectCalls, rec.lastConnectErr)
	}
	if rec.firstMsg != 20*time.Millisecond {
		t.Fatalf("OnFirstMessage: got %v", rec.firstMsg)
	}
	if rec.exitCode != 0 || !rec.exitRequested {
		t.Fatalf("OnSubprocessExit: code=%d requested=%v", rec.exitCode, rec.exitRequested)
	}
	if rec.parseErr != 2 || rec.parseGiveUp != 6 || rec.backpressure != 1 {
		t.Fatalf("parse/backpressure: err=%d giveup=%d bp=%d", rec.parseErr, rec.parseGiveUp, rec.backpressure)
	}
	if len(rec.unknownMessages) != 1 || rec.unknownMessages[0] != "future_block" {
		t.Fatalf("OnUnknownMessage: got %v", rec.unknownMessages)
	}
}
