//go:build integration
// +build integration

// Shared test helpers for the real-CLI integration suite. See integration_test.go
// for the documented gating and run-command conventions.

package tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	claude "github.com/hishamkaram/claude-agent-sdk-go"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// setupClient builds options (defaulting to a bypass-permissions real-CLI
// client), connects, and registers Close() on t.Cleanup. Gates on requireAuth
// and requireClaude — the test skips with a useful message if either is
// missing. The returned Client and ctx are bound to a 60s timeout.
//
// Pass a configure func to customize options before Connect.
//
//	client, ctx := setupClient(t, func(opts *types.ClaudeAgentOptions) {
//	    opts.WithModel("claude-sonnet-4-5")
//	})
func setupClient(t *testing.T, configure func(*types.ClaudeAgentOptions)) (*claude.Client, context.Context) {
	t.Helper()

	requireAuth(t)
	cliPath := requireClaude(t)

	opts := types.NewClaudeAgentOptions().
		WithCLIPath(cliPath).
		WithPermissionMode(types.PermissionModeBypassPermissions)
	if configure != nil {
		configure(opts)
	}

	ctx, cancel := CreateTestContext(t, 60*time.Second)
	t.Cleanup(cancel)

	client, err := claude.NewClient(ctx, opts)
	if err != nil {
		t.Fatalf("setupClient: NewClient: %v", err)
	}
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("setupClient: Connect: %v", err)
	}

	t.Cleanup(func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		_ = client.Close(closeCtx)
	})

	return client, ctx
}

// collectUntilResult reads from the ReceiveResponse channel until it sees a
// ResultMessage or ctx expires. Returns all messages received (including the
// terminator). Fails the test if no ResultMessage arrives.
func collectUntilResult(t *testing.T, ctx context.Context, client *claude.Client) []types.Message {
	t.Helper()

	var collected []types.Message
	for msg := range client.ReceiveResponse(ctx) {
		collected = append(collected, msg)
		if _, ok := msg.(*types.ResultMessage); ok {
			return collected
		}
		if ctx.Err() != nil {
			t.Fatalf("collectUntilResult: context expired after %d messages", len(collected))
		}
	}

	t.Fatalf("collectUntilResult: channel closed before ResultMessage (received %d messages)", len(collected))
	return collected
}

// findAssistantText scans collected messages and returns the concatenated
// text content of all AssistantMessage TextBlock entries, lowercased for
// easy assertions.
func findAssistantText(msgs []types.Message) string {
	var sb strings.Builder
	for _, msg := range msgs {
		ass, ok := msg.(*types.AssistantMessage)
		if !ok {
			continue
		}
		for _, block := range ass.Content {
			if tb, ok := block.(*types.TextBlock); ok {
				sb.WriteString(tb.Text)
				sb.WriteString(" ")
			}
		}
	}
	return strings.ToLower(sb.String())
}

// requireClaude resolves the real `claude` CLI, skipping the test if it is not
// on PATH or at a known install location.
func requireClaude(t *testing.T) string {
	t.Helper()
	return FindRealCLI(t)
}

// requireAuth skips the test unless Claude credentials are available via
// ANTHROPIC_API_KEY, CLAUDE_API_KEY, or a logged-in CLI credentials file.
func requireAuth(t *testing.T) {
	t.Helper()

	if os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("CLAUDE_API_KEY") != "" {
		return
	}

	credentialsPath := filepath.Join(os.Getenv("HOME"), ".claude", ".credentials.json")
	if _, err := os.Stat(credentialsPath); err == nil {
		return
	}

	t.Skip("Claude auth not available - set ANTHROPIC_API_KEY, CLAUDE_API_KEY, or log in with Claude CLI")
}

// requireRunTurns skips token-spending tests unless CLAUDE_SDK_RUN_TURNS=1 is
// set, keeping quota-consuming turns opt-in.
func requireRunTurns(t *testing.T) {
	t.Helper()

	if os.Getenv("CLAUDE_SDK_RUN_TURNS") != "1" {
		t.Skip("CLAUDE_SDK_RUN_TURNS=1 not set - skipping token-spending integration test")
	}
}

// safetyNetSettings snapshots and restores ~/.claude/settings.json around a
// test that may mutate it.
func safetyNetSettings(t *testing.T) {
	t.Helper()
	safetyNetClaudeConfigFile(t, "settings.json")
}

// safetyNetHooks snapshots and restores ~/.claude/hooks.json around a test that
// may mutate it.
func safetyNetHooks(t *testing.T) {
	t.Helper()
	safetyNetClaudeConfigFile(t, "hooks.json")
}

// safetyNetClaudeConfigFile snapshots a ~/.claude config file before the test
// and restores its original contents (or removes a test-created file) on
// cleanup, so integration tests never leak edits into the developer's real
// Claude configuration.
func safetyNetClaudeConfigFile(t *testing.T, name string) {
	t.Helper()

	path := filepath.Join(os.Getenv("HOME"), ".claude", name)
	data, err := os.ReadFile(path)
	existed := err == nil
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("snapshot %s: %v", path, err)
	}

	t.Cleanup(func() {
		if existed {
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Errorf("restore %s parent: %v", path, err)
				return
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Errorf("restore %s: %v", path, err)
			}
			return
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			t.Errorf("remove test-created %s: %v", path, err)
		}
	})
}
