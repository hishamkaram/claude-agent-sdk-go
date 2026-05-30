package claude

import (
	"context"
	"fmt"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

type SessionKey = types.SessionKey
type SessionStoreEntry = types.SessionStoreEntry
type SessionStoreListEntry = types.SessionStoreListEntry
type SessionSummaryEntry = types.SessionSummaryEntry
type SessionStore = types.SessionStore
type SessionStoreLister = types.SessionStoreLister
type SessionStoreSummaryLister = types.SessionStoreSummaryLister
type SessionStoreSubkeyLister = types.SessionStoreSubkeyLister
type SessionStoreDeleter = types.SessionStoreDeleter
type UnsupportedSessionStoreOperationError = types.UnsupportedSessionStoreOperationError

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
		return nil, types.NewUnsupportedSessionStoreOperationError("ListSessions")
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
	entry, err := b.load(ctx, sessionStoreKeyFromDir(sessionID, getSessionInfoDir(opts), ""))
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
	entry, err := b.load(ctx, sessionStoreKeyFromDir(sessionID, getSessionMessagesDir(opts), ""))
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
	if lister, ok := b.Store.(SessionStoreSubkeyLister); ok {
		keys, err := lister.ListSubkeys(ctx, sessionStoreKeyFromDir(sessionID, listSubagentsDir(opts), ""))
		if err != nil {
			return nil, err
		}
		out := make([]types.SubagentInfo, 0, len(keys))
		for _, key := range keys {
			if key.Subpath == "" {
				continue
			}
			out = append(out, types.SubagentInfo{
				ID:              key.Subpath,
				Name:            key.Subpath,
				ParentSessionID: sessionID,
			})
		}
		return out, nil
	}
	return nil, types.NewUnsupportedSessionStoreOperationError("ListSubagents")
}

func (b *SessionStoreBackend) GetSubagentMessages(ctx context.Context, sessionID string, subagentID string, opts *types.GetSubagentMessagesOptions) ([]types.SessionMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if subagentID == "" {
		return nil, types.ErrInvalidSessionID
	}
	entry, err := b.load(ctx, sessionStoreKeyFromDir(sessionID, getSubagentMessagesDir(opts), subagentID))
	if err != nil {
		return nil, err
	}
	messages := make([]types.SessionMessage, len(entry.Messages))
	copy(messages, entry.Messages)
	return applyMessageWindow(messages, getSubagentMessagesOffset(opts), getSubagentMessagesLimit(opts)), nil
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

func sessionStoreKeyFromDir(sessionID, dir, subpath string) SessionKey {
	key := SessionKey{SessionID: sessionID, Dir: dir, Subpath: subpath}
	if dir != "" {
		if projectKey, err := ClaudeProjectKey(dir); err == nil {
			key.ProjectKey = projectKey
		}
	}
	return key
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
	// SessionStoreListEntry mirrors SDKSessionInfo field-for-field (only the
	// JSON tags differ), so a direct struct conversion is exact. The compiler
	// enforces the mirror: if either struct's field set, types, or order
	// diverges, this conversion stops compiling and forces an explicit mapping
	// here rather than silently copying the wrong fields.
	return types.SDKSessionInfo(entry)
}
