package internal

import (
	"encoding/json"
	"testing"
)

// BenchmarkParseMessage_User benchmarks parsing a user message.
func BenchmarkParseMessage_User(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ParseMessage(userMessageSimple)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseMessage_Assistant benchmarks parsing an assistant message.
func BenchmarkParseMessage_Assistant(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ParseMessage(assistantMessageText)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseMessage_AssistantComplex benchmarks parsing a complex assistant message.
func BenchmarkParseMessage_AssistantComplex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ParseMessage(assistantMessageMixed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseContentBlock benchmarks parsing a single content block.
func BenchmarkParseContentBlock(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ParseContentBlock(textBlockJSON)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseContentBlocks benchmarks parsing multiple content blocks.
func BenchmarkParseContentBlocks(b *testing.B) {
	rawBlocks := make([]json.RawMessage, len(multipleContentBlocks))
	for i, block := range multipleContentBlocks {
		rawBlocks[i] = json.RawMessage(block)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseContentBlocks(rawBlocks)
		if err != nil {
			b.Fatal(err)
		}
	}
}
