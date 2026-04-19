# Real-CLI Integration Coverage Matrix

Tracks which SDK public surfaces have real-CLI integration tests and which
are deferred. Every `✅` row is exercised by at least one test in
`tests/integration_*_test.go` under the `//go:build integration` tag.

## Legend

- `✅` — Covered by a real-CLI test that runs without `CLAUDE_SDK_RUN_TURNS=1` (cheap; control-protocol round-trip only).
- `🔒` — Covered, but quota-gated via `CLAUDE_SDK_RUN_TURNS=1` (spends model tokens).
- `⚠️` — Partial coverage; gap documented in "Notes" column.
- `⏭` — Deferred to a future pass; rationale in "Notes".
- `N/A` — Not applicable to this surface.

## Run commands

```bash
# No-quota subset (default; no model calls):
go test -tags=integration -race -count=1 -p 1 ./tests/... -timeout=300s

# Full suite including turn-driving tests (burns tokens):
CLAUDE_SDK_RUN_TURNS=1 go test -tags=integration -race -count=1 -p 1 ./tests/... -timeout=900s

# 5-count stability per .claude/rules/stress-test-flake-fixes.md:
go test -tags=integration -race -count=5 -p 1 ./tests/... -timeout=1800s
```

## Client methods (client.go)

| Method | File:Line | Status | Test | Notes |
|---|---|---|---|---|
| `NewClient` + `Connect` | :100, :187 | ✅ | `setupClient` helper exercises both on every test | |
| `Close` | :506 | ✅ | `TestClient_Close_DisconnectsCleanly`, `TestInteractions_DoubleClose_IsIdempotent` | |
| `Query` | :308 | 🔒 | `TestClient_Query_HappyPath` | |
| `QueryWithContent` | :379 | 🔒 | `TestClient_QueryWithContent_StructuredBlocks` | |
| `ReceiveResponse` | :446 | 🔒 | Every turn-driving test consumes the channel | |
| `IsConnected` | :590 | ✅ | `TestClient_IsConnected` | |
| `InitResult` | :601 | ✅ | `TestClient_InitResult` | |
| `SlashCommands` | :610 | ✅ | `TestClient_SlashCommands` | Asserts `/help` present |
| `SupportedModels` | :621 | ✅ | `TestClient_SupportedModels` | |
| `SetModel` | :633 | ✅ | `TestClient_SetModel` | Cycles through advertised models |
| `SetPermissionMode` | :656 | ✅ | `TestClient_SetPermissionMode` | 3 modes tested |
| `ProcessID` | :678 | ✅ | `TestClient_ProcessID`, `TestInteractions_ReconnectAfterClose` | |
| `Interrupt` | :693 | 🔒 | `TestClient_Interrupt_DuringStream` | |
| `StreamInput` | :713 | 🔒 | `TestClient_StreamInput_AppendMidTurn` | Skips if CLI rejects |
| `StopTask` | :748 | 🔒 | `TestClient_StopTask` | |
| `MCPServerStatus` | :773 | ✅ | `TestMCPServerStatus_EmptyConfig`, `_WireShape` | Feature-170 class |
| `ReconnectMCPServer` | :808 | ✅ | `TestReconnectMCPServer_UnknownServer` | Negative-path |
| `ToggleMCPServer` | :833 | ✅ | `TestToggleMCPServer_Disabled` | |
| `SetMCPServers` | :859 | ✅ | `TestSetMCPServers_EmptyMap`, `_Minimal` | |
| `RewindFiles` | :896 | ✅ | `TestClient_RewindFiles_DryRun` | Dry-run only |
| `GetContextUsage` | :935 | ✅ | `TestClient_GetContextUsage`, `TestInteractions_ConcurrentReads_AreSafe` | |
| `GetSettings` | :966 | ✅ | `TestClient_GetSettings`, `TestInteractions_ConcurrentReads_AreSafe` | |
| `ReloadPlugins` | :996 | ✅ | `TestClient_ReloadPlugins` | |
| `EnableChannel` | :1016 | ✅ | `TestClient_EnableChannel` | |
| `SupportedAgents` | :1036 | ✅ | `TestClient_SupportedAgents` | |

## Sessions API (sessions.go)

| Function | File:Line | Status | Test | Notes |
|---|---|---|---|---|
| `ListSessions` | :16 | ✅ | `TestSessions_List`, `_ExistingSession` | Wire-shape probe included |
| `GetSessionMessages` | :57 | ✅ | `TestSessions_GetMessages_InvalidID` | Negative-path only |
| `GetSessionInfo` | :103 | ✅ | `TestSessions_GetInfo_InvalidID`, `_List_ExistingSession` | |
| `RenameSession` | :138 | ✅ | `TestSessions_Rename_InvalidID` | Negative-path |
| `TagSession` | :168 | ✅ | `TestSessions_Tag_InvalidID` | Negative-path |
| `ForkSession` | :200 | ✅ | `TestSessions_Fork_InvalidID` | Negative-path |
| `ListSubagents` | :235 | ✅ | `TestSessions_ListSubagents_InvalidID` | Accepts empty or error |
| `GetSubagentMessages` | :278 | ✅ | `TestSessions_GetSubagentMessages_InvalidID` | Negative-path |

Happy-path tests for Rename/Tag/Fork against a real session ID are ⏭ — they
require a multi-turn setup flow that's best covered by a dedicated "session
lifecycle" suite in a future pass.

## Hook events (types/control.go:74-99)

All 23 events are registered in `TestHooks_RegisterAllEvents` — the
control-protocol init response acknowledges each. A failing registration
surfaces wire-drift on a hook event name. Per-event flow tests:

| Event | Status | Test | Notes |
|---|---|---|---|
| `PreToolUse` | 🔒 | `TestHooks_PreToolUse_Fires` | Drives a Bash tool-use turn |
| `PostToolUse` | ⏭ | — | Covered indirectly by PreToolUse turn |
| `UserPromptSubmit` | 🔒 | `TestHooks_UserPromptSubmit_Fires` | |
| `Stop` / `SubagentStop` / `PreCompact` | ⏭ | — | |
| `PostToolUseFailure` | ⏭ | — | Requires failing tool call |
| `Notification` / `SessionStart` / `SessionEnd` | ⏭ | — | |
| `StopFailure` / `SubagentStart` / `PostCompact` | ⏭ | — | |
| `PermissionRequest` / `Setup` / `TeammateIdle` | ⏭ | — | |
| `TaskCompleted` / `Elicitation` / `ElicitationResult` | ⏭ | — | |
| `ConfigChange` / `WorktreeCreate` / `WorktreeRemove` | ⏭ | — | |
| `InstructionsLoaded` | ⏭ | — | |

Per-event flow coverage for the 17 new events deferred until a dedicated
hooks lifecycle pass; `TestHooks_RegisterAllEvents` is the structural
floor — any event-name drift surfaces at Connect() time.

## CLI flags (internal/transport/subprocess_cli.go:buildCommandArgs)

| Category | Status | Test | Notes |
|---|---|---|---|
| All 28 visible flags cross-checked against `claude --help` | ✅ | `TestFlags_AllEmittedFlagsAcceptedByCLI` | Drift detector |
| Hidden flag documentation (`--resume-session-at`) | ✅ | `TestFlags_HiddenFlagsDocumented` | Feature 142 lineage |

SDK-side argv unit tests already live in
`internal/transport/subprocess_cli_test.go:60+` (20+ tests) — they cover the
SDK emission side. This file verifies the CLI consumption side.

## Cross-cutting / interactions

| Concern | Status | Test | Notes |
|---|---|---|---|
| Concurrent read safety | ✅ | `TestInteractions_ConcurrentReads_AreSafe` | `-race` required |
| Double-close idempotence | ✅ | `TestInteractions_DoubleClose_IsIdempotent` | |
| Context cancel → channel close | 🔒 | `TestInteractions_ContextCancel_ClosesChannel` | |
| Reconnect fresh subprocess | ✅ | `TestInteractions_ReconnectAfterClose` | Asserts PID differs |

## Wire-shape fixtures

`tests/fixtures/` holds captured CLI responses as JSON baselines. A Pass 3
probe task will run once with `CLAUDE_SDK_PROBE=1` to regenerate fixtures
after a CLI upgrade; divergence from the baseline is a review flag.

## Gaps + follow-ups

Consolidated in `.gates/deferred-issues.md` once the feature lands. Current
known gaps:

1. Happy-path Rename/Tag/Fork against a real session (requires lifecycle setup).
2. Per-event flow coverage for 20 of 23 hook events (structural floor only today).
3. Schema-drift guard analogous to codex's `make regen-schema` — pending CLI support for a JSON-schema dump command.
4. Nightly CI workflow not yet present (`.github/workflows/integration.yml` is a Pass 3 deliverable).
