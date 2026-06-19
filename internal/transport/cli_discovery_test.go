package transport

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func resetFindCLICacheForTest(t *testing.T) {
	t.Helper()
	resetFindCLICache()
	t.Cleanup(resetFindCLICache)
}

func resetFindCLICache() {
	findCLIState.mu.Lock()
	defer findCLIState.mu.Unlock()

	findCLIState.cachedPath = ""
	findCLIState.inFlight = nil
}

func setCommonCLIInstallLocationsForTest(t *testing.T, locations []string) {
	t.Helper()

	original := commonCLIInstallLocations
	commonCLIInstallLocations = locations
	t.Cleanup(func() {
		commonCLIInstallLocations = original
	})
}

// TestFindCLI_InPATH tests that FindCLI discovers a CLI binary placed in PATH.
func TestFindCLI_InPATH(t *testing.T) {
	// Cannot use t.Parallel() because t.Setenv modifies process state.
	resetFindCLICacheForTest(t)

	// Skip version checking so a fake binary works.
	t.Setenv("CLAUDE_AGENT_SDK_SKIP_VERSION_CHECK", "1")

	// Create a temporary directory with a mock claude binary.
	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "claude")
	if err := os.WriteFile(claudePath, []byte("#!/bin/sh\necho mock"), 0o755); err != nil {
		t.Fatalf("failed to create mock binary: %v", err)
	}

	// Set PATH so only our temp dir is searched.
	t.Setenv("PATH", tmpDir)

	got, err := FindCLI(context.Background())
	if err != nil {
		t.Fatalf("FindCLI(context.Background()) unexpected error: %v", err)
	}

	if got == "" {
		t.Fatal("FindCLI(context.Background()) returned empty path")
	}

	if got != claudePath {
		t.Errorf("FindCLI(context.Background()) = %q, want %q", got, claudePath)
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
	resetFindCLICacheForTest(t)
	setCommonCLIInstallLocationsForTest(t, nil)

	t.Setenv("CLAUDE_AGENT_SDK_SKIP_VERSION_CHECK", "1")

	// Set PATH to an empty temp dir so exec.LookPath finds nothing.
	t.Setenv("PATH", t.TempDir())

	// Set HOME to an empty temp dir so common-location stat calls all fail.
	t.Setenv("HOME", t.TempDir())

	_, err := FindCLI(context.Background())
	if err == nil {
		t.Fatal("FindCLI(context.Background()) expected error when CLI is not installed, got nil")
	}

	if !types.IsCLINotFoundError(err) {
		t.Errorf("FindCLI(context.Background()) error type = %T, want *types.CLINotFoundError", err)
	}
}

func TestFindCLI_ConcurrentCallsShareDiscovery(t *testing.T) {
	// Cannot use t.Parallel() because t.Setenv modifies process state.
	resetFindCLICacheForTest(t)

	tmpDir := t.TempDir()
	countPath := filepath.Join(tmpDir, "version-count")
	t.Setenv("VERSION_COUNT_FILE", countPath)

	claudePath := filepath.Join(tmpDir, "claude")
	script := `#!/bin/sh
if [ "$1" = "--version" ]; then
  /bin/sleep 0.1
  printf x >> "$VERSION_COUNT_FILE"
  printf '2.0.0\n'
  exit 0
fi
exit 0
`
	if err := os.WriteFile(claudePath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to create mock binary: %v", err)
	}

	t.Setenv("PATH", tmpDir)

	const callers = 64
	type result struct {
		path string
		err  error
	}
	results := make([]result, callers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		i := i
		go func() {
			defer wg.Done()
			<-start
			results[i].path, results[i].err = FindCLI(context.Background())
		}()
	}

	close(start)
	wg.Wait()

	for i, result := range results {
		if result.err != nil {
			t.Fatalf("FindCLI(context.Background()) caller %d unexpected error: %v", i, result.err)
		}
		if result.path != claudePath {
			t.Fatalf("FindCLI(context.Background()) caller %d = %q, want %q", i, result.path, claudePath)
		}
	}

	data, err := os.ReadFile(countPath)
	if err != nil {
		t.Fatalf("failed to read version count: %v", err)
	}
	if got := len(data); got != 1 {
		t.Fatalf("version check count after concurrent calls = %d, want 1", got)
	}

	got, err := FindCLI(context.Background())
	if err != nil {
		t.Fatalf("FindCLI(context.Background()) after cache unexpected error: %v", err)
	}
	if got != claudePath {
		t.Fatalf("FindCLI(context.Background()) after cache = %q, want %q", got, claudePath)
	}

	data, err = os.ReadFile(countPath)
	if err != nil {
		t.Fatalf("failed to read version count after cache hit: %v", err)
	}
	if got := len(data); got != 1 {
		t.Fatalf("version check count after cache hit = %d, want 1", got)
	}
}

func TestFindCLI_FailedLookupIsNotCached(t *testing.T) {
	// Cannot use t.Parallel() because t.Setenv modifies process state.
	resetFindCLICacheForTest(t)
	setCommonCLIInstallLocationsForTest(t, nil)

	t.Setenv("CLAUDE_AGENT_SDK_SKIP_VERSION_CHECK", "1")
	t.Setenv("PATH", t.TempDir())

	_, err := FindCLI(context.Background())
	if err == nil {
		t.Fatal("FindCLI(context.Background()) expected initial lookup error, got nil")
	}
	if !types.IsCLINotFoundError(err) {
		t.Fatalf("FindCLI(context.Background()) initial error type = %T, want *types.CLINotFoundError", err)
	}

	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "claude")
	if writeErr := os.WriteFile(claudePath, []byte("#!/bin/sh\necho mock"), 0o755); writeErr != nil {
		t.Fatalf("failed to create mock binary: %v", writeErr)
	}

	t.Setenv("PATH", tmpDir)

	got, err := FindCLI(context.Background())
	if err != nil {
		t.Fatalf("FindCLI(context.Background()) after PATH update unexpected error: %v", err)
	}
	if got != claudePath {
		t.Fatalf("FindCLI(context.Background()) after PATH update = %q, want %q", got, claudePath)
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
