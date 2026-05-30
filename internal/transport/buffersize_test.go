package transport

import (
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func intPtr(i int) *int { return &i }

// TestMaxBufferSize_HonorsOption proves the previously-ignored MaxBufferSize option
// is now authoritative over the JSON line reader, with DefaultMaxBufferSize as the
// fallback for nil / unset / non-positive values.
func TestMaxBufferSize_HonorsOption(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options *types.ClaudeAgentOptions
		want    int
	}{
		{name: "nil options → default", options: nil, want: DefaultMaxBufferSize},
		{name: "nil MaxBufferSize → default", options: &types.ClaudeAgentOptions{}, want: DefaultMaxBufferSize},
		{name: "zero → default (guards footgun)", options: &types.ClaudeAgentOptions{MaxBufferSize: intPtr(0)}, want: DefaultMaxBufferSize},
		{name: "negative → default (guards footgun)", options: &types.ClaudeAgentOptions{MaxBufferSize: intPtr(-1)}, want: DefaultMaxBufferSize},
		{name: "positive override is honored", options: &types.ClaudeAgentOptions{MaxBufferSize: intPtr(5 * 1024 * 1024)}, want: 5 * 1024 * 1024},
		{name: "small positive override is honored", options: &types.ClaudeAgentOptions{MaxBufferSize: intPtr(2048)}, want: 2048},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tr := &SubprocessCLITransport{options: tt.options}
			if got := tr.maxBufferSize(); got != tt.want {
				t.Fatalf("maxBufferSize() = %d, want %d", got, tt.want)
			}
		})
	}
}
