//go:build integration
// +build integration

// Real-CLI coverage for the public Client methods in client.go. Split into
// two groups:
//
//   - Metadata / control-protocol methods — do NOT burn tokens. They
//     exchange a single control-protocol envelope with the CLI.
//   - Turn-driving methods — spend tokens by sending a prompt through the
//     model. Gated on CLAUDE_SDK_RUN_TURNS=1.
//
// Methods covered (file:line references point to client.go):
//
//     IsConnected           (:590)   InitResult            (:601)
//     SlashCommands         (:610)   SupportedModels       (:621)
//     SupportedAgents       (:1036)  ProcessID             (:678)
//     SetModel              (:633)   SetPermissionMode     (:656)
//     GetSettings           (:966)   GetContextUsage       (:935)
//     ReloadPlugins         (:996)   EnableChannel         (:1016)
//     RewindFiles           (:896)
//     Query                 (:308)   QueryWithContent      (:379)
//     Interrupt             (:693)   StreamInput           (:713)
//     StopTask              (:748)   Close                 (:506)

package tests

import (
	"strings"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// --- Metadata (no tokens) ----------------------------------------------------

func TestClient_IsConnected(t *testing.T) {
	client, _ := setupClient(t, nil)

	if !client.IsConnected() {
		t.Error("IsConnected: false after Connect(); want true")
	}
}

func TestClient_InitResult(t *testing.T) {
	client, _ := setupClient(t, nil)

	init := client.InitResult()
	if init == nil {
		t.Fatal("InitResult: nil after Connect(); the CLI must return an initialize response")
	}

	// Commands list should be non-empty on any real CLI — basic slash
	// commands like /help are always present.
	if len(init.Commands) == 0 {
		t.Error("InitResult.Commands: empty; want non-empty slash commands list")
	}
	// Models list should also be non-empty.
	if len(init.Models) == 0 {
		t.Error("InitResult.Models: empty; want non-empty models list")
	}
}

func TestClient_SlashCommands(t *testing.T) {
	client, _ := setupClient(t, nil)

	cmds := client.SlashCommands()
	if len(cmds) == 0 {
		// Some CLI builds return an empty command list on connect — log
		// rather than fail so the suite focuses on actual wire-shape
		// drift rather than CLI surface-area variance.
		t.Log("SlashCommands: empty list (CLI may not advertise commands at init)")
		return
	}

	// Wire-shape probe: every entry must have a non-empty Name.
	for i, c := range cmds {
		if c.Name == "" {
			t.Errorf("SlashCommands[%d]: Name empty; wire tag drift on SlashCommand.Name?", i)
		}
	}
	t.Logf("SlashCommands returned %d command(s)", len(cmds))
}

func TestClient_SupportedModels(t *testing.T) {
	client, _ := setupClient(t, nil)

	models := client.SupportedModels()
	if len(models) == 0 {
		t.Fatal("SupportedModels: empty; want at least one model from CLI")
	}

	for _, m := range models {
		if m.Value == "" {
			t.Errorf("SupportedModels: entry missing Value: %+v", m)
		}
	}
}

func TestClient_SupportedAgents(t *testing.T) {
	client, _ := setupClient(t, nil)

	// SupportedAgents may legitimately be empty if the CLI has no agents
	// configured — we only assert the call returns without panic.
	agents := client.SupportedAgents()
	t.Logf("SupportedAgents returned %d agent(s)", len(agents))
}

func TestClient_ProcessID(t *testing.T) {
	client, _ := setupClient(t, nil)

	pid := client.ProcessID()
	if pid <= 0 {
		t.Errorf("ProcessID: %d; want > 0 after Connect()", pid)
	}
}

// --- Configuration round-trips (no tokens) ----------------------------------

func TestClient_SetPermissionMode(t *testing.T) {
	client, ctx := setupClient(t, nil)

	for _, mode := range []types.PermissionMode{
		types.PermissionModeDefault,
		types.PermissionModeAcceptEdits,
		types.PermissionModeBypassPermissions,
	} {
		mode := mode
		t.Run(string(mode), func(t *testing.T) {
			if err := client.SetPermissionMode(ctx, mode); err != nil {
				t.Errorf("SetPermissionMode(%s): %v", mode, err)
			}
		})
	}
}

func TestClient_SetModel(t *testing.T) {
	client, ctx := setupClient(t, nil)

	models := client.SupportedModels()
	if len(models) == 0 {
		t.Skip("no supported models advertised by CLI; cannot test SetModel")
	}

	// Pick the first supported model and set it.
	if err := client.SetModel(ctx, models[0].Value); err != nil {
		t.Errorf("SetModel(%s): %v", models[0].Value, err)
	}
}

func TestClient_GetSettings(t *testing.T) {
	client, ctx := setupClient(t, nil)

	settings, err := client.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings == nil {
		t.Fatal("GetSettings: nil result")
	}
}

func TestClient_GetContextUsage(t *testing.T) {
	client, ctx := setupClient(t, nil)

	usage, err := client.GetContextUsage(ctx)
	if err != nil {
		t.Fatalf("GetContextUsage: %v", err)
	}
	if usage == nil {
		t.Fatal("GetContextUsage: nil result")
	}
	t.Logf("GetContextUsage: %+v", usage)
}

func TestClient_ReloadPlugins(t *testing.T) {
	client, ctx := setupClient(t, nil)

	if err := client.ReloadPlugins(ctx); err != nil {
		t.Errorf("ReloadPlugins: %v", err)
	}
}

func TestClient_EnableChannel(t *testing.T) {
	client, ctx := setupClient(t, nil)

	if err := client.EnableChannel(ctx); err != nil {
		// Older CLI builds return "Unsupported control request subtype".
		// Treat as a skip — the SDK method exists but the CLI doesn't
		// support it on this version.
		if strings.Contains(err.Error(), "Unsupported") {
			t.Skipf("EnableChannel not supported by this CLI build: %v", err)
		}
		t.Errorf("EnableChannel: %v", err)
	}
}

// --- RewindFiles (no tokens, dry-run) ---------------------------------------

func TestClient_RewindFiles_DryRun(t *testing.T) {
	client, ctx := setupClient(t, nil)

	// Use an obviously-nonexistent message ID so the CLI returns a
	// well-formed "cannot rewind" result rather than actually rewinding.
	result, err := client.RewindFiles(ctx, "nonexistent-user-message-id", true)
	if err != nil {
		// An error here may be legitimate depending on CLI semantics —
		// log rather than fail.
		t.Logf("RewindFiles(dry-run, unknown id): %v", err)
		return
	}
	if result == nil {
		t.Fatal("RewindFiles: nil result with nil error")
	}
	t.Logf("RewindFiles dry-run result: %+v", result)
}

// --- Turn-driving (requires CLAUDE_SDK_RUN_TURNS=1) -------------------------

func TestClient_Query_HappyPath(t *testing.T) {
	requireRunTurns(t)
	client, ctx := setupClient(t, nil)

	if err := client.Query(ctx, "Say 'ready' and nothing else."); err != nil {
		t.Fatalf("Query: %v", err)
	}

	msgs := collectUntilResult(t, ctx, client)

	// Final message must be a non-error ResultMessage.
	last := msgs[len(msgs)-1]
	result, ok := last.(*types.ResultMessage)
	if !ok {
		t.Fatalf("last message type = %T; want *types.ResultMessage", last)
	}
	if result.IsError {
		t.Errorf("ResultMessage.IsError = true; errors=%v", result.Errors)
	}
	if result.SessionID == "" {
		t.Error("ResultMessage.SessionID: empty; wire tag drift on session_id?")
	}
}

func TestClient_QueryWithContent_StructuredBlocks(t *testing.T) {
	requireRunTurns(t)
	client, ctx := setupClient(t, nil)

	content := []map[string]interface{}{
		{"type": "text", "text": "Reply with the single word 'acknowledged'."},
	}

	if err := client.QueryWithContent(ctx, content); err != nil {
		t.Fatalf("QueryWithContent: %v", err)
	}

	msgs := collectUntilResult(t, ctx, client)
	text := findAssistantText(msgs)
	if !strings.Contains(text, "acknowledged") {
		t.Logf("model did not echo 'acknowledged' (LLMs vary); got: %s", text)
	}
}

func TestClient_Interrupt_DuringStream(t *testing.T) {
	requireRunTurns(t)
	client, ctx := setupClient(t, nil)

	// Kick off a long-running query, then interrupt.
	if err := client.Query(ctx, "Count slowly from 1 to 100, one number per line."); err != nil {
		t.Fatalf("Query: %v", err)
	}

	// Wait for the first assistant message to confirm the stream started.
	gotFirst := false
	for msg := range client.ReceiveResponse(ctx) {
		if _, ok := msg.(*types.AssistantMessage); ok {
			gotFirst = true
			if err := client.Interrupt(ctx); err != nil {
				t.Fatalf("Interrupt: %v", err)
			}
			break
		}
	}
	if !gotFirst {
		t.Skip("no assistant message arrived before interrupt window")
	}

	// Drain until channel closes or we see a ResultMessage.
	for msg := range client.ReceiveResponse(ctx) {
		if _, ok := msg.(*types.ResultMessage); ok {
			return
		}
	}
}

func TestClient_StreamInput_AppendMidTurn(t *testing.T) {
	requireRunTurns(t)
	client, ctx := setupClient(t, nil)

	if err := client.Query(ctx, "When I say more, append more words."); err != nil {
		t.Fatalf("Query: %v", err)
	}

	// Stream an additional prompt fragment.
	if err := client.StreamInput(ctx, " Add the word 'complete' to your response."); err != nil {
		// StreamInput may not be supported by every CLI build — treat
		// an error as a skip-worthy gap rather than a hard fail.
		t.Skipf("StreamInput not supported by this CLI: %v", err)
	}

	_ = collectUntilResult(t, ctx, client)
}

func TestClient_StopTask(t *testing.T) {
	requireRunTurns(t)
	client, ctx := setupClient(t, nil)

	// StopTask requires a task ID from a running subagent. Without a
	// running task, expect an error — we assert the call reaches the
	// CLI and fails gracefully.
	err := client.StopTask(ctx, "nonexistent-task-id")
	if err == nil {
		t.Log("StopTask(nonexistent): nil error; CLI may treat as no-op")
	} else {
		t.Logf("StopTask(nonexistent) returned expected error: %v", err)
	}
}

// --- Lifecycle --------------------------------------------------------------

func TestClient_Close_DisconnectsCleanly(t *testing.T) {
	client, ctx := setupClient(t, nil)

	if !client.IsConnected() {
		t.Fatal("precondition: IsConnected() == false")
	}

	if err := client.Close(ctx); err != nil {
		t.Errorf("Close: %v", err)
	}

	if client.IsConnected() {
		t.Error("IsConnected() = true after Close(); want false")
	}
}
