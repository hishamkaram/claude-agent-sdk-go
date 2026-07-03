package transport

import (
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

type permissionModeCacheKey struct {
	cliPath string
	version string
}

var permissionModeDiscoveryCache struct {
	mu    sync.Mutex
	modes map[permissionModeCacheKey][]types.SupportedPermissionMode
}

// DiscoverPermissionModes returns the installed Claude CLI permission-mode
// choices, falling back to conservative modes when help/version discovery fails.
func DiscoverPermissionModes(ctx context.Context, cliPath string) []types.SupportedPermissionMode {
	versionString := ""
	if version, ok := tryGetCLIVersion(ctx, cliPath); ok {
		versionString = version.String()
		key := permissionModeCacheKey{cliPath: cliPath, version: versionString}
		if cached := cachedPermissionModes(key); len(cached) > 0 {
			return cached
		}
		discovered := discoverPermissionModesUncached(ctx, cliPath, versionString)
		if shouldCachePermissionModes(discovered) {
			storePermissionModes(key, discovered)
		}
		return cloneSupportedPermissionModes(discovered)
	}

	return discoverPermissionModesUncached(ctx, cliPath, versionString)
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

// FallbackPermissionModes returns the safe static fallback set used when the
// installed CLI cannot be inspected.
func FallbackPermissionModes(version string) []types.SupportedPermissionMode {
	values := []string{
		string(types.PermissionModeDefault),
		string(types.PermissionModePlan),
		string(types.PermissionModeAcceptEdits),
	}
	modes := make([]types.SupportedPermissionMode, 0, len(values))
	for _, value := range values {
		modes = append(modes, types.SupportedPermissionMode{
			ProviderValue: value,
			Source:        types.PermissionModeSourceFallback,
			Version:       version,
		})
	}
	return modes
}

func discoverPermissionModesUncached(ctx context.Context, cliPath, version string) []types.SupportedPermissionMode {
	helpCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(helpCtx, cliPath, "--help")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return FallbackPermissionModes(version)
	}

	values := ParsePermissionModesFromHelp(stdout.String() + "\n" + stderr.String())
	if len(values) == 0 {
		return FallbackPermissionModes(version)
	}
	modes := make([]types.SupportedPermissionMode, 0, len(values))
	for _, value := range values {
		modes = append(modes, types.SupportedPermissionMode{
			ProviderValue: value,
			Source:        types.PermissionModeSourceCLIHelp,
			Version:       version,
		})
	}
	return modes
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
	if len(modes) == 0 {
		return false
	}
	for _, mode := range modes {
		if mode.Source == types.PermissionModeSourceFallback {
			return false
		}
	}
	return true
}

func cloneSupportedPermissionModes(modes []types.SupportedPermissionMode) []types.SupportedPermissionMode {
	if len(modes) == 0 {
		return nil
	}
	out := make([]types.SupportedPermissionMode, len(modes))
	copy(out, modes)
	return out
}
