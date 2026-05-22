package claude

import (
	"context"
	"fmt"
	"sync"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// HistoryBackend reads Claude Code session history without starting Claude Code.
type HistoryBackend interface {
	ListSessions(ctx context.Context, opts *types.ListSessionsOptions) ([]types.SDKSessionInfo, error)
	GetSessionInfo(ctx context.Context, sessionID string, opts *types.GetSessionInfoOptions) (*types.SDKSessionInfo, error)
	GetSessionMessages(ctx context.Context, sessionID string, opts *types.GetSessionMessagesOptions) ([]types.SessionMessage, error)
	ListSubagents(ctx context.Context, sessionID string, opts *types.ListSubagentsOptions) ([]types.SubagentInfo, error)
	GetSubagentMessages(ctx context.Context, sessionID string, subagentID string, opts *types.GetSubagentMessagesOptions) ([]types.SessionMessage, error)
}

var (
	historyBackendMu sync.RWMutex
	historyBackend   HistoryBackend = NewLocalTranscriptBackend()
)

// SetHistoryBackend replaces the package-level history backend and returns a restore function.
// Passing nil resets the backend to the read-only local transcript backend.
func SetHistoryBackend(backend HistoryBackend) func() {
	historyBackendMu.Lock()
	previous := historyBackend
	if backend == nil {
		historyBackend = NewLocalTranscriptBackend()
	} else {
		historyBackend = backend
	}
	historyBackendMu.Unlock()

	return func() {
		historyBackendMu.Lock()
		historyBackend = previous
		historyBackendMu.Unlock()
	}
}

func selectedHistoryBackend() HistoryBackend {
	historyBackendMu.RLock()
	backend := historyBackend
	historyBackendMu.RUnlock()
	if backend == nil {
		return NewLocalTranscriptBackend()
	}
	return backend
}

// ListSessions lists Claude Code sessions from read-only transcript storage.
func ListSessions(ctx context.Context, opts *types.ListSessionsOptions) ([]types.SDKSessionInfo, error) {
	sessions, err := selectedHistoryBackend().ListSessions(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("ListSessions: %w", err)
	}
	return sessions, nil
}

// GetSessionMessages retrieves message history for a specific session from read-only transcript storage.
func GetSessionMessages(ctx context.Context, sessionID string, opts *types.GetSessionMessagesOptions) ([]types.SessionMessage, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("GetSessionMessages: sessionID: %w", types.ErrEmptyParameter)
	}
	messages, err := selectedHistoryBackend().GetSessionMessages(ctx, sessionID, opts)
	if err != nil {
		return nil, fmt.Errorf("GetSessionMessages: %w", err)
	}
	return messages, nil
}

// GetSessionInfo retrieves metadata for a specific session from read-only transcript storage.
func GetSessionInfo(ctx context.Context, sessionID string, opts *types.GetSessionInfoOptions) (*types.SDKSessionInfo, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("GetSessionInfo: sessionID: %w", types.ErrEmptyParameter)
	}
	info, err := selectedHistoryBackend().GetSessionInfo(ctx, sessionID, opts)
	if err != nil {
		return nil, fmt.Errorf("GetSessionInfo: %w", err)
	}
	return info, nil
}

// RenameSession is unsupported by the read-only history backend.
func RenameSession(ctx context.Context, sessionID string, title string, opts *types.RenameSessionOptions) error {
	if sessionID == "" {
		return fmt.Errorf("RenameSession: sessionID: %w", types.ErrEmptyParameter)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("RenameSession: %w", err)
	}
	return fmt.Errorf("RenameSession: %w", types.ErrSessionHistoryUnsupported)
}

// TagSession is unsupported by the read-only history backend.
func TagSession(ctx context.Context, sessionID string, tag string, opts *types.TagSessionOptions) error {
	if sessionID == "" {
		return fmt.Errorf("TagSession: sessionID: %w", types.ErrEmptyParameter)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("TagSession: %w", err)
	}
	return fmt.Errorf("TagSession: %w", types.ErrSessionHistoryUnsupported)
}

// ForkSession is unsupported by the read-only history backend.
func ForkSession(ctx context.Context, sessionID string, opts *types.ForkSessionOptions) (*types.ForkSessionResult, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("ForkSession: sessionID: %w", types.ErrEmptyParameter)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("ForkSession: %w", err)
	}
	return nil, fmt.Errorf("ForkSession: %w", types.ErrSessionHistoryUnsupported)
}

// ListSubagents lists subagents if the selected read-only backend supports subagent history.
func ListSubagents(ctx context.Context, sessionID string, opts *types.ListSubagentsOptions) ([]types.SubagentInfo, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("ListSubagents: sessionID: %w", types.ErrEmptyParameter)
	}
	agents, err := selectedHistoryBackend().ListSubagents(ctx, sessionID, opts)
	if err != nil {
		return nil, fmt.Errorf("ListSubagents: %w", err)
	}
	return agents, nil
}

// GetSubagentMessages retrieves subagent messages if the selected read-only backend supports them.
func GetSubagentMessages(ctx context.Context, subagentID string, opts *types.GetSubagentMessagesOptions) ([]types.SessionMessage, error) {
	if subagentID == "" {
		return nil, fmt.Errorf("GetSubagentMessages: subagentID: %w", types.ErrEmptyParameter)
	}
	sessionID := ""
	if opts != nil {
		sessionID = opts.ParentSessionID
	}
	messages, err := selectedHistoryBackend().GetSubagentMessages(ctx, sessionID, subagentID, opts)
	if err != nil {
		return nil, fmt.Errorf("GetSubagentMessages: %w", err)
	}
	return messages, nil
}
