package claude

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hishamkaram/claude-agent-sdk-go/internal"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func (c *Client) activeQuery(notConnectedMessage string) (*internal.Query, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected || c.query == nil {
		return nil, types.NewCLIConnectionError(notConnectedMessage)
	}
	return c.query, nil
}

// SetModel changes the model used for subsequent responses.
func (c *Client) SetModel(ctx context.Context, model string) error {
	q, err := c.activeQuery("not connected - call Connect() first")
	if err != nil {
		return err
	}
	req := map[string]interface{}{"subtype": "set_model"}
	if model != "" {
		req["model"] = model
	}
	return sendControlNoResponse(ctx, q, req, "Client.SetModel")
}

// SetPermissionMode changes the permission mode mid-session.
func (c *Client) SetPermissionMode(ctx context.Context, mode types.PermissionMode) error {
	q, err := c.activeQuery("not connected - call Connect() first")
	if err != nil {
		return err
	}
	return sendControlNoResponse(
		ctx,
		q,
		map[string]interface{}{"subtype": "set_permission_mode", "mode": string(mode)},
		"Client.SetPermissionMode",
	)
}

// Interrupt sends an interrupt control request to cancel the active query.
func (c *Client) Interrupt(ctx context.Context) error {
	q, err := c.activeQuery("Client.Interrupt: not connected - call Connect() first")
	if err != nil {
		return err
	}
	return sendControlNoResponse(ctx, q, map[string]interface{}{"subtype": "interrupt"}, "Client.Interrupt")
}

// StreamInput writes user content to the subprocess stdin during an active stream.
func (c *Client) StreamInput(ctx context.Context, content string) error {
	if content == "" {
		return fmt.Errorf("Client.StreamInput: content: %w", types.ErrEmptyParameter)
	}
	if _, err := c.activeQuery("Client.StreamInput: not connected - call Connect() first"); err != nil {
		return err
	}
	return c.writeStreamInput(ctx, content)
}

func (c *Client) writeStreamInput(ctx context.Context, content string) error {
	data, err := json.Marshal(map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"role":    "user",
			"content": content,
		},
		"parent_tool_use_id": nil,
		"session_id":         "default",
	})
	if err != nil {
		return fmt.Errorf("Client.StreamInput: failed to marshal message: %w", err)
	}
	if err := c.transport.Write(ctx, string(data)); err != nil {
		return fmt.Errorf("Client.StreamInput: %w", err)
	}
	return nil
}

// StopTask sends a stop_task control request to cancel a specific background task.
func (c *Client) StopTask(ctx context.Context, taskID string) error {
	if taskID == "" {
		return fmt.Errorf("Client.StopTask: taskID: %w", types.ErrEmptyParameter)
	}
	q, err := c.activeQuery("Client.StopTask: not connected - call Connect() first")
	if err != nil {
		return err
	}
	return sendControlNoResponse(
		ctx,
		q,
		map[string]interface{}{"subtype": "stop_task", "task_id": taskID},
		"Client.StopTask",
	)
}

// ReloadPlugins triggers a reload of all configured plugins in the current session.
func (c *Client) ReloadPlugins(ctx context.Context) error {
	q, err := c.activeQuery("Client.ReloadPlugins: not connected - call Connect() first")
	if err != nil {
		return err
	}
	return sendControlNoResponse(
		ctx,
		q,
		map[string]interface{}{"subtype": "reload_plugins"},
		"Client.ReloadPlugins",
	)
}

// EnableChannel enables channel communication for the current session.
func (c *Client) EnableChannel(ctx context.Context) error {
	q, err := c.activeQuery("Client.EnableChannel: not connected - call Connect() first")
	if err != nil {
		return err
	}
	return sendControlNoResponse(
		ctx,
		q,
		map[string]interface{}{"subtype": "enable_channel"},
		"Client.EnableChannel",
	)
}

func sendControlNoResponse(
	ctx context.Context,
	q *internal.Query,
	req map[string]interface{},
	method string,
) error {
	_, err := q.SendControlMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	return nil
}
