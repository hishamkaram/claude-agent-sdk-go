package transport

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestGetExitCode tests the getExitCode helper function.
func TestGetExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "nil error returns -1",
			err:  nil,
			want: -1,
		},
		{
			name: "generic error returns -1",
			err:  errors.New("something broke"),
			want: -1,
		},
		{
			name: "wrapped generic error returns -1",
			err:  errors.New("wrapped: something broke"),
			want: -1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getExitCode(tt.err)
			if got != tt.want {
				t.Errorf("getExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestGetExitCode_ExecExitError tests getExitCode with a real exec.ExitError.
func TestGetExitCode_ExecExitError(t *testing.T) {
	t.Parallel()

	// Run a command that exits with code 42 to produce a real *exec.ExitError
	cmd := exec.Command("sh", "-c", "exit 42")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected command to fail")
	}

	got := getExitCode(err)
	if got != 42 {
		t.Errorf("getExitCode(exit 42) = %d, want 42", got)
	}
}

// TestGetExitCode_WrappedExitError tests that getExitCode works through error wrapping.
func TestGetExitCode_WrappedExitError(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("sh", "-c", "exit 7")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected command to fail")
	}

	// Verify errors.As works for the wrapped ExitError
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal("expected exec.ExitError")
	}

	got := getExitCode(err)
	if got != 7 {
		t.Errorf("getExitCode(exit 7) = %d, want 7", got)
	}
}

// TestBuildEnvMap tests the buildEnvMap method of SubprocessCLITransport.
func TestBuildEnvMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		opts       *types.ClaudeAgentOptions
		customEnv  map[string]string
		wantKeys   map[string]string // key -> expected value
		wantAbsent []string          // keys that should NOT be present
	}{
		{
			name:     "nil options — only SDK env vars",
			opts:     nil,
			wantKeys: map[string]string{"CLAUDE_CODE_ENTRYPOINT": "agent", "CLAUDE_AGENT_SDK_VERSION": SDKVersion},
		},
		{
			name: "with model",
			opts: func() *types.ClaudeAgentOptions {
				o := types.NewClaudeAgentOptions()
				o.WithModel("claude-3")
				return o
			}(),
			wantKeys: map[string]string{"ANTHROPIC_MODEL": "claude-3"},
		},
		{
			name: "with base URL",
			opts: func() *types.ClaudeAgentOptions {
				o := types.NewClaudeAgentOptions()
				o.WithBaseURL("https://custom.api.com")
				return o
			}(),
			wantKeys: map[string]string{"ANTHROPIC_BASE_URL": "https://custom.api.com"},
		},
		{
			name:      "custom env overrides SDK defaults",
			opts:      nil,
			customEnv: map[string]string{"CLAUDE_CODE_ENTRYPOINT": "custom", "MY_VAR": "val"},
			wantKeys:  map[string]string{"CLAUDE_CODE_ENTRYPOINT": "custom", "MY_VAR": "val"},
		},
		{
			name: "model and base URL both set",
			opts: func() *types.ClaudeAgentOptions {
				o := types.NewClaudeAgentOptions()
				o.WithModel("claude-3").WithBaseURL("https://api.example.com")
				return o
			}(),
			wantKeys: map[string]string{
				"ANTHROPIC_MODEL":    "claude-3",
				"ANTHROPIC_BASE_URL": "https://api.example.com",
			},
		},
		{
			name:       "no model — ANTHROPIC_MODEL absent",
			opts:       types.NewClaudeAgentOptions(),
			wantAbsent: []string{"ANTHROPIC_MODEL"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			transport := NewSubprocessCLITransport(
				"/usr/local/bin/claude",
				"",
				tt.customEnv,
				log.NewLogger(false),
				"",
				tt.opts,
			)

			envMap := transport.buildEnvMap()

			for key, wantVal := range tt.wantKeys {
				if gotVal, ok := envMap[key]; !ok {
					t.Errorf("envMap missing key %q", key)
				} else if gotVal != wantVal {
					t.Errorf("envMap[%q] = %q, want %q", key, gotVal, wantVal)
				}
			}

			for _, key := range tt.wantAbsent {
				if _, ok := envMap[key]; ok {
					t.Errorf("envMap should not contain key %q", key)
				}
			}
		})
	}
}

// TestExpandHome_AllBranches tests the expandHome function for all code paths.
func TestExpandHome_AllBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantPrefix string // expected prefix after expansion
		wantExact  string // exact match (only checked if non-empty)
	}{
		{
			name:      "no tilde — unchanged",
			input:     "/usr/local/bin/claude",
			wantExact: "/usr/local/bin/claude",
		},
		{
			name:      "empty string",
			input:     "",
			wantExact: "",
		},
		{
			name:       "tilde slash expands home dir",
			input:      "~/.claude/local/claude",
			wantPrefix: "/",
		},
		{
			name:       "bare tilde expands to home dir",
			input:      "~",
			wantPrefix: "/",
		},
		{
			name:      "tilde-username — not expanded",
			input:     "~otheruser/bin",
			wantExact: "~otheruser/bin",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := expandHome(tt.input)

			if tt.wantExact != "" {
				if got != tt.wantExact {
					t.Errorf("expandHome(%q) = %q, want %q", tt.input, got, tt.wantExact)
				}
			} else if tt.wantPrefix != "" {
				if !strings.HasPrefix(got, tt.wantPrefix) {
					t.Errorf("expandHome(%q) = %q, want prefix %q", tt.input, got, tt.wantPrefix)
				}
			}

			// Tilde paths should never still start with ~ after expansion (except ~username)
			if strings.HasPrefix(tt.input, "~/") && strings.HasPrefix(got, "~") {
				t.Errorf("expandHome(%q) = %q, tilde should have been expanded", tt.input, got)
			}
		})
	}
}

// TestJSONLineReader_MultipleLines tests reading multiple JSON lines.
func TestJSONLineReader_MultipleLines(t *testing.T) {
	t.Parallel()

	input := `{"type":"user","content":"hello"}
{"type":"assistant","content":"hi"}
{"type":"result","subtype":"success"}
`
	reader := NewJSONLineReader(strings.NewReader(input))

	wantLines := []string{
		`{"type":"user","content":"hello"}`,
		`{"type":"assistant","content":"hi"}`,
		`{"type":"result","subtype":"success"}`,
	}

	for i, want := range wantLines {
		line, err := reader.ReadLine()
		if err != nil {
			t.Fatalf("ReadLine() %d: unexpected error: %v", i, err)
		}
		if string(line) != want {
			t.Errorf("ReadLine() %d = %q, want %q", i, string(line), want)
		}
	}

	// Next read should be EOF
	_, err := reader.ReadLine()
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

// TestJSONLineWriter_Success tests successful writes with newlines.
func TestJSONLineWriter_Success(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writer := NewJSONLineWriter(&buf)

	if err := writer.WriteLine(`{"type":"test"}`); err != nil {
		t.Fatalf("WriteLine() error: %v", err)
	}

	got := buf.String()
	want := `{"type":"test"}` + "\n"
	if got != want {
		t.Errorf("written = %q, want %q", got, want)
	}
}

// TestJSONLineWriter_MultipleWrites tests multiple consecutive writes.
func TestJSONLineWriter_MultipleWrites(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writer := NewJSONLineWriter(&buf)

	lines := []string{
		`{"first":"line"}`,
		`{"second":"line"}`,
		`{"third":"line"}`,
	}

	for _, line := range lines {
		if err := writer.WriteLine(line); err != nil {
			t.Fatalf("WriteLine() error: %v", err)
		}
	}

	got := buf.String()
	for _, line := range lines {
		if !strings.Contains(got, line) {
			t.Errorf("output missing line %q", line)
		}
	}
}

// TestJSONLineWriter_FlushError tests that flush errors are propagated with context.
func TestJSONLineWriter_FlushError(t *testing.T) {
	t.Parallel()

	// Use a writer that always fails
	fw := &errorWriter{err: errors.New("write error")}
	writer := NewJSONLineWriter(fw)

	err := writer.WriteLine(`{"data":"test"}`)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "transport.JSONLineWriter.WriteLine") {
		t.Errorf("error = %q, want context prefix", err.Error())
	}
}

// TestParseSemanticVersion_AdditionalCases tests additional version parsing cases.
func TestParseSemanticVersion_AdditionalCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		version   string
		wantMajor int
		wantMinor int
		wantPatch int
		wantErr   bool
	}{
		{
			name:      "standard version",
			version:   "2.1.0",
			wantMajor: 2,
			wantMinor: 1,
			wantPatch: 0,
		},
		{
			name:      "v prefix",
			version:   "v2.1.0",
			wantMajor: 2,
			wantMinor: 1,
			wantPatch: 0,
		},
		{
			name:      "with rc suffix",
			version:   "2.1.0-rc1",
			wantMajor: 2,
			wantMinor: 1,
			wantPatch: 0,
		},
		{
			name:      "with build metadata",
			version:   "3.0.5+build.123",
			wantMajor: 3,
			wantMinor: 0,
			wantPatch: 5,
		},
		{
			name:    "empty string",
			version: "",
			wantErr: true,
		},
		{
			name:    "single number",
			version: "2",
			wantErr: true,
		},
		{
			name:    "two numbers",
			version: "2.1",
			wantErr: true,
		},
		{
			name:    "letters only",
			version: "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ver, err := ParseSemanticVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseSemanticVersion(%q) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
			if !tt.wantErr {
				if ver.Major != tt.wantMajor || ver.Minor != tt.wantMinor || ver.Patch != tt.wantPatch {
					t.Errorf("ParseSemanticVersion(%q) = %v, want %d.%d.%d",
						tt.version, ver, tt.wantMajor, tt.wantMinor, tt.wantPatch)
				}
			}
		})
	}
}
