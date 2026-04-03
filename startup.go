package claude

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/transport"
)

// startupConfig holds configuration for the Startup pre-warm function.
type startupConfig struct {
	cliPath string
	cwd     string
	env     map[string]string
}

// StartupOption is a functional option for configuring Startup.
type StartupOption func(*startupConfig)

// WithStartupCLIPath sets a custom CLI binary path for pre-warming.
// If not set, the CLI is discovered automatically.
func WithStartupCLIPath(path string) StartupOption {
	return func(c *startupConfig) {
		c.cliPath = path
	}
}

// WithStartupCWD sets the working directory for the pre-warmed subprocess.
func WithStartupCWD(cwd string) StartupOption {
	return func(c *startupConfig) {
		c.cwd = cwd
	}
}

// WithStartupEnv sets additional environment variables for the pre-warmed subprocess.
func WithStartupEnv(env map[string]string) StartupOption {
	return func(c *startupConfig) {
		c.env = env
	}
}

// Startup pre-warms the Claude Code CLI subprocess for faster first query.
// Call once at application startup. The warmed process is reused by the next
// Query() or NewClient().Connect() call.
//
// Safe to call multiple times — returns nil immediately if a warm process
// already exists in the pool. The warmed process is automatically cleaned
// up on context cancellation.
func Startup(ctx context.Context, opts ...StartupOption) error {
	// Check if pool already has a process
	existing := transport.ConsumeWarmProcess()
	if existing != nil {
		// Put it back — pool was already populated
		transport.StoreWarmProcess(existing)
		return nil
	}

	cfg := &startupConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	cliPath := cfg.cliPath
	if cliPath == "" {
		var err error
		cliPath, err = transport.FindCLI()
		if err != nil {
			return fmt.Errorf("Startup: %w", err)
		}
	}

	// Spawn subprocess in warm state
	cmd := exec.CommandContext(ctx, cliPath,
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
	)

	if cfg.cwd != "" {
		cmd.Dir = cfg.cwd
	}

	// Set environment
	cmd.Env = os.Environ()
	for key, value := range cfg.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("Startup: failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("Startup: failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdout.Close()
		_ = stdin.Close()
		return fmt.Errorf("Startup: failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stderr.Close()
		_ = stdout.Close()
		_ = stdin.Close()
		return fmt.Errorf("Startup: failed to start subprocess: %w", err)
	}

	done := make(chan struct{})

	wp := &transport.WarmProcess{
		Cmd:    cmd,
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Done:   done,
	}

	// Store in pool
	transport.StoreWarmProcess(wp)

	// Background goroutine: wait for process exit and close done channel.
	go func() {
		_ = cmd.Wait()
		close(done)
		// Clear pool if this process is still in it (it may have been consumed)
		current := transport.ConsumeWarmProcess()
		if current != nil && current != wp {
			// Someone else's process — put it back
			transport.StoreWarmProcess(current)
		}
		// If current == wp, we just removed our dead process. Good.
		// If current == nil, it was already consumed. Also good.
	}()

	return nil
}
