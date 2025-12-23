package domain

import "time"

// SessionType represents the type of session message.
type SessionType string

const (
	SessionTypeUser                SessionType = "user"
	SessionTypeAssistant           SessionType = "assistant"
	SessionTypeSystem              SessionType = "system"
	SessionTypeSummary             SessionType = "summary"
	SessionTypeFileHistorySnapshot SessionType = "file-history-snapshot"
	SessionTypeQueueOperation      SessionType = "queue-operation"
)

// SessionRecord represents a single entry from Claude Code JSONL.
type SessionRecord struct {
	UUID          string         `json:"uuid"`
	ParentUUID    string         `json:"parentUuid"`
	SessionID     string         `json:"sessionId"`
	Type          SessionType    `json:"type"`
	Timestamp     time.Time      `json:"timestamp"`
	CWD           string         `json:"cwd"`
	GitBranch     string         `json:"gitBranch"`
	Version       string         `json:"version"`
	UserType      string         `json:"userType"`
	IsSidechain   bool           `json:"isSidechain"`
	IsMeta        bool           `json:"isMeta,omitempty"`
	Slug          string         `json:"slug,omitempty"`
	RequestID     string         `json:"requestId,omitempty"`
	Message       *Message       `json:"message,omitempty"`
	ToolUseResult *ToolUseResult `json:"toolUseResult,omitempty"`
}

// Usage contains token usage information (snake_case to match actual logs).
type Usage struct {
	InputTokens              int            `json:"input_tokens"`
	OutputTokens             int            `json:"output_tokens"`
	CacheCreationInputTokens int            `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int            `json:"cache_read_input_tokens"`
	CacheCreation            *CacheCreation `json:"cache_creation,omitempty"`
	ServiceTier              string         `json:"service_tier,omitempty"`
}

// CacheCreation contains ephemeral cache token counts.
type CacheCreation struct {
	Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
	Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
}

// ToolUseResult contains metadata about tool execution results.
type ToolUseResult struct {
	Type string       `json:"type"`
	File *ToolUseFile `json:"file,omitempty"`
}

// ToolUseFile contains file metadata from tool results.
type ToolUseFile struct {
	FilePath   string `json:"filePath"`
	Content    string `json:"content"`
	NumLines   int    `json:"numLines"`
	StartLine  int    `json:"startLine"`
	TotalLines int    `json:"totalLines"`
}
