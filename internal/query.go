package internal

import (
	"context"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/hishamkaram/claude-agent-sdk-go/internal/log"
	"github.com/hishamkaram/claude-agent-sdk-go/internal/transport"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// Query manages bidirectional control message handling.
// It orchestrates message routing between the transport and application callbacks,
// handling permissions, hooks, and MCP message routing.
type Query struct {
	// Transport and lifecycle
	transport transport.Transport
	ctx       context.Context
	cancel    context.CancelFunc
	logger    *log.Logger

	// Request tracking
	mu                 sync.Mutex
	requestMap         map[string]chan responseResult
	nextRequestID      int64
	hookCallbacks      map[string]types.HookCallbackFunc
	nextHookCallbackID int64

	// Options (for init protocol fields)
	options         *types.ClaudeAgentOptions
	sessionStore    types.SessionStore
	sessionStoreKey types.SessionKey

	// Callbacks
	canUseTool types.CanUseToolFunc
	hooks      map[types.HookEvent][]types.HookMatcher
	mcpServers map[string]types.MCPServer

	// Message handling
	messagesChan               chan types.Message
	messagesBackpressureWarned atomic.Bool
	closeMessagesOnce          sync.Once // guards close(messagesChan) — called from Stop() or messageLoop()
	stopOnce                   sync.Once
	stopChan                   chan struct{}
	readLoopDone               chan struct{}
	handlerWg                  sync.WaitGroup // tracks in-flight handleControlRequest goroutines
	started                    bool
	initialized                bool
	initializeResult           map[string]interface{}
	isStreamingMode            bool
}

// responseResult wraps the response or error from a control request.
type responseResult struct {
	response map[string]interface{}
	err      error
}

// NewQuery creates a new Query handler.
func NewQuery(ctx context.Context, tr transport.Transport, opts *types.ClaudeAgentOptions, logger *log.Logger, isStreamingMode bool) *Query {
	queryCtx, cancel := context.WithCancel(ctx)

	q := &Query{
		transport:       tr,
		ctx:             queryCtx,
		cancel:          cancel,
		logger:          logger,
		requestMap:      make(map[string]chan responseResult),
		hookCallbacks:   make(map[string]types.HookCallbackFunc),
		messagesChan:    make(chan types.Message, 100),
		stopChan:        make(chan struct{}),
		readLoopDone:    make(chan struct{}),
		isStreamingMode: isStreamingMode,
		mcpServers:      make(map[string]types.MCPServer),
	}

	if opts != nil {
		q.options = opts
		q.canUseTool = opts.CanUseTool
		q.hooks = opts.Hooks
		q.sessionStore = opts.SessionStore
		if opts.SessionStoreKey != nil {
			q.sessionStoreKey = *opts.SessionStoreKey
		}
	}

	return q
}

// Initialize sends initialization control request if in streaming mode.
func (q *Query) Initialize(ctx context.Context) (map[string]interface{}, error) {
	if !q.isStreamingMode {
		return nil, nil
	}

	if q.initialized {
		return q.initializeResult, nil
	}

	q.logger.Debug("Initializing control protocol...")

	request := q.buildInitializeRequest(q.buildHooksConfig())

	result, err := q.sendControlRequest(ctx, request)
	if err != nil {
		q.clearHookCallbacks()
		q.logger.Error("control protocol initialization failed", zap.Error(err))
		return nil, types.NewControlProtocolErrorWithCause("initialization failed", err)
	}

	q.initialized = true
	q.initializeResult = result
	q.logger.Debug("Control protocol initialized successfully")
	return result, nil
}
