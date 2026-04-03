package types

import (
	"encoding/json"
	"testing"
)

func TestContextUsage_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		usage ContextUsage
	}{
		{
			name: "all fields populated",
			usage: ContextUsage{
				TotalTokens:    50000,
				MaxTokens:      200000,
				UtilizationPct: 25.0,
				ByCategory: map[string]CategoryUsage{
					"system":    {Tokens: 10000, Pct: 5.0},
					"user":      {Tokens: 15000, Pct: 7.5},
					"assistant": {Tokens: 25000, Pct: 12.5},
				},
			},
		},
		{
			name: "no categories",
			usage: ContextUsage{
				TotalTokens:    1000,
				MaxTokens:      200000,
				UtilizationPct: 0.5,
			},
		},
		{
			name:  "zero values",
			usage: ContextUsage{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.usage)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded ContextUsage
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.TotalTokens != tt.usage.TotalTokens {
				t.Errorf("TotalTokens = %d, want %d", decoded.TotalTokens, tt.usage.TotalTokens)
			}
			if decoded.MaxTokens != tt.usage.MaxTokens {
				t.Errorf("MaxTokens = %d, want %d", decoded.MaxTokens, tt.usage.MaxTokens)
			}
			if decoded.UtilizationPct != tt.usage.UtilizationPct {
				t.Errorf("UtilizationPct = %f, want %f", decoded.UtilizationPct, tt.usage.UtilizationPct)
			}
			if len(decoded.ByCategory) != len(tt.usage.ByCategory) {
				t.Errorf("ByCategory len = %d, want %d", len(decoded.ByCategory), len(tt.usage.ByCategory))
			}
			for k, v := range tt.usage.ByCategory {
				got, ok := decoded.ByCategory[k]
				if !ok {
					t.Errorf("ByCategory missing key %q", k)
					continue
				}
				if got.Tokens != v.Tokens {
					t.Errorf("ByCategory[%q].Tokens = %d, want %d", k, got.Tokens, v.Tokens)
				}
				if got.Pct != v.Pct {
					t.Errorf("ByCategory[%q].Pct = %f, want %f", k, got.Pct, v.Pct)
				}
			}
		})
	}
}

func TestContextUsage_WireFormatKeys(t *testing.T) {
	t.Parallel()
	usage := ContextUsage{
		TotalTokens:    100,
		MaxTokens:      1000,
		UtilizationPct: 10.0,
		ByCategory:     map[string]CategoryUsage{"test": {Tokens: 100, Pct: 10.0}},
	}

	data, err := json.Marshal(usage)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	for _, key := range []string{"totalTokens", "maxTokens", "utilizationPct", "byCategory"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected camelCase JSON key %q", key)
		}
	}
}

func FuzzContextUsage_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"totalTokens":100,"maxTokens":1000,"utilizationPct":10.0}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"totalTokens":0,"maxTokens":0,"utilizationPct":0,"byCategory":{}}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var usage ContextUsage
		_ = json.Unmarshal(data, &usage)
	})
}
