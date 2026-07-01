package types

import "encoding/json"

// UnknownMessage represents an unrecognized message type.
// It preserves the raw JSON for forward compatibility with future CLI versions.
type UnknownMessage struct {
	Type    string          `json:"type"`
	RawJSON json.RawMessage `json:"-"`
}

func (m *UnknownMessage) GetMessageType() string    { return m.Type }
func (m *UnknownMessage) ShouldDisplayToUser() bool { return false }
func (m *UnknownMessage) isMessage()                {}

// truncateRaw truncates raw JSON for safe inclusion in error messages.
// Prevents sensitive subprocess output from leaking into error structs.
func truncateRaw(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// UnmarshalMessage unmarshals a JSON message into the appropriate message type.
// msgPtr constrains PT to a *T that also satisfies Message, so decodeMessage
// can return the typed pointer through the Message interface. Go infers PT from
// T (the constraint's core type is *T) at each decoder-table entry.
type msgPtr[T any] interface {
	*T
	Message
}

// decodeMessage unmarshals data into a fresh T and returns it as a Message,
// wrapping any decode failure with the given context. Every concrete message
// case shares this single body.
func decodeMessage[T any, PT msgPtr[T]](data []byte, context string) (Message, error) {
	var msg T
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, NewJSONDecodeErrorWithCause(context, truncateRaw(string(data), 200), err)
	}
	return PT(&msg), nil
}

// decodeMessageFunc binds a concrete message type and its error context into a
// dispatch-table decoder.
func decodeMessageFunc[T any, PT msgPtr[T]](context string) func([]byte) (Message, error) {
	return func(data []byte) (Message, error) {
		return decodeMessage[T, PT](data, context)
	}
}

// messageDecoders maps each wire "type" to its concrete-type decoder. The
// "system" type routes through unmarshalSystemMessage for subtype dispatch; any
// type absent from this table falls through to UnknownMessage in
// UnmarshalMessage. control_request/control_response both decode to
// SystemMessage, preserving the original combined switch case.
var messageDecoders = map[string]func([]byte) (Message, error){
	"user":              decodeMessageFunc[UserMessage]("failed to unmarshal user message"),
	"assistant":         decodeMessageFunc[AssistantMessage]("failed to unmarshal assistant message"),
	"system":            unmarshalSystemMessage,
	"control_request":   decodeMessageFunc[SystemMessage]("failed to unmarshal system message"),
	"control_response":  decodeMessageFunc[SystemMessage]("failed to unmarshal system message"),
	"result":            decodeMessageFunc[ResultMessage]("failed to unmarshal result message"),
	"stream_event":      decodeMessageFunc[StreamEvent]("failed to unmarshal stream event"),
	"tool_progress":     decodeMessageFunc[ToolProgressMessage]("failed to unmarshal tool progress message"),
	"auth_status":       decodeMessageFunc[AuthStatusMessage]("failed to unmarshal auth status message"),
	"tool_use_summary":  decodeMessageFunc[ToolUseSummaryMessage]("failed to unmarshal tool use summary message"),
	"rate_limit_event":  decodeMessageFunc[RateLimitEvent]("failed to unmarshal rate limit event"),
	"prompt_suggestion": decodeMessageFunc[PromptSuggestionMessage]("failed to unmarshal prompt suggestion message"),
}

func UnmarshalMessage(data []byte) (Message, error) {
	var typeCheck struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typeCheck); err != nil {
		return nil, NewJSONDecodeErrorWithCause("failed to determine message type", truncateRaw(string(data), 200), err)
	}

	if typeCheck.Type == "" {
		return nil, NewMessageParseError("missing or empty type field")
	}

	if decode, ok := messageDecoders[typeCheck.Type]; ok {
		return decode(data)
	}
	return &UnknownMessage{
		Type:    typeCheck.Type,
		RawJSON: append(json.RawMessage(nil), data...),
	}, nil
}

// systemMessageDecoders maps each system "subtype" to its concrete-type
// decoder. Any subtype absent from this table (including the empty subtype)
// falls through to the plain SystemMessage decode in unmarshalSystemMessage.
var systemMessageDecoders = map[string]func([]byte) (Message, error){
	SystemSubtypeCompactBoundary:  decodeMessageFunc[CompactBoundaryMessage]("failed to unmarshal compact boundary message"),
	SystemSubtypeStatus:           decodeMessageFunc[StatusMessage]("failed to unmarshal status message"),
	SystemSubtypeHookStarted:      decodeMessageFunc[HookStartedMessage]("failed to unmarshal hook started message"),
	SystemSubtypeHookProgress:     decodeMessageFunc[HookProgressMessage]("failed to unmarshal hook progress message"),
	SystemSubtypeHookResponse:     decodeMessageFunc[HookResponseMessage]("failed to unmarshal hook response message"),
	SystemSubtypeTaskNotification: decodeMessageFunc[TaskNotificationMessage]("failed to unmarshal task notification message"),
	SystemSubtypeTaskStarted:      decodeMessageFunc[TaskStartedMessage]("failed to unmarshal task started message"),
	SystemSubtypeTaskProgress:     decodeMessageFunc[TaskProgressMessage]("failed to unmarshal task progress message"),
	SystemSubtypeTaskUpdated:      decodeMessageFunc[TaskUpdatedMessage]("failed to unmarshal task updated message"),
	SystemSubtypeFilesPersisted:   decodeMessageFunc[FilesPersistedEvent]("failed to unmarshal files persisted event"),
}

// unmarshalSystemMessage handles system message subtype routing.
func unmarshalSystemMessage(data []byte) (Message, error) {
	var subtypeCheck struct {
		Subtype string `json:"subtype"`
	}
	if err := json.Unmarshal(data, &subtypeCheck); err != nil {
		return nil, NewJSONDecodeErrorWithCause("failed to extract system subtype", truncateRaw(string(data), 200), err)
	}

	if decode, ok := systemMessageDecoders[subtypeCheck.Subtype]; ok {
		return decode(data)
	}
	return decodeMessage[SystemMessage](data, "failed to unmarshal system message")
}
