package claude

import (
	"context"
	"time"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

const (
	terminalTaskUpdateCloseGrace = 500 * time.Millisecond

	taskTypeLocalWorkflow = "local_workflow"
	taskTypeLocalAgent    = "local_agent"
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
	tasks := newActiveTaskTracker()
	var closeTimer *time.Timer
	var closeTimerC <-chan time.Time
	defer stopResponseCloseTimer(closeTimer)
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.ctx.Done():
			return
		case <-closeTimerC:
			return
		case msg, ok := <-messagesChan:
			if !ok {
				return
			}
			decision := c.forwardResponseMessage(ctx, msg, outputChan, tasks)
			var closeNow bool
			closeTimer, closeTimerC, closeNow = applyResponseForwardDecision(closeTimer, decision, tasks)
			if closeNow {
				return
			}
		}
	}
}

func (c *Client) forwardResponseMessage(
	ctx context.Context,
	msg types.Message,
	outputChan chan types.Message,
	tasks *activeTaskTracker,
) responseForwardDecision {
	select {
	case outputChan <- msg:
		return tasks.observe(msg)
	case <-ctx.Done():
		return responseForwardCloseNow
	case <-c.ctx.Done():
		return responseForwardCloseNow
	}
}

type responseForwardDecision uint8

const (
	responseForwardContinue responseForwardDecision = iota
	responseForwardCloseNow
	responseForwardCloseAfterGrace
)

type activeTaskTracker struct {
	tasks                 map[string]activeTask
	pendingWorkflowResult bool
	pendingGraceClose     bool
}

type activeTask struct {
	waitForNotification bool
	terminalUpdated     bool
	taskType            string
}

func newActiveTaskTracker() *activeTaskTracker {
	return &activeTaskTracker{tasks: make(map[string]activeTask)}
}

func (t *activeTaskTracker) observe(msg types.Message) responseForwardDecision {
	switch m := msg.(type) {
	case *types.TaskStartedMessage:
		t.add(m.TaskID, m.ToolUseID, m.TaskType)
	case *types.TaskNotificationMessage:
		task, ok := t.delete(m.TaskID, m.ToolUseID)
		if ok && t.readyToCloseAfterTask(task) {
			return responseForwardCloseNow
		}
	case *types.TaskUpdatedMessage:
		return t.observeTaskUpdated(m)
	case *types.ResultMessage:
		return t.observeResult()
	}
	return responseForwardContinue
}

func (t *activeTaskTracker) observeTaskUpdated(m *types.TaskUpdatedMessage) responseForwardDecision {
	if !isTerminalTaskStatus(m.Patch.Status) {
		return responseForwardContinue
	}
	if t.waitsForNotification(m.TaskID, m.ToolUseID) {
		t.markTerminalUpdated(m.TaskID, m.ToolUseID)
		return t.closeAfterGraceWhenTerminal()
	}
	task, ok := t.delete(m.TaskID, m.ToolUseID)
	if ok && t.readyToCloseAfterTask(task) {
		return responseForwardCloseNow
	}
	return responseForwardContinue
}

func (t *activeTaskTracker) observeResult() responseForwardDecision {
	if t.empty() {
		return responseForwardCloseNow
	}
	t.pendingWorkflowResult = t.allActiveTasksAreWorkflow()
	return t.closeAfterGraceWhenTerminal()
}

func (t *activeTaskTracker) closeAfterGraceWhenTerminal() responseForwardDecision {
	if t.pendingWorkflowResult && t.allActiveTasksTerminalUpdated() {
		t.pendingGraceClose = true
		return responseForwardCloseAfterGrace
	}
	return responseForwardContinue
}

func (t *activeTaskTracker) add(taskID string, toolUseID, taskType *string) {
	if key := taskTrackerKey(taskID, toolUseID); key != "" {
		if taskType == nil || !trackableTaskType(*taskType) {
			return
		}
		t.pendingWorkflowResult = false
		t.pendingGraceClose = false
		task := activeTask{taskType: *taskType}
		task.waitForNotification = taskShouldWaitForNotification(toolUseID, task.taskType)
		t.tasks[key] = task
	}
}

func (t *activeTaskTracker) delete(taskID string, toolUseID *string) (activeTask, bool) {
	if key := taskTrackerKey(taskID, toolUseID); key != "" {
		task, ok := t.tasks[key]
		delete(t.tasks, key)
		return task, ok
	}
	return activeTask{}, false
}

func (t *activeTaskTracker) markTerminalUpdated(taskID string, toolUseID *string) {
	if key := taskTrackerKey(taskID, toolUseID); key != "" {
		task, ok := t.tasks[key]
		if ok {
			task.terminalUpdated = true
			t.tasks[key] = task
		}
	}
}

func (t *activeTaskTracker) empty() bool {
	return len(t.tasks) == 0
}

func (t *activeTaskTracker) allActiveTasksAreWorkflow() bool {
	for _, task := range t.tasks {
		if task.taskType != taskTypeLocalWorkflow {
			return false
		}
	}
	return len(t.tasks) > 0
}

func (t *activeTaskTracker) allActiveTasksTerminalUpdated() bool {
	for _, task := range t.tasks {
		if !task.terminalUpdated {
			return false
		}
	}
	return len(t.tasks) > 0
}

func (t *activeTaskTracker) readyToCloseAfterTask(task activeTask) bool {
	return task.taskType == taskTypeLocalWorkflow && t.pendingWorkflowResult && t.empty()
}

func (t *activeTaskTracker) hasPendingGraceClose() bool {
	return t.pendingGraceClose
}

func (t *activeTaskTracker) waitsForNotification(taskID string, toolUseID *string) bool {
	task, ok := t.tasks[taskTrackerKey(taskID, toolUseID)]
	return ok && task.waitForNotification
}

func taskTrackerKey(taskID string, toolUseID *string) string {
	if taskID != "" {
		return taskID
	}
	if toolUseID != nil {
		return *toolUseID
	}
	return ""
}

func isTerminalTaskStatus(status string) bool {
	switch status {
	case "completed", "failed", "canceled":
		return true
	default:
		return status == "cancel"+"led"
	}
}

func trackableTaskType(taskType string) bool {
	return taskType == taskTypeLocalWorkflow || taskType == taskTypeLocalAgent
}

func taskShouldWaitForNotification(toolUseID *string, taskType string) bool {
	return toolUseID != nil && *toolUseID != "" &&
		(taskType == taskTypeLocalWorkflow || taskType == taskTypeLocalAgent)
}

func applyResponseForwardDecision(
	timer *time.Timer,
	decision responseForwardDecision,
	tasks *activeTaskTracker,
) (*time.Timer, <-chan time.Time, bool) {
	switch decision {
	case responseForwardCloseNow:
		return timer, nil, true
	case responseForwardCloseAfterGrace:
		stopResponseCloseTimer(timer)
		timer = time.NewTimer(terminalTaskUpdateCloseGrace)
		return timer, timer.C, false
	case responseForwardContinue:
		if timer != nil && !tasks.hasPendingGraceClose() {
			stopResponseCloseTimer(timer)
			return nil, nil, false
		}
		if timer != nil {
			return timer, timer.C, false
		}
		return nil, nil, false
	default:
		return timer, nil, false
	}
}

func stopResponseCloseTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}
