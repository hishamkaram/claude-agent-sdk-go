package types

import (
	"context"
	"fmt"
)

// SessionKey identifies one provider-native mirrored transcript stream.
//
// ProjectKey is Claude Code's stable project-directory key. Dir is retained as
// a deprecated cwd-oriented compatibility hint for older callers.
type SessionKey struct {
	ProjectKey string `json:"projectKey,omitempty"`
	SessionID  string `json:"sessionId"`
	Subpath    string `json:"subpath,omitempty"`
	Dir        string `json:"dir,omitempty"` // Deprecated: use ProjectKey.
}

// SessionStoreEntry is the full mirrored transcript entry for one session.
type SessionStoreEntry struct {
	Key          SessionKey       `json:"key"`
	SessionID    string           `json:"sessionId,omitempty"`
	Summary      string           `json:"summary,omitempty"`
	LastModified int64            `json:"lastModified,omitempty"`
	FileSize     int64            `json:"fileSize,omitempty"`
	CustomTitle  string           `json:"customTitle,omitempty"`
	FirstPrompt  string           `json:"firstPrompt,omitempty"`
	GitBranch    string           `json:"gitBranch,omitempty"`
	CWD          string           `json:"cwd,omitempty"`
	Tag          string           `json:"tag,omitempty"`
	CreatedAt    int64            `json:"createdAt,omitempty"`
	Messages     []SessionMessage `json:"messages,omitempty"`
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

// SessionStore mirrors Claude Code transcript entries. Load hydrates resume
// state; Append mirrors runtime transcript entries after Claude Code has
// already durably written its local transcript.
type SessionStore interface {
	Append(ctx context.Context, key SessionKey, entries []SessionMessage) error
	Load(ctx context.Context, key SessionKey) (*SessionStoreEntry, error)
}

// SessionStoreLister is implemented by stores that can list full session metadata efficiently.
type SessionStoreLister interface {
	ListSessions(ctx context.Context, opts *ListSessionsOptions) ([]SessionStoreListEntry, error)
}

// SessionStoreSummaryLister is implemented by stores that can list session summaries efficiently.
type SessionStoreSummaryLister interface {
	ListSessionSummaries(ctx context.Context, opts *ListSessionsOptions) ([]SessionSummaryEntry, error)
}

// SessionStoreSubkeyLister is implemented by stores that expose opaque child
// transcript keys such as subagent transcript streams.
type SessionStoreSubkeyLister interface {
	ListSubkeys(ctx context.Context, key SessionKey) ([]SessionKey, error)
}

// SessionStoreDeleter is implemented by stores that support safe deletion.
type SessionStoreDeleter interface {
	Delete(ctx context.Context, key SessionKey) error
}

// UnsupportedSessionStoreOperationError marks optional SessionStore operations
// that the selected store intentionally does not implement.
type UnsupportedSessionStoreOperationError struct {
	Operation string
}

func (e *UnsupportedSessionStoreOperationError) Error() string {
	if e == nil || e.Operation == "" {
		return ErrSessionHistoryUnsupported.Error()
	}
	return fmt.Sprintf("%s: %v", e.Operation, ErrSessionHistoryUnsupported)
}

func (e *UnsupportedSessionStoreOperationError) Is(target error) bool {
	return target == ErrSessionHistoryUnsupported
}

// NewUnsupportedSessionStoreOperationError returns a typed unsupported error.
func NewUnsupportedSessionStoreOperationError(operation string) error {
	return &UnsupportedSessionStoreOperationError{Operation: operation}
}
