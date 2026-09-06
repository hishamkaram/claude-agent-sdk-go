package types

import (
	"encoding/json"
	"testing"
)

func TestAssistantMessagePreservesProviderMessageID(t *testing.T) {
	t.Parallel()
	for _, input := range []string{
		`{"type":"assistant","uuid":"envelope-id","message":{"id":"msg-provider","content":[]}}`,
		`{"type":"assistant","uuid":"envelope-id","message_id":"msg-provider","content":[]}`,
	} {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			var message AssistantMessage
			if err := json.Unmarshal([]byte(input), &message); err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(message)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]interface{}
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if got["message_id"] != "msg-provider" || got["uuid"] != "envelope-id" {
				t.Fatalf("identity lost: %s", data)
			}
		})
	}
}
