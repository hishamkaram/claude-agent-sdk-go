# Changelog

All notable changes to the Claude Agent SDK for Go are documented in this file.

## [Unreleased]

### Changed
- Permission-mode discovery now fails closed when the installed CLI cannot be
  inspected; the SDK no longer invents a static fallback set. Canonical
  `default` is translated to the provider spelling reported by the CLI and is
  omitted from startup arguments when the CLI default is requested.
- Reorganized public client, query, options, and message type implementation
  across smaller domain files for maintainability. This is intended as a
  backward-compatible source-layout change with no public API behavior change.

### Added
- `Client.ListModels()` and `DiscoverRuntimeCapabilities()` expose the
  connected CLI's provider-owned model catalog, model capability flags,
  permission modes, and CLI version.
- `Client.SupportedPermissionModes()` and `DiscoverSupportedPermissionModes()`
  for discovering the installed Claude Code CLI's supported permission modes
  from structured init metadata or `claude --help`.
- Real-CLI integration test suite under `tests/integration_*_test.go`,
  gated by the `integration` Go build tag. Covers every public `Client`
  method, the `sessions` package API, all 23 `HookEvent` constants, the
  MCP lifecycle, cross-cutting concurrency, and a CLI-flag drift detector.
  38 passing, 0 failing against a live CLI.
- `tests/coverage_matrix.md` documenting per-method status.
- `tests/fixtures/` placeholder directory for wire-shape probe baselines.
- `.github/workflows/integration.yml` — nightly workflow that runs the
  integration suite against `secrets.ANTHROPIC_API_KEY`. Manual dispatch
  supports `run_turns` and `stress` inputs.
- `make test-integration-quota` Makefile target for the full suite
  including turn-driving tests (requires `CLAUDE_SDK_RUN_TURNS=1`).
- README "Integration testing" section documenting install, env gates,
  and local run recipes.

### Fixed
- `make test-integration` now correctly passes `-tags=integration` —
  previously the tagged suite compiled but never ran anywhere (latent).
- `AgentDefinition` now serializes documented camelCase fields in `--agents`
  payloads (`disallowedTools`, `maxTurns`, `mcpServers`, `hooks`,
  `initialPrompt`, `permissionMode`, `isolation`, `color`) and exposes current
  definition fields (`memory`, `background`, `effort`) while preserving legacy
  snake_case unmarshal compatibility.
- Added `PermissionModeAuto` and `WithSessionAgent`, emitting the supported
  `--agent <name>` CLI flag for sessions that should run as a configured agent.

### Known issues (surfaced by the new flag drift test)
- `WithAgentProgressSummaries` and `WithSubagentExecution` emit CLI flags
  (`--agent-progress-summaries`, `--subagent-execution`) that are not
  accepted by Claude Code CLI 2.1.132. They are intentionally retained for
  forward-compatible callers, but setting either option today causes
  `Connect()` to fail with an unknown-flag error. Tracked by
  `TestFlags_UnsupportedFlagsAreDocumented`.

## [0.2.9] - 2025-12-07

### Added
- Beta API feature support via `WithBetas()` and `WithBeta()` builder methods
- Support for Anthropic beta APIs like extended context windows (`context-1m-2025-08-07`)
- Pass beta feature flags to Claude Code CLI via `--betas` flag
- Comprehensive example demonstrating beta feature usage (`examples/with_betas/`)
- Extensive unit tests for betas functionality in both options and transport layers

### Details
- Implements feature parity with Python SDK v0.1.12+ for beta support
- Full CLI flag generation testing for `--betas` argument passing
- Supports multiple simultaneous beta features
- Method chaining support for fluent API usage
- Closes #23

## [0.2.2] - 2025-10-19

### Added
- Permission mode support with proper CLI flag passing
- Verbose logging option that can be enabled via `ClaudeAgentOptions.Verbose`
- System prompt support via `--system-prompt` flag to Claude CLI
- Permission prompt tool flag (`--permission-prompt-tool stdio`) for control protocol

### Fixed
- Control request handling for CLI-initiated requests without `request_id`
  - SDK now automatically generates request IDs for CLI-initiated control requests
  - Fixes permission callbacks that were failing silently
- Request ID parsing from top-level field in control_request messages
  - CLI sends `request_id` at top level, not inside request object
  - Fixes issue where control responses weren't matched to requests
  - Permission approvals are now properly recognized by CLI
- Client now properly passes options to transport layer
- Control protocol initialization and bidirectional communication

### Changed
- Enhanced control request logging for better debugging
- Updated `SubprocessCLITransport` to accept and use `ClaudeAgentOptions`
- Improved `SystemMessage` type with `RequestID` field for control protocol

## [0.1.0] - 2025-10-18

### Initial Release - Complete Port from Python SDK

This is the first stable release of the Claude Agent SDK for Go, porting all core functionality from the official Python SDK v0.1.3.

#### Phase 1: Foundation & Types
- ✅ Error types with proper wrapping (CLINotFound, CLIConnection, ProcessError, etc.)
- ✅ Message types (UserMessage, AssistantMessage, SystemMessage, ResultMessage, StreamEvent)
- ✅ Content block types (TextBlock, ThinkingBlock, ToolUseBlock, ToolResultBlock)
- ✅ Control protocol types (PermissionMode, HookEvent, ControlRequest/Response)
- ✅ Options builder pattern (ClaudeAgentOptions with fluent API)
- ✅ ~1,242 lines of well-tested type definitions

#### Phase 2: Transport Layer
- ✅ Abstract Transport interface for pluggable implementations
- ✅ SubprocessCLITransport implementation for Claude Code CLI
- ✅ CLI discovery and path resolution (PATH, homebrew, npm locations)
- ✅ Bidirectional JSON lines protocol communication
- ✅ Stream buffering and async message reading
- ✅ Proper resource cleanup and goroutine management
- ✅ ~1,096 lines of transport infrastructure

#### Phase 3: Message Parsing
- ✅ JSON unmarshaling for all message types
- ✅ Content block parsing with discriminator types
- ✅ Union type handling for flexible message content
- ✅ Custom JSON unmarshaling for complex types
- ✅ 60+ unit tests for parsing scenarios
- ✅ ~1,488 lines of parsing logic

#### Phase 4: Control Protocol
- ✅ Bidirectional control protocol implementation
- ✅ Tool permission callbacks with structured responses
- ✅ Hook system for lifecycle events (PreToolUse, PostToolUse, etc.)
- ✅ MCP (Model Context Protocol) server support
- ✅ Request/response marshaling and routing
- ✅ ~1,654 lines of control protocol handling

#### Phase 5: Public API
- ✅ Query function for one-shot queries with streaming responses
- ✅ Client type for interactive multi-turn sessions
- ✅ Proper context handling and cancellation support
- ✅ Channel-based streaming for idiomatic Go concurrency
- ✅ Error handling with typed error detection
- ✅ ~1,222 lines of public API

#### Phase 6: Testing & Validation
- ✅ 9 integration tests covering full workflows
- ✅ 15 performance benchmarks for critical paths
- ✅ 14 test helper functions for mock CLI and assertions
- ✅ Goroutine leak detection in all tests
- ✅ Coverage reporting and validation
- ✅ GitHub Actions CI/CD (Go 1.20, 1.21, 1.22)
- ✅ 60%+ code coverage across packages
- ✅ ~2,079 lines of test code

#### Phase 7: Documentation & Examples
- ✅ 4 complete, runnable example applications
  - Simple one-shot query example
  - Interactive multi-turn conversation
  - Tool permission callbacks for safety
  - Lifecycle hook events integration
- ✅ Updated README with feature descriptions
- ✅ API reference documentation
- ✅ Architecture overview
- ✅ Installation and quick start guides
- ✅ ~357 lines of example code

#### Phase 8: Polish & Release
- ✅ Version file (0.1.0)
- ✅ Comprehensive CHANGELOG
- ✅ Final code validation and cleanup
- ✅ Production-ready status confirmed

### Features

#### Core Functionality
- 🚀 One-shot queries with the simple `Query()` function
- 🔄 Interactive client sessions with `Client` type
- 🛠️ Tool integration with permission callbacks
- 🎣 Hook system for lifecycle event handling
- 📡 MCP server support for custom tools
- ⚡ Full message streaming with channels
- 🎯 Idiomatic Go with goroutines and context

#### Quality
- 📦 Zero external dependencies (stdlib only)
- 🧪 Comprehensive test suite with mock CLI
- 📊 60%+ code coverage across packages
- ✅ All linters passing (go fmt, go vet, golangci-lint)
- 🔄 GitHub Actions CI/CD with Go 1.20, 1.21, 1.22
- 📝 Extensive documentation and examples

#### Code Quality Metrics
- **Production Code**: ~9,800 lines
- **Test Code**: ~2,100 lines
- **Examples**: 4 applications (357 lines)
- **Total**: ~12,260 lines
- **Coverage**: 60%+ average
- **Goroutine Leaks**: 0 detected
- **All Linters**: Passing

### Supported Go Versions
- Go 1.24+

### Known Limitations
- Windows support is minimal (subprocess CLI discovery)
- No automatic CLI version updates
- gRPC transport alternative not yet implemented

### Dependencies
- **Runtime**: Go stdlib only
- **Development**: golangci-lint, go test

### Breaking Changes
None - this is the first release.

### Bug Fixes
- Fixed CLI invocation command flags to use correct protocol format (#9)
  - Changed from `agent --stdio` to `--print --input-format=stream-json --output-format=stream-json --verbose`
  - Updated query message structure to match Python SDK format with nested message object
  - Added `parent_tool_use_id` and `session_id` fields to protocol messages
- Added support for nested message format in AssistantMessage parsing
  - Handle nested `message.content` format from Claude CLI responses
  - Extract model field from nested message structure
  - Fall back to top-level content for backward compatibility
- Fixed interactive client connection hang and added verbose logging (#10)
  - Made verbose logging configurable via `CLAUDE_AGENT_VERBOSE` environment variable
  - Fixed Client.Connect() to wait for control protocol initialization
  - Added stderr logging to file at `~/.claude/agents_server/cli_stderr.log`
  - Improved error handling in control protocol initialization

### Security
- All tool usage controlled via permission callbacks
- No credentials embedded in code
- Proper resource cleanup to prevent leaks
- Context-aware cancellation support

### Contributors
- Rasmus Schlunsen (https://github.com/schlunsen)

### Acknowledgments
- Official [Claude Agent SDK for Python](https://github.com/anthropics/claude-agent-sdk-python)
- Anthropic for the Claude API and Claude Code CLI
