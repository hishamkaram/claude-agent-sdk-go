package transport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestFindCLI_InPATH tests that FindCLI discovers a CLI binary placed in PATH.
func TestFindCLI_InPATH(t *testing.T) {
	// Cannot use t.Parallel() because t.Setenv modifies process state.

	// Skip version checking so a fake binary works.
	t.Setenv("CLAUDE_AGENT_SDK_SKIP_VERSION_CHECK", "1")

	// Create a temporary directory with a mock claude binary.
	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "claude")
	if err := os.WriteFile(claudePath, []byte("#!/bin/sh\necho mock"), 0755); err != nil {
		t.Fatalf("failed to create mock binary: %v", err)
	}

	// Set PATH so only our temp dir is searched.
	t.Setenv("PATH", tmpDir)

	got, err := FindCLI()
	if err != nil {
		t.Fatalf("FindCLI() unexpected error: %v", err)
	}

	if got == "" {
		t.Fatal("FindCLI() returned empty path")
	}

	if got != claudePath {
		t.Errorf("FindCLI() = %q, want %q", got, claudePath)
	}
}

// Note: Testing FindCLI's common-location fallback (e.g. ~/.claude/local/claude)
// is not feasible in isolation because expandHome() uses os/user.Current() which
// reads from /etc/passwd and cannot be overridden via environment variables.
// The existing TestFindCLI in transport_test.go covers the PATH-based discovery.
// The common-location code path is covered by TestExpandHome and TestExpandHome_EdgeCases.

// TestFindCLI_NotFound tests that FindCLI returns a CLINotFoundError when
// the CLI binary is not in PATH or any common location.
func TestFindCLI_NotFound(t *testing.T) {
	// Cannot use t.Parallel() because t.Setenv modifies process state.

	t.Setenv("CLAUDE_AGENT_SDK_SKIP_VERSION_CHECK", "1")

	// Set PATH to an empty temp dir so exec.LookPath finds nothing.
	t.Setenv("PATH", t.TempDir())

	// Set HOME to an empty temp dir so common-location stat calls all fail.
	t.Setenv("HOME", t.TempDir())

	_, err := FindCLI()
	if err == nil {
		t.Fatal("FindCLI() expected error when CLI is not installed, got nil")
	}

	if !types.IsCLINotFoundError(err) {
		t.Errorf("FindCLI() error type = %T, want *types.CLINotFoundError", err)
	}
}

// TestExpandHome_EdgeCases covers paths that are not covered by the existing
// TestExpandHome in transport_test.go.
func TestExpandHome_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string // expected behavior description; exact value checked below
	}{
		{
			name:  "tilde followed by non-slash character is unchanged",
			input: "~username/dir",
			want:  "~username/dir",
		},
		{
			name:  "absolute path is unchanged",
			input: "/usr/local/bin/claude",
			want:  "/usr/local/bin/claude",
		},
		{
			name:  "empty string is unchanged",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := expandHome(tt.input)
			if got != tt.want {
				t.Errorf("expandHome(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
