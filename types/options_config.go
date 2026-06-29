package types

import (
	"context"
	"fmt"
	"io"
)

// EffortLevel represents reasoning effort levels.
type EffortLevel string

const (
	EffortLow    EffortLevel = "low"
	EffortMedium EffortLevel = "medium"
	EffortHigh   EffortLevel = "high"
	EffortXHigh  EffortLevel = "xhigh"
	EffortMax    EffortLevel = "max"
)

// ThinkingConfig controls Claude's thinking/reasoning behavior.
type ThinkingConfig struct {
	Type         string `json:"type"`                   // "adaptive", "enabled", "disabled"
	BudgetTokens *int   `json:"budgetTokens,omitempty"` // Only for type="enabled"
	Display      string `json:"display,omitempty"`      // Optional provider display mode, e.g. "summarized"
}

// OutputFormat represents structured output configuration (JsonSchemaOutputFormat).
type OutputFormat struct {
	Type   string                 `json:"type"`             // Always "json_schema"
	Schema map[string]interface{} `json:"schema,omitempty"` // JSON schema definition
	Name   *string                `json:"name,omitempty"`   // Optional schema name
}

// SandboxConfig matches TS SDK's SandboxSettings.
type SandboxConfig struct {
	Enabled                      *bool                    `json:"enabled,omitempty"`
	AutoAllowBashIfSandboxed     *bool                    `json:"autoAllowBashIfSandboxed,omitempty"`
	AllowUnsandboxedCommands     *bool                    `json:"allowUnsandboxedCommands,omitempty"`
	Network                      *SandboxNetworkConfig    `json:"network,omitempty"`
	Filesystem                   *SandboxFilesystemConfig `json:"filesystem,omitempty"`
	IgnoreViolations             map[string][]string      `json:"ignoreViolations,omitempty"`
	EnableWeakerNestedSandbox    *bool                    `json:"enableWeakerNestedSandbox,omitempty"`
	EnableWeakerNetworkIsolation *bool                    `json:"enableWeakerNetworkIsolation,omitempty"`
	ExcludedCommands             []string                 `json:"excludedCommands,omitempty"`
}

// SandboxNetworkConfig represents network sandbox configuration.
type SandboxNetworkConfig struct {
	AllowedDomains          []string `json:"allowedDomains,omitempty"`
	AllowManagedDomainsOnly *bool    `json:"allowManagedDomainsOnly,omitempty"`
	AllowUnixSockets        []string `json:"allowUnixSockets,omitempty"`
	AllowAllUnixSockets     *bool    `json:"allowAllUnixSockets,omitempty"`
	AllowLocalBinding       *bool    `json:"allowLocalBinding,omitempty"`
	HttpProxyPort           *int     `json:"httpProxyPort,omitempty"`
	SocksProxyPort          *int     `json:"socksProxyPort,omitempty"`
}

// SandboxFilesystemConfig represents filesystem sandbox configuration.
type SandboxFilesystemConfig struct {
	AllowWrite                []string `json:"allowWrite,omitempty"`
	DenyWrite                 []string `json:"denyWrite,omitempty"`
	DenyRead                  []string `json:"denyRead,omitempty"`
	AllowRead                 []string `json:"allowRead,omitempty"`
	AllowManagedReadPathsOnly *bool    `json:"allowManagedReadPathsOnly,omitempty"`
}

// SpawnOptions contains everything needed to start a Claude Code process.
// It is passed to a ProcessSpawner to create a subprocess in custom environments
// (Docker, VMs, remote SSH, etc.).
type SpawnOptions struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	CWD     string            `json:"cwd,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// SpawnedProcess abstracts a running Claude Code process.
// Consumers implement this interface to support custom execution environments
// (Docker, SSH, VMs, remote processes). The SDK uses this interface to
// communicate with the process via stdin/stdout/stderr pipes.
type SpawnedProcess interface {
	Stdin() io.WriteCloser
	Stdout() io.ReadCloser
	Stderr() io.ReadCloser
	Kill() error
	Wait() error
	ExitCode() int
	Killed() bool
}

// ProcessSpawner creates a Claude Code process from spawn options.
// It is the injection point for custom process creation. When set on
// ClaudeAgentOptions.SpawnProcess, the SDK calls this function instead of
// using the default exec.Command subprocess.
type ProcessSpawner func(ctx context.Context, opts SpawnOptions) (SpawnedProcess, error)

// ToolConfig configures built-in tool behavior.
// Delivered via settings JSON with key "toolConfig".
type ToolConfig struct {
	Bash     *BashToolConfig     `json:"bash,omitempty"`
	Computer *ComputerToolConfig `json:"computer,omitempty"`
}

// BashToolConfig configures bash tool behavior.
type BashToolConfig struct {
	Timeout *int    `json:"timeout,omitempty"` // Command timeout in milliseconds
	Command *string `json:"command,omitempty"` // Shell command override
}

// ComputerToolConfig configures computer tool behavior.
type ComputerToolConfig struct {
	Display *int `json:"display,omitempty"` // Display number
	Width   *int `json:"width,omitempty"`   // Screen width in pixels
	Height  *int `json:"height,omitempty"`  // Screen height in pixels
}

// SettingSource represents where settings are loaded from.
type SettingSource string

const (
	SettingSourceUser    SettingSource = "user"
	SettingSourceProject SettingSource = "project"
	SettingSourceLocal   SettingSource = "local"
)

// SystemPromptPreset represents a preset system prompt configuration.
type SystemPromptPreset struct {
	Type   string  `json:"type"`   // "preset"
	Preset string  `json:"preset"` // "claude_code"
	Append *string `json:"append,omitempty"`
}

// PluginConfig represents a Claude Code plugin configuration.
// Currently only local plugins are supported via the 'local' type.
type PluginConfig struct {
	Type string `json:"type"` // "local" - plugin type
	Path string `json:"path"` // Absolute or relative path to plugin directory
}

// NewPluginConfig creates a new PluginConfig with validation.
// Returns an error if the plugin type is not supported or path is empty.
func NewPluginConfig(pluginType, path string) (*PluginConfig, error) {
	if pluginType != "local" {
		return nil, fmt.Errorf("unsupported plugin type %q: only 'local' is supported", pluginType)
	}
	if path == "" {
		return nil, fmt.Errorf("plugin path cannot be empty")
	}
	return &PluginConfig{
		Type: pluginType,
		Path: path,
	}, nil
}

// NewLocalPluginConfig creates a new local plugin configuration.
// This is a convenience function for the most common plugin type.
func NewLocalPluginConfig(path string) *PluginConfig {
	return &PluginConfig{
		Type: "local",
		Path: path,
	}
}

// McpStdioServerConfig represents an MCP stdio server configuration.
type McpStdioServerConfig struct {
	Type    *string           `json:"type,omitempty"` // "stdio" - optional for backwards compatibility
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// McpSSEServerConfig represents an MCP SSE server configuration.
type McpSSEServerConfig struct {
	Type    string            `json:"type"` // "sse"
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// McpHTTPServerConfig represents an MCP HTTP server configuration.
type McpHTTPServerConfig struct {
	Type    string            `json:"type"` // "http"
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// McpSdkServerConfig represents an SDK MCP server configuration.
type McpSdkServerConfig struct {
	Type     string      `json:"type"` // "sdk"
	Name     string      `json:"name"`
	Instance interface{} `json:"instance"` // MCP Server instance - type depends on MCP SDK
}

// CanUseToolFunc is a callback function for tool permission requests.
// It receives the tool name, input parameters, and context, and returns a permission result.
type CanUseToolFunc func(ctx context.Context, toolName string, input map[string]interface{}, permCtx ToolPermissionContext) (interface{}, error)

// HookCallbackFunc is a callback function for hook events.
// It receives the hook input, optional tool use ID, and context, and returns hook output.
type HookCallbackFunc func(ctx context.Context, input interface{}, toolUseID *string, hookCtx HookContext) (interface{}, error)

// HookMatcher represents a hook matcher configuration.
type HookMatcher struct {
	Matcher *string            `json:"matcher,omitempty"` // Regex pattern for matching (e.g., "Bash", "Write|Edit")
	Hooks   []HookCallbackFunc `json:"-"`                 // List of hook callback functions (not marshaled)
}

// StderrCallbackFunc is a callback function for stderr output from the CLI.
type StderrCallbackFunc func(line string)
