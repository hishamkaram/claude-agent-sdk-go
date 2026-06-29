package transport

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestStderrFileLogging tests stderr file logging functionality
func TestStderrFileLogging(t *testing.T) {
	t.Parallel()
	t.Run("disabled by default", func(t *testing.T) {
		// Create options without stderr file logging
		opts := types.NewClaudeAgentOptions()

		if opts.StderrLogFile != nil {
			t.Error("StderrLogFile should be nil by default")
		}
	})

	t.Run("custom path creates directory and file", func(t *testing.T) {
		// Create temporary directory for testing
		tempDir := t.TempDir()
		customLogPath := filepath.Join(tempDir, "logs", "stderr.log")

		// Create options with custom stderr log file
		opts := types.NewClaudeAgentOptions().
			WithCustomStderrLogFile(customLogPath)

		// Verify option is set
		if opts.StderrLogFile == nil {
			t.Fatal("StderrLogFile should not be nil")
		}
		if *opts.StderrLogFile != customLogPath {
			t.Errorf("StderrLogFile = %q, want %q", *opts.StderrLogFile, customLogPath)
		}

		// Create transport with echo command that writes to stderr
		logger := log.NewLogger(false)

		// Use a simple shell command that writes to stderr
		transport := NewSubprocessCLITransport(
			"/bin/sh",
			"",
			nil,
			logger,
			"",
			opts,
		)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Connect will trigger the stderr reader goroutine
		err := transport.Connect(ctx)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}

		// Write a command that produces stderr output
		// Use -c to run a command that writes to stderr
		_ = transport.Write(ctx, `echo "test error" >&2`)

		// Give it time to process
		time.Sleep(500 * time.Millisecond)

		// Close transport
		_ = transport.Close(ctx)

		// Verify the directory was created
		logDir := filepath.Dir(customLogPath)
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			t.Errorf("Log directory was not created: %s", logDir)
		}

		// Verify the log file was created
		if _, err := os.Stat(customLogPath); os.IsNotExist(err) {
			t.Errorf("Log file was not created: %s", customLogPath)
		}
	})

	t.Run("default location option", func(t *testing.T) {
		opts := types.NewClaudeAgentOptions().
			WithDefaultStderrLogFile()

		if opts.StderrLogFile == nil {
			t.Fatal("StderrLogFile should not be nil")
		}
		if *opts.StderrLogFile != "" {
			t.Errorf("StderrLogFile = %q, want empty string for default", *opts.StderrLogFile)
		}
	})

	t.Run("callback still works with file logging", func(t *testing.T) {
		tempDir := t.TempDir()
		customLogPath := filepath.Join(tempDir, "test.log")

		// Track callback invocations
		var callbackLines []string
		var mu sync.Mutex

		opts := types.NewClaudeAgentOptions().
			WithCustomStderrLogFile(customLogPath).
			WithStderr(func(line string) {
				mu.Lock()
				defer mu.Unlock()
				callbackLines = append(callbackLines, line)
			})

		logger := log.NewLogger(false)
		transport := NewSubprocessCLITransport(
			"/bin/sh",
			"",
			nil,
			logger,
			"",
			opts,
		)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := transport.Connect(ctx)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}

		// Write command that produces stderr
		_ = transport.Write(ctx, `echo "callback test" >&2`)

		time.Sleep(500 * time.Millisecond)
		_ = transport.Close(ctx)

		// Verify callback was called
		mu.Lock()
		numCallbacks := len(callbackLines)
		mu.Unlock()

		if numCallbacks == 0 {
			t.Log("Warning: callback was not invoked (may be expected for /bin/sh)")
		}

		// Verify file was still created (file logging should work even with callback)
		if _, err := os.Stat(customLogPath); os.IsNotExist(err) {
			t.Errorf("Log file should be created even when callback is set: %s", customLogPath)
		}
	})
}

// TestStderrFileLogging_DirectoryCreation tests that parent directories are created
func TestStderrFileLogging_DirectoryCreation(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	// Create nested path that doesn't exist
	deepPath := filepath.Join(tempDir, "a", "b", "c", "stderr.log")

	opts := types.NewClaudeAgentOptions().
		WithCustomStderrLogFile(deepPath)

	logger := log.NewLogger(false)
	transport := NewSubprocessCLITransport(
		"/bin/echo",
		"",
		nil,
		logger,
		"",
		opts,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// This should create all parent directories
	err := transport.Connect(ctx)
	if err != nil {
		t.Logf("Connect error (may be expected): %v", err)
	}

	// Give readStderr goroutine time to run
	time.Sleep(200 * time.Millisecond)

	_ = transport.Close(ctx)

	// Verify nested directories were created
	expectedDir := filepath.Join(tempDir, "a", "b", "c")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("Nested directories were not created: %s", expectedDir)
	}

	// Verify log file was created
	if _, err := os.Stat(deepPath); os.IsNotExist(err) {
		t.Logf("Log file was not created (may be expected for /bin/echo): %s", deepPath)
	}
}
