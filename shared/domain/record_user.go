package domain

import (
	"bytes"
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

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

type UserRecordToolUseResult struct {
	Type string                       `json:"type" bson:"type"`
	File *UserRecordToolUseResultFile `json:"file,omitempty" bson:"file,omitempty"`
}

type UserRecordToolUseResultFile struct {
	FilePath   string `json:"filePath" bson:"filePath"`
	Content    string `json:"content" bson:"content"`
	NumLines   int    `json:"numLines" bson:"numLines"`
	StartLine  int    `json:"startLine" bson:"startLine"`
	TotalLines int    `json:"totalLines" bson:"totalLines"`
}

type UserMessageBlockContent struct {
	Type   string                         `json:"type" bson:"type"`
	Text   *string                        `json:"text,omitempty" bson:"text,omitempty"`
	Source *UserMessageBlockContentSource `json:"source,omitempty" bson:"source,omitempty"`

	*UserMessageBlockContentToolResult `json:",omitempty" bson:",inline,omitempty"`
}

type UserMessageBlockContentToolResult struct {
	ToolUseID string `json:"tool_use_id" bson:"toolUseId"`
	Content   any    `json:"content" bson:"content"`
}

type UserMessageBlockContentSource struct {
	Type       string `json:"type" bson:"type"`
	Media_type string `json:"media_type" bson:"media_type"`
	Data       string `json:"data" bson:"data"`
}

type UserMessage struct {
	Role    UserMessageRole `json:"role" bson:"role"`
	Content any             `json:"content" bson:"content"`
}

// userMessageAlias is used to prevent infinite recursion during JSON marshaling.
// It has the same fields as UserMessage but without custom marshal methods.
type userMessageAlias struct {
	Role    UserMessageRole `json:"role"`
	Content any             `json:"content"`
}

// userMessageRaw is used for initial JSON unmarshaling to capture Content as raw bytes.
type userMessageRaw struct {
	Role    UserMessageRole `json:"role"`
	Content json.RawMessage `json:"content"`
}

// UnmarshalJSON deserializes JSON into UserMessage with polymorphic Content handling.
// It detects whether Content is a string or array and unmarshals accordingly.
func (m *UserMessage) UnmarshalJSON(data []byte) error {
	// 1. Unmarshal data into userMessageRaw to capture Role and raw Content bytes.
	var raw userMessageRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to unmarshal UserMessage: %w", err)
	}

	// 2. Set m.Role from the raw struct.
	m.Role = raw.Role

	// 3. If raw.Content is empty or null, set m.Content to nil and return.
	if len(raw.Content) == 0 || bytes.Equal(raw.Content, []byte("null")) {
		m.Content = nil
		return nil
	}

	// 4. Trim whitespace from raw.Content to detect first character.
	trimmed := bytes.TrimSpace(raw.Content)
	if len(trimmed) == 0 {
		m.Content = nil
		return nil
	}

	// 5. If first character is '"' (string):
	if trimmed[0] == '"' {
		// a. Unmarshal raw.Content into a string variable.
		var str string
		if err := json.Unmarshal(raw.Content, &str); err != nil {
			return fmt.Errorf("failed to unmarshal UserMessage.Content as string: %w", err)
		}
		// b. Set m.Content to the string value.
		m.Content = str
		return nil
	}

	// 6. Else if first character is '[' (array):
	if trimmed[0] == '[' {
		// a. Unmarshal raw.Content into []*UserMessageBlockContent.
		var blocks []*UserMessageBlockContent
		if err := json.Unmarshal(raw.Content, &blocks); err != nil {
			return fmt.Errorf("failed to unmarshal UserMessage.Content as array: %w", err)
		}
		// b. Set m.Content to the slice value.
		m.Content = blocks
		return nil
	}

	// 7. Else: Return error for unexpected type.
	return fmt.Errorf("UserMessage.Content must be string or array, got unexpected JSON type starting with '%c'", trimmed[0])
}

// MarshalJSON serializes UserMessage to JSON, preserving the Content type.
// String content is marshaled as JSON string, array content as JSON array.
func (m UserMessage) MarshalJSON() ([]byte, error) {
	// 1. Create userMessageAlias from m to use default struct marshaling.
	alias := userMessageAlias{
		Role:    m.Role,
		Content: m.Content,
	}
	// 2. Marshal the alias to JSON bytes.
	// 3. Return the marshaled bytes.
	// Note: Since Content is `any` type, json.Marshal will correctly serialize
	//       string as JSON string and []*UserMessageBlockContent as JSON array.
	return json.Marshal(alias)
}

// userMessageBSONRaw is used for initial BSON unmarshaling to capture Content as raw value.
type userMessageBSONRaw struct {
	Role    UserMessageRole `bson:"role"`
	Content bson.RawValue   `bson:"content"`
}

// UnmarshalBSON deserializes BSON into UserMessage with polymorphic Content handling.
// It detects whether Content is a string or array and unmarshals accordingly.
func (m *UserMessage) UnmarshalBSON(data []byte) error {
	// 1. Unmarshal data into userMessageBSONRaw to capture Role and raw Content.
	var raw userMessageBSONRaw
	if err := bson.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to unmarshal UserMessage BSON: %w", err)
	}

	// 2. Set m.Role from the raw struct.
	m.Role = raw.Role

	// 3. If raw.Content is empty (Type == 0) or null, set m.Content to nil and return.
	if raw.Content.Type == 0 || raw.Content.Type == bson.TypeNull {
		m.Content = nil
		return nil
	}

	// 4. Check raw.Content.Type:
	switch raw.Content.Type {
	case bson.TypeString:
		// a. If bson.TypeString: Unmarshal raw.Content into a string variable.
		var str string
		if err := raw.Content.Unmarshal(&str); err != nil {
			return fmt.Errorf("failed to unmarshal UserMessage.Content as string from BSON: %w", err)
		}
		// Set m.Content to the string value.
		m.Content = str

	case bson.TypeArray:
		// b. If bson.TypeArray: Unmarshal raw.Content into []*UserMessageBlockContent.
		var blocks []*UserMessageBlockContent
		if err := raw.Content.Unmarshal(&blocks); err != nil {
			return fmt.Errorf("failed to unmarshal UserMessage.Content as array from BSON: %w", err)
		}
		// Set m.Content to the slice value.
		m.Content = blocks

	default:
		// c. Else: Return error for unexpected type.
		return fmt.Errorf("UserMessage.Content must be string or array in BSON, got type %v", raw.Content.Type)
	}

	// 5. Return nil on success.
	return nil
}

// MarshalBSON serializes UserMessage to BSON, preserving the Content type.
// String content is marshaled as BSON string, array content as BSON array.
func (m UserMessage) MarshalBSON() ([]byte, error) {
	// 1. Create a bson.D document with Role and Content fields.
	doc := bson.D{
		{Key: "role", Value: m.Role},
		{Key: "content", Value: m.Content},
	}
	// 2. Marshal the document to BSON bytes using bson.Marshal.
	// 3. Return the marshaled bytes.
	// Note: bson.Marshal will correctly serialize string as BSON string
	//       and []*UserMessageBlockContent as BSON array.
	return bson.Marshal(doc)
}

type UserRecord struct {
	MessageMetadata `bson:"inline"`
	Message         UserMessage `json:"message" bson:"message"`

	IsMeta           bool                        `json:"isMeta" bson:"isMeta"`
	ThinkingMetadata *UserRecordThinkingMetadata `json:"thinkingMetadata,omitempty" bson:"thinkingMetadata,omitempty"`
	Todos            []*UserRecordTodo           `json:"todos,omitempty" bson:"todos,omitempty"`
	ToolUseResult    *UserRecordToolUseResult    `json:"toolUseResult,omitempty" bson:"toolUseResult,omitempty"`
}
