package transport

import (
	"context"
	"os/exec"

	"go.uber.org/zap"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func (t *SubprocessCLITransport) wantsThinkingDisplay() bool {
	return t.options != nil &&
		t.options.Thinking != nil &&
		t.options.Thinking.Type != "disabled" &&
		t.options.Thinking.Display != ""
}

func (t *SubprocessCLITransport) usesCustomSpawner() bool {
	return t.options != nil && t.options.SpawnProcess != nil
}

func (t *SubprocessCLITransport) detectThinkingDisplaySupport(ctx context.Context) bool {
	version, err := GetCLIVersion(ctx, t.cliPath)
	if err != nil {
		t.logger.Warn("unable to determine Claude CLI thinking display support; omitting thinking display flag",
			zap.Error(err),
		)
		return false
	}
	if SupportsThinkingDisplay(version) {
		return true
	}
	t.logger.Warn("Claude CLI version does not support --thinking-display; omitting thinking display flag",
		zap.String("version", version.String()),
		zap.String("minimum", MinimumThinkingDisplayCLIVersion),
	)
	return false
}

// detectExperimentalFlagSupport resolves whether the installed CLI accepts the
// experimental --agent-progress-summaries / --subagent-execution flags, so
// buildCommandArgs never emits one the CLI would reject (crashing Connect). It only
// probes the version when a consumer actually opted into one of the flags; otherwise
// the support fields stay false and the flags are never emitted anyway.
func (t *SubprocessCLITransport) detectExperimentalFlagSupport(ctx context.Context) {
	if t.options == nil {
		return
	}
	wantsProgress := t.options.AgentProgressSummaries
	wantsSubagent := t.options.SubagentExecution != nil
	if !wantsProgress && !wantsSubagent {
		return
	}
	// A flag was requested: default to NOT emitting it unless support is positively
	// confirmed, so a version-detection failure fails safe (no Connect crash).
	t.agentProgressSummariesSupported = false
	t.subagentExecutionSupported = false
	version, err := GetCLIVersion(ctx, t.cliPath)
	if err != nil {
		t.logger.Warn("unable to determine Claude CLI version; skipping experimental flags to avoid Connect failure",
			zap.Error(err))
		return
	}
	t.agentProgressSummariesSupported = SupportsAgentProgressSummaries(version)
	t.subagentExecutionSupported = SupportsSubagentExecution(version)
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
	if err := validateSpawnedPipes(process); err != nil {
		return err
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
	t.stderrDone = make(chan struct{})
	capturedProcess := t.customProcess
	capturedCtx := ctx
	capturedProcDone := t.procDone
	capturedStderrDone := t.stderrDone
	capturedStdout := t.stdout
	capturedStderr := t.stderr
	stderrTail := t.stderrTail
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.watchCustomProcessExit(capturedProcess, capturedProcDone, capturedStderrDone)
	}()

	// Launch context monitor goroutine for custom spawner (Bug C14).
	// Unlike exec.CommandContext, custom processes don't auto-kill on context cancel.
	// This goroutine kills the process when the transport context is canceled,
	// unblocking Wait() and the pipe readers.
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.monitorCustomProcessCtx(capturedCtx, capturedProcess, capturedProcDone)
	}()

	// Launch message reader loop in goroutine (tracked by wg for clean shutdown).
	// Capture stdout/stderr here (under caller's mutex) to avoid a race with
	// a subsequent Connect() call overwriting the struct fields.
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.messageReaderLoop(capturedCtx, capturedStdout, capturedProcDone)
	}()

	// Launch stderr reader for debugging (tracked by wg for clean shutdown)
	t.wg.Add(1)
	go t.readStderr(capturedCtx, capturedStderr, stderrTail, capturedStderrDone)

	// Mark as ready
	t.ready = true
	t.logger.Debug("Transport ready for communication")

	return nil
}

// validateSpawnedPipes ensures a custom-spawned process exposes the three pipes
// the transport requires.
func validateSpawnedPipes(process types.SpawnedProcess) error {
	if process.Stdin() == nil {
		return types.NewCLIConnectionError("custom spawner returned nil stdin")
	}
	if process.Stdout() == nil {
		return types.NewCLIConnectionError("custom spawner returned nil stdout")
	}
	if process.Stderr() == nil {
		return types.NewCLIConnectionError("custom spawner returned nil stderr")
	}
	return nil
}

// watchCustomProcessExit waits for a custom-spawned process to exit, then signals
// procDone, drains stderr, records the exit cause and telemetry, and cancels the
// transport context if the process had become ready. process.Wait() is called
// exactly once here — this goroutine is the sole owner of procDone's close.
func (t *SubprocessCLITransport) watchCustomProcessExit(process types.SpawnedProcess, procDone, stderrDone chan struct{}) {
	waitErr := process.Wait()
	close(procDone) // signal Close() that Wait() is done — BEFORE acquiring mutex
	t.drainStderr(stderrDone)
	t.mu.Lock()
	wasReady := t.ready
	t.ready = false
	requested := t.shutdownRequested
	exitCode := process.ExitCode()
	if waitErr != nil && t.err == nil && !requested {
		t.err = t.newProcessErrorWithDiagnostics("subprocess exited unexpectedly", exitCode)
	}
	var cause error
	if !requested {
		cause = t.err
	}
	cancelFn := t.cancel // capture under lock to avoid race with Close()
	t.mu.Unlock()
	t.observer().OnSubprocessExit(exitCode, requested, cause)
	if wasReady && cancelFn != nil {
		cancelFn()
	}
}

// monitorCustomProcessCtx kills a custom-spawned process when ctx is canceled,
// unblocking Wait() and the pipe readers (custom processes, unlike
// exec.CommandContext, are not auto-killed on context cancel). It returns once
// the process has exited (procDone closed).
func (t *SubprocessCLITransport) monitorCustomProcessCtx(ctx context.Context, process types.SpawnedProcess, procDone chan struct{}) {
	select {
	case <-ctx.Done():
		select {
		case <-procDone:
			return
		default:
		}
		if err := process.Kill(); err != nil {
			t.logger.Debug("context cancel: process kill returned error (process may have already exited)",
				zap.Error(err))
		}
	case <-procDone:
		// Process already exited — nothing to do
	}
}

// connectWithExecCommand uses the default exec.Command to create the process.
// runCtx is the cancellable connection context (t.ctx, a WithCancel child of the
// caller's ctx established in Connect); the subprocess and reader goroutines are
// bound to it so Close can cancel them via t.cancel after this returns.
func (t *SubprocessCLITransport) connectWithExecCommand(runCtx context.Context, args []string, envMap map[string]string) error {
	// Create command with arguments
	t.cmd = exec.CommandContext(runCtx, t.cliPath, args...)

	// Set working directory if provided
	if t.cwd != "" {
		t.cmd.Dir = t.cwd
	}

	// Set up environment variables with SDK/provider values overriding inherited
	// entries without duplicate keys.
	t.cmd.Env = buildEffectiveProcessEnvironment(envMap)

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
	t.stderrDone = make(chan struct{})
	capturedCmd := t.cmd
	capturedCtx := runCtx
	capturedProcDone := t.procDone
	capturedStderrDone := t.stderrDone
	capturedStdout := t.stdout
	capturedStderr := t.stderr
	stderrTail := t.stderrTail
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		waitErr := capturedCmd.Wait()
		close(capturedProcDone) // signal Close() that Wait() is done — BEFORE acquiring mutex
		t.drainStderr(capturedStderrDone)
		t.mu.Lock()
		wasReady := t.ready
		t.ready = false
		requested := t.shutdownRequested
		exitCode := getExitCode(waitErr)
		if waitErr != nil && t.err == nil && !requested {
			t.err = t.newProcessErrorWithDiagnostics("subprocess exited unexpectedly", exitCode)
		}
		var cause error
		if !requested {
			cause = t.err
		}
		cancelFn := t.cancel // capture under lock to avoid race with Close()
		t.mu.Unlock()
		t.observer().OnSubprocessExit(exitCode, requested, cause)
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
		t.messageReaderLoop(capturedCtx, capturedStdout, capturedProcDone)
	}()

	// Launch stderr reader for debugging (tracked by wg for clean shutdown)
	t.wg.Add(1)
	go t.readStderr(capturedCtx, capturedStderr, stderrTail, capturedStderrDone)

	// Mark as ready
	t.ready = true
	t.logger.Debug("Transport ready for communication")

	return nil
}

// buildEnvMap constructs the environment variable map for the subprocess.
func (t *SubprocessCLITransport) buildEnvMap() map[string]string {
	envMap := BuildRuntimeEnvironment(t.options, t.env)
	if t.options != nil && t.options.Model != nil {
		t.logger.Debug("setting ANTHROPIC_MODEL environment variable", zap.String("model", *t.options.Model))
	}
	if t.options != nil && t.options.BaseURL != nil {
		t.logger.Debug("setting ANTHROPIC_BASE_URL environment variable", zap.String("base_url", *t.options.BaseURL))
	}
	for key := range t.env {
		t.logger.Debug("setting custom environment variable", zap.String("key", key))
	}
	return envMap
}

// BuildRuntimeEnvironment constructs the environment sent to every Claude CLI
// process, including capability probes.
func BuildRuntimeEnvironment(
	options *types.ClaudeAgentOptions,
	customEnv map[string]string,
) map[string]string {
	envMap := map[string]string{
		"CLAUDE_CODE_ENTRYPOINT":   "agent",
		"CLAUDE_AGENT_SDK_VERSION": SDKVersion,
	}
	if options != nil && options.Model != nil {
		envMap["ANTHROPIC_MODEL"] = *options.Model
	}
	if options != nil && options.BaseURL != nil {
		envMap["ANTHROPIC_BASE_URL"] = *options.BaseURL
	}
	for key, value := range customEnv {
		envMap[key] = value
	}
	return envMap
}
