package main

import (
	"context"
	"fmt"

	sdk "github.com/hishamkaram/claude-agent-sdk-go"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func main() {
	ctx := context.Background()

	calculator, err := newCalculatorServer()
	if err != nil {
		panic(fmt.Sprintf("failed to create MCP server: %v", err))
	}

	prompt := "What is 15 * 23? Also, what is the sum of 100 and 50?"
	fmt.Printf("User: %s\n\n", prompt)

	messages, err := sdk.Query(ctx, prompt, newCalculatorOptions(calculator))
	if err != nil {
		panic(fmt.Sprintf("query failed: %v", err))
	}

	printMessages(messages)
}

// newCalculatorServer creates an MCP server using the factory function.
// This eliminates most boilerplate code compared to manual implementation.
func newCalculatorServer() (*types.SDKMCPServer, error) {
	return types.NewSDKMCPServer("calculator",
		calculatorTool("add", "Add two numbers together", "addition", func(a, b float64) float64 {
			return a + b
		}),
		calculatorTool("multiply", "Multiply two numbers together", "multiplication", func(a, b float64) float64 {
			return a * b
		}),
	)
}

func calculatorTool(name, description, operation string, calculate func(a, b float64) float64) types.Tool {
	return types.Tool{
		Name:        name,
		Description: description,
		InputSchema: numberPairInputSchema(),
		Handler: func(ctx context.Context, args map[string]any) (any, error) {
			a, _ := args["a"].(float64)
			b, _ := args["b"].(float64)
			return map[string]any{
				"result":    calculate(a, b),
				"operation": operation,
			}, nil
		},
	}
}

func numberPairInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"a": map[string]interface{}{"type": "number", "description": "First number"},
			"b": map[string]interface{}{"type": "number", "description": "Second number"},
		},
		"required": []string{"a", "b"},
	}
}

func newCalculatorOptions(calculator *types.SDKMCPServer) *types.ClaudeAgentOptions {
	return types.NewClaudeAgentOptions().
		WithModel("claude-opus-4-20250514").
		WithMcpServers(map[string]interface{}{
			"calculator": calculator,
		}).
		WithSystemPrompt(`You are a helpful calculator assistant. You have access to a calculator tool with two operations:
	- add(a, b): adds two numbers
	- multiply(a, b): multiplies two numbers

	Use these tools to help the user with mathematical calculations.`)
}

func printMessages(messages <-chan types.Message) {
	for msg := range messages {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			printAssistantMessage(m)
		case *types.ResultMessage:
			printResultMessage(m)
		}
	}
}

func printAssistantMessage(message *types.AssistantMessage) {
	for _, block := range message.Content {
		switch c := block.(type) {
		case *types.TextBlock:
			fmt.Printf("Assistant: %s\n", c.Text)
		case *types.ToolUseBlock:
			fmt.Printf("🔧 Using tool: %s\n", c.Name)
			fmt.Printf("   Input: %v\n", c.Input)
		}
	}
}

func printResultMessage(message *types.ResultMessage) {
	fmt.Printf("\n📊 Result: %s\n", message.Type)
	if message.TotalCostUSD != nil {
		fmt.Printf("   Total cost: $%.4f\n", *message.TotalCostUSD)
	}
}
