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

	// Decode through the typed shape first so malformed known fields are surfaced
	// to callers instead of being silently dropped by the permissive init parser.
	var models []types.ModelInfo
	if err := decodeControlResponse(modelsRaw, &models); err != nil {
		return nil, fmt.Errorf("Client.ListModels: %w", err)
	}

	// Preserve the original CLI rows independently of the typed fields so callers
	// retain metadata added by newer CLI versions.
	var rawModels []map[string]interface{}
	if err := decodeControlResponse(modelsRaw, &rawModels); err != nil {
		return nil, fmt.Errorf("Client.ListModels: %w", err)
	}
	for i := range models {
		models[i].Raw = cloneInitMap(rawModels[i])
	}
	return models, nil
}
