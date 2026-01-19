package geminicliadapter

import (
	"encoding/json"
	"os"

	v2 "github.com/team-attention/cops/shared/domain/v2"
)

// Adapter converts Gemini CLI session logs to v2 transcript format.
type Adapter struct{}

// NewAdapter creates a new Gemini CLI adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// AdaptSession converts an entire Gemini session to v2 transcripts.
// Returns multiple transcripts (one or more per message).
func (a *Adapter) AdaptSession(session *GeminiSession) ([]*v2.Transcript, error) {
	var results []*v2.Transcript

	for _, msg := range session.Messages {
		transcripts, err := a.AdaptMessage(msg, session.SessionID)
		if err != nil {
			return nil, err
		}
		results = append(results, transcripts...)
	}

	return results, nil
}

// AdaptMessage converts a single Gemini message to v2 transcripts.
// May return multiple transcripts (e.g., AgentMessage + ToolExecutions).
func (a *Adapter) AdaptMessage(msg *GeminiMessage, sessionID string) ([]*v2.Transcript, error) {
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
