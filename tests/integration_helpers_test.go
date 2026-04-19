//go:build integration
// +build integration

// Shared test helpers for the real-CLI integration suite. See integration_test.go
// for the documented gating and run-command conventions.

package tests

import (
	"context"
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
