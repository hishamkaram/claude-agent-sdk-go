package transport

import (
	"slices"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestBuildCommandArgsPartialMessages(t *testing.T) {
	t.Parallel()
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "default", true: "enabled"}[enabled], func(t *testing.T) {
			t.Parallel()
			transport := newTestTransport(t, types.NewClaudeAgentOptions().WithIncludePartialMessages(enabled))
			if got := slices.Contains(transport.buildCommandArgs(), "--include-partial-messages"); got != enabled {
				t.Fatalf("partial output flag = %v, want %v", got, enabled)
			}
		})
	}
}
