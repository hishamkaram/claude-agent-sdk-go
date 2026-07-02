package claude

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal"
	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

type runtimeSettingsTransport struct {
	mu       sync.Mutex
	messages chan types.Message
	writes   []string
	closed   bool
}

func newRuntimeSettingsTransport() *runtimeSettingsTransport {
	return &runtimeSettingsTransport{messages: make(chan types.Message, 10)}
}

func (t *runtimeSettingsTransport) Connect(context.Context) error { return nil }

func (t *runtimeSettingsTransport) Close(context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.closed {
		close(t.messages)
		t.closed = true
	}
	return nil
}

func (t *runtimeSettingsTransport) Write(_ context.Context, data string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.writes = append(t.writes, data)
	return nil
}

func (t *runtimeSettingsTransport) ReadMessages(context.Context) <-chan types.Message {
	return t.messages
}

func (t *runtimeSettingsTransport) OnError(error) {}
func (t *runtimeSettingsTransport) IsReady() bool { return true }
func (t *runtimeSettingsTransport) GetError() error {
	return nil
}

func (t *runtimeSettingsTransport) writeAt(index int) (map[string]interface{}, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.writes) <= index {
		return nil, false
	}
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(t.writes[index]), &msg); err != nil {
		return nil, false
	}
	return msg, true
}

func (t *runtimeSettingsTransport) sendControlResponse(request map[string]interface{}, fields map[string]interface{}) {
	requestID, _ := request["request_id"].(string)
	if fields == nil {
		fields = map[string]interface{}{}
	}
	resp := map[string]interface{}{
		"subtype":    "success",
		"request_id": requestID,
		"response":   fields,
	}
	t.messages <- &types.SystemMessage{
		Type:     "control_response",
		Subtype:  "control_response",
		Response: resp,
	}
}

func newRuntimeSettingsClient(t *testing.T) (*Client, *runtimeSettingsTransport) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	transport := newRuntimeSettingsTransport()
	query := internal.NewQuery(ctx, transport, types.NewClaudeAgentOptions(), log.NewLogger(false), true)
	if err := query.Start(ctx); err != nil {
		t.Fatalf("query.Start: %v", err)
	}
	t.Cleanup(func() { _ = query.Stop(ctx) })

	return &Client{query: query, connected: true}, transport
}

func waitForControlRequest(t *testing.T, transport *runtimeSettingsTransport, index int) map[string]interface{} {
	t.Helper()
	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if msg, ok := transport.writeAt(index); ok {
			return msg
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for control request %d", index)
		case <-ticker.C:
		}
	}
}

func controlRequestPayload(t *testing.T, outer map[string]interface{}) map[string]interface{} {
	t.Helper()
	payload, ok := outer["request"].(map[string]interface{})
	if !ok {
		t.Fatalf("request payload missing or wrong type: %#v", outer["request"])
	}
	return payload
}

func TestClientSetEffort_ApplyFlagSettingsAndVerifiesReadback(t *testing.T) {
	t.Parallel()
	client, transport := newRuntimeSettingsClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.SetEffort(ctx, types.EffortHigh)
	}()

	applyReq := waitForControlRequest(t, transport, 0)
	applyPayload := controlRequestPayload(t, applyReq)
	if applyPayload["subtype"] != "apply_flag_settings" {
		t.Fatalf("subtype = %q, want apply_flag_settings", applyPayload["subtype"])
	}
	flagSettings, ok := applyPayload["flagSettings"].(map[string]interface{})
	if !ok {
		t.Fatalf("flagSettings missing or wrong type: %#v", applyPayload["flagSettings"])
	}
	if flagSettings["effortLevel"] != "high" {
		t.Fatalf("effortLevel = %q, want high", flagSettings["effortLevel"])
	}
	transport.sendControlResponse(applyReq, nil)

	getReq := waitForControlRequest(t, transport, 1)
	getPayload := controlRequestPayload(t, getReq)
	if getPayload["subtype"] != "get_settings" {
		t.Fatalf("subtype = %q, want get_settings", getPayload["subtype"])
	}
	transport.sendControlResponse(getReq, map[string]interface{}{
		"applied": map[string]interface{}{
			"model":  "sonnet",
			"effort": "high",
		},
	})

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("SetEffort returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SetEffort")
	}
}

func TestClientSetUltracode_ApplyFlagSettingsAndVerifiesReadback(t *testing.T) {
	t.Parallel()
	client, transport := newRuntimeSettingsClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.SetUltracode(ctx, true)
	}()

	applyReq := waitForControlRequest(t, transport, 0)
	flagSettings := controlRequestPayload(t, applyReq)["flagSettings"].(map[string]interface{})
	if flagSettings["ultracode"] != true {
		t.Fatalf("ultracode = %#v, want true", flagSettings["ultracode"])
	}
	transport.sendControlResponse(applyReq, nil)

	getReq := waitForControlRequest(t, transport, 1)
	transport.sendControlResponse(getReq, map[string]interface{}{
		"applied": map[string]interface{}{
			"model":     "sonnet",
			"effort":    "xhigh",
			"ultracode": true,
		},
	})

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("SetUltracode returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SetUltracode")
	}
}

func TestClientSetUltracode_MissingReadbackFails(t *testing.T) {
	t.Parallel()
	client, transport := newRuntimeSettingsClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.SetUltracode(ctx, false)
	}()

	applyReq := waitForControlRequest(t, transport, 0)
	flagSettings := controlRequestPayload(t, applyReq)["flagSettings"].(map[string]interface{})
	if flagSettings["ultracode"] != false {
		t.Fatalf("ultracode = %#v, want false", flagSettings["ultracode"])
	}
	transport.sendControlResponse(applyReq, nil)

	getReq := waitForControlRequest(t, transport, 1)
	transport.sendControlResponse(getReq, map[string]interface{}{
		"applied": map[string]interface{}{
			"model":  "sonnet",
			"effort": "xhigh",
		},
	})

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected missing ultracode readback error")
		}
		if !strings.Contains(err.Error(), "applied ultracode missing") {
			t.Fatalf("error = %v, want missing readback", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SetUltracode")
	}
}

func TestClientSetEffort_ReadbackMismatchFails(t *testing.T) {
	t.Parallel()
	client, transport := newRuntimeSettingsClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.SetEffort(ctx, types.EffortMax)
	}()

	applyReq := waitForControlRequest(t, transport, 0)
	transport.sendControlResponse(applyReq, nil)
	getReq := waitForControlRequest(t, transport, 1)
	transport.sendControlResponse(getReq, map[string]interface{}{
		"applied": map[string]interface{}{
			"model":  "sonnet",
			"effort": "xhigh",
		},
	})

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected readback mismatch error")
		}
		if !strings.Contains(err.Error(), `applied effort "xhigh", want "max"`) {
			t.Fatalf("error = %v, want readback mismatch", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SetEffort")
	}
}

func TestClientSetEffort_EmptyEffort(t *testing.T) {
	t.Parallel()
	client, _ := newRuntimeSettingsClient(t)
	err := client.SetEffort(context.Background(), "")
	if !errors.Is(err, types.ErrEmptyParameter) {
		t.Fatalf("SetEffort error = %v, want ErrEmptyParameter", err)
	}
}
