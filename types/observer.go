package types

import "time"

// Observer receives structured lifecycle and health telemetry from the SDK's
// transport and query layers. It is the single sink for SDK-internal observability
// signals: the SDK owns emission, the consumer owns aggregation (e.g. exporting to
// Prometheus). A nil Observer is equivalent to NopObserver — telemetry is simply
// dropped — so wiring an Observer is always optional and backwards-compatible.
//
// Concurrency: the SDK invokes Observer methods from transport and query goroutines,
// sometimes concurrently. Implementations MUST be safe for concurrent use and MUST
// be non-blocking — an Observer that blocks will stall the message-reader loop.
// Aggregate cheaply (atomic counters, histograms); never perform I/O inline.
//
// Forward compatibility: implementations SHOULD embed NopObserver so that future
// event additions to this interface do not break them.
type Observer interface {
	// OnConnect fires once per transport Connect attempt, with the wall-clock
	// duration from dial to ready and the terminal error (nil on success).
	OnConnect(d time.Duration, err error)

	// OnFirstMessage fires once per successful connection with the latency from
	// Connect-complete to the first decoded message — a first-token proxy.
	OnFirstMessage(d time.Duration)

	// OnSubprocessExit fires exactly once when the CLI subprocess terminates.
	// exitCode is the process exit code, or -1 if unknown. requested is true when
	// the SDK initiated the shutdown (clean Close), false when the process died on
	// its own. err carries the cause when the exit was unexpected (nil if clean).
	OnSubprocessExit(exitCode int, requested bool, err error)

	// OnParseError fires on each CLI JSON parse failure, carrying the running count
	// of consecutive failures (reset to 0 on the next successful parse).
	OnParseError(consecutive uint, err error)

	// OnParseGiveUp fires when consecutive parse failures cross the transport's
	// configured threshold and the transport terminates the subprocess as
	// unrecoverable. It is always followed by an OnSubprocessExit.
	OnParseGiveUp(consecutive uint)

	// OnBackpressure fires when the message channel saturates and the producer must
	// block waiting for the consumer to drain. Use it to surface slow consumers.
	OnBackpressure()

	// OnUnknownMessage fires when the parser encounters a message type or content
	// block it does not recognize — a signal of CLI wire-format drift ahead of the
	// SDK. discriminator is the unrecognized "type" value.
	OnUnknownMessage(discriminator string)
}

// TransportHealth is a point-in-time snapshot of subprocess/transport health. The
// transport is the sole owner of this truth; consumers (e.g. a daemon health
// endpoint) read it rather than reconstructing liveness from their own state.
type TransportHealth struct {
	// Connected is true when a subprocess is spawned and has not yet exited.
	Connected bool
	// Ready is true when the transport is ready to send and receive messages.
	Ready bool
	// PID is the subprocess OS process ID, or 0 when there is no live exec process
	// (e.g. before Connect, after exit, or for a custom-spawned process).
	PID int
	// LastError is the most recent transport error, or nil when healthy.
	LastError error
}

// NopObserver is an Observer whose methods do nothing. It is the implicit default
// when no Observer is configured, and it is the recommended embedding base for real
// Observer implementations: embed it to inherit forward-compatible no-op defaults
// and override only the events you care about.
type NopObserver struct{}

// Ensure NopObserver satisfies Observer at compile time.
var _ Observer = NopObserver{}

func (NopObserver) OnConnect(time.Duration, error)    {}
func (NopObserver) OnFirstMessage(time.Duration)      {}
func (NopObserver) OnSubprocessExit(int, bool, error) {}
func (NopObserver) OnParseError(uint, error)          {}
func (NopObserver) OnParseGiveUp(uint)                {}
func (NopObserver) OnBackpressure()                   {}
func (NopObserver) OnUnknownMessage(string)           {}
