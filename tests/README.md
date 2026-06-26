# Test Suite Documentation

This directory contains the Claude Agent SDK's real-CLI integration suite, mock-CLI plumbing tests, benchmarks, coverage helpers, and fixture guidance.

## Test Layout

| File or pattern | Purpose |
|-----------------|---------|
| `integration_test.go`, build-tagged `integration_*_test.go` files except `integration_mock_test.go` | Real Claude CLI tests behind the `integration` Go build tag. These use `requireClaude`, `requireAuth`, and `requireRunTurns` guards from `integration_helpers_test.go`. |
| `coldstart_realcli_integration_test.go`, `observer_realcli_integration_test.go` | Real-CLI latency and observer behavior suites behind the same `integration` tag. |
| `integration_mock_test.go` | Shell-script fake CLI tests. These skip under `-short`, so `make test` excludes them and `make test-all` includes them. |
| `benchmarks_test.go` | Benchmarks for query/client flows, message parsing, options, channels, JSON, and error paths. |
| `coverage_test.go` | Text and HTML coverage report helpers. |
| `test_helpers.go` | Mock CLI creation, message collection, assertion helpers, timeouts, CLI discovery, auth guards, and goroutine leak checks. |
| `coverage_matrix.md` | Real-CLI public-surface coverage matrix and follow-up notes. |
| `fixtures/` | Reserved for redacted real-CLI wire-shape baselines. |

## Running Tests

Run commands from the repository root.

```bash
# Fast unit loop; no Claude CLI required.
make test

# Unit tests plus mock-CLI tests and real-CLI integration tests.
make test-all

# Real-CLI no-quota subset. Turn-driving tests skip unless CLAUDE_SDK_RUN_TURNS=1.
make test-integration

# Full real-CLI suite, including tests that drive model turns and spend tokens.
CLAUDE_SDK_RUN_TURNS=1 make test-integration-quota

# Raw integration command used by the Makefile target.
go test -tags=integration -race -count=1 -p 1 ./tests/... -timeout=300s

# Five-count stability check before shipping a new real-CLI test.
go test -tags=integration -race -count=5 -p 1 ./tests/... -timeout=1800s

# Benchmarks.
make bench
go test -bench=. -benchmem ./tests/...

# Coverage report.
make coverage
go test -run TestCoverageReport ./tests/...
```

## Environment Gates

Real-CLI tests skip rather than fail when required local state is missing:

| Variable or state | Purpose |
|-------------------|---------|
| `CLAUDE_CLI_PATH` | Optional absolute path override for the `claude` binary. |
| `ANTHROPIC_API_KEY`, `CLAUDE_API_KEY`, or `~/.claude/.credentials.json` | Auth source checked by `requireAuth`. |
| `CLAUDE_SDK_RUN_TURNS=1` | Opts into tests that drive a full model turn and spend tokens. |

`CLAUDE_AGENT_SDK_SKIP_VERSION_CHECK=1` is available to skip CLI version validation in SDK setup, but it does not bypass the test auth or run-turn guards above.

## Design Principles

- `make test` stays cheap and deterministic: it runs `go test -race -short -count=1 -p 4 ./...`.
- Mock-CLI tests verify SDK plumbing only; real-CLI tests exist to catch wire-shape drift between the SDK and the actual `claude` subprocess.
- Real-CLI tests use explicit build tags and guard helpers so missing CLI/auth state is a skip, not a surprise local failure.
- Tests use context timeouts and goroutine leak checks to keep subprocess lifecycle issues visible.
- Fixture files must be redacted before commit; do not store session IDs, home-directory paths, API keys, OAuth tokens, or raw user prompts.

## Troubleshooting

| Symptom | Check |
|---------|-------|
| Real-CLI tests are skipped | Run `claude --version`, set `CLAUDE_CLI_PATH` if needed, and confirm one supported auth source is present. |
| Turn-driving tests are skipped | Set `CLAUDE_SDK_RUN_TURNS=1` and use `make test-integration-quota`. |
| Tests hang | Re-run with a shorter `-timeout`, inspect subprocess cleanup, and check goroutine leak output. |
| Benchmark results vary | Use `-benchtime=1s` and `-count=5`; avoid running benchmark comparisons on a busy machine. |

## References

- Main SDK tests live at the repository root and under `internal/`.
- Public surface coverage lives in `tests/coverage_matrix.md`.
- Fixture policy lives in `tests/fixtures/README.md`.
