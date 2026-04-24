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

func TestContextUsage_UnmarshalSupportsCamelCaseAndSnakeCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want ContextUsage
	}{
		{
			name: "camelCase control response",
			raw: `{
				"totalTokens": 50000,
				"maxTokens": 200000,
				"utilizationPct": 25,
				"byCategory": {
					"conversation": {"tokens": 30000, "pct": 15},
					"system": {"tokens": 20000, "pct": 10}
				}
			}`,
			want: ContextUsage{
				TotalTokens:    50000,
				MaxTokens:      200000,
				UtilizationPct: 25,
				ByCategory: map[string]CategoryUsage{
					"conversation": {Tokens: 30000, Pct: 15},
					"system":       {Tokens: 20000, Pct: 10},
				},
			},
		},
		{
			name: "snake_case control response",
			raw: `{
				"total_tokens": 75000,
				"max_tokens": 200000,
				"utilization_pct": 37.5,
				"by_category": {
					"conversation": {"tokens": 50000, "pct": 25},
					"system": {"tokens": 25000, "pct": 12.5}
				}
			}`,
			want: ContextUsage{
				TotalTokens:    75000,
				MaxTokens:      200000,
				UtilizationPct: 37.5,
				ByCategory: map[string]CategoryUsage{
					"conversation": {Tokens: 50000, Pct: 25},
					"system":       {Tokens: 25000, Pct: 12.5},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got ContextUsage
			if err := json.Unmarshal([]byte(tt.raw), &got); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if got.TotalTokens != tt.want.TotalTokens {
				t.Errorf("TotalTokens = %d, want %d", got.TotalTokens, tt.want.TotalTokens)
			}
			if got.MaxTokens != tt.want.MaxTokens {
				t.Errorf("MaxTokens = %d, want %d", got.MaxTokens, tt.want.MaxTokens)
			}
			if got.UtilizationPct != tt.want.UtilizationPct {
				t.Errorf("UtilizationPct = %f, want %f", got.UtilizationPct, tt.want.UtilizationPct)
			}
			if len(got.ByCategory) != len(tt.want.ByCategory) {
				t.Fatalf("ByCategory len = %d, want %d", len(got.ByCategory), len(tt.want.ByCategory))
			}
			for key, want := range tt.want.ByCategory {
				gotCat, ok := got.ByCategory[key]
				if !ok {
					t.Fatalf("ByCategory missing key %q", key)
				}
				if gotCat.Tokens != want.Tokens {
					t.Errorf("ByCategory[%q].Tokens = %d, want %d", key, gotCat.Tokens, want.Tokens)
				}
				if gotCat.Pct != want.Pct {
					t.Errorf("ByCategory[%q].Pct = %f, want %f", key, gotCat.Pct, want.Pct)
				}
			}
		})
	}
}

func FuzzContextUsage_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"totalTokens":100,"maxTokens":1000,"utilizationPct":10.0}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"totalTokens":0,"maxTokens":0,"utilizationPct":0,"byCategory":{}}`))
	f.Add([]byte(`{"total_tokens":100,"max_tokens":1000,"utilization_pct":10.0}`))
	f.Add([]byte(`{"total_tokens":0,"max_tokens":0,"utilization_pct":0,"by_category":{}}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var usage ContextUsage
		_ = json.Unmarshal(data, &usage)
	})
}
