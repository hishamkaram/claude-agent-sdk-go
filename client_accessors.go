package claude

import "github.com/hishamkaram/claude-agent-sdk-go/types"

// IsConnected returns true if the client is currently connected to Claude.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// InitResult returns the parsed initialization response from the control protocol.
func (c *Client) InitResult() *types.InitializeResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.initResult
}

// SlashCommands returns the slash commands/skills available in the current session.
func (c *Client) SlashCommands() []types.SlashCommand {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.initResult == nil {
		return nil
	}
	return c.initResult.Commands
}

// SupportedModels returns the list of models available in this session.
func (c *Client) SupportedModels() []types.ModelInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.initResult == nil {
		return nil
	}
	return c.initResult.Models
}

// SupportedAgents returns the list of supported agent types from the init result.
func (c *Client) SupportedAgents() []types.AgentInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.initResult == nil {
		return nil
	}
	return c.initResult.Agents
}

// ProcessID returns the OS process ID of the Claude Code subprocess.
func (c *Client) ProcessID() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	type pidProvider interface {
		ProcessID() int
	}
	if pp, ok := c.transport.(pidProvider); ok {
		return pp.ProcessID()
	}
	return 0
}

// Health returns a snapshot of the underlying transport/subprocess health.
func (c *Client) Health() types.TransportHealth {
	c.mu.Lock()
	defer c.mu.Unlock()

	type healthProvider interface {
		Health() types.TransportHealth
	}
	if hp, ok := c.transport.(healthProvider); ok {
		return hp.Health()
	}
	return types.TransportHealth{}
}
