package transport

import (
	"errors"
	"strings"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestExtractSessionNotFoundError tests parsing of session not found errors from stderr
func TestExtractSessionNotFoundError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		stderrText    string
		wantMatched   bool
		wantSessionID string
	}{
		{
			name:          "valid session not found error",
			stderrText:    "No conversation found with session ID: 8587b432-e504-42c8-b9a7-e3fd0b4b2c60",
			wantMatched:   true,
			wantSessionID: "8587b432-e504-42c8-b9a7-e3fd0b4b2c60",
		},
		{
			name:          "session not found with extra text",
			stderrText:    "Error: No conversation found with session ID: 12345678-1234-1234-1234-123456789abc. Please check the ID.",
			wantMatched:   true,
			wantSessionID: "12345678-1234-1234-1234-123456789abc.",
		},
		{
			name:          "session not found with leading whitespace",
			stderrText:    "No conversation found with session ID:   abc123-def456  ",
			wantMatched:   true,
			wantSessionID: "abc123-def456",
		},
		{
			name:          "different error message",
			stderrText:    "Connection failed: timeout",
			wantMatched:   false,
			wantSessionID: "",
		},
		{
			name:          "partial match",
			stderrText:    "No conversation found",
			wantMatched:   false,
			wantSessionID: "",
		},
		{
			name:          "empty string",
			stderrText:    "",
			wantMatched:   false,
			wantSessionID: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotMatched, gotSessionID := extractSessionNotFoundError(tt.stderrText)

			if gotMatched != tt.wantMatched {
				t.Errorf("extractSessionNotFoundError() matched = %v, want %v", gotMatched, tt.wantMatched)
			}

			if gotSessionID != tt.wantSessionID {
				t.Errorf("extractSessionNotFoundError() sessionID = %q, want %q", gotSessionID, tt.wantSessionID)
			}
		})
	}
}

// TestParseStderrError tests the stderr error parsing and error creation
func TestParseStderrError(t *testing.T) {
	t.Parallel()
	logger := log.NewLogger(false)
	transport := &SubprocessCLITransport{
		logger:   logger,
		messages: make(chan types.Message, 10),
	}

	// Test session not found error
	stderrText := "No conversation found with session ID: 8587b432-e504-42c8-b9a7-e3fd0b4b2c60"
	transport.parseStderrError(stderrText)

	// Check that error was stored
	err := transport.GetError()
	if err == nil {
		t.Fatal("parseStderrError() should have stored an error")
	}

	// Check that it's the right error type
	if !types.IsSessionNotFoundError(err) {
		t.Errorf("parseStderrError() stored error type = %T, want SessionNotFoundError", err)
	}

	// Check session ID is in the error
	var sessionErr *types.SessionNotFoundError
	if errors.As(err, &sessionErr) {
		if sessionErr.SessionID != "8587b432-e504-42c8-b9a7-e3fd0b4b2c60" {
			t.Errorf("SessionNotFoundError.SessionID = %q, want %q",
				sessionErr.SessionID, "8587b432-e504-42c8-b9a7-e3fd0b4b2c60")
		}
	}
}

// TestForkSessionFlag tests that --fork-session flag is passed when ForkSession is true
func TestForkSessionFlag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		resumeSessionID string
		forkSession     bool
		wantResumeFlag  bool
		wantForkFlag    bool
	}{
		{
			name:            "with resume and fork session",
			resumeSessionID: "test-session-id",
			forkSession:     true,
			wantResumeFlag:  true,
			wantForkFlag:    true,
		},
		{
			name:            "with resume but no fork session",
			resumeSessionID: "test-session-id",
			forkSession:     false,
			wantResumeFlag:  true,
			wantForkFlag:    false,
		},
		{
			name:            "with fork session but no resume",
			resumeSessionID: "",
			forkSession:     true,
			wantResumeFlag:  false,
			wantForkFlag:    true,
		},
		{
			name:            "without resume and fork session",
			resumeSessionID: "",
			forkSession:     false,
			wantResumeFlag:  false,
			wantForkFlag:    false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create options with fork session setting
			opts := types.NewClaudeAgentOptions().
				WithForkSession(tt.forkSession)

			logger := log.NewLogger(false)
			transport := NewSubprocessCLITransport("/bin/echo", "", nil, logger, tt.resumeSessionID, opts)

			// Build command args (without actually connecting)
			args := transport.buildCommandArgs()

			// Convert to string for easier searching
			argsStr := strings.Join(args, " ")
			t.Logf("CLI args: %v", args)

			// Check for --resume flag
			hasResumeFlag := contains(args, "--resume")
			if hasResumeFlag != tt.wantResumeFlag {
				t.Errorf("--resume flag present = %v, want %v", hasResumeFlag, tt.wantResumeFlag)
			}

			// Check for session ID if resume flag is expected
			if tt.wantResumeFlag {
				hasSessionID := contains(args, tt.resumeSessionID)
				if !hasSessionID {
					t.Errorf("session ID %q not found in args: %v", tt.resumeSessionID, args)
				}
			}

			// Check for --fork-session flag
			hasForkFlag := contains(args, "--fork-session")
			if hasForkFlag != tt.wantForkFlag {
				t.Errorf("--fork-session flag present = %v, want %v\nArgs: %s", hasForkFlag, tt.wantForkFlag, argsStr)
			}
		})
	}
}

// contains checks if a slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
