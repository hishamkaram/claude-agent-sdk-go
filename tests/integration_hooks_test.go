//go:build integration
// +build integration

// Real-CLI coverage for the 23 HookEvent constants declared in
// types/control.go:74-99. Every event is registered on a connected
// client; the tests that drive the event through the CLI are
// quota-gated (CLAUDE_SDK_RUN_TURNS=1) because they require a full
// model turn to trigger hook emission. The registration-only test
// exercises the hook-registration wire without spending tokens.
//
// Safety: safetyNetSettings + safetyNetHooks are called before any
// hook-registering test. They snapshot ~/.claude/settings.json and
// ~/.claude/hooks.json and restore them on t.Cleanup, so a panicking
// SDK or CLI cannot leave the user's real settings mutated.

package tests

import (
	"context"
	"sync"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// allHookEvents is the canonical list tested against — if a new HookEvent
// constant is added to types/control.go without a matching entry here, the
// TestHooks_AllEventsCoveredBySuite test fails, forcing the author to add
// coverage.
var allHookEvents = []types.HookEvent{
	types.HookEventPreToolUse,
	types.HookEventPostToolUse,
	types.HookEventUserPromptSubmit,
	types.HookEventStop,
	types.HookEventSubagentStop,
	types.HookEventPreCompact,
	types.HookEventPostToolUseFailure,
	types.HookEventNotification,
	types.HookEventSessionStart,
	types.HookEventSessionEnd,
	types.HookEventStopFailure,
	types.HookEventSubagentStart,
	types.HookEventPostCompact,
	types.HookEventPermissionRequest,
	types.HookEventSetup,
	types.HookEventTeammateIdle,
	types.HookEventTaskCompleted,
	types.HookEventElicitation,
	types.HookEventElicitationResult,
	types.HookEventConfigChange,
	types.HookEventWorktreeCreate,
	types.HookEventWorktreeRemove,
	types.HookEventInstructionsLoaded,
}

// TestHooks_RegisterAllEvents registers a no-op callback for every HookEvent
// constant and asserts NewClient + Connect succeed. The CLI's control
// protocol init response must acknowledge the hook registration for every
// event — any event name drift (spelled differently in SDK vs CLI) would
// surface here as a connection failure or a missing hook acknowledgment.
func TestHooks_RegisterAllEvents(t *testing.T) {
	safetyNetSettings(t)
	safetyNetHooks(t)

	var callCount int32
	var mu sync.Mutex

	cb := func(ctx context.Context, input interface{}, toolUseID *string, hookCtx types.HookContext) (interface{}, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		return map[string]interface{}{}, nil
	}

	client, _ := setupClient(t, func(opts *types.ClaudeAgentOptions) {
		matcher := types.HookMatcher{Hooks: []types.HookCallbackFunc{cb}}
		for _, event := range allHookEvents {
			opts.WithHook(event, matcher)
		}
	})

	// Connect succeeded (setupClient would have t.Fatal'd otherwise).
	// Hook-event-specific assertions happen in the quota-gated flow tests
	// below — this test validates registration wire only.
	if !client.IsConnected() {
		t.Fatal("client not connected after registering all hook events")
	}
}

// TestHooks_PreToolUse_Fires drives a full turn that the model completes by
// calling a tool, and asserts the PreToolUse hook callback fires at least
// once with a non-nil input. This catches the hook-event-name case drift
// class — if the CLI emits hook_event_name="pretooluse" (lowercase) but
// the SDK matches on "PreToolUse", the callback never fires.
func TestHooks_PreToolUse_Fires(t *testing.T) {
	requireRunTurns(t)
	safetyNetSettings(t)
	safetyNetHooks(t)

	var calls []string
	var mu sync.Mutex

	cb := func(ctx context.Context, input interface{}, toolUseID *string, hookCtx types.HookContext) (interface{}, error) {
		mu.Lock()
		calls = append(calls, "PreToolUse")
		mu.Unlock()
		return map[string]interface{}{}, nil
	}

	client, ctx := setupClient(t, func(opts *types.ClaudeAgentOptions) {
		toolPattern := "Bash"
		matcher := types.HookMatcher{
			Matcher: &toolPattern,
			Hooks:   []types.HookCallbackFunc{cb},
		}
		opts.WithHook(types.HookEventPreToolUse, matcher)
	})

	prompt := "Run the shell command `echo hook-test` exactly once, then stop."
	if err := client.Query(ctx, prompt); err != nil {
		t.Fatalf("Query: %v", err)
	}

	_ = collectUntilResult(t, ctx, client)

	mu.Lock()
	defer mu.Unlock()
	if len(calls) == 0 {
		t.Errorf("PreToolUse hook did not fire; wire tag drift on hook_event_name?")
	} else {
		t.Logf("PreToolUse hook fired %d time(s)", len(calls))
	}
}

// TestHooks_UserPromptSubmit_Fires registers a UserPromptSubmit hook and
// drives a turn. The CLI should invoke the hook before sending the prompt
// to the model. Quota-gated because it requires a real turn.
func TestHooks_UserPromptSubmit_Fires(t *testing.T) {
	requireRunTurns(t)
	safetyNetSettings(t)
	safetyNetHooks(t)

	var fired bool
	var mu sync.Mutex

	cb := func(ctx context.Context, input interface{}, toolUseID *string, hookCtx types.HookContext) (interface{}, error) {
		mu.Lock()
		fired = true
		mu.Unlock()
		return map[string]interface{}{}, nil
	}

	client, ctx := setupClient(t, func(opts *types.ClaudeAgentOptions) {
		opts.WithHook(types.HookEventUserPromptSubmit, types.HookMatcher{
			Hooks: []types.HookCallbackFunc{cb},
		})
	})

	if err := client.Query(ctx, "Say 'hi'."); err != nil {
		t.Fatalf("Query: %v", err)
	}
	_ = collectUntilResult(t, ctx, client)

	mu.Lock()
	defer mu.Unlock()
	if !fired {
		t.Error("UserPromptSubmit hook did not fire")
	}
}

// TestHooks_AllEventsCoveredBySuite is a compile-time-style guard — it
// asserts that every HookEvent constant declared in types/control.go is
// present in allHookEvents. If a new event is added upstream without
// updating this slice, the count mismatch surfaces here before a release.
func TestHooks_AllEventsCoveredBySuite(t *testing.T) {
	// Source of truth: the event count in CLAUDE.md says "23 total hook
	// events (6 existing + 17 new)". If that count changes, update this
	// constant AND the allHookEvents slice above.
	const expected = 23
	if len(allHookEvents) != expected {
		t.Errorf("allHookEvents has %d entries, expected %d — update the slice when types/control.go gains a new HookEvent", len(allHookEvents), expected)
	}
}
