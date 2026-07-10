package claude

import (
	"context"
	"fmt"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/transport"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func providerPermissionMode(
	modes []types.SupportedPermissionMode,
	canonical types.PermissionMode,
) (string, error) {
	requested := string(canonical)
	if requested == "" {
		return "", fmt.Errorf("permission mode must not be empty")
	}
	for _, mode := range modes {
		if mode.CanonicalValue == canonical && mode.ProviderValue != "" {
			return mode.ProviderValue, nil
		}
		if mode.ProviderValue == requested {
			return mode.ProviderValue, nil
		}
	}
	return "", fmt.Errorf("permission mode %q is not advertised by the installed Claude CLI", canonical)
}

// DiscoverSupportedPermissionModes discovers the installed Claude CLI's
// permission-mode choices without opening a session.
func DiscoverSupportedPermissionModes(ctx context.Context, options *types.ClaudeAgentOptions) []types.SupportedPermissionMode {
	modes, _, _ := discoverSupportedPermissionModes(ctx, options)
	return modes
}

func discoverSupportedPermissionModes(
	ctx context.Context,
	options *types.ClaudeAgentOptions,
) (modes []types.SupportedPermissionMode, version string, err error) {
	if options == nil {
		options = types.NewClaudeAgentOptions()
	}
	cliPath, err := clientCLIPath(ctx, options)
	if err != nil {
		return nil, "", err
	}
	if options.SpawnProcess != nil {
		return transport.DiscoverPermissionModesWithSpawner(
			ctx,
			cliPath,
			clientWorkingDirectory(options),
			clientProbeEnvironment(options),
			options.SpawnProcess,
		)
	}
	return transport.DiscoverPermissionModesAndVersionWithEnvironment(
		ctx,
		cliPath,
		clientWorkingDirectory(options),
		clientProbeEnvironment(options),
	)
}

func clientProbeEnvironment(options *types.ClaudeAgentOptions) map[string]string {
	return transport.BuildRuntimeEnvironment(options, clientEnvironment(options))
}

// SupportedPermissionModes returns the permission modes supported by this
// client's Claude CLI. Future structured initialize metadata wins over CLI help.
func (c *Client) SupportedPermissionModes() []types.SupportedPermissionMode {
	c.mu.Lock()
	ctx := c.ctx
	c.mu.Unlock()
	modes, _, _ := c.supportedPermissionModes(ctx)
	return modes
}

func (c *Client) supportedPermissionModes(
	ctx context.Context,
) (modes []types.SupportedPermissionMode, version string, err error) {
	c.mu.Lock()
	initResult := c.initResult
	cliPath := c.cliPath
	options := clonePermissionDiscoveryOptions(c.options, cliPath)
	cached := cloneClientPermissionModes(c.permissionModes)
	cachedVersion := c.permissionModeVersion
	c.mu.Unlock()

	if initModes := supportedPermissionModesFromInit(initResult); len(initModes) > 0 {
		version = permissionModeVersion(initModes)
		if version == "" {
			version = cachedVersion
		}
		if version == "" && ctx != nil {
			version, err = discoverSupportedPermissionModeVersion(ctx, options)
			if err != nil {
				return initModes, "", err
			}
			if version != "" {
				c.mu.Lock()
				c.permissionModes = cloneClientPermissionModes(initModes)
				c.permissionModeVersion = version
				c.mu.Unlock()
			}
		}
		return initModes, version, nil
	}
	if len(cached) > 0 {
		return cached, cachedVersion, nil
	}
	if ctx == nil {
		return nil, "", nil
	}
	modes, version, err = discoverSupportedPermissionModes(ctx, options)
	if len(modes) > 0 {
		c.mu.Lock()
		c.permissionModes = cloneClientPermissionModes(modes)
		c.permissionModeVersion = version
		c.mu.Unlock()
	}
	return modes, version, err
}

func discoverSupportedPermissionModeVersion(
	ctx context.Context,
	options *types.ClaudeAgentOptions,
) (string, error) {
	if options == nil {
		options = types.NewClaudeAgentOptions()
	}
	cliPath, err := clientCLIPath(ctx, options)
	if err != nil {
		return "", err
	}
	cwd := clientWorkingDirectory(options)
	env := clientProbeEnvironment(options)
	if options.SpawnProcess != nil {
		return transport.DiscoverCLIVersionWithSpawner(
			ctx,
			cliPath,
			cwd,
			env,
			options.SpawnProcess,
		)
	}
	return transport.DiscoverCLIVersionWithEnvironment(ctx, cliPath, cwd, env)
}

func clonePermissionDiscoveryOptions(
	options *types.ClaudeAgentOptions,
	cliPath string,
) *types.ClaudeAgentOptions {
	if options == nil {
		options = types.NewClaudeAgentOptions()
	}
	cloned := *options
	if cliPath != "" {
		cloned.CLIPath = &cliPath
	}
	return &cloned
}

func cloneClientPermissionModes(
	modes []types.SupportedPermissionMode,
) []types.SupportedPermissionMode {
	if len(modes) == 0 {
		return nil
	}
	cloned := make([]types.SupportedPermissionMode, len(modes))
	copy(cloned, modes)
	return cloned
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
			ProviderValue:  value,
			CanonicalValue: canonicalPermissionMode(value),
			Source:         types.PermissionModeSourceInit,
			Version:        version,
		})
	}
	return modes
}

func canonicalPermissionMode(value string) types.PermissionMode {
	switch value {
	case "manual", string(types.PermissionModeDefault):
		return types.PermissionModeDefault
	case string(types.PermissionModeAcceptEdits):
		return types.PermissionModeAcceptEdits
	case string(types.PermissionModeAuto):
		return types.PermissionModeAuto
	case string(types.PermissionModePlan):
		return types.PermissionModePlan
	case string(types.PermissionModeBypassPermissions):
		return types.PermissionModeBypassPermissions
	case string(types.PermissionModeDontAsk):
		return types.PermissionModeDontAsk
	default:
		return ""
	}
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
