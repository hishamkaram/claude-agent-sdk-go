//go:build integration
// +build integration

// Real-CLI flag-drift detector. The SDK emits roughly 30 distinct CLI
// flags via buildCommandArgs() in internal/transport/subprocess_cli.go.
// SDK-side unit tests (internal/transport/subprocess_cli_test.go)
// already verify the correct argv shape — this file verifies the OTHER
// half of the contract: that every flag the SDK emits is still accepted
// by the real CLI binary.
//
// Implementation: spawns `claude --help`, greps the output for each
// flag name, and reports any missing flag as a likely upstream rename.
// Hidden flags are documented separately and excluded from the help check.
//
// No auth is required — `claude --help` runs without a credential.

package tests

import (
	"bytes"
	"context"
	"os/exec"
	"sort"
	"strings"
	"testing"
	"time"
)

// sdkEmittedFlags enumerates every CLI flag the SDK's buildCommandArgs()
// emits. Keep in sync with internal/transport/subprocess_cli.go (~lines
// 607-899). If a new flag is added to that function, append it here.
var sdkEmittedFlags = []string{
	"--permission-prompt-tool",
	"--permission-mode",
	"--system-prompt",
	"--append-system-prompt",
	"--model",
	"--resume",
	"--resume-session-at",
	"--fork-session",
	"--allow-dangerously-skip-permissions",
	"--dangerously-skip-permissions",
	"--max-thinking-tokens",
	"--max-budget-usd",
	"--betas",
	"--plugin-dir",
	"--setting-sources",
	"--agents",
	"--agent",
	"--effort",
	"--fallback-model",
	"--session-id",
	"--no-session-persistence",
	"--json-schema",
	"--settings",
	"--replay-user-messages",
	"--subagent-execution",
	"--tools",
	"--debug-file",
	"--strict-mcp-config",
	"--task-budget",
	"--agent-progress-summaries",
}

// hiddenFlags are SDK-emitted flags that are intentionally absent from
// the CLI's `--help` output and therefore cannot be classified by the
// help-output check alone.
//
// Verified 2026-05-07 against Claude Code CLI 2.1.132 with `claude --help`
// plus targeted SDK transport tests. If a hidden flag starts failing at
// runtime, re-run a targeted CLI invocation or binary substring check.
//
// Re-run the substring check whenever the CLI is upgraded:
//
//	strings $(npm root -g)/@anthropic-ai/claude-code/node_modules/\
//	  @anthropic-ai/claude-code-linux-x64/claude | \
//	  grep -cF <flag-name>
var hiddenFlags = []string{
	"--resume-session-at",
	"--max-thinking-tokens",
	"--permission-prompt-tool",
	"--task-budget",
}

// unsupportedFlags are SDK-emitted flags that the current CLI parser rejects.
// The SDK still emits them when specific options are set (SubagentExecution,
// AgentProgressSummaries), but the CLI rejects them at Connect time as unknown
// flags.
//
// Verified 2026-05-07 against Claude Code CLI 2.1.132.
//
// Resolution options (maintainer decision — out of scope for this file):
//  1. Keep the emitting branch for forward-compatible callers until the CLI
//     accepts or permanently removes the feature.
//  2. If the feature comes back under a new name, update the emit
//     branch to the new flag and move the entry to hiddenFlags.
//  3. If the feature is permanently removed upstream, drop the emitting branch
//     in internal/transport/subprocess_cli.go and the corresponding option
//     field in types/options.go.
//
// The test below tolerates these (skips them from the help check) but
// TestFlags_UnsupportedFlagsAreDocumented surfaces them so a future
// cross-check session cannot lose the signal.
var unsupportedFlags = []string{
	"--agent-progress-summaries", // types/options.go: AgentProgressSummaries bool
	"--subagent-execution",       // types/options.go: SubagentExecution
}

// TestFlags_AllEmittedFlagsAcceptedByCLI spawns `claude --help` and asserts
// every non-hidden flag emitted by the SDK appears in the help output. A
// missing flag indicates the CLI renamed or removed it — the SDK must be
// updated or the flag dropped from buildCommandArgs() to avoid runtime
// "unknown flag" errors.
func TestFlags_AllEmittedFlagsAcceptedByCLI(t *testing.T) {
	cliPath := requireClaude(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cliPath, "--help")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("claude --help: %v (stderr: %s)", err, stderr.String())
	}

	helpText := stdout.String() + stderr.String()

	// Build the skip set: hidden flags AND unsupported flags are both
	// permitted to be absent from --help. (Unsupported flags are
	// separately surfaced by TestFlags_UnsupportedFlagsAreDocumented.)
	skipSet := make(map[string]bool, len(hiddenFlags)+len(unsupportedFlags))
	for _, f := range hiddenFlags {
		skipSet[f] = true
	}
	for _, f := range unsupportedFlags {
		skipSet[f] = true
	}

	var missing []string
	for _, flag := range sdkEmittedFlags {
		if skipSet[flag] {
			continue
		}
		if !strings.Contains(helpText, flag) {
			missing = append(missing, flag)
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("claude --help does not mention %d SDK-emitted flag(s): %v\n\n"+
			"Possible causes:\n"+
			"  1. CLI upstream renamed the flag — update internal/transport/subprocess_cli.go\n"+
			"  2. Flag became hidden — run the substring check documented above;\n"+
			"     if the flag IS in the binary, add to hiddenFlags in this file;\n"+
			"     if NOT, add to unsupportedFlags and file an SDK cleanup.\n"+
			"  3. Flag was removed — drop from buildCommandArgs()\n", len(missing), missing)
	}
}

// TestFlags_UnsupportedFlagsAreDocumented is a running reminder that the
// SDK still emits flags the CLI does not accept in the currently verified CLI
// release. Every entry in unsupportedFlags needs a maintainer decision (see
// the rationale block above the variable). The test always passes — its
// purpose is to surface the count in CI output so the gap stays visible.
func TestFlags_UnsupportedFlagsAreDocumented(t *testing.T) {
	if len(unsupportedFlags) == 0 {
		t.Log("unsupportedFlags is empty — the SDK no longer emits any flag the CLI rejects")
		return
	}
	t.Logf("SDK still emits %d flag(s) the CLI does not support: %v", len(unsupportedFlags), unsupportedFlags)
	t.Log("Each needs a maintainer decision — see the unsupportedFlags rationale block")
}

// TestFlags_HiddenFlagsDocumented ensures each entry in hiddenFlags is
// actually emitted by the SDK. A hidden flag that the SDK no longer
// emits is dead documentation — clean it up here.
func TestFlags_HiddenFlagsDocumented(t *testing.T) {
	emittedSet := make(map[string]bool, len(sdkEmittedFlags))
	for _, f := range sdkEmittedFlags {
		emittedSet[f] = true
	}

	for _, f := range hiddenFlags {
		if !emittedSet[f] {
			t.Errorf("hidden flag %q is not listed in sdkEmittedFlags", f)
		}
	}

	// Sanity: hiddenFlags must have at least one entry or the
	// "feature 142 / --resume-session-at" rationale is bit-rotting.
	if len(hiddenFlags) == 0 {
		t.Log("hiddenFlags is empty — if you removed --resume-session-at, update the rationale comment above")
	}
}
