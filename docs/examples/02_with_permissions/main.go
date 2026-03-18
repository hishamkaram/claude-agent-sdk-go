/*
Example 2: Permission Control
Go Version

This example demonstrates how to control which tools Claude can use
by implementing a permission callback.
*/

package main

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/hishamkaram/claude-agent-sdk-go"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func checkPermission(ctx context.Context,
	toolName string,
	input map[string]interface{},
	permCtx types.ToolPermissionContext) (interface{}, error) {
	/*
		Permission callback that controls tool access.
		Returns PermissionResultAllow or PermissionResultDeny to grant or restrict tool use.
	*/

	fmt.Printf("🔐 Checking permission for tool: %s\n", toolName)

	// Block dangerous bash commands
	if toolName == "Bash" {
		command := ""
		if cmd, ok := input["command"].(string); ok {
			command = cmd
		}

		dangerousCommands := []string{"rm -rf", "dd if=/dev/zero", ":(){ :|:& };:"}

		for _, dangerous := range dangerousCommands {
			if strings.Contains(command, dangerous) {
				fmt.Printf("❌ Blocked dangerous command: %s\n", dangerous)
				return &types.PermissionResultDeny{
					Behavior:  "deny",
					Message:   "Dangerous command blocked",
					Interrupt: false,
				}, nil
			}
		}
	}

	fmt.Printf("✅ Permission granted for %s\n", toolName)
	return &types.PermissionResultAllow{
		Behavior: "allow",
	}, nil
}

func main() {
	ctx := context.Background()

	// Configure options with permission callback
	options := types.NewClaudeAgentOptions().
		WithCanUseTool(checkPermission).
		WithAllowedTools("Bash", "Write", "Read").
		WithSystemPrompt("You are a helpful assistant. You have access to tools but must ask before using them.")

	prompt := "Create a test file at /tmp/test.txt and list its contents"
	fmt.Printf("User: %s\n\n", prompt)

	messages, err := sdk.Query(ctx, prompt, options)
	if err != nil {
		panic(fmt.Sprintf("query failed: %v", err))
	}

	for message := range messages {
		if assistantMsg, ok := message.(*types.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if textBlock, ok := block.(*types.TextBlock); ok {
					fmt.Printf("Assistant: %s\n", textBlock.Text)
				}
			}
		}
	}
}
