# Claude Agent SDK for Go

## What This Is

Go SDK for the Claude Code CLI subprocess protocol — handles subprocess transport, streaming message parsing, and control protocol (tool use, approval gates).

**Module**: `github.com/hishamkaram/claude-agent-sdk-go` | **Go**: 1.24+ | **License**: MIT

## Build & Test

```bash
go build ./...                    # Build all packages
go test -race -count=1 -p 4 ./...  # Run all tests with race detector (parallelism limited)
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

## Configuration Options

### Option Delivery Mechanisms

| Mechanism | Options | Implementation |
|-----------|---------|----------------|
| CLI flags | `--effort`, `--fallback-model`, `--session-id`, `--no-session-persistence`, `--json-schema`, `--resume-session-at`, `--tools`, `--debug-file`, `--strict-mcp-config` | `buildCommandArgs()` in `internal/transport/subprocess_cli.go` |
| Settings JSON (`--settings`) | Thinking, Sandbox, EnableFileCheckpointing, ToolConfig | `buildSettingsJSON()` in `internal/transport/subprocess_cli.go` |
| Transport hook | SpawnProcess (custom ProcessSpawner) | `connectWithCustomSpawner()` in `internal/transport/subprocess_cli.go` |
| Init control protocol | PromptSuggestions, JsonSchema | `Initialize()` in `internal/query.go` |

### New Types (types/options.go)

| Type | Purpose |
|------|---------|
| `EffortLevel` | String enum: `Low`, `Medium`, `High`, `Max` |
| `ThinkingConfig` | Thinking mode: `Type` (adaptive/enabled/disabled), `BudgetTokens` |
| `OutputFormat` | Structured output: `Type` (json), `Schema`, `Name` |
| `SandboxConfig` | Sandbox controls: `Network` (SandboxNetworkConfig), `Filesystem` (SandboxFilesystemConfig) |
| `SpawnOptions` | Custom process spawner input: `Command`, `Args`, `CWD`, `Env` |
| `SpawnedProcess` | Interface for custom-spawned process: `Stdin()`, `Stdout()`, `Stderr()`, `Kill()`, `Wait()`, `ExitCode()`, `Killed()` |
| `ProcessSpawner` | Function type: `func(ctx, SpawnOptions) (SpawnedProcess, error)` — inject via `WithSpawnProcess()` |
| `ToolConfig` | Built-in tool configuration: `Bash` (BashToolConfig), `Computer` (ComputerToolConfig) |

### Hook Events (types/control.go)

23 total hook events (6 existing + 17 new). Each has an input type embedding `BaseHookInput`. Events with output types implement `HookSpecificOutput` interface.

New events: `PostToolUseFailure`, `Notification`, `SessionStart`, `SessionEnd`, `StopFailure`, `SubagentStart`, `PostCompact`, `PermissionRequest`, `Setup`, `TeammateIdle`, `TaskCompleted`, `Elicitation`, `ElicitationResult`, `ConfigChange`, `WorktreeCreate`, `WorktreeRemove`, `InstructionsLoaded`

## Full Specification

See `.claude/rules/go-source.md`, `go-tests.md`, `error-handling.md`, `logging.md` for full coding standard rules.
