package types

import (
	"testing"
)

// TestPluginConfig tests PluginConfig type and validation.
func TestPluginConfig(t *testing.T) {
	t.Parallel()
	t.Run("NewLocalPluginConfig", func(t *testing.T) {
		plugin := NewLocalPluginConfig("/path/to/plugin")
		if plugin.Type != "local" {
			t.Errorf("expected Type 'local', got %s", plugin.Type)
		}
		if plugin.Path != "/path/to/plugin" {
			t.Errorf("expected Path '/path/to/plugin', got %s", plugin.Path)
		}
	})

	t.Run("NewPluginConfig with valid type", func(t *testing.T) {
		plugin, err := NewPluginConfig("local", "/path/to/plugin")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if plugin.Type != "local" {
			t.Errorf("expected Type 'local', got %s", plugin.Type)
		}
		if plugin.Path != "/path/to/plugin" {
			t.Errorf("expected Path '/path/to/plugin', got %s", plugin.Path)
		}
	})

	t.Run("NewPluginConfig with invalid type", func(t *testing.T) {
		_, err := NewPluginConfig("remote", "/path/to/plugin")
		if err == nil {
			t.Error("expected error for unsupported plugin type")
		}
	})

	t.Run("NewPluginConfig with empty path", func(t *testing.T) {
		_, err := NewPluginConfig("local", "")
		if err == nil {
			t.Error("expected error for empty path")
		}
	})
}

// TestClaudeAgentOptions_Plugins tests plugin builder methods.
func TestClaudeAgentOptions_Plugins(t *testing.T) {
	t.Parallel()
	t.Run("WithPlugins", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		plugins := []PluginConfig{
			*NewLocalPluginConfig("/path/to/plugin1"),
			*NewLocalPluginConfig("/path/to/plugin2"),
		}
		opts.WithPlugins(plugins)

		if len(opts.Plugins) != 2 {
			t.Errorf("expected 2 plugins, got %d", len(opts.Plugins))
		}
	})

	t.Run("WithPlugin", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		plugin := *NewLocalPluginConfig("/path/to/plugin")
		opts.WithPlugin(plugin)

		if len(opts.Plugins) != 1 {
			t.Errorf("expected 1 plugin, got %d", len(opts.Plugins))
		}
		if opts.Plugins[0].Path != "/path/to/plugin" {
			t.Errorf("expected Path '/path/to/plugin', got %s", opts.Plugins[0].Path)
		}
	})

	t.Run("WithLocalPlugin", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		opts.WithLocalPlugin("/path/to/plugin")

		if len(opts.Plugins) != 1 {
			t.Errorf("expected 1 plugin, got %d", len(opts.Plugins))
		}
		if opts.Plugins[0].Type != "local" {
			t.Errorf("expected Type 'local', got %s", opts.Plugins[0].Type)
		}
		if opts.Plugins[0].Path != "/path/to/plugin" {
			t.Errorf("expected Path '/path/to/plugin', got %s", opts.Plugins[0].Path)
		}
	})

	t.Run("multiple plugins via WithPlugin", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		opts.WithPlugin(*NewLocalPluginConfig("/path/1")).
			WithPlugin(*NewLocalPluginConfig("/path/2")).
			WithPlugin(*NewLocalPluginConfig("/path/3"))

		if len(opts.Plugins) != 3 {
			t.Errorf("expected 3 plugins, got %d", len(opts.Plugins))
		}
	})

	t.Run("multiple plugins via WithLocalPlugin chaining", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		opts.WithLocalPlugin("/path/1").
			WithLocalPlugin("/path/2").
			WithLocalPlugin("/path/3")

		if len(opts.Plugins) != 3 {
			t.Errorf("expected 3 plugins, got %d", len(opts.Plugins))
		}

		// Verify paths
		expectedPaths := []string{"/path/1", "/path/2", "/path/3"}
		for i, plugin := range opts.Plugins {
			if plugin.Path != expectedPaths[i] {
				t.Errorf("plugin[%d].Path = %s, want %s", i, plugin.Path, expectedPaths[i])
			}
		}
	})

	t.Run("empty plugins by default", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		if opts.Plugins == nil {
			t.Error("Plugins should not be nil")
		}
		if len(opts.Plugins) != 0 {
			t.Errorf("expected 0 plugins by default, got %d", len(opts.Plugins))
		}
	})
}

// TestWithBetas tests the WithBetas builder method.
func TestWithBetas(t *testing.T) {
	t.Parallel()
	t.Run("WithBetas sets multiple beta flags", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		betas := []string{"context-1m-2025-08-07"}

		result := opts.WithBetas(betas)

		// Verify the method returns the same instance for chaining
		if result != opts {
			t.Error("WithBetas should return the same instance for chaining")
		}

		// Verify the values are set correctly
		if len(opts.Betas) != 1 {
			t.Errorf("expected 1 beta, got %d", len(opts.Betas))
		}

		if opts.Betas[0] != "context-1m-2025-08-07" {
			t.Errorf("expected beta 'context-1m-2025-08-07', got %s", opts.Betas[0])
		}
	})

	t.Run("WithBetas empty list", func(t *testing.T) {
		opts := NewClaudeAgentOptions().WithBetas([]string{})

		if len(opts.Betas) != 0 {
			t.Errorf("expected 0 betas, got %d", len(opts.Betas))
		}
	})

	t.Run("WithBetas replaces existing betas", func(t *testing.T) {
		opts := NewClaudeAgentOptions().
			WithBeta("beta-1").
			WithBeta("beta-2").
			WithBetas([]string{"beta-3", "beta-4"})

		if len(opts.Betas) != 2 {
			t.Errorf("expected 2 betas after WithBetas, got %d", len(opts.Betas))
		}

		if opts.Betas[0] != "beta-3" || opts.Betas[1] != "beta-4" {
			t.Errorf("expected betas [beta-3, beta-4], got %v", opts.Betas)
		}
	})
}

// TestWithBeta tests the WithBeta builder method.
func TestWithBeta(t *testing.T) {
	t.Parallel()
	t.Run("WithBeta adds single beta flag", func(t *testing.T) {
		opts := NewClaudeAgentOptions()

		result := opts.WithBeta("context-1m-2025-08-07")

		// Verify the method returns the same instance for chaining
		if result != opts {
			t.Error("WithBeta should return the same instance for chaining")
		}

		// Verify the value is set correctly
		if len(opts.Betas) != 1 {
			t.Errorf("expected 1 beta, got %d", len(opts.Betas))
		}

		if opts.Betas[0] != "context-1m-2025-08-07" {
			t.Errorf("expected beta 'context-1m-2025-08-07', got %s", opts.Betas[0])
		}
	})

	t.Run("WithBeta multiple calls accumulate", func(t *testing.T) {
		opts := NewClaudeAgentOptions().
			WithBeta("beta-1").
			WithBeta("beta-2").
			WithBeta("beta-3")

		if len(opts.Betas) != 3 {
			t.Errorf("expected 3 betas, got %d", len(opts.Betas))
		}

		expectedBetas := []string{"beta-1", "beta-2", "beta-3"}
		for i, beta := range opts.Betas {
			if beta != expectedBetas[i] {
				t.Errorf("beta[%d] = %s, expected %s", i, beta, expectedBetas[i])
			}
		}
	})

	t.Run("empty betas by default", func(t *testing.T) {
		opts := NewClaudeAgentOptions()
		if opts.Betas == nil {
			t.Error("Betas should not be nil")
		}
		if len(opts.Betas) != 0 {
			t.Errorf("expected 0 betas by default, got %d", len(opts.Betas))
		}
	})
}
