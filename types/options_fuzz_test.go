package types

import (
	"encoding/json"
	"testing"
)

// --- Fuzz Tests (Phase C) ---

func FuzzThinkingConfig_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"type":"adaptive"}`))
	f.Add([]byte(`{"type":"enabled","budgetTokens":10000}`))
	f.Add([]byte(`{"type":"disabled"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"type":"","budgetTokens":null}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var tc ThinkingConfig
		_ = json.Unmarshal(data, &tc)
	})
}

func FuzzOutputFormat_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"type":"json_schema","schema":{"type":"object"},"name":"test"}`))
	f.Add([]byte(`{"type":"json_schema"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"type":"","schema":null}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var of OutputFormat
		_ = json.Unmarshal(data, &of)
	})
}

func FuzzSandboxConfig_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"enabled":true,"network":{"allowedDomains":["example.com"]}}`))
	f.Add([]byte(`{"enabled":false,"filesystem":{"allowWrite":["/tmp"]}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"autoAllowBashIfSandboxed":true,"excludedCommands":["rm"]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var sc SandboxConfig
		_ = json.Unmarshal(data, &sc)
	})
}

// ===== Phase D: Fuzz Tests =====

func FuzzToolConfig_Unmarshal(f *testing.F) {
	f.Add([]byte(`{"bash":{"timeout":30000,"command":"/bin/bash"},"computer":{"display":1,"width":1920,"height":1080}}`))
	f.Add([]byte(`{"bash":{"timeout":0}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"computer":null,"bash":null}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var tc ToolConfig
		_ = json.Unmarshal(data, &tc)
	})
}
