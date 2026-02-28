package opencodeadapter

import (
	"fmt"

	session "github.com/team-attention/cops/shared/domain/v2"
)

// Adapter converts OpenCode messages to v2 session format.
type Adapter struct{}

// NewAdapter creates a new OpenCode adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// AdaptMessage converts a single OpenCode message to v2 sessions.
// May return multiple sessions (e.g., AgentMessage + ToolExecutions).
//
// Returns error for structural failures including:
//   - Malformed JSON in the parts field (ParseParts error)
//
// Internal conversion methods (adaptUserMessage, adaptAssistantMessage) return
// only a session slice without error because they operate on already-parsed
// Go structs where no I/O or parsing can fail. This matches the Gemini adapter
// pattern where AdaptMessage returns error but adaptUserMessage/adaptGeminiMessage do not.
func (a *Adapter) AdaptMessage(msg *OpenCodeMessage) ([]*session.Session, error) {
	parts, err := msg.ParseParts()
	if err != nil {
		return nil, fmt.Errorf("adapt message %s: %w", msg.ID, err)
	}

	switch msg.Role {
	case "user":
		return a.adaptUserMessage(msg, parts), nil
	case "assistant":
		return a.adaptAssistantMessage(msg, parts), nil
	default:
		return nil, nil
	}
}

// AdaptBatch converts multiple OpenCode messages to v2 sessions.
// Stops on the first error, matching the pattern of claudecodeadapter.AdaptBatch
// and geminicliadapter.AdaptSession. Per-event error collection is handled by
// the session service's convertEventsToSessions, which calls AdaptMessage
// individually for each event and tracks failedIDs.
func (a *Adapter) AdaptBatch(messages []*OpenCodeMessage) ([]*session.Session, error) {
	var result []*session.Session
	for _, msg := range messages {
		adapted, err := a.AdaptMessage(msg)
		if err != nil {
			return nil, err
		}
		result = append(result, adapted...)
	}
	return result, nil
}
