# Claude Agent SDK for Go

[![Go 1.25](https://img.shields.io/badge/Go-1.25-00add8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Go SDK for the Claude Code CLI subprocess protocol — spawns Claude Code as a child process, communicates via JSON lines, and provides a typed Go API for queries, multi-turn conversations, and tool use hooks.

## Why this SDK?

Claude Code exposes a powerful subprocess protocol over JSON lines, but consuming it directly means parsing raw JSON, managing subprocess lifecycle, handling streaming, and implementing the 23-event hook system from scratch. This SDK handles all of that and exposes a clean, typed Go API.

## Feature highlights

| Feature | Value |
|---------|-------|
| **Typed Go API** | Work with `Message`, `ContentBlock`, and `HookEvent` structs instead of parsing raw JSON. The compiler catches protocol mismatches before runtime. |
| **23 hook event callbacks** | Intercept tool use, permission requests, session lifecycle, notifications, and more — all with strongly-typed input/output structs and a consistent callback pattern. |
| **One-shot and interactive modes** | `Query()` for fire-and-forget prompts that return a channel of messages. `Client` for multi-turn conversations with session persistence, resume, and fork. |
| **Zero external runtime deps** | Only `golang.org/x/net` at build time. No CGO, no gRPC, no framework overhead. |

## Prerequisites

- Go 1.25+
- Claude Code CLI installed: `npm install -g @anthropic-ai/claude-code`
- Authentication (one required):
  - `CLAUDE_API_KEY` environment variable (pay-as-you-go)
  - `CLAUDE_CODE_OAUTH_TOKEN` environment variable (Max subscription)

## Getting started

```bash
go get github.com/hishamkaram/claude-agent-sdk-go
```

### One-shot query

```go
import claude "github.com/hishamkaram/claude-agent-sdk-go"

msgs := claude.Query(ctx, "Explain this Go code", claude.NewClaudeAgentOptions())
for msg := range msgs {
    fmt.Println(msg)
}
```

### Interactive multi-turn session

```go
client, err := claude.NewClient(ctx, claude.NewClaudeAgentOptions())
if err != nil {
    log.Fatal(err)
}
defer client.Close()

if err := client.Connect(); err != nil {
    log.Fatal(err)
}

response := client.Query(ctx, "What files are in this directory?")
```

### Hook event interception

```go
opts := claude.NewClaudeAgentOptions().
    WithPermissionRequestHandler(func(ctx context.Context, e types.PermissionRequestInput) types.PermissionRequestResult {
        // inspect e.ToolName, e.ToolInput — allow or deny
        return types.PermissionRequestResult{Behavior: types.PermissionAllow}
    })

client, _ := claude.NewClient(ctx, opts)
```

## What it does

- Spawns Claude Code CLI as a subprocess with `--agent --stdio` flags
- Streams typed messages (assistant output, tool use, errors, hooks) via channels
- Supports one-shot queries (`Query()`) and interactive multi-turn sessions (`Client`)
- Provides 23 hook event callbacks for intercepting tool use, permissions, and lifecycle events
- Handles MCP (Model Context Protocol) server configuration and tool routing

## Project structure

```
claude-agent-sdk-go/
├── query.go                # Query() — one-shot prompt → channel of messages
├── client.go               # Client — multi-turn interactive sessions
├── sessions.go             # Session management (list, info, rename, tag, fork, subagents)
├── types/                  # Public types (all exported)
│   ├── messages.go         # Message types, content blocks (text, tool_use, thinking, etc.)
│   ├── control.go          # 23 hook event types, hook callbacks, permission results
│   ├── options.go          # ClaudeAgentOptions builder (With* methods)
│   ├── errors.go           # Typed errors with predicate functions
│   ├── mcp.go              # MCP server config and tool types
│   ├── mcp_types.go        # MCP type definitions
│   └── session_types.go    # Session-related types
├── internal/
│   ├── transport/          # Subprocess CLI transport, CLI discovery, stream reader
│   │   ├── subprocess_cli.go  # Spawns Claude CLI, manages stdin/stdout pipes
│   │   ├── cli_version.go     # CLI version detection and validation
│   │   ├── cli_discovery.go   # CLI binary discovery
│   │   ├── stream.go          # JSON line reader with configurable buffer
│   │   └── transport.go       # Transport interface
│   ├── message_parser.go   # JSON to Go type conversion
│   ├── query.go            # Control protocol handler
│   └── log/                # Internal logging utilities
├── examples/               # Working examples
│   ├── simple_query/       # Basic one-shot query
│   ├── interactive_client/ # Multi-turn conversation
│   ├── with_permissions/   # Tool permission callbacks
│   ├── with_hooks/         # Hook event handling
│   ├── with_plugins/       # Plugin system example
│   ├── with_betas/         # Beta features example
│   ├── plugins/            # Plugin definitions
│   └── mcp_server_simple/  # MCP server example
├── Makefile                # build, test, test-all, lint, coverage
├── VERSION                 # 0.5.1
└── go.mod
```

## Available commands

| Command | Description |
|---------|-------------|
| `make build` | Build all packages |
| `make test` | Run unit tests (skips integration — no Claude CLI needed) |
| `make test-all` | Run all tests including integration (requires Claude CLI) |
| `make test-integration` | Run real-CLI integration tests, no-quota subset (skips turn-driving tests) |
| `make test-integration-quota` | Run the full real-CLI integration suite including model turns (burns tokens) |
| `make coverage` | Unit tests with coverage report |
| `make lint` | Run `go vet` + golangci-lint |
| `make fmt` | Format all Go files |

## Integration testing

The SDK ships a real-CLI integration suite under `tests/integration_*_test.go`,
gated by the `integration` Go build tag. These tests spawn the actual
`claude` subprocess against real network endpoints — they catch wire-shape
drift between the SDK and the CLI that mock-based tests cannot see.

### Install the CLI

```bash
npm install -g @anthropic-ai/claude-code
claude --version
```

### Environment gates

Each integration test is guarded by helpers in `tests/test_helpers.go`:

- `requireClaude(t)` — skips if the `claude` binary is not on PATH or in a
  common install location (`~/.claude/local/claude`, `~/.npm-global/bin/...`,
  `/usr/local/bin/...`, `/opt/homebrew/bin/...`). Override with
  `CLAUDE_CLI_PATH=/path/to/claude`.
- `requireAuth(t)` — skips unless one of: `ANTHROPIC_API_KEY`,
  `CLAUDE_API_KEY`, or `~/.claude/.credentials.json` (via `claude login`).
- `requireRunTurns(t)` — skips unless `CLAUDE_SDK_RUN_TURNS=1`. Use for
  tests that drive a full model turn and therefore spend tokens.

### Running locally

```bash
# Cheap no-quota subset — exercises init/transport/control-protocol paths:
make test-integration

# Full suite including turn-driving tests (~$1-3 in tokens):
CLAUDE_SDK_RUN_TURNS=1 make test-integration-quota

# 5-count stability check (recommended before shipping a new test):
go test -tags=integration -race -count=5 -p 1 ./tests/... -timeout=1800s
```

### CI

`.github/workflows/integration.yml` runs the suite nightly against
`secrets.ANTHROPIC_API_KEY`. Manual `workflow_dispatch` triggers default to
the no-quota subset; flip `run_turns=true` to include quota-gated tests.

See `tests/coverage_matrix.md` for the per-method coverage table.

## Environment variables

| Variable | Required | Description |
|----------|----------|-------------|
| `ANTHROPIC_API_KEY` | One of these | Anthropic API key (preferred; what CI uses) |
| `CLAUDE_API_KEY` | One of these | Legacy alias for `ANTHROPIC_API_KEY` |
| `CLAUDE_CODE_OAUTH_TOKEN` | One of these | OAuth token (Max subscription) |
| `CLAUDE_CLI_PATH` | Optional | Absolute path to the `claude` binary (override PATH lookup) |
| `CLAUDE_SDK_RUN_TURNS` | Optional | Set to `1` to enable quota-gated integration tests |
| `CLAUDE_AGENT_SDK_SKIP_VERSION_CHECK` | Optional | Skip Claude CLI version validation |

## Known limitations

- The SDK spawns Claude Code as a subprocess — it does not make direct API calls to Anthropic
- Requires Claude Code CLI v2.0.0+ (checked at connect time, skip with `CLAUDE_AGENT_SDK_SKIP_VERSION_CHECK=1`)
- `make test` runs in `-short` mode (no Claude CLI needed); `make test-all` spawns real Claude processes and requires authentication
- `v0.2.0` is retracted in go.mod — do not use that version
- Two option fields emit CLI flags that the current `claude` CLI (0.5.1) does
  not recognize — setting either will cause Connect() to fail with an
  unknown-flag error. Surfaced by `TestFlags_UnsupportedFlagsAreDocumented`:
  - `WithAgentProgressSummaries` → `--agent-progress-summaries`
  - `WithSubagentExecution` → `--subagent-execution`

## Contributing

Issues and pull requests welcome. Run `make test` before submitting — no Claude CLI required for the unit test suite.

## License

[MIT](LICENSE)
