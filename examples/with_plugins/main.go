package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/hishamkaram/claude-agent-sdk-go"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("with_plugins example failed: %v", err)
	}
}

func run() error {
	// Check for required environment variable
	if os.Getenv("CLAUDE_API_KEY") == "" {
		return fmt.Errorf("CLAUDE_API_KEY environment variable must be set")
	}

	// Create options with a local plugin
	// This assumes you have a Claude Code plugin directory at ../plugins/demo-plugin
	options := types.NewClaudeAgentOptions().
		WithLocalPlugin("../plugins/demo-plugin").
		WithVerbose(false)

	// You can also add multiple plugins:
	// options.WithLocalPlugin("./my-plugin-1").
	//         WithLocalPlugin("./my-plugin-2")

	// Or use the WithPlugins method with an array:
	// plugins := []types.PluginConfig{
	//     *types.NewLocalPluginConfig("./plugin1"),
	//     *types.NewLocalPluginConfig("./plugin2"),
	// }
	// options.WithPlugins(plugins)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Execute query with plugin support
	fmt.Println("=== Asking Claude to use the demo plugin ===")
	fmt.Println()
	messages, err := sdk.Query(ctx, "Please use the /greet command from the demo plugin", options)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	// Stream messages
	fmt.Println("=== Claude Response ===")
	fmt.Println()
	for msg := range messages {
		switch m := msg.(type) {
		case *types.UserMessage:
			// User messages can have string or structured content
			if strContent, ok := m.Content.(string); ok {
				fmt.Printf("[User] %s\n", strContent)
			}

		case *types.AssistantMessage:
			for _, block := range m.Content {
				switch b := block.(type) {
				case *types.TextBlock:
					fmt.Printf("[Assistant] %s\n", b.Text)
				case *types.ToolUseBlock:
					fmt.Printf("[Tool Use] %s (id: %s)\n", b.Name, b.ID)
				}
			}

		case *types.ResultMessage:
			fmt.Printf("\n=== Query Complete ===\n")
			fmt.Printf("Session ID: %s\n", m.SessionID)
			fmt.Printf("Duration: %dms\n", m.DurationMs)
			if m.TotalCostUSD != nil {
				fmt.Printf("Cost: $%.4f\n", *m.TotalCostUSD)
			}
			fmt.Printf("Result: %s\n", m.Subtype)
		}
	}

	return nil
}
