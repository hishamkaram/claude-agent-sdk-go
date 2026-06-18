package claude

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
	"golang.org/x/text/unicode/norm"
)

const (
	defaultTranscriptMaxLineSize = 16 * 1024 * 1024
	maxClaudeProjectKeyLen       = 255
	claudeProjectHashLen         = 16
)

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// LocalTranscriptBackend reads Claude Code JSONL transcripts from ~/.claude/projects.
type LocalTranscriptBackend struct {
	// ConfigDir overrides CLAUDE_CONFIG_DIR. When empty, CLAUDE_CONFIG_DIR is honored,
	// then ~/.claude is used.
	ConfigDir string

	// ProjectsDir overrides ConfigDir/projects. Tests can point this at a fixture tree.
	ProjectsDir string

	// MaxLineSize bounds JSONL scanner memory. Zero uses a safe default.
	MaxLineSize int
}

// NewLocalTranscriptBackend returns the default read-only transcript backend.
func NewLocalTranscriptBackend() *LocalTranscriptBackend {
	return &LocalTranscriptBackend{}
}

func (b *LocalTranscriptBackend) ListSessions(ctx context.Context, opts *types.ListSessionsOptions) ([]types.SDKSessionInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	paths, err := b.sessionTranscriptPaths(ctx, listSessionsDir(opts))
	if err != nil {
		return nil, err
	}

	infos := make([]types.SDKSessionInfo, 0, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		sessionID := strings.TrimSuffix(filepath.Base(path), ".jsonl")
		if !isValidUUID(sessionID) {
			continue
		}
		info, err := b.readSessionInfo(ctx, path, sessionID)
		if err != nil {
			return nil, err
		}
		infos = append(infos, *info)
	}

	sort.SliceStable(infos, func(i, j int) bool {
		return infos[i].LastModified > infos[j].LastModified
	})
	return applySessionWindow(infos, listSessionsOffset(opts), listSessionsLimit(opts)), nil
}

func (b *LocalTranscriptBackend) GetSessionInfo(ctx context.Context, sessionID string, opts *types.GetSessionInfoOptions) (*types.SDKSessionInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !isValidUUID(sessionID) {
		return nil, types.ErrInvalidSessionID
	}
	path, err := b.findSessionTranscript(ctx, sessionID, getSessionInfoDir(opts))
	if err != nil {
		return nil, err
	}
	return b.readSessionInfo(ctx, path, sessionID)
}

func (b *LocalTranscriptBackend) GetSessionMessages(ctx context.Context, sessionID string, opts *types.GetSessionMessagesOptions) ([]types.SessionMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !isValidUUID(sessionID) {
		return nil, types.ErrInvalidSessionID
	}
	path, err := b.findSessionTranscript(ctx, sessionID, getSessionMessagesDir(opts))
	if err != nil {
		return nil, err
	}
	entries, err := b.readTranscriptEntries(ctx, path, sessionID)
	if err != nil {
		return nil, err
	}
	chain, err := buildParentChain(entries)
	if err != nil {
		return nil, err
	}
	messages := make([]types.SessionMessage, 0, len(chain))
	for i := range chain {
		messages = append(messages, types.SessionMessage{
			Type:      chain[i].Type,
			UUID:      chain[i].UUID,
			SessionID: sessionID,
			Message:   cloneRaw(chain[i].Message),
		})
	}
	return applyMessageWindow(messages, getSessionMessagesOffset(opts), getSessionMessagesLimit(opts)), nil
}

func (b *LocalTranscriptBackend) ListSubagents(ctx context.Context, sessionID string, opts *types.ListSubagentsOptions) ([]types.SubagentInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !isValidUUID(sessionID) {
		return nil, types.ErrInvalidSessionID
	}
	parentPath, err := b.findSessionTranscript(ctx, sessionID, listSubagentsDir(opts))
	if err != nil {
		return nil, err
	}
	projectDir := filepath.Dir(parentPath)
	var out []types.SubagentInfo
	err = filepath.WalkDir(projectDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if d.IsDir() || path == parentPath || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		rel, err := filepath.Rel(projectDir, path)
		if err != nil {
			return err
		}
		if rel == filepath.Base(rel) {
			return nil
		}
		id := strings.TrimSuffix(filepath.ToSlash(rel), ".jsonl")
		out = append(out, types.SubagentInfo{
			ID:              id,
			Name:            filepath.Base(id),
			ParentSessionID: sessionID,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return applySubagentWindow(out, listSubagentsOffset(opts), listSubagentsLimit(opts)), nil
}

func (b *LocalTranscriptBackend) GetSubagentMessages(ctx context.Context, sessionID, subagentID string, opts *types.GetSubagentMessagesOptions) ([]types.SessionMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if sessionID == "" {
		sessionID = getSubagentMessagesParentSessionID(opts)
	}
	if !isValidUUID(sessionID) {
		return nil, types.ErrInvalidSessionID
	}
	if !isValidSubagentSubpath(subagentID) {
		return nil, types.ErrInvalidSessionID
	}
	parentPath, err := b.findSessionTranscript(ctx, sessionID, getSubagentMessagesDir(opts))
	if err != nil {
		return nil, err
	}
	projectDir := filepath.Dir(parentPath)
	path := filepath.Join(projectDir, filepath.FromSlash(subagentID)+".jsonl")
	if pathErr := ensurePathUnder(projectDir, path); pathErr != nil {
		return nil, pathErr
	}
	entries, err := b.readTranscriptEntries(ctx, path, sessionID)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, types.ErrSessionNotFound
		}
		return nil, err
	}
	chain, err := buildParentChain(entries)
	if err != nil {
		return nil, err
	}
	messages := make([]types.SessionMessage, 0, len(chain))
	for i := range chain {
		messages = append(messages, types.SessionMessage{
			Type:      chain[i].Type,
			UUID:      chain[i].UUID,
			SessionID: sessionID,
			Message:   cloneRaw(chain[i].Message),
		})
	}
	return applyMessageWindow(messages, getSubagentMessagesOffset(opts), getSubagentMessagesLimit(opts)), nil
}

func isValidSubagentSubpath(value string) bool {
	if value == "" || strings.Contains(value, `\`) || strings.HasPrefix(value, "/") {
		return false
	}
	clean := filepath.Clean(filepath.FromSlash(value))
	if clean == "." || clean != filepath.FromSlash(value) {
		return false
	}
	for _, part := range strings.Split(filepath.ToSlash(clean), "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

// ClaudeProjectKey returns the Claude Code projects-directory key for cwd.
func ClaudeProjectKey(cwd string) (string, error) {
	if strings.TrimSpace(cwd) == "" {
		return "", fmt.Errorf("cwd is required")
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}
	if resolved, realErr := filepath.EvalSymlinks(abs); realErr == nil {
		abs = resolved
	}
	normalized := norm.NFC.String(filepath.Clean(abs))
	var b strings.Builder
	b.Grow(len(normalized))
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	key := b.String()
	if len(key) <= maxClaudeProjectKeyLen {
		return key, nil
	}
	sum := sha256.Sum256([]byte(normalized))
	suffix := "-" + hex.EncodeToString(sum[:])[:claudeProjectHashLen]
	return key[:maxClaudeProjectKeyLen-len(suffix)] + suffix, nil
}

type transcriptEntry struct {
	Type        string          `json:"type"`
	UUID        string          `json:"uuid"`
	ParentUUID  *string         `json:"parentUuid"`
	SessionID   string          `json:"sessionId"`
	Message     json.RawMessage `json:"message"`
	IsMeta      bool            `json:"isMeta"`
	IsSidechain bool            `json:"isSidechain"`
	UserType    string          `json:"userType"`
	CWD         string          `json:"cwd"`
	GitBranch   string          `json:"gitBranch"`
	Timestamp   string          `json:"timestamp"`
	Summary     string          `json:"summary"`
	index       int
}

func (b *LocalTranscriptBackend) projectsDir() (string, error) {
	if b != nil && b.ProjectsDir != "" {
		return filepath.Abs(b.ProjectsDir)
	}
	configDir := ""
	if b != nil {
		configDir = b.ConfigDir
	}
	if configDir == "" {
		configDir = os.Getenv("CLAUDE_CONFIG_DIR")
	}
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".claude")
	}
	abs, err := filepath.Abs(configDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(abs, "projects"), nil
}

func (b *LocalTranscriptBackend) projectDirForCWD(cwd string) (string, error) {
	projectsDir, err := b.projectsDir()
	if err != nil {
		return "", err
	}
	key, err := ClaudeProjectKey(cwd)
	if err != nil {
		return "", err
	}
	path := filepath.Join(projectsDir, key)
	if err := ensurePathUnder(projectsDir, path); err != nil {
		return "", err
	}
	return path, nil
}

func (b *LocalTranscriptBackend) sessionTranscriptPaths(ctx context.Context, cwd string) ([]string, error) {
	projectsDir, err := b.projectsDir()
	if err != nil {
		return nil, err
	}
	if cwd != "" {
		projectDir, cwdErr := b.projectDirForCWD(cwd)
		if cwdErr != nil {
			return nil, cwdErr
		}
		return listJSONLFiles(ctx, projectsDir, projectDir)
	}

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.IsDir() {
			continue
		}
		projectDir := filepath.Join(projectsDir, entry.Name())
		projectPaths, err := listJSONLFiles(ctx, projectsDir, projectDir)
		if err != nil {
			return nil, err
		}
		paths = append(paths, projectPaths...)
	}
	return paths, nil
}

func (b *LocalTranscriptBackend) findSessionTranscript(ctx context.Context, sessionID, cwd string) (string, error) {
	paths, err := b.sessionTranscriptPaths(ctx, cwd)
	if err != nil {
		return "", err
	}
	want := sessionID + ".jsonl"
	for _, path := range paths {
		if filepath.Base(path) == want {
			return path, nil
		}
	}
	return "", types.ErrSessionNotFound
}

func (b *LocalTranscriptBackend) readSessionInfo(ctx context.Context, path, sessionID string) (*types.SDKSessionInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	info := &types.SDKSessionInfo{
		SessionID:    sessionID,
		LastModified: stat.ModTime().UnixMilli(),
		FileSize:     stat.Size(),
	}

	err = b.scanTranscript(path, func(lineNo int, raw []byte) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		var entry transcriptEntry
		if unmarshalErr := json.Unmarshal(raw, &entry); unmarshalErr != nil {
			return malformedTranscript(path, lineNo, unmarshalErr)
		}
		if entry.CWD != "" && info.CWD == "" {
			info.CWD = entry.CWD
		}
		if entry.GitBranch != "" && info.GitBranch == "" {
			info.GitBranch = entry.GitBranch
		}
		if entry.Summary != "" {
			info.Summary = entry.Summary
		}
		if entry.Timestamp != "" {
			if millis, ok := parseTranscriptTimeMillis(entry.Timestamp); ok {
				if info.CreatedAt == 0 || millis < info.CreatedAt {
					info.CreatedAt = millis
				}
			}
		}
		if info.FirstPrompt == "" && isValidHistoryEntry(entry, sessionID) && entry.Type == "user" {
			info.FirstPrompt = extractClaudeMessageText(entry.Message)
			if info.Summary == "" {
				info.Summary = info.FirstPrompt
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if info.CreatedAt == 0 {
		info.CreatedAt = stat.ModTime().UnixMilli()
	}
	return info, nil
}

func (b *LocalTranscriptBackend) readTranscriptEntries(ctx context.Context, path, sessionID string) ([]transcriptEntry, error) {
	var entries []transcriptEntry
	err := b.scanTranscript(path, func(lineNo int, raw []byte) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		var entry transcriptEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			return malformedTranscript(path, lineNo, err)
		}
		if !isValidHistoryEntry(entry, sessionID) {
			return nil
		}
		entry.index = len(entries)
		entry.Message = cloneRaw(entry.Message)
		entries = append(entries, entry)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (b *LocalTranscriptBackend) scanTranscript(path string, visit func(lineNo int, raw []byte) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	maxLineSize := defaultTranscriptMaxLineSize
	if b != nil && b.MaxLineSize > 0 {
		maxLineSize = b.MaxLineSize
	}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), maxLineSize)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		cp := cloneRaw(line)
		if err := visit(lineNo, cp); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return malformedTranscript(path, lineNo+1, err)
	}
	return nil
}

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
