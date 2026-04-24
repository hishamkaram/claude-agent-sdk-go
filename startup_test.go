package claude

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/transport"
)

func blockingCLIPath(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "fake-claude")
	script := "#!/bin/sh\nwhile :; do sleep 3600; done\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to create fake CLI: %v", err)
	}
	return path
}

// TestStartup runs all startup tests sequentially since they share the global warmPool.
// The subtests are NOT parallel — warm pool is a singleton resource.
func TestStartup(t *testing.T) {
	// Reset pool before all subtests
	transport.ConsumeWarmProcess()

	t.Run("InvalidCLIPath", func(t *testing.T) {
		transport.ConsumeWarmProcess()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := Startup(ctx, WithStartupCLIPath("/nonexistent/cli/path"))
		if err == nil {
			t.Fatal("expected error for invalid CLI path, got nil")
		}
	})

	t.Run("ConsumeWarmEmpty", func(t *testing.T) {
		transport.ConsumeWarmProcess()

		wp := transport.ConsumeWarmProcess()
		if wp != nil {
			t.Fatal("expected nil from empty warm pool")
		}
	})

	t.Run("SpawnAndConsume", func(t *testing.T) {
		transport.ConsumeWarmProcess()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := Startup(ctx, WithStartupCLIPath(blockingCLIPath(t)))
		if err != nil {
			t.Fatalf("Startup failed: %v", err)
		}

		wp := transport.ConsumeWarmProcess()
		if wp == nil {
			t.Fatal("expected non-nil warm process after Startup")
		}

		// Second consume should return nil (exactly-once)
		wp2 := transport.ConsumeWarmProcess()
		if wp2 != nil {
			t.Fatal("expected nil on second consume (exactly-once)")
		}

		// Clean up the process
		wp.Kill()
	})

	t.Run("ContextCancel", func(t *testing.T) {
		transport.ConsumeWarmProcess()

		ctx, cancel := context.WithCancel(context.Background())

		err := Startup(ctx, WithStartupCLIPath(blockingCLIPath(t)))
		if err != nil {
			t.Fatalf("Startup failed: %v", err)
		}

		// Cancel context — exec.CommandContext kills the process
		cancel()

		// Give cleanup goroutine time to run
		time.Sleep(200 * time.Millisecond)

		// Pool should be empty now (background goroutine clears dead process)
		wp := transport.ConsumeWarmProcess()
		if wp != nil {
			if wp.IsAlive() {
				t.Error("expected process to be dead after context cancel")
			}
			<-wp.Done
		}
	})

	t.Run("DoubleCallNoop", func(t *testing.T) {
		transport.ConsumeWarmProcess()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cliPath := blockingCLIPath(t)
		err := Startup(ctx, WithStartupCLIPath(cliPath))
		if err != nil {
			t.Fatalf("first Startup failed: %v", err)
		}

		// Second call should be no-op (pool already has a process)
		err = Startup(ctx, WithStartupCLIPath(cliPath))
		if err != nil {
			t.Fatalf("second Startup should be no-op, got error: %v", err)
		}

		// Only one process in pool
		wp := transport.ConsumeWarmProcess()
		if wp == nil {
			t.Fatal("expected non-nil warm process")
		}
		wp.Kill()

		// Pool empty after single consume
		wp2 := transport.ConsumeWarmProcess()
		if wp2 != nil {
			t.Error("expected nil — only one process should have been in pool")
			wp2.Kill()
		}
	})

	t.Run("WithCWD", func(t *testing.T) {
		transport.ConsumeWarmProcess()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := Startup(ctx, WithStartupCLIPath(blockingCLIPath(t)), WithStartupCWD("/tmp"))
		if err != nil {
			t.Fatalf("Startup with CWD failed: %v", err)
		}

		wp := transport.ConsumeWarmProcess()
		if wp == nil {
			t.Fatal("expected non-nil warm process")
		}
		wp.Kill()
	})

	t.Run("WithEnv", func(t *testing.T) {
		transport.ConsumeWarmProcess()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := Startup(ctx,
			WithStartupCLIPath(blockingCLIPath(t)),
			WithStartupEnv(map[string]string{"FOO": "bar"}),
		)
		if err != nil {
			t.Fatalf("Startup with Env failed: %v", err)
		}

		wp := transport.ConsumeWarmProcess()
		if wp == nil {
			t.Fatal("expected non-nil warm process")
		}
		wp.Kill()
	})

	t.Run("AllOptions", func(t *testing.T) {
		transport.ConsumeWarmProcess()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cliPath := blockingCLIPath(t)
		err := Startup(ctx,
			WithStartupCLIPath(cliPath),
			WithStartupCWD("/tmp"),
			WithStartupEnv(map[string]string{"FOO": "bar"}),
		)
		if err != nil {
			t.Fatalf("Startup with all options failed: %v", err)
		}

		wp := transport.ConsumeWarmProcess()
		if wp == nil {
			t.Fatal("expected non-nil warm process")
		}
		wp.Kill()
	})

	// Final cleanup
	transport.ConsumeWarmProcess()
}
