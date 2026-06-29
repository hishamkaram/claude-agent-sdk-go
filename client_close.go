package claude

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Close gracefully terminates the Claude session and cleans up resources.
func (c *Client) Close(ctx context.Context) error {
	c.mu.Lock()
	if !c.connected {
		if c.connecting {
			c.closePending = true
			if c.cancel != nil {
				c.cancel()
			}
			c.logger.Info("Close requested during Connect — flagged for cleanup")
			c.mu.Unlock()
			return nil
		}
		c.mu.Unlock()
		c.cleanupSessionStoreRuntime()
		return nil
	}

	c.logger.Info("Closing Claude connection...")
	errs := c.stopConnectedRuntime(ctx)
	c.connected = false
	c.mu.Unlock()

	c.waitForReceivers()
	c.logger.Debug("Connection closed")
	c.cleanupSessionStoreRuntime()
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (c *Client) stopConnectedRuntime(ctx context.Context) []error {
	var errs []error
	if c.query != nil {
		if err := c.query.Stop(ctx); err != nil {
			c.logger.Warn("error stopping query handler", zap.Error(err))
			errs = append(errs, err)
		}
		c.query = nil
	}
	if c.transport != nil {
		if err := c.transport.Close(ctx); err != nil {
			c.logger.Warn("error closing transport", zap.Error(err))
			errs = append(errs, err)
		}
	}
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	return errs
}

func (c *Client) waitForReceivers() {
	recvDone := make(chan struct{})
	go func() {
		c.recvWg.Wait()
		close(recvDone)
	}()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-recvDone:
	case <-timer.C:
		c.logger.Warn("timed out waiting for ReceiveResponse goroutines to exit")
	}
}

func (c *Client) cleanupSessionStoreRuntime() {
	c.mu.Lock()
	cleanup := c.sessionStoreCleanup
	c.sessionStoreCleanup = nil
	c.mu.Unlock()
	if cleanup != nil {
		cleanup()
	}
}
