package transport

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// DiscoverPermissionModesWithSpawner probes the CLI through the caller's
// process boundary. Results are intentionally not cached globally because a
// custom spawner may route identical command paths to different remote hosts.
func DiscoverPermissionModesWithSpawner(
	ctx context.Context,
	cliPath string,
	cwd string,
	env map[string]string,
	spawner types.ProcessSpawner,
) (modes []types.SupportedPermissionMode, version string, err error) {
	version, versionErr := DiscoverCLIVersionWithSpawner(ctx, cliPath, cwd, env, spawner)
	stdout, stderr, err := runSpawnerCLICommand(ctx, cliPath, cwd, env, spawner, "--help")
	if err != nil {
		return nil, version, err
	}
	modes = permissionModesFromHelp(stdout+"\n"+stderr, version)
	if versionErr != nil {
		return modes, "", versionErr
	}
	return modes, version, nil
}

// DiscoverCLIVersionWithSpawner probes the CLI version through the caller's
// process boundary.
func DiscoverCLIVersionWithSpawner(
	ctx context.Context,
	cliPath string,
	cwd string,
	env map[string]string,
	spawner types.ProcessSpawner,
) (string, error) {
	stdout, _, err := runSpawnerCLICommand(ctx, cliPath, cwd, env, spawner, "--version")
	if err != nil {
		return "", err
	}
	version, err := ParseSemanticVersion(strings.TrimSpace(stdout))
	if err != nil {
		return "", fmt.Errorf("parse Claude CLI version probe: %w", err)
	}
	return version.String(), nil
}

type cliProbeReadResult struct {
	data string
	err  error
}

func runSpawnerCLICommand(
	ctx context.Context,
	cliPath string,
	cwd string,
	env map[string]string,
	spawner types.ProcessSpawner,
	arg string,
) (stdoutText, stderrText string, runErr error) {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := probeCtx.Err(); err != nil {
		return "", "", err
	}

	process, err := spawner(probeCtx, types.SpawnOptions{
		Command: cliPath,
		Args:    []string{arg},
		CWD:     cwd,
		Env:     cloneProbeEnv(env),
	})
	if err != nil {
		return "", "", fmt.Errorf("spawn Claude CLI probe %s: %w", arg, err)
	}
	if process == nil {
		return "", "", fmt.Errorf("spawn Claude CLI probe %s: process is nil", arg)
	}
	stdin := process.Stdin()
	stdout := process.Stdout()
	stderr := process.Stderr()
	if stdin == nil || stdout == nil || stderr == nil {
		_ = process.Kill()
		return "", "", fmt.Errorf("spawn Claude CLI probe %s: process returned nil pipe", arg)
	}

	_ = stdin.Close()
	defer func() { _ = stdout.Close() }()
	defer func() { _ = stderr.Close() }()

	stdoutCh := readCLIProbePipe(stdout)
	stderrCh := readCLIProbePipe(stderr)
	waitCh := make(chan error, 1)
	go func() { waitCh <- process.Wait() }()

	select {
	case waitErr := <-waitCh:
		if err := probeCtx.Err(); err != nil {
			return "", "", err
		}
		stdoutResult, err := awaitCLIProbeRead(probeCtx, stdoutCh)
		if err != nil {
			_ = process.Kill()
			return "", "", err
		}
		stderrResult, err := awaitCLIProbeRead(probeCtx, stderrCh)
		if err != nil {
			_ = process.Kill()
			return "", "", err
		}
		if waitErr != nil {
			return stdoutResult.data, stderrResult.data, fmt.Errorf("claude CLI probe %s failed: %w", arg, waitErr)
		}
		if stdoutResult.err != nil {
			return "", stderrResult.data, fmt.Errorf("read Claude CLI probe %s stdout: %w", arg, stdoutResult.err)
		}
		if stderrResult.err != nil {
			return stdoutResult.data, "", fmt.Errorf("read Claude CLI probe %s stderr: %w", arg, stderrResult.err)
		}
		return stdoutResult.data, stderrResult.data, nil
	case <-probeCtx.Done():
		_ = process.Kill()
		_ = stdout.Close()
		_ = stderr.Close()
		return "", "", probeCtx.Err()
	}
}

func awaitCLIProbeRead(
	ctx context.Context,
	result <-chan cliProbeReadResult,
) (cliProbeReadResult, error) {
	select {
	case readResult := <-result:
		return readResult, nil
	case <-ctx.Done():
		return cliProbeReadResult{}, ctx.Err()
	}
}

func readCLIProbePipe(reader io.Reader) <-chan cliProbeReadResult {
	result := make(chan cliProbeReadResult, 1)
	go func() {
		data, err := io.ReadAll(reader)
		result <- cliProbeReadResult{data: string(data), err: err}
	}()
	return result
}

func cloneProbeEnv(env map[string]string) map[string]string {
	if len(env) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(env))
	for key, value := range env {
		cloned[key] = value
	}
	return cloned
}
