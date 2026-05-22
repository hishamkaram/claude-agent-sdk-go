package claude

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

const (
	fixtureSessionID = "11111111-1111-4111-8111-111111111111"
	fixtureUser1     = "22222222-2222-4222-8222-222222222222"
	fixtureAsst1     = "33333333-3333-4333-8333-333333333333"
	fixtureUser2     = "44444444-4444-4444-8444-444444444444"
	fixtureAsst2     = "55555555-5555-4555-8555-555555555555"
)

func TestLocalTranscriptBackendGetMessagesParentChainAndFilters(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	backend, _ := writeTranscriptFixture(t, cwd, fixtureSessionID, strings.Join([]string{
		`{"type":"summary","summary":"Compacted summary","timestamp":"2026-05-01T10:00:00Z"}`,
		`{"type":"user","uuid":"` + fixtureUser1 + `","parentUuid":null,"sessionId":"` + fixtureSessionID + `","cwd":"` + cwd + `","timestamp":"2026-05-01T10:00:00Z","message":{"role":"user","content":[{"type":"text","text":"first prompt"}]}}`,
		`{"type":"assistant","uuid":"` + fixtureAsst1 + `","parentUuid":"` + fixtureUser1 + `","sessionId":"` + fixtureSessionID + `","timestamp":"2026-05-01T10:00:01Z","message":{"role":"assistant","content":[{"type":"text","text":"first answer"}]}}`,
		`{"type":"assistant","uuid":"66666666-6666-4666-8666-666666666666","parentUuid":"` + fixtureAsst1 + `","sessionId":"` + fixtureSessionID + `","isSidechain":true,"message":{"role":"assistant","content":[{"type":"text","text":"filtered sidechain"}]}}`,
		`{"type":"user","uuid":"77777777-7777-4777-8777-777777777777","parentUuid":"` + fixtureAsst1 + `","sessionId":"` + fixtureSessionID + `","isMeta":true,"message":{"role":"user","content":"filtered meta"}}`,
		`{"type":"user","uuid":"88888888-8888-4888-8888-888888888888","parentUuid":"` + fixtureAsst1 + `","sessionId":"` + fixtureSessionID + `","userType":"team","message":{"role":"user","content":"filtered team"}}`,
		`{"type":"user","uuid":"` + fixtureUser2 + `","parentUuid":"` + fixtureAsst1 + `","sessionId":"` + fixtureSessionID + `","message":{"role":"user","content":"compact summary retained as user content"}}`,
		`{"type":"assistant","uuid":"` + fixtureAsst2 + `","parentUuid":"` + fixtureUser2 + `","sessionId":"` + fixtureSessionID + `","message":{"role":"assistant","content":[{"type":"text","text":"second answer"}]}}`,
	}, "\n"))

	messages, err := backend.GetSessionMessages(context.Background(), fixtureSessionID, &types.GetSessionMessagesOptions{Dir: cwd})
	if err != nil {
		t.Fatalf("GetSessionMessages() error = %v", err)
	}
	if got, want := len(messages), 4; got != want {
		t.Fatalf("len(messages) = %d, want %d: %#v", got, want, messages)
	}
	wantUUIDs := []string{fixtureUser1, fixtureAsst1, fixtureUser2, fixtureAsst2}
	for i, want := range wantUUIDs {
		if messages[i].UUID != want {
			t.Fatalf("messages[%d].UUID = %q, want %q", i, messages[i].UUID, want)
		}
		if messages[i].SessionID != fixtureSessionID {
			t.Fatalf("messages[%d].SessionID = %q, want %q", i, messages[i].SessionID, fixtureSessionID)
		}
	}
	if got := extractClaudeMessageText(messages[2].Message); got != "compact summary retained as user content" {
		t.Fatalf("compact summary text = %q", got)
	}
}

func TestLocalTranscriptBackendListAndInfo(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	backend, _ := writeTranscriptFixture(t, cwd, fixtureSessionID, strings.Join([]string{
		`{"type":"summary","summary":"Session summary","timestamp":"2026-05-01T09:00:00Z"}`,
		`{"type":"user","uuid":"` + fixtureUser1 + `","sessionId":"` + fixtureSessionID + `","cwd":"` + cwd + `","gitBranch":"main","timestamp":"2026-05-01T09:00:00Z","message":{"role":"user","content":"hello"}}`,
	}, "\n"))

	sessions, err := backend.ListSessions(context.Background(), &types.ListSessionsOptions{Dir: cwd})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(sessions))
	}
	if sessions[0].SessionID != fixtureSessionID || sessions[0].Summary != "Session summary" || sessions[0].FirstPrompt != "hello" || sessions[0].GitBranch != "main" {
		t.Fatalf("unexpected session info: %+v", sessions[0])
	}

	info, err := backend.GetSessionInfo(context.Background(), fixtureSessionID, &types.GetSessionInfoOptions{Dir: cwd})
	if err != nil {
		t.Fatalf("GetSessionInfo() error = %v", err)
	}
	if info.CWD != cwd || info.CreatedAt == 0 || info.LastModified == 0 {
		t.Fatalf("unexpected info: %+v", info)
	}
}

func TestLocalTranscriptBackendTypedErrors(t *testing.T) {
	t.Parallel()
	backend := NewLocalTranscriptBackend()
	if _, err := backend.GetSessionMessages(context.Background(), "not-a-uuid", nil); !errors.Is(err, types.ErrInvalidSessionID) {
		t.Fatalf("invalid session error = %v, want ErrInvalidSessionID", err)
	}
	if _, err := backend.GetSessionInfo(context.Background(), fixtureSessionID, &types.GetSessionInfoOptions{Dir: t.TempDir()}); !errors.Is(err, types.ErrSessionNotFound) {
		t.Fatalf("missing session error = %v, want ErrSessionNotFound", err)
	}
	if _, err := backend.GetSubagentMessages(context.Background(), fixtureSessionID, "../bad", nil); !errors.Is(err, types.ErrInvalidSessionID) {
		t.Fatalf("subagent traversal error = %v, want ErrInvalidSessionID", err)
	}
}

func TestLocalTranscriptBackendSubagentMessages(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	backend, parentPath := writeTranscriptFixture(t, cwd, fixtureSessionID, `{"type":"user","uuid":"`+fixtureUser1+`","sessionId":"`+fixtureSessionID+`","cwd":"`+cwd+`","message":{"role":"user","content":"parent"}}`)
	projectDir := filepath.Dir(parentPath)
	subPath := filepath.Join(projectDir, "subagents", "worker.jsonl")
	if err := os.MkdirAll(filepath.Dir(subPath), 0o700); err != nil {
		t.Fatalf("MkdirAll subagent: %v", err)
	}
	if err := os.WriteFile(subPath, []byte(strings.Join([]string{
		`{"type":"user","uuid":"` + fixtureUser1 + `","sessionId":"` + fixtureSessionID + `","message":{"role":"user","content":"sub prompt"}}`,
		`{"type":"assistant","uuid":"` + fixtureAsst1 + `","parentUuid":"` + fixtureUser1 + `","sessionId":"` + fixtureSessionID + `","message":{"role":"assistant","content":[{"type":"text","text":"sub answer"}]}}`,
	}, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile subagent: %v", err)
	}

	agents, err := backend.ListSubagents(context.Background(), fixtureSessionID, &types.ListSubagentsOptions{Dir: cwd})
	if err != nil {
		t.Fatalf("ListSubagents: %v", err)
	}
	if len(agents) != 1 || agents[0].ID != "subagents/worker" || agents[0].ParentSessionID != fixtureSessionID {
		t.Fatalf("subagents = %+v", agents)
	}
	messages, err := backend.GetSubagentMessages(context.Background(), fixtureSessionID, "subagents/worker", &types.GetSubagentMessagesOptions{Dir: cwd})
	if err != nil {
		t.Fatalf("GetSubagentMessages: %v", err)
	}
	if len(messages) != 2 || messages[1].UUID != fixtureAsst1 {
		t.Fatalf("subagent messages = %+v", messages)
	}
}

func TestLocalTranscriptBackendMalformedJSONL(t *testing.T) {
	t.Parallel()
	cwd := t.TempDir()
	backend, _ := writeTranscriptFixture(t, cwd, fixtureSessionID, `{"type":"user","uuid":"`+fixtureUser1+`","sessionId":"`+fixtureSessionID+`","message":{"role":"user","content":"ok"}}
{bad-json`)

	_, err := backend.GetSessionMessages(context.Background(), fixtureSessionID, &types.GetSessionMessagesOptions{Dir: cwd})
	if !errors.Is(err, types.ErrMalformedTranscript) {
		t.Fatalf("GetSessionMessages() error = %v, want ErrMalformedTranscript", err)
	}
}

func TestClaudeProjectKeyNormalizationAndLongPath(t *testing.T) {
	t.Parallel()
	decomposed, err := ClaudeProjectKey("/tmp/Cafe\u0301")
	if err != nil {
		t.Fatalf("ClaudeProjectKey(decomposed): %v", err)
	}
	composed, err := ClaudeProjectKey("/tmp/Caf\u00e9")
	if err != nil {
		t.Fatalf("ClaudeProjectKey(composed): %v", err)
	}
	if decomposed != composed {
		t.Fatalf("NFC keys differ: %q != %q", decomposed, composed)
	}

	longPath := "/tmp/" + strings.Repeat("segment/", 80)
	key, err := ClaudeProjectKey(longPath)
	if err != nil {
		t.Fatalf("ClaudeProjectKey(long): %v", err)
	}
	if len(key) > maxClaudeProjectKeyLen {
		t.Fatalf("long key length = %d, want <= %d", len(key), maxClaudeProjectKeyLen)
	}
	if !strings.Contains(key, "-") {
		t.Fatalf("long key should include hash suffix separator: %q", key)
	}
}

func TestLocalTranscriptBackendClaudeConfigDirOverride(t *testing.T) {
	configDir := t.TempDir()
	cwd := t.TempDir()
	key, err := ClaudeProjectKey(cwd)
	if err != nil {
		t.Fatalf("ClaudeProjectKey: %v", err)
	}
	projectDir := filepath.Join(configDir, "projects", key)
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, fixtureSessionID+".jsonl"), []byte(`{"type":"user","uuid":"`+fixtureUser1+`","sessionId":"`+fixtureSessionID+`","cwd":"`+cwd+`","message":{"role":"user","content":"from env"}}`+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)

	sessions, err := NewLocalTranscriptBackend().ListSessions(context.Background(), &types.ListSessionsOptions{Dir: cwd})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 1 || sessions[0].SessionID != fixtureSessionID {
		t.Fatalf("sessions = %+v, want fixture session", sessions)
	}
}

func writeTranscriptFixture(t *testing.T, cwd, sessionID, content string) (*LocalTranscriptBackend, string) {
	t.Helper()
	projectsDir := t.TempDir()
	key, err := ClaudeProjectKey(cwd)
	if err != nil {
		t.Fatalf("ClaudeProjectKey: %v", err)
	}
	projectDir := filepath.Join(projectsDir, key)
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(projectDir, sessionID+".jsonl")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return &LocalTranscriptBackend{ProjectsDir: projectsDir}, path
}
