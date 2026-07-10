package claude

import (
	"context"
	"fmt"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// ListModels returns the current provider-owned model catalog for this session.
func (c *Client) ListModels(ctx context.Context) ([]types.ModelInfo, error) {
	q, err := c.activeQuery("Client.ListModels: not connected - call Connect() first")
	if err != nil {
		return nil, err
	}
	resp, err := sendControlResponse(
		ctx,
		q,
		map[string]interface{}{"subtype": "list_models"},
		"Client.ListModels",
	)
	if err != nil {
		return nil, err
	}

	modelsRaw, ok := resp["models"]
	if !ok {
		return nil, nil
	}
	var models []types.ModelInfo
	if err := decodeControlResponse(modelsRaw, &models); err != nil {
		return nil, fmt.Errorf("Client.ListModels: %w", err)
	}
	return models, nil
}
