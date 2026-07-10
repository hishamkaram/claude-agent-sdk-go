package claude

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// DiscoverRuntimeCapabilities opens a no-prompt streaming session, reads the
// provider model catalog, and always closes the subprocess before returning.
func DiscoverRuntimeCapabilities(
	ctx context.Context,
	options *types.ClaudeAgentOptions,
) (*types.RuntimeCapabilities, error) {
	client, err := NewClient(ctx, options)
	if err != nil {
		return nil, capabilityDiscoveryError(err)
	}
	defer closeDiscoveryClient(ctx, client)

	if connectErr := client.Connect(ctx); connectErr != nil {
		return nil, capabilityDiscoveryError(connectErr)
	}
	models, err := client.ListModels(ctx)
	if err != nil {
		return nil, capabilityDiscoveryError(err)
	}
	if len(models) == 0 {
		return nil, unsupportedCapabilityDiscoveryError(
			"Claude CLI did not report an available model catalog",
			nil,
		)
	}

	modes, version, discoveryErr := client.supportedPermissionModes(ctx)
	if discoveryErr != nil {
		return nil, capabilityDiscoveryError(discoveryErr)
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, capabilityDiscoveryError(ctxErr)
	}
	if len(modes) == 0 {
		return nil, unsupportedCapabilityDiscoveryError(
			"Claude CLI did not report permission-mode capabilities",
			nil,
		)
	}
	if version == "" {
		return nil, unsupportedCapabilityDiscoveryError(
			"Claude CLI did not report its version",
			nil,
		)
	}
	return &types.RuntimeCapabilities{
		Models:          models,
		PermissionModes: modes,
		CLIVersion:      version,
	}, nil
}

func closeDiscoveryClient(ctx context.Context, client *Client) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_ = client.Close(cleanupCtx)
}

func permissionModeVersion(modes []types.SupportedPermissionMode) string {
	for _, mode := range modes {
		if mode.Version != "" {
			return mode.Version
		}
	}
	return ""
}

func capabilityDiscoveryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &types.CapabilityDiscoveryError{
			Code: types.CapabilityDiscoveryTimeout, Message: "Claude capability discovery timed out", Retryable: true, Cause: err,
		}
	}
	if types.IsCLINotFoundError(err) || errors.Is(err, os.ErrNotExist) {
		return &types.CapabilityDiscoveryError{
			Code: types.CapabilityDiscoveryCLINotFound, Message: "Claude CLI is not installed", Retryable: true, Cause: err,
		}
	}
	lower := strings.ToLower(err.Error())
	if !errors.Is(err, os.ErrPermission) && isClaudeAuthenticationFailure(lower) {
		return &types.CapabilityDiscoveryError{
			Code: types.CapabilityDiscoveryAuthRequired, Message: "Claude authentication is required", Retryable: true, Cause: err,
		}
	}
	if types.IsControlProtocolError(err) && strings.Contains(lower, "unsupported") {
		return unsupportedCapabilityDiscoveryError(
			"Installed Claude CLI does not support capability discovery",
			err,
		)
	}
	return &types.CapabilityDiscoveryError{
		Code: types.CapabilityDiscoveryProvider, Message: "Claude capability discovery failed", Retryable: true, Cause: err,
	}
}

func unsupportedCapabilityDiscoveryError(
	message string,
	cause error,
) *types.CapabilityDiscoveryError {
	return &types.CapabilityDiscoveryError{
		Code:    types.CapabilityDiscoveryUnsupportedCLI,
		Message: message,
		Cause:   cause,
	}
}

func isClaudeAuthenticationFailure(message string) bool {
	for _, marker := range []string{
		"authentication required",
		"authentication failed",
		"oauth",
		"unauthorized",
		"invalid api key",
		"invalid token",
		"log in to claude",
		"login required",
		"not logged in",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}
