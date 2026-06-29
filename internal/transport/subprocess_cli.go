package transport

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

const (
	// SDKVersion is the version identifier for this SDK
	SDKVersion = "0.1.0"

	// ShutdownGrace is the time allowed for the subprocess to exit cleanly
	// after stdin is closed before interrupt/cancel escalation.
	ShutdownGrace = 3 * time.Second

	// TerminateGrace is the time allowed after interrupt/cancel before kill.
	TerminateGrace = 2 * time.Second

	// StderrRingSize is the bounded diagnostic stderr tail retained after close.
	StderrRingSize = 64 * 1024

	stderrReadChunkSize = 32 * 1024
)

// getExitCode extracts the numeric exit code from an *exec.ExitError.
// Returns -1 for any other error type.
func getExitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

// SubprocessCLITransport implements Transport using a Claude Code CLI subprocess.
// It manages the subprocess lifecycle, stdin/stdout/stderr pipes, and message streaming.
type SubprocessCLITransport struct {
	cliPath         string
	cwd             string
	env             map[string]string
	logger          *log.Logger
	resumeSessionID string                    // Optional session ID to resume conversation
	options         *types.ClaudeAgentOptions // Options for CLI configuration

	cmd           *exec.Cmd
	customProcess types.SpawnedProcess // Non-nil when using custom spawner
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	stderr        io.ReadCloser
	stderrTail    *ringBuffer
	stderrDone    chan struct{}

	ctx    context.Context
	cancel context.CancelFunc

	// Message streaming
	messages chan types.Message

	// Writer for stdin
	writer *JSONLineWriter

	// procDone is closed by the watcher goroutine after cmd.Wait() / customProcess.Wait() returns.
	// closeExecCommand / closeCustomProcess drains this channel instead of spawning a second Wait().
	procDone chan struct{}

	// wg tracks the readStderr goroutine so Close() can wait for it to finish.
	wg sync.WaitGroup

	// Error tracking
	mu    sync.Mutex
	err   error
	ready bool

	// parseErrorBackoff returns the delay after N consecutive CLI JSON parse
	// errors. Tests replace this with a short deterministic backoff; production
	// uses the exponential backoff returned by defaultParseErrorBackoff.
	parseErrorBackoff func(consecutive uint) time.Duration

	// thinkingDisplaySupported gates --thinking-display emission. The flag was
	// added after the SDK's minimum supported Claude CLI version, so Connect()
	// probes the installed CLI when a caller requests display control.
	thinkingDisplaySupported bool

	// agentProgressSummariesSupported and subagentExecutionSupported gate emission of
	// the experimental --agent-progress-summaries / --subagent-execution flags so the
	// transport never sends a flag the installed CLI rejects (which crashes Connect).
	// Resolved once in Connect from the detected CLI version (or assumed true for a
	// custom spawner, which the consumer owns), and read by buildCommandArgs ONLY when
	// the matching option is requested — see the constructor for the full invariant.
	agentProgressSummariesSupported bool
	subagentExecutionSupported      bool

	// shutdownRequested is set before stdin is closed during Close. Watchers use
	// it to distinguish expected close exits from actionable subprocess crashes.
	shutdownRequested bool
}

// NewSubprocessCLITransport creates a new transport instance.
// The cliPath should point to the claude binary.
// The cwd is the working directory for the subprocess (empty string uses current directory).
// The env map contains additional environment variables to set for the subprocess.
// The logger is used for debug/diagnostic output.
// The resumeSessionID is an optional session ID to resume a previous conversation.
// The options contains configuration for the CLI.
func NewSubprocessCLITransport(cliPath, cwd string, env map[string]string, logger *log.Logger, resumeSessionID string, options *types.ClaudeAgentOptions) *SubprocessCLITransport {
	return &SubprocessCLITransport{
		cliPath:                  cliPath,
		cwd:                      cwd,
		env:                      env,
		logger:                   logger,
		resumeSessionID:          resumeSessionID,
		options:                  options,
		messages:                 make(chan types.Message, 10), // Buffered channel for smooth streaming
		stderrTail:               newRingBuffer(StderrRingSize),
		parseErrorBackoff:        defaultParseErrorBackoff,
		thinkingDisplaySupported: true,
		// INVARIANT: these are read by buildCommandArgs ONLY when the matching
		// option (AgentProgressSummaries / SubagentExecution) is requested. Before
		// that read, Connect always resolves them — detectExperimentalFlagSupport
		// sets them false-first then to the version's true support, or the custom-
		// spawner path assumes true (the consumer owns that CLI). So when a flag is
		// requested against an unverified CLI the value is the detected one, never
		// this default; when no flag is requested the default is simply never read.
		// The default is therefore inert for emission and only seeds the
		// build-args-without-Connect unit tests; keep it true so those tests
		// (which don't call Connect) see "supported".
		agentProgressSummariesSupported: true,
		subagentExecutionSupported:      true,
	}
}

// Connect starts the Claude Code CLI subprocess and establishes communication pipes.
// It launches the subprocess with "agent --stdio" arguments and sets up the environment.
func (t *SubprocessCLITransport) Connect(ctx context.Context) (err error) {
	t.mu.Lock()

	if t.cmd != nil || t.customProcess != nil {
		t.mu.Unlock()
		return nil // Already connected — no telemetry for a no-op
	}

	// Emit connect telemetry once the attempt completes. Registered before the
	// unlock defer so it runs AFTER t.mu is released (never call the Observer under
	// the lock). err is the named return, so the deferred closure sees the outcome.
	connectStart := time.Now()
	defer func() { t.observer().OnConnect(time.Since(connectStart), err) }()
	defer t.mu.Unlock()

	t.logger.Debug("starting Claude CLI subprocess", zap.String("cli_path", t.cliPath))

	// Re-create the messages channel for this connection attempt.
	// A previous Connect/Close cycle may have closed the old channel.
	t.messages = make(chan types.Message, 10)
	t.stderrTail = newRingBuffer(StderrRingSize)
	t.stderrDone = nil
	t.shutdownRequested = false
	t.err = nil

	// Create cancellable context. connCtx is a local handle on the same context
	// stored in t.ctx; passing the local (rather than the t.ctx field) to the
	// connect helpers keeps the inheritance from ctx statically visible.
	connCtx, connCancel := context.WithCancel(ctx)
	t.ctx, t.cancel = connCtx, connCancel

	usesCustomSpawner := t.usesCustomSpawner()
	wantsThinkingDisplay := t.wantsThinkingDisplay()
	if wantsThinkingDisplay && !usesCustomSpawner {
		t.thinkingDisplaySupported = t.detectThinkingDisplaySupport(ctx)
	} else {
		t.thinkingDisplaySupported = true
	}

	// Resolve experimental-flag capabilities before buildCommandArgs runs. Custom
	// spawners are the consumer's responsibility, so assume supported there.
	if usesCustomSpawner {
		t.agentProgressSummariesSupported = true
		t.subagentExecutionSupported = true
	} else {
		t.detectExperimentalFlagSupport(ctx)
	}

	// Build command arguments
	args := t.buildCommandArgs()

	// Log the command for debugging. Redact sensitive flag VALUES (inline
	// --mcp-config JSON, delegate --sock path) — defense-in-depth so they
	// never land in a Debug args dump. The actual spawned argv (args) is
	// unchanged; redactArgsForLog returns a copy and never mutates args.
	t.logger.Debug("Claude CLI command", zap.String("cli_path", t.cliPath), zap.Strings("args", redactArgsForLog(args)))

	// Build environment variables map
	envMap := t.buildEnvMap()

	// Check if a custom process spawner is provided
	if t.options != nil && t.options.SpawnProcess != nil {
		err = t.connectWithCustomSpawner(connCtx, args, envMap)
	} else {
		// connCtx is the cancellable child of ctx established above; pass it so the
		// subprocess and reader goroutines are bound to the connection lifecycle.
		err = t.connectWithExecCommand(connCtx, args, envMap)
	}
	if err != nil {
		t.cancel()
		t.cancel = nil
		return err
	}
	return nil
}
