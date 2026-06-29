package claude

import (
	"context"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

// ReceiveResponse returns a channel of response messages from Claude.
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

		c.forwardResponseMessages(ctx, messagesChan, outputChan)
	}()

	return outputChan
}

func (c *Client) forwardResponseMessages(
	ctx context.Context,
	messagesChan <-chan types.Message,
	outputChan chan types.Message,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.ctx.Done():
			return
		case msg, ok := <-messagesChan:
			if !ok {
				return
			}
			if c.forwardResponseMessage(ctx, msg, outputChan) {
				return
			}
		}
	}
}

func (c *Client) forwardResponseMessage(
	ctx context.Context,
	msg types.Message,
	outputChan chan types.Message,
) bool {
	select {
	case outputChan <- msg:
		_, isResult := msg.(*types.ResultMessage)
		return isResult
	case <-ctx.Done():
		return true
	case <-c.ctx.Done():
		return true
	}
}
