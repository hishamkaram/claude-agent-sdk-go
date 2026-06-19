package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	claude "github.com/hishamkaram/claude-agent-sdk-go"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// InteractiveClient demonstrates the interactive Client for multi-turn conversations.
// This allows back-and-forth conversation with Claude while maintaining session state.
func main() {
	if err := run(); err != nil {
		log.Fatalf("%v", err)
	}
}

// run wires up the client and drives the interactive session. It returns an error
// instead of calling log.Fatalf mid-flow so the deferred client.Close always runs
// (log.Fatalf would os.Exit and skip pending defers).
func run() error {
	ctx := context.Background()

	// Create options for the interactive client
	opts := types.NewClaudeAgentOptions()

	// Create client
	client, err := claude.NewClient(ctx, opts)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	// Connect to Claude
	fmt.Println("Connecting to Claude....")
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	fmt.Println("Connected! Type your questions (press Ctrl+C to exit)")
	fmt.Println("---")

	if err := interactiveLoop(ctx, client); err != nil {
		return err
	}

	fmt.Println("\nGoodbye!")
	return nil
}

// interactiveLoop reads prompts from stdin and streams Claude's responses until EOF
// (Ctrl+D) or a read error. EOF ends the loop cleanly; any other read error is
// returned to the caller.
func interactiveLoop(ctx context.Context, client *claude.Client) error {
	reader := bufio.NewReader(os.Stdin)
	for {
		// Read user input
		fmt.Print("You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			if strings.Contains(err.Error(), "EOF") {
				return nil
			}
			return fmt.Errorf("read input: %w", err)
		}

		prompt := strings.TrimSpace(input)
		if prompt == "" {
			continue
		}

		// Send query to Claude
		if err := client.Query(ctx, prompt); err != nil {
			fmt.Printf("Error sending query: %v\n", err)
			continue
		}

		// Receive and print responses
		printResponses(ctx, client)
	}
}

// printResponses streams one response turn to stdout, prefixing the first
// assistant chunk with "Claude: " and emitting blank lines after the result.
func printResponses(ctx context.Context, client *claude.Client) {
	foundResponse := false
	for msg := range client.ReceiveResponse(ctx) {
		switch msg.GetMessageType() {
		case "assistant":
			if !foundResponse {
				fmt.Print("Claude: ")
				foundResponse = true
			}
			if assistantMsg, ok := msg.(*types.AssistantMessage); ok {
				for _, block := range assistantMsg.Content {
					if textBlock, ok := block.(*types.TextBlock); ok {
						fmt.Print(textBlock.Text)
					}
				}
			}
		case "result":
			fmt.Println()
			fmt.Println()
		}
	}
}
