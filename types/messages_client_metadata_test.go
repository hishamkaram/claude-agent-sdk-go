package types

import (
	"encoding/json"
	"testing"
)

func TestAgentInfo_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		info AgentInfo
	}{
		{
			name: "all fields populated",
			info: AgentInfo{
				Name:        "code-reviewer",
				Description: "Reviews code for style and correctness",
				Model:       "claude-sonnet-4-5-20250929",
			},
		},
		{
			name: "without optional model",
			info: AgentInfo{
				Name:        "researcher",
				Description: "Performs web research",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := json.Marshal(tt.info)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded AgentInfo
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.Name != tt.info.Name {
				t.Errorf("Name = %q, want %q", decoded.Name, tt.info.Name)
			}
			if decoded.Description != tt.info.Description {
				t.Errorf("Description = %q, want %q", decoded.Description, tt.info.Description)
			}
			if decoded.Model != tt.info.Model {
				t.Errorf("Model = %q, want %q", decoded.Model, tt.info.Model)
			}
		})
	}
}

func TestAgentInfo_OmitEmptyModel(t *testing.T) {
	t.Parallel()
	info := AgentInfo{
		Name:        "test-agent",
		Description: "A test agent",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["model"]; ok {
		t.Error("expected 'model' to be omitted when empty")
	}
	if _, ok := raw["name"]; !ok {
		t.Error("expected 'name' to be present")
	}
	if _, ok := raw["description"]; !ok {
		t.Error("expected 'description' to be present")
	}
}

func TestRewindFilesResult_JSONRoundtrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		result RewindFilesResult
	}{
		{
			name: "can rewind with changes",
			result: RewindFilesResult{
				CanRewind:    true,
				FilesChanged: []string{"main.go", "util.go"},
				Insertions:   15,
				Deletions:    8,
			},
		},
		{
			name: "cannot rewind with error",
			result: RewindFilesResult{
				CanRewind: false,
				Error:     "no checkpoint found",
			},
		},
		{
			name: "can rewind no changes",
			result: RewindFilesResult{
				CanRewind: true,
			},
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

			var decoded RewindFilesResult
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if decoded.CanRewind != tt.result.CanRewind {
				t.Errorf("CanRewind = %v, want %v", decoded.CanRewind, tt.result.CanRewind)
			}
			if decoded.Error != tt.result.Error {
				t.Errorf("Error = %q, want %q", decoded.Error, tt.result.Error)
			}
			if len(decoded.FilesChanged) != len(tt.result.FilesChanged) {
				t.Fatalf("FilesChanged len = %d, want %d", len(decoded.FilesChanged), len(tt.result.FilesChanged))
			}
			for i, f := range tt.result.FilesChanged {
				if decoded.FilesChanged[i] != f {
					t.Errorf("FilesChanged[%d] = %q, want %q", i, decoded.FilesChanged[i], f)
				}
			}
			if decoded.Insertions != tt.result.Insertions {
				t.Errorf("Insertions = %d, want %d", decoded.Insertions, tt.result.Insertions)
			}
			if decoded.Deletions != tt.result.Deletions {
				t.Errorf("Deletions = %d, want %d", decoded.Deletions, tt.result.Deletions)
			}
		})
	}
}

func TestRewindFilesResult_OmitEmpty(t *testing.T) {
	t.Parallel()
	result := RewindFilesResult{
		CanRewind: true,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// canRewind is required (not omitempty), so it should be present
	if _, ok := raw["canRewind"]; !ok {
		t.Error("expected 'canRewind' to be present")
	}

	// Optional fields should be omitted when zero
	for _, key := range []string{"error", "filesChanged", "insertions", "deletions"} {
		if _, ok := raw[key]; ok {
			t.Errorf("expected JSON key %q to be omitted when zero", key)
		}
	}
}

func TestRewindFilesResult_WireFormatKeys(t *testing.T) {
	t.Parallel()
	result := RewindFilesResult{
		CanRewind:    true,
		Error:        "test",
		FilesChanged: []string{"a.go"},
		Insertions:   1,
		Deletions:    2,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	expectedKeys := []string{"canRewind", "error", "filesChanged", "insertions", "deletions"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected camelCase JSON key %q", key)
		}
	}
}

func TestInitializeResult_AgentsField(t *testing.T) {
	t.Parallel()
	result := InitializeResult{
		Commands: []SlashCommand{{Name: "help", Description: "Show help"}},
		Models:   []ModelInfo{{Value: "claude-3-opus", DisplayName: "Opus"}},
		Agents: []AgentInfo{
			{Name: "coder", Description: "Writes code", Model: "claude-sonnet-4-5-20250929"},
			{Name: "reviewer", Description: "Reviews code"},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded InitializeResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(decoded.Agents) != 2 {
		t.Fatalf("Agents len = %d, want 2", len(decoded.Agents))
	}
	if decoded.Agents[0].Name != "coder" {
		t.Errorf("Agents[0].Name = %q, want %q", decoded.Agents[0].Name, "coder")
	}
	if decoded.Agents[1].Model != "" {
		t.Errorf("Agents[1].Model = %q, want empty", decoded.Agents[1].Model)
	}
}

func TestInitializeResult_AgentsOmitEmpty(t *testing.T) {
	t.Parallel()
	result := InitializeResult{
		Commands: []SlashCommand{{Name: "help", Description: "Show help"}},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["agents"]; ok {
		t.Error("expected 'agents' to be omitted when nil/empty")
	}
}
