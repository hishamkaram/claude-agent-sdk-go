package transport

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestParsePermissionModesFromHelp(t *testing.T) {
	t.Parallel()

	help := `Usage: claude [options]
  --permission-mode <mode>              Permission mode to use for the session
                                        (choices: "acceptEdits", "auto",
                                        "bypassPermissions", "default",
                                        "dontAsk", "plan")
  --plugin-dir <path>                   Load a plugin from a directory
`

	got := ParsePermissionModesFromHelp(help)
	want := []string{"acceptEdits", "auto", "bypassPermissions", "default", "dontAsk", "plan"}
	if len(got) != len(want) {
		t.Fatalf("ParsePermissionModesFromHelp() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ParsePermissionModesFromHelp()[%d] = %q, want %q (all = %v)", i, got[i], want[i], got)
		}
	}
}

func TestDiscoverPermissionModesCanonicalizesManualAlias(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cliPath := filepath.Join(dir, "claude")
	writeFakeClaude(t, cliPath, "2.1.206", `"acceptEdits", "auto", "manual", "plan"`)

	got := DiscoverPermissionModes(context.Background(), cliPath)
	for _, mode := range got {
		if mode.ProviderValue != "manual" {
			continue
		}
		if mode.CanonicalValue != types.PermissionModeDefault {
			t.Fatalf("manual CanonicalValue = %q, want %q", mode.CanonicalValue, types.PermissionModeDefault)
		}
		return
	}
	t.Fatalf("DiscoverPermissionModes() = %v, want raw manual mode", got)
}

func TestDiscoverPermissionModesReturnsEmptyWhenHelpFails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cliPath := filepath.Join(dir, "claude")
	writeFakeClaudeFailingHelp(t, cliPath, "2.1.197")
	if got := DiscoverPermissionModes(context.Background(), cliPath); len(got) != 0 {
		t.Fatalf("DiscoverPermissionModes() = %v, want empty unavailable result", got)
	}
}

func TestDiscoverPermissionModesCacheInvalidatesOnVersionChange(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cliPath := filepath.Join(dir, "claude")
	writeFakeClaude(t, cliPath, "2.1.197", `"default", "plan", "auto"`)

	ctx := context.Background()
	first := DiscoverPermissionModes(ctx, cliPath)
	if !containsProviderMode(first, "auto") {
		t.Fatalf("DiscoverPermissionModes() first = %v, want auto from help choices", first)
	}

	writeFakeClaude(t, cliPath, "2.1.198", `"default", "plan"`)
	second := DiscoverPermissionModes(ctx, cliPath)
	if containsProviderMode(second, "auto") {
		t.Fatalf("DiscoverPermissionModes() second = %v, want cache invalidated by version change", second)
	}
}

func TestDiscoverPermissionModesDoesNotCacheFallback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cliPath := filepath.Join(dir, "claude")
	writeFakeClaudeFailingHelp(t, cliPath, "2.1.197")

	ctx := context.Background()
	first := DiscoverPermissionModes(ctx, cliPath)
	if len(first) != 0 {
		t.Fatalf("DiscoverPermissionModes() first = %v, want empty unavailable result", first)
	}

	writeFakeClaude(t, cliPath, "2.1.197", `"default", "plan", "auto"`)
	second := DiscoverPermissionModes(ctx, cliPath)
	if !containsProviderMode(second, "auto") {
		t.Fatalf("DiscoverPermissionModes() second = %v, want retry to discover auto", second)
	}
}

func TestDiscoverPermissionModesCacheIncludesInheritedEnvironment(t *testing.T) {
	dir := t.TempDir()
	cliPath := filepath.Join(dir, "claude")
	body := `#!/bin/sh
if [ "$1" = "--version" ]; then echo '2.9.1 (Claude Code)'; exit 0; fi
if [ "$1" = "--help" ]; then echo "--permission-mode <mode> (choices: \"$PERMISSION_MODE_CHOICE\")"; exit 0; fi
exit 1
`
	if err := os.WriteFile(cliPath, []byte(body), 0o700); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}
	t.Setenv("PERMISSION_MODE_CHOICE", "auto")
	first := DiscoverPermissionModes(context.Background(), cliPath)
	if !containsProviderMode(first, "auto") {
		t.Fatalf("first discovery = %v, want inherited auto", first)
	}
	t.Setenv("PERMISSION_MODE_CHOICE", "plan")
	second := DiscoverPermissionModes(context.Background(), cliPath)
	if !containsProviderMode(second, "plan") || containsProviderMode(second, "auto") {
		t.Fatalf("second discovery = %v, want cache invalidated for inherited plan", second)
	}
}

func TestDiscoverPermissionModesCacheIncludesEffectiveWorkingDirectory(t *testing.T) {
	originalWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWorkingDirectory); chdirErr != nil {
			t.Errorf("restore working directory: %v", chdirErr)
		}
	})

	root := t.TempDir()
	firstDirectory := filepath.Join(root, "auto")
	secondDirectory := filepath.Join(root, "plan")
	for _, directory := range []string{firstDirectory, secondDirectory} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			t.Fatalf("create working directory: %v", err)
		}
	}
	cliPath := filepath.Join(root, "claude")
	body := `#!/bin/sh
if [ "$1" = "--version" ]; then echo '2.9.1 (Claude Code)'; exit 0; fi
if [ "$1" = "--help" ]; then echo "--permission-mode <mode> (choices: \"$(basename "$PWD")\")"; exit 0; fi
exit 1
`
	if err := os.WriteFile(cliPath, []byte(body), 0o700); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}

	if err := os.Chdir(firstDirectory); err != nil {
		t.Fatalf("change to first working directory: %v", err)
	}
	first := DiscoverPermissionModes(context.Background(), cliPath)
	if !containsProviderMode(first, "auto") {
		t.Fatalf("first discovery = %v, want mode from auto working directory", first)
	}
	if err := os.Chdir(secondDirectory); err != nil {
		t.Fatalf("change to second working directory: %v", err)
	}
	second := DiscoverPermissionModes(context.Background(), cliPath)
	if !containsProviderMode(second, "plan") || containsProviderMode(second, "auto") {
		t.Fatalf("second discovery = %v, want cache invalidated for plan working directory", second)
	}
}

func TestBuildEffectiveProcessEnvironmentUsesWindowsKeySemantics(t *testing.T) {
	t.Parallel()

	got := buildEffectiveProcessEnvironmentForOS(
		[]string{"Path=C:\\inherited", "HOME=C:\\home"},
		map[string]string{"PATH": `C:\configured`},
		"windows",
	)

	var pathEntries []string
	for _, entry := range got {
		key, _, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, "PATH") {
			pathEntries = append(pathEntries, entry)
		}
	}
	if len(pathEntries) != 1 || pathEntries[0] != `PATH=C:\configured` {
		t.Fatalf("Windows PATH entries = %v, want configured override only (all = %v)", pathEntries, got)
	}
}

func TestDiscoverPermissionModesPreservesProbeTimeout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cliPath := filepath.Join(dir, "claude")
	body := `#!/bin/sh
if [ "$1" = "--version" ]; then echo '2.9.1 (Claude Code)'; exit 0; fi
sleep 1
`
	if err := os.WriteFile(cliPath, []byte(body), 0o700); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	_, _, err := DiscoverPermissionModesAndVersionWithEnvironment(ctx, cliPath, "", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("discovery error = %v, want deadline exceeded", err)
	}
}

func writeFakeClaude(t *testing.T, path, version, choices string) {
	t.Helper()

	body := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then echo \"" + version + " (Claude Code)\"; exit 0; fi\n" +
		"if [ \"$1\" = \"--help\" ]; then\n" +
		"  echo '  --permission-mode <mode>              Permission mode to use for the session'\n" +
		"  echo '                                        (choices: " + choices + ")'\n" +
		"  echo '  --plugin-dir <path>                   Load a plugin from a directory'\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 1\n"
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}
}

func writeFakeClaudeFailingHelp(t *testing.T, path, version string) {
	t.Helper()

	body := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then echo \"" + version + " (Claude Code)\"; exit 0; fi\n" +
		"if [ \"$1\" = \"--help\" ]; then exit 2; fi\n" +
		"exit 1\n"
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatalf("write fake claude failing help: %v", err)
	}
}

func containsProviderMode(modes []types.SupportedPermissionMode, want string) bool {
	for _, mode := range modes {
		if mode.ProviderValue == want {
			return true
		}
	}
	return false
}
