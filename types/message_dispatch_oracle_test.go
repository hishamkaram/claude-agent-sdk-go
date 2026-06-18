package types

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// message_dispatch_oracle_test.go is the behavior-equivalence oracle (golden
// master) guarding the UnmarshalMessage / unmarshalSystemMessage refactor that
// replaces the hand-written type/subtype switches with lookup tables.
//
// The refactor's failure class is a dropped or mis-keyed case, or a case wired
// to the wrong concrete type / error-context string. The signature per corpus
// entry is the concrete Go type name PLUS the json.Marshal of the parsed result
// (or the error string), so a mis-binding that yields the same outer type but a
// different field set (e.g. control_request vs control_response, both
// *SystemMessage) is still caught by the marshaled bytes.
//
// Regenerate with UPDATE_MESSAGE_ORACLE=1. Committed green on the unmodified
// switch code first; the dispatch refactor is allowed only while it stays green.

// messageCorpus exercises every top-level "type" handled by UnmarshalMessage,
// plus the empty-type and malformed-JSON guards and an unknown type.
var messageCorpus = []struct {
	name string
	raw  string
}{
	{"user", `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"hi"}]}}`},
	{"assistant", `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"ok"}]}}`},
	{"system_no_subtype", `{"type":"system"}`},
	{"control_request", `{"type":"control_request","request_id":"r1"}`},
	{"control_response", `{"type":"control_response","request_id":"r1"}`},
	{"result", `{"type":"result","subtype":"success","session_id":"s1"}`},
	{"stream_event", `{"type":"stream_event","event":{"type":"x"}}`},
	{"tool_progress", `{"type":"tool_progress"}`},
	{"auth_status", `{"type":"auth_status"}`},
	{"tool_use_summary", `{"type":"tool_use_summary"}`},
	{"rate_limit_event", `{"type":"rate_limit_event"}`},
	{"prompt_suggestion", `{"type":"prompt_suggestion"}`},
	{"unknown_type", `{"type":"some_future_type","extra":1}`},
	{"empty_type", `{"type":""}`},
	{"missing_type", `{"foo":"bar"}`},
	{"malformed", `{not json`},
}

// systemSubtypeCorpus exercises every "subtype" handled by
// unmarshalSystemMessage, plus an unknown subtype and the no-subtype default.
var systemSubtypeCorpus = []struct {
	name string
	raw  string
}{
	{"compact_boundary", fmt.Sprintf(`{"type":"system","subtype":%q}`, SystemSubtypeCompactBoundary)},
	{"status", fmt.Sprintf(`{"type":"system","subtype":%q}`, SystemSubtypeStatus)},
	{"hook_started", fmt.Sprintf(`{"type":"system","subtype":%q}`, SystemSubtypeHookStarted)},
	{"hook_progress", fmt.Sprintf(`{"type":"system","subtype":%q}`, SystemSubtypeHookProgress)},
	{"hook_response", fmt.Sprintf(`{"type":"system","subtype":%q}`, SystemSubtypeHookResponse)},
	{"task_notification", fmt.Sprintf(`{"type":"system","subtype":%q}`, SystemSubtypeTaskNotification)},
	{"task_started", fmt.Sprintf(`{"type":"system","subtype":%q}`, SystemSubtypeTaskStarted)},
	{"task_progress", fmt.Sprintf(`{"type":"system","subtype":%q}`, SystemSubtypeTaskProgress)},
	{"files_persisted", fmt.Sprintf(`{"type":"system","subtype":%q}`, SystemSubtypeFilesPersisted)},
	{"unknown_subtype", `{"type":"system","subtype":"some_future_subtype"}`},
	{"no_subtype_default", `{"type":"system","subtype":""}`},
}

func messageSignature(msg Message, err error) string {
	if err != nil {
		return "ERR: " + err.Error()
	}
	b, mErr := json.Marshal(msg)
	if mErr != nil {
		return "MARSHAL-ERR: " + mErr.Error()
	}
	return fmt.Sprintf("%T | %s", msg, string(b))
}

func computeMessageGolden(t *testing.T) map[string]string {
	t.Helper()
	golden := make(map[string]string, len(messageCorpus)+len(systemSubtypeCorpus))
	for _, c := range messageCorpus {
		msg, err := UnmarshalMessage([]byte(c.raw))
		golden["type:"+c.name] = messageSignature(msg, err)
	}
	for _, c := range systemSubtypeCorpus {
		msg, err := unmarshalSystemMessage([]byte(c.raw))
		golden["subtype:"+c.name] = messageSignature(msg, err)
	}
	return golden
}

// TestMessageDispatchOracle pins UnmarshalMessage / unmarshalSystemMessage
// dispatch against a committed golden. Regenerate with UPDATE_MESSAGE_ORACLE=1.
func TestMessageDispatchOracle(t *testing.T) {
	t.Parallel()

	goldenPath := filepath.Join("testdata", "message_dispatch_oracle.golden.json")
	got := computeMessageGolden(t)

	if os.Getenv("UPDATE_MESSAGE_ORACLE") == "1" {
		keys := make([]string, 0, len(got))
		for k := range got {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		ordered := make([][2]string, 0, len(keys))
		for _, k := range keys {
			ordered = append(ordered, [2]string{k, got[k]})
		}
		buf, err := json.MarshalIndent(ordered, "", "  ")
		if err != nil {
			t.Fatalf("marshal golden: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, append(buf, '\n'), 0o600); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated golden: %s (%d entries)", goldenPath, len(got))
		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (regenerate with UPDATE_MESSAGE_ORACLE=1): %v", err)
	}
	var ordered [][2]string
	if err := json.Unmarshal(data, &ordered); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}
	want := make(map[string]string, len(ordered))
	for _, kv := range ordered {
		want[kv[0]] = kv[1]
	}

	for k, wantSig := range want {
		gotSig, ok := got[k]
		if !ok {
			t.Errorf("dispatch case %q present in golden but missing from current run", k)
			continue
		}
		if gotSig != wantSig {
			t.Errorf("dispatch drift for %q:\n  want: %s\n  got:  %s", k, wantSig, gotSig)
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("dispatch case %q present in current run but missing from golden", k)
		}
	}
}
