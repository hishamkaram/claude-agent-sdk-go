package types

// Message is an interface for all message types from Claude.
type Message interface {
	GetMessageType() string
	ShouldDisplayToUser() bool
	isMessage()
}
