package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// maxParseErrorBackoff is the ceiling for exponential backoff on consecutive parse errors.
const maxParseErrorBackoff = 30 * time.Second

// maxConsecutiveParseErrors is the number of consecutive JSON parse errors
// before messageReaderLoop gives up and exits. This prevents the SDK from
// stalling forever when the subprocess sends garbage.
const maxConsecutiveParseErrors uint = 6

// ErrParseGiveUp is the terminal error surfaced when messageReaderLoop gives up
// after crossing the consecutive-parse-error threshold and terminates the
// subprocess as unrecoverable. Consumers can detect it with errors.Is. Mirrors
// codex-agent-sdk-go's jsonrpc.ErrParseGiveUp.
var ErrParseGiveUp = errors.New("transport: too many consecutive parse errors, subprocess terminated")

// messageReaderLoop reads JSON lines from stdout and parses them into messages.
// It runs in a goroutine and sends messages to the messages channel.
// It respects context cancellation and closes the messages channel when done.
// stdout and procDone are passed as parameters to avoid a data race with Connect()
// overwriting the corresponding struct fields.
//
// Note on lifecycle cancellation: ReadLine() blocks on stdout, which cannot be
// interrupted by context cancel directly. Since the transport owns stdout after
// Connect(), this loop closes stdout when the context is canceled or the
// captured process-done channel closes. That unblocks ReadLine() even when a
// subprocess exits while stdout remains open with a partial line.
//
// Parse errors trigger exponential backoff (1s, 2s, 4s, ... up to 30s) to prevent
// CPU spin on repeated invalid JSON. The backoff counter resets on successful parse.
// maxBufferSize returns the maximum size (bytes) of a single JSON line the message
// reader will accept. It honors ClaudeAgentOptions.MaxBufferSize when the caller set
// a positive value, otherwise falls back to DefaultMaxBufferSize. This is the single
// source of the line-size rule for every reader the transport creates, so the public
// MaxBufferSize option is authoritative (it was previously silently ignored).
func (t *SubprocessCLITransport) maxBufferSize() int {
	if t.options != nil && t.options.MaxBufferSize != nil && *t.options.MaxBufferSize > 0 {
		return *t.options.MaxBufferSize
	}
	return DefaultMaxBufferSize
}

// observer returns the telemetry Observer configured on the transport's options,
// or NopObserver when none is set. Single accessor so emission sites never repeat
// the nil-guard logic.
func (t *SubprocessCLITransport) observer() types.Observer {
	return t.options.ObserverOrNop()
}

// maxParseErrors returns the configured consecutive-parse-error threshold, honoring
// ClaudeAgentOptions.MaxConsecutiveParseErrors when positive, else the package
// default. Single source of the threshold rule.
func (t *SubprocessCLITransport) maxParseErrors() uint {
	if t.options != nil && t.options.MaxConsecutiveParseErrors != nil && *t.options.MaxConsecutiveParseErrors > 0 {
		return *t.options.MaxConsecutiveParseErrors
	}
	return maxConsecutiveParseErrors
}

// terminateOnUnrecoverableError forcibly terminates the subprocess after an
// unrecoverable transport condition (e.g. sustained CLI parse failures). It is the
// error-path analog of Close(): it records the cause as the transport error, kills
// the process so it cannot linger as a zombie, and cancels the transport context so
// dependent goroutines unwind. The watcher goroutine then observes procDone and
// emits the subprocess-exit telemetry. Safe to call from the message-reader
// goroutine; Kill and cancel are idempotent.
//
// reason is the AUTHORITATIVE terminal cause and overrides any prior transient
// error (e.g. an individual parse failure recorded via OnError): consumers asking
// GetError() after termination want "why was the subprocess killed", and reason
// (which wraps ErrParseGiveUp) carries that — and stays errors.Is-detectable.
func (t *SubprocessCLITransport) terminateOnUnrecoverableError(reason error) {
	t.mu.Lock()
	t.err = reason
	cancelFn := t.cancel
	var proc *os.Process
	if t.cmd != nil {
		proc = t.cmd.Process
	}
	custom := t.customProcess
	t.mu.Unlock()

	if proc != nil {
		_ = proc.Kill()
	}
	if custom != nil {
		_ = custom.Kill()
	}
	if cancelFn != nil {
		cancelFn()
	}
}

func (t *SubprocessCLITransport) messageReaderLoop(ctx context.Context, stdout io.ReadCloser, procDone <-chan struct{}) {
	ch := t.messages

	if stdout == nil {
		close(ch)
		return
	}

	readDone := make(chan struct{})
	monitorDone := make(chan struct{})
	go t.closeStdoutOnDone(ctx, stdout, procDone, readDone, monitorDone)

	defer func() {
		close(readDone)
		<-monitorDone
		close(ch)
	}()

	t.logger.Debug("Message reader loop started")
	reader := NewJSONLineReaderWithSize(stdout, t.maxBufferSize())

	state := messageReaderState{
		loopStart:    time.Now(),
		firstMessage: true,
	}

	for {
		if t.messageReaderShouldStop(ctx, procDone) {
			return
		}

		msg, ok, stop := t.readNextStdoutMessage(ctx, reader, procDone, &state)
		if stop {
			return
		}
		if !ok {
			continue
		}

		if t.sendStdoutMessage(ctx, procDone, ch, msg) {
			return
		}
	}
}

type messageReaderState struct {
	consecutiveParseErrors uint
	loopStart              time.Time
	firstMessage           bool
}

func (t *SubprocessCLITransport) messageReaderShouldStop(ctx context.Context, procDone <-chan struct{}) bool {
	select {
	case <-ctx.Done():
		t.logger.Debug("Message reader loop stopped: context canceled")
		return true
	case <-procDone:
		t.logger.Debug("Message reader loop stopped: process exited")
		return true
	default:
		return false
	}
}

func (t *SubprocessCLITransport) readNextStdoutMessage(ctx context.Context, reader *JSONLineReader, procDone <-chan struct{}, state *messageReaderState) (types.Message, bool, bool) {
	line, err := reader.ReadLine()
	if err != nil {
		t.handleStdoutReadError(ctx, line, err, procDone)
		return nil, false, true
	}
	if len(line) == 0 {
		return nil, false, false
	}
	msg, err := types.UnmarshalMessage(line)
	if err != nil {
		return nil, false, t.handleParseError(ctx, err, &state.consecutiveParseErrors, procDone)
	}
	state.consecutiveParseErrors = 0
	t.observeStdoutMessage(state, msg)
	return msg, true, false
}

func (t *SubprocessCLITransport) observeStdoutMessage(state *messageReaderState, msg types.Message) {
	if state.firstMessage {
		state.firstMessage = false
		t.observer().OnFirstMessage(time.Since(state.loopStart))
	}
	if unknown, ok := msg.(*types.UnknownMessage); ok {
		t.observer().OnUnknownMessage(unknown.GetMessageType())
	}
	t.logger.Debug("received message from CLI", zap.String("type", msg.GetMessageType()))
}

func (t *SubprocessCLITransport) sendStdoutMessage(ctx context.Context, procDone <-chan struct{}, ch chan<- types.Message, msg types.Message) bool {
	select {
	case <-ctx.Done():
		return true
	case ch <- msg:
		return false
	case <-procDone:
		return true
	}
}

// closeStdoutOnDone closes stdout when ctx is canceled or the process exits,
// unblocking a ReadLine() blocked on a partial line. It signals monitorDone on
// return and does nothing if the read loop finished first (readDone closed).
func (t *SubprocessCLITransport) closeStdoutOnDone(ctx context.Context, stdout io.ReadCloser, procDone, readDone <-chan struct{}, monitorDone chan<- struct{}) {
	defer close(monitorDone)
	select {
	case <-ctx.Done():
		_ = stdout.Close()
	case <-procDone:
		_ = stdout.Close()
	case <-readDone:
		// Read loop finished normally — nothing to do.
	}
}

// handleStdoutReadError records a stdout read error before the reader loop
// returns. EOF and shutdown-induced errors (ctx canceled, or process already
// exited) are expected and logged at Debug; a genuine read failure is surfaced
// via OnError. ctx is the passed-in context (not t.ctx) to avoid a data race
// with Connect() overwriting t.ctx under t.mu.
func (t *SubprocessCLITransport) handleStdoutReadError(ctx context.Context, line []byte, err error, procDone <-chan struct{}) {
	if errors.Is(err, io.EOF) {
		t.logger.Debug("Message reader loop stopped: EOF from CLI")
		return
	}
	if ctx.Err() != nil {
		t.logger.Debug("message reader loop stopped during shutdown", zap.Error(err))
		return
	}
	if procDoneClosed(procDone) {
		t.logger.Debug("message reader loop stopped after process exit", zap.Error(err))
		return
	}
	t.logger.Error("failed to read from CLI stdout", zap.Error(err))
	t.OnError(types.NewJSONDecodeErrorWithCause(
		"failed to read JSON line from subprocess",
		string(line),
		err,
	))
}

// handleParseError records a message parse failure and reports whether the reader
// loop should stop. It increments *consecutiveErrors, emits telemetry, and either
// terminates the subprocess after too many consecutive failures (returns true) or
// backs off and signals the caller to continue (returns false). A ctx or procDone
// signal observed during the backoff also returns true.
func (t *SubprocessCLITransport) handleParseError(ctx context.Context, err error, consecutiveErrors *uint, procDone <-chan struct{}) bool {
	*consecutiveErrors++
	t.observer().OnParseError(*consecutiveErrors, err)
	t.logger.Warn("failed to parse message from CLI",
		zap.Error(err),
		zap.Uint("consecutive_errors", *consecutiveErrors),
	)

	// Exit after too many consecutive parse failures — the subprocess is
	// emitting garbage and is unrecoverable. Terminate it authoritatively
	// (reap the process, surface the error) rather than leaving a zombie.
	if *consecutiveErrors >= t.maxParseErrors() {
		t.logger.Error("too many consecutive parse errors, terminating subprocess",
			zap.Uint("consecutive_errors", *consecutiveErrors),
		)
		t.observer().OnParseGiveUp(*consecutiveErrors)
		t.terminateOnUnrecoverableError(
			fmt.Errorf("transport.messageReaderLoop: %d consecutive parse errors (last: %w): %w", *consecutiveErrors, err, ErrParseGiveUp),
		)
		return true
	}

	// Store parse error but continue reading after backoff
	t.OnError(err)

	backoffFn := t.parseErrorBackoff
	if backoffFn == nil {
		backoffFn = defaultParseErrorBackoff
	}
	backoff := backoffFn(*consecutiveErrors)
	t.logger.Debug("parse error backoff",
		zap.Duration("backoff", backoff),
		zap.Uint("consecutive_errors", *consecutiveErrors),
	)
	if backoff <= 0 {
		return false
	}
	timer := time.NewTimer(backoff)
	select {
	case <-ctx.Done():
		timer.Stop()
		t.logger.Debug("Message reader loop stopped during backoff: context canceled")
		return true
	case <-procDone:
		timer.Stop()
		t.logger.Debug("Message reader loop stopped during backoff: process exited")
		return true
	case <-timer.C:
		// Backoff complete, continue reading
		return false
	}
}

func defaultParseErrorBackoff(consecutive uint) time.Duration {
	// Exponential backoff: 1s, 2s, 4s, 8s, ..., capped at maxParseErrorBackoff.
	// Cap shift count to prevent integer overflow. 2^5 = 32s already exceeds
	// maxParseErrorBackoff (30s), so clamping at 5 is sufficient.
	if consecutive == 0 {
		return 0
	}
	shift := consecutive - 1
	if shift > 5 {
		shift = 5
	}
	backoff := time.Duration(1<<shift) * time.Second
	if backoff > maxParseErrorBackoff {
		return maxParseErrorBackoff
	}
	return backoff
}

// Write sends a JSON message to the subprocess stdin.
// The data should be a complete JSON string (newline will be added automatically).
func (t *SubprocessCLITransport) Write(ctx context.Context, data string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.ready {
		return types.NewCLIConnectionError("transport is not ready for writing")
	}

	if t.writer == nil {
		return types.NewCLIConnectionError("stdin writer not initialized")
	}

	t.logger.Debug("Sending message to CLI stdin")

	// Write JSON line (includes newline and flush)
	if err := t.writer.WriteLine(data); err != nil {
		t.ready = false
		t.err = types.NewCLIConnectionErrorWithCause("failed to write to subprocess stdin", err)
		t.logger.Error("failed to write to CLI stdin", zap.Error(err))
		return t.err
	}

	return nil
}

// ReadMessages returns a channel of incoming messages from the subprocess.
// The channel is closed when the subprocess exits or an error occurs.
func (t *SubprocessCLITransport) ReadMessages(ctx context.Context) <-chan types.Message {
	return t.messages
}
