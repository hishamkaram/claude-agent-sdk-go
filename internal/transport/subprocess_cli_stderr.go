package transport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// readStderr reads stderr output in a goroutine for debugging.
// This is a helper function for monitoring subprocess errors.
// It also parses known error patterns and stores them as typed errors.
// stderr is passed as a parameter to avoid a data race with Connect() overwriting t.stderr.
func (t *SubprocessCLITransport) readStderr(ctx context.Context, stderr io.ReadCloser, tail *ringBuffer, done chan<- struct{}) {
	defer t.wg.Done()
	if done != nil {
		defer close(done)
	}

	if stderr == nil {
		return
	}

	logFile := t.openStderrLogFile()
	// Ensure cleanup if file was opened
	if logFile != nil {
		defer func() {
			_ = logFile.Close()
		}()
	}

	// Launch a goroutine that closes stderr when ctx is canceled.
	// This unblocks ReadLine() if the subprocess hangs without closing stderr (Bug C16).
	// The goroutine exits when either ctx is canceled (close stderr) or the read loop
	// finishes (readDone closed).
	readDone := make(chan struct{})
	monitorDone := make(chan struct{})
	defer func() {
		close(readDone)
		<-monitorDone
	}()
	go func() {
		defer close(monitorDone)
		t.closeStderrOnCancel(ctx, stderr, readDone)
	}()

	partial := make([]byte, 0, StderrRingSize)
	emit := func(fragment []byte) {
		t.emitStderrFragment(logFile, fragment)
	}
	flushPartial := func() {
		if len(partial) == 0 {
			return
		}
		emit(partial)
		partial = partial[:0]
	}

	buf := make([]byte, stderrReadChunkSize)
	for {
		n, err := stderr.Read(buf)
		if n > 0 {
			partial = t.consumeStderrChunk(tail, partial, buf[:n], emit)
		}
		if err != nil {
			flushPartial()
			return
		}
	}
}

// openStderrLogFile resolves and opens the stderr log file when StderrLogFile is
// configured, returning the open file or nil when logging is disabled or the
// file cannot be created. The caller owns closing the returned file. Failures to
// determine the home directory, create the directory, or open the file disable
// file logging (return nil); directory/open failures are reported to os.Stderr
// as best-effort diagnostics.
func (t *SubprocessCLITransport) openStderrLogFile() *os.File {
	if t.options == nil || t.options.StderrLogFile == nil {
		return nil
	}
	logPath := *t.options.StderrLogFile
	if logPath == "" {
		// Use default location.
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.logger.Warn("readStderr: could not determine home directory, stderr logging disabled",
				zap.Error(err))
			return nil
		}
		logPath = fmt.Sprintf("%s/.claude/agents_server/cli_stderr.log", homeDir)
	}

	// Create parent directory if it doesn't exist.
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr,
			"[SDK] Failed to create stderr log directory %s: %v\n"+
				"Stderr file logging disabled. To fix, create directory:\n"+
				"  mkdir -p %s\n",
			logDir, err, logDir)
		return nil
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"[SDK] Failed to open stderr log file %s: %v\n"+
				"Stderr file logging disabled. Possible fixes:\n"+
				"  1. Ensure directory exists: mkdir -p %s\n"+
				"  2. Check file permissions: chmod 644 %s\n"+
				"  3. Use custom path: opts.WithCustomStderrLogFile(\"/path/to/file.log\")\n",
			logPath, err, logDir, logPath)
		return nil
	}
	t.logger.Debug("stderr file logging enabled", zap.String("path", logPath))
	return logFile
}

// closeStderrOnCancel closes stderr when ctx is canceled, unblocking a Read()
// that would otherwise hang if the subprocess never closes stderr (Bug C16). It
// returns without closing when the read loop finishes first (readDone closed).
func (t *SubprocessCLITransport) closeStderrOnCancel(ctx context.Context, stderr io.ReadCloser, readDone <-chan struct{}) {
	select {
	case <-ctx.Done():
		_ = stderr.Close()
	case <-readDone:
		// Read loop finished normally — nothing to do
	}
}

func (t *SubprocessCLITransport) consumeStderrChunk(tail *ringBuffer, partial, chunk []byte, emit func([]byte)) []byte {
	for len(chunk) > 0 {
		newline := bytes.IndexByte(chunk, '\n')
		if newline >= 0 {
			partial = appendStderrPartial(tail, partial, chunk[:newline], emit)
			if len(partial) > 0 {
				emit(partial)
			}
			partial = partial[:0]
			chunk = chunk[newline+1:]
			continue
		}

		return appendStderrPartial(tail, partial, chunk, emit)
	}
	return partial
}

func appendStderrPartial(tail *ringBuffer, partial, data []byte, emit func([]byte)) []byte {
	for len(data) > 0 {
		room := StderrRingSize - len(partial)
		if room == 0 {
			emit(partial)
			partial = partial[:0]
			room = StderrRingSize
		}
		if len(data) <= room {
			captureStderrTail(tail, data)
			return append(partial, data...)
		}

		piece := data[:room]
		captureStderrTail(tail, piece)
		partial = append(partial, piece...)
		emit(partial)
		partial = partial[:0]
		data = data[room:]
	}
	return partial
}

func (t *SubprocessCLITransport) emitStderrFragment(logFile *os.File, fragment []byte) {
	if len(fragment) == 0 {
		return
	}
	if fragment[len(fragment)-1] == '\r' {
		fragment = fragment[:len(fragment)-1]
		if len(fragment) == 0 {
			return
		}
	}

	stderrText := string(fragment)

	// Write to log file if enabled and file is open
	if logFile != nil {
		_, _ = fmt.Fprintf(logFile, "[Claude CLI stderr]: %s\n", stderrText)
		_ = logFile.Sync() // Flush to disk immediately
	}

	// Call stderr callback if configured (for runtime control)
	if t.options != nil && t.options.Stderr != nil {
		t.options.Stderr(stderrText)
	}

	// Parse known error patterns and create typed errors
	t.parseStderrError(stderrText)
}

// parseStderrError parses stderr text for known error patterns and stores typed errors.
func (t *SubprocessCLITransport) parseStderrError(stderrText string) {
	// Check for "No conversation found with session ID:" error
	if matched, sessionID := extractSessionNotFoundError(stderrText); matched {
		// Create typed error
		err := types.NewSessionNotFoundError(
			sessionID,
			"Claude CLI could not find this conversation. It may have been deleted or the CLI was reinstalled.",
		)

		// Store error for retrieval
		t.OnError(err)

		// Log it
		t.logger.Error("Claude session not found", zap.String("session_id", sessionID))
	}
}

// extractSessionNotFoundError checks if the stderr text contains a session not found error.
// Returns (matched, id) — (true, id) when the CLI emitted the
// "No conversation found with session ID: <uuid>" diagnostic, (false, "") otherwise.
func extractSessionNotFoundError(stderrText string) (matched bool, id string) {
	const pattern = "No conversation found with session ID:"

	if idx := findSubstring(stderrText, pattern); idx >= 0 {
		// Extract session ID after the pattern
		sessionIDStart := idx + len(pattern)
		if sessionIDStart < len(stderrText) {
			// Trim whitespace and extract the session ID
			remaining := stderrText[sessionIDStart:]
			sessionID := trimWhitespace(remaining)
			// Session ID is the first token (UUID format)
			if sessionID != "" {
				// Take everything up to the first whitespace or end of string
				endIdx := 0
				for endIdx < len(sessionID) && !isWhitespace(rune(sessionID[endIdx])) {
					endIdx++
				}
				sessionID = sessionID[:endIdx]
				return true, sessionID
			}
		}
	}

	return false, ""
}

// Helper functions for string parsing

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func trimWhitespace(s string) string {
	start := 0
	for start < len(s) && isWhitespace(rune(s[start])) {
		start++
	}
	end := len(s)
	for end > start && isWhitespace(rune(s[end-1])) {
		end--
	}
	return s[start:end]
}

func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}
