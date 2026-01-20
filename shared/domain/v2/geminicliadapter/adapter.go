package geminicliadapter

import (
	"encoding/json"
	"os"

	session "github.com/team-attention/cops/shared/domain/v2"
)

// Adapter converts Gemini CLI session logs to v2 session format.
type Adapter struct{}

// NewAdapter creates a new Gemini CLI adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// AdaptSession converts an entire Gemini session to v2 sessions.
// Returns multiple sessions (one or more per message).
func (a *Adapter) AdaptSession(sess *GeminiSession) ([]*session.Session, error) {
	var results []*session.Session

	for _, msg := range sess.Messages {
		sessions, err := a.AdaptMessage(msg, sess.SessionID)
		if err != nil {
			return nil, err
		}
		results = append(results, sessions...)
	}

	return results, nil
}

// AdaptMessage converts a single Gemini message to v2 sessions.
// May return multiple sessions (e.g., AgentMessage + ToolExecutions).
func (a *Adapter) AdaptMessage(msg *GeminiMessage, sessionID string) ([]*session.Session, error) {
	switch msg.Type {
	case "user":
		return a.adaptUserMessage(msg, sessionID), nil
	case "gemini":
		return a.adaptGeminiMessage(msg, sessionID), nil
	case "info":
		return a.adaptInfoMessage(msg, sessionID), nil
	default:
		return nil, nil
	}
}

// ParseSessionFile reads and parses a Gemini session JSON file.
func ParseSessionFile(path string) (*GeminiSession, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var session GeminiSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}
