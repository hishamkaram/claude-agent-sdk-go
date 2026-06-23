package transport

import (
	"slices"
	"testing"
)

// TestRedactArgsForLog verifies the logging-only redaction helper elides the
// inline --mcp-config JSON value and the delegate --sock path value while
// leaving every other flag/value verbatim. This is defense-in-depth: the inline
// MCP config envelope and the UDS socket path are sensitive under the
// same-UID threat model and must not land in a Debug args dump.
func TestRedactArgsForLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "nil",
			in:   nil,
			want: nil,
		},
		{
			name: "no sensitive flags passes through unchanged",
			in:   []string{"--output-format", "stream-json", "--verbose"},
			want: []string{"--output-format", "stream-json", "--verbose"},
		},
		{
			name: "mcp-config value redacted, flag presence kept",
			in:   []string{"--mcp-config", `{"mcpServers":{"a":{},"b":{}}}`, "--verbose"},
			want: []string{"--mcp-config", "[redacted]", "--verbose"},
		},
		{
			name: "sock value redacted",
			in:   []string{"__delegate-shim", "--sock", "/tmp/agentd-delegate-abc.sock", "--verbose"},
			want: []string{"__delegate-shim", "--sock", "[redacted]", "--verbose"},
		},
		{
			name: "both redacted in one argv",
			in: []string{
				"--mcp-config", `{"mcpServers":{"x":{}}}`,
				"--sock", "/run/agentd/d.sock",
				"--output-format", "stream-json",
			},
			want: []string{
				"--mcp-config", "[redacted]",
				"--sock", "[redacted]",
				"--output-format", "stream-json",
			},
		},
		{
			name: "trailing sensitive flag with no value is left as-is",
			in:   []string{"--verbose", "--mcp-config"},
			want: []string{"--verbose", "--mcp-config"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := redactArgsForLog(tt.in)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("redactArgsForLog(%v) = %v, want %v", tt.in, got, tt.want)
			}
			// Logging-only invariant: the input slice must not be mutated.
			if tt.in != nil {
				for i := range tt.in {
					if tt.in[i] == "[redacted]" {
						t.Fatalf("redactArgsForLog mutated input slice at index %d: %v", i, tt.in)
					}
				}
			}
		})
	}
}
