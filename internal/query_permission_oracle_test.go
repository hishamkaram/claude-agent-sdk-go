package internal

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// permissionOraclePath is the golden file locking the exact response map (or
// error string) handlePermissionRequest produces for every branch. Regenerate
// with UPDATE_PERMISSION_ORACLE=1 after an INTENTIONAL behavior change only.
var permissionOraclePath = filepath.Join("testdata", "permission_oracle.golden.json")

// permissionOracleCase drives handlePermissionRequest with a fixed requestData
// and a fixed canUseTool callback result, capturing the externally-observable
// outcome. canUseToolNil omits the callback entirely (nil-callback branch).
type permissionOracleCase struct {
	name           string
	requestData    map[string]interface{}
	canUseToolNil  bool
	callbackResult interface{}
	callbackError  error
	// captureSuggestions, when set, makes the callback echo the number of
	// parsed permission suggestions into the allow result's UpdatedInput so the
	// golden is sensitive to the suggestion-parsing loop.
	captureSuggestions bool
}

func permissionOracleCases() []permissionOracleCase {
	upd := func(m map[string]interface{}) *map[string]interface{} { return &m }
	return []permissionOracleCase{
		{
			name:           "allow_value_default_input",
			requestData:    map[string]interface{}{"tool_name": "Bash", "input": map[string]interface{}{"command": "ls"}},
			callbackResult: types.PermissionResultAllow{Behavior: "allow"},
		},
		{
			name:        "allow_value_updated_input",
			requestData: map[string]interface{}{"tool_name": "Write", "input": map[string]interface{}{"file_path": "/tmp/a"}},
			callbackResult: types.PermissionResultAllow{
				Behavior:     "allow",
				UpdatedInput: upd(map[string]interface{}{"file_path": "/tmp/sanitized"}),
			},
		},
		{
			name:        "allow_value_updated_permissions",
			requestData: map[string]interface{}{"tool_name": "Bash", "input": map[string]interface{}{"command": "ls"}},
			callbackResult: types.PermissionResultAllow{
				Behavior:           "allow",
				UpdatedPermissions: []types.PermissionUpdate{{Type: "addRules"}},
			},
		},
		{
			name:           "allow_pointer_default_input",
			requestData:    map[string]interface{}{"tool_name": "Bash", "input": map[string]interface{}{"command": "ls"}},
			callbackResult: &types.PermissionResultAllow{Behavior: "allow"},
		},
		{
			name:        "allow_pointer_updated_input_and_permissions",
			requestData: map[string]interface{}{"tool_name": "Write", "input": map[string]interface{}{"file_path": "/tmp/a"}},
			callbackResult: &types.PermissionResultAllow{
				Behavior:           "allow",
				UpdatedInput:       upd(map[string]interface{}{"file_path": "/tmp/b"}),
				UpdatedPermissions: []types.PermissionUpdate{{Type: "setMode"}},
			},
		},
		{
			name:           "deny_value_message",
			requestData:    map[string]interface{}{"tool_name": "Write", "input": map[string]interface{}{"file_path": "/etc/passwd"}},
			callbackResult: types.PermissionResultDeny{Behavior: "deny", Message: "Access denied"},
		},
		{
			name:           "deny_value_message_interrupt",
			requestData:    map[string]interface{}{"tool_name": "Write", "input": map[string]interface{}{}},
			callbackResult: types.PermissionResultDeny{Behavior: "deny", Message: "stop", Interrupt: true},
		},
		{
			name:           "deny_pointer_message",
			requestData:    map[string]interface{}{"tool_name": "Write", "input": map[string]interface{}{}},
			callbackResult: &types.PermissionResultDeny{Behavior: "deny", Message: "nope"},
		},
		{
			name:           "deny_pointer_interrupt_only",
			requestData:    map[string]interface{}{"tool_name": "Write", "input": map[string]interface{}{}},
			callbackResult: &types.PermissionResultDeny{Behavior: "deny", Interrupt: true},
		},
		{
			name:           "nil_input_normalized",
			requestData:    map[string]interface{}{"tool_name": "ExitPlanMode", "input": nil},
			callbackResult: types.PermissionResultAllow{Behavior: "allow"},
		},
		{
			name: "with_suggestions_parsed",
			requestData: map[string]interface{}{
				"tool_name": "Bash",
				"input":     map[string]interface{}{"command": "ls"},
				"permission_suggestions": []interface{}{
					map[string]interface{}{"type": "addRules"},
					map[string]interface{}{"type": "setMode"},
				},
			},
			captureSuggestions: true,
		},
		{
			name:          "err_callback_nil",
			requestData:   map[string]interface{}{"tool_name": "Bash", "input": map[string]interface{}{}},
			canUseToolNil: true,
		},
		{
			name:        "err_tool_name_wrong_type",
			requestData: map[string]interface{}{"tool_name": 42, "input": map[string]interface{}{}},
		},
		{
			name:        "err_input_wrong_type",
			requestData: map[string]interface{}{"tool_name": "Bash", "input": "not-a-map"},
		},
		{
			name: "err_suggestions_wrong_type",
			requestData: map[string]interface{}{
				"tool_name":              "Bash",
				"input":                  map[string]interface{}{},
				"permission_suggestions": "not-an-array",
			},
		},
		{
			name:        "err_missing_tool_name",
			requestData: map[string]interface{}{"input": map[string]interface{}{}},
		},
		{
			// Precedence lock: input-type check fires BEFORE the missing-tool_name
			// check, so an absent tool_name + malformed input yields the input
			// error, not the missing-tool_name error.
			name:        "err_precedence_missing_toolname_bad_input",
			requestData: map[string]interface{}{"input": "not-a-map"},
		},
		{
			// Precedence lock: suggestions-type check fires BEFORE the
			// missing-tool_name check.
			name:        "err_precedence_missing_toolname_bad_suggestions",
			requestData: map[string]interface{}{"permission_suggestions": "not-an-array"},
		},
		{
			name:          "err_callback_error",
			requestData:   map[string]interface{}{"tool_name": "Bash", "input": map[string]interface{}{}},
			callbackError: types.NewControlProtocolError("boom"),
		},
		{
			name:           "err_invalid_result_type",
			requestData:    map[string]interface{}{"tool_name": "Bash", "input": map[string]interface{}{}},
			callbackResult: 12345,
		},
	}
}

// permissionSignature is the deterministic outcome string captured per case:
// "OK|<sorted-key json of response>" on success, "ERR|<error string>" on error.
func permissionSignature(t *testing.T, tc permissionOracleCase) string {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	transport := newMockTransport()
	opts := types.NewClaudeAgentOptions()
	if !tc.canUseToolNil {
		opts = opts.WithCanUseTool(
			func(_ context.Context, _ string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
				if tc.callbackError != nil {
					return nil, tc.callbackError
				}
				if tc.captureSuggestions {
					return types.PermissionResultAllow{
						Behavior:     "allow",
						UpdatedInput: &map[string]interface{}{"suggestion_count": len(permCtx.Suggestions)},
					}, nil
				}
				return tc.callbackResult, nil
			},
		)
	}
	q := NewQuery(ctx, transport, opts, log.NewLogger(false), true)

	resp, err := q.handlePermissionRequest(ctx, tc.requestData)
	if err != nil {
		return "ERR|" + err.Error()
	}
	data, mErr := json.Marshal(resp)
	if mErr != nil {
		t.Fatalf("marshal response: %v", mErr)
	}
	return "OK|" + string(data)
}

// TestHandlePermissionRequest_Oracle is the golden-master regression net for the
// handlePermissionRequest refactor. It locks the exact response/error for every
// branch so an extraction that drifts any mapping is caught byte-for-byte.
func TestHandlePermissionRequest_Oracle(t *testing.T) {
	t.Parallel()

	got := make(map[string]string)
	for _, tc := range permissionOracleCases() {
		got[tc.name] = permissionSignature(t, tc)
	}

	if os.Getenv("UPDATE_PERMISSION_ORACLE") == "1" {
		writePermissionOracle(t, got)
		return
	}

	want := readPermissionOracle(t)
	// Every current case must match the golden.
	for name, sig := range got {
		if want[name] != sig {
			t.Errorf("case %q:\n got: %s\nwant: %s", name, sig, want[name])
		}
	}
	// The golden must not carry stale cases the test no longer produces.
	for name := range want {
		if _, ok := got[name]; !ok {
			t.Errorf("golden has stale case %q not produced by the test", name)
		}
	}
}

func writePermissionOracle(t *testing.T, got map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(permissionOraclePath), 0o755); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}
	// Marshal with sorted keys for a stable golden diff.
	names := make([]string, 0, len(got))
	for name := range got {
		names = append(names, name)
	}
	sort.Strings(names)
	ordered := make(map[string]string, len(got))
	for _, n := range names {
		ordered[n] = got[n]
	}
	data, err := json.MarshalIndent(ordered, "", "  ")
	if err != nil {
		t.Fatalf("marshal golden: %v", err)
	}
	if err := os.WriteFile(permissionOraclePath, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write golden: %v", err)
	}
	t.Logf("regenerated %s with %d cases", permissionOraclePath, len(got))
}

func readPermissionOracle(t *testing.T) map[string]string {
	t.Helper()
	data, err := os.ReadFile(permissionOraclePath)
	if err != nil {
		t.Fatalf("read golden (run UPDATE_PERMISSION_ORACLE=1 to seed): %v", err)
	}
	var want map[string]string
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}
	return want
}
