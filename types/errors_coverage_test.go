package types

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestAllErrorTypes_Constructor_Error_Is_Unwrap_As exercises every error type:
// constructor, .Error(), .Is(), .Unwrap(), errors.As() — with and without cause.
func TestAllErrorTypes_Constructor_Error_Is_Unwrap_As(t *testing.T) {
	t.Parallel()

	cause := errors.New("root cause")

	tests := []struct {
		name          string
		makeNoCause   func() error
		makeWithCause func(error) error
		isCheck       func(error) bool
		wantMsgSub    string // substring expected in .Error() without cause
	}{
		{
			name:          "CLINotFoundError",
			makeNoCause:   func() error { return NewCLINotFoundError("cli not found") },
			makeWithCause: func(c error) error { return NewCLINotFoundErrorWithCause("cli not found", c) },
			isCheck:       IsCLINotFoundError,
			wantMsgSub:    "cli not found",
		},
		{
			name:          "CLIConnectionError",
			makeNoCause:   func() error { return NewCLIConnectionError("connection failed") },
			makeWithCause: func(c error) error { return NewCLIConnectionErrorWithCause("connection failed", c) },
			isCheck:       IsCLIConnectionError,
			wantMsgSub:    "connection failed",
		},
		{
			name:          "ProcessError",
			makeNoCause:   func() error { return NewProcessError("process died") },
			makeWithCause: func(c error) error { return NewProcessErrorWithCause("process died", c) },
			isCheck:       IsProcessError,
			wantMsgSub:    "process died",
		},
		{
			name:          "JSONDecodeError",
			makeNoCause:   func() error { return NewJSONDecodeError("bad json") },
			makeWithCause: func(c error) error { return NewJSONDecodeErrorWithCause("bad json", "", c) },
			isCheck:       IsJSONDecodeError,
			wantMsgSub:    "bad json",
		},
		{
			name:          "MessageParseError",
			makeNoCause:   func() error { return NewMessageParseError("bad msg") },
			makeWithCause: func(c error) error { return NewMessageParseErrorWithCause("bad msg", "unknown", c) },
			isCheck:       IsMessageParseError,
			wantMsgSub:    "bad msg",
		},
		{
			name:          "ControlProtocolError",
			makeNoCause:   func() error { return NewControlProtocolError("protocol error") },
			makeWithCause: func(c error) error { return NewControlProtocolErrorWithCause("protocol error", c) },
			isCheck:       IsControlProtocolError,
			wantMsgSub:    "protocol error",
		},
		{
			name:          "PermissionDeniedError",
			makeNoCause:   func() error { return NewPermissionDeniedError("denied") },
			makeWithCause: func(c error) error { return NewPermissionDeniedErrorWithCause("denied", c) },
			isCheck:       IsPermissionDeniedError,
			wantMsgSub:    "denied",
		},
		{
			name:          "SessionNotFoundError",
			makeNoCause:   func() error { return NewSessionNotFoundError("sess-1", "not found") },
			makeWithCause: func(c error) error { return NewSessionNotFoundErrorWithCause("sess-1", "not found", c) },
			isCheck:       IsSessionNotFoundError,
			wantMsgSub:    "not found",
		},
		{
			name:          "ValidationError",
			makeNoCause:   func() error { return NewValidationError("invalid input") },
			makeWithCause: func(c error) error { return NewValidationErrorWithCause("invalid input", c) },
			isCheck:       IsValidationError,
			wantMsgSub:    "invalid input",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name+"/no_cause", func(t *testing.T) {
			t.Parallel()
			err := tt.makeNoCause()

			// .Error() contains the expected substring
			if !strings.Contains(err.Error(), tt.wantMsgSub) {
				t.Errorf("Error() = %q, want substring %q", err.Error(), tt.wantMsgSub)
			}

			// Is helper returns true for same-type
			if !tt.isCheck(err) {
				t.Errorf("Is helper returned false for same error type")
			}

			// Is helper returns false for unrelated error
			if tt.isCheck(errors.New("unrelated")) {
				t.Errorf("Is helper returned true for unrelated error")
			}

			// Unwrap returns nil when no cause
			unwrapper, ok := err.(interface{ Unwrap() error })
			if ok && unwrapper.Unwrap() != nil {
				t.Errorf("Unwrap() should be nil when no cause is set")
			}
		})

		t.Run(tt.name+"/with_cause", func(t *testing.T) {
			t.Parallel()
			err := tt.makeWithCause(cause)

			// .Error() includes cause string
			if !strings.Contains(err.Error(), cause.Error()) {
				t.Errorf("Error() = %q, should contain cause %q", err.Error(), cause.Error())
			}

			// Unwrap returns the cause
			unwrapper, ok := err.(interface{ Unwrap() error })
			if !ok {
				t.Fatal("error does not implement Unwrap()")
			}
			if unwrapper.Unwrap() != cause {
				t.Errorf("Unwrap() = %v, want %v", unwrapper.Unwrap(), cause)
			}

			// errors.As works for the correct type
			if !tt.isCheck(err) {
				t.Errorf("Is helper returned false for error with cause")
			}
		})

		t.Run(tt.name+"/wrapped_errors_as", func(t *testing.T) {
			t.Parallel()
			err := tt.makeWithCause(cause)
			wrapped := fmt.Errorf("outer: %w", err)

			// Is helper works through wrapping
			if !tt.isCheck(wrapped) {
				t.Errorf("Is helper returned false for wrapped error")
			}
		})
	}
}

// TestProcessErrorWithCode tests ProcessError with various exit codes.
func TestProcessErrorWithCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		msg      string
		exitCode int
		wantSub  string
	}{
		{
			name:     "nonzero exit code in error string",
			msg:      "process failed",
			exitCode: 1,
			wantSub:  "exit code: 1",
		},
		{
			name:     "zero exit code omitted from string",
			msg:      "process ok",
			exitCode: 0,
			wantSub:  "process ok",
		},
		{
			name:     "negative exit code displayed",
			msg:      "process killed",
			exitCode: -1,
			wantSub:  "exit code: -1",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := NewProcessErrorWithCode(tt.msg, tt.exitCode)
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("Error() = %q, want substring %q", err.Error(), tt.wantSub)
			}
			if err.ExitCode != tt.exitCode {
				t.Errorf("ExitCode = %d, want %d", err.ExitCode, tt.exitCode)
			}
		})
	}
}

// TestProcessErrorWithCauseAndCode tests ProcessError with both cause and exit code.
func TestProcessErrorWithCauseAndCode(t *testing.T) {
	t.Parallel()
	cause := errors.New("signal killed")
	err := &ProcessError{
		Message:  "process died",
		ExitCode: 137,
		Cause:    cause,
	}

	got := err.Error()
	if !strings.Contains(got, "exit code: 137") {
		t.Errorf("Error() = %q, want exit code substring", got)
	}
	if !strings.Contains(got, "signal killed") {
		t.Errorf("Error() = %q, want cause substring", got)
	}
}

// TestJSONDecodeErrorWithRaw_Truncation tests that raw data > 100 chars is truncated.
func TestJSONDecodeErrorWithRaw_Truncation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		wantDot bool // expect "..." suffix
	}{
		{
			name:    "short raw included verbatim",
			raw:     "short",
			wantDot: false,
		},
		{
			name:    "exactly 100 chars not truncated",
			raw:     strings.Repeat("x", 100),
			wantDot: false,
		},
		{
			name:    "101 chars truncated",
			raw:     strings.Repeat("y", 101),
			wantDot: true,
		},
		{
			name:    "empty raw omitted from message",
			raw:     "",
			wantDot: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := NewJSONDecodeErrorWithRaw("decode failed", tt.raw)
			msg := err.Error()
			if tt.wantDot && !strings.HasSuffix(msg, "...)") {
				t.Errorf("Error() = %q, expected truncation suffix '...'", msg)
			}
			if !tt.wantDot && tt.raw != "" && !strings.Contains(msg, tt.raw) {
				t.Errorf("Error() = %q, expected to contain raw %q", msg, tt.raw)
			}
		})
	}
}

// TestJSONDecodeErrorWithCause_RawAndCause tests the full constructor.
func TestJSONDecodeErrorWithCause_RawAndCause(t *testing.T) {
	t.Parallel()
	cause := errors.New("unexpected token")
	err := NewJSONDecodeErrorWithCause("parse failed", `{"broken`, cause)

	got := err.Error()
	if !strings.Contains(got, "parse failed") {
		t.Errorf("Error() missing message, got %q", got)
	}
	if !strings.Contains(got, `{"broken`) {
		t.Errorf("Error() missing raw, got %q", got)
	}
	if !strings.Contains(got, "unexpected token") {
		t.Errorf("Error() missing cause, got %q", got)
	}
	if err.Unwrap() != cause {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), cause)
	}
}

// TestMessageParseErrorWithCause tests constructor with type and cause.
func TestMessageParseErrorWithCause(t *testing.T) {
	t.Parallel()
	cause := errors.New("missing field")
	err := NewMessageParseErrorWithCause("parse failed", "assistant", cause)

	got := err.Error()
	if !strings.Contains(got, "assistant") {
		t.Errorf("Error() missing type, got %q", got)
	}
	if !strings.Contains(got, "missing field") {
		t.Errorf("Error() missing cause, got %q", got)
	}
	if err.Unwrap() != cause {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), cause)
	}
}

// TestPermissionDeniedError_ToolAndReason tests combined tool+reason display.
func TestPermissionDeniedError_ToolAndReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *PermissionDeniedError
		wantSubs []string
	}{
		{
			name: "tool only",
			err: &PermissionDeniedError{
				Message:  "denied",
				ToolName: "Bash",
			},
			wantSubs: []string{"denied", "tool: Bash"},
		},
		{
			name: "tool and reason",
			err: &PermissionDeniedError{
				Message:  "denied",
				ToolName: "Bash",
				Reason:   "unsafe",
			},
			wantSubs: []string{"denied", "tool: Bash", "unsafe"},
		},
		{
			name: "reason only (no tool)",
			err: &PermissionDeniedError{
				Message: "denied",
				Reason:  "policy",
			},
			wantSubs: []string{"denied", "policy"},
		},
		{
			name: "with cause",
			err: &PermissionDeniedError{
				Message:  "denied",
				ToolName: "Write",
				Reason:   "policy",
				Cause:    errors.New("upstream"),
			},
			wantSubs: []string{"denied", "Write", "policy", "upstream"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.err.Error()
			for _, sub := range tt.wantSubs {
				if !strings.Contains(got, sub) {
					t.Errorf("Error() = %q, missing %q", got, sub)
				}
			}
		})
	}
}

// TestSessionNotFoundError_Display tests SessionNotFoundError display variations.
func TestSessionNotFoundError_Display(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *SessionNotFoundError
		wantSubs []string
	}{
		{
			name:     "empty session ID omits ID from message",
			err:      &SessionNotFoundError{Message: "not found"},
			wantSubs: []string{"not found"},
		},
		{
			name:     "with session ID",
			err:      &SessionNotFoundError{SessionID: "abc-123", Message: "not found"},
			wantSubs: []string{"not found", "session ID: abc-123"},
		},
		{
			name: "with cause",
			err: &SessionNotFoundError{
				SessionID: "abc",
				Message:   "not found",
				Cause:     errors.New("db error"),
			},
			wantSubs: []string{"not found", "abc", "db error"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.err.Error()
			for _, sub := range tt.wantSubs {
				if !strings.Contains(got, sub) {
					t.Errorf("Error() = %q, missing %q", got, sub)
				}
			}
		})
	}
}

// TestIsCheckers_ReturnsFalseForNil tests that Is* helpers return false for nil.
func TestIsCheckers_ReturnsFalseForNil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		checker func(error) bool
	}{
		{"IsCLINotFoundError", IsCLINotFoundError},
		{"IsCLIConnectionError", IsCLIConnectionError},
		{"IsProcessError", IsProcessError},
		{"IsJSONDecodeError", IsJSONDecodeError},
		{"IsMessageParseError", IsMessageParseError},
		{"IsControlProtocolError", IsControlProtocolError},
		{"IsPermissionDeniedError", IsPermissionDeniedError},
		{"IsSessionNotFoundError", IsSessionNotFoundError},
		{"IsValidationError", IsValidationError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name+"_nil", func(t *testing.T) {
			t.Parallel()
			if tt.checker(nil) {
				t.Errorf("%s(nil) = true, want false", tt.name)
			}
		})
		t.Run(tt.name+"_generic_error", func(t *testing.T) {
			t.Parallel()
			if tt.checker(errors.New("generic")) {
				t.Errorf("%s(generic error) = true, want false", tt.name)
			}
		})
	}
}

// TestSentinelErrors tests the package-level sentinel errors.
func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{"ErrNoActiveQuery", ErrNoActiveQuery, "no active query"},
		{"ErrClientClosed", ErrClientClosed, "client is closed"},
		{"ErrEmptyParameter", ErrEmptyParameter, "required parameter is empty"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.err.Error() != tt.want {
				t.Errorf("%s.Error() = %q, want %q", tt.name, tt.err.Error(), tt.want)
			}
			// Sentinel errors should work with errors.Is
			wrapped := fmt.Errorf("wrapped: %w", tt.err)
			if !errors.Is(wrapped, tt.err) {
				t.Errorf("errors.Is(wrapped, %s) = false, want true", tt.name)
			}
		})
	}
}

// TestValidationError_WithAndWithoutCause tests ValidationError display.
func TestValidationError_WithAndWithoutCause(t *testing.T) {
	t.Parallel()

	t.Run("without cause", func(t *testing.T) {
		t.Parallel()
		err := NewValidationError("name required")
		if err.Error() != "name required" {
			t.Errorf("Error() = %q, want %q", err.Error(), "name required")
		}
		if err.Unwrap() != nil {
			t.Errorf("Unwrap() should be nil")
		}
	})

	t.Run("with cause", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("empty")
		err := NewValidationErrorWithCause("name required", cause)
		if !strings.Contains(err.Error(), "empty") {
			t.Errorf("Error() = %q, missing cause", err.Error())
		}
		if err.Unwrap() != cause {
			t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), cause)
		}
	})
}

// TestControlProtocolError_Display tests ControlProtocolError display.
func TestControlProtocolError_Display(t *testing.T) {
	t.Parallel()

	t.Run("without cause", func(t *testing.T) {
		t.Parallel()
		err := NewControlProtocolError("bad version")
		if err.Error() != "bad version" {
			t.Errorf("Error() = %q", err.Error())
		}
	})

	t.Run("with cause", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("mismatch")
		err := NewControlProtocolErrorWithCause("bad version", cause)
		if !strings.Contains(err.Error(), "mismatch") {
			t.Errorf("Error() = %q, missing cause", err.Error())
		}
	})
}

// TestCLIConnectionError_Display tests CLIConnectionError display.
func TestCLIConnectionError_Display(t *testing.T) {
	t.Parallel()

	t.Run("without cause", func(t *testing.T) {
		t.Parallel()
		err := NewCLIConnectionError("connect failed")
		if err.Error() != "connect failed" {
			t.Errorf("Error() = %q", err.Error())
		}
		if err.Unwrap() != nil {
			t.Errorf("Unwrap() should be nil")
		}
	})

	t.Run("with cause", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("timeout")
		err := NewCLIConnectionErrorWithCause("connect failed", cause)
		if !strings.Contains(err.Error(), "timeout") {
			t.Errorf("Error() = %q, missing cause", err.Error())
		}
		if err.Unwrap() != cause {
			t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), cause)
		}
	})
}
