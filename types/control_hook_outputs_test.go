package types

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// Hook output types: GetHookEventName() and JSON roundtrip for Phase C outputs
// ---------------------------------------------------------------------------

func TestPhaseCHookSpecificOutputs_GetHookEventName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output HookSpecificOutput
		want   string
	}{
		{
			name:   "PostToolUseFailureHookSpecificOutput",
			output: &PostToolUseFailureHookSpecificOutput{HookEventName: "PostToolUseFailure"},
			want:   "PostToolUseFailure",
		},
		{
			name:   "NotificationHookSpecificOutput",
			output: &NotificationHookSpecificOutput{HookEventName: "Notification"},
			want:   "Notification",
		},
		{
			name:   "SessionStartHookSpecificOutput",
			output: &SessionStartHookSpecificOutput{HookEventName: "SessionStart"},
			want:   "SessionStart",
		},
		{
			name:   "SubagentStartHookSpecificOutput",
			output: &SubagentStartHookSpecificOutput{HookEventName: "SubagentStart"},
			want:   "SubagentStart",
		},
		{
			name:   "PermissionRequestHookSpecificOutput",
			output: &PermissionRequestHookSpecificOutput{HookEventName: "PermissionRequest"},
			want:   "PermissionRequest",
		},
		{
			name:   "SetupHookSpecificOutput",
			output: &SetupHookSpecificOutput{HookEventName: "Setup"},
			want:   "Setup",
		},
		{
			name:   "ElicitationHookSpecificOutput",
			output: &ElicitationHookSpecificOutput{HookEventName: "Elicitation"},
			want:   "Elicitation",
		},
		{
			name:   "ElicitationResultHookSpecificOutput",
			output: &ElicitationResultHookSpecificOutput{HookEventName: "ElicitationResult"},
			want:   "ElicitationResult",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.output.GetHookEventName()
			if got != tt.want {
				t.Errorf("GetHookEventName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPostToolUseFailureHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output PostToolUseFailureHookSpecificOutput
	}{
		{
			name: "with additional context",
			output: PostToolUseFailureHookSpecificOutput{
				HookEventName:     "PostToolUseFailure",
				AdditionalContext: stringPtr("tool failed due to timeout"),
			},
		},
		{
			name: "without additional context",
			output: PostToolUseFailureHookSpecificOutput{
				HookEventName: "PostToolUseFailure",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded PostToolUseFailureHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			if (decoded.AdditionalContext == nil) != (tt.output.AdditionalContext == nil) {
				t.Errorf("AdditionalContext nil mismatch: got nil=%v, want nil=%v",
					decoded.AdditionalContext == nil, tt.output.AdditionalContext == nil)
			}
			if decoded.AdditionalContext != nil && *decoded.AdditionalContext != *tt.output.AdditionalContext {
				t.Errorf("AdditionalContext = %q, want %q", *decoded.AdditionalContext, *tt.output.AdditionalContext)
			}

			// Verify wire-format key is camelCase
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal to map: %v", err)
			}
			if _, ok := raw["hookEventName"]; !ok {
				t.Error("expected JSON key 'hookEventName' (camelCase)")
			}
		})
	}
}

func TestNotificationHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output NotificationHookSpecificOutput
	}{
		{
			name: "with additional context",
			output: NotificationHookSpecificOutput{
				HookEventName:     "Notification",
				AdditionalContext: stringPtr("notification acknowledged"),
			},
		},
		{
			name: "without additional context",
			output: NotificationHookSpecificOutput{
				HookEventName: "Notification",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded NotificationHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			if (decoded.AdditionalContext == nil) != (tt.output.AdditionalContext == nil) {
				t.Errorf("AdditionalContext nil mismatch")
			}
			if decoded.AdditionalContext != nil && *decoded.AdditionalContext != *tt.output.AdditionalContext {
				t.Errorf("AdditionalContext = %q, want %q", *decoded.AdditionalContext, *tt.output.AdditionalContext)
			}
		})
	}
}

func TestSessionStartHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output SessionStartHookSpecificOutput
	}{
		{
			name: "with context",
			output: SessionStartHookSpecificOutput{
				HookEventName:     "SessionStart",
				AdditionalContext: stringPtr("session initialized"),
			},
		},
		{
			name: "bare",
			output: SessionStartHookSpecificOutput{
				HookEventName: "SessionStart",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SessionStartHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			if (decoded.AdditionalContext == nil) != (tt.output.AdditionalContext == nil) {
				t.Errorf("AdditionalContext nil mismatch")
			}
		})
	}
}

func TestSubagentStartHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output SubagentStartHookSpecificOutput
	}{
		{
			name: "with context",
			output: SubagentStartHookSpecificOutput{
				HookEventName:     "SubagentStart",
				AdditionalContext: stringPtr("subagent spawning"),
			},
		},
		{
			name: "bare",
			output: SubagentStartHookSpecificOutput{
				HookEventName: "SubagentStart",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SubagentStartHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			if (decoded.AdditionalContext == nil) != (tt.output.AdditionalContext == nil) {
				t.Errorf("AdditionalContext nil mismatch")
			}
		})
	}
}

func TestPermissionRequestHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output PermissionRequestHookSpecificOutput
	}{
		{
			name: "allow with reason",
			output: PermissionRequestHookSpecificOutput{
				HookEventName:            "PermissionRequest",
				PermissionDecision:       stringPtr("allow"),
				PermissionDecisionReason: stringPtr("trusted tool"),
			},
		},
		{
			name: "deny with reason",
			output: PermissionRequestHookSpecificOutput{
				HookEventName:            "PermissionRequest",
				PermissionDecision:       stringPtr("deny"),
				PermissionDecisionReason: stringPtr("dangerous operation"),
			},
		},
		{
			name: "no decision",
			output: PermissionRequestHookSpecificOutput{
				HookEventName: "PermissionRequest",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded PermissionRequestHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			if (decoded.PermissionDecision == nil) != (tt.output.PermissionDecision == nil) {
				t.Errorf("PermissionDecision nil mismatch")
			}
			if decoded.PermissionDecision != nil && *decoded.PermissionDecision != *tt.output.PermissionDecision {
				t.Errorf("PermissionDecision = %q, want %q", *decoded.PermissionDecision, *tt.output.PermissionDecision)
			}
			if (decoded.PermissionDecisionReason == nil) != (tt.output.PermissionDecisionReason == nil) {
				t.Errorf("PermissionDecisionReason nil mismatch")
			}
			if decoded.PermissionDecisionReason != nil && *decoded.PermissionDecisionReason != *tt.output.PermissionDecisionReason {
				t.Errorf("PermissionDecisionReason = %q, want %q", *decoded.PermissionDecisionReason, *tt.output.PermissionDecisionReason)
			}

			// Verify wire-format keys
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal to map: %v", err)
			}
			if _, ok := raw["hookEventName"]; !ok {
				t.Error("expected JSON key 'hookEventName'")
			}
		})
	}
}

func TestSetupHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output SetupHookSpecificOutput
	}{
		{
			name: "with context",
			output: SetupHookSpecificOutput{
				HookEventName:     "Setup",
				AdditionalContext: stringPtr("environment ready"),
			},
		},
		{
			name: "bare",
			output: SetupHookSpecificOutput{
				HookEventName: "Setup",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SetupHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			if (decoded.AdditionalContext == nil) != (tt.output.AdditionalContext == nil) {
				t.Errorf("AdditionalContext nil mismatch")
			}
		})
	}
}

func TestElicitationHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output ElicitationHookSpecificOutput
	}{
		{
			name: "with answers",
			output: ElicitationHookSpecificOutput{
				HookEventName: "Elicitation",
				Answers:       map[string]interface{}{"q1": "yes", "q2": float64(42)},
			},
		},
		{
			name: "empty answers",
			output: ElicitationHookSpecificOutput{
				HookEventName: "Elicitation",
				Answers:       map[string]interface{}{},
			},
		},
		{
			name: "nil answers",
			output: ElicitationHookSpecificOutput{
				HookEventName: "Elicitation",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded ElicitationHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			// For non-nil answers, verify length matches
			if tt.output.Answers != nil {
				if len(decoded.Answers) != len(tt.output.Answers) {
					t.Errorf("Answers length = %d, want %d", len(decoded.Answers), len(tt.output.Answers))
				}
			}

			// Verify wire-format keys
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("failed to unmarshal to map: %v", err)
			}
			if _, ok := raw["hookEventName"]; !ok {
				t.Error("expected JSON key 'hookEventName'")
			}
		})
	}
}

func TestElicitationResultHookSpecificOutput_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output ElicitationResultHookSpecificOutput
	}{
		{
			name: "with context",
			output: ElicitationResultHookSpecificOutput{
				HookEventName:     "ElicitationResult",
				AdditionalContext: stringPtr("results processed"),
			},
		},
		{
			name: "bare",
			output: ElicitationResultHookSpecificOutput{
				HookEventName: "ElicitationResult",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.output)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded ElicitationResultHookSpecificOutput
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.HookEventName != tt.output.HookEventName {
				t.Errorf("HookEventName = %q, want %q", decoded.HookEventName, tt.output.HookEventName)
			}
			if (decoded.AdditionalContext == nil) != (tt.output.AdditionalContext == nil) {
				t.Errorf("AdditionalContext nil mismatch")
			}
			if decoded.AdditionalContext != nil && *decoded.AdditionalContext != *tt.output.AdditionalContext {
				t.Errorf("AdditionalContext = %q, want %q", *decoded.AdditionalContext, *tt.output.AdditionalContext)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Interface satisfaction: verify all 8 Phase C output types implement HookSpecificOutput
// ---------------------------------------------------------------------------

func TestPhaseCOutputs_ImplementHookSpecificOutput(t *testing.T) {
	t.Parallel()

	// Compile-time check via interface assignment.
	// If any type fails to implement HookSpecificOutput, this won't compile.
	tests := []struct {
		name   string
		output HookSpecificOutput
	}{
		{name: "PostToolUseFailure", output: &PostToolUseFailureHookSpecificOutput{}},
		{name: "Notification", output: &NotificationHookSpecificOutput{}},
		{name: "SessionStart", output: &SessionStartHookSpecificOutput{}},
		{name: "SubagentStart", output: &SubagentStartHookSpecificOutput{}},
		{name: "PermissionRequest", output: &PermissionRequestHookSpecificOutput{}},
		{name: "Setup", output: &SetupHookSpecificOutput{}},
		{name: "Elicitation", output: &ElicitationHookSpecificOutput{}},
		{name: "ElicitationResult", output: &ElicitationResultHookSpecificOutput{}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.output == nil {
				t.Fatal("output should not be nil")
			}
			// GetHookEventName on zero-value should return empty string (not panic)
			_ = tt.output.GetHookEventName()
		})
	}
}
