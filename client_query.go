package claude

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// Query sends a plain text prompt to Claude in the current session.
func (c *Client) Query(ctx context.Context, prompt string) error {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return types.NewCLIConnectionError("not connected - call Connect() first")
	}
	c.mu.Unlock()

	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}
	return c.writeUserMessage(ctx, prompt, "Client.Query")
}

// QueryWithContent sends a structured content query (text + images) to Claude.
func (c *Client) QueryWithContent(ctx context.Context, content interface{}) error {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return types.NewCLIConnectionError("not connected - call Connect() first")
	}
	c.mu.Unlock()

	if content == nil {
		return fmt.Errorf("content cannot be nil")
	}
	return c.writeUserMessage(ctx, content, "Client.QueryWithContent")
}

func (c *Client) writeUserMessage(ctx context.Context, content interface{}, method string) error {
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
		return types.NewControlProtocolErrorWithCause("failed to marshal query", err)
	}
	if err := c.transport.Write(ctx, string(data)); err != nil {
		return fmt.Errorf("%s: write to transport: %w", method, err)
	}
	return nil
}
