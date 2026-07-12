package claude

import (
	"context"
	"slices"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestParseInitResult_Nil(t *testing.T) {
	t.Parallel()
	result := parseInitResult(nil)
	if result != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestParseInitResult_EmptyMap(t *testing.T) {
	t.Parallel()
	result := parseInitResult(map[string]interface{}{})
	if result == nil {
		t.Fatal("expected non-nil result for empty map")
	}
	if len(result.Commands) != 0 {
		t.Fatalf("expected 0 commands, got %d", len(result.Commands))
	}
}

func TestParseInitResult_WithCommands(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	// Contract: a non-array "commands" value leaves the field nil (untouched).
	if result.Commands != nil {
		t.Errorf("expected nil Commands for invalid type, got %#v", result.Commands)
	}
}

func TestClient_SlashCommands_BeforeConnect(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestParseInitResult_WithModelCapabilities(t *testing.T) {
	t.Parallel()

	result := parseInitResult(map[string]interface{}{
		"models": []interface{}{
			map[string]interface{}{
				"value":                    "provider-model-a",
				"resolvedModel":            "provider-model-a-20260710",
				"displayName":              "Provider Model A",
				"description":              "Current account model",
				"supportsEffort":           true,
				"supportedEffortLevels":    []interface{}{"low", "high", "max"},
				"supportsAdaptiveThinking": true,
				"supportsFastMode":         true,
				"supportsAutoMode":         false,
				"disabled":                 true,
			},
		},
	})

	if result == nil || len(result.Models) != 1 {
		t.Fatalf("parseInitResult() = %#v, want one model", result)
	}
	model := result.Models[0]
	if model.ResolvedModel != "provider-model-a-20260710" || !model.SupportsEffort {
		t.Fatalf("model metadata = %+v", model)
	}
	wantEfforts := []types.EffortLevel{types.EffortLow, types.EffortHigh, types.EffortMax}
	if !slices.Equal(model.SupportedEffortLevels, wantEfforts) {
		t.Fatalf("SupportedEffortLevels = %v, want %v", model.SupportedEffortLevels, wantEfforts)
	}
	if !model.SupportsAdaptiveThinking || !model.SupportsFastMode || model.SupportsAutoMode || !model.Disabled {
		t.Fatalf("model capability flags = %+v", model)
	}
	if model.Raw["resolvedModel"] != "provider-model-a-20260710" {
		t.Fatalf("raw model row was not preserved: %#v", model.Raw)
	}
}

// TestParseInitResult_ModelsEmptyArray verifies empty models array is handled.
func TestParseInitResult_ModelsEmptyArray(t *testing.T) {
	t.Parallel()
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
	// Contract: a present "models" array (even empty) yields a non-nil slice, so
	// the field is set to [] rather than left nil.
	if result.Models == nil {
		t.Error("expected non-nil empty Models slice when the key is present, got nil")
	}
}

// TestParseInitResult_ModelsInvalidType verifies graceful handling of unexpected type.
func TestParseInitResult_ModelsInvalidType(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"models": "not-an-array",
	}

	result := parseInitResult(raw)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Contract: a non-array "models" value leaves the field untouched (nil), not
	// an empty slice — distinguishing absent/invalid from present-but-empty.
	if result.Models != nil {
		t.Errorf("expected nil Models for invalid type, got %#v", result.Models)
	}
}

// TestParseInitResult_ModelsMissingFields verifies partial model entries are still parsed.
func TestParseInitResult_ModelsMissingFields(t *testing.T) {
	t.Parallel()
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

// TestParseInitResult_ModelsKeepsCLIDefaultRow verifies CLI default rows can have
// no concrete value and must still survive catalog discovery.
func TestParseInitResult_ModelsKeepsCLIDefaultRow(t *testing.T) {
	t.Parallel()
	raw := map[string]interface{}{
		"models": []interface{}{
			map[string]interface{}{
				"value":         "",
				"displayName":   "Default",
				"resolvedModel": "claude-sonnet-5-20260701",
				"futureFlag":    "future-value",
			},
			map[string]interface{}{
				// truly empty rows are ignored
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
	if len(result.Models) != 2 {
		t.Fatalf("expected 2 models (keeping CLI default row), got %d", len(result.Models))
	}
	if result.Models[0].Value != "" || result.Models[0].DisplayName != "Default" {
		t.Fatalf("default row = %+v", result.Models[0])
	}
	if result.Models[0].ResolvedModel != "claude-sonnet-5-20260701" {
		t.Fatalf("default row resolved model = %q", result.Models[0].ResolvedModel)
	}
	if result.Models[0].Raw["futureFlag"] != "future-value" {
		t.Fatalf("future raw field not preserved: %#v", result.Models[0].Raw)
	}
	if result.Models[1].Value != "haiku" {
		t.Errorf("expected 'haiku', got %q", result.Models[1].Value)
	}
}

// TestSupportedModels_ReturnsFromInitResult verifies SupportedModels uses the stored init result.
func TestSupportedModels_ReturnsFromInitResult(t *testing.T) {
	t.Parallel()
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
