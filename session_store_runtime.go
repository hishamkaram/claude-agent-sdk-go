package claude

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

type runtimeTranscriptEntry struct {
	Type      string          `json:"type"`
	UUID      string          `json:"uuid,omitempty"`
	SessionID string          `json:"sessionId"`
	Message   json.RawMessage `json:"message,omitempty"`
}

func prepareSessionStoreRuntime(ctx context.Context, options *types.ClaudeAgentOptions, cwd, resumeID string, env map[string]string) (func(), error) {
	if options == nil || options.SessionStore == nil {
		return func() {}, nil
	}
	key, err := resolveSessionStoreKey(options, cwd, resumeID)
	if err != nil {
		return nil, err
	}
	options.SessionStoreKey = &key
	if resumeID == "" {
		return func() {}, nil
	}

	entry, err := options.SessionStore.Load(ctx, key)
	if err != nil {
		if errors.Is(err, types.ErrSessionNotFound) {
			return func() {}, nil
		}
		return nil, fmt.Errorf("session store resume load: %w", err)
	}
	if entry == nil || len(entry.Messages) == 0 {
		return func() {}, nil
	}

	configDir, err := os.MkdirTemp("", "claude-agent-sdk-session-store-*")
	if err != nil {
		return nil, fmt.Errorf("session store config dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(configDir) }
	projectDir := filepath.Join(configDir, "projects", key.ProjectKey)
	if mkErr := os.MkdirAll(projectDir, 0o700); mkErr != nil {
		cleanup()
		return nil, fmt.Errorf("session store project dir: %w", mkErr)
	}
	transcriptPath := filepath.Join(projectDir, key.SessionID+".jsonl")
	f, err := os.OpenFile(transcriptPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("session store transcript open: %w", err)
	}
	for _, msg := range entry.Messages {
		line, err := json.Marshal(runtimeTranscriptEntry{
			Type:      msg.Type,
			UUID:      msg.UUID,
			SessionID: coalesce(msg.SessionID, key.SessionID),
			Message:   cloneRawMessage(msg.Message),
		})
		if err != nil {
			_ = f.Close()
			cleanup()
			return nil, fmt.Errorf("session store transcript marshal: %w", err)
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			_ = f.Close()
			cleanup()
			return nil, fmt.Errorf("session store transcript write: %w", err)
		}
	}
	if err := f.Close(); err != nil {
		cleanup()
		return nil, fmt.Errorf("session store transcript close: %w", err)
	}
	env["CLAUDE_CONFIG_DIR"] = configDir
	return cleanup, nil
}

func resolveSessionStoreKey(options *types.ClaudeAgentOptions, cwd, sessionID string) (types.SessionKey, error) {
	var key types.SessionKey
	if options != nil && options.SessionStoreKey != nil {
		key = *options.SessionStoreKey
	}
	if key.SessionID == "" {
		key.SessionID = sessionID
	}
	if key.Dir == "" {
		key.Dir = cwd
	}
	if key.ProjectKey == "" {
		projectCWD := cwd
		if projectCWD == "" {
			wd, err := os.Getwd()
			if err != nil {
				return key, err
			}
			projectCWD = wd
		}
		projectKey, err := ClaudeProjectKey(projectCWD)
		if err != nil {
			return key, err
		}
		key.ProjectKey = projectKey
	}
	return key, nil
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	cp := make([]byte, len(raw))
	copy(cp, raw)
	return cp
}

func coalesce(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
