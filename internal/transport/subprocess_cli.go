package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

const (
	// SDKVersion is the version identifier for this SDK
	SDKVersion = "0.1.0"
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
		cliPath:         cliPath,
		cwd:             cwd,
		env:             env,
		logger:          logger,
		resumeSessionID: resumeSessionID,
		options:         options,
		messages:        make(chan types.Message, 10), // Buffered channel for smooth streaming
	}
}

// Connect starts the Claude Code CLI subprocess and establishes communication pipes.
// It launches the subprocess with "agent --stdio" arguments and sets up the environment.
func (t *SubprocessCLITransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cmd != nil || t.customProcess != nil {
		return nil // Already connected
	}

	t.logger.Debug("starting Claude CLI subprocess", zap.String("cli_path", t.cliPath))

	// Re-create the messages channel for this connection attempt.
	// A previous Connect/Close cycle may have closed the old channel.
	t.messages = make(chan types.Message, 10)

	// Create cancellable context
	t.ctx, t.cancel = context.WithCancel(ctx)

	// Try warm pool first — if a pre-warmed process is available, use it
	if warm := ConsumeWarmProcess(); warm != nil && warm.IsAlive() {
		t.logger.Debug("using pre-warmed subprocess from Startup()")
		err := t.connectWithWarmProcess(warm)
		if err == nil {
			return nil
		}
		// Warm process failed — fall through to normal spawn
		t.logger.Debug("warm process unusable, falling through to normal spawn", zap.Error(err))
	}

	// Build command arguments
	args := t.buildCommandArgs()

	// Log the full command for debugging
	t.logger.Debug("Claude CLI command", zap.String("cli_path", t.cliPath), zap.Strings("args", args))

	// Build environment variables map
	envMap := t.buildEnvMap()

	// Check if a custom process spawner is provided
	var err error
	if t.options != nil && t.options.SpawnProcess != nil {
		err = t.connectWithCustomSpawner(ctx, args, envMap)
	} else {
		err = t.connectWithExecCommand(args, envMap)
	}
	if err != nil {
		t.cancel()
		t.cancel = nil
		return err
	}
	return nil
}

// connectWithCustomSpawner uses the user-provided ProcessSpawner to create the process.
func (t *SubprocessCLITransport) connectWithCustomSpawner(ctx context.Context, args []string, envMap map[string]string) error {
	spawnOpts := types.SpawnOptions{
		Command: t.cliPath,
		Args:    args,
		CWD:     t.cwd,
		Env:     envMap,
	}

	process, err := t.options.SpawnProcess(ctx, spawnOpts)
	if err != nil {
		t.logger.Error("custom process spawner failed", zap.Error(err))
		return types.NewCLIConnectionErrorWithCause("custom process spawner failed", err)
	}

	// Validate the spawned process provides required pipes
	if process.Stdin() == nil {
		return types.NewCLIConnectionError("custom spawner returned nil stdin")
	}
	if process.Stdout() == nil {
		return types.NewCLIConnectionError("custom spawner returned nil stdout")
	}
	if process.Stderr() == nil {
		return types.NewCLIConnectionError("custom spawner returned nil stderr")
	}

	t.customProcess = process
	t.stdin = process.Stdin()
	t.stdout = process.Stdout()
	t.stderr = process.Stderr()

	t.logger.Debug("Custom-spawned process connected successfully")

	// Create JSON line writer for stdin
	t.writer = NewJSONLineWriter(t.stdin)

	// Launch watcher goroutine: mirrors the exec-command path.
	// customProcess.Wait() is called exactly once here; closeCustomProcess drains procDone.
	t.procDone = make(chan struct{})
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		waitErr := t.customProcess.Wait()
		close(t.procDone) // signal Close() that Wait() is done — BEFORE acquiring mutex
		t.mu.Lock()
		wasReady := t.ready
		t.ready = false
		if waitErr != nil && t.err == nil {
			t.err = types.NewProcessErrorWithCode("subprocess exited unexpectedly", t.customProcess.ExitCode())
		}
		cancelFn := t.cancel // capture under lock to avoid race with Close()
		t.mu.Unlock()
		if wasReady {
			if cancelFn != nil {
				cancelFn()
			}
		}
	}()

	// Launch context monitor goroutine for custom spawner (Bug C14).
	// Unlike exec.CommandContext, custom processes don't auto-kill on context cancel.
	// This goroutine kills the process when the transport context is cancelled,
	// unblocking Wait() and the pipe readers.
	capturedCtx := t.ctx
	capturedProcess := t.customProcess
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		select {
		case <-capturedCtx.Done():
			if err := capturedProcess.Kill(); err != nil {
				t.logger.Debug("context cancel: process kill returned error (process may have already exited)",
					zap.Error(err))
			}
		case <-t.procDone:
			// Process already exited — nothing to do
		}
	}()

	// Launch message reader loop in goroutine (tracked by wg for clean shutdown).
	// Capture stdout/stderr here (under caller's mutex) to avoid a race with
	// a subsequent Connect() call overwriting the struct fields.
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.messageReaderLoop(t.ctx, t.stdout)
	}()

	// Launch stderr reader for debugging (tracked by wg for clean shutdown)
	t.wg.Add(1)
	go t.readStderr(t.ctx, t.stderr)

	// Mark as ready
	t.ready = true
	t.logger.Debug("Transport ready for communication")

	return nil
}

// connectWithExecCommand uses the default exec.Command to create the process.
func (t *SubprocessCLITransport) connectWithExecCommand(args []string, envMap map[string]string) error {
	// Create command with arguments
	t.cmd = exec.CommandContext(t.ctx, t.cliPath, args...)

	// Set working directory if provided
	if t.cwd != "" {
		t.cmd.Dir = t.cwd
	}

	// Set up environment variables
	t.cmd.Env = os.Environ()
	for key, value := range envMap {
		t.cmd.Env = append(t.cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Set up pipes
	var err error

	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		return types.NewCLIConnectionErrorWithCause("failed to create stdin pipe", err)
	}

	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		_ = t.stdin.Close()
		t.stdin = nil
		return types.NewCLIConnectionErrorWithCause("failed to create stdout pipe", err)
	}

	t.stderr, err = t.cmd.StderrPipe()
	if err != nil {
		_ = t.stdout.Close()
		t.stdout = nil
		_ = t.stdin.Close()
		t.stdin = nil
		return types.NewCLIConnectionErrorWithCause("failed to create stderr pipe", err)
	}

	// Start the process
	if err := t.cmd.Start(); err != nil {
		_ = t.stderr.Close()
		t.stderr = nil
		_ = t.stdout.Close()
		t.stdout = nil
		_ = t.stdin.Close()
		t.stdin = nil
		t.logger.Error("failed to start subprocess", zap.Error(err))
		return types.NewCLIConnectionErrorWithCause("failed to start subprocess", err)
	}
	t.logger.Debug("CLI subprocess started successfully", zap.Int("pid", t.cmd.Process.Pid))

	// Create JSON line writer for stdin
	t.writer = NewJSONLineWriter(t.stdin)

	// Launch watcher goroutine: calls cmd.Wait() exactly once, sets ready=false
	// immediately on subprocess exit, and cancels the transport context so that
	// messageReaderLoop and readStderr exit cleanly.
	//
	// IMPORTANT: procDone is closed BEFORE acquiring t.mu to prevent a deadlock
	// where Close() holds t.mu while waiting for procDone, and the watcher needs
	// t.mu to close procDone.
	t.procDone = make(chan struct{})
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		waitErr := t.cmd.Wait()
		close(t.procDone) // signal Close() that Wait() is done — BEFORE acquiring mutex
		t.mu.Lock()
		wasReady := t.ready
		t.ready = false
		if waitErr != nil && t.err == nil {
			t.err = types.NewProcessErrorWithCode("subprocess exited unexpectedly", getExitCode(waitErr))
		}
		cancelFn := t.cancel // capture under lock to avoid race with Close()
		t.mu.Unlock()
		if wasReady {
			if cancelFn != nil {
				cancelFn()
			}
		}
	}()

	// Launch message reader loop in goroutine (tracked by wg for clean shutdown).
	// Capture stdout/stderr here (under caller's mutex) to avoid a race with
	// a subsequent Connect() call overwriting the struct fields.
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.messageReaderLoop(t.ctx, t.stdout)
	}()

	// Launch stderr reader for debugging (tracked by wg for clean shutdown)
	t.wg.Add(1)
	go t.readStderr(t.ctx, t.stderr)

	// Mark as ready
	t.ready = true
	t.logger.Debug("Transport ready for communication")

	return nil
}

// buildEnvMap constructs the environment variable map for the subprocess.
func (t *SubprocessCLITransport) buildEnvMap() map[string]string {
	envMap := make(map[string]string)

	// SDK-specific variables
	envMap["CLAUDE_CODE_ENTRYPOINT"] = "agent"
	envMap["CLAUDE_AGENT_SDK_VERSION"] = SDKVersion

	// Model environment variable
	if t.options != nil && t.options.Model != nil {
		envMap["ANTHROPIC_MODEL"] = *t.options.Model
		t.logger.Debug("setting ANTHROPIC_MODEL environment variable", zap.String("model", *t.options.Model))
	}

	// Base URL environment variable
	if t.options != nil && t.options.BaseURL != nil {
		envMap["ANTHROPIC_BASE_URL"] = *t.options.BaseURL
		t.logger.Debug("setting ANTHROPIC_BASE_URL environment variable", zap.String("base_url", *t.options.BaseURL))
	}

	// Custom environment variables (can override the above)
	for key, value := range t.env {
		envMap[key] = value
		t.logger.Debug("setting custom environment variable", zap.String("key", key))
	}

	return envMap
}

// maxParseErrorBackoff is the ceiling for exponential backoff on consecutive parse errors.
const maxParseErrorBackoff = 30 * time.Second

// maxConsecutiveParseErrors is the number of consecutive JSON parse errors
// before messageReaderLoop gives up and exits. This prevents the SDK from
// stalling forever when the subprocess sends garbage.
const maxConsecutiveParseErrors uint = 6

// messageReaderLoop reads JSON lines from stdout and parses them into messages.
// It runs in a goroutine and sends messages to the messages channel.
// It respects context cancellation and closes the messages channel when done.
// stdout is passed as a parameter to avoid a data race with Connect() overwriting t.stdout.
//
// Note on context cancellation: ReadLine() blocks on stdout, which cannot be interrupted
// by context cancel directly. Instead, when the context is cancelled, the process is killed
// (via the context monitor goroutine or exec.CommandContext), which closes the stdout pipe,
// causing ReadLine() to return io.EOF and exit the loop.
//
// Parse errors trigger exponential backoff (1s, 2s, 4s, ... up to 30s) to prevent
// CPU spin on repeated invalid JSON. The backoff counter resets on successful parse.
func (t *SubprocessCLITransport) messageReaderLoop(ctx context.Context, stdout io.Reader) {
	ch := t.messages
	defer close(ch)

	t.logger.Debug("Message reader loop started")
	reader := NewJSONLineReader(stdout)

	var consecutiveParseErrors uint

	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			t.logger.Debug("Message reader loop stopped: context cancelled")
			return
		default:
		}

		// Read next JSON line
		line, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				t.logger.Debug("Message reader loop stopped: EOF from CLI")
				// Normal end of stream
				return
			}

			// If the transport context is already cancelled, read errors
			// are expected (pipe closed during shutdown) — log at Debug.
			// Use the passed-in ctx (not t.ctx) to avoid a data race with
			// Connect() overwriting t.ctx under t.mu.
			if ctx.Err() != nil {
				t.logger.Debug("message reader loop stopped during shutdown", zap.Error(err))
				return
			}

			t.logger.Error("failed to read from CLI stdout", zap.Error(err))
			// Store error and return
			t.OnError(types.NewJSONDecodeErrorWithCause(
				"failed to read JSON line from subprocess",
				string(line),
				err,
			))
			return
		}

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Parse JSON into message
		msg, err := types.UnmarshalMessage(line)
		if err != nil {
			consecutiveParseErrors++
			t.logger.Warn("failed to parse message from CLI",
				zap.Error(err),
				zap.Uint("consecutive_errors", consecutiveParseErrors),
			)

			// Exit after too many consecutive parse failures — subprocess is broken.
			if consecutiveParseErrors >= maxConsecutiveParseErrors {
				t.logger.Error("too many consecutive parse errors, closing message reader",
					zap.Uint("consecutive_errors", consecutiveParseErrors),
				)
				t.OnError(fmt.Errorf("transport.messageReaderLoop: %d consecutive parse errors, giving up", consecutiveParseErrors))
				return
			}

			// Store parse error but continue reading after backoff
			t.OnError(err)

			// Exponential backoff: 1s, 2s, 4s, 8s, ..., capped at maxParseErrorBackoff.
			// Use time.NewTimer with select on ctx.Done() to remain cancellable.
			// Cap shift count to prevent integer overflow. 2^5 = 32s already
			// exceeds maxParseErrorBackoff (30s), so clamping at 5 is sufficient.
			shift := consecutiveParseErrors - 1
			if shift > 5 {
				shift = 5
			}
			backoff := time.Duration(1<<shift) * time.Second
			if backoff > maxParseErrorBackoff {
				backoff = maxParseErrorBackoff
			}
			t.logger.Debug("parse error backoff",
				zap.Duration("backoff", backoff),
				zap.Uint("consecutive_errors", consecutiveParseErrors),
			)
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				t.logger.Debug("Message reader loop stopped during backoff: context cancelled")
				return
			case <-timer.C:
				// Backoff complete, continue reading
			}
			continue
		}

		// Successful parse — reset backoff counter
		consecutiveParseErrors = 0

		t.logger.Debug("received message from CLI", zap.String("type", msg.GetMessageType()))

		// Send message to channel (respect context cancellation and process exit)
		select {
		case <-ctx.Done():
			return
		case ch <- msg:
			// Message sent successfully
		case <-t.procDone:
			return
		}
	}
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

// connectWithWarmProcess uses a pre-warmed subprocess from the warm pool.
// The warm process was spawned by Startup() and has stdin/stdout/stderr pipes ready.
func (t *SubprocessCLITransport) connectWithWarmProcess(warm *WarmProcess) error {
	t.cmd = warm.Cmd
	t.stdin = warm.Stdin
	t.stdout = warm.Stdout
	t.stderr = warm.Stderr

	// Create JSON line writer for stdin
	t.writer = NewJSONLineWriter(t.stdin)

	// Use the warm process's done channel as procDone
	t.procDone = warm.Done

	// Launch watcher goroutine: mirrors connectWithExecCommand.
	// The warm process background goroutine already called cmd.Wait() and closed Done.
	// This goroutine watches for procDone to set ready=false and cancel context.
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		<-t.procDone
		t.mu.Lock()
		wasReady := t.ready
		t.ready = false
		if t.err == nil {
			exitCode := 0
			if t.cmd.ProcessState != nil {
				exitCode = t.cmd.ProcessState.ExitCode()
			}
			t.err = types.NewProcessErrorWithCode("subprocess exited unexpectedly", exitCode)
		}
		cancelFn := t.cancel
		t.mu.Unlock()
		if wasReady {
			if cancelFn != nil {
				cancelFn()
			}
		}
	}()

	// Launch message reader loop
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.messageReaderLoop(t.ctx, t.stdout)
	}()

	// Launch stderr reader
	t.wg.Add(1)
	go t.readStderr(t.ctx, t.stderr)

	// Mark as ready
	t.ready = true
	t.logger.Debug("Transport ready via warm process")

	return nil
}

// buildCommandArgs builds the command line arguments for the CLI subprocess.
// This is extracted into a separate method to allow for testing.
func (t *SubprocessCLITransport) buildCommandArgs() []string {
	args := []string{
		"--input-format=stream-json",
		"--output-format=stream-json",
		"--verbose",
	}

	// Add permission prompt tool if specified
	if t.options != nil && t.options.PermissionPromptToolName != nil {
		args = append(args, "--permission-prompt-tool", *t.options.PermissionPromptToolName)
		t.logger.Debug("setting permission prompt tool", zap.String("tool", *t.options.PermissionPromptToolName))
	}

	// Add permission mode if specified
	if t.options != nil && t.options.PermissionMode != nil {
		args = append(args, "--permission-mode", string(*t.options.PermissionMode))
		t.logger.Debug("setting permission mode", zap.String("mode", string(*t.options.PermissionMode)))
	}

	// Add system prompt - always pass the flag to match Python SDK behavior
	// When nil, pass empty string to prevent unintended Claude Code defaults
	if t.options != nil {
		if t.options.SystemPrompt == nil {
			// Default to empty system prompt when not specified
			args = append(args, "--system-prompt", "")
			t.logger.Debug("Setting empty system prompt (default)")
		} else if promptStr, ok := t.options.SystemPrompt.(string); ok {
			// Handle string prompt
			args = append(args, "--system-prompt", promptStr)
			t.logger.Debug("setting system prompt", zap.String("prompt", promptStr))
		} else if preset, ok := t.options.SystemPrompt.(types.SystemPromptPreset); ok {
			// Handle preset case - append to default Claude Code prompt
			if preset.Append != nil {
				args = append(args, "--append-system-prompt", *preset.Append)
				t.logger.Debug("appending to system prompt preset", zap.String("append", *preset.Append))
			}
		}
	} else {
		// No options provided, use empty system prompt
		args = append(args, "--system-prompt", "")
		t.logger.Debug("Setting empty system prompt (no options)")
	}

	// Add model if specified
	if t.options != nil && t.options.Model != nil {
		args = append(args, "--model", *t.options.Model)
		t.logger.Debug("setting model", zap.String("model", *t.options.Model))
	}

	// Add --resume flag if resuming a conversation
	if t.resumeSessionID != "" {
		args = append(args, "--resume", t.resumeSessionID)
		t.logger.Debug("resuming Claude CLI conversation", zap.String("session_id", t.resumeSessionID))
	}

	// Add --fork-session flag if forking a resumed session
	if t.options != nil && t.options.ForkSession {
		args = append(args, "--fork-session")
		t.logger.Debug("Forking resumed session to new session ID")
	}

	// Add permission bypass flags if enabled
	if t.options != nil {
		// Must set allow flag first (acts as safety switch)
		if t.options.AllowDangerouslySkipPermissions {
			args = append(args, "--allow-dangerously-skip-permissions")
			t.logger.Debug("Allowing permission bypass (safety switch enabled)")

			// Only add skip flag if allow flag is also set
			if t.options.DangerouslySkipPermissions {
				args = append(args, "--dangerously-skip-permissions")
				t.logger.Debug("DANGER: Bypassing all permissions - use only in sandboxed environments!")
			}
		}
	}

	// Add extended thinking token limit if specified
	if t.options != nil && t.options.MaxThinkingTokens != nil {
		args = append(args, "--max-thinking-tokens", fmt.Sprintf("%d", *t.options.MaxThinkingTokens))
		t.logger.Debug("setting max thinking tokens", zap.Int("max_thinking_tokens", *t.options.MaxThinkingTokens))
	}

	// Add budget limit if specified
	if t.options != nil && t.options.MaxBudgetUSD != nil {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", *t.options.MaxBudgetUSD))
		t.logger.Debug("setting max budget", zap.Float64("max_budget_usd", *t.options.MaxBudgetUSD))
	}

	// Add beta feature flags if specified
	if t.options != nil && len(t.options.Betas) > 0 {
		for _, beta := range t.options.Betas {
			args = append(args, "--betas", beta)
			t.logger.Debug("adding beta feature flag", zap.String("beta", beta))
		}
	}

	// Add plugin directories
	if t.options != nil && len(t.options.Plugins) > 0 {
		for _, plugin := range t.options.Plugins {
			if plugin.Type == "local" {
				args = append(args, "--plugin-dir", plugin.Path)
				t.logger.Debug("adding plugin directory", zap.String("path", plugin.Path))
			} else {
				// This shouldn't happen if NewPluginConfig is used, but handle it anyway
				t.logger.Warn("skipping unsupported plugin type", zap.String("type", plugin.Type))
			}
		}
	}

	// Add setting sources if specified (enables local slash commands, CLAUDE.md, etc.)
	if t.options != nil && len(t.options.SettingSources) > 0 {
		sources := make([]string, len(t.options.SettingSources))
		for i, src := range t.options.SettingSources {
			sources[i] = string(src)
		}
		args = append(args, "--setting-sources", joinStrings(sources, ","))
		t.logger.Debug("setting sources", zap.String("sources", joinStrings(sources, ",")))
	}

	// Add agents if specified
	if t.options != nil && len(t.options.Agents) > 0 {
		agentsJSON := make(map[string]map[string]interface{})

		for name, agent := range t.options.Agents {
			agentMap := make(map[string]interface{})
			agentMap["description"] = agent.Description
			agentMap["prompt"] = agent.Prompt

			// Add optional fields only if set
			if len(agent.Tools) > 0 {
				agentMap["tools"] = agent.Tools
			}
			if agent.Model != nil {
				agentMap["model"] = *agent.Model
			}
			if agent.ExecutionMode != nil {
				agentMap["execution_mode"] = string(*agent.ExecutionMode)
			}
			if agent.Timeout != nil {
				agentMap["timeout"] = *agent.Timeout
			}
			if agent.MaxTurns != nil {
				agentMap["max_turns"] = *agent.MaxTurns
			}
			if len(agent.DisallowedTools) > 0 {
				agentMap["disallowed_tools"] = agent.DisallowedTools
			}
			if len(agent.McpServers) > 0 {
				agentMap["mcp_servers"] = agent.McpServers
			}
			if len(agent.Skills) > 0 {
				agentMap["skills"] = agent.Skills
			}
			if agent.CriticalSystemReminder != nil {
				agentMap["criticalSystemReminder_EXPERIMENTAL"] = *agent.CriticalSystemReminder
			}

			agentsJSON[name] = agentMap
		}

		agentsJSONBytes, err := json.Marshal(agentsJSON)
		if err != nil {
			t.logger.Warn("failed to marshal agents to JSON", zap.Error(err))
		} else {
			args = append(args, "--agents", string(agentsJSONBytes))
			t.logger.Debug("agents configuration", zap.String("agents_json", string(agentsJSONBytes)))
		}
	}

	// Add effort level if specified
	if t.options != nil && t.options.Effort != nil {
		args = append(args, "--effort", string(*t.options.Effort))
		t.logger.Debug("setting effort level", zap.String("effort", string(*t.options.Effort)))
	}

	// Add fallback model if specified
	if t.options != nil && t.options.FallbackModel != nil {
		args = append(args, "--fallback-model", *t.options.FallbackModel)
		t.logger.Debug("setting fallback model", zap.String("fallback_model", *t.options.FallbackModel))
	}

	// Add session ID if specified
	if t.options != nil && t.options.SessionID != nil {
		args = append(args, "--session-id", *t.options.SessionID)
		t.logger.Debug("setting session ID", zap.String("session_id", *t.options.SessionID))
	}

	// Add no-session-persistence flag if PersistSession is explicitly false
	if t.options != nil && t.options.PersistSession != nil && !*t.options.PersistSession {
		args = append(args, "--no-session-persistence")
		t.logger.Debug("Disabling session persistence")
	}

	// Add JSON schema output format if specified
	if t.options != nil && t.options.OutputFormat != nil {
		schemaJSON, err := json.Marshal(t.options.OutputFormat)
		if err != nil {
			t.logger.Warn("failed to marshal output format to JSON", zap.Error(err))
		} else {
			args = append(args, "--json-schema", string(schemaJSON))
			t.logger.Debug("setting JSON schema output format", zap.String("schema", string(schemaJSON)))
		}
	}

	// Build and add settings JSON if needed (thinking, sandbox, file checkpointing)
	if t.options != nil {
		settingsJSON := t.buildSettingsJSON()
		if settingsJSON != "" {
			args = append(args, "--settings", settingsJSON)
			t.logger.Debug("setting settings JSON", zap.String("settings", settingsJSON))
		}
	}

	// When file checkpointing is enabled, also request user message UUIDs
	// for checkpoint targeting (rewind, branch-at-message).
	if t.options != nil && t.options.EnableFileCheckpointing {
		args = append(args, "--replay-user-messages")
	}

	// Add subagent execution configuration if specified
	if t.options != nil && t.options.SubagentExecution != nil {
		subagentJSON := make(map[string]interface{})

		if t.options.SubagentExecution.MultiInvocation != "" {
			subagentJSON["multi_invocation"] = string(t.options.SubagentExecution.MultiInvocation)
		}
		if t.options.SubagentExecution.MaxConcurrent > 0 {
			subagentJSON["max_concurrent"] = t.options.SubagentExecution.MaxConcurrent
		}
		if t.options.SubagentExecution.ErrorHandling != "" {
			subagentJSON["error_handling"] = string(t.options.SubagentExecution.ErrorHandling)
		}

		if len(subagentJSON) > 0 {
			subagentJSONBytes, err := json.Marshal(subagentJSON)
			if err != nil {
				t.logger.Warn("failed to marshal subagent execution config to JSON", zap.Error(err))
			} else {
				args = append(args, "--subagent-execution", string(subagentJSONBytes))
				t.logger.Debug("subagent execution configuration", zap.String("config", string(subagentJSONBytes)))
			}
		}
	}

	// Add --resume-session-at flag
	if t.options != nil && t.options.ResumeSessionAt != nil {
		args = append(args, "--resume-session-at", *t.options.ResumeSessionAt)
		t.logger.Debug("setting resume session at", zap.String("resume_at", *t.options.ResumeSessionAt))
	}

	// Add --tools flag ([]string joined by comma, or JSON for preset)
	if t.options != nil && t.options.Tools != nil {
		switch v := t.options.Tools.(type) {
		case []string:
			if len(v) > 0 {
				args = append(args, "--tools", joinStrings(v, ","))
				t.logger.Debug("setting tools", zap.Strings("tools", v))
			}
		default:
			// Serialize non-string-slice values as JSON (e.g., preset objects)
			toolsJSON, err := json.Marshal(v)
			if err != nil {
				t.logger.Warn("failed to marshal tools to JSON", zap.Error(err))
			} else {
				args = append(args, "--tools", string(toolsJSON))
				t.logger.Debug("setting tools (JSON)", zap.String("tools_json", string(toolsJSON)))
			}
		}
	}

	// Add --debug-file flag
	if t.options != nil && t.options.DebugFile != nil {
		args = append(args, "--debug-file", *t.options.DebugFile)
		t.logger.Debug("setting debug file", zap.String("debug_file", *t.options.DebugFile))
	}

	// Add --strict-mcp-config flag
	if t.options != nil && t.options.StrictMcpConfig {
		args = append(args, "--strict-mcp-config")
		t.logger.Debug("Enabling strict MCP config")
	}

	// Add --task-budget flag if specified
	if t.options != nil && t.options.TaskBudget != nil {
		args = append(args, "--task-budget", fmt.Sprintf("%.2f", *t.options.TaskBudget))
		t.logger.Debug("setting task budget", zap.Float64("task_budget", *t.options.TaskBudget))
	}

	// Add --agent-progress-summaries flag if enabled
	if t.options != nil && t.options.AgentProgressSummaries {
		args = append(args, "--agent-progress-summaries")
		t.logger.Debug("Enabling agent progress summaries")
	}

	return args
}

// joinStrings joins strings with a separator (avoiding strings import)
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// buildSettingsJSON constructs the --settings JSON string from typed option fields.
// It merges typed fields (Thinking, Sandbox, EnableFileCheckpointing) on top of
// any user-provided Settings string. Typed fields take precedence on conflict.
func (t *SubprocessCLITransport) buildSettingsJSON() string {
	hasThinking := t.options.Thinking != nil
	hasSandbox := t.options.Sandbox != nil
	hasCheckpointing := t.options.EnableFileCheckpointing
	hasToolConfig := t.options.ToolConfig != nil
	hasIncludeHookEvents := t.options.IncludeHookEvents

	if !hasThinking && !hasSandbox && !hasCheckpointing && !hasToolConfig && !hasIncludeHookEvents && t.options.Settings == nil {
		return ""
	}

	// Start with user-provided settings as base (if any)
	settings := make(map[string]interface{})
	if t.options.Settings != nil && *t.options.Settings != "" {
		if err := json.Unmarshal([]byte(*t.options.Settings), &settings); err != nil {
			t.logger.Warn("failed to parse user settings JSON, using typed fields only", zap.Error(err))
		}
	}

	// If no typed fields are set, just return the original settings string
	if !hasThinking && !hasSandbox && !hasCheckpointing && !hasToolConfig && !hasIncludeHookEvents {
		if t.options.Settings != nil {
			return *t.options.Settings
		}
		return ""
	}

	// Typed fields override user-provided settings
	if hasThinking {
		settings["thinking"] = t.options.Thinking
	}
	if hasSandbox {
		settings["sandbox"] = t.options.Sandbox
	}
	if hasCheckpointing {
		settings["enableFileCheckpointing"] = true
	}
	if hasToolConfig {
		settings["toolConfig"] = t.options.ToolConfig
	}
	if hasIncludeHookEvents {
		settings["includeHookEvents"] = true
	}

	result, err := json.Marshal(settings)
	if err != nil {
		t.logger.Warn("failed to marshal settings JSON", zap.Error(err))
		return ""
	}
	return string(result)
}

// Close terminates the subprocess and cleans up all resources.
// It closes stdin first to allow the subprocess to exit gracefully,
// then waits for exit. The transport context is cancelled only after
// the process has exited (cleanup) or as a fallback kill if the
// caller's context expires before the process exits.
func (t *SubprocessCLITransport) Close(ctx context.Context) error {
	t.mu.Lock()

	if t.cmd == nil && t.customProcess == nil {
		t.mu.Unlock()
		return nil // Not connected
	}

	t.logger.Debug("Closing CLI subprocess...")
	t.ready = false

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
	select {
	case <-ctx.Done():
		// Caller's timeout expired — force kill, then drain.
		_ = t.customProcess.Kill()
		if t.cancel != nil {
			t.cancel()
			t.cancel = nil
		}
		// Safe to block here while holding t.mu: the watcher goroutine guarantees
		// close(procDone) happens BEFORE it acquires t.mu, so no deadlock is possible.
		<-t.procDone
		t.customProcess = nil
		return types.NewProcessError("subprocess did not exit gracefully, killed")

	case <-t.procDone:
		// Watcher already called Wait() and closed procDone.
		if t.cancel != nil {
			t.cancel()
			t.cancel = nil
		}
	}

	t.customProcess = nil
	return nil
}

// closeExecCommand handles cleanup for exec.Command-spawned process.
// cmd.Wait() is owned by the watcher goroutine (launched in connectWithExecCommand).
// This method drains procDone — it must NOT call cmd.Wait() again.
func (t *SubprocessCLITransport) closeExecCommand(ctx context.Context) error {
	select {
	case <-ctx.Done():
		// Caller's timeout expired — force kill via context cancel, then drain.
		if t.cancel != nil {
			t.cancel()
			t.cancel = nil
		}
		// Safe to block here while holding t.mu: the watcher goroutine guarantees
		// close(procDone) happens BEFORE it acquires t.mu, so no deadlock is possible.
		<-t.procDone
		t.cmd = nil
		return types.NewProcessError("subprocess did not exit gracefully, killed")

	case <-t.procDone:
		// Watcher already called Wait() and closed procDone — just cancel context.
		if t.cancel != nil {
			t.cancel()
			t.cancel = nil
		}
	}

	t.cmd = nil
	return nil
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

// readStderr reads stderr output in a goroutine for debugging.
// This is a helper function for monitoring subprocess errors.
// It also parses known error patterns and stores them as typed errors.
// stderr is passed as a parameter to avoid a data race with Connect() overwriting t.stderr.
func (t *SubprocessCLITransport) readStderr(ctx context.Context, stderr io.ReadCloser) {
	defer t.wg.Done()

	if stderr == nil {
		return
	}

	// Determine if file logging is enabled via StderrLogFile option
	var logFile *os.File
	if t.options != nil && t.options.StderrLogFile != nil {
		// Resolve log file path
		logPath := *t.options.StderrLogFile
		if logPath == "" {
			// Use default location
			homeDir, err := os.UserHomeDir()
			if err != nil {
				t.logger.Warn("readStderr: could not determine home directory, stderr logging disabled",
					zap.Error(err))
				return
			}
			logPath = fmt.Sprintf("%s/.claude/agents_server/cli_stderr.log", homeDir)
		}

		// Create parent directory if it doesn't exist
		logDir := filepath.Dir(logPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr,
				"[SDK] Failed to create stderr log directory %s: %v\n"+
					"Stderr file logging disabled. To fix, create directory:\n"+
					"  mkdir -p %s\n",
				logDir, err, logDir)
		} else {
			// Try to open log file
			var err error
			logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr,
					"[SDK] Failed to open stderr log file %s: %v\n"+
						"Stderr file logging disabled. Possible fixes:\n"+
						"  1. Ensure directory exists: mkdir -p %s\n"+
						"  2. Check file permissions: chmod 644 %s\n"+
						"  3. Use custom path: opts.WithCustomStderrLogFile(\"/path/to/file.log\")\n",
					logPath, err, logDir, logPath)
			} else {
				t.logger.Debug("stderr file logging enabled", zap.String("path", logPath))
			}
		}
	}

	// Ensure cleanup if file was opened
	if logFile != nil {
		defer func() {
			_ = logFile.Close()
		}()
	}

	// Launch a goroutine that closes stderr when ctx is cancelled.
	// This unblocks ReadLine() if the subprocess hangs without closing stderr (Bug C16).
	// The goroutine exits when either ctx is cancelled (close stderr) or the read loop
	// finishes (readDone closed).
	readDone := make(chan struct{})
	defer close(readDone)
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		select {
		case <-ctx.Done():
			_ = stderr.Close()
		case <-readDone:
			// Read loop finished normally — nothing to do
		}
	}()

	reader := NewJSONLineReader(stderr)
	for {
		line, err := reader.ReadLine()
		if err != nil {
			return
		}

		// Check for context cancellation between reads
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Process stderr output
		if len(line) > 0 {
			stderrText := string(line)

			// Write to log file if enabled and file is open
			if logFile != nil {
				_, _ = fmt.Fprintf(logFile, "[Claude CLI stderr]: %s\n", stderrText)
				_ = logFile.Sync() // Flush to disk immediately
			}

			// Call stderr callback if configured (for runtime control)
			if t.options != nil && t.options.Stderr != nil {
				t.options.Stderr(stderrText)
			}

			// Parse known error patterns and create typed errors
			t.parseStderrError(stderrText)
		}
	}
}

// parseStderrError parses stderr text for known error patterns and stores typed errors.
func (t *SubprocessCLITransport) parseStderrError(stderrText string) {
	// Check for "No conversation found with session ID:" error
	if matched, sessionID := extractSessionNotFoundError(stderrText); matched {
		// Create typed error
		err := types.NewSessionNotFoundError(
			sessionID,
			"Claude CLI could not find this conversation. It may have been deleted or the CLI was reinstalled.",
		)

		// Store error for retrieval
		t.OnError(err)

		// Log it
		t.logger.Error("Claude session not found", zap.String("session_id", sessionID))
	}
}

// extractSessionNotFoundError checks if the stderr text contains a session not found error.
// Returns (true, sessionID) if matched, (false, "") otherwise.
func extractSessionNotFoundError(stderrText string) (bool, string) {
	// Pattern: "No conversation found with session ID: <uuid>"
	// Example: "No conversation found with session ID: 8587b432-e504-42c8-b9a7-e3fd0b4b2c60"
	const pattern = "No conversation found with session ID:"

	if idx := findSubstring(stderrText, pattern); idx >= 0 {
		// Extract session ID after the pattern
		sessionIDStart := idx + len(pattern)
		if sessionIDStart < len(stderrText) {
			// Trim whitespace and extract the session ID
			remaining := stderrText[sessionIDStart:]
			sessionID := trimWhitespace(remaining)
			// Session ID is the first token (UUID format)
			if len(sessionID) > 0 {
				// Take everything up to the first whitespace or end of string
				endIdx := 0
				for endIdx < len(sessionID) && !isWhitespace(rune(sessionID[endIdx])) {
					endIdx++
				}
				sessionID = sessionID[:endIdx]
				return true, sessionID
			}
		}
	}

	return false, ""
}

// Helper functions for string parsing

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func trimWhitespace(s string) string {
	start := 0
	for start < len(s) && isWhitespace(rune(s[start])) {
		start++
	}
	end := len(s)
	for end > start && isWhitespace(rune(s[end-1])) {
		end--
	}
	return s[start:end]
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}
