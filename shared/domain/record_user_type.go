package domain

import "time"

// ------- Type User -------

type UserRecordUserType string

const (
	UserRecordUserTypeExternal UserRecordUserType = "external"
)

type UserRecordMessageRole string

const (
	UserRecordMessageRoleUser UserRecordMessageRole = "user"
)

type UserRecordMessage struct {
	Role    UserRecordMessageRole `json:"role" bson:"role"`
	Content string                `json:"content" bson:"content"`
}

type UserRecordThinkingTrigger struct {
	Start int    `json:"start" bson:"start"`
	End   int    `json:"end" bson:"end"`
	Text  string `json:"text" bson:"text"`
}

type UserRecordThinkingMetadata struct {
	Level    string                       `json:"level" bson:"level"`
	Disabled bool                         `json:"disabled" bson:"disabled"`
	Triggers []*UserRecordThinkingTrigger `json:"triggers" bson:"triggers"`
}

type UserRecordTodo struct {
	Content    string `json:"content" bson:"content"`
	Status     string `json:"status" bson:"status"`
	ActiveForm string `json:"activeForm" bson:"activeForm"`
}

type UserRecord struct {
	ParentUUID       *string                     `json:"parentUuid,omitempty" bson:"parentUuid,omitempty"`
	IsSidechain      bool                        `json:"isSidechain" bson:"isSidechain"`
	UserType         UserRecordUserType          `json:"userType" bson:"userType"`
	CWD              string                      `json:"cwd" bson:"cwd"`
	SessionID        string                      `json:"sessionId" bson:"sessionId"`
	Version          string                      `json:"version" bson:"version"`
	GitBranch        string                      `json:"gitBranch" bson:"gitBranch"`
	Message          UserRecordMessage           `json:"message" bson:"message"`
	IsMeta           bool                        `json:"isMeta" bson:"isMeta"`
	UUID             string                      `json:"uuid" bson:"uuid"`
	Timestamp        time.Time                   `json:"timestamp" bson:"timestamp"`
	ThinkingMetadata *UserRecordThinkingMetadata `json:"thinkingMetadata,omitempty" bson:"thinkingMetadata,omitempty"`
	Todos            []*UserRecordTodo           `json:"todos,omitempty" bson:"todos,omitempty"`
}
