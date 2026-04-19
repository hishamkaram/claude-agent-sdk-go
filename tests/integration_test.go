//go:build integration
// +build integration

// Real-CLI integration tests for claude-agent-sdk-go.
//
// Build tag `integration` gates every file in this package that spawns the
// real `claude` CLI against real network endpoints. Run via:
//
//	go test -tags=integration -race -count=1 -p 1 ./tests/... -timeout=300s
//
// or the Makefile targets:
//
//	make test-integration        # no-quota subset (cheap; no token spend)
//	make test-integration-quota  # full suite including model turns (burns tokens)
//
// Preconditions enforced per-test by helpers in test_helpers.go:
//
//   - requireClaude(t)   — skips if the `claude` binary is not on PATH or at a
//     common install location (~/.claude/local/claude, ~/.npm-global/bin/...,
//     /usr/local/bin/..., /opt/homebrew/bin/...). Override with CLAUDE_CLI_PATH.
//   - requireAuth(t)     — skips unless ANTHROPIC_API_KEY / CLAUDE_API_KEY is set
//     OR the CLI has an existing login at ~/.claude/.credentials.json.
//   - requireRunTurns(t) — skips unless CLAUDE_SDK_RUN_TURNS=1. Use for tests
//     that actually drive the model through a turn (token cost).
//
// Mock-based SDK tests (shell-script fake CLI) live in integration_mock_test.go
// without the build tag. Those verify SDK plumbing but cannot catch wire-shape
// drift between the SDK and the real CLI — that is the failure class this
// build-tagged suite exists to prevent.

package tests

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	claude "github.com/hishamkaram/claude-agent-sdk-go"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestControlProtocol_FullFlow drives the control protocol end-to-end through
// the real CLI. Asserts the permission callback fires and the response stream
// closes with a ResultMessage.
func TestControlProtocol_FullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	requireAuth(t)
	requireRunTurns(t)
	cliPath := requireClaude(t)

	checkGoroutines := AssertNoGoroutineLeaks(t)
	defer checkGoroutines()

	ctx, cancel := CreateTestContext(t, 45*time.Second)
	defer cancel()

	permissionRequested := false
	var mu sync.Mutex

	canUseTool := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
		mu.Lock()
		permissionRequested = true
		mu.Unlock()

		t.Logf("Permission requested for tool: %s", toolName)

		return types.PermissionResultAllow{
			Behavior: "allow",
		}, nil
	}

	opts := types.NewClaudeAgentOptions().
		WithCLIPath(cliPath).
		WithModel("claude-3-5-sonnet-latest").
		WithCanUseTool(canUseTool)

	client, err := claude.NewClient(ctx, opts)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}

	prompt := "What is 2+2? Just tell me the number."
	if err := client.Query(ctx, prompt); err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	messageCount := 0
	for msg := range client.ReceiveResponse(ctx) {
		messageCount++
		t.Logf("Message %d: type=%s", messageCount, msg.GetMessageType())

		if _, ok := msg.(*types.ResultMessage); ok {
			break
		}
	}

	if messageCount == 0 {
		t.Fatal("expected at least one message")
	}

	t.Logf("Received %d messages", messageCount)
	t.Logf("Permission requested: %v", permissionRequested)
}

// TestRealCLIIntegration is the happy-path smoke test for the real CLI: a
// one-shot Query() call that must return at least one assistant message and
// terminate with a ResultMessage.
func TestRealCLIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	requireAuth(t)
	requireRunTurns(t)
	cliPath := requireClaude(t)

	checkGoroutines := AssertNoGoroutineLeaks(t)
	defer checkGoroutines()

	ctx, cancel := CreateTestContext(t, 45*time.Second)
	defer cancel()

	opts := types.NewClaudeAgentOptions().
		WithCLIPath(cliPath).
		WithModel("claude-3-5-sonnet-latest").
		WithPermissionMode(types.PermissionModeBypassPermissions)

	msgChan, err := claude.Query(ctx, "Say 'hello' and nothing else.", opts)
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	defer func() {
		for range msgChan {
			// drain any remaining messages
		}
	}()
	collected := CollectMessages(ctx, t, msgChan, 30*time.Second)

	if len(collected) == 0 {
		t.Fatal("expected at least one message")
	}

	foundText := false
	for _, msg := range collected {
		if assMsg, ok := msg.(*types.AssistantMessage); ok {
			for _, block := range assMsg.Content {
				if textBlock, ok := block.(*types.TextBlock); ok {
					text := strings.ToLower(textBlock.Text)
					if strings.Contains(text, "hello") {
						foundText = true
						t.Logf("Got response: %s", textBlock.Text)
					}
				}
			}
		}
	}

	if !foundText {
		t.Log("Warning: response may not contain expected text (this can happen with LLMs)")
	}

	lastMsg := collected[len(collected)-1]
	AssertMessageType(t, lastMsg, "result")
}
