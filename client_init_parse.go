package claude

import "github.com/hishamkaram/claude-agent-sdk-go/types"

// parseInitResult converts the raw initialize response map into a typed InitializeResult.
func parseInitResult(raw map[string]interface{}) *types.InitializeResult {
	if raw == nil {
		return nil
	}

	result := &types.InitializeResult{Raw: raw}
	if commands := parseInitCommands(raw); commands != nil {
		result.Commands = commands
	}
	if models := parseInitModels(raw); models != nil {
		result.Models = models
	}
	if agents := parseInitAgents(raw); agents != nil {
		result.Agents = agents
	}
	return result
}

func initResultSlice(raw map[string]interface{}, key string) ([]interface{}, bool) {
	v, ok := raw[key]
	if !ok {
		return nil, false
	}
	slice, ok := v.([]interface{})
	if !ok {
		return nil, false
	}
	return slice, true
}

func parseInitCommands(raw map[string]interface{}) []types.SlashCommand {
	slice, ok := initResultSlice(raw, "commands")
	if !ok {
		return nil
	}
	commands := make([]types.SlashCommand, 0, len(slice))
	for _, item := range slice {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		cmd := types.SlashCommand{}
		if name, ok := m["name"].(string); ok {
			cmd.Name = name
		}
		if desc, ok := m["description"].(string); ok {
			cmd.Description = desc
		}
		if hint, ok := m["argumentHint"].(string); ok {
			cmd.ArgumentHint = hint
		}
		if cmd.Name != "" {
			commands = append(commands, cmd)
		}
	}
	return commands
}

func parseInitModels(raw map[string]interface{}) []types.ModelInfo {
	slice, ok := initResultSlice(raw, "models")
	if !ok {
		return nil
	}
	models := make([]types.ModelInfo, 0, len(slice))
	for _, item := range slice {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		info := parseInitModel(m)
		if shouldKeepInitModel(info) {
			models = append(models, info)
		}
	}
	return models
}

func parseInitModel(raw map[string]interface{}) types.ModelInfo {
	effortLevels, _ := raw["supportedEffortLevels"].([]interface{})
	return types.ModelInfo{
		Value:                    modelStringField(raw, "value"),
		DisplayName:              modelStringField(raw, "displayName"),
		Description:              modelStringField(raw, "description"),
		ResolvedModel:            modelStringField(raw, "resolvedModel"),
		SupportsEffort:           modelBoolField(raw, "supportsEffort"),
		SupportedEffortLevels:    parseEffortLevels(effortLevels),
		SupportsAdaptiveThinking: modelBoolField(raw, "supportsAdaptiveThinking"),
		SupportsFastMode:         modelBoolField(raw, "supportsFastMode"),
		SupportsAutoMode:         modelBoolField(raw, "supportsAutoMode"),
		Disabled:                 modelBoolField(raw, "disabled"),
		Raw:                      cloneInitMap(raw),
	}
}

func shouldKeepInitModel(info types.ModelInfo) bool {
	return info.Value != "" || info.DisplayName != "" || info.ResolvedModel != "" || info.Description != ""
}

func cloneInitMap(raw map[string]interface{}) map[string]interface{} {
	if raw == nil {
		return nil
	}
	out := make(map[string]interface{}, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}

func modelStringField(raw map[string]interface{}, key string) string {
	value, _ := raw[key].(string)
	return value
}

func modelBoolField(raw map[string]interface{}, key string) bool {
	value, _ := raw[key].(bool)
	return value
}

func parseEffortLevels(values []interface{}) []types.EffortLevel {
	levels := make([]types.EffortLevel, 0, len(values))
	for _, value := range values {
		raw, ok := value.(string)
		if !ok || raw == "" {
			continue
		}
		levels = append(levels, types.EffortLevel(raw))
	}
	return levels
}

func parseInitAgents(raw map[string]interface{}) []types.AgentInfo {
	slice, ok := initResultSlice(raw, "agents")
	if !ok {
		return nil
	}
	agents := make([]types.AgentInfo, 0, len(slice))
	for _, item := range slice {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		agent := types.AgentInfo{}
		if v, ok := m["name"].(string); ok {
			agent.Name = v
		}
		if v, ok := m["description"].(string); ok {
			agent.Description = v
		}
		if v, ok := m["model"].(string); ok {
			agent.Model = v
		}
		if agent.Name != "" {
			agents = append(agents, agent)
		}
	}
	return agents
}
