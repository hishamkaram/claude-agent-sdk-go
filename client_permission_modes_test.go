package claude

import (
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestSupportedPermissionModesFromInitMetadata(t *testing.T) {
	t.Parallel()

	init := &types.InitializeResult{
		Raw: map[string]interface{}{
			"version": "2.2.0",
			"capabilities": map[string]interface{}{
				"permissionModes": []interface{}{
					"default",
					map[string]interface{}{"value": "auto"},
					map[string]interface{}{"providerValue": "futureMode"},
				},
			},
		},
	}

	got := supportedPermissionModesFromInit(init)
	want := []string{"default", "auto", "futureMode"}
	if len(got) != len(want) {
		t.Fatalf("supportedPermissionModesFromInit() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i].ProviderValue != want[i] {
			t.Fatalf("supportedPermissionModesFromInit()[%d].ProviderValue = %q, want %q", i, got[i].ProviderValue, want[i])
		}
		if got[i].Source != types.PermissionModeSourceInit {
			t.Fatalf("supportedPermissionModesFromInit()[%d].Source = %q, want %q", i, got[i].Source, types.PermissionModeSourceInit)
		}
		if got[i].Version != "2.2.0" {
			t.Fatalf("supportedPermissionModesFromInit()[%d].Version = %q, want 2.2.0", i, got[i].Version)
		}
	}
}

func TestClientSupportedPermissionModesBeforeConnectUsesCLIDiscovery(t *testing.T) {
	t.Parallel()

	client := &Client{
		cliPath: "/path/that/does/not/exist",
	}

	got := client.SupportedPermissionModes()
	if len(got) == 0 {
		t.Fatal("SupportedPermissionModes() returned empty fallback before connect")
	}
	for _, mode := range got {
		if mode.Source != types.PermissionModeSourceFallback {
			t.Fatalf("SupportedPermissionModes() source = %q, want fallback for missing CLI", mode.Source)
		}
	}
}
