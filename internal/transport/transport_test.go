package transport

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestFindCLI tests CLI discovery in various scenarios
func TestFindCLI(t *testing.T) {
	// Cannot use t.Parallel() because t.Setenv is used below.
	resetFindCLICacheForTest(t)

	// Disable version checking for these tests since we're using mock binaries
	t.Setenv("CLAUDE_AGENT_SDK_SKIP_VERSION_CHECK", "1")

	tests := []struct {
		name      string
		setup     func() func() // Returns cleanup function
		wantError bool
	}{
		{
			name: "CLI in PATH",
			setup: func() func() {
				// Save original PATH
				origPath := os.Getenv("PATH")

				// Create temporary directory with a mock claude binary
				tmpDir := t.TempDir()
				claudePath := filepath.Join(tmpDir, "claude")

				// Create mock binary
				f, err := os.Create(claudePath)
				if err != nil {
					t.Fatalf("Failed to create mock binary: %v", err)
				}
				_ = f.Close()

				// Make it executable
				if err := os.Chmod(claudePath, 0o755); err != nil {
					t.Fatalf("Failed to chmod mock binary: %v", err)
				}

				// Add to PATH
				_ = os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+origPath)

				// Return cleanup function
				return func() {
					_ = os.Setenv("PATH", origPath)
				}
			},
			wantError: false,
		},
		// Note: "CLI not found" test is skipped because it's environment-dependent
		// If Claude CLI is installed in standard locations (like ~/.local/bin/claude),
		// it will be found even when PATH/HOME are cleared since FindCLI checks
		// hardcoded paths using user.Current(). This is actually desired behavior.
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			path, err := FindCLI(context.Background())

			if tt.wantError {
				if err == nil {
					t.Errorf("FindCLI(context.Background()) expected error, got nil (found path: %s, PATH=%s, HOME=%s)", path, os.Getenv("PATH"), os.Getenv("HOME"))
				}
				var cliNotFoundErr *types.CLINotFoundError
				if err != nil && !types.IsCLINotFoundError(err) {
					t.Errorf("FindCLI(context.Background()) error type = %T, want *types.CLINotFoundError", err)
				}
				_ = cliNotFoundErr
			} else {
				if err != nil {
					t.Errorf("FindCLI(context.Background()) unexpected error: %v", err)
				}
				if path == "" {
					t.Errorf("FindCLI(context.Background()) returned empty path")
				}
			}
		})
	}
}

// TestExpandHome tests home directory expansion
func TestExpandHome(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "tilde only",
			input: "~",
			want:  "HOME_DIR", // Will be replaced in test
		},
		{
			name:  "tilde with path",
			input: "~/.config/claude",
			want:  "HOME_DIR/.config/claude",
		},
		{
			name:  "no tilde",
			input: "/usr/local/bin/claude",
			want:  "/usr/local/bin/claude",
		},
		{
			name:  "relative path",
			input: "./bin/claude",
			want:  "./bin/claude",
		},
	}

	// Get actual home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Replace placeholder with actual home dir
			want := strings.ReplaceAll(tt.want, "HOME_DIR", homeDir)

			got := expandHome(tt.input)
			if got != want {
				t.Errorf("expandHome(%q) = %q, want %q", tt.input, got, want)
			}
		})
	}
}
