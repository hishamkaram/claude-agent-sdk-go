package types

import (
	"encoding/json"
	"reflect"
	"testing"
)

// ---------------------------------------------------------------------------
// Phase C: Configuration Parity — Type Constants, JSON Roundtrip, Builders
// ---------------------------------------------------------------------------

// TestEffortLevelConstants verifies each EffortLevel constant maps to the
// correct string value expected by the Claude Code CLI.
func TestEffortLevelConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level EffortLevel
		want  string
	}{
		{name: "low", level: EffortLow, want: "low"},
		{name: "medium", level: EffortMedium, want: "medium"},
		{name: "high", level: EffortHigh, want: "high"},
		{name: "xhigh", level: EffortXHigh, want: "xhigh"},
		{name: "max", level: EffortMax, want: "max"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := string(tt.level); got != tt.want {
				t.Errorf("EffortLevel(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

// TestThinkingConfig_JSONRoundtrip verifies ThinkingConfig can be marshaled
// and unmarshaled without data loss for each variant (adaptive, enabled, disabled).
func TestThinkingConfig_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	budgetTokens := 10000

	tests := []struct {
		name   string
		config ThinkingConfig
		// checkJSON validates the intermediate JSON bytes if non-nil.
		checkJSON func(t *testing.T, data []byte)
	}{
		{
			name:   "adaptive type without budget",
			config: ThinkingConfig{Type: "adaptive"},
			checkJSON: func(t *testing.T, data []byte) {
				t.Helper()
				var m map[string]interface{}
				if err := json.Unmarshal(data, &m); err != nil {
					t.Fatalf("unmarshal to map: %v", err)
				}
				if _, ok := m["budgetTokens"]; ok {
					t.Error("budgetTokens should be omitted for adaptive config")
				}
			},
		},
		{
			name:   "enabled type with budget",
			config: ThinkingConfig{Type: "enabled", BudgetTokens: &budgetTokens},
			checkJSON: func(t *testing.T, data []byte) {
				t.Helper()
				var m map[string]interface{}
				if err := json.Unmarshal(data, &m); err != nil {
					t.Fatalf("unmarshal to map: %v", err)
				}
				v, ok := m["budgetTokens"]
				if !ok {
					t.Fatal("budgetTokens should be present for enabled config")
				}
				if int(v.(float64)) != 10000 {
					t.Errorf("budgetTokens = %v, want 10000", v)
				}
			},
		},
		{
			name:   "disabled type",
			config: ThinkingConfig{Type: "disabled"},
		},
		{
			name:   "adaptive type with summarized display",
			config: ThinkingConfig{Type: "adaptive", Display: "summarized"},
			checkJSON: func(t *testing.T, data []byte) {
				t.Helper()
				var m map[string]interface{}
				if err := json.Unmarshal(data, &m); err != nil {
					t.Fatalf("unmarshal to map: %v", err)
				}
				if m["display"] != "summarized" {
					t.Errorf("display = %v, want summarized", m["display"])
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			if tt.checkJSON != nil {
				tt.checkJSON(t, data)
			}

			var got ThinkingConfig
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			if got.Type != tt.config.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.config.Type)
			}
			if got.Display != tt.config.Display {
				t.Errorf("Display = %q, want %q", got.Display, tt.config.Display)
			}

			if tt.config.BudgetTokens == nil {
				if got.BudgetTokens != nil {
					t.Errorf("BudgetTokens should be nil, got %d", *got.BudgetTokens)
				}
			} else {
				if got.BudgetTokens == nil {
					t.Fatal("BudgetTokens should not be nil")
				}
				if *got.BudgetTokens != *tt.config.BudgetTokens {
					t.Errorf("BudgetTokens = %d, want %d", *got.BudgetTokens, *tt.config.BudgetTokens)
				}
			}
		})
	}
}

// TestOutputFormat_JSONRoundtrip verifies OutputFormat marshal/unmarshal
// with and without schema and name.
func TestOutputFormat_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	schemaName := "my_schema"

	tests := []struct {
		name   string
		format OutputFormat
	}{
		{
			name: "json_schema with schema and name",
			format: OutputFormat{
				Type: "json_schema",
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"result": map[string]interface{}{
							"type": "string",
						},
					},
				},
				Name: &schemaName,
			},
		},
		{
			name: "json_schema without schema or name",
			format: OutputFormat{
				Type: "json_schema",
			},
		},
		{
			name: "json_schema with schema only",
			format: OutputFormat{
				Type: "json_schema",
				Schema: map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.format)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var got OutputFormat
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			if got.Type != tt.format.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.format.Type)
			}

			if !reflect.DeepEqual(got.Schema, tt.format.Schema) {
				t.Errorf("Schema mismatch: got %v, want %v", got.Schema, tt.format.Schema)
			}

			if tt.format.Name == nil {
				if got.Name != nil {
					t.Errorf("Name should be nil, got %q", *got.Name)
				}
			} else {
				if got.Name == nil {
					t.Fatal("Name should not be nil")
				}
				if *got.Name != *tt.format.Name {
					t.Errorf("Name = %q, want %q", *got.Name, *tt.format.Name)
				}
			}
		})
	}
}

// TestSandboxConfig_JSONRoundtrip verifies SandboxConfig marshal/unmarshal
// including nested network and filesystem configs.
func TestSandboxConfig_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	boolTrue := true
	boolFalse := false
	proxyPort := 8080

	tests := []struct {
		name   string
		config SandboxConfig
	}{
		{
			name: "full config with nested network and filesystem",
			config: SandboxConfig{
				Enabled:                  &boolTrue,
				AutoAllowBashIfSandboxed: &boolTrue,
				AllowUnsandboxedCommands: &boolFalse,
				Network: &SandboxNetworkConfig{
					AllowedDomains:          []string{"example.com", "api.example.com"},
					AllowManagedDomainsOnly: &boolTrue,
					AllowUnixSockets:        []string{"/var/run/docker.sock"},
					AllowAllUnixSockets:     &boolFalse,
					AllowLocalBinding:       &boolTrue,
					HttpProxyPort:           &proxyPort,
				},
				Filesystem: &SandboxFilesystemConfig{
					AllowWrite:                []string{"/tmp", "/home/user/project"},
					DenyWrite:                 []string{"/etc"},
					DenyRead:                  []string{"/root"},
					AllowRead:                 []string{"/usr/local"},
					AllowManagedReadPathsOnly: &boolFalse,
				},
				IgnoreViolations: map[string][]string{
					"network": {"dns"},
				},
				EnableWeakerNestedSandbox:    &boolFalse,
				EnableWeakerNetworkIsolation: &boolFalse,
				ExcludedCommands:             []string{"rm", "dd"},
			},
		},
		{
			name: "minimal config with enabled only",
			config: SandboxConfig{
				Enabled: &boolTrue,
			},
		},
		{
			name:   "empty config",
			config: SandboxConfig{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var got SandboxConfig
			if unmarshalErr := json.Unmarshal(data, &got); unmarshalErr != nil {
				t.Fatalf("Unmarshal() error = %v", unmarshalErr)
			}

			// Compare using JSON round-trip: marshal both and compare bytes.
			// This avoids deep comparison complexity with pointers.
			wantData, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("re-Marshal original error = %v", err)
			}
			gotData, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("re-Marshal roundtripped error = %v", err)
			}
			if string(gotData) != string(wantData) {
				t.Errorf("JSON roundtrip mismatch:\n  got:  %s\n  want: %s", gotData, wantData)
			}
		})
	}
}

// TestSandboxNetworkConfig_JSONRoundtrip verifies SandboxNetworkConfig in isolation.
func TestSandboxNetworkConfig_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	boolTrue := true
	boolFalse := false
	httpPort := 3128
	socksPort := 1080

	tests := []struct {
		name   string
		config SandboxNetworkConfig
	}{
		{
			name: "all fields populated",
			config: SandboxNetworkConfig{
				AllowedDomains:          []string{"*.example.com"},
				AllowManagedDomainsOnly: &boolTrue,
				AllowUnixSockets:        []string{"/run/app.sock"},
				AllowAllUnixSockets:     &boolFalse,
				AllowLocalBinding:       &boolTrue,
				HttpProxyPort:           &httpPort,
				SocksProxyPort:          &socksPort,
			},
		},
		{
			name:   "empty config",
			config: SandboxNetworkConfig{},
		},
		{
			name: "domains only",
			config: SandboxNetworkConfig{
				AllowedDomains: []string{"api.github.com"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var got SandboxNetworkConfig
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			wantData, _ := json.Marshal(tt.config)
			gotData, _ := json.Marshal(got)
			if string(gotData) != string(wantData) {
				t.Errorf("JSON roundtrip mismatch:\n  got:  %s\n  want: %s", gotData, wantData)
			}
		})
	}
}

// TestSandboxFilesystemConfig_JSONRoundtrip verifies SandboxFilesystemConfig in isolation.
func TestSandboxFilesystemConfig_JSONRoundtrip(t *testing.T) {
	t.Parallel()

	boolTrue := true

	tests := []struct {
		name   string
		config SandboxFilesystemConfig
	}{
		{
			name: "all fields populated",
			config: SandboxFilesystemConfig{
				AllowWrite:                []string{"/tmp", "/var/data"},
				DenyWrite:                 []string{"/etc", "/usr"},
				DenyRead:                  []string{"/root/.ssh"},
				AllowRead:                 []string{"/opt/app"},
				AllowManagedReadPathsOnly: &boolTrue,
			},
		},
		{
			name:   "empty config",
			config: SandboxFilesystemConfig{},
		},
		{
			name: "write paths only",
			config: SandboxFilesystemConfig{
				AllowWrite: []string{"/tmp"},
				DenyWrite:  []string{"/etc"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			var got SandboxFilesystemConfig
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			wantData, _ := json.Marshal(tt.config)
			gotData, _ := json.Marshal(got)
			if string(gotData) != string(wantData) {
				t.Errorf("JSON roundtrip mismatch:\n  got:  %s\n  want: %s", gotData, wantData)
			}
		})
	}
}
