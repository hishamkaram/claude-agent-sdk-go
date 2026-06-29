package internal

import (
	"encoding/json"

	"github.com/hishamkaram/claude-agent-sdk-go/types"
)

func (q *Query) appendSessionStoreMessage(msg types.Message) error {
	if q.sessionStore == nil {
		return nil
	}
	entry, ok := sessionStoreMessageFromMessage(msg)
	if !ok {
		return nil
	}
	q.mu.Lock()
	key := q.sessionStoreKey
	if key.SessionID == "" && entry.SessionID != "" {
		key.SessionID = entry.SessionID
		q.sessionStoreKey.SessionID = entry.SessionID
	}
	q.mu.Unlock()
	if key.SessionID == "" {
		return nil
	}
	return q.sessionStore.Append(q.ctx, key, []types.SessionMessage{entry})
}

func sessionStoreMessageFromMessage(msg types.Message) (types.SessionMessage, bool) {
	raw, err := json.Marshal(msg)
	if err != nil {
		return types.SessionMessage{}, false
	}
	out := types.SessionMessage{
		Type:    msg.GetMessageType(),
		Message: json.RawMessage(raw),
	}
	switch m := msg.(type) {
	case *types.UserMessage:
		out.UUID = m.UUID
		out.SessionID = m.SessionID
		out.ParentToolUseID = m.ParentToolUseID
	case *types.AssistantMessage:
		out.UUID = m.UUID
		out.SessionID = m.SessionID
		out.ParentToolUseID = m.ParentToolUseID
	case *types.ResultMessage:
		out.UUID = m.UUID
		out.SessionID = m.SessionID
	case *types.StreamEvent:
		out.UUID = m.UUID
		out.SessionID = m.SessionID
		out.ParentToolUseID = m.ParentToolUseID
	default:
		var probe struct {
			UUID      string `json:"uuid"`
			SessionID string `json:"session_id"`
		}
		_ = json.Unmarshal(raw, &probe)
		out.UUID = probe.UUID
		out.SessionID = probe.SessionID
	}
	return out, true
}
