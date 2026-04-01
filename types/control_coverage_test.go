package types

import (
	"testing"
)

// TestGetHookEventName tests the GetHookEventName method on all hook-specific output types.
func TestGetHookEventName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		output   HookSpecificOutput
		wantName string
	}{
		{
			name:     "PreToolUseHookSpecificOutput",
			output:   &PreToolUseHookSpecificOutput{HookEventName: "PreToolUse"},
			wantName: "PreToolUse",
		},
		{
			name:     "PostToolUseHookSpecificOutput",
			output:   &PostToolUseHookSpecificOutput{HookEventName: "PostToolUse"},
			wantName: "PostToolUse",
		},
		{
			name:     "UserPromptSubmitHookSpecificOutput",
			output:   &UserPromptSubmitHookSpecificOutput{HookEventName: "UserPromptSubmit"},
			wantName: "UserPromptSubmit",
		},
		{
			name:     "PostToolUseFailureHookSpecificOutput",
			output:   &PostToolUseFailureHookSpecificOutput{HookEventName: "PostToolUseFailure"},
			wantName: "PostToolUseFailure",
		},
		{
			name:     "NotificationHookSpecificOutput",
			output:   &NotificationHookSpecificOutput{HookEventName: "Notification"},
			wantName: "Notification",
		},
		{
			name:     "SessionStartHookSpecificOutput",
			output:   &SessionStartHookSpecificOutput{HookEventName: "SessionStart"},
			wantName: "SessionStart",
		},
		{
			name:     "SubagentStartHookSpecificOutput",
			output:   &SubagentStartHookSpecificOutput{HookEventName: "SubagentStart"},
			wantName: "SubagentStart",
		},
		{
			name:     "PermissionRequestHookSpecificOutput",
			output:   &PermissionRequestHookSpecificOutput{HookEventName: "PermissionRequest"},
			wantName: "PermissionRequest",
		},
		{
			name:     "SetupHookSpecificOutput",
			output:   &SetupHookSpecificOutput{HookEventName: "Setup"},
			wantName: "Setup",
		},
		{
			name:     "ElicitationHookSpecificOutput",
			output:   &ElicitationHookSpecificOutput{HookEventName: "Elicitation"},
			wantName: "Elicitation",
		},
		{
			name:     "ElicitationResultHookSpecificOutput",
			output:   &ElicitationResultHookSpecificOutput{HookEventName: "ElicitationResult"},
			wantName: "ElicitationResult",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.output.GetHookEventName()
			if got != tt.wantName {
				t.Errorf("GetHookEventName() = %q, want %q", got, tt.wantName)
			}
		})
	}
}

// TestNewHookEventConstants_PhaseC tests that all Phase C hook event constants are non-empty.
func TestNewHookEventConstants_PhaseC(t *testing.T) {
	t.Parallel()

	events := []HookEvent{
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

	for _, event := range events {
		event := event
		t.Run(string(event), func(t *testing.T) {
			t.Parallel()
			if event == "" {
				t.Errorf("hook event constant should not be empty")
			}
		})
	}
}

// TestPermissionBehaviorConstants tests PermissionBehavior constants.
func TestPermissionBehaviorConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value PermissionBehavior
		want  string
	}{
		{"allow", PermissionBehaviorAllow, "allow"},
		{"deny", PermissionBehaviorDeny, "deny"},
		{"ask", PermissionBehaviorAsk, "ask"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if string(tt.value) != tt.want {
				t.Errorf("PermissionBehavior = %q, want %q", tt.value, tt.want)
			}
		})
	}
}

// TestPermissionUpdateDestinationConstants tests PermissionUpdateDestination constants.
func TestPermissionUpdateDestinationConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value PermissionUpdateDestination
		want  string
	}{
		{"userSettings", DestinationUserSettings, "userSettings"},
		{"projectSettings", DestinationProjectSettings, "projectSettings"},
		{"localSettings", DestinationLocalSettings, "localSettings"},
		{"session", DestinationSession, "session"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if string(tt.value) != tt.want {
				t.Errorf("PermissionUpdateDestination = %q, want %q", tt.value, tt.want)
			}
		})
	}
}
