package claude

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/hishamkaram/claude-agent-sdk-go/internal"
	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// Connect establishes a streaming connection to Claude Code CLI.
func (c *Client) Connect(ctx context.Context) error {
	if err := c.beginConnect(); err != nil {
		return err
	}

	connectSucceeded := false
	defer func() {
		c.finishConnect(connectSucceeded)
	}()

	c.logger.Info("Connecting to Claude CLI...")
	if err := c.connectTransport(ctx); err != nil {
		return err
	}

	query, initResult, err := c.startInitializedQuery(ctx)
	if err != nil {
		return err
	}

	if err := c.commitConnect(ctx, query, initResult); err != nil {
		return err
	}

	c.logger.Info("Successfully connected to Claude")
	connectSucceeded = true
	return nil
}

func (c *Client) beginConnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.connected {
		return types.NewControlProtocolError("client already connected")
	}
	if c.connecting {
		return types.NewControlProtocolError("client is already connecting")
	}
	c.connecting = true
	return nil
}

func (c *Client) finishConnect(connectSucceeded bool) {
	c.mu.Lock()
	c.connecting = false
	c.closePending = false
	c.mu.Unlock()
	if !connectSucceeded {
		c.cleanupSessionStoreRuntime()
	}
}

func (c *Client) connectTransport(ctx context.Context) error {
	if err := c.transport.Connect(ctx); err != nil {
		c.logger.Error("failed to connect transport", zap.Error(err))
		return types.NewCLIConnectionErrorWithCause("failed to connect to Claude CLI", err)
	}
	c.logger.Debug("Transport connected successfully")

	select {
	case <-c.ctx.Done():
		_ = c.transport.Close(ctx)
		return types.NewControlProtocolErrorWithCause("client connect canceled", c.ctx.Err())
	default:
		if err := c.transport.GetError(); err != nil {
			c.logger.Error("transport error detected during connection", zap.Error(err))
			_ = c.transport.Close(ctx)
			return err
		}
	}
	return nil
}

func (c *Client) startInitializedQuery(
	ctx context.Context,
) (*internal.Query, *types.InitializeResult, error) {
	query := internal.NewQuery(ctx, c.transport, c.options, c.logger, true)
	c.logger.Debug("Query handler created")

	if err := query.Start(ctx); err != nil {
		c.logger.Error("failed to start message processing", zap.Error(err))
		_ = c.transport.Close(ctx)
		return nil, nil, fmt.Errorf("client.Connect: start message processing: %w", err)
	}
	c.logger.Debug("Message processing started")

	initRaw, err := query.Initialize(ctx)
	if err != nil {
		c.logger.Error("failed to initialize control protocol", zap.Error(err))
		c.cleanupFailedInitialize(ctx, query)
		return nil, nil, types.NewControlProtocolErrorWithCause(
			"failed to initialize control protocol",
			err,
		)
	}
	c.logger.Debug("Control protocol initialized")
	return query, parseInitResult(initRaw), nil
}

func (c *Client) cleanupFailedInitialize(ctx context.Context, query *internal.Query) {
	cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cleanupCancel()
	_ = query.Stop(cleanupCtx)
	_ = c.transport.Close(cleanupCtx)
}

func (c *Client) commitConnect(
	ctx context.Context,
	query *internal.Query,
	initResult *types.InitializeResult,
) error {
	c.mu.Lock()
	if c.closePending {
		c.closePending = false
		c.mu.Unlock()
		c.logger.Info("Connect completed but Close was requested — cleaning up")
		_ = query.Stop(ctx)
		_ = c.transport.Close(ctx)
		return types.NewControlProtocolError("client closed during connect")
	}
	c.query = query
	c.initResult = initResult
	c.permissionModes = nil
	c.permissionModeVersion = ""
	c.connected = true
	c.mu.Unlock()
	return nil
}
