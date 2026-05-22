package claude

import (
	"context"
	"errors"
	"testing"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

type fakeSessionStore struct {
	entry     *SessionStoreEntry
	list      []SessionStoreListEntry
	summaries []SessionSummaryEntry
	loadKey   SessionKey
}

func (s *fakeSessionStore) Load(ctx context.Context, key SessionKey) (*SessionStoreEntry, error) {
	s.loadKey = key
	return s.entry, nil
}

type fakeListStore struct {
	fakeSessionStore
}

func (s *fakeListStore) ListSessions(ctx context.Context, opts *types.ListSessionsOptions) ([]SessionStoreListEntry, error) {
	return s.list, nil
}

type fakeSummaryStore struct {
	fakeSessionStore
}

func (s *fakeSummaryStore) ListSessionSummaries(ctx context.Context, opts *types.ListSessionsOptions) ([]SessionSummaryEntry, error) {
	return s.summaries, nil
}

func TestSessionStoreBackendLoadBackedMessages(t *testing.T) {
	t.Parallel()
	store := &fakeSessionStore{entry: &SessionStoreEntry{
		Key:       SessionKey{SessionID: fixtureSessionID},
		SessionID: fixtureSessionID,
		Messages: []types.SessionMessage{
			{Type: "user", UUID: fixtureUser1, SessionID: fixtureSessionID, Message: []byte(`{"role":"user","content":"one"}`)},
			{Type: "assistant", UUID: fixtureAsst1, SessionID: fixtureSessionID, Message: []byte(`{"role":"assistant","content":"two"}`)},
		},
	}}
	backend, err := NewSessionStoreBackend(store)
	if err != nil {
		t.Fatalf("NewSessionStoreBackend: %v", err)
	}
	messages, err := backend.GetSessionMessages(context.Background(), fixtureSessionID, &types.GetSessionMessagesOptions{Dir: "/tmp/project", Offset: 1})
	if err != nil {
		t.Fatalf("GetSessionMessages: %v", err)
	}
	if len(messages) != 1 || messages[0].UUID != fixtureAsst1 {
		t.Fatalf("messages = %+v", messages)
	}
	if store.loadKey.Dir != "/tmp/project" {
		t.Fatalf("load key dir = %q", store.loadKey.Dir)
	}
}

func TestSessionStoreBackendListFastPaths(t *testing.T) {
	t.Parallel()
	listStore := &fakeListStore{fakeSessionStore: fakeSessionStore{list: []SessionStoreListEntry{
		{SessionID: fixtureSessionID, Summary: "full", LastModified: 2},
	}}}
	backend, _ := NewSessionStoreBackend(listStore)
	sessions, err := backend.ListSessions(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListSessions full path: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Summary != "full" {
		t.Fatalf("full sessions = %+v", sessions)
	}

	summaryStore := &fakeSummaryStore{fakeSessionStore: fakeSessionStore{summaries: []SessionSummaryEntry{
		{SessionID: fixtureSessionID, Summary: "summary", LastModified: 3},
	}}}
	backend, _ = NewSessionStoreBackend(summaryStore)
	sessions, err = backend.ListSessions(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListSessions summary path: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Summary != "summary" {
		t.Fatalf("summary sessions = %+v", sessions)
	}
}

func TestSessionStoreBackendUnsupportedAndMissing(t *testing.T) {
	t.Parallel()
	backend, err := NewSessionStoreBackend(&fakeSessionStore{})
	if err != nil {
		t.Fatalf("NewSessionStoreBackend: %v", err)
	}
	if _, err := backend.ListSessions(context.Background(), nil); !errors.Is(err, types.ErrSessionHistoryUnsupported) {
		t.Fatalf("ListSessions error = %v, want unsupported", err)
	}
	if _, err := backend.GetSessionInfo(context.Background(), fixtureSessionID, nil); !errors.Is(err, types.ErrSessionNotFound) {
		t.Fatalf("GetSessionInfo error = %v, want not found", err)
	}
	if _, err := NewSessionStoreBackend(nil); err == nil {
		t.Fatal("NewSessionStoreBackend(nil) error = nil, want error")
	}
}
