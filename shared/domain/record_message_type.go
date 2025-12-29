package domain

// ------- Type Message -------

type MessageRecordMessageType string

const (
	MessageRecordMessageTypeMessage MessageRecordMessageType = "message"
)

type MessageRecordMessageRole string

const (
	MessageRecordMessageRoleAssistant MessageRecordMessageRole = "assistant"
)

type MessageRecordMessageContentType string

const (
	MessageRecordMessageContentTypeText    MessageRecordMessageContentType = "text"
	MessageRecordMessageContentTypeTool    MessageRecordMessageContentType = "thinking"
	MessageRecordMessageContentTypeToolUse MessageRecordMessageContentType = "tool_use"
)

type MessageRecordMessageContent struct {
	Type     MessageRecordMessageContentType `json:"type" bson:"type"`
	Text     *string                         `json:"text,omitempty" bson:"text,omitempty"`
	Thinking *string                         `json:"thinking,omitempty" bson:"thinking,omitempty"`
}

type MessageRecordMessageToolUseContent struct {
	ID    string `json:"id" bson:"id"`
	Name  string `json:"name" bson:"name"`
	Input any    `json:"input" bson:"input"`
}

type MessageRecordMessageUsage struct {
	InputTokens  int `json:"inputTokens" bson:"inputTokens"`
	OutputTokens int `json:"outputTokens" bson:"outputTokens"`
}

type MessageRecordMessage struct {
	Model        string                     `json:"model" bson:"model"`
	ID           string                     `json:"id" bson:"id"`
	Type         MessageRecordMessageType   `json:"type" bson:"type"`
	Role         MessageRecordMessageRole   `json:"role" bson:"role"`
	Content      []interface{}              `json:"content" bson:"content"`
	StopReason   *string                    `json:"stopReason,omitempty" bson:"stopReason,omitempty"`
	StopSequence *int                       `json:"stopSequence,omitempty" bson:"stopSequence,omitempty"`
	Usage        *MessageRecordMessageUsage `json:"usage,omitempty" bson:"usage,omitempty"`
}

type MessageRecord struct {
	ParentUUID  *string `json:"parentUuid,omitempty" bson:"parentUuid,omitempty"`
	IsSidechain bool    `json:"isSidechain" bson:"isSidechain"`
	UserType    string  `json:"userType" bson:"userType"`
	CWD         string  `json:"cwd" bson:"cwd"`
	SessionID   string  `json:"sessionId" bson:"sessionId"`
	Version     string  `json:"version" bson:"version"`
	GitBranch   string  `json:"gitBranch" bson:"gitBranch"`
	Message     Message `json:"message" bson:"message"`
}
