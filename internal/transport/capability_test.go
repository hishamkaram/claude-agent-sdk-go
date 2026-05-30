package transport

import (
	"slices"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestCapabilityFunctions_AbsentFlags pins the binary-verified fact that no released
// CLI accepts these experimental flags (2.1.158). When a flag ships, its gate changes
// and this test is updated alongside it.
func TestCapabilityFunctions_AbsentFlags(t *testing.T) {
	t.Parallel()

	var anyVersion SemanticVersion
	if SupportsAgentProgressSummaries(anyVersion) {
		t.Error("SupportsAgentProgressSummaries should be false (flag absent from 2.1.158)")
	}
	if SupportsSubagentExecution(anyVersion) {
		t.Error("SupportsSubagentExecution should be false (flag absent from 2.1.158)")
	}
}

// TestBuildCommandArgs_GatesExperimentalFlags proves buildCommandArgs only emits the
// experimental flags when the support gate is set — so an unsupported CLI never
// receives a flag that would crash Connect.
func TestBuildCommandArgs_GatesExperimentalFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		supported bool
		wantFlag  bool
	}{
		{name: "unsupported → flag skipped", supported: false, wantFlag: false},
		{name: "supported → flag emitted", supported: true, wantFlag: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run("agent-progress-summaries/"+tt.name, func(t *testing.T) {
			t.Parallel()
			opts := types.NewClaudeAgentOptions()
			opts.AgentProgressSummaries = true
			tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)
			tr.agentProgressSummariesSupported = tt.supported

			args := tr.buildCommandArgs()
			has := slices.Contains(args, "--agent-progress-summaries")
			if has != tt.wantFlag {
				t.Fatalf("--agent-progress-summaries present=%v, want %v (args=%v)", has, tt.wantFlag, args)
			}
		})

		t.Run("subagent-execution/"+tt.name, func(t *testing.T) {
			t.Parallel()
			opts := types.NewClaudeAgentOptions()
			opts.SubagentExecution = &types.SubagentExecutionConfig{MaxConcurrent: 2}
			tr := NewSubprocessCLITransport("/fake/claude", "", nil, log.NewLogger(false), "", opts)
			tr.subagentExecutionSupported = tt.supported

			args := tr.buildCommandArgs()
			has := slices.Contains(args, "--subagent-execution")
			if has != tt.wantFlag {
				t.Fatalf("--subagent-execution present=%v, want %v (args=%v)", has, tt.wantFlag, args)
			}
		})
	}
}
