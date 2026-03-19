package types

import "encoding/json"

// SDKSessionInfo represents metadata for a Claude Code session on disk.
type SDKSessionInfo struct {
	SessionID    string `json:"sessionId"`
	Summary      string `json:"summary"`
	LastModified int64  `json:"lastModified"`
	FileSize     int64  `json:"fileSize,omitempty"`
	CustomTitle  string `json:"customTitle,omitempty"`
	FirstPrompt  string `json:"firstPrompt,omitempty"`
	GitBranch    string `json:"gitBranch,omitempty"`
	CWD          string `json:"cwd,omitempty"`
	Tag          string `json:"tag,omitempty"`
	CreatedAt    int64  `json:"createdAt,omitempty"`
}

// SessionMessage represents a message from session history.
type SessionMessage struct {
	Type            string          `json:"type"`
	UUID            string          `json:"uuid"`
	SessionID       string          `json:"sessionId"`
	Message         json.RawMessage `json:"message"`
	ParentToolUseID *string         `json:"parentToolUseId,omitempty"`
}

// ListSessionsOptions configures the ListSessions function.
type ListSessionsOptions struct {
	Dir              string `json:"dir,omitempty"`
	Limit            int    `json:"limit,omitempty"`
	Offset           int    `json:"offset,omitempty"`
	IncludeWorktrees bool   `json:"includeWorktrees,omitempty"`
}

// GetSessionMessagesOptions configures the GetSessionMessages function.
type GetSessionMessagesOptions struct {
	Dir    string `json:"dir,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}
