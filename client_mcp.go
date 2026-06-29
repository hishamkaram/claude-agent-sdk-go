package claude

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// MCPServerStatus requests the status of all MCP server connections.
func (c *Client) MCPServerStatus(ctx context.Context) ([]types.McpServerStatusInfo, error) {
	q, err := c.activeQuery("Client.MCPServerStatus: not connected - call Connect() first")
	if err != nil {
		return nil, err
	}
	resp, err := sendControlResponse(ctx, q, map[string]interface{}{"subtype": "mcp_status"}, "Client.MCPServerStatus")
	if err != nil {
		return nil, err
	}
	serversRaw, ok := resp["servers"]
	if !ok {
		return nil, nil
	}

	var servers []types.McpServerStatusInfo
	if err := decodeControlResponse(serversRaw, &servers); err != nil {
		return nil, fmt.Errorf("Client.MCPServerStatus: %w", err)
	}
	return servers, nil
}

// ReconnectMCPServer reconnects a disconnected MCP server by name.
func (c *Client) ReconnectMCPServer(ctx context.Context, serverName string) error {
	if serverName == "" {
		return fmt.Errorf("Client.ReconnectMCPServer: serverName: %w", types.ErrEmptyParameter)
	}
	q, err := c.activeQuery("Client.ReconnectMCPServer: not connected - call Connect() first")
	if err != nil {
		return err
	}
	return sendControlNoResponse(
		ctx,
		q,
		map[string]interface{}{"subtype": "mcp_reconnect", "serverName": serverName},
		"Client.ReconnectMCPServer",
	)
}

// ToggleMCPServer enables or disables an MCP server by name.
func (c *Client) ToggleMCPServer(ctx context.Context, serverName string, enabled bool) error {
	if serverName == "" {
		return fmt.Errorf("Client.ToggleMCPServer: serverName: %w", types.ErrEmptyParameter)
	}
	q, err := c.activeQuery("Client.ToggleMCPServer: not connected - call Connect() first")
	if err != nil {
		return err
	}
	return sendControlNoResponse(
		ctx,
		q,
		map[string]interface{}{"subtype": "mcp_toggle", "serverName": serverName, "enabled": enabled},
		"Client.ToggleMCPServer",
	)
}

// SetMCPServers replaces the set of dynamically managed MCP servers.
func (c *Client) SetMCPServers(ctx context.Context, servers map[string]interface{}) (*types.McpSetServersResult, error) {
	if servers == nil {
		return nil, fmt.Errorf("Client.SetMCPServers: servers: %w", types.ErrEmptyParameter)
	}
	q, err := c.activeQuery("Client.SetMCPServers: not connected - call Connect() first")
	if err != nil {
		return nil, err
	}
	resp, err := sendControlResponse(
		ctx,
		q,
		map[string]interface{}{"subtype": "mcp_set_servers", "servers": servers},
		"Client.SetMCPServers",
	)
	if err != nil {
		return nil, err
	}

	var result types.McpSetServersResult
	if err := decodeControlResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("Client.SetMCPServers: %w", err)
	}
	return &result, nil
}

// RewindFiles restores files to their state at a specific user message checkpoint.
func (c *Client) RewindFiles(ctx context.Context, userMessageID string, dryRun bool) (*types.RewindFilesResult, error) {
	if userMessageID == "" {
		return nil, fmt.Errorf("Client.RewindFiles: userMessageID: %w", types.ErrEmptyParameter)
	}
	q, err := c.activeQuery("Client.RewindFiles: not connected - call Connect() first")
	if err != nil {
		return nil, err
	}
	resp, err := sendControlResponse(
		ctx,
		q,
		map[string]interface{}{
			"subtype":         "rewind_files",
			"user_message_id": userMessageID,
			"dry_run":         dryRun,
		},
		"Client.RewindFiles",
	)
	if err != nil {
		return nil, err
	}

	var result types.RewindFilesResult
	if err := decodeControlResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("Client.RewindFiles: %w", err)
	}
	return &result, nil
}

// GetContextUsage retrieves the current token usage breakdown for the session context.
func (c *Client) GetContextUsage(ctx context.Context) (*types.ContextUsage, error) {
	q, err := c.activeQuery("Client.GetContextUsage: not connected - call Connect() first")
	if err != nil {
		return nil, err
	}
	resp, err := sendControlResponse(
		ctx,
		q,
		map[string]interface{}{"subtype": "get_context_usage"},
		"Client.GetContextUsage",
	)
	if err != nil {
		return nil, err
	}

	var usage types.ContextUsage
	if err := decodeControlResponse(resp, &usage); err != nil {
		return nil, fmt.Errorf("Client.GetContextUsage: %w", err)
	}
	return &usage, nil
}

// GetSettings retrieves the current applied settings for the session.
func (c *Client) GetSettings(ctx context.Context) (*types.SettingsResult, error) {
	q, err := c.activeQuery("Client.GetSettings: not connected - call Connect() first")
	if err != nil {
		return nil, err
	}
	resp, err := sendControlResponse(
		ctx,
		q,
		map[string]interface{}{"subtype": "get_settings"},
		"Client.GetSettings",
	)
	if err != nil {
		return nil, err
	}

	var settings types.SettingsResult
	if err := decodeControlResponse(resp, &settings); err != nil {
		return nil, fmt.Errorf("Client.GetSettings: %w", err)
	}
	return &settings, nil
}

func sendControlResponse(
	ctx context.Context,
	q interface {
		SendControlMessage(context.Context, map[string]interface{}) (map[string]interface{}, error)
	},
	req map[string]interface{},
	method string,
) (map[string]interface{}, error) {
	resp, err := q.SendControlMessage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	return resp, nil
}

func decodeControlResponse(raw, out interface{}) error {
	data, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	return nil
}
