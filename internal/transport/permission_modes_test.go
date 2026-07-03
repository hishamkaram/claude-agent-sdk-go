package transport

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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

func TestFallbackPermissionModesExcludeUnsafeModes(t *testing.T) {
	t.Parallel()

	got := FallbackPermissionModes("2.1.197")
	want := []string{
		string(types.PermissionModeDefault),
		string(types.PermissionModePlan),
		string(types.PermissionModeAcceptEdits),
	}
	if len(got) != len(want) {
		t.Fatalf("FallbackPermissionModes() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i].ProviderValue != want[i] {
			t.Fatalf("FallbackPermissionModes()[%d].ProviderValue = %q, want %q", i, got[i].ProviderValue, want[i])
		}
		if got[i].Source != types.PermissionModeSourceFallback {
			t.Fatalf("FallbackPermissionModes()[%d].Source = %q, want %q", i, got[i].Source, types.PermissionModeSourceFallback)
		}
	}
	for _, mode := range got {
		switch mode.ProviderValue {
		case string(types.PermissionModeAuto), string(types.PermissionModeBypassPermissions), string(types.PermissionModeDontAsk):
			t.Fatalf("FallbackPermissionModes() exposed unsafe mode %q", mode.ProviderValue)
		}
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
	if containsProviderMode(first, "auto") {
		t.Fatalf("DiscoverPermissionModes() first = %v, want fallback without auto", first)
	}
	for _, mode := range first {
		if mode.Source != types.PermissionModeSourceFallback {
			t.Fatalf("DiscoverPermissionModes() first source = %q, want fallback (all = %v)", mode.Source, first)
		}
	}

	writeFakeClaude(t, cliPath, "2.1.197", `"default", "plan", "auto"`)
	second := DiscoverPermissionModes(ctx, cliPath)
	if !containsProviderMode(second, "auto") {
		t.Fatalf("DiscoverPermissionModes() second = %v, want retry to discover auto", second)
	}
	for _, mode := range second {
		if mode.Source == types.PermissionModeSourceFallback {
			t.Fatalf("DiscoverPermissionModes() second still returned fallback mode %v", mode)
		}
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
