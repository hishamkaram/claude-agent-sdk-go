package claude

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func TestCapabilityDiscoveryErrorSanitizesAuthenticationFailure(t *testing.T) {
	t.Parallel()

	raw := types.NewCLIConnectionError("OAuth token has expired at /private/home")
	got := capabilityDiscoveryError(raw)
	var discoveryErr *types.CapabilityDiscoveryError
	if !errors.As(got, &discoveryErr) {
		t.Fatalf("error type = %T", got)
	}
	if discoveryErr.Code != types.CapabilityDiscoveryAuthRequired {
		t.Fatalf("Code = %q", discoveryErr.Code)
	}
	if got.Error() != "Claude authentication is required" {
		t.Fatalf("sanitized error = %q", got)
	}
}

func TestCapabilityDiscoveryErrorJSONExcludesCause(t *testing.T) {
	t.Parallel()

	discoveryErr := &types.CapabilityDiscoveryError{
		Code:    types.CapabilityDiscoveryProvider,
		Message: "Claude capability discovery failed",
		Cause:   errors.New("raw token at /private/home"),
	}
	encoded, err := json.Marshal(discoveryErr)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if strings.Contains(string(encoded), "token") || strings.Contains(string(encoded), "/private") {
		t.Fatalf("serialized discovery error leaked cause: %s", encoded)
	}
}

func TestCapabilityDiscoveryErrorClassifiesWrappedMissingCLI(t *testing.T) {
	t.Parallel()

	raw := types.NewCLIConnectionErrorWithCause("failed to start CLI", os.ErrNotExist)
	got := capabilityDiscoveryError(raw)
	var discoveryErr *types.CapabilityDiscoveryError
	if !errors.As(got, &discoveryErr) || discoveryErr.Code != types.CapabilityDiscoveryCLINotFound {
		t.Fatalf("error = %#v, want cli_not_found", got)
	}
}

func TestCapabilityDiscoveryErrorDoesNotTreatExecutablePathAsAuthFailure(t *testing.T) {
	t.Parallel()

	raw := types.NewCLIConnectionErrorWithCause(
		"failed to start /tmp/author-tools/claude",
		os.ErrPermission,
	)
	got := capabilityDiscoveryError(raw)
	var discoveryErr *types.CapabilityDiscoveryError
	if !errors.As(got, &discoveryErr) || discoveryErr.Code != types.CapabilityDiscoveryProvider {
		t.Fatalf("error = %#v, want provider_error", got)
	}
}

func TestCapabilityDiscoveryErrorClassifiesTimeout(t *testing.T) {
	t.Parallel()

	got := capabilityDiscoveryError(context.DeadlineExceeded)
	var discoveryErr *types.CapabilityDiscoveryError
	if !errors.As(got, &discoveryErr) || discoveryErr.Code != types.CapabilityDiscoveryTimeout {
		t.Fatalf("error = %#v", got)
	}
}

func TestUnsupportedCapabilityDiscoveryErrorIsNotRetryable(t *testing.T) {
	t.Parallel()

	got := unsupportedCapabilityDiscoveryError("unsupported", errors.New("provider detail"))
	if got.Code != types.CapabilityDiscoveryUnsupportedCLI {
		t.Fatalf("Code = %q, want unsupported_cli", got.Code)
	}
	if got.Retryable {
		t.Fatal("Retryable = true, want false for unsupported CLI")
	}
}
