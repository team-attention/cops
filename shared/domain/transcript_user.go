package domain

// UserTranscript represents a user input entry in the conversation tree.
type UserTranscript struct {
	TreeNodeMeta `bson:",inline"`

	Message                 UserTranscriptMessage `json:"message" bson:"message"`
	IsMeta                  bool                  `json:"isMeta,omitempty" bson:"isMeta,omitempty"`
	ThinkingMetadata        *ThinkingMetadata     `json:"thinkingMetadata,omitempty" bson:"thinkingMetadata,omitempty"`
	Todos                   []*Todo               `json:"todos,omitempty" bson:"todos,omitempty"`
	ToolUseResult           *ToolUseResult        `json:"toolUseResult,omitempty" bson:"toolUseResult,omitempty"`
	SourceToolAssistantUUID *string               `json:"sourceToolAssistantUUID,omitempty" bson:"sourceToolAssistantUUID,omitempty"`
}

// UserTranscriptMessage represents the message content of a user entry.
type UserTranscriptMessage struct {
	Role    string `json:"role" bson:"role"`       // Always "user"
	Content any    `json:"content" bson:"content"` // string or []*UserMessageBlock
}

// UserMessageBlock represents a block in array-form user message content.
type UserMessageBlock struct {
	Type      string       `json:"type" bson:"type"` // "text", "tool_result", "image"
	Text      *string      `json:"text,omitempty" bson:"text,omitempty"`
	ToolUseID *string      `json:"tool_use_id,omitempty" bson:"toolUseId,omitempty"`
	Content   any          `json:"content,omitempty" bson:"content,omitempty"` // For tool_result
	Source    *ImageSource `json:"source,omitempty" bson:"source,omitempty"`   // For image
}

// ImageSource represents base64 image data.
type ImageSource struct {
	Type      string `json:"type" bson:"type"` // "base64"
	MediaType string `json:"media_type" bson:"mediaType"`
	Data      string `json:"data" bson:"data"`
}

// ThinkingMetadata contains extended thinking configuration.
type ThinkingMetadata struct {
	Level    string             `json:"level" bson:"level"` // "high", "medium", "low"
	Disabled bool               `json:"disabled" bson:"disabled"`
	Triggers []*ThinkingTrigger `json:"triggers,omitempty" bson:"triggers,omitempty"`
}

// ThinkingTrigger represents a trigger keyword for extended thinking.
type ThinkingTrigger struct {
	Start int    `json:"start" bson:"start"`
	End   int    `json:"end" bson:"end"`
	Text  string `json:"text" bson:"text"`
}

// Todo represents a task item.
type Todo struct {
	Content    string `json:"content" bson:"content"`
	Status     string `json:"status" bson:"status"` // "pending", "in_progress", "completed"
	ActiveForm string `json:"activeForm" bson:"activeForm"`
}

// ToolUseResult represents the result of a tool/command execution.
type ToolUseResult struct {
	Success     bool    `json:"success" bson:"success"`
	CommandName *string `json:"commandName,omitempty" bson:"commandName,omitempty"`
	Model       *string `json:"model,omitempty" bson:"model,omitempty"`
}
