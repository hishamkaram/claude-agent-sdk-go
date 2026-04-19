//go:build integration
// +build integration

// Real-CLI integration tests for the MCP server lifecycle methods on the
// Client. This is the feature-170 incident class — the PWA/daemon wire
// mismatch (daemon emitted "mcp_list_servers_response", PWA listened for
// "mcp_list_response") shipped through all 5 gates because the mock daemon
// used the consumer-side type string. Only a real-peer test catches drift
// between the CLI's wire shape and the SDK's parser.
//
// Covered methods (from client.go):
//   - MCPServerStatus        (~/.go:773)
//   - ReconnectMCPServer     (:808)
//   - ToggleMCPServer        (:833)
//   - SetMCPServers          (:859)
//
// None of these tests spawn a subagent or drive the model through a turn,
// so they do NOT require CLAUDE_SDK_RUN_TURNS=1. They do spend a small
// amount of tokens via the control protocol init (handshake only).

package tests

import (
	"strings"
	"testing"
)

// TestMCPServerStatus_EmptyConfig asserts that querying MCP status on a
// client with no MCP servers configured returns a valid (possibly empty)
// slice without error — i.e. the method successfully reaches the CLI and
// round-trips a valid wire message.
func TestMCPServerStatus_EmptyConfig(t *testing.T) {
	client, ctx := setupClient(t, nil)

	status, err := client.MCPServerStatus(ctx)
	if err != nil {
		t.Fatalf("MCPServerStatus: %v", err)
	}

	// A nil slice is permitted (no configured servers). What matters is
	// no error and — if populated — valid Name fields.
	t.Logf("MCPServerStatus returned %d server(s)", len(status))
	for _, s := range status {
		if s.Name == "" {
			t.Errorf("MCPServerStatus: server entry missing Name: %+v", s)
		}
	}
}

// TestSetMCPServers_EmptyMap configures an empty MCP server set via the
// control protocol and asserts the SDK parses the response envelope. This
// exercises the same request_id correlation path that feature 170 found
// broken on the daemon side.
func TestSetMCPServers_EmptyMap(t *testing.T) {
	client, ctx := setupClient(t, nil)

	result, err := client.SetMCPServers(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("SetMCPServers: %v", err)
	}

	if result == nil {
		t.Fatal("SetMCPServers: nil result; want non-nil with parsed wire envelope")
	}

	t.Logf("SetMCPServers result: %+v", result)
}

// TestSetMCPServers_Minimal sets a minimal stdio MCP server config and
// verifies the wire call round-trips. Persistence behavior (whether the
// server shows up in a subsequent MCPServerStatus call) varies by CLI
// version — older CLIs require the MCP config at startup via options
// rather than runtime SetMCPServers. This test asserts the wire contract
// only; persistence is a softer log-level observation.
func TestSetMCPServers_Minimal(t *testing.T) {
	client, ctx := setupClient(t, nil)

	servers := map[string]interface{}{
		"pass-through": map[string]interface{}{
			"type":    "stdio",
			"command": "/bin/true",
			"args":    []string{},
		},
	}

	result, err := client.SetMCPServers(ctx, servers)
	if err != nil {
		t.Fatalf("SetMCPServers: %v", err)
	}
	if result == nil {
		t.Fatal("SetMCPServers: nil result")
	}
	t.Logf("SetMCPServers result: %+v", result)

	// Best-effort: check whether the server is visible in status. Not
	// required — some CLI versions do not persist runtime-added servers.
	status, err := client.MCPServerStatus(ctx)
	if err != nil {
		t.Fatalf("MCPServerStatus after SetMCPServers: %v", err)
	}
	for _, s := range status {
		if s.Name == "pass-through" {
			t.Logf("pass-through server visible in status (CLI persists runtime config)")
			return
		}
	}
	t.Logf("pass-through server NOT visible in status — CLI may require startup-time MCP config")
}

// TestToggleMCPServer_Disabled exercises the toggle wire round-trip. If
// the runtime-seeded server persists in the CLI's state (varies by CLI
// version), the test asserts the Status field decoded correctly. If not,
// the test verifies the toggle call itself did not error.
func TestToggleMCPServer_Disabled(t *testing.T) {
	client, ctx := setupClient(t, nil)

	servers := map[string]interface{}{
		"toggle-target": map[string]interface{}{
			"type":    "stdio",
			"command": "/bin/true",
		},
	}
	if _, err := client.SetMCPServers(ctx, servers); err != nil {
		t.Fatalf("seed SetMCPServers: %v", err)
	}

	err := client.ToggleMCPServer(ctx, "toggle-target", false)
	// Some CLI versions return an error on toggle-of-unknown-server if
	// SetMCPServers didn't persist. Treat "not found" as a soft skip.
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			t.Skipf("ToggleMCPServer: CLI did not persist seeded server; runtime config limitation: %v", err)
		}
		t.Fatalf("ToggleMCPServer(false): %v", err)
	}

	status, err := client.MCPServerStatus(ctx)
	if err != nil {
		t.Fatalf("MCPServerStatus: %v", err)
	}

	for _, s := range status {
		if s.Name == "toggle-target" {
			if s.Status == "" {
				t.Errorf("ToggleMCPServer: server returned with empty Status field; wire decode likely drifted")
			}
			return
		}
	}
	t.Logf("ToggleMCPServer: toggle-target not in post-toggle status; CLI persistence varies by version")
}

// TestReconnectMCPServer_UnknownServer asserts that reconnecting a
// non-existent MCP server returns an error through the SDK — the CLI
// reports the error envelope and the SDK propagates it, rather than
// silently succeeding.
func TestReconnectMCPServer_UnknownServer(t *testing.T) {
	client, ctx := setupClient(t, nil)

	err := client.ReconnectMCPServer(ctx, "definitely-not-a-real-server-xyz")
	if err == nil {
		t.Error("ReconnectMCPServer(unknown): nil error; want error propagated from CLI")
	} else {
		t.Logf("ReconnectMCPServer(unknown) returned expected error: %v", err)
	}
}

// TestMCPServerStatusInfo_WireShape captures an MCPServerStatus response
// and asserts the McpServerStatusInfo struct decoded every field
// correctly. Probes any server the CLI returns — the user's global
// ~/.claude/mcp.json, runtime-added, or a seeded server. Skips if the
// CLI returns an empty status (no MCP servers available anywhere).
//
// Analogous to codex's probe_v040_shapes_test.go — catches field-name
// case drift between the CLI's JSON and the SDK's struct tags.
func TestMCPServerStatusInfo_WireShape(t *testing.T) {
	client, ctx := setupClient(t, nil)

	// Best-effort seed; tolerated if the CLI does not persist.
	_, _ = client.SetMCPServers(ctx, map[string]interface{}{
		"probe-target": map[string]interface{}{
			"type":    "stdio",
			"command": "/bin/true",
		},
	})

	status, err := client.MCPServerStatus(ctx)
	if err != nil {
		t.Fatalf("MCPServerStatus: %v", err)
	}

	if len(status) == 0 {
		t.Skip("no MCP servers available to probe on this system")
	}

	// Probe the first entry — whichever server the CLI advertises.
	probe := status[0]

	// Every entry MUST decode its Name field.
	if probe.Name == "" {
		t.Errorf("McpServerStatusInfo.Name: empty after decode; wire tag drift?")
	}
	// Status is the other mandatory field (non-empty string).
	if probe.Status == "" {
		t.Errorf("McpServerStatusInfo.Status: empty string; wire tag drift on mcp_types.go json tag")
	}
	t.Logf("Probed MCP server: %+v", probe)
}
