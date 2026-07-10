//go:build integration

package tests

import (
	"context"
	"testing"

	claude "github.com/hishamkaram/claude-agent-sdk-go"
)

func TestIntegration_PermissionModesComeFromInstalledCLI(t *testing.T) {
	modes := claude.DiscoverSupportedPermissionModes(context.Background(), nil)
	if len(modes) == 0 {
		t.Fatal("installed Claude CLI advertised no permission modes")
	}
	for _, mode := range modes {
		if mode.ProviderValue == "" || mode.Version == "" || mode.Source == "" {
			t.Fatalf("incomplete discovered permission mode: %+v", mode)
		}
	}
}
