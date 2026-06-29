package transport

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestTransportInterfaceDoesNotExposeStderr(t *testing.T) {
	t.Parallel()

	transportType := reflect.TypeOf((*Transport)(nil)).Elem()
	if _, ok := transportType.MethodByName("Stderr"); ok {
		t.Fatal("Transport interface must not expose Stderr(); diagnostics stay on concrete subprocess transport")
	}
}

// TestBuildCommandArgs_Effort tests --effort flag generation.
func TestBuildCommandArgs_Effort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		effort    *types.EffortLevel
		wantFlag  bool
		wantValue string
	}{
		{
			name:      "effort high",
			effort:    effortPtr(types.EffortHigh),
			wantFlag:  true,
			wantValue: "high",
		},
		{
			name:      "effort xhigh",
			effort:    effortPtr(types.EffortXHigh),
			wantFlag:  true,
			wantValue: "xhigh",
		},
		{
			name:      "effort low",
			effort:    effortPtr(types.EffortLow),
			wantFlag:  true,
			wantValue: "low",
		},
		{
			name:      "effort medium",
			effort:    effortPtr(types.EffortMedium),
			wantFlag:  true,
			wantValue: "medium",
		},
		{
			name:      "effort max",
			effort:    effortPtr(types.EffortMax),
			wantFlag:  true,
			wantValue: "max",
		},
		{
			name:     "effort nil — flag absent",
			effort:   nil,
			wantFlag: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.NewClaudeAgentOptions()
			if tt.effort != nil {
				opts.WithEffort(*tt.effort)
			}

			transport := newTestTransport(t, opts)
			args := transport.buildCommandArgs()

			if got := hasFlag(args, "--effort"); got != tt.wantFlag {
				t.Errorf("--effort flag present = %v, want %v", got, tt.wantFlag)
			}

			if tt.wantFlag {
				val, ok := flagValue(args, "--effort")
				if !ok {
					t.Fatal("--effort flag present but no value follows")
				}
				if val != tt.wantValue {
					t.Errorf("--effort value = %q, want %q", val, tt.wantValue)
				}
			}
		})
	}
}

// TestBuildCommandArgs_McpConfig tests --mcp-config inline-JSON flag generation
// from options.McpServers. The CLI expects an inline config object of the form
// {"mcpServers": {<name>: <config>, ...}} (verified against the installed Claude
// CLI binary strings: ".mcp.json is malformed (... or mcpServers is not an
// object)" and "Rename the top-level \"servers\" key to \"mcpServers\""). The
// option holds the INNER name->config map, so the transport must wrap it in the
// {"mcpServers": <map>} envelope. nil/empty maps emit no flag (backward compat).
func TestBuildCommandArgs_McpConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mcpServers interface{}
		wantFlag   bool
		// wantEnvelope is the decoded envelope the inline JSON must equal.
		// Only checked when wantFlag is true and wantEnvelope is non-nil.
		wantEnvelope map[string]interface{}
	}{
		{
			name:       "single server wraps in mcpServers envelope",
			mcpServers: map[string]interface{}{"delegate": map[string]interface{}{"command": "echo", "args": []interface{}{"noop"}}},
			wantFlag:   true,
			wantEnvelope: map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"delegate": map[string]interface{}{"command": "echo", "args": []interface{}{"noop"}},
				},
			},
		},
		{
			name: "multiple servers all wrapped",
			mcpServers: map[string]interface{}{
				"scanner": map[string]interface{}{"command": "scanner_bin"},
				"editor":  map[string]interface{}{"command": "editor_bin"},
			},
			wantFlag: true,
			wantEnvelope: map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"scanner": map[string]interface{}{"command": "scanner_bin"},
					"editor":  map[string]interface{}{"command": "editor_bin"},
				},
			},
		},
		{
			name:       "nil McpServers — flag absent (backward compatible)",
			mcpServers: nil,
			wantFlag:   false,
		},
		{
			name:       "empty map — flag absent (backward compatible)",
			mcpServers: map[string]interface{}{},
			wantFlag:   false,
		},
		{
			name:       "non-map string type — flag absent (defensive)",
			mcpServers: "not-a-map",
			wantFlag:   false,
		},
		{
			name:       "non-map slice type — flag absent (defensive)",
			mcpServers: []interface{}{"scanner"},
			wantFlag:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.NewClaudeAgentOptions()
			if tt.mcpServers != nil {
				opts.WithMcpServers(tt.mcpServers)
			}

			transport := newTestTransport(t, opts)
			args := transport.buildCommandArgs()

			if got := hasFlag(args, "--mcp-config"); got != tt.wantFlag {
				t.Errorf("--mcp-config flag present = %v, want %v", got, tt.wantFlag)
			}

			if !tt.wantFlag {
				return
			}

			val, ok := flagValue(args, "--mcp-config")
			if !ok {
				t.Fatal("--mcp-config flag present but no value follows")
			}
			// Inline JSON only — the value must NOT be a file path; it must parse as JSON.
			var gotEnvelope map[string]interface{}
			if err := json.Unmarshal([]byte(val), &gotEnvelope); err != nil {
				t.Fatalf("--mcp-config value is not inline JSON (got %q): %v", val, err)
			}
			if tt.wantEnvelope != nil && !reflect.DeepEqual(gotEnvelope, tt.wantEnvelope) {
				t.Errorf("--mcp-config envelope = %#v, want %#v", gotEnvelope, tt.wantEnvelope)
			}
		})
	}
}

// TestBuildCommandArgs_FallbackModel tests --fallback-model flag generation.
func TestBuildCommandArgs_FallbackModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		model     *string
		wantFlag  bool
		wantValue string
	}{
		{
			name:      "fallback model set",
			model:     strPtr("claude-3-haiku"),
			wantFlag:  true,
			wantValue: "claude-3-haiku",
		},
		{
			name:     "fallback model nil — flag absent",
			model:    nil,
			wantFlag: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.NewClaudeAgentOptions()
			if tt.model != nil {
				opts.WithFallbackModel(*tt.model)
			}

			transport := newTestTransport(t, opts)
			args := transport.buildCommandArgs()

			if got := hasFlag(args, "--fallback-model"); got != tt.wantFlag {
				t.Errorf("--fallback-model flag present = %v, want %v", got, tt.wantFlag)
			}

			if tt.wantFlag {
				val, ok := flagValue(args, "--fallback-model")
				if !ok {
					t.Fatal("--fallback-model flag present but no value follows")
				}
				if val != tt.wantValue {
					t.Errorf("--fallback-model value = %q, want %q", val, tt.wantValue)
				}
			}
		})
	}
}

// TestBuildCommandArgs_SessionID tests --session-id flag generation.
func TestBuildCommandArgs_SessionID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sessionID *string
		wantFlag  bool
		wantValue string
	}{
		{
			name:      "session ID set",
			sessionID: strPtr("abc-123"),
			wantFlag:  true,
			wantValue: "abc-123",
		},
		{
			name:      "session ID with UUID",
			sessionID: strPtr("8587b432-e504-42c8-b9a7-e3fd0b4b2c60"),
			wantFlag:  true,
			wantValue: "8587b432-e504-42c8-b9a7-e3fd0b4b2c60",
		},
		{
			name:      "session ID nil — flag absent",
			sessionID: nil,
			wantFlag:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.NewClaudeAgentOptions()
			if tt.sessionID != nil {
				opts.WithSessionID(*tt.sessionID)
			}

			transport := newTestTransport(t, opts)
			args := transport.buildCommandArgs()

			if got := hasFlag(args, "--session-id"); got != tt.wantFlag {
				t.Errorf("--session-id flag present = %v, want %v", got, tt.wantFlag)
			}

			if tt.wantFlag {
				val, ok := flagValue(args, "--session-id")
				if !ok {
					t.Fatal("--session-id flag present but no value follows")
				}
				if val != tt.wantValue {
					t.Errorf("--session-id value = %q, want %q", val, tt.wantValue)
				}
			}
		})
	}
}

// TestBuildCommandArgs_PersistSession tests --no-session-persistence flag generation.
func TestBuildCommandArgs_PersistSession(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		persist  *bool
		wantFlag bool
	}{
		{
			name:     "PersistSession false — flag present",
			persist:  boolPtr(false),
			wantFlag: true,
		},
		{
			name:     "PersistSession true — flag absent",
			persist:  boolPtr(true),
			wantFlag: false,
		},
		{
			name:     "PersistSession nil — flag absent",
			persist:  nil,
			wantFlag: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := types.NewClaudeAgentOptions()
			if tt.persist != nil {
				opts.WithPersistSession(*tt.persist)
			}

			transport := newTestTransport(t, opts)
			args := transport.buildCommandArgs()

			if got := hasFlag(args, "--no-session-persistence"); got != tt.wantFlag {
				t.Errorf("--no-session-persistence flag present = %v, want %v", got, tt.wantFlag)
			}
		})
	}
}

// TestBuildCommandArgs_OutputFormat tests --json-schema flag generation.
