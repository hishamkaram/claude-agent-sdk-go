# Wire-Shape Fixtures

Captured JSON responses from the real `claude` CLI, used as regression
baselines for the SDK's message parser. Analogous to codex-agent-sdk-go's
`tests/fixtures/v040_probes/` directory.

## Regenerating fixtures

A future Pass 3 task adds a `CLAUDE_SDK_PROBE=1` mode that writes each
method's raw CLI response to this directory. Until that task lands, this
directory is intentionally empty except for this README.

When implemented, the regen flow will be:

```bash
CLAUDE_SDK_PROBE=1 CLAUDE_SDK_RUN_TURNS=1 \
  go test -tags=integration -run TestProbe ./tests/... -count=1
```

Probe tests MUST redact any PII (session IDs, paths under `$HOME`, API
keys) before writing the fixture. Fixtures are committed as regression
baselines — any diff between a fresh probe and the committed fixture is
a review flag for potential CLI/SDK wire drift.
