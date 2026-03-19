package claude

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
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

// --- Phase B: New method tests ---

// TestInterrupt_BeforeConnect ensures Interrupt returns connection error when not connected.
func TestInterrupt_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.Interrupt(ctx)
	if err == nil {
		t.Fatal("expected error when calling Interrupt before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestStreamInput_BeforeConnect ensures StreamInput returns connection error when not connected.
func TestStreamInput_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.StreamInput(ctx, "hello")
	if err == nil {
		t.Fatal("expected error when calling StreamInput before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestStreamInput_EmptyContent ensures StreamInput rejects empty content.
func TestStreamInput_EmptyContent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.StreamInput(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

// TestStopTask_BeforeConnect ensures StopTask returns connection error when not connected.
func TestStopTask_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.StopTask(ctx, "task-123")
	if err == nil {
		t.Fatal("expected error when calling StopTask before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestStopTask_EmptyTaskID ensures StopTask rejects empty task ID.
func TestStopTask_EmptyTaskID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.StopTask(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty taskID")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

// TestMCPServerStatus_BeforeConnect ensures MCPServerStatus returns connection error when not connected.
func TestMCPServerStatus_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	_, err = client.MCPServerStatus(ctx)
	if err == nil {
		t.Fatal("expected error when calling MCPServerStatus before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestReconnectMCPServer_BeforeConnect ensures ReconnectMCPServer returns connection error when not connected.
func TestReconnectMCPServer_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.ReconnectMCPServer(ctx, "my-server")
	if err == nil {
		t.Fatal("expected error when calling ReconnectMCPServer before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestReconnectMCPServer_EmptyName ensures ReconnectMCPServer rejects empty server name.
func TestReconnectMCPServer_EmptyName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.ReconnectMCPServer(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty serverName")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

// TestToggleMCPServer_BeforeConnect ensures ToggleMCPServer returns connection error when not connected.
func TestToggleMCPServer_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.ToggleMCPServer(ctx, "my-server", true)
	if err == nil {
		t.Fatal("expected error when calling ToggleMCPServer before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestToggleMCPServer_EmptyName ensures ToggleMCPServer rejects empty server name.
func TestToggleMCPServer_EmptyName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.ToggleMCPServer(ctx, "", false)
	if err == nil {
		t.Fatal("expected error for empty serverName")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

// TestSetMCPServers_BeforeConnect ensures SetMCPServers returns connection error when not connected.
func TestSetMCPServers_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	_, err = client.SetMCPServers(ctx, map[string]interface{}{"server1": map[string]interface{}{}})
	if err == nil {
		t.Fatal("expected error when calling SetMCPServers before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestSetMCPServers_NilConfig ensures SetMCPServers rejects nil servers config.
func TestSetMCPServers_NilConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	_, err = client.SetMCPServers(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil servers")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

// TestRewindFiles_BeforeConnect ensures RewindFiles returns connection error when not connected.
func TestRewindFiles_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	_, err = client.RewindFiles(ctx, "msg-123", false)
	if err == nil {
		t.Fatal("expected error when calling RewindFiles before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

// TestRewindFiles_EmptyUserMessageID ensures RewindFiles rejects empty user message ID.
func TestRewindFiles_EmptyUserMessageID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	_, err = client.RewindFiles(ctx, "", false)
	if err == nil {
		t.Fatal("expected error for empty userMessageID")
	}
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Errorf("expected ErrEmptyParameter, got: %v", err)
	}
}

// TestSupportedAgents_BeforeConnect ensures SupportedAgents returns nil when not connected.
func TestSupportedAgents_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	agents := client.SupportedAgents()
	if agents != nil {
		t.Errorf("expected nil before Connect(), got %v", agents)
	}
}

// TestSupportedAgents_ReturnsFromInitResult verifies SupportedAgents uses the stored init result.
func TestSupportedAgents_ReturnsFromInitResult(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	client.mu.Lock()
	client.initResult = &types.InitializeResult{
		Agents: []types.AgentInfo{
			{Name: "Explore", Description: "Fast agent for exploring codebases", Model: "sonnet"},
			{Name: "Plan", Description: "Software architect agent"},
		},
	}
	client.mu.Unlock()

	agents := client.SupportedAgents()
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	if agents[0].Name != "Explore" {
		t.Errorf("expected 'Explore', got %q", agents[0].Name)
	}
	if agents[0].Model != "sonnet" {
		t.Errorf("expected model 'sonnet', got %q", agents[0].Model)
	}
	if agents[1].Name != "Plan" {
		t.Errorf("expected 'Plan', got %q", agents[1].Name)
	}
	if agents[1].Model != "" {
		t.Errorf("expected empty model, got %q", agents[1].Model)
	}
}

// TestParseInitResult_WithAgents verifies that agents are parsed from the init result.
func TestParseInitResult_WithAgents(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"agents": []interface{}{
			map[string]interface{}{
				"name":        "Explore",
				"description": "Fast agent for exploring codebases",
				"model":       "sonnet",
			},
			map[string]interface{}{
				"name":        "Plan",
				"description": "Software architect agent",
			},
		},
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(result.Agents))
	}
	if result.Agents[0].Name != "Explore" {
		t.Errorf("expected 'Explore', got %q", result.Agents[0].Name)
	}
	if result.Agents[0].Description != "Fast agent for exploring codebases" {
		t.Errorf("unexpected description: %q", result.Agents[0].Description)
	}
	if result.Agents[0].Model != "sonnet" {
		t.Errorf("expected model 'sonnet', got %q", result.Agents[0].Model)
	}
	if result.Agents[1].Name != "Plan" {
		t.Errorf("expected 'Plan', got %q", result.Agents[1].Name)
	}
	if result.Agents[1].Model != "" {
		t.Errorf("expected empty model for Plan, got %q", result.Agents[1].Model)
	}
}

// TestParseInitResult_AgentsSkipsEmptyName verifies agents with empty names are skipped.
func TestParseInitResult_AgentsSkipsEmptyName(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"agents": []interface{}{
			map[string]interface{}{
				"name":        "Valid",
				"description": "A valid agent",
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
	if len(result.Agents) != 1 {
		t.Fatalf("expected 1 agent (skipping empty names), got %d", len(result.Agents))
	}
	if result.Agents[0].Name != "Valid" {
		t.Errorf("expected 'Valid', got %q", result.Agents[0].Name)
	}
}

// TestParseInitResult_AgentsInvalidType verifies graceful handling when agents is not an array.
func TestParseInitResult_AgentsInvalidType(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"agents": "not-an-array",
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Agents) != 0 {
		t.Errorf("expected 0 agents for invalid type, got %d", len(result.Agents))
	}
}

// TestParseInitResult_AllFieldsTogether verifies commands, models, and agents together.
func TestParseInitResult_AllFieldsTogether(t *testing.T) {
	t.Parallel()
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
		"agents": []interface{}{
			map[string]interface{}{
				"name":        "Explore",
				"description": "Explorer",
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
	if len(result.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(result.Agents))
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
