package internal

import (
	"fmt"
	"sync/atomic"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// buildHooksConfig builds the initialize request's hooks configuration, keyed by
// event name. It registers each hook callback (assigning a stable callback id)
// as a side effect and skips events with no matchers. The returned map is empty
// (never nil) when no hooks are configured.
func (q *Query) buildHooksConfig() map[string]interface{} {
	hooksConfig := make(map[string]interface{})
	if q.hooks == nil {
		return hooksConfig
	}
	for event, matchers := range q.hooks {
		if len(matchers) == 0 {
			continue
		}
		hooksConfig[string(event)] = q.buildEventHooks(matchers)
	}
	return hooksConfig
}

// buildEventHooks converts the matchers for a single hook event into the wire
// shape ([]{hookCallbackIds, matcher?}), registering each callback to obtain its
// id.
func (q *Query) buildEventHooks(matchers []types.HookMatcher) []map[string]interface{} {
	eventHooks := make([]map[string]interface{}, 0, len(matchers))
	for _, matcher := range matchers {
		callbackIDs := make([]string, 0, len(matcher.Hooks))
		for _, callback := range matcher.Hooks {
			callbackIDs = append(callbackIDs, q.registerHookCallback(callback))
		}
		hookConfig := map[string]interface{}{
			"hookCallbackIds": callbackIDs,
		}
		if matcher.Matcher != nil {
			hookConfig["matcher"] = *matcher.Matcher
		}
		eventHooks = append(eventHooks, hookConfig)
	}
	return eventHooks
}

// buildInitializeRequest assembles the initialize control request from the
// prepared hooks config and the option-gated fields (promptSuggestions,
// jsonSchema). Keys are added only when their source is present so the emitted
// request shape is unchanged from the original inline construction.
func (q *Query) buildInitializeRequest(hooksConfig map[string]interface{}) map[string]interface{} {
	request := map[string]interface{}{
		"subtype": "initialize",
	}
	if len(hooksConfig) > 0 {
		request["hooks"] = hooksConfig
	}
	if q.options != nil && q.options.PromptSuggestions {
		request["promptSuggestions"] = true
	}
	if q.options != nil && q.options.OutputFormat != nil && q.options.OutputFormat.Schema != nil {
		request["jsonSchema"] = q.options.OutputFormat.Schema
	}
	return request
}

// handleHookCallback handles a hook callback request.
func (q *Query) handleHookCallback(requestData map[string]interface{}) (map[string]interface{}, error) {
	callbackID, _ := requestData["callback_id"].(string)
	input := requestData["input"]
	var toolUseID *string
	if raw, ok := requestData["tool_use_id"].(string); ok && raw != "" {
		toolUseID = &raw
	}

	if callbackID == "" {
		return nil, types.NewControlProtocolError("missing callback_id in hook callback request")
	}

	// Find callback
	q.mu.Lock()
	callback, exists := q.hookCallbacks[callbackID]
	q.mu.Unlock()

	if !exists {
		return nil, types.NewControlProtocolError("no hook callback found for ID: " + callbackID)
	}

	// Build hook context
	hookCtx := types.HookContext{}

	// Call hook callback
	hookOutput, err := callback(q.ctx, input, toolUseID, hookCtx)
	if err != nil {
		return nil, err
	}

	// Convert hook output to response
	// The callback should return a map[string]interface{} representing the hook output
	response, ok := hookOutput.(map[string]interface{})
	if !ok {
		return nil, types.NewControlProtocolError("hook callback must return map[string]interface{}")
	}

	return response, nil
}

// registerHookCallback registers a hook callback and returns its ID.
func (q *Query) registerHookCallback(callback types.HookCallbackFunc) string {
	q.mu.Lock()
	defer q.mu.Unlock()

	id := atomic.AddInt64(&q.nextHookCallbackID, 1)
	callbackID := fmt.Sprintf("hook_%d", id)
	q.hookCallbacks[callbackID] = callback
	return callbackID
}

func (q *Query) clearHookCallbacks() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.hookCallbacks = make(map[string]types.HookCallbackFunc)
}
