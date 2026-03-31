package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/hishamkaram/claude-agent-sdk-go/internal"
	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/internal/transport"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// Client provides bidirectional communication with Claude Code CLI for interactive sessions.
//
// Unlike the Query function which is designed for one-shot interactions, Client maintains
// a persistent connection and supports multiple query/response cycles, permission callbacks,
// hooks, and full control protocol features.
//
// Lifecycle:
//  1. Create client with NewClient()
//  2. Connect with Connect()
//  3. Send queries with Query()
//  4. Receive responses with ReceiveResponse()
//  5. Repeat steps 3-4 as needed
//  6. Clean up with Close()
//
// Example usage:
//
//	ctx := context.Background()
//	opts := types.NewClaudeAgentOptions().
//	    WithModel("claude-3-5-sonnet-latest").
//	    WithPermissionMode(types.PermissionModeAcceptEdits)
//
//	client, err := NewClient(ctx, opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close(ctx)
//
//	// Connect to Claude
//	if err := client.Connect(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
//	// First query
//	if err := client.Query(ctx, "List files in current directory"); err != nil {
//	    log.Fatal(err)
//	}
//
//	for msg := range client.ReceiveResponse(ctx) {
//	    // Process messages
//	}
//
//	// Second query in same session
//	if err := client.Query(ctx, "Create a new file"); err != nil {
//	    log.Fatal(err)
//	}
//
//	for msg := range client.ReceiveResponse(ctx) {
//	    // Process messages
//	}
//
// Thread Safety:
//
// Client is not thread-safe. All methods should be called from the same goroutine,
// or you must provide your own synchronization.
type Client struct {
	options   *types.ClaudeAgentOptions
	transport transport.Transport
	query     *internal.Query
	logger    *log.Logger

	mu           sync.Mutex
	connected    bool
	connecting   bool // true while Connect() is performing blocking operations without the lock
	closePending bool // true when Close() was called during an in-progress Connect()
	ctx          context.Context
	cancel       context.CancelFunc
	initResult   *types.InitializeResult // Parsed initialization response.

	recvWg sync.WaitGroup // tracks in-flight ReceiveResponse goroutines
}

// NewClient creates a new interactive client with the given options.
//
// This does not establish a connection; you must call Connect() before sending queries.
//
// Parameters:
//   - ctx: Parent context for the client lifecycle
//   - options: Configuration options (nil uses defaults)
//
// Returns:
//   - A new Client instance
//   - An error if the CLI cannot be found or options are invalid
func NewClient(ctx context.Context, options *types.ClaudeAgentOptions) (*Client, error) {
	// Use default options if not provided
	if options == nil {
		options = types.NewClaudeAgentOptions()
	}

	// Validate permission callback configuration
	if options.CanUseTool != nil && options.PermissionPromptToolName != nil {
		return nil, fmt.Errorf("can_use_tool callback cannot be used with permission_prompt_tool_name")
	}

	// If CanUseTool is provided, automatically set PermissionPromptToolName to "stdio"
	if options.CanUseTool != nil && options.PermissionPromptToolName == nil {
		stdio := "stdio"
		options.PermissionPromptToolName = &stdio
	}

	// Find CLI path
	cliPath := ""
	if options.CLIPath != nil {
		cliPath = *options.CLIPath
	} else {
		var err error
		cliPath, err = transport.FindCLI()
		if err != nil {
			return nil, err
		}
	}

	// Determine working directory
	cwd := ""
	if options.CWD != nil {
		cwd = *options.CWD
	}

	// Prepare environment
	env := make(map[string]string)
	if options.Env != nil {
		for k, v := range options.Env {
			env[k] = v
		}
	}

	// Create client context
	clientCtx, cancel := context.WithCancel(ctx)

	// Create logger
	logger := log.NewLogger(options.Verbose)

	// Determine resume session ID from options
	resumeID := ""
	if options.Resume != nil && *options.Resume != "" {
		resumeID = *options.Resume
	}

	// Create subprocess transport with optional resume and options
	transportInst := transport.NewSubprocessCLITransport(cliPath, cwd, env, logger, resumeID, options)

	return &Client{
		options:   options,
		transport: transportInst,
		logger:    logger,
		connected: false,
		ctx:       clientCtx,
		cancel:    cancel,
	}, nil
}

// Connect establishes a connection to Claude Code CLI in streaming mode.
//
// This must be called before sending any queries. The connection uses streaming mode
// which enables full control protocol support including permissions, hooks, and
// bidirectional communication.
//
// Returns an error if:
//   - Already connected
//   - CLI subprocess fails to start
//   - Initialization fails
//
// Example:
//
//	if err := client.Connect(ctx); err != nil {
//	    if types.IsCLIConnectionError(err) {
//	        log.Fatal("Failed to connect:", err)
//	    }
//	    log.Fatal(err)
//	}
func (c *Client) Connect(ctx context.Context) error {
	// Phase 1: Acquire lock, check state, set connecting flag, release lock.
	// This allows Close() and other methods to proceed without blocking on
	// the long-running transport.Connect / query.Initialize calls.
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return types.NewControlProtocolError("client already connected")
	}
	if c.connecting {
		c.mu.Unlock()
		return types.NewControlProtocolError("client is already connecting")
	}
	c.connecting = true
	c.mu.Unlock()

	// Ensure the connecting and closePending flags are cleared on any exit path.
	defer func() {
		c.mu.Lock()
		c.connecting = false
		c.closePending = false
		c.mu.Unlock()
	}()

	c.logger.Info("Connecting to Claude CLI...")

	// Phase 2: Perform blocking operations WITHOUT holding the lock.

	// Connect transport
	if err := c.transport.Connect(ctx); err != nil {
		c.logger.Error("failed to connect transport", zap.Error(err))
		return types.NewCLIConnectionErrorWithCause("failed to connect to Claude CLI", err)
	}
	c.logger.Debug("Transport connected successfully")

	// Check for immediate errors (like session not found)
	select {
	case <-c.ctx.Done():
		_ = c.transport.Close(ctx)
		return ctx.Err()
	default:
		if err := c.transport.GetError(); err != nil {
			c.logger.Error("transport error detected during connection", zap.Error(err))
			_ = c.transport.Close(ctx)
			return err
		}
	}

	// Create query handler in streaming mode
	query := internal.NewQuery(ctx, c.transport, c.options, c.logger, true)
	c.logger.Debug("Query handler created")

	// Start message processing
	if err := query.Start(ctx); err != nil {
		c.logger.Error("failed to start message processing", zap.Error(err))
		_ = c.transport.Close(ctx)
		return err
	}
	c.logger.Debug("Message processing started")

	// Initialize control protocol
	initRaw, err := query.Initialize(ctx)
	if err != nil {
		c.logger.Error("failed to initialize control protocol", zap.Error(err))
		_ = query.Stop(ctx)
		_ = c.transport.Close(ctx)
		return types.NewControlProtocolErrorWithCause("failed to initialize control protocol", err)
	}
	c.logger.Debug("Control protocol initialized")

	// Parse the init result into typed structure.
	initResult := parseInitResult(initRaw)

	// Phase 3: Re-acquire lock and commit the connected state.
	c.mu.Lock()
	if c.closePending {
		// Close() was called while we were connecting. Clean up instead of completing.
		c.closePending = false
		c.mu.Unlock()

		c.logger.Info("Connect completed but Close was requested — cleaning up")
		_ = query.Stop(ctx)
		_ = c.transport.Close(ctx)
		return types.NewControlProtocolError("client closed during connect")
	}
	c.query = query
	c.initResult = initResult
	c.connected = true
	c.mu.Unlock()

	c.logger.Info("Successfully connected to Claude")
	return nil
}

// Query sends a prompt to Claude in the current session.
//
// This returns immediately after sending the prompt. Use ReceiveResponse() to
// get the response messages.
//
// Multiple calls to Query() can be made in sequence to have a multi-turn conversation.
// Each query/response cycle should be completed before sending the next query.
//
// Parameters:
//   - ctx: Context for cancellation
//   - prompt: The text prompt to send
//
// Returns an error if:
//   - Not connected (call Connect() first)
//   - Write to CLI fails
//   - Context is cancelled
//
// Example:
//
//	if err := client.Query(ctx, "What files are in this directory?"); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Now receive the response
//	for msg := range client.ReceiveResponse(ctx) {
//	    // Process messages
//	}
func (c *Client) Query(ctx context.Context, prompt string) error {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return types.NewCLIConnectionError("not connected - call Connect() first")
	}
	c.mu.Unlock()

	// Validate prompt
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	// Build query message
	queryMsg := map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"role":    "user",
			"content": prompt,
		},
		"parent_tool_use_id": nil,
		"session_id":         "default",
	}

	// Marshal and send
	data, err := json.Marshal(queryMsg)
	if err != nil {
		return types.NewControlProtocolErrorWithCause("failed to marshal query", err)
	}

	if err := c.transport.Write(ctx, string(data)); err != nil {
		return fmt.Errorf("Client.Query: write to transport: %w", err)
	}

	return nil
}

// QueryWithContent sends a structured content query (text + images) to Claude.
//
// This method allows sending messages with mixed content types (text and images),
// following the Claude API's content block format. Unlike Query() which only accepts
// plain text, this method accepts an array of content blocks.
//
// Content blocks can be:
//   - Text blocks: map[string]interface{}{"type": "text", "text": "..."}
//   - Image blocks: map[string]interface{}{"type": "image", "source": {...}}
//
// Example usage:
//
//	content := []interface{}{
//	    map[string]interface{}{
//	        "type": "text",
//	        "text": "What's in this image?",
//	    },
//	    map[string]interface{}{
//	        "type": "image",
//	        "source": map[string]interface{}{
//	            "type":       "base64",
//	            "media_type": "image/png",
//	            "data":       "iVBORw0KG...",
//	        },
//	    },
//	}
//
//	if err := client.QueryWithContent(ctx, content); err != nil {
//	    log.Fatal(err)
//	}
//
//	for msg := range client.ReceiveResponse(ctx) {
//	    // Process messages
//	}
func (c *Client) QueryWithContent(ctx context.Context, content interface{}) error {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return types.NewCLIConnectionError("not connected - call Connect() first")
	}
	c.mu.Unlock()

	// Validate content
	if content == nil {
		return fmt.Errorf("content cannot be nil")
	}

	// Build query message with structured content
	queryMsg := map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"role":    "user",
			"content": content, // This can be a string or []ContentBlock
		},
		"parent_tool_use_id": nil,
		"session_id":         "default",
	}

	// Marshal and send
	data, err := json.Marshal(queryMsg)
	if err != nil {
		return types.NewControlProtocolErrorWithCause("failed to marshal query", err)
	}

	if err := c.transport.Write(ctx, string(data)); err != nil {
		return fmt.Errorf("Client.QueryWithContent: write to transport: %w", err)
	}

	return nil
}

// ReceiveResponse returns a channel of response messages from Claude.
//
// This should be called after Query() to receive the response. The channel will
// receive messages until a ResultMessage is received, then it will be closed.
//
// The channel yields:
//   - UserMessage: Messages from the user (echoed back)
//   - AssistantMessage: Claude's text responses and tool uses
//   - SystemMessage: System notifications and control messages
//   - ResultMessage: Final result with cost/usage info (last message)
//
// The channel is closed when:
//   - A ResultMessage is received
//   - An error occurs
//   - The context is cancelled
//
// Example:
//
//	for msg := range client.ReceiveResponse(ctx) {
//	    switch m := msg.(type) {
//	    case *types.AssistantMessage:
//	        for _, block := range m.Content {
//	            if tb, ok := block.(*types.TextBlock); ok {
//	                fmt.Println("Claude:", tb.Text)
//	            }
//	        }
//	    case *types.ResultMessage:
//	        fmt.Printf("Done. Cost: $%.4f\n", *m.TotalCostUSD)
//	    }
//	}
func (c *Client) ReceiveResponse(ctx context.Context) <-chan types.Message {
	outputChan := make(chan types.Message, 10)

	c.recvWg.Add(1)
	go func() {
		defer c.recvWg.Done()
		defer close(outputChan)

		c.mu.Lock()
		if !c.connected || c.query == nil {
			c.mu.Unlock()
			return
		}
		messagesChan := c.query.GetMessages(ctx)
		c.mu.Unlock()

		for {
			select {
			case <-ctx.Done():
				return
			case <-c.ctx.Done():
				return
			case msg, ok := <-messagesChan:
				if !ok {
					// Messages channel closed
					return
				}

				// Forward message to output
				select {
				case outputChan <- msg:
					// Check if this is a result message (end of response)
					if _, isResult := msg.(*types.ResultMessage); isResult {
						return
					}
				case <-ctx.Done():
					return
				case <-c.ctx.Done():
					return
				}
			}
		}
	}()

	return outputChan
}

// Close gracefully terminates the Claude session and cleans up resources.
//
// This should be called when you're done with the client, typically using defer:
//
//	client, err := NewClient(ctx, opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close(ctx)
//
// After Close() is called, the client cannot be reused. Create a new client if needed.
//
// Returns an error if cleanup fails, but the client is marked as disconnected regardless.
func (c *Client) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		// If Connect() is in progress (Phase 2, lock released), set closePending
		// so Connect() will clean up in Phase 3 instead of completing.
		if c.connecting {
			c.closePending = true
			// Cancel the client context so blocking transport/init calls unblock.
			if c.cancel != nil {
				c.cancel()
			}
			c.logger.Info("Close requested during Connect — flagged for cleanup")
			return nil
		}
		return nil
	}

	c.logger.Info("Closing Claude connection...")

	var errs []error

	// Stop query handler
	if c.query != nil {
		if err := c.query.Stop(ctx); err != nil {
			c.logger.Warn("error stopping query handler", zap.Error(err))
			errs = append(errs, err)
		}
		c.query = nil
	}

	// Close transport
	if c.transport != nil {
		if err := c.transport.Close(ctx); err != nil {
			c.logger.Warn("error closing transport", zap.Error(err))
			errs = append(errs, err)
		}
	}

	// Cancel context so ReceiveResponse goroutines unblock
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}

	// Wait for in-flight ReceiveResponse goroutines with a bounded timeout
	// to prevent Close from blocking indefinitely.
	recvDone := make(chan struct{})
	go func() {
		c.recvWg.Wait()
		close(recvDone)
	}()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-recvDone:
		// All ReceiveResponse goroutines have exited.
	case <-timer.C:
		c.logger.Warn("timed out waiting for ReceiveResponse goroutines to exit")
	}

	c.connected = false
	c.logger.Debug("Connection closed")

	// Return first error if any
	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// IsConnected returns true if the client is currently connected to Claude.
//
// This can be used to check connection state before calling methods that require
// an active connection.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// InitResult returns the parsed initialization response from the control protocol.
// Returns nil if Connect() has not been called or initialization did not return data.
//
// The result contains available slash commands/skills, and the raw response map
// for forward compatibility with future CLI features.
func (c *Client) InitResult() *types.InitializeResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.initResult
}

// SlashCommands returns the slash commands/skills available in the current session.
// This is a convenience accessor for InitResult().Commands.
// Returns nil if Connect() has not been called or no commands were returned.
func (c *Client) SlashCommands() []types.SlashCommand {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.initResult == nil {
		return nil
	}
	return c.initResult.Commands
}

// SupportedModels returns the list of models available in this session.
// Returns nil if Connect() has not been called or the CLI did not return model info.
func (c *Client) SupportedModels() []types.ModelInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.initResult == nil {
		return nil
	}
	return c.initResult.Models
}

// SetModel changes the model used for subsequent responses.
// Pass an empty string to revert to the session's default model.
// Only valid after Connect() has been called.
func (c *Client) SetModel(ctx context.Context, model string) error {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return types.NewCLIConnectionError("not connected - call Connect() first")
	}
	c.mu.Unlock()

	req := map[string]interface{}{"subtype": "set_model"}
	if model != "" {
		req["model"] = model
	}
	_, err := c.query.SendControlMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("Client.SetModel: %w", err)
	}
	return nil
}

// SetPermissionMode changes the permission mode mid-session.
// Use types.PermissionModePlan to enter plan mode (/plan equivalent).
// Only valid after Connect() has been called.
func (c *Client) SetPermissionMode(ctx context.Context, mode types.PermissionMode) error {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return types.NewCLIConnectionError("not connected - call Connect() first")
	}
	c.mu.Unlock()

	req := map[string]interface{}{
		"subtype": "set_permission_mode",
		"mode":    string(mode),
	}
	_, err := c.query.SendControlMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("Client.SetPermissionMode: %w", err)
	}
	return nil
}

// ProcessID returns the OS process ID of the Claude Code subprocess.
// Returns 0 before Connect() completes or if the transport does not support PIDs.
func (c *Client) ProcessID() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	type pidProvider interface {
		ProcessID() int
	}
	if pp, ok := c.transport.(pidProvider); ok {
		return pp.ProcessID()
	}
	return 0
}

// Interrupt sends an interrupt control request to cancel the active query.
// Returns an error if the client is not connected.
func (c *Client) Interrupt(ctx context.Context) error {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return types.NewCLIConnectionError("Client.Interrupt: not connected - call Connect() first")
	}
	c.mu.Unlock()

	req := map[string]interface{}{"subtype": "interrupt"}
	_, err := c.query.SendControlMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("Client.Interrupt: %w", err)
	}
	return nil
}

// StreamInput writes user content to the subprocess stdin during an active stream.
// This is used to respond to interactive prompts (e.g., AskUserQuestion tool calls)
// without starting a new Query() call.
func (c *Client) StreamInput(ctx context.Context, content string) error {
	if content == "" {
		return fmt.Errorf("Client.StreamInput: content: %w", types.ErrEmptyParameter)
	}

	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return types.NewCLIConnectionError("Client.StreamInput: not connected - call Connect() first")
	}
	c.mu.Unlock()

	msg := map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"role":    "user",
			"content": content,
		},
		"parent_tool_use_id": nil,
		"session_id":         "default",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("Client.StreamInput: failed to marshal message: %w", err)
	}

	if err := c.transport.Write(ctx, string(data)); err != nil {
		return fmt.Errorf("Client.StreamInput: %w", err)
	}

	return nil
}

// StopTask sends a stop_task control request to cancel a specific background task.
func (c *Client) StopTask(ctx context.Context, taskID string) error {
	if taskID == "" {
		return fmt.Errorf("Client.StopTask: taskID: %w", types.ErrEmptyParameter)
	}

	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return types.NewCLIConnectionError("Client.StopTask: not connected - call Connect() first")
	}
	c.mu.Unlock()

	req := map[string]interface{}{
		"subtype": "stop_task",
		"task_id": taskID,
	}
	_, err := c.query.SendControlMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("Client.StopTask: %w", err)
	}
	return nil
}

// MCPServerStatus requests the status of all MCP server connections.
func (c *Client) MCPServerStatus(ctx context.Context) ([]types.McpServerStatusInfo, error) {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return nil, types.NewCLIConnectionError("Client.MCPServerStatus: not connected - call Connect() first")
	}
	c.mu.Unlock()

	req := map[string]interface{}{"subtype": "mcp_status"}
	resp, err := c.query.SendControlMessage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Client.MCPServerStatus: %w", err)
	}

	// Parse response into typed slice
	serversRaw, ok := resp["servers"]
	if !ok {
		return nil, nil
	}

	data, err := json.Marshal(serversRaw)
	if err != nil {
		return nil, fmt.Errorf("Client.MCPServerStatus: failed to marshal servers response: %w", err)
	}

	var servers []types.McpServerStatusInfo
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil, fmt.Errorf("Client.MCPServerStatus: failed to parse servers response: %w", err)
	}

	return servers, nil
}

// ReconnectMCPServer reconnects a disconnected MCP server by name.
func (c *Client) ReconnectMCPServer(ctx context.Context, serverName string) error {
	if serverName == "" {
		return fmt.Errorf("Client.ReconnectMCPServer: serverName: %w", types.ErrEmptyParameter)
	}

	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return types.NewCLIConnectionError("Client.ReconnectMCPServer: not connected - call Connect() first")
	}
	c.mu.Unlock()

	req := map[string]interface{}{
		"subtype":    "mcp_reconnect",
		"serverName": serverName,
	}
	_, err := c.query.SendControlMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("Client.ReconnectMCPServer: %w", err)
	}
	return nil
}

// ToggleMCPServer enables or disables an MCP server by name.
func (c *Client) ToggleMCPServer(ctx context.Context, serverName string, enabled bool) error {
	if serverName == "" {
		return fmt.Errorf("Client.ToggleMCPServer: serverName: %w", types.ErrEmptyParameter)
	}

	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return types.NewCLIConnectionError("Client.ToggleMCPServer: not connected - call Connect() first")
	}
	c.mu.Unlock()

	req := map[string]interface{}{
		"subtype":    "mcp_toggle",
		"serverName": serverName,
		"enabled":    enabled,
	}
	_, err := c.query.SendControlMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("Client.ToggleMCPServer: %w", err)
	}
	return nil
}

// SetMCPServers replaces the set of dynamically managed MCP servers.
func (c *Client) SetMCPServers(ctx context.Context, servers map[string]interface{}) (*types.McpSetServersResult, error) {
	if servers == nil {
		return nil, fmt.Errorf("Client.SetMCPServers: servers: %w", types.ErrEmptyParameter)
	}

	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return nil, types.NewCLIConnectionError("Client.SetMCPServers: not connected - call Connect() first")
	}
	c.mu.Unlock()

	req := map[string]interface{}{
		"subtype": "mcp_set_servers",
		"servers": servers,
	}
	resp, err := c.query.SendControlMessage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Client.SetMCPServers: %w", err)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("Client.SetMCPServers: failed to marshal response: %w", err)
	}

	var result types.McpSetServersResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("Client.SetMCPServers: failed to parse response: %w", err)
	}

	return &result, nil
}

// RewindFiles restores files to their state at a specific user message checkpoint.
// Requires file checkpointing to be enabled.
func (c *Client) RewindFiles(ctx context.Context, userMessageID string, dryRun bool) (*types.RewindFilesResult, error) {
	if userMessageID == "" {
		return nil, fmt.Errorf("Client.RewindFiles: userMessageID: %w", types.ErrEmptyParameter)
	}

	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return nil, types.NewCLIConnectionError("Client.RewindFiles: not connected - call Connect() first")
	}
	c.mu.Unlock()

	req := map[string]interface{}{
		"subtype":         "rewind_files",
		"user_message_id": userMessageID,
		"dry_run":         dryRun,
	}
	resp, err := c.query.SendControlMessage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Client.RewindFiles: %w", err)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("Client.RewindFiles: failed to marshal response: %w", err)
	}

	var result types.RewindFilesResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("Client.RewindFiles: failed to parse response: %w", err)
	}

	return &result, nil
}

// SupportedAgents returns the list of supported agent types from the init result.
// Returns nil if Connect() has not been called or no agents were returned.
func (c *Client) SupportedAgents() []types.AgentInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.initResult == nil {
		return nil
	}
	return c.initResult.Agents
}

// parseInitResult converts the raw initialize response map into a typed InitializeResult.
func parseInitResult(raw map[string]interface{}) *types.InitializeResult {
	if raw == nil {
		return nil
	}

	result := &types.InitializeResult{Raw: raw}

	// Parse "commands" array: each element has "name", "description", "argumentHint".
	if cmdsRaw, ok := raw["commands"]; ok {
		if cmdsSlice, ok := cmdsRaw.([]interface{}); ok {
			commands := make([]types.SlashCommand, 0, len(cmdsSlice))
			for _, item := range cmdsSlice {
				m, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				cmd := types.SlashCommand{}
				if name, ok := m["name"].(string); ok {
					cmd.Name = name
				}
				if desc, ok := m["description"].(string); ok {
					cmd.Description = desc
				}
				if hint, ok := m["argumentHint"].(string); ok {
					cmd.ArgumentHint = hint
				}
				if cmd.Name != "" {
					commands = append(commands, cmd)
				}
			}
			result.Commands = commands
		}
	}

	// Parse "models" array: each element has "value", "displayName", "description".
	if modelsRaw, ok := raw["models"]; ok {
		if modelsSlice, ok := modelsRaw.([]interface{}); ok {
			models := make([]types.ModelInfo, 0, len(modelsSlice))
			for _, item := range modelsSlice {
				m, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				info := types.ModelInfo{}
				if v, ok := m["value"].(string); ok {
					info.Value = v
				}
				if v, ok := m["displayName"].(string); ok {
					info.DisplayName = v
				}
				if v, ok := m["description"].(string); ok {
					info.Description = v
				}
				if info.Value != "" {
					models = append(models, info)
				}
			}
			result.Models = models
		}
	}

	// Parse "agents" array: each element has "name", "description", optional "model".
	if agentsRaw, ok := raw["agents"]; ok {
		if agentsSlice, ok := agentsRaw.([]interface{}); ok {
			agents := make([]types.AgentInfo, 0, len(agentsSlice))
			for _, item := range agentsSlice {
				m, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				agent := types.AgentInfo{}
				if v, ok := m["name"].(string); ok {
					agent.Name = v
				}
				if v, ok := m["description"].(string); ok {
					agent.Description = v
				}
				if v, ok := m["model"].(string); ok {
					agent.Model = v
				}
				if agent.Name != "" {
					agents = append(agents, agent)
				}
			}
			result.Agents = agents
		}
	}

	return result
}
