package domain

// ------- Type Message -------

type AssistantMessageType string

const (
	AssistantMessageTypeMessage AssistantMessageType = "message"
)

type AssistantMessageRole string

const (
	AssistantMessageRoleAssistant AssistantMessageRole = "assistant"
)

type AssistantMessageContentType string

const (
	AssistantMessageContentTypeText    AssistantMessageContentType = "text"
	AssistantMessageContentTypeTool    AssistantMessageContentType = "thinking"
	AssistantMessageContentTypeToolUse AssistantMessageContentType = "tool_use"
)

type AssistantMessageContent struct {
	Type                            AssistantMessageContentType `json:"type" bson:"type"`
	Text                            *string                     `json:"text,omitempty" bson:"text,omitempty"`
	Thinking                        *string                     `json:"thinking,omitempty" bson:"thinking,omitempty"`
	*AssistantMessageToolUseContent `json:",omitempty" bson:",inline,omitempty"`
}

type AssistantMessageToolUseContent struct {
	ID    string `json:"id" bson:"id"`
	Name  string `json:"name" bson:"name"`
	Input any    `json:"input" bson:"input"`
}

type AssistantMessageUsage struct {
	InputTokens              int `json:"inputTokens" bson:"inputTokens"`
	OutputTokens             int `json:"outputTokens" bson:"outputTokens"`
	CacheCreationInputTokens int `json:"cacheCreationInputTokens" bson:"cacheCreationInputTokens"`
	CacheReadInputTokens     int `json:"cacheReadInputTokens" bson:"cacheReadInputTokens"`
	CacheCreation            struct {
		Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens" bson:"ephemeral5MInputTokens"`
		Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens" bson:"ephemeral1HInputTokens"`
	} `json:"cacheCreation" bson:"cacheCreation"`
	ServiceTier string `json:"serviceTier" bson:"serviceTier"`
}

type AssistantMessage struct {
	Model   string                     `json:"model" bson:"model"`
	ID      string                     `json:"id" bson:"id"`
	Type    AssistantMessageType       `json:"type" bson:"type"`
	Role    AssistantMessageRole       `json:"role" bson:"role"`
	Content []*AssistantMessageContent `json:"content" bson:"content"`

	// StopReason null or string
	StopReason *string `json:"stopReason" bson:"stopReason"`
	// StopSequence null or int
	StopSequence *int                  `json:"stopSequence" bson:"stopSequence"`
	Usage        AssistantMessageUsage `json:"usage" bson:"usage"`
}

type AssistantRecord struct {
	MessageMetadata `bson:"inline"`
	RequestID       string           `json:"requestId" bson:"requestId"`
	Message         AssistantMessage `json:"message" bson:"message"`
}
