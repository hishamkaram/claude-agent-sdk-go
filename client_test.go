package claude

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal"
	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// clientTestTransport implements transport.Transport for client_test.go.
type clientTestTransport struct {
	mu           sync.Mutex
	messagesChan chan types.Message
	writtenData  []string
	closed       bool
	closeErr     error
}

func newClientTestTransport() *clientTestTransport {
	return &clientTestTransport{
		messagesChan: make(chan types.Message, 100),
		writtenData:  make([]string, 0),
	}
}

func (m *clientTestTransport) Connect(_ context.Context) error { return nil }
func (m *clientTestTransport) Close(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		close(m.messagesChan)
		m.closed = true
	}
	return m.closeErr
}

func (m *clientTestTransport) Write(_ context.Context, data string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writtenData = append(m.writtenData, data)
	return nil
}

func (m *clientTestTransport) ReadMessages(_ context.Context) <-chan types.Message {
	return m.messagesChan
}
func (m *clientTestTransport) OnError(_ error) {}
func (m *clientTestTransport) IsReady() bool   { return true }
func (m *clientTestTransport) GetError() error { return nil }

func (m *clientTestTransport) sendMessage(msg types.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.messagesChan <- msg
	}
}

// makeConnectedClient creates a Client that is wired up as if Connect() succeeded,
// using a mock transport and a real internal.Query. This is used for testing
// ReceiveResponse goroutine tracking without a live CLI process.
func makeConnectedClient(t *testing.T) (*Client, *clientTestTransport) {
	t.Helper()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	mockTransport := newClientTestTransport()
	logger := log.NewLogger(false)
	query := internal.NewQuery(ctx, mockTransport, opts, logger, true)
	if err := query.Start(ctx); err != nil {
		t.Fatalf("query.Start failed: %v", err)
	}

	client.mu.Lock()
	client.transport = mockTransport
	client.query = query
	client.connected = true
	client.mu.Unlock()

	return client, mockTransport
}

func TestNewClient_NilOptions(t *testing.T) {
	// Cannot use t.Parallel() because t.Setenv modifies process state.
	t.Setenv("CLAUDE_AGENT_SDK_SKIP_VERSION_CHECK", "1")

	ctx := context.Background()

	client, err := NewClient(ctx, nil)
	if err == nil {
		// CLI might be installed - that's OK, just clean up
		if client != nil {
			_ = client.Close(ctx)
		}
		return
	}

	// Should get CLINotFoundError
	if !types.IsCLINotFoundError(err) {
		t.Logf("Expected CLINotFoundError but got: %v", err)
	}
}

func TestNewClient_InvalidCLIPath(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/nonexistent/path/to/claude")

	client, err := NewClient(ctx, opts)
	if err != nil {
		// Expected - CLI path doesn't exist
		// However, NewClient doesn't validate the path, only Connect does
		// So we might get a client back
		if client != nil {
			_ = client.Close(ctx)
		}
	}
}

func TestNewClient_ConflictingPermissionOptions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Create a dummy callback
	canUseTool := func(ctx context.Context, toolName string, input map[string]interface{}, permCtx types.ToolPermissionContext) (interface{}, error) {
		return types.PermissionResultAllow{Behavior: "allow"}, nil
	}

	// This should fail because both are set
	promptTool := "cli"
	opts := types.NewClaudeAgentOptions().
		WithCLIPath("/bin/echo").
		WithCanUseTool(canUseTool).
		WithPermissionPromptToolName(promptTool)

	_, err := NewClient(ctx, opts)
	if err == nil {
		t.Fatal("expected error for conflicting permission options")
	}

	if err.Error() != "can_use_tool callback cannot be used with permission_prompt_tool_name" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestClient_ConnectBeforeQuery(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	// Try to query without connecting
	err = client.Query(ctx, "test")
	if err == nil {
		t.Fatal("expected error when querying without connecting")
	}

	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError, got: %T - %v", err, err)
	}
}

func TestClient_EmptyPrompt(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	// Without connecting, should get connection error first
	err = client.Query(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty prompt without connection")
	}

	// Should be connection error since we haven't connected
	if !types.IsCLIConnectionError(err) {
		t.Errorf("expected CLIConnectionError when not connected, got: %v", err)
	}
}

func TestClient_IsConnected(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	// Should not be connected initially
	if client.IsConnected() {
		t.Error("client should not be connected before Connect()")
	}

	// After close, should not be connected
	_ = client.Close(ctx)
	if client.IsConnected() {
		t.Error("client should not be connected after Close()")
	}
}

func TestClient_DoubleConnect(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	// First connect attempt (will likely fail with /bin/echo)
	err1 := client.Connect(ctx)

	// Second connect attempt
	err2 := client.Connect(ctx)

	// If first connect succeeded, second should fail with "already connected"
	if err1 == nil && err2 == nil {
		t.Error("expected error on second Connect() call")
	}

	// If second connect got an error, check if it's the right one
	if err2 != nil && types.IsControlProtocolError(err2) {
		// Good - got the expected error
		if err2.Error() != "client already connected" {
			t.Logf("Got control protocol error but unexpected message: %v", err2)
		}
	}
}

func TestClient_CloseIdempotent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}

	// Close multiple times should not panic or error
	err1 := client.Close(ctx)
	err2 := client.Close(ctx)
	err3 := client.Close(ctx)

	// All should succeed (or at least not panic)
	_ = err1
	_ = err2
	_ = err3
}

func TestClient_ContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	opts := types.NewClaudeAgentOptions().WithCLIPath("/bin/echo")

	client, err := NewClient(ctx, opts)
	if err != nil {
		t.Skip("Could not create client")
	}
	defer func() {
		_ = client.Close(ctx)
	}()

	// Cancel context before operations
	cancel()

	// Operations should respect cancellation
	err = client.Connect(ctx)
	// May fail due to cancellation or other reasons - just ensure no panic
	_ = err
}
