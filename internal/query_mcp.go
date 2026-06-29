package internal

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// handleMCPMessage handles an MCP message request.
func (q *Query) handleMCPMessage(requestData map[string]interface{}) (map[string]interface{}, error) {
	serverName, serverNameOk := requestData["server_name"].(string)
	if !serverNameOk && requestData["server_name"] != nil {
		q.logger.Warn("handleMCPMessage: server_name has unexpected type",
			zap.Any("server_name", requestData["server_name"]))
		return nil, types.NewControlProtocolError("server_name must be a string in MCP request")
	}

	message, messageOk := requestData["message"].(map[string]interface{})
	if !messageOk && requestData["message"] != nil {
		q.logger.Warn("handleMCPMessage: message has unexpected type",
			zap.Any("message_type", fmt.Sprintf("%T", requestData["message"])))
		return nil, types.NewControlProtocolError("message must be a map in MCP request")
	}

	if serverName == "" || message == nil {
		return nil, types.NewControlProtocolError("missing server_name or message in MCP request")
	}

	// Find MCP server
	q.mu.Lock()
	server, exists := q.mcpServers[serverName]
	q.mu.Unlock()

	if !exists {
		// Return JSONRPC error response
		messageID := message["id"]
		return map[string]interface{}{
			"mcp_response": map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      messageID,
				"error": map[string]interface{}{
					"code":    -32601,
					"message": fmt.Sprintf("Server '%s' not found", serverName),
				},
			},
		}, nil
	}

	// Route message to MCP server
	mcpResponse, mcpErr := server.HandleMessage(message)
	if mcpErr != nil {
		// Surface the MCP failure as a JSONRPC error response (protocol-level
		// error), not a Go transport error. The control loop expects a response
		// map with no Go error.
		return mcpServerErrorResponse(message["id"], mcpErr), nil
	}

	return map[string]interface{}{
		"mcp_response": mcpResponse,
	}, nil
}

// mcpServerErrorResponse wraps an MCP server failure as a JSONRPC internal-error
// response payload. The failure is intentionally reported in-band (as a JSONRPC
// error object) rather than as a Go transport error.
func mcpServerErrorResponse(messageID interface{}, cause error) map[string]interface{} {
	return map[string]interface{}{
		"mcp_response": map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      messageID,
			"error": map[string]interface{}{
				"code":    -32603,
				"message": cause.Error(),
			},
		},
	}
}

// AddMCPServer adds an MCP server for handling MCP messages.
func (q *Query) AddMCPServer(name string, server types.MCPServer) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.mcpServers[name] = server
}
