package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/internal"
	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/internal/transport"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// Query executes a single Claude query in non-streaming mode and returns a channel of messages.
// This is the simplest way to interact with Claude for one-off questions or batch processing.
//
// The function:
//   - Finds and connects to Claude Code CLI
//   - Sends the prompt in non-streaming mode (--print flag)
//   - Streams response messages to the returned channel
//   - Automatically cleans up resources when done
//
// The returned channel is read-only and will be closed when:
//   - All messages have been received (including the final ResultMessage)
//   - An error occurs
//   - The context is canceled
//
// Error handling:
//   - Connection errors are returned immediately
//   - Parse errors during message reading are sent to options.OnError callback if provided
//   - Context cancellation is respected throughout
//
// Example usage:
//
//	ctx := context.Background()
//	opts := types.NewClaudeAgentOptions()
//	messages, err := Query(ctx, "What is 2+2?", opts)
//	if err != nil {
//	    if types.IsCLINotFoundError(err) {
//	        log.Fatal("Claude CLI not installed")
//	    }
//	    log.Fatal(err)
//	}
//
//	for msg := range messages {
//	    switch m := msg.(type) {
//	    case *types.AssistantMessage:
//	        for _, block := range m.Content {
//	            if tb, ok := block.(*types.TextBlock); ok {
//	                fmt.Println(tb.Text)
//	            }
//	        }
//	    case *types.ResultMessage:
//	        fmt.Printf("Cost: $%.4f\n", *m.TotalCostUSD)
//	    }
//	}
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - prompt: The text prompt to send to Claude
//   - options: Configuration options (nil uses defaults)
//
// Returns:
//   - A read-only channel of Message types
//   - An error if connection or initialization fails
func Query(ctx context.Context, prompt string, options *types.ClaudeAgentOptions) (<-chan types.Message, error) {
	if options == nil {
		options = types.NewClaudeAgentOptions()
	}
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	runtime, err := prepareQueryRuntime(ctx, options)
	if err != nil {
		return nil, err
	}

	transportInst := transport.NewSubprocessCLITransport(runtime.cliPath, runtime.cwd, runtime.env, runtime.logger, runtime.resumeID, options)
	if connectErr := transportInst.Connect(ctx); connectErr != nil {
		runtime.cleanupSessionStore()
		return nil, types.NewCLIConnectionErrorWithCause("failed to connect to Claude CLI", connectErr)
	}

	queryHandler := internal.NewQuery(ctx, transportInst, options, runtime.logger, false)
	if startErr := queryHandler.Start(ctx); startErr != nil {
		_ = transportInst.Close(ctx)
		runtime.cleanupSessionStore()
		return nil, startErr
	}

	if err := sendQueryPrompt(ctx, queryHandler, transportInst, prompt, runtime.resumeID); err != nil {
		return nil, err
	}

	outputChan := make(chan types.Message, 10)
	go forwardQueryMessages(ctx, queryHandler, transportInst, runtime.cleanupSessionStore, outputChan)

	return outputChan, nil
}

type queryRuntime struct {
	cliPath             string
	cwd                 string
	env                 map[string]string
	logger              *log.Logger
	resumeID            string
	cleanupSessionStore func()
}

func prepareQueryRuntime(ctx context.Context, options *types.ClaudeAgentOptions) (*queryRuntime, error) {
	cliPath, err := resolveCLIPath(ctx, options)
	if err != nil {
		return nil, err
	}
	cwd := ""
	if options.CWD != nil {
		cwd = *options.CWD
	}
	env := copyOptionsEnv(options)
	logger := log.NewLogger(options != nil && options.Verbose)
	resumeID := ""
	if options.Resume != nil && *options.Resume != "" {
		resumeID = *options.Resume
	}
	cleanupSessionStore, err := prepareSessionStoreRuntime(ctx, options, cwd, resumeID, env)
	if err != nil {
		return nil, err
	}
	return &queryRuntime{
		cliPath:             cliPath,
		cwd:                 cwd,
		env:                 env,
		logger:              logger,
		resumeID:            resumeID,
		cleanupSessionStore: cleanupSessionStore,
	}, nil
}

func sendQueryPrompt(
	ctx context.Context,
	queryHandler *internal.Query,
	transportInst *transport.SubprocessCLITransport,
	prompt string,
	resumeID string,
) error {
	data, err := json.Marshal(buildQueryMessage(prompt, querySessionID(resumeID)))
	if err != nil {
		_ = queryHandler.Stop(ctx)
		_ = transportInst.Close(ctx)
		return types.NewControlProtocolErrorWithCause("failed to marshal query", err)
	}
	if err := transportInst.Write(ctx, string(data)); err != nil {
		_ = queryHandler.Stop(ctx)
		_ = transportInst.Close(ctx)
		return err
	}
	return nil
}

func buildQueryMessage(prompt, sessionID string) map[string]interface{} {
	return map[string]interface{}{
		"type": "user",
		"message": map[string]interface{}{
			"role":    "user",
			"content": prompt,
		},
		"parent_tool_use_id": nil,
		"session_id":         sessionID,
	}
}

func querySessionID(resumeID string) string {
	if resumeID != "" {
		return resumeID
	}
	return "default-session"
}

// resolveCLIPath returns the configured CLI path, or discovers it on PATH.
func resolveCLIPath(ctx context.Context, options *types.ClaudeAgentOptions) (string, error) {
	if options.CLIPath != nil {
		return *options.CLIPath, nil
	}
	return transport.FindCLI(ctx)
}

// copyOptionsEnv returns an independent copy of the options' environment map.
func copyOptionsEnv(options *types.ClaudeAgentOptions) map[string]string {
	env := make(map[string]string)
	if options.Env != nil {
		for k, v := range options.Env {
			env[k] = v
		}
	}
	return env
}

// forwardQueryMessages drains the query handler's messages into outputChan until
// a result has arrived and no active Claude task remains, the source channel
// closes, or ctx is canceled.
// It then closes outputChan and tears down the handler, transport, and
// session-store temp dir. All dependencies are passed in (no closure capture).
func forwardQueryMessages(
	ctx context.Context,
	queryHandler *internal.Query,
	transportInst *transport.SubprocessCLITransport,
	cleanupSessionStore func(),
	outputChan chan types.Message,
) {
	defer close(outputChan)
	defer func() {
		_ = queryHandler.Stop(ctx)
		_ = transportInst.Close(ctx)
		cleanupSessionStore()
	}()

	messagesChan := queryHandler.GetMessages(ctx)
	tasks := newActiveTaskTracker()
	var closeTimer *time.Timer
	var closeTimerC <-chan time.Time
	defer stopResponseCloseTimer(closeTimer)

	for {
		select {
		case <-ctx.Done():
			return
		case <-closeTimerC:
			return
		case msg, ok := <-messagesChan:
			if !ok {
				// Messages channel closed
				return
			}

			// Forward message to output
			select {
			case outputChan <- msg:
				decision := tasks.observe(msg)
				var closeNow bool
				closeTimer, closeTimerC, closeNow = applyResponseForwardDecision(closeTimer, decision, tasks)
				if closeNow {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}
}
