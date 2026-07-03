package claude

import (
	"context"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/transport"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// DiscoverSupportedPermissionModes discovers the installed Claude CLI's
// permission-mode choices without opening a session.
func DiscoverSupportedPermissionModes(ctx context.Context, options *types.ClaudeAgentOptions) []types.SupportedPermissionMode {
	if options == nil {
		options = types.NewClaudeAgentOptions()
	}
	cliPath, err := clientCLIPath(ctx, options)
	if err != nil {
		return transport.FallbackPermissionModes("")
	}
	return transport.DiscoverPermissionModes(ctx, cliPath)
}

// SupportedPermissionModes returns the permission modes supported by this
// client's Claude CLI. Future structured initialize metadata wins over CLI help.
func (c *Client) SupportedPermissionModes() []types.SupportedPermissionMode {
	c.mu.Lock()
	initResult := c.initResult
	cliPath := c.cliPath
	ctx := c.ctx
	c.mu.Unlock()

	if modes := supportedPermissionModesFromInit(initResult); len(modes) > 0 {
		return modes
	}
	if ctx == nil {
		return transport.FallbackPermissionModes("")
	}
	options := types.NewClaudeAgentOptions()
	if cliPath != "" {
		options.WithCLIPath(cliPath)
	}
	return DiscoverSupportedPermissionModes(ctx, options)
}

func supportedPermissionModesFromInit(initResult *types.InitializeResult) []types.SupportedPermissionMode {
	if initResult == nil || initResult.Raw == nil {
		return nil
	}
	values := permissionModeValuesFromRaw(initResult.Raw)
	if len(values) == 0 {
		return nil
	}
	version := permissionModeVersionFromRaw(initResult.Raw)
	modes := make([]types.SupportedPermissionMode, 0, len(values))
	for _, value := range values {
		modes = append(modes, types.SupportedPermissionMode{
			ProviderValue: value,
			Source:        types.PermissionModeSourceInit,
			Version:       version,
		})
	}
	return modes
}

func permissionModeValuesFromRaw(raw map[string]interface{}) []string {
	for _, candidate := range permissionModeRawMaps(raw) {
		if values := permissionModeValuesFromMap(candidate); len(values) > 0 {
			return values
		}
	}
	return nil
}

func permissionModeRawMaps(raw map[string]interface{}) []map[string]interface{} {
	maps := []map[string]interface{}{raw}
	for _, key := range []string{"capabilities", "permissions", "permissionMode", "permission_mode", "metadata"} {
		if nested, ok := raw[key].(map[string]interface{}); ok {
			maps = append(maps, nested)
		}
	}
	return maps
}

func permissionModeValuesFromMap(raw map[string]interface{}) []string {
	for _, key := range []string{
		"permissionModes",
		"permission_modes",
		"supportedPermissionModes",
		"supported_permission_modes",
		"permissionModeChoices",
		"permission_mode_choices",
	} {
		slice, ok := initResultSlice(raw, key)
		if !ok {
			continue
		}
		return permissionModeValuesFromSlice(slice)
	}
	return nil
}

func permissionModeValuesFromSlice(slice []interface{}) []string {
	values := make([]string, 0, len(slice))
	seen := make(map[string]struct{}, len(slice))
	for _, item := range slice {
		value := permissionModeValueFromItem(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func permissionModeValueFromItem(item interface{}) string {
	if value, ok := item.(string); ok {
		return value
	}
	m, ok := item.(map[string]interface{})
	if !ok {
		return ""
	}
	for _, key := range []string{"providerValue", "provider_value", "value", "mode", "name"} {
		if value, ok := m[key].(string); ok {
			return value
		}
	}
	return ""
}

func permissionModeVersionFromRaw(raw map[string]interface{}) string {
	for _, key := range []string{"version", "cliVersion", "cli_version"} {
		if value, ok := raw[key].(string); ok {
			return value
		}
	}
	return ""
}
