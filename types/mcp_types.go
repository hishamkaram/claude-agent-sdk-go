package types

// McpServerStatusInfo represents the runtime status of an MCP server connection.
type McpServerStatusInfo struct {
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	ServerInfo *McpServerInfo `json:"serverInfo,omitempty"`
	Error      string         `json:"error,omitempty"`
	Scope      string         `json:"scope,omitempty"`
	Tools      []McpToolInfo  `json:"tools,omitempty"`
}

// McpServerInfo represents a connected MCP server's self-reported info.
type McpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// McpToolInfo represents a tool provided by an MCP server.
type McpToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// McpSetServersResult represents the result of a SetMCPServers operation.
type McpSetServersResult struct {
	Added   []string          `json:"added"`
	Removed []string          `json:"removed"`
	Errors  map[string]string `json:"errors"`
}
