package transport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// subprocess_cli_args_oracle_test.go locks the FULL cross-flag argument
// sequence emitted by buildCommandArgs. The ~20 isolated TestBuildCommandArgs_*
// tests verify each flag group in isolation; this golden verifies the order
// ACROSS groups, which the upcoming group-extraction refactor of buildCommandArgs
// (cognitive 106 -> ordered append* helpers) must preserve byte-for-byte.
//
// Regenerate with UPDATE_ARGS_ORACLE=1. Committed green on the monolithic
// buildCommandArgs first.

func buildOracleArgs(t *testing.T) []string {
	t.Helper()
	opts := types.NewClaudeAgentOptions()
	opts.WithPermissionPromptToolName("approve-tool")
	opts.WithSystemPromptString("be concise")
	opts.WithModel("claude-opus-4")
	opts.WithForkSession(true)
	// Set the field directly — the WithMaxThinkingTokens builder is deprecated,
	// but buildCommandArgs still emits --max-thinking-tokens for a non-nil field,
	// and the oracle must cover that branch.
	maxThinkingTokens := 2048
	opts.MaxThinkingTokens = &maxThinkingTokens
	opts.WithMaxBudgetUSD(7.5)
	opts.WithBeta("beta-one")
	opts.WithBeta("beta-two")
	opts.WithEffort(types.EffortHigh)
	opts.WithFallbackModel("claude-haiku")
	opts.WithSessionID("sess-xyz")
	opts.WithDebugFile("/tmp/dbg.log")
	opts.WithStrictMcpConfig(true)

	transport := newTestTransport(t, opts)
	// Exercise the capability-gated branches in their "supported" state so the
	// order of those flags is captured too.
	transport.thinkingDisplaySupported = true
	transport.subagentExecutionSupported = true
	transport.agentProgressSummariesSupported = true
	return transport.buildCommandArgs()
}

func TestBuildCommandArgs_FullSequenceOracle(t *testing.T) {
	t.Parallel()

	got := buildOracleArgs(t)
	goldenPath := filepath.Join("testdata", "buildcommandargs_oracle.golden.json")

	if os.Getenv("UPDATE_ARGS_ORACLE") == "1" {
		buf, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			t.Fatalf("marshal golden: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, append(buf, '\n'), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated golden: %s (%d args)", goldenPath, len(got))
		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (regenerate with UPDATE_ARGS_ORACLE=1): %v", err)
	}
	var want []string
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("arg count drift: got %d, want %d\n got:  %v\n want: %v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg[%d] drift: got %q, want %q", i, got[i], want[i])
		}
	}
}
