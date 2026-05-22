package claude

import (
	"context"
	"fmt"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// SessionKey identifies one mirrored session in a SessionStore.
type SessionKey struct {
	SessionID string `json:"sessionId"`
	Dir       string `json:"dir,omitempty"`
}

// SessionStoreEntry is the full mirrored transcript entry for one session.
type SessionStoreEntry struct {
	Key          SessionKey             `json:"key"`
	SessionID    string                 `json:"sessionId,omitempty"`
	Summary      string                 `json:"summary,omitempty"`
	LastModified int64                  `json:"lastModified,omitempty"`
	FileSize     int64                  `json:"fileSize,omitempty"`
	CustomTitle  string                 `json:"customTitle,omitempty"`
	FirstPrompt  string                 `json:"firstPrompt,omitempty"`
	GitBranch    string                 `json:"gitBranch,omitempty"`
	CWD          string                 `json:"cwd,omitempty"`
	Tag          string                 `json:"tag,omitempty"`
	CreatedAt    int64                  `json:"createdAt,omitempty"`
	Messages     []types.SessionMessage `json:"messages,omitempty"`
}

// SessionStoreListEntry is a list-optimized mirrored session entry.
type SessionStoreListEntry struct {
	SessionID    string `json:"sessionId"`
	Summary      string `json:"summary,omitempty"`
	LastModified int64  `json:"lastModified,omitempty"`
	FileSize     int64  `json:"fileSize,omitempty"`
	CustomTitle  string `json:"customTitle,omitempty"`
	FirstPrompt  string `json:"firstPrompt,omitempty"`
	GitBranch    string `json:"gitBranch,omitempty"`
	CWD          string `json:"cwd,omitempty"`
	Tag          string `json:"tag,omitempty"`
	CreatedAt    int64  `json:"createdAt,omitempty"`
}

// SessionSummaryEntry is a summary-only mirrored session entry.
type SessionSummaryEntry struct {
	SessionID    string `json:"sessionId"`
	Summary      string `json:"summary,omitempty"`
	LastModified int64  `json:"lastModified,omitempty"`
	CWD          string `json:"cwd,omitempty"`
	CreatedAt    int64  `json:"createdAt,omitempty"`
}

// SessionStore loads mirrored session transcripts.
type SessionStore interface {
	Load(ctx context.Context, key SessionKey) (*SessionStoreEntry, error)
}

// SessionStoreLister is implemented by stores that can list full session metadata efficiently.
type SessionStoreLister interface {
	ListSessions(ctx context.Context, opts *types.ListSessionsOptions) ([]SessionStoreListEntry, error)
}

// SessionStoreSummaryLister is implemented by stores that can list session summaries efficiently.
type SessionStoreSummaryLister interface {
	ListSessionSummaries(ctx context.Context, opts *types.ListSessionsOptions) ([]SessionSummaryEntry, error)
}

// SessionStoreBackend adapts an official-style SessionStore to HistoryBackend.
type SessionStoreBackend struct {
	Store SessionStore
}

var _ HistoryBackend = (*SessionStoreBackend)(nil)

// NewSessionStoreBackend creates a history backend over a mirrored SessionStore.
func NewSessionStoreBackend(store SessionStore) (*SessionStoreBackend, error) {
	if store == nil {
		return nil, fmt.Errorf("NewSessionStoreBackend: store is required")
	}
	return &SessionStoreBackend{Store: store}, nil
}

func (b *SessionStoreBackend) ListSessions(ctx context.Context, opts *types.ListSessionsOptions) ([]types.SDKSessionInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if b == nil || b.Store == nil {
		return nil, types.ErrSessionHistoryUnsupported
	}
	if lister, ok := b.Store.(SessionStoreLister); ok {
		entries, err := lister.ListSessions(ctx, opts)
		if err != nil {
			return nil, err
		}
		infos := make([]types.SDKSessionInfo, 0, len(entries))
		for _, entry := range entries {
			if !isValidUUID(entry.SessionID) {
				continue
			}
			infos = append(infos, sessionStoreListEntryInfo(entry))
		}
		return infos, nil
	}
	if lister, ok := b.Store.(SessionStoreSummaryLister); ok {
		entries, err := lister.ListSessionSummaries(ctx, opts)
		if err != nil {
			return nil, err
		}
		infos := make([]types.SDKSessionInfo, 0, len(entries))
		for _, entry := range entries {
			if !isValidUUID(entry.SessionID) {
				continue
			}
			infos = append(infos, types.SDKSessionInfo{
				SessionID:    entry.SessionID,
				Summary:      entry.Summary,
				LastModified: entry.LastModified,
				CWD:          entry.CWD,
				CreatedAt:    entry.CreatedAt,
			})
		}
		return infos, nil
	}
	return nil, types.ErrSessionHistoryUnsupported
}

func (b *SessionStoreBackend) GetSessionInfo(ctx context.Context, sessionID string, opts *types.GetSessionInfoOptions) (*types.SDKSessionInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !isValidUUID(sessionID) {
		return nil, types.ErrInvalidSessionID
	}
	entry, err := b.load(ctx, SessionKey{SessionID: sessionID, Dir: getSessionInfoDir(opts)})
	if err != nil {
		return nil, err
	}
	info := sessionStoreEntryInfo(*entry)
	return &info, nil
}

func (b *SessionStoreBackend) GetSessionMessages(ctx context.Context, sessionID string, opts *types.GetSessionMessagesOptions) ([]types.SessionMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !isValidUUID(sessionID) {
		return nil, types.ErrInvalidSessionID
	}
	entry, err := b.load(ctx, SessionKey{SessionID: sessionID, Dir: getSessionMessagesDir(opts)})
	if err != nil {
		return nil, err
	}
	messages := make([]types.SessionMessage, len(entry.Messages))
	copy(messages, entry.Messages)
	return applyMessageWindow(messages, getSessionMessagesOffset(opts), getSessionMessagesLimit(opts)), nil
}

func (b *SessionStoreBackend) ListSubagents(ctx context.Context, sessionID string, opts *types.ListSubagentsOptions) ([]types.SubagentInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !isValidUUID(sessionID) {
		return nil, types.ErrInvalidSessionID
	}
	return nil, types.ErrSessionHistoryUnsupported
}

func (b *SessionStoreBackend) GetSubagentMessages(ctx context.Context, sessionID string, subagentID string, opts *types.GetSubagentMessagesOptions) ([]types.SessionMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, types.ErrSessionHistoryUnsupported
}

func (b *SessionStoreBackend) load(ctx context.Context, key SessionKey) (*SessionStoreEntry, error) {
	if b == nil || b.Store == nil {
		return nil, types.ErrSessionHistoryUnsupported
	}
	entry, err := b.Store.Load(ctx, key)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, types.ErrSessionNotFound
	}
	if entry.SessionID == "" {
		entry.SessionID = entry.Key.SessionID
	}
	if entry.SessionID == "" {
		entry.SessionID = key.SessionID
	}
	if !isValidUUID(entry.SessionID) {
		return nil, types.ErrInvalidSessionID
	}
	return entry, nil
}

func sessionStoreEntryInfo(entry SessionStoreEntry) types.SDKSessionInfo {
	sessionID := entry.SessionID
	if sessionID == "" {
		sessionID = entry.Key.SessionID
	}
	return types.SDKSessionInfo{
		SessionID:    sessionID,
		Summary:      entry.Summary,
		LastModified: entry.LastModified,
		FileSize:     entry.FileSize,
		CustomTitle:  entry.CustomTitle,
		FirstPrompt:  entry.FirstPrompt,
		GitBranch:    entry.GitBranch,
		CWD:          entry.CWD,
		Tag:          entry.Tag,
		CreatedAt:    entry.CreatedAt,
	}
}

func sessionStoreListEntryInfo(entry SessionStoreListEntry) types.SDKSessionInfo {
	return types.SDKSessionInfo{
		SessionID:    entry.SessionID,
		Summary:      entry.Summary,
		LastModified: entry.LastModified,
		FileSize:     entry.FileSize,
		CustomTitle:  entry.CustomTitle,
		FirstPrompt:  entry.FirstPrompt,
		GitBranch:    entry.GitBranch,
		CWD:          entry.CWD,
		Tag:          entry.Tag,
		CreatedAt:    entry.CreatedAt,
	}
}
