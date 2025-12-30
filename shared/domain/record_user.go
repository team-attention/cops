package domain

// ------- Type User -------

type UserMessageRole string

const (
	UserMessageRoleUser UserMessageRole = "user"
)

type UserRecordThinkingMetadataTrigger struct {
	Start int    `json:"start" bson:"start"`
	End   int    `json:"end" bson:"end"`
	Text  string `json:"text" bson:"text"`
}

type UserRecordThinkingMetadata struct {
	Level    string                               `json:"level" bson:"level"`
	Disabled bool                                 `json:"disabled" bson:"disabled"`
	Triggers []*UserRecordThinkingMetadataTrigger `json:"triggers" bson:"triggers"`
}

type UserRecordTodo struct {
	Content    string `json:"content" bson:"content"`
	Status     string `json:"status" bson:"status"`
	ActiveForm string `json:"activeForm" bson:"activeForm"`
}

type UserMessage struct {
	Role    UserMessageRole `json:"role" bson:"role"`
	Content string          `json:"content" bson:"content"`
}

type UserRecord struct {
	MessageMetadata `bson:"inline"`
	Message         UserMessage `json:"message" bson:"message"`

	IsMeta           bool                        `json:"isMeta" bson:"isMeta"`
	ThinkingMetadata *UserRecordThinkingMetadata `json:"thinkingMetadata,omitempty" bson:"thinkingMetadata,omitempty"`
	Todos            []*UserRecordTodo           `json:"todos,omitempty" bson:"todos,omitempty"`
}
