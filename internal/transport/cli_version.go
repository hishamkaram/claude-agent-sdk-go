package transport

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

const (
	// MinimumCLIVersion is the minimum required CLI version (2.0.0)
	MinimumCLIVersion = "2.0.0"

	// MinimumCLIMajor is the minimum major version
	MinimumCLIMajor = 2

	// MinimumCLIMinor is the minimum minor version
	MinimumCLIMinor = 0

	// MinimumCLIPatch is the minimum patch version
	MinimumCLIPatch = 0

	// MinimumThinkingDisplayCLIVersion is the first validated Claude Code CLI
	// version that accepts --thinking-display during --print execution.
	MinimumThinkingDisplayCLIVersion = "2.1.94"

	// MinimumThinkingDisplayCLIMajor is the minimum major version for --thinking-display.
	MinimumThinkingDisplayCLIMajor = 2

	// MinimumThinkingDisplayCLIMinor is the minimum minor version for --thinking-display.
	MinimumThinkingDisplayCLIMinor = 1

	// MinimumThinkingDisplayCLIPatch is the minimum patch version for --thinking-display.
	MinimumThinkingDisplayCLIPatch = 94
)

// SemanticVersion represents a semantic version number (major.minor.patch)
type SemanticVersion struct {
	Major int
	Minor int
	Patch int
}

// String returns the string representation of the version
func (v SemanticVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// IsAtLeast checks if this version is at least the specified version
func (v SemanticVersion) IsAtLeast(required SemanticVersion) bool {
	if v.Major > required.Major {
		return true
	}
	if v.Major < required.Major {
		return false
	}

	// Major versions are equal, check minor
	if v.Minor > required.Minor {
		return true
	}
	if v.Minor < required.Minor {
		return false
	}

	// Major and minor are equal, check patch
	return v.Patch >= required.Patch
}

// ParseSemanticVersion parses a semantic version string (e.g., "2.1.0")
func ParseSemanticVersion(versionStr string) (SemanticVersion, error) {
	// Clean the version string (remove leading 'v' if present)
	versionStr = strings.TrimSpace(versionStr)
	versionStr = strings.TrimPrefix(versionStr, "v")

	// Use regex to extract version numbers
	// Pattern: major.minor.patch with optional pre-release/metadata
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)

	if len(matches) != 4 {
		return SemanticVersion{}, fmt.Errorf("invalid version format: %s", versionStr)
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return SemanticVersion{}, fmt.Errorf("invalid major version: %s", matches[1])
	}

	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return SemanticVersion{}, fmt.Errorf("invalid minor version: %s", matches[2])
	}

	patch, err := strconv.Atoi(matches[3])
	if err != nil {
		return SemanticVersion{}, fmt.Errorf("invalid patch version: %s", matches[3])
	}

	return SemanticVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

// GetCLIVersion retrieves the version of the Claude CLI binary. The caller's
// ctx bounds the `claude --version` subprocess; a 5s timeout is layered on top
// so a hung CLI cannot block indefinitely even when ctx has no deadline.
func GetCLIVersion(ctx context.Context, cliPath string) (SemanticVersion, error) {
	// Layer a hard timeout on the caller's context to prevent hanging.
	versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Run: claude --version
	cmd := exec.CommandContext(versionCtx, cliPath, "--version")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return SemanticVersion{}, fmt.Errorf("failed to get CLI version: %w (stderr: %s)", err, stderr.String())
	}

	// Parse version from output
	versionStr := strings.TrimSpace(stdout.String())
	return ParseSemanticVersion(versionStr)
}

// tryGetCLIVersion returns the parsed CLI version and true when it can be
// determined. When the version cannot be determined (older CLIs without
// `--version`, parse failures, timeouts), it returns false so callers can apply
// a backwards-compatible "accept" policy without inspecting the discovery error.
func tryGetCLIVersion(ctx context.Context, cliPath string) (SemanticVersion, bool) {
	version, err := GetCLIVersion(ctx, cliPath)
	if err != nil {
		return SemanticVersion{}, false
	}
	return version, true
}

// SupportsThinkingDisplay reports whether the parsed Claude CLI version accepts
// --thinking-display. Older supported CLIs continue to work without the flag;
// callers just receive the CLI's default thinking display behavior.
func SupportsThinkingDisplay(version SemanticVersion) bool {
	return version.IsAtLeast(SemanticVersion{
		Major: MinimumThinkingDisplayCLIMajor,
		Minor: MinimumThinkingDisplayCLIMinor,
		Patch: MinimumThinkingDisplayCLIPatch,
	})
}

// SupportsAgentProgressSummaries reports whether the CLI version accepts the
// --agent-progress-summaries flag. No released version supports it: the flag string
// is absent from the 2.1.158 CLI binary (verified by inspection), and emitting it
// crashes Connect with "unknown option". This is the single authoritative gate —
// when the flag ships, replace the constant false with a version.IsAtLeast(...)
// check against its real minimum version.
func SupportsAgentProgressSummaries(_ SemanticVersion) bool {
	return false
}

// SupportsSubagentExecution reports whether the CLI version accepts the
// --subagent-execution flag. Absent from every released version through 2.1.158
// (verified by binary inspection); returns false until the flag ships and this gate
// is updated with its real minimum version.
func SupportsSubagentExecution(_ SemanticVersion) bool {
	return false
}

// CheckCLIVersion verifies that the CLI version meets minimum requirements.
// Returns nil if version is acceptable, or an error if not. The caller's ctx
// bounds the version-probe subprocess.
func CheckCLIVersion(ctx context.Context, cliPath string) error {
	// Check if version checking is disabled via environment variable
	if os.Getenv("CLAUDE_AGENT_SDK_SKIP_VERSION_CHECK") != "" {
		return nil
	}

	// Get the CLI version. If we can't determine it, accept the CLI rather than
	// fail — older CLIs that predate `--version` must keep working (backwards
	// compatibility). The discovery error is intentionally non-fatal here.
	version, ok := tryGetCLIVersion(ctx, cliPath)
	if !ok {
		return nil
	}

	// Check minimum version requirement
	minVersion := SemanticVersion{
		Major: MinimumCLIMajor,
		Minor: MinimumCLIMinor,
		Patch: MinimumCLIPatch,
	}

	if !version.IsAtLeast(minVersion) {
		return types.NewCLINotFoundError(fmt.Sprintf(
			"Claude CLI version %s is installed, but version %s or higher is required.\n"+
				"Please update with:\n"+
				"  npm install -g @anthropic-ai/claude-code@latest\n"+
				"\nTo skip this check, set:\n"+
				"  export CLAUDE_AGENT_SDK_SKIP_VERSION_CHECK=1",
			version.String(),
			minVersion.String(),
		))
	}

	return nil
}
