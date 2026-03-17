package claude

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/schlunsen/claude-agent-sdk-go/types"
)

func TestNewClient_NilOptions(t *testing.T) {
	// Disable version checking to speed up tests
	t.Setenv("CLAUDE_AGENT_SDK_SKIP_VERSION_CHECK", "1")

	ctx := context.Background()

	client, err := NewClient(ctx, nil)
	if err == nil {
		// CLI might be installed - that's OK, just clean up
		if client != nil {
			_ = client.Close(ctx)
		}
		return
	}

	// Should get CLINotFoundError
	if !types.IsCLINotFoundError(err) {
		t.Logf("Expected CLINotFoundError but got: %v", err)
	}
}

func TestNewClient_InvalidCLIPath(t *testing.T) {
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/nonexistent/path/to/claude")

	client, err := NewClient(ctx, opts)
	if err != nil {
		// Expected - CLI path doesn't exist
		// However, NewClient doesn't validate the path, only Connect does
		// So we might get a client back
		if client != nil {
			_ = client.Close(ctx)
		}
	}
}

func TestNewClient_ConflictingPermissionOptions(t *testing.T) {
	ctx := context.Background()

	// Create a dummy callback
	canUseTool := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
		return types.PermissionResultAllow{Behavior: "allow"}, nil
	}

	// This should fail because both are set
	promptTool := "cli"
	opts := types.NewClaudeAgentOptions().
		WithCLIPath("/bin/echo").
		WithCanUseTool(canUseTool).
		WithPermissionPromptToolName(promptTool)

	_, err := NewClient(ctx, opts)
	if err == nil {
		t.Fatal("expected error for conflicting permission options")
	}

	if err.Error() != "can_use_tool callback cannot be used with permission_prompt_tool_name" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestClient_ConnectBeforeQuery(t *testing.T) {
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	// Try to query without connecting
	err = client.Query(ctx, "test")
	if err == nil {
		t.Fatal("expected error when querying without connecting")
	}

	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

func TestClient_EmptyPrompt(t *testing.T) {
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	// Without connecting, should get connection error first
	err = client.Query(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty prompt without connection")
	}

	// Should be connection error since we haven't connected
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError when not connected, got: %v", err)
	}
}

func TestClient_IsConnected(t *testing.T) {
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	// Should not be connected initially
	if client.IsConnected() {
		t.Error("client should not be connected before Connect()")
	}

	// After close, should not be connected
	_ = client.Close(ctx)
	if client.IsConnected() {
		t.Error("client should not be connected after Close()")
	}
}

func TestClient_DoubleConnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	// First connect attempt (will likely fail with /bin/echo)
	err1 := client.Connect(ctx)

	// Second connect attempt
	err2 := client.Connect(ctx)

	// If first connect succeeded, second should fail with "already connected"
	if err1 == nil && err2 == nil {
		t.Error("expected error on second Connect() call")
	}

	// If second connect got an error, check if it's the right one
	if err2 != nil && types.IsControlProtocolError(err2) {
		// Good - got the expected error
		if err2.Error() != "client already connected" {
			t.Logf("Got control protocol error but unexpected message: %v", err2)
		}
	}
}

func TestClient_CloseIdempotent(t *testing.T) {
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}

	// Close multiple times should not panic or error
	err1 := client.Close(ctx)
	err2 := client.Close(ctx)
	err3 := client.Close(ctx)

	// All should succeed (or at least not panic)
	_ = err1
	_ = err2
	_ = err3
}

func TestClient_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	// Cancel context before operations
	cancel()

	// Operations should respect cancellation
	err = client.Connect(ctx)
	// May fail due to cancellation or other reasons - just ensure no panic
	_ = err
}

// TestClient_Integration is an integration test that requires Claude CLI to be installed.
func TestClient_Integration(t *testing.T) {
	// This test requires actual Claude CLI and API key
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration test (set RUN_INTEGRATION_TESTS=1 to run)")
	}

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := types.NewClaudeAgentOptions().
		WithModel("claude-3-5-sonnet-latest").
		WithPermissionMode(types.PermissionModeBypassPermissions)

	client, err := NewClient(ctx, opts)
	if err != nil {
		if types.IsCLINotFoundError(err) {
			t.Skip("Claude CLI not installed")
		}
		t.Fatal(err)
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	// Connect
	if err := client.Connect(ctx); err != nil {
		if types.IsCLIConnectionError(err) {
			t.Skip("Could not connect to Claude CLI")
		}
		t.Fatal(err)
	}

	// First query
	if err := client.Query(ctx, "What is 2+2? Reply with just the number."); err != nil {
		t.Fatal(err)
	}

	// Receive response
	var messageCount int
	for msg := range client.ReceiveResponse(ctx) {
		messageCount++
		if _, ok := msg.(*types.ResultMessage); ok {
			break
		}
	}

	if messageCount == 0 {
		t.Fatal("expected at least one message")
	}

	t.Logf("First query received %d messages", messageCount)

	// Second query in same session
	if err := client.Query(ctx, "What is 3+3? Reply with just the number."); err != nil {
		t.Fatal(err)
	}

	// Receive second response
	messageCount = 0
	for msg := range client.ReceiveResponse(ctx) {
		messageCount++
		if _, ok := msg.(*types.ResultMessage); ok {
			break
		}
	}

	if messageCount == 0 {
		t.Fatal("expected at least one message in second query")
	}

	t.Logf("Second query received %d messages", messageCount)
}

// TestClient_MultipleQueries tests multiple query/response cycles
func TestClient_MultipleQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	opts := types.NewClaudeAgentOptions().
		WithModel("claude-3-5-sonnet-latest").
		WithPermissionMode(types.PermissionModeBypassPermissions)

	client, err := NewClient(ctx, opts)
	if err != nil {
		if types.IsCLINotFoundError(err) {
			t.Skip("Claude CLI not installed")
		}
		t.Fatal(err)
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	if err := client.Connect(ctx); err != nil {
		if types.IsCLIConnectionError(err) {
			t.Skip("Could not connect to Claude CLI")
		}
		t.Fatal(err)
	}

	// Send 3 queries in sequence
	queries := []string{
		"Say 'first'",
		"Say 'second'",
		"Say 'third'",
	}

	for i, prompt := range queries {
		if err := client.Query(ctx, prompt); err != nil {
			t.Fatalf("Query %d failed: %v", i+1, err)
		}

		// Receive response
		gotResult := false
		for msg := range client.ReceiveResponse(ctx) {
			if _, ok := msg.(*types.ResultMessage); ok {
				gotResult = true
				break
			}
		}

		if !gotResult {
			t.Fatalf("Query %d did not receive ResultMessage", i+1)
		}

		t.Logf("Query %d completed", i+1)
	}
}

func TestParseInitResult_Nil(t *testing.T) {
	result := parseInitResult(nil)
	if result != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestParseInitResult_EmptyMap(t *testing.T) {
	result := parseInitResult(map[string]interface{}{})
	if result == nil {
		t.Fatal("expected non-nil result for empty map")
	}
	if len(result.Commands) != 0 {
		t.Fatalf("expected 0 commands, got %d", len(result.Commands))
	}
}

func TestParseInitResult_WithCommands(t *testing.T) {
	raw := map[string]interface{}{
		"commands": []interface{}{
			map[string]interface{}{
				"name":         "compact",
				"description":  "Compact conversation context",
				"argumentHint": "",
			},
			map[string]interface{}{
				"name":         "dev",
				"description":  "Run development workflow",
				"argumentHint": "[phase]",
			},
			map[string]interface{}{
				"name":         "plan",
				"description":  "Enter plan mode",
				"argumentHint": "",
			},
		},
		"models": []interface{}{}, // other fields should not break parsing
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(result.Commands))
	}

	// Verify first command
	if result.Commands[0].Name != "compact" {
		t.Errorf("expected first command name 'compact', got %q", result.Commands[0].Name)
	}
	if result.Commands[0].Description != "Compact conversation context" {
		t.Errorf("unexpected description: %q", result.Commands[0].Description)
	}

	// Verify second command with argumentHint
	if result.Commands[1].Name != "dev" {
		t.Errorf("expected second command name 'dev', got %q", result.Commands[1].Name)
	}
	if result.Commands[1].ArgumentHint != "[phase]" {
		t.Errorf("expected argumentHint '[phase]', got %q", result.Commands[1].ArgumentHint)
	}

	// Verify raw is preserved
	if result.Raw == nil {
		t.Error("expected raw map to be preserved")
	}
}

func TestParseInitResult_SkipsEmptyNames(t *testing.T) {
	raw := map[string]interface{}{
		"commands": []interface{}{
			map[string]interface{}{
				"name":        "valid",
				"description": "A valid command",
			},
			map[string]interface{}{
				"description": "Missing name",
			},
			map[string]interface{}{
				"name":        "",
				"description": "Empty name",
			},
		},
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Commands) != 1 {
		t.Fatalf("expected 1 command (skipping empty names), got %d", len(result.Commands))
	}
	if result.Commands[0].Name != "valid" {
		t.Errorf("expected 'valid', got %q", result.Commands[0].Name)
	}
}

func TestParseInitResult_InvalidCommandsType(t *testing.T) {
	raw := map[string]interface{}{
		"commands": "not an array",
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Commands) != 0 {
		t.Fatalf("expected 0 commands for invalid type, got %d", len(result.Commands))
	}
}

func TestClient_SlashCommands_BeforeConnect(t *testing.T) {
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	cmds := client.SlashCommands()
	if cmds != nil {
		t.Errorf("expected nil before connect, got %v", cmds)
	}
}

func TestClient_InitResult_BeforeConnect(t *testing.T) {
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	result := client.InitResult()
	if result != nil {
		t.Errorf("expected nil before connect, got %v", result)
	}
}

// TestSetModel_BeforeConnect ensures SetModel returns CLIConnectionError when not connected.
func TestSetModel_BeforeConnect(t *testing.T) {
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.SetModel(ctx, "haiku")
	if err == nil {
		t.Fatal("expected error when calling SetModel before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestSetPermissionMode_BeforeConnect ensures SetPermissionMode returns CLIConnectionError when not connected.
func TestSetPermissionMode_BeforeConnect(t *testing.T) {
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.SetPermissionMode(ctx, types.PermissionModePlan)
	if err == nil {
		t.Fatal("expected error when calling SetPermissionMode before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestSupportedModels_BeforeConnect ensures SupportedModels returns nil when not connected.
func TestSupportedModels_BeforeConnect(t *testing.T) {
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	models := client.SupportedModels()
	if models != nil {
		t.Errorf("expected nil before Connect(), got %v", models)
	}
}

// TestParseInitResult_WithModels verifies that the models array is parsed correctly.
func TestParseInitResult_WithModels(t *testing.T) {
	raw := map[string]interface{}{
		"models": []interface{}{
			map[string]interface{}{
				"value":       "haiku",
				"displayName": "Haiku",
				"description": "Fast model",
			},
			map[string]interface{}{
				"value":       "sonnet",
				"displayName": "Sonnet",
			},
		},
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(result.Models))
	}
	if result.Models[0].Value != "haiku" {
		t.Errorf("expected first model value 'haiku', got %q", result.Models[0].Value)
	}
	if result.Models[0].DisplayName != "Haiku" {
		t.Errorf("expected first model displayName 'Haiku', got %q", result.Models[0].DisplayName)
	}
	if result.Models[0].Description != "Fast model" {
		t.Errorf("expected first model description 'Fast model', got %q", result.Models[0].Description)
	}
	if result.Models[1].Value != "sonnet" {
		t.Errorf("expected second model value 'sonnet', got %q", result.Models[1].Value)
	}
	if result.Models[1].Description != "" {
		t.Errorf("expected second model description empty, got %q", result.Models[1].Description)
	}
}

// TestParseInitResult_ModelsEmptyArray verifies empty models array is handled.
func TestParseInitResult_ModelsEmptyArray(t *testing.T) {
	raw := map[string]interface{}{
		"models": []interface{}{},
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Models) != 0 {
		t.Errorf("expected 0 models, got %d", len(result.Models))
	}
}

// TestParseInitResult_ModelsInvalidType verifies graceful handling of unexpected type.
func TestParseInitResult_ModelsInvalidType(t *testing.T) {
	raw := map[string]interface{}{
		"models": "not-an-array",
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Should not panic, models should be nil/empty.
	if len(result.Models) != 0 {
		t.Errorf("expected 0 models for invalid type, got %d", len(result.Models))
	}
}

// TestParseInitResult_ModelsMissingFields verifies partial model entries are still parsed.
func TestParseInitResult_ModelsMissingFields(t *testing.T) {
	raw := map[string]interface{}{
		"models": []interface{}{
			map[string]interface{}{
				"value": "haiku",
				// no displayName or description
			},
		},
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(result.Models))
	}
	if result.Models[0].Value != "haiku" {
		t.Errorf("expected 'haiku', got %q", result.Models[0].Value)
	}
	if result.Models[0].DisplayName != "" {
		t.Errorf("expected empty displayName, got %q", result.Models[0].DisplayName)
	}
}

// TestParseInitResult_ModelsAndCommandsTogether verifies both fields are parsed when present.
func TestParseInitResult_ModelsAndCommandsTogether(t *testing.T) {
	raw := map[string]interface{}{
		"commands": []interface{}{
			map[string]interface{}{
				"name":        "compact",
				"description": "Compact context",
			},
		},
		"models": []interface{}{
			map[string]interface{}{
				"value":       "haiku",
				"displayName": "Haiku",
			},
		},
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(result.Commands))
	}
	if len(result.Models) != 1 {
		t.Errorf("expected 1 model, got %d", len(result.Models))
	}
	if result.Commands[0].Name != "compact" {
		t.Errorf("unexpected command name: %q", result.Commands[0].Name)
	}
	if result.Models[0].Value != "haiku" {
		t.Errorf("unexpected model value: %q", result.Models[0].Value)
	}
}

// TestParseInitResult_ModelsSkipsEmptyValue verifies models with no value are skipped.
func TestParseInitResult_ModelsSkipsEmptyValue(t *testing.T) {
	raw := map[string]interface{}{
		"models": []interface{}{
			map[string]interface{}{
				"displayName": "No Value Model",
				// value intentionally missing
			},
			map[string]interface{}{
				"value":       "haiku",
				"displayName": "Haiku",
			},
		},
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Models) != 1 {
		t.Fatalf("expected 1 model (skipping empty value), got %d", len(result.Models))
	}
	if result.Models[0].Value != "haiku" {
		t.Errorf("expected 'haiku', got %q", result.Models[0].Value)
	}
}

// TestSupportedModels_ReturnsFromInitResult verifies SupportedModels uses the stored init result.
func TestSupportedModels_ReturnsFromInitResult(t *testing.T) {
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	// Manually inject an initResult to test SupportedModels() without a live connection.
	client.mu.Lock()
	client.initResult = &types.InitializeResult{
		Models: []types.ModelInfo{
			{Value: "haiku", DisplayName: "Haiku", Description: "Fast"},
			{Value: "sonnet", DisplayName: "Sonnet"},
		},
	}
	client.mu.Unlock()

	models := client.SupportedModels()
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].Value != "haiku" {
		t.Errorf("expected 'haiku', got %q", models[0].Value)
	}
	if models[1].Value != "sonnet" {
		t.Errorf("expected 'sonnet', got %q", models[1].Value)
	}
}

// BenchmarkClient benchmarks the Client type
func BenchmarkClient_Create(b *testing.B) {
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, err := NewClient(ctx, opts)
		if err == nil {
			_ = client.Close(ctx)
		}
	}
}
