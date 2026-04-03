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
		if opts.IncludeSystemMessages {
			args = append(args, "--include-system-messages")
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

// GetSessionInfo retrieves metadata for a specific session.
// This spawns a standalone CLI process and does not require an active Client connection.
// Wraps: claude sessions info <sessionID> --output-format json
func GetSessionInfo(ctx context.Context, sessionID string, opts *types.GetSessionInfoOptions) (*types.SDKSessionInfo, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("GetSessionInfo: sessionID: %w", types.ErrEmptyParameter)
	}

	cliPath, err := transport.FindCLI()
	if err != nil {
		return nil, fmt.Errorf("GetSessionInfo: %w", err)
	}

	args := []string{"sessions", "info", sessionID, "--output-format", "json"}
	if opts != nil && opts.Dir != "" {
		args = append(args, "--dir", opts.Dir)
	}

	cmd := exec.CommandContext(ctx, cliPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("GetSessionInfo: %w", ctx.Err())
		}
		return nil, fmt.Errorf("GetSessionInfo: CLI command failed: %w", err)
	}

	var info types.SDKSessionInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("GetSessionInfo: failed to parse CLI output: %w", err)
	}

	return &info, nil
}

// RenameSession sets a custom title for a session.
// This spawns a standalone CLI process and does not require an active Client connection.
// Wraps: claude sessions rename <sessionID> <title>
func RenameSession(ctx context.Context, sessionID string, title string, opts *types.RenameSessionOptions) error {
	if sessionID == "" {
		return fmt.Errorf("RenameSession: sessionID: %w", types.ErrEmptyParameter)
	}

	cliPath, err := transport.FindCLI()
	if err != nil {
		return fmt.Errorf("RenameSession: %w", err)
	}

	args := []string{"sessions", "rename", sessionID, title}
	if opts != nil && opts.Dir != "" {
		args = append(args, "--dir", opts.Dir)
	}

	cmd := exec.CommandContext(ctx, cliPath, args...)
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("RenameSession: %w", ctx.Err())
		}
		return fmt.Errorf("RenameSession: CLI command failed: %w", err)
	}

	return nil
}

// TagSession applies or clears a tag on a session.
// This spawns a standalone CLI process and does not require an active Client connection.
// Pass an empty string for tag to clear the existing tag.
// Wraps: claude sessions tag <sessionID> [<tag>]
func TagSession(ctx context.Context, sessionID string, tag string, opts *types.TagSessionOptions) error {
	if sessionID == "" {
		return fmt.Errorf("TagSession: sessionID: %w", types.ErrEmptyParameter)
	}

	cliPath, err := transport.FindCLI()
	if err != nil {
		return fmt.Errorf("TagSession: %w", err)
	}

	args := []string{"sessions", "tag", sessionID}
	if tag != "" {
		args = append(args, tag)
	}
	if opts != nil && opts.Dir != "" {
		args = append(args, "--dir", opts.Dir)
	}

	cmd := exec.CommandContext(ctx, cliPath, args...)
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("TagSession: %w", ctx.Err())
		}
		return fmt.Errorf("TagSession: CLI command failed: %w", err)
	}

	return nil
}

// ForkSession creates a copy of an existing session.
// This spawns a standalone CLI process and does not require an active Client connection.
// Wraps: claude sessions fork <sessionID> --output-format json
func ForkSession(ctx context.Context, sessionID string, opts *types.ForkSessionOptions) (*types.ForkSessionResult, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("ForkSession: sessionID: %w", types.ErrEmptyParameter)
	}

	cliPath, err := transport.FindCLI()
	if err != nil {
		return nil, fmt.Errorf("ForkSession: %w", err)
	}

	args := []string{"sessions", "fork", sessionID, "--output-format", "json"}
	if opts != nil && opts.Dir != "" {
		args = append(args, "--dir", opts.Dir)
	}

	cmd := exec.CommandContext(ctx, cliPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("ForkSession: %w", ctx.Err())
		}
		return nil, fmt.Errorf("ForkSession: CLI command failed: %w", err)
	}

	var result types.ForkSessionResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("ForkSession: failed to parse CLI output: %w", err)
	}

	return &result, nil
}

// ListSubagents lists subagent processes within a session.
// This spawns a standalone CLI process and does not require an active Client connection.
// Wraps: claude sessions subagents <sessionID> --output-format json
func ListSubagents(ctx context.Context, sessionID string, opts *types.ListSubagentsOptions) ([]types.SubagentInfo, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("ListSubagents: sessionID: %w", types.ErrEmptyParameter)
	}

	cliPath, err := transport.FindCLI()
	if err != nil {
		return nil, fmt.Errorf("ListSubagents: %w", err)
	}

	args := []string{"sessions", "subagents", sessionID, "--output-format", "json"}
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
			return nil, fmt.Errorf("ListSubagents: %w", ctx.Err())
		}
		return nil, fmt.Errorf("ListSubagents: CLI command failed: %w", err)
	}

	var agents []types.SubagentInfo
	if err := json.Unmarshal(output, &agents); err != nil {
		return nil, fmt.Errorf("ListSubagents: failed to parse CLI output: %w", err)
	}

	return agents, nil
}

// GetSubagentMessages retrieves the message history for a specific subagent.
// This spawns a standalone CLI process and does not require an active Client connection.
// Wraps: claude sessions subagent-messages <subagentID> --output-format json
func GetSubagentMessages(ctx context.Context, subagentID string, opts *types.GetSubagentMessagesOptions) ([]types.SessionMessage, error) {
	if subagentID == "" {
		return nil, fmt.Errorf("GetSubagentMessages: subagentID: %w", types.ErrEmptyParameter)
	}

	cliPath, err := transport.FindCLI()
	if err != nil {
		return nil, fmt.Errorf("GetSubagentMessages: %w", err)
	}

	args := []string{"sessions", "subagent-messages", subagentID, "--output-format", "json"}
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
			return nil, fmt.Errorf("GetSubagentMessages: %w", ctx.Err())
		}
		return nil, fmt.Errorf("GetSubagentMessages: CLI command failed: %w", err)
	}

	var messages []types.SessionMessage
	if err := json.Unmarshal(output, &messages); err != nil {
		return nil, fmt.Errorf("GetSubagentMessages: failed to parse CLI output: %w", err)
	}

	return messages, nil
}
