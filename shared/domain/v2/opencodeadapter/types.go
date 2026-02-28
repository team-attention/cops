// Package opencodeadapter provides conversion from OpenCode session data to v2 session format.
//
// # Data Flow
//
// The daemon's polling handler reads from the OpenCode SQLite database:
//
//	~/.local/share/opencode/opencode.db
//
// It queries the `messages` table and serializes each row as JSON:
//
//	{
//	  "id": "msg_xxx",
//	  "sessionId": "sess_xxx",
//	  "role": "user" | "assistant",
//	  "parts": "[{\"type\":\"text\",\"text\":\"hello\"}]",
//	  "model": "claude-sonnet-4-20250514",
//	  "createdAt": 1700000000,
//	  "updatedAt": 1700000005,
//	  "finishedAt": 1700000010
//	}
//
// The `parts` field is a JSON string (not a parsed array) because it is stored
// as TEXT in SQLite and serialized as-is by the daemon.
//
// # Type Mapping: OpenCode -> v2
//
//	| OpenCode Field      | v2 Type                 | Notes                                |
//	|---------------------|-------------------------|--------------------------------------|
//	| role="user"         | HumanMessage            | User input text                      |
//	| role="assistant"    | AgentMessage            | AI response text                     |
//	| model               | AgentMessage.Model      | Model identifier                     |
//	| createdAt           | TreeNodeMeta.Timestamp  | Message creation timestamp (unix)    |
//	| parts (parsed)      | Content blocks + Tools  | Parsed from JSON string              |
//
// # Result Field Design
//
// OpenCodePart.Result uses `any` type to match the established adapter pattern
// (Gemini's GeminiToolCall.Result uses `[]any`). Go's json.Unmarshal into `any`
// automatically discriminates types: string -> string, number -> float64,
// object -> map[string]any, array -> []any. This eliminates the need for a
// separate parseResultContent helper function.
package opencodeadapter

import (
	"encoding/json"
	"fmt"
)

// OpenCodeMessage represents a single message as serialized by the daemon's polling handler.
// Fields match the JSON produced by daemon/internal/service/logwatcher/inbound/worker/polling/handler.go.
// JSON tags use camelCase to match the daemon's openCodeMessage struct.
//
// v2 Mapping by Role:
//   - role="user"      -> HumanMessage (text parts -> HumanContentBlock, tool-result parts -> ToolExecution)
//   - role="assistant"  -> AgentMessage (text parts -> AgentContentBlock, tool-invocation parts -> ToolExecution)
type OpenCodeMessage struct {
	ID         string `json:"id"`
	SessionID  string `json:"sessionId"`
	Role       string `json:"role"`  // "user", "assistant"
	Parts      string `json:"parts"` // JSON string containing array of OpenCodePart
	Model      string `json:"model"`
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`
	FinishedAt *int64 `json:"finishedAt,omitempty"`
}

// OpenCodePart represents a structured content part parsed from the Parts JSON string.
// This is a discriminated union keyed by the Type field:
//   - type="text":            Text is set
//   - type="tool-invocation": ToolID, Name, Args, State are set; Result set when State="result"
//   - type="tool-result":     ToolID, Name, Result are set
//
// Discriminated union member fields use pointer types per go-struct.md to distinguish
// "absent" (nil) from "empty string" (""). The Type field is always present (required)
// and uses a value type.
//
// The Result field uses `any` type for consistency with the Gemini adapter pattern.
// Go's json.Unmarshal into `any` produces the correct Go type automatically
// (string, float64, map[string]any, []any, bool, nil).
type OpenCodePart struct {
	Type   string         `json:"type"`                       // "text", "tool-invocation", "tool-result"
	Text   *string        `json:"text,omitempty"`             // Set for type="text"
	ToolID *string        `json:"toolInvocationId,omitempty"` // Set for type="tool-invocation", "tool-result"
	Name   *string        `json:"toolName,omitempty"`         // Set for type="tool-invocation", "tool-result"
	Args   map[string]any `json:"args,omitempty"`
	Result any            `json:"result,omitempty"`
	State  *string        `json:"state,omitempty"` // "result", "call", "partial-call"; set for type="tool-invocation"
}

// ParseParts parses the Parts JSON string into a slice of OpenCodePart.
// Returns an error if the JSON is malformed, allowing callers to report
// conversion failures through the session service's error collection mechanism
// rather than silently producing empty messages.
// Returns (nil, nil) if Parts is empty.
func (m *OpenCodeMessage) ParseParts() ([]*OpenCodePart, error) {
	if m == nil || m.Parts == "" {
		return nil, nil
	}
	var parts []*OpenCodePart
	if err := json.Unmarshal([]byte(m.Parts), &parts); err != nil {
		return nil, fmt.Errorf("parse parts JSON: %w", err)
	}
	return parts, nil
}
