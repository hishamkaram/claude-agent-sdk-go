//go:build integration

package tests

import (
	"sort"
	"testing"
	"time"

	claude "github.com/hishamkaram/claude-agent-sdk-go"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestColdStartLatency_RealCLI measures real end-to-end cold-start latency (spawn →
// Connect ready) of the native Claude CLI, with no model turn (no tokens spent). It
// establishes the baseline that gates the warm-pool go/no-go decision: if cold start
// is already fast on the native binary, a pre-warm cache is not worth its risk.
//
// It does not assert a hard threshold (latency is environment-dependent) — it records
// min/median/max so the warm-pool decision is made from measured facts, not guesses.
func TestColdStartLatency_RealCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	requireAuth(t)
	cliPath := requireClaude(t)

	const iterations = 5
	durations := make([]time.Duration, 0, iterations)

	for i := 0; i < iterations; i++ {
		ctx, cancel := CreateTestContext(t, 30*time.Second)

		opts := types.NewClaudeAgentOptions().WithCLIPath(cliPath)
		client, err := claude.NewClient(ctx, opts)
		if err != nil {
			cancel()
			t.Fatalf("iteration %d: NewClient() failed: %v", i, err)
		}

		start := time.Now()
		if err := client.Connect(ctx); err != nil {
			_ = client.Close(ctx)
			cancel()
			t.Fatalf("iteration %d: Connect() failed: %v", i, err)
		}
		d := time.Since(start)
		durations = append(durations, d)
		t.Logf("cold start %d/%d: %v", i+1, iterations, d)

		_ = client.Close(ctx)
		cancel()
	}

	sort.Slice(durations, func(a, b int) bool { return durations[a] < durations[b] })
	min := durations[0]
	max := durations[len(durations)-1]
	median := durations[len(durations)/2]

	t.Logf("COLD-START BASELINE (native claude CLI, %d runs): min=%v median=%v max=%v",
		iterations, min, median, max)
	t.Logf("WARM-POOL DECISION INPUT: if median (%v) is small relative to a session's "+
		"lifetime, the warm pool's flag/CWD-fingerprint risk is not worth the marginal "+
		"saving — prefer DROP.", median)
}
