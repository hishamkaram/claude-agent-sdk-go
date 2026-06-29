package types

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Tests for Phase C: 17 new hook event constants and their input/output types
// ---------------------------------------------------------------------------

func TestHookEventConstants_PhaseC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		constant HookEvent
		want     string
	}{
		{name: "PostToolUseFailure", constant: HookEventPostToolUseFailure, want: "PostToolUseFailure"},
		{name: "Notification", constant: HookEventNotification, want: "Notification"},
		{name: "SessionStart", constant: HookEventSessionStart, want: "SessionStart"},
		{name: "SessionEnd", constant: HookEventSessionEnd, want: "SessionEnd"},
		{name: "StopFailure", constant: HookEventStopFailure, want: "StopFailure"},
		{name: "SubagentStart", constant: HookEventSubagentStart, want: "SubagentStart"},
		{name: "PostCompact", constant: HookEventPostCompact, want: "PostCompact"},
		{name: "PermissionRequest", constant: HookEventPermissionRequest, want: "PermissionRequest"},
		{name: "Setup", constant: HookEventSetup, want: "Setup"},
		{name: "TeammateIdle", constant: HookEventTeammateIdle, want: "TeammateIdle"},
		{name: "TaskCompleted", constant: HookEventTaskCompleted, want: "TaskCompleted"},
		{name: "Elicitation", constant: HookEventElicitation, want: "Elicitation"},
		{name: "ElicitationResult", constant: HookEventElicitationResult, want: "ElicitationResult"},
		{name: "ConfigChange", constant: HookEventConfigChange, want: "ConfigChange"},
		{name: "WorktreeCreate", constant: HookEventWorktreeCreate, want: "WorktreeCreate"},
		{name: "WorktreeRemove", constant: HookEventWorktreeRemove, want: "WorktreeRemove"},
		{name: "InstructionsLoaded", constant: HookEventInstructionsLoaded, want: "InstructionsLoaded"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := string(tt.constant)
			if got != tt.want {
				t.Errorf("HookEvent constant = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHookEventConstants_PhaseC_NoDuplicates(t *testing.T) {
	t.Parallel()

	allEvents := []HookEvent{
		// Phase A (original 6)
		HookEventPreToolUse,
		HookEventPostToolUse,
		HookEventUserPromptSubmit,
		HookEventStop,
		HookEventSubagentStop,
		HookEventPreCompact,
		// Phase C (17 new)
		HookEventPostToolUseFailure,
		HookEventNotification,
		HookEventSessionStart,
		HookEventSessionEnd,
		HookEventStopFailure,
		HookEventSubagentStart,
		HookEventPostCompact,
		HookEventPermissionRequest,
		HookEventSetup,
		HookEventTeammateIdle,
		HookEventTaskCompleted,
		HookEventElicitation,
		HookEventElicitationResult,
		HookEventConfigChange,
		HookEventWorktreeCreate,
		HookEventWorktreeRemove,
		HookEventInstructionsLoaded,
	}

	seen := make(map[HookEvent]bool, len(allEvents))
	for _, ev := range allEvents {
		if ev == "" {
			t.Errorf("hook event constant should not be empty")
		}
		if seen[ev] {
			t.Errorf("duplicate hook event constant %q", ev)
		}
		seen[ev] = true
	}
}
