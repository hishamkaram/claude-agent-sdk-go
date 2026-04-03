package types

import (
	"encoding/json"
	"testing"
)

func TestSettingsResult_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		result SettingsResult
	}{
		{
			name: "all fields populated",
			result: SettingsResult{
				Applied: AppliedSettings{
					Model:  "claude-sonnet-4-6",
					Effort: "high",
				},
				Raw: map[string]interface{}{
					"model":  "claude-sonnet-4-6",
					"effort": "high",
					"custom": "value",
				},
			},
		},
		{
			name: "applied only",
			result: SettingsResult{
				Applied: AppliedSettings{
					Model:  "claude-opus-4-6",
					Effort: "max",
				},
			},
		},
		{
			name:   "empty",
			result: SettingsResult{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded SettingsResult
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.Applied.Model != tt.result.Applied.Model {
				t.Errorf("Applied.Model = %q, want %q", decoded.Applied.Model, tt.result.Applied.Model)
			}
			if decoded.Applied.Effort != tt.result.Applied.Effort {
				t.Errorf("Applied.Effort = %q, want %q", decoded.Applied.Effort, tt.result.Applied.Effort)
			}
			if len(decoded.Raw) != len(tt.result.Raw) {
				t.Errorf("Raw len = %d, want %d", len(decoded.Raw), len(tt.result.Raw))
			}
		})
	}
}

func TestSettingsResult_WireFormatKeys(t *testing.T) {
	t.Parallel()
	result := SettingsResult{
		Applied: AppliedSettings{Model: "test", Effort: "low"},
		Raw:     map[string]interface{}{"key": "val"},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	for _, key := range []string{"applied", "raw"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected JSON key %q", key)
		}
	}

	// Check nested applied keys
	applied, ok := raw["applied"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'applied' to be a map")
	}
	for _, key := range []string{"model", "effort"} {
		if _, ok := applied[key]; !ok {
			t.Errorf("expected applied JSON key %q", key)
		}
	}
}

func TestSettingsResult_OmitEmptyRaw(t *testing.T) {
	t.Parallel()
	result := SettingsResult{
		Applied: AppliedSettings{Model: "test", Effort: "low"},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, ok := raw["raw"]; ok {
		t.Error("expected 'raw' to be omitted when nil")
	}
}

func FuzzSettingsResult_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"applied":{"model":"m","effort":"e"},"raw":{}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"applied":{"model":"","effort":""}}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var result SettingsResult
		_ = json.Unmarshal(data, &result)
	})
}
