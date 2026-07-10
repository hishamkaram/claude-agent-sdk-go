package claude

import (
	"context"
	"fmt"
	"sync"

	"github.com/hishamkaram/claude-agent-sdk-go/internal"
	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/internal/transport"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// Client provides bidirectional communication with Claude Code CLI for
// interactive sessions. Client is not thread-safe; methods should be called
// from one goroutine unless the caller provides synchronization.
type Client struct {
	options   *types.ClaudeAgentOptions
	transport transport.Transport
	query     *internal.Query
	logger    *log.Logger
	cliPath   string

	mu                    sync.Mutex
	connected             bool
	connecting            bool
	closePending          bool
	ctx                   context.Context
	cancel                context.CancelFunc
	initResult            *types.InitializeResult
	permissionModes       []types.SupportedPermissionMode
	permissionModeVersion string

	recvWg sync.WaitGroup

	sessionStoreCleanup func()
}

// NewClient creates a new interactive client. It does not establish a
// connection; call Connect before sending queries.
func NewClient(ctx context.Context, options *types.ClaudeAgentOptions) (*Client, error) {
	if options == nil {
		options = types.NewClaudeAgentOptions()
	}
	if options.CanUseTool != nil && options.PermissionPromptToolName != nil {
		return nil, fmt.Errorf("can_use_tool callback cannot be used with permission_prompt_tool_name")
	}
	if options.CanUseTool != nil && options.PermissionPromptToolName == nil {
		stdio := "stdio"
		options.PermissionPromptToolName = &stdio
	}

	cliPath, err := clientCLIPath(ctx, options)
	if err != nil {
		return nil, err
	}
	cwd := clientWorkingDirectory(options)
	env := clientEnvironment(options)
	clientCtx, cancel := context.WithCancel(ctx)
	logger := log.NewLogger(options.Verbose)
	resumeID := clientResumeID(options)

	cleanupSessionStore, err := prepareSessionStoreRuntime(ctx, options, cwd, resumeID, env)
	if err != nil {
		cancel()
		return nil, err
	}

	return &Client{
		options:             options,
		transport:           transport.NewSubprocessCLITransport(cliPath, cwd, env, logger, resumeID, options),
		logger:              logger,
		cliPath:             cliPath,
		ctx:                 clientCtx,
		cancel:              cancel,
		sessionStoreCleanup: cleanupSessionStore,
	}, nil
}

func clientCLIPath(ctx context.Context, options *types.ClaudeAgentOptions) (string, error) {
	if options.CLIPath != nil {
		return *options.CLIPath, nil
	}
	return transport.FindCLI(ctx)
}

func clientWorkingDirectory(options *types.ClaudeAgentOptions) string {
	if options.CWD != nil {
		return *options.CWD
	}
	return ""
}

func clientEnvironment(options *types.ClaudeAgentOptions) map[string]string {
	env := make(map[string]string)
	if options.Env == nil {
		return env
	}
	for k, v := range options.Env {
		env[k] = v
	}
	return env
}

func clientResumeID(options *types.ClaudeAgentOptions) string {
	if options.Resume != nil && *options.Resume != "" {
		return *options.Resume
	}
	return ""
}
