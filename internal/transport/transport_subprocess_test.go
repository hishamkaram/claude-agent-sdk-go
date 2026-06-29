package transport

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestSubprocessCLITransportConnect tests subprocess connection
func TestSubprocessCLITransportConnect(t *testing.T) {
	t.Parallel()
	echoPath, err := FindMockCLI(t)
	if err != nil {
		t.Skip("No mock CLI available for testing")
	}

	logger := log.NewLogger(false) // Non-verbose for tests
	transport := NewSubprocessCLITransport(echoPath, "", nil, logger, "", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect should succeed
	if err := transport.Connect(ctx); err != nil {
		t.Errorf("Connect() unexpected error: %v", err)
	}

	// Should be ready
	if !transport.IsReady() {
		t.Errorf("IsReady() = false, want true after Connect()")
	}

	// Clean up
	if err := transport.Close(ctx); err != nil {
		t.Logf("Close() error (may be expected): %v", err)
	}
}

// TestSubprocessCLITransportWrite tests writing to subprocess
func TestSubprocessCLITransportWrite(t *testing.T) {
	t.Parallel()
	catPath, err := FindMockCLI(t)
	if err != nil {
		t.Skip("No mock CLI available for testing")
	}

	logger := log.NewLogger(false) // Non-verbose for tests
	transport := NewSubprocessCLITransport(catPath, "", nil, logger, "", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("Connect() unexpected error: %v", err)
	}
	defer func() {
		_ = transport.Close(ctx)
	}()

	// Write should succeed
	testJSON := `{"type":"test","data":"hello"}`
	if err := transport.Write(ctx, testJSON); err != nil {
		t.Errorf("Write() unexpected error: %v", err)
	}
}

// TestSubprocessCLITransportClose tests subprocess cleanup
func TestSubprocessCLITransportClose(t *testing.T) {
	t.Parallel()
	echoPath, err := FindMockCLI(t)
	if err != nil {
		t.Skip("No mock CLI available for testing")
	}

	logger := log.NewLogger(false) // Non-verbose for tests
	transport := NewSubprocessCLITransport(echoPath, "", nil, logger, "", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect and then close
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("Connect() unexpected error: %v", err)
	}

	if err := transport.Close(ctx); err != nil {
		t.Logf("Close() error (may be expected for echo): %v", err)
	}

	// Should not be ready after close
	if transport.IsReady() {
		t.Errorf("IsReady() = true, want false after Close()")
	}
}

// TestMessageReaderLoop tests message reading and parsing
func TestMessageReaderLoop(t *testing.T) {
	t.Parallel()
	// Create a mock JSON stream
	jsonStream := `{"type":"user","content":"hello"}` + "\n" +
		`{"type":"assistant","content":[{"type":"text","text":"hi"}],"model":"claude-3"}` + "\n" +
		`{"type":"system","subtype":"info","data":{}}` + "\n"

	// Create a pipe to simulate subprocess output
	pr, pw := io.Pipe()

	// Write mock data in a goroutine
	go func() {
		defer func() {
			_ = pw.Close()
		}()
		_, _ = pw.Write([]byte(jsonStream))
	}()

	// Create transport with custom stdout
	logger := log.NewLogger(false) // Non-verbose for tests
	transport := &SubprocessCLITransport{
		messages: make(chan types.Message, 10),
		ready:    true,
		logger:   logger,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	transport.ctx = ctx
	transport.stdout = pr

	// Start reader loop
	go transport.messageReaderLoop(ctx, pr, nil)

	// Read messages from channel
	var messages []types.Message
	for msg := range transport.messages {
		messages = append(messages, msg)
	}

	// Should have parsed 3 messages
	if len(messages) != 3 {
		t.Errorf("messageReaderLoop() parsed %d messages, want 3", len(messages))
	}

	// Verify message types
	expectedTypes := []string{"user", "assistant", "system"}
	for i, msg := range messages {
		if i >= len(expectedTypes) {
			break
		}
		if msg.GetMessageType() != expectedTypes[i] {
			t.Errorf("message[%d].Type = %q, want %q", i, msg.GetMessageType(), expectedTypes[i])
		}
	}
}

// TestSubprocessEnvironment tests environment variable setup
func TestSubprocessEnvironment(t *testing.T) {
	t.Parallel()
	echoPath, err := FindMockCLI(t)
	if err != nil {
		t.Skip("No mock CLI available for testing")
	}

	env := map[string]string{
		"TEST_VAR":    "test_value",
		"ANOTHER_VAR": "another_value",
	}

	logger := log.NewLogger(false) // Non-verbose for tests
	transport := NewSubprocessCLITransport(echoPath, "", env, logger, "", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("Connect() unexpected error: %v", err)
	}
	defer func() {
		_ = transport.Close(ctx)
	}()

	// Check that environment variables were set (we can't directly verify,
	// but we can check that Connect succeeded with the env)
	if !transport.IsReady() {
		t.Errorf("IsReady() = false after Connect() with custom env")
	}
}

// FindMockCLI creates a mock CLI script for testing. The transport always passes
// flags like --input-format=stream-json to the subprocess, which real commands
// (e.g. cat, echo) don't understand and exit on. This wrapper ignores all args
// and reads from stdin via exec cat, matching the expected transport behavior.
func FindMockCLI(t *testing.T) (string, error) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		return "", types.NewCLINotFoundError("sh not found")
	}
	scriptPath := filepath.Join(t.TempDir(), "mock-claude")
	f, err := os.OpenFile(scriptPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", types.NewCLINotFoundError("failed to create mock CLI: " + err.Error())
	}
	if _, err := f.WriteString("#!/bin/sh\nexec cat\n"); err != nil {
		_ = f.Close()
		return "", types.NewCLINotFoundError("failed to write mock CLI: " + err.Error())
	}
	if err := f.Close(); err != nil {
		return "", types.NewCLINotFoundError("failed to close mock CLI: " + err.Error())
	}
	if err := os.Chmod(scriptPath, 0o755); err != nil {
		return "", types.NewCLINotFoundError("failed to chmod mock CLI: " + err.Error())
	}
	return scriptPath, nil
}

// TestIntegrationSubprocessCLI tests end-to-end subprocess communication
// This test requires the actual Claude CLI to be installed
func TestIntegrationSubprocessCLI(t *testing.T) {
	resetFindCLICacheForTest(t)
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Try to find Claude CLI
	cliPath, err := FindCLI(context.Background())
	if err != nil {
		t.Skipf("Claude CLI not found, skipping integration test: %v", err)
	}

	logger := log.NewLogger(false) // Non-verbose for tests
	transport := NewSubprocessCLITransport(cliPath, "", nil, logger, "", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to CLI
	if connectErr := transport.Connect(ctx); connectErr != nil {
		t.Fatalf("Connect() failed: %v", connectErr)
	}
	defer func() {
		_ = transport.Close(ctx)
	}()

	// Should be ready
	if !transport.IsReady() {
		t.Errorf("IsReady() = false after successful Connect()")
	}

	// Try to write a simple query
	query := map[string]interface{}{
		"type":    "control",
		"subtype": "query",
		"prompt":  "Hello, Claude!",
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		t.Fatalf("Failed to marshal query: %v", err)
	}

	if err := transport.Write(ctx, string(queryJSON)); err != nil {
		t.Errorf("Write() failed: %v", err)
	}

	// Read messages (with timeout)
	messages := transport.ReadMessages(ctx)

	select {
	case msg := <-messages:
		if msg == nil {
			t.Errorf("Received nil message")
		} else {
			t.Logf("Received message type: %s", msg.GetMessageType())
		}
	case <-time.After(5 * time.Second):
		t.Logf("Timeout waiting for response (may be expected for this test)")
	}
}

// TestExtractSessionNotFoundError tests parsing of session not found errors from stderr
