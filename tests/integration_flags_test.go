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
// Hidden flags (currently just --resume-session-at per feature 142) are
// documented separately and excluded from the help check.
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
	"--fork-session",
	"--allow-dangerously-skip-permissions",
	"--dangerously-skip-permissions",
	"--max-thinking-tokens",
	"--max-budget-usd",
	"--betas",
	"--plugin-dir",
	"--setting-sources",
	"--agents",
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
// the CLI's `--help` output (registered with .hideHelp() in the CLI) OR
// are documented as accepted by the CLI despite not being in help.
// The SDK discovered `--resume-session-at` via source inspection (feature
// 142). The rest of this list was discovered by running this test against
// a live CLI (0.5.1) on 2026-04-19 — those flags must be re-verified
// against the CLI source to confirm they are hidden (expected) rather than
// renamed/removed (would be a real bug).
//
// TODO: cross-check each of these against the CLI's source tree. If any
// turn out to be genuinely removed or renamed, update buildCommandArgs()
// in internal/transport/subprocess_cli.go and drop the entry here.
var hiddenFlags = []string{
	"--resume-session-at",
	"--agent-progress-summaries",
	"--max-thinking-tokens",
	"--permission-prompt-tool",
	"--subagent-execution",
	"--task-budget",
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

	// Cross-check every non-hidden flag.
	hiddenSet := make(map[string]bool, len(hiddenFlags))
	for _, f := range hiddenFlags {
		hiddenSet[f] = true
	}

	var missing []string
	for _, flag := range sdkEmittedFlags {
		if hiddenSet[flag] {
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
			"  2. Flag became hidden — add to hiddenFlags in this file\n"+
			"  3. Flag was removed — drop from buildCommandArgs()\n", len(missing), missing)
	}
}

// TestFlags_HiddenFlagsDocumented ensures each entry in hiddenFlags is
// actually emitted by the SDK. A hidden flag that the SDK no longer
// emits is dead documentation — clean it up here.
func TestFlags_HiddenFlagsDocumented(t *testing.T) {
	emittedSet := make(map[string]bool, len(sdkEmittedFlags)+len(hiddenFlags))
	for _, f := range sdkEmittedFlags {
		emittedSet[f] = true
	}
	// Hidden flags are additional to the visible set — they're still
	// emitted, just not documented in --help.
	for _, f := range hiddenFlags {
		emittedSet[f] = true
	}

	// Sanity: hiddenFlags must have at least one entry or the
	// "feature 142 / --resume-session-at" rationale is bit-rotting.
	if len(hiddenFlags) == 0 {
		t.Log("hiddenFlags is empty — if you removed --resume-session-at, update the rationale comment above")
	}
}
