# Go SDK vs Python SDK Feature Parity Guide

## At a Glance

| Metric | Value |
|--------|-------|
| **Go SDK Version** | v0.5.1 |
| **Python SDK Version** | v0.1.18+ |
| **Feature Parity** | ~95% |
| **Status** | Production-Ready |
| **Last Updated** | 2026-05-07 |

## Overview

This document provides a comprehensive comparison of features between the Go Agent SDK and the official Python Agent SDK. Both SDKs are fully functional and production-ready for most use cases.

### Key Differences in Approach

- **Language Differences**: Python (async/await) vs Go (goroutines/channels)
- **Configuration**: Python (Pydantic dataclasses) vs Go (builder pattern)
- **Threading**: Python (GIL) vs Go (explicit concurrency)
- **Error Handling**: Python (exceptions) vs Go (error types)

---

## Complete Feature Comparison Matrix

### Core API & Session Management

| Feature | Python | Go | Notes |
|---------|--------|-----|-------|
| **Query (one-shot)** | ✅ | ✅ | Both SDKs support simple one-off requests |
| **Client (interactive)** | ✅ | ✅ | Full bidirectional communication |
| **Message streaming** | ✅ | ✅ | Python: async generators, Go: channels |
| **Session resumption** | ✅ | ✅ | Continue previous conversations with session ID |
| **Session forking** | ✅ | ✅ | Create new branch from existing session |
| **Message history** | ✅ | ✅ | Access and replay message sequences |
| **Model selection** | ✅ | ✅ | Switch between available models |
| **Max turns control** | ✅ | ✅ | Limit conversation length |

### Message Types & Content Blocks

| Feature | Python | Go | Notes |
|---------|--------|-----|-------|
| **User messages** | ✅ | ✅ | User input handling; `IsReplay` field for replayed messages |
| **Assistant messages** | ✅ | ✅ | Claude responses |
| **System messages** | ✅ | ✅ | System notifications; generic subtype routing |
| **Result messages** | ✅ | ✅ | Final cost/token info; `Subtype`, `Errors`, `PermissionDenials`, `ModelUsageMap`, `StopReason`, `TotalCostUSD`, `UUID` |
| **Text content blocks** | ✅ | ✅ | Plain text responses |
| **Tool use blocks** | ✅ | ✅ | Tool invocation requests |
| **Tool result blocks** | ✅ | ✅ | Tool execution results |
| **Thinking blocks** | ✅ | ✅ | Extended thinking output (v0.1.17+) |
| **ToolProgressMessage** | ✅ | ✅ | Periodic tool execution progress events |
| **AuthStatusMessage** | ✅ | ✅ | Authentication flow status |
| **ToolUseSummaryMessage** | ✅ | ✅ | Summary of grouped tool uses |
| **RateLimitEvent** | ✅ | ✅ | Rate limit encountered events |
| **PromptSuggestionMessage** | ✅ | ✅ | Predicted next user prompt |
| **CompactBoundaryMessage** | ✅ | ✅ | Context compaction boundary (system subtype) |
| **StatusMessage** | ✅ | ✅ | System status/permission mode change (system subtype) |
| **HookStartedMessage** | ✅ | ✅ | Hook execution started (system subtype) |
| **HookProgressMessage** | ✅ | ✅ | Hook execution stdout/stderr output (system subtype) |
| **HookResponseMessage** | ✅ | ✅ | Hook execution completed with outcome (system subtype) |
| **TaskNotificationMessage** | ✅ | ✅ | Background task completed (system subtype) |
| **TaskStartedMessage** | ✅ | ✅ | Background task started (system subtype) |
| **TaskProgressMessage** | ✅ | ✅ | Background task progress (system subtype) |
| **FilesPersistedEvent** | ✅ | ✅ | Files persisted to checkpoint (system subtype) |
| **UnknownMessage** | ❌ | ✅ | Forward-compatible catch-all for unrecognized types |

### Tool Integration & Permissions

| Feature | Python | Go | Notes |
|---------|--------|-----|-------|
| **Tool permissions** | ✅ | ✅ | Permission callbacks for tool use |
| **Permission modes** | ✅ | ✅ | default, acceptEdits, auto, dontAsk, plan, bypassPermissions |
| **Tool filtering** | ✅ | ✅ | AllowedTools, DisallowedTools |
| **Tool use callbacks** | ✅ | ✅ | React to tool execution |
| **Permission storage** | ✅ | ✅ | Save permissions to user/project/local settings |
| **Updated input support** | ✅ | ✅ | Callbacks can modify tool inputs |

### MCP (Model Context Protocol) Support

| Feature | Python | Go | Notes |
|---------|--------|-----|-------|
| **SDK MCP servers** | ✅ | ✅ | Create tools in-process (NEW: factory in Go) |
| **External MCP servers** | ✅ | ✅ | stdio, SSE, HTTP connections |
| **Custom MCP servers** | ✅ | ✅ | Implement MCPServer interface |
| **Tool schema validation** | ✅ | ✅ | JSON schema input validation |
| **Tool listing** | ✅ | ✅ | Discover available tools dynamically |
| **MCP factory function** | ❌ | ✅ | Go SDK has convenient factory (Issue #24) |

### Hook System

| Feature | Python | Go | Notes |
|---------|--------|-----|-------|
| **Pre-tool hooks** | ✅ | ✅ | Before tool execution |
| **Post-tool hooks** | ✅ | ✅ | After tool execution |
| **User prompt hooks** | ✅ | ✅ | Before user input processing |
| **Hook callbacks** | ✅ | ✅ | Receive context about hook trigger |
| **Regex matching** | ✅ | ✅ | Filter hooks by tool name pattern |
| **Hook continuation** | ✅ | ✅ | Control whether to continue execution |

### System Configuration

| Feature | Python | Go | Notes |
|---------|--------|-----|-------|
| **System prompt** | ✅ | ✅ | Custom system instructions |
| **System prompt presets** | ⚠️ | ⚠️ | claude_code preset (emerging) |
| **Model parameter** | ✅ | ✅ | Specify model version |
| **Temperature** | ❌ | ❌ | Not exposed by current APIs |
| **Top P** | ❌ | ❌ | Not exposed by current APIs |

### Extended Thinking

| Feature | Python | Go | Notes |
|---------|--------|-----|-------|
| **Extended thinking** | ✅ | ✅ | Long-form reasoning (requires beta) |
| **Max thinking tokens** | ✅ | ✅ | Limit thinking output length |
| **Thinking blocks** | ✅ | ✅ | Access reasoning process |
| **Thinking control** | ✅ | ✅ | Enable/disable extended thinking |

### Cost Management

| Feature | Python | Go | Notes |
|---------|--------|-----|-------|
| **Token counting** | ✅ | ✅ | Access input/output token counts |
| **Cost summary** | ✅ | ✅ | Get usage statistics per message |
| **Budget limiting** | ✅ | ✅ | Max budget in USD |
| **Cost estimation** | ✅ | ✅ | Predict costs before execution |

### Plugin Support

| Feature | Python | Go | Notes |
|---------|--------|-----|-------|
| **Local plugins** | ✅ | ✅ | Load from directory |
| **Plugin discovery** | ✅ | ✅ | Auto-detect plugin.json |
| **Plugin commands** | ✅ | ✅ | Access plugin-defined commands |
| **Plugin metadata** | ✅ | ✅ | Version, name, description |

### Beta Features

| Feature | Python | Go | Notes |
|---------|--------|-----|-------|
| **Beta registration** | ✅ | ✅ | Opt-in to experimental features |
| **Beta list** | ✅ | ✅ | Extended thinking, new models, etc. |

### Subagent Support (Emerging)

| Feature | Python | Go | Notes |
|---------|--------|-----|-------|
| **Dynamic agent definitions** | ✅ | ✅ | `--agents` payloads use documented camelCase subagent fields |
| **Session agent selection** | ✅ | ✅ | `WithSessionAgent` emits `--agent <name>` |
| **Subagent execution** | ⚠️ | ⚠️ | SDK option retained, but Claude Code CLI 2.1.132 rejects `--subagent-execution` |
| **Concurrent execution** | ⚠️ | ⚠️ | Run multiple agents in parallel |
| **Error handling** | ⚠️ | ⚠️ | Strategy for subagent failures |

### Error Handling & Validation

| Feature | Python | Go | Notes |
|---------|--------|-----|-------|
| **Typed errors** | ✅ | ✅ | Specific error types for different failures |
| **Error unwrapping** | ✅ | ✅ | Access root cause of errors |
| **CLI not found** | ✅ | ✅ | Detect missing Claude Code CLI |
| **Connection errors** | ✅ | ✅ | Network/subprocess failures |
| **JSON validation** | ✅ | ✅ | Parse and validate JSON responses |
| **Validation errors** | ✅ | ✅ | Configuration validation |

### Transport & Infrastructure

| Feature | Python | Go | Notes |
|---------|--------|-----|-------|
| **Subprocess transport** | ✅ | ✅ | Connect via Claude Code CLI |
| **Custom transports** | ⚠️ | ⚠️ | Extensible interface (advanced) |
| **CLI discovery** | ✅ | ✅ | Auto-find Claude Code CLI |
| **CLI version check** | ✅ | ✅ | Validate compatible CLI version |
| **Environment variables** | ✅ | ✅ | Pass env to subprocess |

### Development & Debugging

| Feature | Python | Go | Notes |
|---------|--------|-----|-------|
| **Verbose logging** | ✅ | ✅ | Debug mode output |
| **Message inspection** | ✅ | ✅ | Inspect raw messages |
| **Error messages** | ✅ | ✅ | Clear, actionable error text |
| **Type hints** | ✅ | ❌ | Python has runtime hints, Go has static types |

---

## Known Gaps & Workarounds

### Minor Gaps

1. **System Prompt Presets** (⚠️ Emerging)
   - Status: Being added to Python SDK
   - Impact: Low - can use custom prompts
   - Timeline: Q1 2026

2. **Advanced Streaming Events** (⚠️ Not Yet)
   - Status: Planned for future releases
   - Impact: Low - can use standard streaming
   - Workaround: Use hook callbacks

3. **Subagent Framework** (⚠️ Development)
   - Status: In Python SDK development branch
   - Impact: Medium for complex agentic systems
   - Timeline: Q1-Q2 2026

### Area Not Implemented

Currently **not implemented** in either SDK:
- Custom tokenizers (rely on Claude API)
- Local model execution
- Custom transport protocols (beyond stdio)

---

## Feature Breakdown by Category

### 100% Feature Parity (19 features)
- Core Query/Client APIs
- Message types and content blocks
- Tool integration and permissions
- MCP server support
- Hook system
- Cost tracking
- Plugin loading
- Error handling
- Transport layer

### ~95% Feature Parity (4 features with minor gaps)
- System configuration (no temperature/top-p)
- Extended thinking (works, requires beta flag)
- Beta features (fully supported)
- Debugging features (Go has less runtime introspection)

### Emerging/Preview (3 features)
- System prompt presets (coming to both)
- Subagent framework (Python developing)
- Advanced streaming (future)

---

## When to Use Each SDK

### Choose Go SDK When:
- ✅ You need concurrent request handling
- ✅ You want strong type safety
- ✅ You prefer Go's concurrency model
- ✅ You need high performance
- ✅ You want a single binary deployment

### Choose Python SDK When:
- ✅ You prefer Python's syntax
- ✅ You need rapid prototyping
- ✅ You're working with Jupyter notebooks
- ✅ You have an existing Python codebase
- ✅ You prefer async/await patterns

### Migration Notes
Both SDKs support the same control protocol, so migrating code between them is straightforward. See [MIGRATION_FROM_PYTHON.md](./MIGRATION_FROM_PYTHON.md) for detailed patterns.

---

## Version Compatibility

| Go SDK Version | Python SDK Version | Compatible |
|---|---|---|
| v0.2.0+ | v0.1.0+ | ✅ Yes |
| v0.1.0+ | v0.1.0+ | ✅ Yes |

Both SDKs use the same control protocol and are forward-compatible with different versions.

---

## Recent Additions (v0.1.17+)

### Python SDK
- Extended thinking blocks in messages
- Beta registration system
- Plugin metadata access
- Subagent framework (branch)

### Go SDK
- Extended thinking support (v0.2.0+)
- Beta registration (v0.2.0+)
- MCP server factory (v0.2.9+) ← Issue #24
- Comprehensive documentation (v0.2.9+) ← Issue #25
- 14 new message types + `UnknownMessage` forward-compat catch-all (016-sdk-message-types)
- Enhanced `ResultMessage`: `Subtype`, `Errors`, `PermissionDenials`, `ModelUsageMap`, `StopReason`, `TotalCostUSD`, `UUID`
- Enhanced `UserMessage`: `IsReplay` field
- System subtype routing via `unmarshalSystemMessage()` for typed dispatch

---

## Roadmap & Future Parity

### Q1 2026
- [ ] System prompt presets (both)
- [ ] Subagent framework (both)
- [ ] Advanced streaming events

### Q2 2026
- [ ] Custom tokenizer support
- [ ] Performance optimizations
- [ ] Enhanced debugging features

---

## Help Us Close the Gaps

- Found a missing feature? [Open an issue](https://github.com/schlunsen/claude-agent-sdk-go/issues)
- Want to contribute? See [DEVELOPMENT.md](../DEVELOPMENT.md)
- Have feedback? [Discussions](https://github.com/schlunsen/claude-agent-sdk-go/discussions)

---

## FAQ

**Q: Which SDK should I use for production?**
A: Both are production-ready. Choose based on your language preference and deployment model.

**Q: Are the SDKs interoperable?**
A: Yes! Both use the same control protocol. You can mix and match in microservices.

**Q: Can I run both SDKs on the same machine?**
A: Yes! They share the same Claude Code CLI, so install once and use both.

**Q: How often is parity checked?**
A: Every SDK release. This document is updated for each version.

**Q: What about backward compatibility?**
A: Both SDKs maintain backward compatibility within major versions.

---

**Last Verified**: 2026-03-19
**Next Review**: 2026-06-30

For the latest SDK versions and features, see:
- [Go SDK Releases](https://github.com/schlunsen/claude-agent-sdk-go/releases)
- [Python SDK Releases](https://github.com/anthropics/claude-agent-sdk-python/releases)
