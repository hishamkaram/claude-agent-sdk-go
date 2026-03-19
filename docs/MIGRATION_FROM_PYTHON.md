# Migration Guide: From Python to Go

This guide helps Python developers transition to the Go Agent SDK. While the APIs are different due to language differences, the concepts are the same.

## Table of Contents
1. [Quick Start Comparison](#quick-start-comparison)
2. [Configuration Migration](#configuration-migration)
3. [Query/Response Patterns](#queryresponse-patterns)
4. [Hook System Migration](#hook-system-migration)
5. [Permission Handling](#permission-handling)
6. [MCP Servers](#mcp-servers)
7. [Error Handling](#error-handling)
8. [Common Patterns](#common-patterns)

---

## Quick Start Comparison

### Python: Simple Query

```python
import asyncio
from claude_agent_sdk import Agent

async def main():
    client = Agent()

    # Stream messages
    async for message in client.query("What is 2+2?"):
        print(f"{message.type}: {message.content}")

asyncio.run(main())
```

### Go: Simple Query

```go
package main

import (
    "context"
    "fmt"
    sdk "github.com/schlunsen/claude-agent-sdk-go"
    "github.com/schlunsen/claude-agent-sdk-go/types"
)

func main() {
    ctx := context.Background()

    // Stream messages
    messages, _ := sdk.Query(ctx, "What is 2+2?", nil)
    for message := range messages {
        switch m := message.(type) {
        case *types.AssistantMessage:
            fmt.Printf("Assistant: %v\n", m.Content)
        case *types.ResultMessage:
            fmt.Printf("Done (subtype=%s)\n", m.Subtype)
        case *types.AuthStatusMessage:
            fmt.Printf("Auth: %v\n", m.Output)
        case *types.RateLimitEvent:
            fmt.Printf("Rate limit: %s\n", m.RateLimitInfo.Status)
        case *types.ToolProgressMessage:
            fmt.Printf("Tool %s running (%.1fs)\n", m.ToolName, m.ElapsedTimeSeconds)
        case *types.TaskStartedMessage:
            fmt.Printf("Task started: %s\n", m.Description)
        case *types.TaskNotificationMessage:
            fmt.Printf("Task done: %s\n", m.Summary)
        case *types.StatusMessage:
            // system subtype "status" — permission mode change, etc.
        case *types.HookResponseMessage:
            // system subtype "hook_response" — hook outcome
        case *types.UnknownMessage:
            // forward-compatible: unrecognized type, raw JSON preserved in m.RawJSON
        }
    }
}
```

---

## Configuration Migration

### Python: Dataclass Configuration

```python
options = AgentOptions(
    model="claude-opus-4-20250514",
    allowed_tools=["Bash", "Read", "Write"],
    system_prompt="You are a helpful assistant",
    max_turns=10,
)
```

### Go: Builder Pattern

```go
options := sdk.NewClaudeAgentOptions().
    WithModel("claude-opus-4-20250514").
    WithAllowedTools("Bash", "Read", "Write").
    WithSystemPrompt("You are a helpful assistant").
    WithMaxTurns(10)
```

**Key Differences:**
- Go uses method chaining (builder pattern)
- Each `With*()` method returns the options object for chaining
- Configuration is verified at compile-time, not runtime
- No need to import from multiple modules

---

## Query/Response Patterns

### Pattern 1: Basic Query

#### Python

```python
import asyncio
from claude_agent_sdk import Agent

async def main():
    client = Agent()

    async for message in client.query("What is Python?"):
        if message.type == "assistant":
            print(message.content)

asyncio.run(main())
```

#### Go

```go
package main

import (
    "context"
    "fmt"
    sdk "github.com/schlunsen/claude-agent-sdk-go"
    "github.com/schlunsen/claude-agent-sdk-go/types"
)

func main() {
    ctx := context.Background()

    messages, _ := sdk.Query(ctx, "What is Go?", nil)
    for msg := range messages {
        switch m := msg.(type) {
        case *types.AssistantMessage:
            for _, block := range m.Content {
                if textBlock, ok := block.(*types.TextBlock); ok {
                    fmt.Println(textBlock.Text)
                }
            }
        case *types.ResultMessage:
            fmt.Printf("Finished: subtype=%s turns=%d\n", m.Subtype, m.NumTurns)
        case *types.UnknownMessage:
            // Unrecognized type — raw JSON preserved in m.RawJSON for forward compat
        }
    }
}
```

**Migration Notes:**
- Python: `async for` → Go: `for ... range`
- Python: Message type string → Go: Type assertion with switch
- Python: Direct content access → Go: Iterate through content blocks

### Pattern 2: Interactive Client with Session

#### Python

```python
import asyncio
from claude_agent_sdk import Agent

async def main():
    client = Agent()

    # Connect to start session
    await client.connect()

    # Send first query
    await client.query("What is the capital of France?")
    messages = []
    async for msg in client.receive_response():
        messages.append(msg)

    # Continue conversation
    await client.query("And Germany?")
    async for msg in client.receive_response():
        messages.append(msg)

    await client.close()
    return messages
```

#### Go

```go
package main

import (
    "context"
    "fmt"
    sdk "github.com/schlunsen/claude-agent-sdk-go"
    "github.com/schlunsen/claude-agent-sdk-go/types"
)

func main() {
    ctx := context.Background()

    options := sdk.NewClaudeAgentOptions()
    client, _ := sdk.NewClient(ctx, options)
    defer client.Close(ctx)

    client.Connect(ctx)

    // Send first query
    client.Query(ctx, "What is the capital of France?")
    messages := []types.Message{}
    for msg := range client.ReceiveResponse(ctx) {
        messages = append(messages, msg)
    }

    // Continue conversation
    client.Query(ctx, "And Germany?")
    for msg := range client.ReceiveResponse(ctx) {
        messages = append(messages, msg)
    }

    client.Close(ctx)
}
```

---

## Hook System Migration

### Python: Hook Decorators

```python
from claude_agent_sdk import Agent, HookEvent

client = Agent()

@client.hook(HookEvent.PRE_TOOL_USE)
async def log_before_tool(input_data, hook_context):
    print(f"About to call: {hook_context.tool_name}")
    return {"continue": True}

@client.hook(HookEvent.POST_TOOL_USE)
async def log_after_tool(input_data, hook_context):
    print(f"Finished calling: {hook_context.tool_name}")
    return {"continue": True}
```

### Go: Hook Matchers in Options

```go
package main

import (
    "context"
    "fmt"
    sdk "github.com/schlunsen/claude-agent-sdk-go"
    "github.com/schlunsen/claude-agent-sdk-go/types"
)

func main() {
    // Pre-tool hook
    preHook := types.HookMatcher{
        Matcher: stringPtr(".*"),  // Regex to match tool names
        Hooks: []types.HookCallbackFunc{
            func(ctx context.Context, input interface{},
                toolUseID *string, hookCtx types.HookContext) (interface{}, error) {
                fmt.Printf("About to call: %s\n", hookCtx.ToolName)
                return map[string]interface{}{"continue": true}, nil
            },
        },
    }

    options := sdk.NewClaudeAgentOptions().
        WithHook(types.HookEventPreToolUse, preHook)

    // ... rest of code
}

func stringPtr(s string) *string {
    return &s
}
```

**Migration Notes:**
- Python: Decorators → Go: Explicit hook callbacks in options
- Python: Async functions → Go: Sync functions returning error
- Python: `@client.hook()` → Go: `.WithHook(event, matcher)`
- Python: Hook matcher per event → Go: Regex patterns in HookMatcher

---

## Permission Handling

### Python: Permission Callbacks

```python
from claude_agent_sdk import Agent, PermissionBehavior

async def check_permissions(tool_name: str, input_data: dict, context):
    """Control which tools Claude can use"""

    # Deny dangerous commands
    if tool_name == "Bash":
        if "rm -rf" in input_data.get("command", ""):
            return PermissionBehavior.DENY

    return PermissionBehavior.ALLOW

client = Agent(
    can_use_tool=check_permissions,
    permission_mode="plan"
)
```

### Go: Permission Callbacks

```go
package main

import (
    "context"
    "strings"
    sdk "github.com/schlunsen/claude-agent-sdk-go"
    "github.com/schlunsen/claude-agent-sdk-go/types"
)

func checkPermissions(ctx context.Context,
    toolName string,
    input map[string]interface{},
    permCtx types.ToolPermissionContext) (interface{}, error) {

    // Deny dangerous commands
    if toolName == "Bash" {
        cmd := input["command"].(string)
        if strings.Contains(cmd, "rm -rf") {
            return &types.PermissionResultDeny{
                Behavior:  "deny",
                Message:   "Dangerous command blocked",
                Interrupt: false,
            }, nil
        }
    }

    return &types.PermissionResultAllow{
        Behavior: "allow",
    }, nil
}

func main() {
    options := sdk.NewClaudeAgentOptions().
        WithCanUseTool(checkPermissions)

    // ... rest of code
}
```

**Migration Notes:**
- Python: Async callback → Go: Sync callback
- Python: Return PermissionBehavior enum → Go: Return struct (Allow/Deny)
- Python: tool_name as parameter → Go: toolName parameter
- Python: Dictionary input → Go: map[string]interface{} input

---

## MCP Servers

### Python: Using Decorator

```python
from claude_agent_sdk import create_sdk_mcp_server, Agent
from mcp import Tool

# Define tools with decorator
@Tool("add")
async def add_tool(a: float, b: float) -> float:
    return a + b

@Tool("multiply")
async def multiply_tool(a: float, b: float) -> float:
    return a * b

# Create server
server = create_sdk_mcp_server(
    name="calculator",
    tools=[add_tool, multiply_tool]
)

# Use with agent
client = Agent(mcp_servers={"calculator": server})
```

### Go: Using Factory Function

```go
package main

import (
    "context"
    sdk "github.com/schlunsen/claude-agent-sdk-go"
    "github.com/schlunsen/claude-agent-sdk-go/types"
)

func main() {
    // Create server with factory
    calculator, _ := types.NewSDKMCPServer("calculator",
        types.Tool{
            Name:        "add",
            Description: "Add two numbers",
            Handler: func(ctx context.Context, args map[string]any) (any, error) {
                a, _ := args["a"].(float64)
                b, _ := args["b"].(float64)
                return a + b, nil
            },
        },
        types.Tool{
            Name:        "multiply",
            Description: "Multiply two numbers",
            Handler: func(ctx context.Context, args map[string]any) (any, error) {
                a, _ := args["a"].(float64)
                b, _ := args["b"].(float64)
                return a * b, nil
            },
        },
    )

    options := sdk.NewClaudeAgentOptions().
        WithMCPServer("calculator", calculator)

    // ... rest of code
}
```

**Migration Notes:**
- Python: `@Tool` decorator → Go: `types.Tool` struct
- Python: Async tool functions → Go: Handler functions
- Python: `create_sdk_mcp_server()` → Go: `types.NewSDKMCPServer()`
- Python: Tools list → Go: Variadic arguments

---

## Error Handling

### Python: Exception Handling

```python
from claude_agent_sdk import (
    Agent,
    PermissionDeniedError,
    CLINotFoundError,
)

try:
    client = Agent()
    async for msg in client.query("Hello"):
        print(msg)

except PermissionDeniedError as e:
    print(f"Permission denied for {e.tool_name}")

except CLINotFoundError as e:
    print(f"CLI not found: {e}")

except Exception as e:
    print(f"Error: {type(e).__name__}: {e}")
```

### Go: Error Type Checking

```go
package main

import (
    "errors"
    "fmt"
    sdk "github.com/schlunsen/claude-agent-sdk-go"
    "github.com/schlunsen/claude-agent-sdk-go/types"
)

func main() {
    ctx := context.Background()

    messages, err := sdk.Query(ctx, "Hello", nil)

    var permError *types.PermissionDeniedError
    var cliError *types.CLINotFoundError

    if errors.As(err, &permError) {
        fmt.Printf("Permission denied for %s\n", permError.ToolName)
    } else if errors.As(err, &cliError) {
        fmt.Printf("CLI not found: %s\n", cliError.Message)
    } else if err != nil {
        fmt.Printf("Error: %v\n", err)
    }

    // Or use helper functions
    if types.IsPermissionDeniedError(err) {
        fmt.Println("Permission denied")
    }

    if types.IsCLINotFoundError(err) {
        fmt.Println("CLI not found")
    }
}
```

**Migration Notes:**
- Python: `try/except` → Go: `errors.As()` type assertions
- Python: Exception class checking → Go: Type predicates or Is() methods
- Python: `isinstance(e, Type)` → Go: `errors.As(err, &var)`
- Python: Access attributes → Go: Access struct fields

---

## Common Patterns

### Pattern: Retry Logic

#### Python

```python
import asyncio
from tenacity import retry, stop_after_attempt, wait_exponential

@retry(
    stop=stop_after_attempt(3),
    wait=wait_exponential(multiplier=1, min=2, max=10)
)
async def query_with_retry(prompt: str):
    client = Agent()
    async for msg in client.query(prompt):
        yield msg
```

#### Go

```go
package main

import (
    "context"
    "fmt"
    "time"
    sdk "github.com/schlunsen/claude-agent-sdk-go"
)

func queryWithRetry(ctx context.Context, prompt string, maxRetries int) (<-chan interface{}, error) {
    var lastErr error

    for attempt := 0; attempt < maxRetries; attempt++ {
        messages, err := sdk.Query(ctx, prompt, nil)
        if err == nil {
            return messages, nil
        }

        lastErr = err
        if attempt < maxRetries-1 {
            // Exponential backoff: 2^attempt seconds
            backoff := time.Duration(1<<uint(attempt)) * time.Second
            time.Sleep(backoff)
        }
    }

    return nil, lastErr
}
```

### Pattern: Concurrent Requests

#### Python

```python
import asyncio
from claude_agent_sdk import Agent

async def main():
    client = Agent()

    # Run multiple queries concurrently
    tasks = [
        client.query("What is Python?"),
        client.query("What is Go?"),
        client.query("What is Rust?"),
    ]

    results = await asyncio.gather(*tasks)
    return results
```

#### Go

```go
package main

import (
    "context"
    "fmt"
    "sync"
    sdk "github.com/schlunsen/claude-agent-sdk-go"
)

func main() {
    ctx := context.Background()
    questions := []string{
        "What is Python?",
        "What is Go?",
        "What is Rust?",
    }

    // Use WaitGroup to wait for all goroutines
    var wg sync.WaitGroup
    results := make([]interface{}, len(questions))
    mu := sync.Mutex{}

    for i, question := range questions {
        wg.Add(1)
        go func(idx int, q string) {
            defer wg.Done()

            messages, _ := sdk.Query(ctx, q, nil)
            for msg := range messages {
                mu.Lock()
                results[idx] = msg
                mu.Unlock()
            }
        }(i, question)
    }

    wg.Wait()
    fmt.Println(results)
}
```

### Pattern: Stream Processing with Backpressure

#### Python

```python
import asyncio
from claude_agent_sdk import Agent

async def main():
    client = Agent()

    async for message in client.query("List 10 items"):
        # Process message
        print(f"Received: {message}")

        # Artificial backpressure: sleep between messages
        await asyncio.sleep(0.5)
```

#### Go

```go
package main

import (
    "context"
    "fmt"
    "time"
    sdk "github.com/schlunsen/claude-agent-sdk-go"
)

func main() {
    ctx := context.Background()

    messages, _ := sdk.Query(ctx, "List 10 items", nil)
    for msg := range messages {
        // Process message
        fmt.Printf("Received: %v\n", msg)

        // Artificial backpressure: sleep between messages
        time.Sleep(500 * time.Millisecond)
    }
}
```

---

## Checklist for Migration

- [ ] Replace `async def` with regular `func`
- [ ] Replace `await` calls with `<-chan` receives
- [ ] Replace `async for` with `for ... range`
- [ ] Convert dataclass config to builder pattern
- [ ] Change exception handling to error type checking
- [ ] Replace decorator hooks with `.WithHook()`
- [ ] Convert async callbacks to sync callbacks
- [ ] Update package imports
- [ ] Add `context.Context` to async operations
- [ ] Replace `asyncio.gather()` with goroutines and channels
- [ ] Update error handling to use `errors.As()`
- [ ] Test with real Claude Code CLI

---

## Performance Differences

### Python vs Go for Similar Operations

```python
# Python: ~100ms startup + network latency
client = Agent()
async for msg in client.query("What is 2+2?"):
    print(msg)

# Go: ~10ms startup + network latency
messages, _ := sdk.Query(ctx, "What is 2+2?", nil)
for msg := range messages {
    fmt.Println(msg)
}
```

**Go advantages:**
- Faster startup (no Python runtime)
- Lower memory overhead
- Better for high-concurrency scenarios
- Compiled binary (easier deployment)

**Python advantages:**
- Faster development cycle
- Easier prototyping
- Rich scientific ecosystem
- More familiar to many users

---

## Troubleshooting

| Issue | Python | Go |
|-------|--------|-----|
| Module not found | `pip install` | `go get` |
| Import error | Check Python path | Check GOPATH |
| Type error | Runtime error | Compile error |
| Async issue | `RuntimeError` | Goroutine issue |
| Channel closed | N/A | `panic` if not checked |

---

## Further Reading

- [Architecture Differences](./ARCHITECTURE.md)
- [Feature Parity Guide](./FEATURE_PARITY.md)
- [Python SDK Docs](https://github.com/anthropics/claude-agent-sdk-python)
- [Go SDK API Reference](../README.md)

---

**Have questions?** Open an [issue](https://github.com/schlunsen/claude-agent-sdk-go/issues) or start a [discussion](https://github.com/schlunsen/claude-agent-sdk-go/discussions).
