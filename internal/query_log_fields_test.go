package internal

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestControlRequestLogsDoNotSerializePermissionInput(t *testing.T) {
	t.Parallel()

	const secretCommand = "SECRET_TOKEN=abc123 ./deploy"
	core, logs := observer.New(zapcore.DebugLevel)
	logger := log.NewLoggerFromZap(zap.New(core))
	opts := types.NewClaudeAgentOptions().WithCanUseTool(
		func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
			return types.PermissionResultAllow{}, nil
		},
	)
	query := NewQuery(context.Background(), newMockTransport(), opts, logger, true)

	query.handlerWg.Add(1)
	query.handleControlRequest(context.Background(), &types.SystemMessage{
		Type:      "system",
		Subtype:   "control_request",
		RequestID: "perm-log",
		Request: map[string]interface{}{
			"subtype":   "can_use_tool",
			"tool_name": "Bash",
			"input": map[string]interface{}{
				"command": secretCommand,
			},
			"permission_suggestions": []interface{}{
				map[string]interface{}{"type": "tool", "value": "Bash"},
			},
		},
	})

	for _, entry := range logs.All() {
		fields := entry.ContextMap()
		for _, forbiddenKey := range []string{"request", "request_data", "input", "data", "result"} {
			if _, ok := fields[forbiddenKey]; ok {
				t.Fatalf("log %q contains raw field %q: %#v", entry.Message, forbiddenKey, fields)
			}
		}
		if strings.Contains(fmt.Sprint(fields), secretCommand) {
			t.Fatalf("log %q serialized permission input: %#v", entry.Message, fields)
		}
	}

	parsed := logs.FilterMessage("handlePermissionRequest: parsed fields").All()
	if len(parsed) == 0 {
		t.Fatal("missing parsed permission log")
	}
	if got := fmt.Sprint(parsed[0].ContextMap()["input_key_count"]); got != "1" {
		t.Fatalf("input_key_count = %s, want 1", got)
	}
	sent := logs.FilterMessage("sendSuccessResponse: sending control_response").All()
	if len(sent) == 0 {
		t.Fatal("missing control response send log")
	}
	if _, ok := sent[0].ContextMap()["data"]; ok {
		t.Fatal("control response send log contains raw data field")
	}
}
