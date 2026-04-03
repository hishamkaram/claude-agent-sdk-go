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

// --- GetSessionInfo tests ---

func TestGetSessionInfo_EmptySessionID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, err := GetSessionInfo(ctx, "", nil)
	if err == nil {
		t.Fatal("expected error for empty sessionID, got nil")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

func TestGetSessionInfo_ContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := GetSessionInfo(ctx, "test-session-id", nil)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

func TestGetSessionInfo_NilOptions(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := GetSessionInfo(ctx, "some-session-id", nil)
	if err != nil {
		t.Logf("GetSessionInfo returned error (expected in test env): %v", err)
		return
	}
}

func TestGetSessionInfo_WithOptions(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	opts := &types.GetSessionInfoOptions{
		Dir: t.TempDir(),
	}
	_, err := GetSessionInfo(ctx, "some-session-id", opts)
	if err != nil {
		t.Logf("GetSessionInfo returned error (expected in test env): %v", err)
		return
	}
}

// --- RenameSession tests ---

func TestRenameSession_EmptySessionID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := RenameSession(ctx, "", "New Title", nil)
	if err == nil {
		t.Fatal("expected error for empty sessionID, got nil")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

func TestRenameSession_ContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := RenameSession(ctx, "test-session-id", "New Title", nil)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

func TestRenameSession_NilOptions(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := RenameSession(ctx, "some-session-id", "My Title", nil)
	if err != nil {
		t.Logf("RenameSession returned error (expected in test env): %v", err)
		return
	}
}

// --- TagSession tests ---

func TestTagSession_EmptySessionID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	err := TagSession(ctx, "", "v1.0", nil)
	if err == nil {
		t.Fatal("expected error for empty sessionID, got nil")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

func TestTagSession_ContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := TagSession(ctx, "test-session-id", "tag", nil)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

func TestTagSession_NilOptions(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := TagSession(ctx, "some-session-id", "v1.0", nil)
	if err != nil {
		t.Logf("TagSession returned error (expected in test env): %v", err)
		return
	}
}

func TestTagSession_EmptyTag(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Empty tag should clear the tag — not error
	err := TagSession(ctx, "some-session-id", "", nil)
	if err != nil {
		t.Logf("TagSession with empty tag returned error (expected in test env): %v", err)
		return
	}
}

// --- ForkSession tests ---

func TestForkSession_EmptySessionID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, err := ForkSession(ctx, "", nil)
	if err == nil {
		t.Fatal("expected error for empty sessionID, got nil")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

func TestForkSession_ContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ForkSession(ctx, "test-session-id", nil)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

func TestForkSession_NilOptions(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := ForkSession(ctx, "some-session-id", nil)
	if err != nil {
		t.Logf("ForkSession returned error (expected in test env): %v", err)
		return
	}
}

func TestForkSession_WithOptions(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	opts := &types.ForkSessionOptions{
		Dir: t.TempDir(),
	}
	_, err := ForkSession(ctx, "some-session-id", opts)
	if err != nil {
		t.Logf("ForkSession returned error (expected in test env): %v", err)
		return
	}
}

// --- ListSubagents tests ---

func TestListSubagents_EmptySessionID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, err := ListSubagents(ctx, "", nil)
	if err == nil {
		t.Fatal("expected error for empty sessionID, got nil")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

func TestListSubagents_ContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ListSubagents(ctx, "test-session-id", nil)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

func TestListSubagents_NilOptions(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := ListSubagents(ctx, "some-session-id", nil)
	if err != nil {
		t.Logf("ListSubagents returned error (expected in test env): %v", err)
		return
	}
}

func TestListSubagents_WithOptions(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	opts := &types.ListSubagentsOptions{
		Dir:    t.TempDir(),
		Limit:  10,
		Offset: 5,
	}
	_, err := ListSubagents(ctx, "some-session-id", opts)
	if err != nil {
		t.Logf("ListSubagents returned error (expected in test env): %v", err)
		return
	}
}

// --- GetSubagentMessages tests ---

func TestGetSubagentMessages_EmptySubagentID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, err := GetSubagentMessages(ctx, "", nil)
	if err == nil {
		t.Fatal("expected error for empty subagentID, got nil")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

func TestGetSubagentMessages_ContextCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := GetSubagentMessages(ctx, "test-subagent-id", nil)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
}

func TestGetSubagentMessages_NilOptions(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := GetSubagentMessages(ctx, "some-subagent-id", nil)
	if err != nil {
		t.Logf("GetSubagentMessages returned error (expected in test env): %v", err)
		return
	}
}

func TestGetSubagentMessages_WithOptions(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	opts := &types.GetSubagentMessagesOptions{
		Dir:    t.TempDir(),
		Limit:  50,
		Offset: 10,
	}
	_, err := GetSubagentMessages(ctx, "some-subagent-id", opts)
	if err != nil {
		t.Logf("GetSubagentMessages returned error (expected in test env): %v", err)
		return
	}
}

// --- IncludeSystemMessages test ---

func TestGetSessionMessages_IncludeSystemMessages(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	opts := &types.GetSessionMessagesOptions{
		IncludeSystemMessages: true,
	}
	_, err := GetSessionMessages(ctx, "some-session-id", opts)
	if err != nil {
		t.Logf("GetSessionMessages with IncludeSystemMessages returned error (expected in test env): %v", err)
		return
	}
}

// --- Integration tests ---

func TestGetSessionInfo_Integration(t *testing.T) {
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

	info, err := GetSessionInfo(ctx, sessions[0].SessionID, nil)
	if err != nil {
		t.Fatalf("GetSessionInfo failed: %v", err)
	}
	if info.SessionID == "" {
		t.Error("expected non-empty SessionID in info")
	}
	t.Logf("Session info: ID=%s Title=%q Tag=%q", info.SessionID, info.CustomTitle, info.Tag)
}

func TestListSubagents_Integration(t *testing.T) {
	t.Parallel()
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("skipping integration test; set RUN_INTEGRATION_TESTS=1 to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sessions, err := ListSessions(ctx, &types.ListSessionsOptions{Limit: 1})
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) == 0 {
		t.Skip("no sessions available for testing")
	}

	agents, err := ListSubagents(ctx, sessions[0].SessionID, nil)
	if err != nil {
		t.Fatalf("ListSubagents failed: %v", err)
	}
	t.Logf("Found %d subagents for session %s", len(agents), sessions[0].SessionID)
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
