// Package geminicliadapter provides conversion from Gemini CLI session logs to v2 transcript format.
//
// # Gemini CLI Log Location
//
// Gemini CLI produces session JSON files at:
//
//	~/.gemini/tmp/{project_hash}/chats/session-*.json
//
// # Type Mapping: Gemini → v2
//
// The following table shows how Gemini CLI types map to v2 transcript types:
//
//	| Gemini Field/Type       | v2 Type                              | Notes                                    |
//	|-------------------------|--------------------------------------|------------------------------------------|
//	| Message.Type="user"     | HumanMessage                         | User input text                          |
//	| Message.Type="gemini"   | AgentMessage + ToolExecution[]       | Agent response with optional tool calls  |
//	| Message.Type="info"     | SystemMessage (subtype=info)         | System-level informational message       |
//	| Message.Thoughts[]      | AgentContentBlock (type=thinking)    | Extended thinking blocks                 |
//	| Message.ToolCalls[]     | ToolExecution                        | Tool call + result combined              |
//	| Message.Tokens          | TokenUsage                           | Token usage statistics                   |
//	| ToolCall.Status         | ToolResultStatus                     | success→Success, error→Error, cancelled→Skipped |
//
// # Key Difference from Claude Code
//
// In Claude Code, tool calls are split across two messages:
//   - tool_use block in assistant message (call)
//   - tool_result block in user message (result)
//
// In Gemini CLI, tool calls and results are combined in a single toolCalls entry,
// which maps more naturally to v2's ToolExecution type.
package geminicliadapter

import (
	"time"
)

// GeminiSession represents the root structure of a Gemini CLI session log.
//
// v2 Mapping:
//   - SessionID → TreeNodeMeta.SessionID (propagated to all transcripts)
//   - Messages  → []Transcript (each message produces one or more transcripts)
type GeminiSession struct {
	SessionID   string           `json:"sessionId"`
	ProjectHash string           `json:"projectHash"`
	StartTime   time.Time        `json:"startTime"`
	LastUpdated time.Time        `json:"lastUpdated"`
	Messages    []*GeminiMessage `json:"messages"`
}

// GeminiMessage represents a single message in a Gemini CLI session.
// The Type field discriminates between user, gemini, and info messages.
//
// v2 Mapping by Type:
//   - Type="user"   → HumanMessage (Content → HumanContentBlock with type=text)
//   - Type="gemini" → AgentMessage + ToolExecution[] (one per ToolCalls entry)
//   - Type="info"   → SystemMessage with Subtype=info
//
// Field Mapping for Type="gemini":
//   - ID        → TreeNodeMeta.UUID, AgentMessage.UUID
//   - Timestamp → TreeNodeMeta.Timestamp
//   - Content   → AgentContentBlock (type=text)
//   - Thoughts  → AgentContentBlock[] (type=thinking)
//   - Tokens    → AgentMessage.Usage (TokenUsage)
//   - Model     → AgentMessage.Model
//   - ToolCalls → ToolExecution[] + AgentContentBlock[] (type=tool_call_ref)
type GeminiMessage struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	Type      string            `json:"type"` // "user", "gemini", "info"
	Content   string            `json:"content"`
	Thoughts  []*GeminiThought  `json:"thoughts,omitempty"`
	Tokens    *GeminiTokens     `json:"tokens,omitempty"`
	Model     string            `json:"model,omitempty"`
	ToolCalls []*GeminiToolCall `json:"toolCalls,omitempty"`
}

// GeminiThought represents an extended thinking block from Gemini.
//
// v2 Mapping:
//   - → AgentContentBlock (type=thinking)
//   - Subject + Description → ThinkingBlock.Content (concatenated with newline)
type GeminiThought struct {
	Subject     string    `json:"subject"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

// GeminiTokens represents token usage statistics from Gemini.
//
// v2 Mapping:
//   - → TokenUsage
//   - Input  → TokenUsage.InputTokens
//   - Output → TokenUsage.OutputTokens
//   - Cached → TokenUsage.CacheReadInputTokens
//   - Thoughts, Tool, Total → not mapped (Gemini-specific)
type GeminiTokens struct {
	Input    int `json:"input"`
	Output   int `json:"output"`
	Cached   int `json:"cached"`
	Thoughts int `json:"thoughts"`
	Tool     int `json:"tool"`
	Total    int `json:"total"`
}

// GeminiToolCall represents a tool call with its result in Gemini CLI format.
// Unlike Claude Code where tool_use and tool_result are separate, Gemini combines them.
//
// v2 Mapping:
//   - → ToolExecution
//   - ID        → ToolExecution.ID, ToolExecution.TreeNodeMeta.UUID
//   - Name      → ToolExecution.ToolName
//   - Args      → ToolExecution.Input
//   - Result    → ToolExecution.Result.Content
//   - Status    → ToolExecution.Result.Status (success→Success, error→Error, cancelled→Skipped)
//   - Timestamp → ToolExecution.TreeNodeMeta.Timestamp
//   - ResultDisplay → ToolExecution.Result.Error (when Status="error")
//   - DisplayName   → not mapped (Gemini-specific display hint)
type GeminiToolCall struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Args          map[string]any `json:"args"`
	Result        []any          `json:"result"`
	Status        string         `json:"status"` // "success", "error", "cancelled"
	Timestamp     time.Time      `json:"timestamp"`
	ResultDisplay string         `json:"resultDisplay,omitempty"`
	DisplayName   string         `json:"displayName,omitempty"`
}
