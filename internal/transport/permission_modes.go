package transport

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

type permissionModeCacheKey struct {
	cliPath string
	version string
	runtime string
}

var permissionModeDiscoveryCache struct {
	mu    sync.Mutex
	modes map[permissionModeCacheKey][]types.SupportedPermissionMode
}

// DiscoverPermissionModes returns the installed Claude CLI permission-mode
// choices. An empty result means discovery was unavailable.
func DiscoverPermissionModes(ctx context.Context, cliPath string) []types.SupportedPermissionMode {
	modes, _ := DiscoverPermissionModesAndVersion(ctx, cliPath)
	return modes
}

// DiscoverPermissionModesAndVersion returns permission modes and the version
// reported by the same CLI process boundary.
func DiscoverPermissionModesAndVersion(
	ctx context.Context,
	cliPath string,
) (modes []types.SupportedPermissionMode, version string) {
	modes, version, _ = DiscoverPermissionModesAndVersionWithEnvironment(ctx, cliPath, "", nil)
	return modes, version
}

// DiscoverPermissionModesAndVersionWithEnvironment probes the same native CLI
// environment and working directory used by the session runtime.
func DiscoverPermissionModesAndVersionWithEnvironment(
	ctx context.Context,
	cliPath string,
	cwd string,
	env map[string]string,
) (modes []types.SupportedPermissionMode, version string, err error) {
	versionString, versionErr := DiscoverCLIVersionWithEnvironment(ctx, cliPath, cwd, env)
	if versionErr == nil {
		key := permissionModeCacheKey{
			cliPath: cliPath,
			version: versionString,
			runtime: cliProbeCacheIdentity(cwd, env),
		}
		if cached := cachedPermissionModes(key); len(cached) > 0 {
			return cached, versionString, nil
		}
		discovered, discoveryErr := discoverPermissionModesUncached(ctx, cliPath, cwd, env, versionString)
		if discoveryErr != nil {
			return nil, versionString, discoveryErr
		}
		if shouldCachePermissionModes(discovered) {
			storePermissionModes(key, discovered)
		}
		return cloneSupportedPermissionModes(discovered), versionString, nil
	}

	discovered, discoveryErr := discoverPermissionModesUncached(ctx, cliPath, cwd, env, "")
	if discoveryErr != nil {
		return nil, "", discoveryErr
	}
	return discovered, "", versionErr
}

// DiscoverCLIVersionWithEnvironment probes the installed CLI version using the
// same working directory and effective environment as a native session.
func DiscoverCLIVersionWithEnvironment(
	ctx context.Context,
	cliPath string,
	cwd string,
	env map[string]string,
) (string, error) {
	version, err := getCLIVersionForProbe(ctx, cliPath, cwd, env)
	if err != nil {
		return "", err
	}
	return version.String(), nil
}

// ParsePermissionModesFromHelp extracts quoted --permission-mode choices from
// the CLI help text. It intentionally only reads the --permission-mode stanza.
func ParsePermissionModesFromHelp(help string) []string {
	lines := strings.Split(help, "\n")
	collecting := false
	var stanza strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(line, "--permission-mode") {
			collecting = true
		} else if collecting && strings.HasPrefix(trimmed, "--") {
			break
		}
		if collecting {
			stanza.WriteString(line)
			stanza.WriteByte('\n')
		}
	}
	if !collecting {
		return nil
	}

	quoted := regexp.MustCompile(`"([^"]+)"`)
	matches := quoted.FindAllStringSubmatch(stanza.String(), -1)
	values := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) < 2 || match[1] == "" {
			continue
		}
		if _, ok := seen[match[1]]; ok {
			continue
		}
		seen[match[1]] = struct{}{}
		values = append(values, match[1])
	}
	return values
}

func discoverPermissionModesUncached(
	ctx context.Context,
	cliPath string,
	cwd string,
	env map[string]string,
	version string,
) ([]types.SupportedPermissionMode, error) {
	helpCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(helpCtx, cliPath, "--help")
	cmd.WaitDelay = 100 * time.Millisecond
	configureNativeCLIProbe(cmd, cwd, env)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctxErr := helpCtx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("claude CLI permission-mode help probe: %w", ctxErr)
		}
		return nil, fmt.Errorf("claude CLI permission-mode help probe: %w", err)
	}

	return permissionModesFromHelp(stdout.String()+"\n"+stderr.String(), version), nil
}

func getCLIVersionForProbe(
	ctx context.Context,
	cliPath string,
	cwd string,
	env map[string]string,
) (SemanticVersion, error) {
	versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(versionCtx, cliPath, "--version")
	cmd.WaitDelay = 100 * time.Millisecond
	configureNativeCLIProbe(cmd, cwd, env)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		if ctxErr := versionCtx.Err(); ctxErr != nil {
			return SemanticVersion{}, fmt.Errorf("claude CLI version probe: %w", ctxErr)
		}
		return SemanticVersion{}, fmt.Errorf("claude CLI version probe: %w", err)
	}
	version, err := ParseSemanticVersion(strings.TrimSpace(stdout.String()))
	if err != nil {
		return SemanticVersion{}, fmt.Errorf("claude CLI version probe: %w", err)
	}
	return version, nil
}

func configureNativeCLIProbe(cmd *exec.Cmd, cwd string, env map[string]string) {
	if cwd != "" {
		cmd.Dir = cwd
	}
	cmd.Env = buildEffectiveProcessEnvironment(env)
}

func cliProbeCacheIdentity(cwd string, env map[string]string) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(effectiveWorkingDirectory(cwd)))
	for _, entry := range buildEffectiveProcessEnvironment(env) {
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(entry))
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func buildEffectiveProcessEnvironment(overrides map[string]string) []string {
	return buildEffectiveProcessEnvironmentForOS(os.Environ(), overrides, runtime.GOOS)
}

type processEnvironmentValue struct {
	key   string
	value string
}

func buildEffectiveProcessEnvironmentForOS(
	inherited []string,
	overrides map[string]string,
	goos string,
) []string {
	values := make(map[string]processEnvironmentValue, len(inherited)+len(overrides))
	for _, entry := range inherited {
		key, value, ok := splitProcessEnvironmentEntry(entry)
		if !ok {
			continue
		}
		values[processEnvironmentKey(key, goos)] = processEnvironmentValue{key: key, value: value}
	}

	overrideKeys := make([]string, 0, len(overrides))
	for key := range overrides {
		overrideKeys = append(overrideKeys, key)
	}
	sort.Strings(overrideKeys)
	for _, key := range overrideKeys {
		values[processEnvironmentKey(key, goos)] = processEnvironmentValue{key: key, value: overrides[key]}
	}

	result := make([]string, 0, len(values))
	for _, entry := range values {
		result = append(result, entry.key+"="+entry.value)
	}
	sort.Strings(result)
	return result
}

func splitProcessEnvironmentEntry(entry string) (key, value string, ok bool) {
	separator := strings.IndexByte(entry, '=')
	if separator == 0 {
		nextSeparator := strings.IndexByte(entry[1:], '=')
		if nextSeparator >= 0 {
			separator = nextSeparator + 1
		}
	}
	if separator < 0 {
		return "", "", false
	}
	return entry[:separator], entry[separator+1:], true
}

func processEnvironmentKey(key, goos string) string {
	if goos == "windows" {
		return strings.ToUpper(key)
	}
	return key
}

func effectiveWorkingDirectory(cwd string) string {
	if cwd == "" {
		cwd = "."
	}
	absolute, err := filepath.Abs(cwd)
	if err != nil {
		return filepath.Clean(cwd)
	}
	return filepath.Clean(absolute)
}

func permissionModesFromHelp(help, version string) []types.SupportedPermissionMode {
	values := ParsePermissionModesFromHelp(help)
	if len(values) == 0 {
		return nil
	}
	modes := make([]types.SupportedPermissionMode, 0, len(values))
	for _, value := range values {
		modes = append(modes, types.SupportedPermissionMode{
			ProviderValue:  value,
			CanonicalValue: canonicalPermissionMode(value),
			Source:         types.PermissionModeSourceCLIHelp,
			Version:        version,
		})
	}
	return modes
}

func canonicalPermissionMode(value string) types.PermissionMode {
	switch value {
	case "manual", string(types.PermissionModeDefault):
		return types.PermissionModeDefault
	case string(types.PermissionModeAcceptEdits):
		return types.PermissionModeAcceptEdits
	case string(types.PermissionModeAuto):
		return types.PermissionModeAuto
	case string(types.PermissionModePlan):
		return types.PermissionModePlan
	case string(types.PermissionModeBypassPermissions):
		return types.PermissionModeBypassPermissions
	case string(types.PermissionModeDontAsk):
		return types.PermissionModeDontAsk
	default:
		return ""
	}
}

func cachedPermissionModes(key permissionModeCacheKey) []types.SupportedPermissionMode {
	permissionModeDiscoveryCache.mu.Lock()
	defer permissionModeDiscoveryCache.mu.Unlock()
	if permissionModeDiscoveryCache.modes == nil {
		return nil
	}
	return cloneSupportedPermissionModes(permissionModeDiscoveryCache.modes[key])
}

func storePermissionModes(key permissionModeCacheKey, modes []types.SupportedPermissionMode) {
	permissionModeDiscoveryCache.mu.Lock()
	defer permissionModeDiscoveryCache.mu.Unlock()
	if permissionModeDiscoveryCache.modes == nil {
		permissionModeDiscoveryCache.modes = make(map[permissionModeCacheKey][]types.SupportedPermissionMode)
	}
	permissionModeDiscoveryCache.modes[key] = cloneSupportedPermissionModes(modes)
}

func shouldCachePermissionModes(modes []types.SupportedPermissionMode) bool {
	return len(modes) != 0
}

func cloneSupportedPermissionModes(modes []types.SupportedPermissionMode) []types.SupportedPermissionMode {
	if len(modes) == 0 {
		return nil
	}
	out := make([]types.SupportedPermissionMode, len(modes))
	copy(out, modes)
	return out
}
