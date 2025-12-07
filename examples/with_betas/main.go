package main

import (
	"context"
	"fmt"
	"log"

	claude "github.com/schlunsen/claude-agent-sdk-go"
	"github.com/schlunsen/claude-agent-sdk-go/types"
)

// WithBetas demonstrates how to use beta API features.
// Beta features allow opt-in to Anthropic's experimental APIs.
// Example: context-1m-2025-08-07 provides extended context window support.
func main() {
	ctx := context.Background()

	// Create options with beta features enabled
	opts := types.NewClaudeAgentOptions().
		WithModel("claude-3-5-sonnet-latest").
		// Add beta features - example: extended context window
		WithBeta("context-1m-2025-08-07")

	// Alternative: Set multiple betas at once
	// opts := types.NewClaudeAgentOptions().
	// 	WithModel("claude-3-5-sonnet-latest").
	// 	WithBetas([]string{"context-1m-2025-08-07"})

	fmt.Println("Claude Agent SDK - Beta Features Example")
	fmt.Println("========================================")
	fmt.Println("Using beta features: context-1m-2025-08-07")
	fmt.Println("This provides extended context window support (up to 1M tokens)")
	fmt.Println("---")
	fmt.Println()

	// Send a query that can take advantage of the extended context
	prompt := `Please summarize the key differences between Go and Python programming languages.
Focus on:
1. Concurrency models
2. Type systems
3. Performance characteristics
4. Use cases and ecosystem`

	fmt.Printf("Sending query...\n")
	fmt.Println("---")

	messages, err := claude.Query(ctx, prompt, opts)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	// Process messages from the channel
	for msg := range messages {
		msgType := msg.GetMessageType()

		switch msgType {
		case "assistant":
			if assistantMsg, ok := msg.(*types.AssistantMessage); ok {
				for _, block := range assistantMsg.Content {
					if textBlock, ok := block.(*types.TextBlock); ok {
						fmt.Printf("%s\n", textBlock.Text)
					}
				}
			}
		case "result":
			fmt.Println("---")
			fmt.Println("Query completed successfully with beta features")
		}
	}
}
