package claude

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	if len(got) != 0 {
		t.Fatalf("SupportedPermissionModes() = %v, want empty unavailable result", got)
	}
}

func TestProviderPermissionModeUsesInstalledCLISpelling(t *testing.T) {
	t.Parallel()

	modes := []types.SupportedPermissionMode{{
		ProviderValue: "manual", CanonicalValue: types.PermissionModeDefault,
	}}
	got, err := providerPermissionMode(modes, types.PermissionModeDefault)
	if err != nil {
		t.Fatalf("providerPermissionMode() error = %v", err)
	}
	if got != "manual" {
		t.Fatalf("providerPermissionMode() = %q, want manual", got)
	}
}

func TestProviderPermissionModeRejectsUnadvertisedMode(t *testing.T) {
	t.Parallel()

	_, err := providerPermissionMode(nil, types.PermissionModeAuto)
	if err == nil {
		t.Fatal("providerPermissionMode() error = nil, want unsupported error")
	}
}

func TestProviderPermissionModeRejectsEmptyModeBeforeFutureAliasMatch(t *testing.T) {
	t.Parallel()

	modes := []types.SupportedPermissionMode{{ProviderValue: "futureMode"}}
	_, err := providerPermissionMode(modes, types.PermissionMode(""))
	if err == nil {
		t.Fatal("providerPermissionMode() error = nil, want empty-mode rejection")
	}
}

func TestProviderPermissionModeRoundTripsFutureProviderMode(t *testing.T) {
	t.Parallel()

	modes := []types.SupportedPermissionMode{{ProviderValue: "futureMode"}}
	got, err := providerPermissionMode(modes, types.PermissionMode("futureMode"))
	if err != nil {
		t.Fatalf("providerPermissionMode() error = %v", err)
	}
	if got != "futureMode" {
		t.Fatalf("providerPermissionMode() = %q, want futureMode", got)
	}
}

func TestDiscoverSupportedPermissionModesUsesCustomSpawner(t *testing.T) {
	t.Parallel()

	var calls []types.SpawnOptions
	spawner := func(_ context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
		calls = append(calls, opts)
		switch {
		case len(opts.Args) == 1 && opts.Args[0] == "--version":
			return newStaticCLIProcess("2.9.1 (Claude Code)\n", ""), nil
		case len(opts.Args) == 1 && opts.Args[0] == "--help":
			return newStaticCLIProcess(
				"--permission-mode <mode> (choices: \"manual\", \"auto\", \"futureMode\")\n--plugin-dir <path>\n",
				"",
			), nil
		default:
			return nil, errors.New("unexpected custom-spawner command")
		}
	}
	options := types.NewClaudeAgentOptions().
		WithCLIPath("/remote/bin/claude").
		WithSpawnProcess(spawner)

	modes := DiscoverSupportedPermissionModes(context.Background(), options)
	if len(modes) != 3 {
		t.Fatalf("DiscoverSupportedPermissionModes() = %+v, want three remote modes", modes)
	}
	if modes[0].ProviderValue != "manual" || modes[0].Version != "2.9.1" {
		t.Fatalf("first mode = %+v, want remote manual mode with version 2.9.1", modes[0])
	}
	if modes[2].ProviderValue != "futureMode" || modes[2].CanonicalValue != "" {
		t.Fatalf("future mode = %+v, want provider-only futureMode", modes[2])
	}
	if len(calls) != 2 {
		t.Fatalf("custom spawner calls = %d, want version and help probes", len(calls))
	}
	for _, call := range calls {
		if call.Command != "/remote/bin/claude" {
			t.Fatalf("custom spawner command = %q, want remote CLI path", call.Command)
		}
		if call.Env["CLAUDE_CODE_ENTRYPOINT"] != "agent" {
			t.Fatalf("custom spawner probe environment = %+v, want SDK runtime markers", call.Env)
		}
	}
}

func TestClientPermissionModesProbeVersionMissingFromInitMetadata(t *testing.T) {
	t.Parallel()

	var calls []types.SpawnOptions
	spawner := func(_ context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
		calls = append(calls, opts)
		switch opts.Args[0] {
		case "--version":
			return newStaticCLIProcess("2.9.1 (Claude Code)\n", ""), nil
		case "--help":
			return newStaticCLIProcess("--permission-mode <mode> (choices: \"manual\")\n", ""), nil
		default:
			return nil, errors.New("unexpected custom-spawner command")
		}
	}
	options := types.NewClaudeAgentOptions().
		WithCLIPath("/remote/bin/claude").
		WithSpawnProcess(spawner)
	client := &Client{
		cliPath: "/remote/bin/claude",
		ctx:     context.Background(),
		options: options,
		initResult: &types.InitializeResult{Raw: map[string]interface{}{
			"capabilities": map[string]interface{}{
				"permissionModes": []interface{}{"futureMode"},
			},
		}},
	}

	modes, version, err := client.supportedPermissionModes(context.Background())
	if err != nil {
		t.Fatalf("supportedPermissionModes() error = %v", err)
	}
	if len(modes) != 1 || modes[0].ProviderValue != "futureMode" || modes[0].Source != types.PermissionModeSourceInit {
		t.Fatalf("supportedPermissionModes() = %+v, want init-provided futureMode", modes)
	}
	if version != "2.9.1" {
		t.Fatalf("supportedPermissionModes() version = %q, want probed 2.9.1", version)
	}
	if len(calls) != 1 || calls[0].Args[0] != "--version" {
		t.Fatalf("custom spawner calls = %+v, want version probe only", calls)
	}
}

func TestSetPermissionModeUsesHelpModesWhenVersionProbeFails(t *testing.T) {
	t.Parallel()

	spawner := func(_ context.Context, opts types.SpawnOptions) (types.SpawnedProcess, error) {
		switch opts.Args[0] {
		case "--version":
			return newStaticCLIProcess("version unavailable\n", ""), nil
		case "--help":
			return newStaticCLIProcess(
				"--permission-mode <mode> (choices: \"manual\", \"auto\")\n",
				"",
			), nil
		default:
			return nil, errors.New("unexpected custom-spawner command")
		}
	}
	options := types.NewClaudeAgentOptions().
		WithCLIPath("/remote/bin/claude").
		WithSpawnProcess(spawner)
	client, controlTransport := newRuntimeSettingsClientWithOptions(t, options)
	client.cliPath = "/remote/bin/claude"
	client.ctx = context.Background()
	client.options = options

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.SetPermissionMode(context.Background(), types.PermissionModeDefault)
	}()

	request := waitForControlRequest(t, controlTransport, 0)
	payload := controlRequestPayload(t, request)
	if payload["subtype"] != "set_permission_mode" || payload["mode"] != "manual" {
		t.Fatalf("permission request = %#v, want provider mode manual", payload)
	}
	controlTransport.sendControlResponse(request, nil)
	if err := <-errCh; err != nil {
		t.Fatalf("SetPermissionMode() error = %v, want help-derived mode despite version failure", err)
	}

	modes := client.SupportedPermissionModes()
	if len(modes) != 2 || modes[0].ProviderValue != "manual" || modes[0].Version != "" {
		t.Fatalf("SupportedPermissionModes() = %+v, want cached unversioned help modes", modes)
	}
}

func TestDiscoverSupportedPermissionModesUsesConfiguredEnvironment(t *testing.T) {
	t.Setenv("SPECIAL_PERMISSION_HELP", "0")
	script := filepath.Join(t.TempDir(), "claude")
	contents := `#!/bin/sh
if [ "$1" = "--version" ]; then
  echo '2.9.1 (Claude Code)'
elif [ "$SPECIAL_PERMISSION_HELP" = "1" ]; then
  echo '--permission-mode <mode> (choices: "manual", "auto")'
else
  echo '--permission-mode <mode> (choices: "plan")'
fi
`
	if err := os.WriteFile(script, []byte(contents), 0o700); err != nil {
		t.Fatalf("write fake CLI: %v", err)
	}
	options := types.NewClaudeAgentOptions().
		WithCLIPath(script).
		WithEnvVar("SPECIAL_PERMISSION_HELP", "1")

	modes := DiscoverSupportedPermissionModes(context.Background(), options)
	if len(modes) != 2 || modes[1].ProviderValue != "auto" {
		t.Fatalf("DiscoverSupportedPermissionModes() = %+v, want configured-environment modes", modes)
	}

	withoutEnv := types.NewClaudeAgentOptions().WithCLIPath(script)
	modes = DiscoverSupportedPermissionModes(context.Background(), withoutEnv)
	if len(modes) != 1 || modes[0].ProviderValue != "plan" {
		t.Fatalf("DiscoverSupportedPermissionModes() cached across environments: %+v", modes)
	}
}

func TestSetPermissionModeHonorsCallerContextDuringDiscovery(t *testing.T) {
	script := filepath.Join(t.TempDir(), "claude")
	contents := "#!/bin/sh\nsleep 0.3\nif [ \"$1\" = \"--version\" ]; then echo '2.9.1 (Claude Code)'; else echo '--permission-mode <mode> (choices: \"manual\", \"auto\")'; fi\n"
	if err := os.WriteFile(script, []byte(contents), 0o700); err != nil {
		t.Fatalf("write fake CLI: %v", err)
	}

	client, _ := newRuntimeSettingsClient(t)
	client.cliPath = script
	client.options = types.NewClaudeAgentOptions().WithCLIPath(script)
	client.ctx = context.Background()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	started := time.Now()
	err := client.SetPermissionMode(ctx, types.PermissionModeAuto)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("SetPermissionMode() error = %v, want caller deadline", err)
	}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("SetPermissionMode() took %v after caller deadline", elapsed)
	}
}

type staticCLIProcess struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func newStaticCLIProcess(stdout, stderr string) *staticCLIProcess {
	return &staticCLIProcess{
		stdin:  nopWriteCloser{Writer: io.Discard},
		stdout: io.NopCloser(bytes.NewBufferString(stdout)),
		stderr: io.NopCloser(bytes.NewBufferString(stderr)),
	}
}

func (p *staticCLIProcess) Stdin() io.WriteCloser { return p.stdin }
func (p *staticCLIProcess) Stdout() io.ReadCloser { return p.stdout }
func (p *staticCLIProcess) Stderr() io.ReadCloser { return p.stderr }
func (p *staticCLIProcess) Kill() error           { return nil }
func (p *staticCLIProcess) Wait() error           { return nil }
func (p *staticCLIProcess) ExitCode() int         { return 0 }
func (p *staticCLIProcess) Killed() bool          { return false }

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }
