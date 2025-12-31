/*
Example 1: Simple Query
Go Version

This example shows how to send a simple query to Claude and receive responses.
*/

package main

import (
	"context"
	"fmt"

	sdk "github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

func main() {
	ctx := context.Background()

	prompt := "What is the capital of France?"

	fmt.Printf("User: %s\n\n", prompt)

	// Query Claude and stream responses
	messages, err := sdk.Query(ctx, prompt, nil)
	if err != nil {
		panic(fmt.Sprintf("query failed: %v", err))
	}

	for message := range messages {
		switch m := message.(type) {
		case *types.AssistantMessage:
			for _, block := range m.Content {
				if textBlock, ok := block.(*types.TextBlock); ok {
					fmt.Printf("Assistant: %s\n", textBlock.Text)
				}
			}
		case *types.ResultMessage:
			fmt.Println("\n✓ Response complete")
			if m.TotalCostUSD != nil {
				fmt.Printf("  Total cost: $%.4f\n", *m.TotalCostUSD)
			}
		}
	}
}
