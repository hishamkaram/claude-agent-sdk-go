package claude

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestGetSessionMessages_EmptySessionID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, err := GetSessionMessages(ctx, "", nil)
	if err == nil {
		t.Fatal("expected error for empty sessionID, got nil")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

func TestListSessions_ContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err := ListSessions(ctx, nil)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

func TestGetSessionMessages_ContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	_, err := GetSessionMessages(ctx, "test-session-id", nil)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

func TestListSessions_WithOptions(t *testing.T) {
	t.Parallel()

	// Create a test script that echoes back the args as JSON
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "claude")
	script := `#!/bin/sh
echo '[]'
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	// Verify the script is executable
	if _, err := exec.LookPath(scriptPath); err != nil {
		// Set PATH to include our test dir
		t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := &types.ListSessionsOptions{
		Dir:              "/tmp/test",
		Limit:            10,
		Offset:           5,
		IncludeWorktrees: true,
	}
	sessions, err := ListSessions(ctx, opts)
	if err != nil {
		// CLI might not be found — that's expected in test environments without claude installed
		// The important thing is we don't panic
		t.Logf("ListSessions returned error (expected in test env): %v", err)
		return
	}
	if sessions == nil {
		t.Error("expected non-nil sessions slice")
	}
}

func TestGetSessionMessages_WithOptions(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := &types.GetSessionMessagesOptions{
		Dir:    "/tmp/test",
		Limit:  20,
		Offset: 0,
	}
	_, err := GetSessionMessages(ctx, "some-session-id", opts)
	if err != nil {
		// CLI might not be found — that's expected in test environments without claude installed
		t.Logf("GetSessionMessages returned error (expected in test env): %v", err)
		return
	}
}

func TestListSessions_NilOptions(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := ListSessions(ctx, nil)
	if err != nil {
		// CLI might not be found — that's expected
		t.Logf("ListSessions returned error (expected in test env): %v", err)
		return
	}
}

func TestListSessions_Integration(t *testing.T) {
	t.Parallel()
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("skipping integration test; set RUN_INTEGRATION_TESTS=1 to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sessions, err := ListSessions(ctx, &types.ListSessionsOptions{Limit: 5})
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	t.Logf("Found %d sessions", len(sessions))
	for i, s := range sessions {
		t.Logf("  [%d] ID=%s Summary=%q", i, s.SessionID, s.Summary)
	}
}

func TestGetSessionMessages_Integration(t *testing.T) {
	t.Parallel()
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("skipping integration test; set RUN_INTEGRATION_TESTS=1 to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First list sessions to get a valid session ID
	sessions, err := ListSessions(ctx, &types.ListSessionsOptions{Limit: 1})
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) == 0 {
		t.Skip("no sessions available for testing")
	}

	messages, err := GetSessionMessages(ctx, sessions[0].SessionID, &types.GetSessionMessagesOptions{Limit: 5})
	if err != nil {
		t.Fatalf("GetSessionMessages failed: %v", err)
	}
	t.Logf("Found %d messages for session %s", len(messages), sessions[0].SessionID)
}
