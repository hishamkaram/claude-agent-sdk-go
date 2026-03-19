package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/transport"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// ListSessions lists all Claude Code sessions, optionally filtered by project directory.
// This spawns a standalone CLI process and does not require an active Client connection.
func ListSessions(ctx context.Context, opts *types.ListSessionsOptions) ([]types.SDKSessionInfo, error) {
	cliPath, err := transport.FindCLI()
	if err != nil {
		return nil, fmt.Errorf("ListSessions: %w", err)
	}

	args := []string{"sessions", "list", "--output-format", "json"}
	if opts != nil {
		if opts.Dir != "" {
			args = append(args, "--dir", opts.Dir)
		}
		if opts.Limit > 0 {
			args = append(args, "--limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset > 0 {
			args = append(args, "--offset", strconv.Itoa(opts.Offset))
		}
		if opts.IncludeWorktrees {
			args = append(args, "--include-worktrees")
		}
	}

	cmd := exec.CommandContext(ctx, cliPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("ListSessions: %w", ctx.Err())
		}
		return nil, fmt.Errorf("ListSessions: CLI command failed: %w", err)
	}

	var sessions []types.SDKSessionInfo
	if err := json.Unmarshal(output, &sessions); err != nil {
		return nil, fmt.Errorf("ListSessions: failed to parse CLI output: %w", err)
	}

	return sessions, nil
}

// GetSessionMessages retrieves the message history for a specific session.
// This spawns a standalone CLI process and does not require an active Client connection.
func GetSessionMessages(ctx context.Context, sessionID string, opts *types.GetSessionMessagesOptions) ([]types.SessionMessage, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("GetSessionMessages: sessionID: %w", types.ErrEmptyParameter)
	}

	cliPath, err := transport.FindCLI()
	if err != nil {
		return nil, fmt.Errorf("GetSessionMessages: %w", err)
	}

	args := []string{"sessions", "messages", sessionID, "--output-format", "json"}
	if opts != nil {
		if opts.Dir != "" {
			args = append(args, "--dir", opts.Dir)
		}
		if opts.Limit > 0 {
			args = append(args, "--limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset > 0 {
			args = append(args, "--offset", strconv.Itoa(opts.Offset))
		}
	}

	cmd := exec.CommandContext(ctx, cliPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("GetSessionMessages: %w", ctx.Err())
		}
		return nil, fmt.Errorf("GetSessionMessages: CLI command failed: %w", err)
	}

	var messages []types.SessionMessage
	if err := json.Unmarshal(output, &messages); err != nil {
		return nil, fmt.Errorf("GetSessionMessages: failed to parse CLI output: %w", err)
	}

	return messages, nil
}
