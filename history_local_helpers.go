package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func buildParentChain(entries []transcriptEntry) ([]transcriptEntry, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	byUUID := make(map[string]transcriptEntry, len(entries))
	for i := range entries {
		if _, exists := byUUID[entries[i].UUID]; exists {
			return nil, fmt.Errorf("%w: duplicate uuid %q", types.ErrMalformedTranscript, entries[i].UUID)
		}
		byUUID[entries[i].UUID] = entries[i]
	}

	leaf := entries[len(entries)-1]
	var reverse []transcriptEntry
	seen := make(map[string]bool, len(entries))
	for {
		if seen[leaf.UUID] {
			return nil, fmt.Errorf("%w: parentUuid cycle at %q", types.ErrMalformedTranscript, leaf.UUID)
		}
		seen[leaf.UUID] = true
		reverse = append(reverse, leaf)
		if leaf.ParentUUID == nil || *leaf.ParentUUID == "" {
			break
		}
		parent, ok := byUUID[*leaf.ParentUUID]
		if !ok {
			break
		}
		leaf = parent
	}

	for i, j := 0, len(reverse)-1; i < j; i, j = i+1, j-1 {
		reverse[i], reverse[j] = reverse[j], reverse[i]
	}
	return reverse, nil
}

func isValidHistoryEntry(entry transcriptEntry, sessionID string) bool {
	if entry.Type != "user" && entry.Type != "assistant" {
		return false
	}
	if !isValidUUID(entry.UUID) {
		return false
	}
	if entry.SessionID != "" && entry.SessionID != sessionID {
		return false
	}
	if entry.IsMeta || entry.IsSidechain {
		return false
	}
	userType := strings.ToLower(strings.TrimSpace(entry.UserType))
	if userType == "system" || userType == "team" || userType == "internal" {
		return false
	}
	return len(entry.Message) > 0 && json.Valid(entry.Message)
}

func extractClaudeMessageText(raw json.RawMessage) string {
	var msg struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil || len(msg.Content) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(msg.Content, &text); err == nil {
		return text
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return ""
	}
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func listJSONLFiles(ctx context.Context, projectsDir, dir string) ([]string, error) {
	if err := ensurePathUnder(projectsDir, dir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := ensurePathUnder(projectsDir, path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func ensurePathUnder(base, target string) error {
	baseAbs, err := resolvedPath(base)
	if err != nil {
		return err
	}
	targetAbs, err := resolvedPath(target)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("%w: path escapes Claude projects directory", types.ErrInvalidSessionID)
	}
	return nil
}

func resolvedPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}
	return abs, nil
}

func malformedTranscript(path string, lineNo int, cause error) error {
	return fmt.Errorf("%w: %s:%d: %w", types.ErrMalformedTranscript, path, lineNo, cause)
}

func parseTranscriptTimeMillis(value string) (int64, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UnixMilli(), true
		}
	}
	return 0, false
}

func isValidUUID(value string) bool {
	return uuidPattern.MatchString(value)
}

func cloneRaw(raw []byte) []byte {
	if raw == nil {
		return nil
	}
	cp := make([]byte, len(raw))
	copy(cp, raw)
	return cp
}

func applySessionWindow(values []types.SDKSessionInfo, offset, limit int) []types.SDKSessionInfo {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(values) {
		return []types.SDKSessionInfo{}
	}
	values = values[offset:]
	if limit > 0 && limit < len(values) {
		values = values[:limit]
	}
	return values
}

func applyMessageWindow(values []types.SessionMessage, offset, limit int) []types.SessionMessage {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(values) {
		return []types.SessionMessage{}
	}
	values = values[offset:]
	if limit > 0 && limit < len(values) {
		values = values[:limit]
	}
	return values
}

func applySubagentWindow(values []types.SubagentInfo, offset, limit int) []types.SubagentInfo {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(values) {
		return []types.SubagentInfo{}
	}
	values = values[offset:]
	if limit > 0 && limit < len(values) {
		values = values[:limit]
	}
	return values
}

func listSessionsDir(opts *types.ListSessionsOptions) string {
	if opts == nil {
		return ""
	}
	return opts.Dir
}

func listSessionsLimit(opts *types.ListSessionsOptions) int {
	if opts == nil {
		return 0
	}
	return opts.Limit
}

func listSessionsOffset(opts *types.ListSessionsOptions) int {
	if opts == nil {
		return 0
	}
	return opts.Offset
}

func getSessionInfoDir(opts *types.GetSessionInfoOptions) string {
	if opts == nil {
		return ""
	}
	return opts.Dir
}

func getSessionMessagesDir(opts *types.GetSessionMessagesOptions) string {
	if opts == nil {
		return ""
	}
	return opts.Dir
}

func getSessionMessagesLimit(opts *types.GetSessionMessagesOptions) int {
	if opts == nil {
		return 0
	}
	return opts.Limit
}

func getSessionMessagesOffset(opts *types.GetSessionMessagesOptions) int {
	if opts == nil {
		return 0
	}
	return opts.Offset
}

func listSubagentsDir(opts *types.ListSubagentsOptions) string {
	if opts == nil {
		return ""
	}
	return opts.Dir
}

func listSubagentsLimit(opts *types.ListSubagentsOptions) int {
	if opts == nil {
		return 0
	}
	return opts.Limit
}

func listSubagentsOffset(opts *types.ListSubagentsOptions) int {
	if opts == nil {
		return 0
	}
	return opts.Offset
}

func getSubagentMessagesDir(opts *types.GetSubagentMessagesOptions) string {
	if opts == nil {
		return ""
	}
	return opts.Dir
}

func getSubagentMessagesParentSessionID(opts *types.GetSubagentMessagesOptions) string {
	if opts == nil {
		return ""
	}
	return opts.ParentSessionID
}

func getSubagentMessagesLimit(opts *types.GetSubagentMessagesOptions) int {
	if opts == nil {
		return 0
	}
	return opts.Limit
}

func getSubagentMessagesOffset(opts *types.GetSubagentMessagesOptions) int {
	if opts == nil {
		return 0
	}
	return opts.Offset
}
