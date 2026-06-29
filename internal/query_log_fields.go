package internal

import (
	"fmt"

	"go.uber.org/zap"
)

func controlRequestLogFields(requestData map[string]interface{}) []zap.Field {
	fields := []zap.Field{zap.Int("request_key_count", len(requestData))}
	if requestData == nil {
		return fields
	}
	if subtype, ok := requestData["subtype"].(string); ok {
		fields = append(fields, zap.String("subtype", subtype))
	}
	if toolName, ok := requestData["tool_name"].(string); ok {
		fields = append(fields, zap.String("tool_name", toolName))
	} else if requestData["tool_name"] != nil {
		fields = append(fields, zap.String("tool_name_type", fmt.Sprintf("%T", requestData["tool_name"])))
	}
	if input, ok := requestData["input"].(map[string]interface{}); ok {
		fields = append(fields, zap.Int("input_key_count", len(input)))
	} else if requestData["input"] != nil {
		fields = append(fields, zap.String("input_type", fmt.Sprintf("%T", requestData["input"])))
	}
	if suggestions, ok := requestData["permission_suggestions"].([]interface{}); ok {
		fields = append(fields, zap.Int("permission_suggestion_count", len(suggestions)))
	} else if requestData["permission_suggestions"] != nil {
		fields = append(fields, zap.String("permission_suggestions_type", fmt.Sprintf("%T", requestData["permission_suggestions"])))
	}
	return fields
}

func permissionResultLogFields(result interface{}, err error) []zap.Field {
	return []zap.Field{
		zap.String("result_type", fmt.Sprintf("%T", result)),
		zap.Error(err),
	}
}
