//go:build integration
// +build integration

// Real-CLI coverage for the sessions.go public API. These tests enumerate
// session state on disk and exercise the rename/tag/fork/list-subagents
// flow. They do NOT burn model tokens — every call is a CLI subprocess
// invocation over structured arguments.
//
// Functions covered (sessions.go line refs):
//
//     ListSessions           (:16)
//     GetSessionMessages     (:57)
//     GetSessionInfo         (:103)
//     RenameSession          (:138)
//     TagSession             (:168)
//     ForkSession            (:200)
//     ListSubagents          (:235)
//     GetSubagentMessages    (:278)

package tests

import (
	"context"
	"testing"
	"time"

	claude "github.com/hishamkaram/claude-agent-sdk-go"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// sessionCtx builds a CLI-bound context with a 30s deadline and an auth
// gate. The sessions.go functions are package-level — they do not need a
// connected Client. They DO need the CLI on PATH.
func sessionCtx(t *testing.T) context.Context {
	t.Helper()
	requireAuth(t)
	_ = requireClaude(t)

	ctx, cancel := CreateTestContext(t, 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestSessions_List(t *testing.T) {
	ctx := sessionCtx(t)

	// Dir not set → CLI falls back to its default. Limit caps the result
	// to keep the test bounded on developers with large session history.
	opts := &types.ListSessionsOptions{Limit: 5}

	sessions, err := claude.ListSessions(ctx, opts)
	if err != nil {
		// Some environments have no sessions directory or an
		// inaccessible one — the CLI returns exit 1. Treat as a skip
		// rather than a hard fail; wire-shape coverage happens when
		// sessions actually exist.
		t.Skipf("ListSessions: skipping — CLI error (no sessions dir?): %v", err)
	}

	t.Logf("ListSessions returned %d session(s)", len(sessions))

	// Wire-shape probe: each entry's SessionID must be a non-empty string.
	// Catches camelCase vs snake_case drift on the sessionId field.
	for i, s := range sessions {
		if s.SessionID == "" {
			t.Errorf("ListSessions[%d]: SessionID empty; wire tag drift on sessionId?", i)
		}
	}
}

func TestSessions_GetMessages_InvalidID(t *testing.T) {
	ctx := sessionCtx(t)

	_, err := claude.GetSessionMessages(ctx, "definitely-not-a-real-session-id", nil)
	if err == nil {
		t.Error("GetSessionMessages(bogus id): nil error; want error propagated from CLI")
	} else {
		t.Logf("GetSessionMessages(bogus id) returned expected error: %v", err)
	}
}

// Note: the Claude CLI is lenient on unknown session IDs — it accepts
// them without error for GetSessionInfo/Rename/Tag/Fork. These tests
// therefore cover the wire-call path only, not the negative path.
// Happy-path coverage against a real session ID is a follow-up.

func TestSessions_GetInfo_InvalidID(t *testing.T) {
	ctx := sessionCtx(t)

	info, err := claude.GetSessionInfo(ctx, "definitely-not-a-real-session-id", nil)
	if err != nil {
		t.Logf("GetSessionInfo(bogus id) returned error: %v", err)
		return
	}
	t.Logf("GetSessionInfo(bogus id) returned info: %+v", info)
}

func TestSessions_Rename_InvalidID(t *testing.T) {
	ctx := sessionCtx(t)

	err := claude.RenameSession(ctx, "definitely-not-a-real-session-id", "new-title", nil)
	if err != nil {
		t.Logf("RenameSession(bogus id) returned error: %v", err)
	} else {
		t.Log("RenameSession(bogus id) returned nil error; CLI accepts unknown IDs silently")
	}
}

func TestSessions_Tag_InvalidID(t *testing.T) {
	ctx := sessionCtx(t)

	err := claude.TagSession(ctx, "definitely-not-a-real-session-id", "tag-value", nil)
	if err != nil {
		t.Logf("TagSession(bogus id) returned error: %v", err)
	} else {
		t.Log("TagSession(bogus id) returned nil error; CLI accepts unknown IDs silently")
	}
}

func TestSessions_Fork_InvalidID(t *testing.T) {
	ctx := sessionCtx(t)

	result, err := claude.ForkSession(ctx, "definitely-not-a-real-session-id", nil)
	if err != nil {
		t.Logf("ForkSession(bogus id) returned error: %v", err)
		return
	}
	t.Logf("ForkSession(bogus id) returned result: %+v", result)
}

func TestSessions_ListSubagents_InvalidID(t *testing.T) {
	ctx := sessionCtx(t)

	subs, err := claude.ListSubagents(ctx, "definitely-not-a-real-session-id", nil)
	// ListSubagents may legitimately return an empty slice rather than an
	// error for unknown parent sessions — accept either outcome and log.
	if err != nil {
		t.Logf("ListSubagents(bogus id) returned error: %v", err)
		return
	}
	t.Logf("ListSubagents(bogus id) returned %d subagent(s)", len(subs))
}

func TestSessions_GetSubagentMessages_InvalidID(t *testing.T) {
	ctx := sessionCtx(t)

	_, err := claude.GetSubagentMessages(ctx, "definitely-not-a-real-subagent-id", nil)
	if err == nil {
		t.Error("GetSubagentMessages(bogus id): nil error; want error")
	}
}

// TestSessions_List_ExistingSession does a round-trip: list sessions, pick
// the first one, call GetSessionInfo on it, assert the returned record
// matches. This is the wire-shape probe for SDKSessionInfo — each field
// that the CLI returns must deserialize into the Go struct.
func TestSessions_List_ExistingSession(t *testing.T) {
	ctx := sessionCtx(t)

	sessions, err := claude.ListSessions(ctx, &types.ListSessionsOptions{Limit: 1})
	if err != nil {
		t.Skipf("ListSessions: skipping — CLI error: %v", err)
	}
	if len(sessions) == 0 {
		t.Skip("no existing sessions on disk; cannot probe wire shape")
	}

	first := sessions[0]
	info, err := claude.GetSessionInfo(ctx, first.SessionID, nil)
	if err != nil {
		t.Fatalf("GetSessionInfo(%s): %v", first.SessionID, err)
	}
	if info == nil {
		t.Fatal("GetSessionInfo: nil result")
	}

	// SessionID MUST round-trip exactly.
	if info.SessionID != first.SessionID {
		t.Errorf("GetSessionInfo: SessionID = %q, want %q", info.SessionID, first.SessionID)
	}
}
