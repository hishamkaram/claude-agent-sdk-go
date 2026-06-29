package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"go.uber.org/zap"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// handlePermissionRequest handles a permission request for tool use. ctx is the
// inherited query lifecycle context; the canUseTool callback runs under a
// timeout derived from it.
func (q *Query) handlePermissionRequest(ctx context.Context, requestData map[string]interface{}) (map[string]interface{}, error) {
	q.logger.Debug("handlePermissionRequest: entered", controlRequestLogFields(requestData)...)

	if q.canUseTool == nil {
		q.logger.Error("handlePermissionRequest: canUseTool callback is nil")
		return nil, types.NewControlProtocolError("canUseTool callback is not provided")
	}
	q.logger.Debug("handlePermissionRequest: canUseTool callback is set")

	toolName, input, suggestions, err := q.parsePermissionRequest(requestData)
	if err != nil {
		return nil, err
	}

	permCtx := types.ToolPermissionContext{
		Suggestions: q.buildPermissionUpdates(suggestions),
	}

	// Call permission callback with a timeout derived from the inherited ctx.
	callbackTimeout := 5 * time.Minute // default
	if q.options != nil && q.options.ToolCallbackTimeout > 0 {
		callbackTimeout = q.options.ToolCallbackTimeout
	}
	callbackCtx, callbackCancel := context.WithTimeout(ctx, callbackTimeout)
	defer callbackCancel()

	q.logger.Debug("handlePermissionRequest: calling canUseTool callback",
		zap.String("tool_name", toolName),
		zap.Duration("timeout", callbackTimeout),
	)
	result, err := q.invokeCanUseTool(callbackCtx, toolName, input, permCtx)
	q.logger.Debug("handlePermissionRequest: canUseTool callback returned", permissionResultLogFields(result, err)...)
	if err != nil {
		q.logger.Error("handlePermissionRequest: canUseTool callback returned error", zap.Error(err))
		return nil, err
	}

	return permissionResultToResponse(result, input)
}

// parsePermissionRequest extracts and validates the tool name, input map, and
// raw permission suggestions from a can_use_tool request. A nil input is
// normalized to an empty map (some tools, e.g. ExitPlanMode, legitimately send
// null input) so CanUseTool is always invoked. Validation order — tool_name
// type, input type, suggestions type, then missing tool_name — is preserved
// from the original inline checks so error precedence is unchanged.
func (q *Query) parsePermissionRequest(requestData map[string]interface{}) (toolName string, input map[string]interface{}, suggestions []interface{}, err error) {
	toolName, toolNameOk := requestData["tool_name"].(string)
	if !toolNameOk {
		if requestData["tool_name"] != nil {
			q.logger.Warn("handlePermissionRequest: tool_name has unexpected type",
				zap.String("tool_name_type", fmt.Sprintf("%T", requestData["tool_name"])))
			return "", nil, nil, types.NewControlProtocolError("tool_name must be a string in permission request")
		}
	}

	input, inputOk := requestData["input"].(map[string]interface{})
	if !inputOk && requestData["input"] != nil {
		q.logger.Warn("handlePermissionRequest: input has unexpected type",
			zap.String("input_type", fmt.Sprintf("%T", requestData["input"])))
		return "", nil, nil, types.NewControlProtocolError("input must be a map in permission request")
	}

	if raw, exists := requestData["permission_suggestions"]; exists && raw != nil {
		var suggestionsOk bool
		suggestions, suggestionsOk = raw.([]interface{})
		if !suggestionsOk {
			q.logger.Warn("handlePermissionRequest: permission_suggestions has unexpected type",
				zap.String("permission_suggestions_type", fmt.Sprintf("%T", raw)))
			return "", nil, nil, types.NewControlProtocolError("permission_suggestions must be an array in permission request")
		}
	}

	q.logger.Debug("handlePermissionRequest: parsed fields",
		zap.String("tool_name", toolName),
		zap.Int("input_key_count", len(input)),
		zap.Int("permission_suggestion_count", len(suggestions)),
	)

	if toolName == "" {
		q.logger.Error("handlePermissionRequest: missing tool_name")
		return "", nil, nil, types.NewControlProtocolError("missing tool_name in permission request")
	}
	if input == nil {
		// Some tools (e.g. ExitPlanMode) legitimately send null input.
		// Normalize to an empty map so CanUseTool is always called.
		q.logger.Debug("handlePermissionRequest: nil input normalized to empty map", zap.String("tool_name", toolName))
		input = map[string]interface{}{}
	}

	return toolName, input, suggestions, nil
}

// buildPermissionUpdates converts raw permission suggestion maps into typed
// PermissionUpdate values, skipping any that are not maps or fail to round-trip
// through JSON. It always returns a non-nil (possibly empty) slice.
func (q *Query) buildPermissionUpdates(suggestions []interface{}) []types.PermissionUpdate {
	permissionUpdates := make([]types.PermissionUpdate, 0)
	for _, s := range suggestions {
		suggestionMap, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		// Parse suggestion into PermissionUpdate
		// This is a simplified version - production code should handle all fields
		suggestionJSON, err := json.Marshal(suggestionMap)
		if err != nil {
			q.logger.Warn("handlePermissionRequest: marshal suggestion", zap.Error(err))
			continue
		}
		var update types.PermissionUpdate
		if err := json.Unmarshal(suggestionJSON, &update); err == nil {
			permissionUpdates = append(permissionUpdates, update)
		}
	}
	return permissionUpdates
}

// permissionResultToResponse maps a CanUseTool callback result into the control
// protocol response map. input is the (normalized) tool input used as the
// default updatedInput for allow results that don't override it. Value and
// pointer result variants share the same mapping.
func permissionResultToResponse(result interface{}, input map[string]interface{}) (map[string]interface{}, error) {
	// Normalize pointer variants to values so each behavior is mapped once.
	switch r := result.(type) {
	case *types.PermissionResultAllow:
		result = *r
	case *types.PermissionResultDeny:
		result = *r
	}

	response := make(map[string]interface{})
	switch r := result.(type) {
	case types.PermissionResultAllow:
		updatedInput := input
		if r.UpdatedInput != nil {
			updatedInput = *r.UpdatedInput
		}
		applyAllowResponse(response, updatedInput, r.UpdatedPermissions)
	case types.PermissionResultDeny:
		applyDenyResponse(response, r.Message, r.Interrupt)
	default:
		return nil, types.NewControlProtocolError("permission callback returned invalid type")
	}
	return response, nil
}

// applyAllowResponse writes the allow-result fields into response. updatedInput
// is the already-resolved input (callback override or the original tool input).
func applyAllowResponse(response, updatedInput map[string]interface{}, updatedPermissions []types.PermissionUpdate) {
	response["behavior"] = "allow"
	response["updatedInput"] = updatedInput
	if len(updatedPermissions) > 0 {
		response["updatedPermissions"] = updatedPermissions
	}
}

// applyDenyResponse writes the deny-result fields into response, omitting an
// empty message and a false interrupt.
func applyDenyResponse(response map[string]interface{}, message string, interrupt bool) {
	response["behavior"] = "deny"
	if message != "" {
		response["message"] = message
	}
	if interrupt {
		response["interrupt"] = interrupt
	}
}

func (q *Query) invokeCanUseTool(callbackCtx context.Context, toolName string, input map[string]interface{}, ctx types.ToolPermissionContext) (result interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			cause, ok := r.(error)
			if !ok {
				cause = fmt.Errorf("%v", r)
			}
			q.logger.Error("panic in canUseTool callback recovered",
				zap.String("panic_type", fmt.Sprintf("%T", r)),
				zap.Stack("stack"),
				zap.String("tool_name", toolName),
			)
			err = types.NewControlProtocolErrorWithCause("permission callback panicked", cause)
		}
	}()

	return q.canUseTool(callbackCtx, toolName, input, ctx)
}

// matchesToolName checks if a tool name matches a matcher pattern.
func matchesToolName(toolName string, pattern *string) bool {
	if pattern == nil || *pattern == "" {
		return true // No pattern means match all
	}

	// Use regex for pattern matching
	regex, err := regexp.Compile(*pattern)
	if err != nil {
		return false
	}

	return regex.MatchString(toolName)
}
