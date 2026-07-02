package claude

import (
	"context"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// TestGetContextUsage_BeforeConnect ensures GetContextUsage returns CLIConnectionError when not connected.
func TestGetContextUsage_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	_, err = client.GetContextUsage(ctx)
	if err == nil {
		t.Fatal("expected error when calling GetContextUsage before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %v", err)
	}
}

// TestGetSettings_BeforeConnect ensures GetSettings returns CLIConnectionError when not connected.
func TestGetSettings_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	_, err = client.GetSettings(ctx)
	if err == nil {
		t.Fatal("expected error when calling GetSettings before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %v", err)
	}
}

func TestSetEffort_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.SetEffort(ctx, types.EffortHigh)
	if err == nil {
		t.Fatal("expected error when calling SetEffort before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %v", err)
	}
}

func TestSetUltracode_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.SetUltracode(ctx, true)
	if err == nil {
		t.Fatal("expected error when calling SetUltracode before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %v", err)
	}
}

// TestReloadPlugins_BeforeConnect ensures ReloadPlugins returns CLIConnectionError when not connected.
func TestReloadPlugins_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.ReloadPlugins(ctx)
	if err == nil {
		t.Fatal("expected error when calling ReloadPlugins before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %v", err)
	}
}

// TestEnableChannel_BeforeConnect ensures EnableChannel returns CLIConnectionError when not connected.
func TestEnableChannel_BeforeConnect(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() { _ = client.Close(ctx) }()

	err = client.EnableChannel(ctx)
	if err == nil {
		t.Fatal("expected error when calling EnableChannel before Connect()")
	}
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %v", err)
	}
}

// BenchmarkClient benchmarks the Client type
func BenchmarkClient_Create(b *testing.B) {
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, err := NewClient(ctx, opts)
		if err == nil {
			_ = client.Close(ctx)
		}
	}
}
