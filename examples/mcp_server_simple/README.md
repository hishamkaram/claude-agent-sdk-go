# Simple MCP Server Example

This example demonstrates how to create a simple MCP (Model Context Protocol) server using the SDK factory function. This is the recommended way to create MCP servers for most use cases.

## Key Features

- **Minimal Boilerplate**: The factory function `NewSDKMCPServer()` handles all JSON-RPC message routing
- **Type-Safe Tools**: Define tools with a simple struct that includes name, description, schema, and handler
- **Input Validation**: Optional JSON schema validation for tool inputs
- **Error Handling**: Proper JSONRPC error responses with correct error codes

## What This Example Shows

### Creating an MCP Server

```go
calculator, err := types.NewSDKMCPServer("calculator",
    types.Tool{
        Name:        "add",
        Description: "Add two numbers",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "a": map[string]interface{}{"type": "number"},
                "b": map[string]interface{}{"type": "number"},
            },
        },
        Handler: func(ctx context.Context, args map[string]any) (any, error) {
            a, _ := args["a"].(float64)
            b, _ := args["b"].(float64)
            return map[string]any{"result": a + b}, nil
        },
    },
)
```

### Registering with Claude

```go
options := sdk.NewClaudeAgentOptions().
    WithMCPServer("calculator", calculator)

messages, _ := sdk.Query(ctx, "What is 15 * 23?", options)
```

## How the Factory Function Works

The `NewSDKMCPServer()` factory:

1. **Validates input**: Checks that server name is provided, tools are provided, and each tool is valid
2. **Handles JSON-RPC 2.0**: Automatically routes incoming messages to the correct handler
3. **Manages tool registry**: Keeps track of all registered tools
4. **Executes handlers**: Calls the appropriate handler function when Claude uses a tool
5. **Formats responses**: Converts handler results into proper MCP content blocks

## Comparison with Manual Implementation

### With Factory (Recommended)

```go
// ~20 lines of code
server, err := types.NewSDKMCPServer("calc",
    types.Tool{
        Name:        "add",
        Description: "Add numbers",
        Handler: func(ctx context.Context, args map[string]any) (any, error) {
            return args["a"].(float64) + args["b"].(float64), nil
        },
    },
)
```

### Manual Implementation (Not Recommended)

```go
// ~100+ lines of code including:
// - Implement MCPServer interface
// - Parse JSON-RPC 2.0 messages
// - Route to tools/list and tools/call
// - Format tool responses
// - Handle errors
// - Validate schemas
```

## Running the Example

```bash
cd examples/mcp_server_simple
go run main.go
```

## Key API Details

### Tool Structure

```go
type Tool struct {
    Name        string                         // Unique tool identifier
    Description string                         // User-facing description
    InputSchema map[string]interface{}         // Optional JSON schema for inputs
    Handler     func(ctx context.Context, args map[string]any) (any, error)
}
```

### Handler Function

The handler receives:
- **ctx**: Context for cancellation and deadlines
- **args**: Map of input arguments from Claude

The handler should return:
- **any**: The tool result (string, map, or slice)
- **error**: Any error that occurred

### Result Formatting

The factory automatically converts handler results to MCP content blocks:

| Return Type | Content Block |
|-------------|---------------|
| `string` | TextBlock with the string |
| `map[string]any` with "text" key | TextBlock with the text value |
| `map[string]any` without "text" key | TextBlock with JSON representation |
| `[]map[string]any` | Multiple content blocks |

## Error Handling

The factory provides proper JSONRPC error responses:

- **-32600**: Invalid Request (missing/malformed JSON-RPC)
- **-32601**: Method not found (unknown RPC method)
- **-32602**: Invalid params (missing or malformed parameters)
- **-32603**: Internal error (tool not found or execution failed)

## Advanced Usage

For more complex MCP servers with:
- Custom request/response handling
- Real-time bidirectional communication
- Multiple protocol versions

See the `mcp_server_advanced` example or implement the `MCPServer` interface directly.

## Related Documentation

- [MCP Specification](https://modelcontextprotocol.io/)
- [SDK Types](../../types/mcp.go)
- [Full API Reference](../../README.md)
