package types

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewSDKMCPServer_Success(t *testing.T) {
	tests := []struct {
		name    string
		tools   []Tool
		wantErr bool
	}{
		{
			name: "single tool",
			tools: []Tool{
				{
					Name:        "add",
					Description: "Add two numbers",
					Handler: func(ctx context.Context, args map[string]any) (any, error) {
						return map[string]any{"result": 5}, nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple tools",
			tools: []Tool{
				{
					Name:        "add",
					Description: "Add two numbers",
					Handler: func(ctx context.Context, args map[string]any) (any, error) {
						return 5, nil
					},
				},
				{
					Name:        "subtract",
					Description: "Subtract two numbers",
					Handler: func(ctx context.Context, args map[string]any) (any, error) {
						return 3, nil
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := NewSDKMCPServer("test-server", tt.tools...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewSDKMCPServer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && server == nil {
				t.Fatal("expected server, got nil")
			}
		})
	}
}

func TestNewSDKMCPServer_Errors(t *testing.T) {
	tests := []struct {
		name      string
		serverName string
		tools     []Tool
		wantErr   bool
		errType   string
	}{
		{
			name:       "empty server name",
			serverName: "",
			tools: []Tool{
				{
					Name:        "test",
					Description: "Test tool",
					Handler:     func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
				},
			},
			wantErr: true,
			errType: "ValidationError",
		},
		{
			name:       "no tools",
			serverName: "test",
			tools:      []Tool{},
			wantErr:    true,
			errType:    "ValidationError",
		},
		{
			name:       "tool with empty name",
			serverName: "test",
			tools: []Tool{
				{
					Name:        "",
					Description: "Test tool",
					Handler:     func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
				},
			},
			wantErr: true,
			errType: "ValidationError",
		},
		{
			name:       "tool with nil handler",
			serverName: "test",
			tools: []Tool{
				{
					Name:        "test",
					Description: "Test tool",
					Handler:     nil,
				},
			},
			wantErr: true,
			errType: "ValidationError",
		},
		{
			name:       "duplicate tool names",
			serverName: "test",
			tools: []Tool{
				{
					Name:        "test",
					Description: "Test tool 1",
					Handler:     func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
				},
				{
					Name:        "test",
					Description: "Test tool 2",
					Handler:     func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
				},
			},
			wantErr: true,
			errType: "ValidationError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSDKMCPServer(tt.serverName, tt.tools...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewSDKMCPServer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !IsValidationError(err) {
				t.Fatalf("expected ValidationError, got %T", err)
			}
		})
	}
}

func TestSDKMCPServer_Name(t *testing.T) {
	server, _ := NewSDKMCPServer("my-server",
		Tool{
			Name:        "test",
			Description: "Test",
			Handler:     func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
		},
	)

	if server.Name() != "my-server" {
		t.Fatalf("expected 'my-server', got %s", server.Name())
	}
}

func TestSDKMCPServer_Version(t *testing.T) {
	server, _ := NewSDKMCPServer("test",
		Tool{
			Name:        "test",
			Description: "Test",
			Handler:     func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
		},
	)

	if server.Version() != "1.0.0" {
		t.Fatalf("expected '1.0.0', got %s", server.Version())
	}
}

func TestSDKMCPServer_HandleListTools(t *testing.T) {
	server, _ := NewSDKMCPServer("test",
		Tool{
			Name:        "add",
			Description: "Add numbers",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"a": map[string]interface{}{"type": "number"},
					"b": map[string]interface{}{"type": "number"},
				},
			},
			Handler: func(ctx context.Context, args map[string]any) (any, error) { return 0, nil },
		},
		Tool{
			Name:        "multiply",
			Description: "Multiply numbers",
			Handler:     func(ctx context.Context, args map[string]any) (any, error) { return 0, nil },
		},
	)

	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	}

	response, err := server.HandleMessage(message)
	if err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}

	// Verify response structure
	if response["jsonrpc"] != "2.0" {
		t.Fatal("invalid jsonrpc version")
	}
	if response["id"] != 1 {
		t.Fatal("id mismatch")
	}

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	tools, ok := result["tools"].([]map[string]interface{})
	if !ok {
		t.Fatal("tools is not an array")
	}

	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	// Check first tool
	if tools[0]["name"] != "add" || tools[0]["description"] != "Add numbers" {
		t.Fatal("first tool mismatch")
	}
	if tools[0]["inputSchema"] == nil {
		t.Fatal("expected inputSchema for add tool")
	}

	// Check second tool
	if tools[1]["name"] != "multiply" || tools[1]["description"] != "Multiply numbers" {
		t.Fatal("second tool mismatch")
	}
	if tools[1]["inputSchema"] != nil {
		t.Fatal("unexpected inputSchema for multiply tool")
	}
}

func TestSDKMCPServer_HandleCallTool_Success(t *testing.T) {
	server, _ := NewSDKMCPServer("calculator",
		Tool{
			Name:        "add",
			Description: "Add two numbers",
			Handler: func(ctx context.Context, args map[string]any) (any, error) {
				a, _ := args["a"].(float64)
				b, _ := args["b"].(float64)
				return map[string]any{"result": a + b}, nil
			},
		},
	)

	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      42,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "add",
			"arguments": map[string]interface{}{"a": float64(10), "b": float64(20)},
		},
	}

	response, err := server.HandleMessage(message)
	if err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}

	if response["jsonrpc"] != "2.0" {
		t.Fatal("invalid jsonrpc version")
	}
	if response["id"] != 42 {
		t.Fatal("id mismatch")
	}

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	content, ok := result["content"].([]map[string]interface{})
	if !ok {
		t.Fatal("content is not an array")
	}

	if len(content) == 0 {
		t.Fatal("expected content blocks")
	}

	// Verify it's text content
	if content[0]["type"] != "text" {
		t.Fatal("expected text content block")
	}

	// The result should contain the computed value
	text, ok := content[0]["text"].(string)
	if !ok || text == "" {
		t.Fatal("expected non-empty text content")
	}

	// The text should contain the result
	if !strings.Contains(text, "30") {
		t.Fatalf("expected result to contain '30', got %s", text)
	}
}

func TestSDKMCPServer_HandleCallTool_StringResult(t *testing.T) {
	server, _ := NewSDKMCPServer("greet",
		Tool{
			Name:        "greet",
			Description: "Greet a user",
			Handler: func(ctx context.Context, args map[string]any) (any, error) {
				name, _ := args["name"].(string)
				return "Hello, " + name + "!", nil
			},
		},
	)

	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "greet",
			"arguments": map[string]interface{}{"name": "Alice"},
		},
	}

	response, _ := server.HandleMessage(message)
	result := response["result"].(map[string]interface{})
	content := result["content"].([]map[string]interface{})

	if content[0]["type"] != "text" {
		t.Fatal("expected text content")
	}

	text := content[0]["text"].(string)
	if text != "Hello, Alice!" {
		t.Fatalf("expected 'Hello, Alice!', got %s", text)
	}
}

func TestSDKMCPServer_HandleCallTool_ToolNotFound(t *testing.T) {
	server, _ := NewSDKMCPServer("test",
		Tool{
			Name:        "exists",
			Description: "A tool that exists",
			Handler:     func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
		},
	)

	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "nonexistent",
			"arguments": map[string]interface{}{},
		},
	}

	response, _ := server.HandleMessage(message)

	if response["error"] == nil {
		t.Fatal("expected error response")
	}

	errObj := response["error"].(map[string]interface{})
	code := errObj["code"]
	expectedCode := -32603
	actualCode := int(0)
	switch v := code.(type) {
	case int:
		actualCode = v
	case float64:
		actualCode = int(v)
	}
	if actualCode != expectedCode {
		t.Fatalf("expected error code %d, got %d", expectedCode, actualCode)
	}
	if !strings.Contains(errObj["message"].(string), "Tool not found") {
		t.Fatalf("expected 'Tool not found' in error message, got %s", errObj["message"])
	}
}

func TestSDKMCPServer_HandleCallTool_MissingParams(t *testing.T) {
	server, _ := NewSDKMCPServer("test",
		Tool{
			Name:        "test",
			Description: "Test",
			Handler:     func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
		},
	)

	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		// Missing params field
	}

	response, _ := server.HandleMessage(message)

	if response["error"] == nil {
		t.Fatal("expected error response")
	}

	errObj := response["error"].(map[string]interface{})
	code := errObj["code"]
	expectedCode := -32602
	actualCode := int(0)
	switch v := code.(type) {
	case int:
		actualCode = v
	case float64:
		actualCode = int(v)
	}
	if actualCode != expectedCode {
		t.Fatalf("expected error code %d, got %d", expectedCode, actualCode)
	}
}

func TestSDKMCPServer_HandleUnknownMethod(t *testing.T) {
	server, _ := NewSDKMCPServer("test",
		Tool{
			Name:        "test",
			Description: "Test",
			Handler:     func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
		},
	)

	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "unknown/method",
	}

	response, _ := server.HandleMessage(message)

	if response["error"] == nil {
		t.Fatal("expected error response for unknown method")
	}

	errObj := response["error"].(map[string]interface{})
	code := errObj["code"]
	expectedCode := -32601
	actualCode := int(0)
	switch v := code.(type) {
	case int:
		actualCode = v
	case float64:
		actualCode = int(v)
	}
	if actualCode != expectedCode {
		t.Fatalf("expected error code %d, got %d", expectedCode, actualCode)
	}
}

func TestSDKMCPServer_HandleCallTool_WithEmptyArgs(t *testing.T) {
	server, _ := NewSDKMCPServer("test",
		Tool{
			Name:        "test",
			Description: "Test",
			Handler: func(ctx context.Context, args map[string]any) (any, error) {
				return "called", nil
			},
		},
	)

	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "test",
			// No arguments
		},
	}

	response, _ := server.HandleMessage(message)

	result := response["result"].(map[string]interface{})
	content := result["content"].([]map[string]interface{})

	if len(content) == 0 {
		t.Fatal("expected content blocks")
	}

	if content[0]["text"] != "called" {
		t.Fatalf("expected 'called', got %s", content[0]["text"])
	}
}

func TestToolValidate(t *testing.T) {
	tests := []struct {
		name    string
		tool    Tool
		wantErr bool
	}{
		{
			name: "valid tool",
			tool: Tool{
				Name:        "test",
				Description: "Test tool",
				Handler:     func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
			},
			wantErr: false,
		},
		{
			name: "missing name",
			tool: Tool{
				Description: "Test tool",
				Handler:     func(ctx context.Context, args map[string]any) (any, error) { return nil, nil },
			},
			wantErr: true,
		},
		{
			name: "missing handler",
			tool: Tool{
				Name:        "test",
				Description: "Test tool",
				Handler:     nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tool.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test for JSON encoding/decoding compatibility
func TestSDKMCPServer_JSONCompatibility(t *testing.T) {
	server, _ := NewSDKMCPServer("test",
		Tool{
			Name:        "test",
			Description: "Test",
			Handler: func(ctx context.Context, args map[string]any) (any, error) {
				return map[string]any{"status": "ok", "count": 42}, nil
			},
		},
	)

	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "test",
			"arguments": map[string]interface{}{},
		},
	}

	response, _ := server.HandleMessage(message)

	// Try to marshal response to JSON and back
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var unmarshaled map[string]interface{}
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Verify the response is valid JSON
	if unmarshaled["jsonrpc"] != "2.0" {
		t.Fatal("jsonrpc field missing after JSON roundtrip")
	}
}
