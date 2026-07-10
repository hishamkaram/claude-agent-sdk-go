package types

// CapabilityDiscoveryErrorCode is a sanitized category for CLI discovery failures.
type CapabilityDiscoveryErrorCode string

const (
	CapabilityDiscoveryCLINotFound    CapabilityDiscoveryErrorCode = "cli_not_found"
	CapabilityDiscoveryAuthRequired   CapabilityDiscoveryErrorCode = "auth_required"
	CapabilityDiscoveryUnsupportedCLI CapabilityDiscoveryErrorCode = "unsupported_cli"
	CapabilityDiscoveryTimeout        CapabilityDiscoveryErrorCode = "timeout"
	CapabilityDiscoveryProvider       CapabilityDiscoveryErrorCode = "provider_error"
)

// CapabilityDiscoveryError intentionally excludes raw CLI output and paths.
type CapabilityDiscoveryError struct {
	Code      CapabilityDiscoveryErrorCode `json:"code"`
	Message   string                       `json:"message"`
	Retryable bool                         `json:"retryable"`
	Cause     error                        `json:"-"`
}

func (e *CapabilityDiscoveryError) Error() string { return e.Message }

func (e *CapabilityDiscoveryError) Unwrap() error { return e.Cause }

type RuntimeCapabilities struct {
	Models          []ModelInfo               `json:"models"`
	PermissionModes []SupportedPermissionMode `json:"permission_modes"`
	CLIVersion      string                    `json:"cli_version"`
}
