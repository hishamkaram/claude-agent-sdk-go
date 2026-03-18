# Claude Agent SDK for Go

## What This Is

Go SDK for the Claude Code CLI subprocess protocol — handles subprocess transport, streaming message parsing, and control protocol (tool use, approval gates). Consumed exclusively by `agentd/internal/agents/claudecode.go`.

**Module**: `github.com/hishamkaram/claude-agent-sdk-go` | **Go**: 1.24+ | **License**: MIT

## Build & Test

```bash
go build ./...                    # Build all packages
go test -race -count=1 ./...      # Run all tests with race detector (run locally)
go vet ./...                      # Static analysis
golangci-lint run ./...           # Linter
```

## Architecture

| Package | Path | Purpose |
|---------|------|---------|
| root | `./` | Public API: `Client`, `Query()` function |
| types | `types/` | Public types: `Message`, `ClaudeAgentOptions`, content block variants, errors |
| transport | `internal/transport/` | Subprocess CLI transport, CLI discovery, stream reading |
| internal | `internal/` | Message parser, control protocol handler (private) |

## Critical Rules

**Errors**: Wrap every error with context:
```go
fmt.Errorf("pkg.Func: context: %w", err)
```
Use `%w` (not `%v`). Define sentinel errors with `errors.New()`. Check with `errors.Is()`/`errors.As()`.

**Logging**: `go.uber.org/zap` only — constructor-injected, never global. No `fmt.Println`, no `log.Print`, no `log.Printf`. Structured fields: `zap.String()`, `zap.Error()`.

**Tests**: Table-driven with `t.Parallel()` and `tt := tt`. Goroutine packages need `goleak.VerifyTestMain`. No global state, no `init()`.

**Goroutines**: Context-cancel every goroutine. `defer cancel()` after `context.WithCancel()`. `errgroup` for multiple goroutines.

**Commits**: `type(scope): subject` format (e.g., `fix(transport): handle nil input in subprocess`, `feat(types): add SlashCommandBlock`).

## Used By

`agentd/internal/agents/claudecode.go` imports:
```go
import (
    claude "github.com/hishamkaram/claude-agent-sdk-go"
    "github.com/hishamkaram/claude-agent-sdk-go/types"
)
```

Upgrade in agentd: `cd agentd && go get github.com/hishamkaram/claude-agent-sdk-go@<tag> && go mod tidy`

## Full Specification

See `AGENTD_PLAN.md` in the workspace root for interfaces, wire formats, phase plans, and architecture decisions.
See `.claude/rules/go-source.md`, `go-tests.md`, `error-handling.md`, `logging.md` for full coding standard rules.
