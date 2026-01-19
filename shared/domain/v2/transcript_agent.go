package v2

// AgentContentBlockType represents the type of content block in an agent message.
type AgentContentBlockType string

const (
	AgentContentBlockTypeText        AgentContentBlockType = "text"
	AgentContentBlockTypeThinking    AgentContentBlockType = "thinking"
	AgentContentBlockTypeToolCallRef AgentContentBlockType = "tool_call_ref"
)

// AgentMessage represents an AI agent response in the conversation tree.
// Contains text, thinking blocks, and references to tool calls.
// Actual tool calls and results are stored in ToolExecution.
type AgentMessage struct {
	TreeNodeMeta `bson:",inline"`

	Provider   string               `json:"provider" bson:"provider"`     // e.g., "anthropic", "openai"
	Model      string               `json:"model" bson:"model"`           // e.g., "claude-sonnet-4-20250514"
	RequestID  string               `json:"requestId" bson:"requestId"`   // Provider's response ID
	Content    []*AgentContentBlock `json:"content" bson:"content"`       // Text, thinking, tool_call_ref blocks
	StopReason *string              `json:"stopReason,omitempty" bson:"stopReason,omitempty"`
	Usage      *TokenUsage          `json:"usage,omitempty" bson:"usage,omitempty"`
}

// AgentContentBlock represents a content block in an agent response.
// Supports text, thinking, and tool_call_ref types.
type AgentContentBlock struct {
	Type AgentContentBlockType `json:"type" bson:"type"`

	// For type="text"
	Text *string `json:"text,omitempty" bson:"text,omitempty"`

	// For type="thinking"
	Thinking *ThinkingBlock `json:"thinking,omitempty" bson:"thinking,omitempty"`

	// For type="tool_call_ref"
	ToolCallRef *ToolCallReference `json:"toolCallRef,omitempty" bson:"toolCallRef,omitempty"`
}

// ThinkingBlock represents extended thinking content from the agent.
type ThinkingBlock struct {
	Content   string  `json:"content" bson:"content"`
	Signature *string `json:"signature,omitempty" bson:"signature,omitempty"` // For verification
}

// ToolCallReference is a reference to a ToolExecution entry.
// The actual tool call details are in the referenced ToolExecution.
type ToolCallReference struct {
	ToolExecutionID string `json:"toolExecutionId" bson:"toolExecutionId"` // References ToolExecution.ID
	ToolName        string `json:"toolName" bson:"toolName"`               // For quick display without lookup
}

// TokenUsage represents provider-agnostic token usage statistics.
type TokenUsage struct {
	InputTokens  int `json:"inputTokens" bson:"inputTokens"`
	OutputTokens int `json:"outputTokens" bson:"outputTokens"`

	// Cache-related metrics (optional, provider-specific)
	CacheCreationInputTokens int `json:"cacheCreationInputTokens,omitempty" bson:"cacheCreationInputTokens,omitempty"`
	CacheReadInputTokens     int `json:"cacheReadInputTokens,omitempty" bson:"cacheReadInputTokens,omitempty"`

	// Service tier info (optional)
	ServiceTier string `json:"serviceTier,omitempty" bson:"serviceTier,omitempty"`
}
