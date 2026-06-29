package transport

import (
	"strings"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestBuildCommandArgs_SystemPrompt tests system prompt handling to match Python SDK behavior
func TestBuildCommandArgs_SystemPrompt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		systemPrompt interface{}
		wantFlag     bool
		wantValue    string
		wantAppend   bool // For preset case
	}{
		{
			name:         "nil system prompt should pass empty string",
			systemPrompt: nil,
			wantFlag:     true,
			wantValue:    "",
		},
		{
			name:         "empty string system prompt",
			systemPrompt: "",
			wantFlag:     true,
			wantValue:    "",
		},
		{
			name:         "custom system prompt",
			systemPrompt: "You are a helpful assistant",
			wantFlag:     true,
			wantValue:    "You are a helpful assistant",
		},
		{
			name:         "multiline system prompt",
			systemPrompt: "You are a helpful assistant.\nAlways be polite.",
			wantFlag:     true,
			wantValue:    "You are a helpful assistant.\nAlways be polite.",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := types.NewClaudeAgentOptions()
			if tt.systemPrompt != nil {
				opts.WithSystemPrompt(tt.systemPrompt)
			}
			// If tt.systemPrompt is explicitly nil in the struct, it will be nil in opts

			logger := log.NewLogger(false)
			transport := NewSubprocessCLITransport(
				"/usr/local/bin/claude",
				"",
				nil,
				logger,
				"",
				opts,
			)

			args := transport.buildCommandArgs()

			// Find --system-prompt flag
			foundFlag := false
			foundValue := ""
			for i, arg := range args {
				if arg == "--system-prompt" && i+1 < len(args) {
					foundFlag = true
					foundValue = args[i+1]
					break
				}
			}

			if foundFlag != tt.wantFlag {
				t.Errorf("--system-prompt flag present = %v, want %v", foundFlag, tt.wantFlag)
			}

			if foundValue != tt.wantValue {
				t.Errorf("--system-prompt value = %q, want %q", foundValue, tt.wantValue)
			}
		})
	}
}

// TestBuildCommandArgs_SystemPromptPreset tests system prompt preset handling
func TestBuildCommandArgs_SystemPromptPreset(t *testing.T) {
	t.Parallel()
	appendText := "Additional instructions here"
	preset := types.SystemPromptPreset{
		Type:   "preset",
		Preset: "claude_code",
		Append: &appendText,
	}

	opts := types.NewClaudeAgentOptions().
		WithSystemPromptPreset(preset)

	logger := log.NewLogger(false)
	transport := NewSubprocessCLITransport(
		"/usr/local/bin/claude",
		"",
		nil,
		logger,
		"",
		opts,
	)

	args := transport.buildCommandArgs()

	// Find --append-system-prompt flag
	foundAppendFlag := false
	foundAppendValue := ""
	for i, arg := range args {
		if arg == "--append-system-prompt" && i+1 < len(args) {
			foundAppendFlag = true
			foundAppendValue = args[i+1]
			break
		}
	}

	if !foundAppendFlag {
		t.Errorf("--append-system-prompt flag not found in args: %v", args)
	}

	if foundAppendValue != appendText {
		t.Errorf("--append-system-prompt value = %q, want %q", foundAppendValue, appendText)
	}

	// Should NOT have --system-prompt flag when using preset
	hasSystemPromptFlag := false
	for _, arg := range args {
		if arg == "--system-prompt" {
			hasSystemPromptFlag = true
			break
		}
	}

	if hasSystemPromptFlag {
		t.Errorf("--system-prompt flag should not be present when using preset, but found in args: %v", args)
	}
}

// TestBuildCommandArgs_NoOptions tests that empty system prompt is used when no options provided
func TestBuildCommandArgs_NoOptions(t *testing.T) {
	t.Parallel()
	logger := log.NewLogger(false)
	transport := NewSubprocessCLITransport(
		"/usr/local/bin/claude",
		"",
		nil,
		logger,
		"",
		nil, // No options
	)

	args := transport.buildCommandArgs()

	// Find --system-prompt flag
	foundFlag := false
	foundValue := ""
	for i, arg := range args {
		if arg == "--system-prompt" && i+1 < len(args) {
			foundFlag = true
			foundValue = args[i+1]
			break
		}
	}

	if !foundFlag {
		t.Errorf("--system-prompt flag should be present even with no options")
	}

	if foundValue != "" {
		t.Errorf("--system-prompt value = %q, want empty string", foundValue)
	}
}

// TestBuildCommandArgs_Plugins tests plugin CLI argument generation
func TestBuildCommandArgs_Plugins(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		plugins   []types.PluginConfig
		wantFlags int // Number of --plugin-dir flags expected
	}{
		{
			name:      "no plugins",
			plugins:   []types.PluginConfig{},
			wantFlags: 0,
		},
		{
			name: "single plugin",
			plugins: []types.PluginConfig{
				*types.NewLocalPluginConfig("/path/to/plugin"),
			},
			wantFlags: 1,
		},
		{
			name: "multiple plugins",
			plugins: []types.PluginConfig{
				*types.NewLocalPluginConfig("/path/to/plugin1"),
				*types.NewLocalPluginConfig("/path/to/plugin2"),
			},
			wantFlags: 2,
		},
		{
			name: "three plugins",
			plugins: []types.PluginConfig{
				*types.NewLocalPluginConfig("./plugins/demo"),
				*types.NewLocalPluginConfig("./plugins/custom"),
				*types.NewLocalPluginConfig("/usr/local/share/claude-plugins/tools"),
			},
			wantFlags: 3,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := types.NewClaudeAgentOptions().WithPlugins(tt.plugins)

			logger := log.NewLogger(false)
			transport := NewSubprocessCLITransport(
				"/usr/local/bin/claude",
				"",
				nil,
				logger,
				"",
				opts,
			)

			args := transport.buildCommandArgs()

			// Count --plugin-dir flags
			count := 0
			pluginDirs := []string{}
			for i, arg := range args {
				if arg == "--plugin-dir" {
					count++
					if i+1 < len(args) {
						pluginDirs = append(pluginDirs, args[i+1])
					}
				}
			}

			if count != tt.wantFlags {
				t.Errorf("expected %d --plugin-dir flags, got %d", tt.wantFlags, count)
			}

			// Verify plugin paths match
			if len(pluginDirs) != len(tt.plugins) {
				t.Errorf("expected %d plugin paths, got %d", len(tt.plugins), len(pluginDirs))
			}

			for i, plugin := range tt.plugins {
				if i >= len(pluginDirs) {
					break
				}
				if pluginDirs[i] != plugin.Path {
					t.Errorf("plugin[%d] path = %s, want %s", i, pluginDirs[i], plugin.Path)
				}
			}
		})
	}
}

// TestBuildCommandArgs_PluginsWithOtherOptions tests plugins work with other options
func TestBuildCommandArgs_PluginsWithOtherOptions(t *testing.T) {
	t.Parallel()
	maxThinkingTokens := 1000
	opts := types.NewClaudeAgentOptions().
		WithLocalPlugin("./my-plugin").
		WithModel("claude-3-5-sonnet-20241022").
		WithSystemPrompt("You are a helpful assistant")
	// Set the legacy max-thinking-tokens field directly (the builder is deprecated)
	// to keep asserting that the --max-thinking-tokens flag is still emitted.
	opts.MaxThinkingTokens = &maxThinkingTokens

	logger := log.NewLogger(false)
	transport := NewSubprocessCLITransport(
		"/usr/local/bin/claude",
		"",
		nil,
		logger,
		"",
		opts,
	)

	args := transport.buildCommandArgs()

	// Verify plugin flag exists
	hasPluginDir := false
	for _, arg := range args {
		if arg == "--plugin-dir" {
			hasPluginDir = true
			break
		}
	}

	if !hasPluginDir {
		t.Error("--plugin-dir flag not found in args")
	}

	// Verify other flags still work
	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "--model") {
		t.Error("--model flag not found")
	}
	if !strings.Contains(argsStr, "--max-thinking-tokens") {
		t.Error("--max-thinking-tokens flag not found")
	}
	if !strings.Contains(argsStr, "--system-prompt") {
		t.Error("--system-prompt flag not found")
	}
}

// TestBuildCommandArgs_Betas tests beta feature flag handling.
func TestBuildCommandArgs_Betas(t *testing.T) {
	t.Parallel()
	t.Run("no betas", func(t *testing.T) {
		opts := types.NewClaudeAgentOptions()

		logger := log.NewLogger(false)
		transport := NewSubprocessCLITransport(
			"/usr/local/bin/claude",
			"",
			nil,
			logger,
			"",
			opts,
		)

		args := transport.buildCommandArgs()
		argsStr := strings.Join(args, " ")

		if strings.Contains(argsStr, "--betas") {
			t.Error("--betas flag should not be present when no betas specified")
		}
	})

	t.Run("single beta", func(t *testing.T) {
		opts := types.NewClaudeAgentOptions().
			WithBeta("context-1m-2025-08-07")

		logger := log.NewLogger(false)
		transport := NewSubprocessCLITransport(
			"/usr/local/bin/claude",
			"",
			nil,
			logger,
			"",
			opts,
		)

		args := transport.buildCommandArgs()

		// Find --betas flag and verify it's followed by the beta value
		hasBetas := false
		for i, arg := range args {
			if arg == "--betas" {
				if i+1 < len(args) && args[i+1] == "context-1m-2025-08-07" {
					hasBetas = true
				}
				break
			}
		}

		if !hasBetas {
			t.Error("--betas flag with correct value not found")
		}
	})

	t.Run("multiple betas", func(t *testing.T) {
		opts := types.NewClaudeAgentOptions().
			WithBeta("context-1m-2025-08-07").
			WithBeta("another-beta-feature")

		logger := log.NewLogger(false)
		transport := NewSubprocessCLITransport(
			"/usr/local/bin/claude",
			"",
			nil,
			logger,
			"",
			opts,
		)

		args := transport.buildCommandArgs()

		// Count how many times --betas appears
		betasCount := 0
		for _, arg := range args {
			if arg == "--betas" {
				betasCount++
			}
		}

		if betasCount != 2 {
			t.Errorf("expected 2 --betas flags, got %d", betasCount)
		}

		// Verify both beta values are present
		argsStr := strings.Join(args, " ")
		if !strings.Contains(argsStr, "context-1m-2025-08-07") {
			t.Error("context-1m-2025-08-07 beta not found in args")
		}
		if !strings.Contains(argsStr, "another-beta-feature") {
			t.Error("another-beta-feature beta not found in args")
		}
	})

	t.Run("betas with other options", func(t *testing.T) {
		maxThinkingTokens := 5000
		opts := types.NewClaudeAgentOptions().
			WithBeta("context-1m-2025-08-07").
			WithModel("claude-3-5-sonnet-20241022")
		// Set the legacy max-thinking-tokens field directly (the builder is
		// deprecated) to keep asserting that --max-thinking-tokens is emitted.
		opts.MaxThinkingTokens = &maxThinkingTokens

		logger := log.NewLogger(false)
		transport := NewSubprocessCLITransport(
			"/usr/local/bin/claude",
			"",
			nil,
			logger,
			"",
			opts,
		)

		args := transport.buildCommandArgs()
		argsStr := strings.Join(args, " ")

		// Verify betas flag
		if !strings.Contains(argsStr, "--betas") {
			t.Error("--betas flag not found")
		}

		// Verify other flags still work
		if !strings.Contains(argsStr, "--model") {
			t.Error("--model flag not found when combined with betas")
		}
		if !strings.Contains(argsStr, "--max-thinking-tokens") {
			t.Error("--max-thinking-tokens flag not found when combined with betas")
		}
	})

	t.Run("WithBetas replaces previous betas", func(t *testing.T) {
		opts := types.NewClaudeAgentOptions().
			WithBeta("beta-1").
			WithBeta("beta-2").
			WithBetas([]string{"beta-3"})

		logger := log.NewLogger(false)
		transport := NewSubprocessCLITransport(
			"/usr/local/bin/claude",
			"",
			nil,
			logger,
			"",
			opts,
		)

		args := transport.buildCommandArgs()

		// Count how many times --betas appears - should be 1
		betasCount := 0
		for _, arg := range args {
			if arg == "--betas" {
				betasCount++
			}
		}

		if betasCount != 1 {
			t.Errorf("expected 1 --betas flag after WithBetas, got %d", betasCount)
		}

		// Verify only the new beta is present
		argsStr := strings.Join(args, " ")
		if !strings.Contains(argsStr, "beta-3") {
			t.Error("beta-3 not found after WithBetas()")
		}
		if strings.Contains(argsStr, "beta-1") || strings.Contains(argsStr, "beta-2") {
			t.Error("old betas should be replaced by WithBetas()")
		}
	})
}
