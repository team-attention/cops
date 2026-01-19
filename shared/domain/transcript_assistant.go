package domain

// AssistantTranscript represents an AI assistant response in the conversation tree.
type AssistantTranscript struct {
	TreeNodeMeta `bson:",inline"`

	RequestID string                     `json:"requestId" bson:"requestId"`
	Message   AssistantTranscriptMessage `json:"message" bson:"message"`
}

// AssistantTranscriptMessage represents the API response message.
type AssistantTranscriptMessage struct {
	Model        string                   `json:"model" bson:"model"`
	ID           string                   `json:"id" bson:"id"`
	Type         string                   `json:"type" bson:"type"` // "message"
	Role         string                   `json:"role" bson:"role"` // "assistant"
	Content      []*AssistantContentBlock `json:"content" bson:"content"`
	StopReason   *string                  `json:"stop_reason" bson:"stopReason"`
	StopSequence *string                  `json:"stop_sequence" bson:"stopSequence"`
	Usage        *AssistantUsage          `json:"usage,omitempty" bson:"usage,omitempty"`
}

// AssistantContentBlock represents a content block in assistant response.
type AssistantContentBlock struct {
	Type string `json:"type" bson:"type"` // "thinking", "text", "tool_use"

	// For type="thinking"
	Thinking  *string `json:"thinking,omitempty" bson:"thinking,omitempty"`
	Signature *string `json:"signature,omitempty" bson:"signature,omitempty"`

	// For type="text"
	Text *string `json:"text,omitempty" bson:"text,omitempty"`

	// For type="tool_use"
	ID    *string        `json:"id,omitempty" bson:"id,omitempty"`
	Name  *string        `json:"name,omitempty" bson:"name,omitempty"`
	Input map[string]any `json:"input,omitempty" bson:"input,omitempty"`
}

// AssistantUsage represents token usage statistics.
type AssistantUsage struct {
	InputTokens              int            `json:"input_tokens" bson:"inputTokens"`
	OutputTokens             int            `json:"output_tokens" bson:"outputTokens"`
	CacheCreationInputTokens int            `json:"cache_creation_input_tokens" bson:"cacheCreationInputTokens"`
	CacheReadInputTokens     int            `json:"cache_read_input_tokens" bson:"cacheReadInputTokens"`
	CacheCreation            *CacheCreation `json:"cache_creation,omitempty" bson:"cacheCreation,omitempty"`
	ServiceTier              string         `json:"service_tier,omitempty" bson:"serviceTier,omitempty"`
}

// CacheCreation represents ephemeral cache token counts.
type CacheCreation struct {
	Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens" bson:"ephemeral5MInputTokens"`
	Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens" bson:"ephemeral1HInputTokens"`
}
