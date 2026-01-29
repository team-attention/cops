package domain

// ProgressDataType represents the subtype of progress entry.
type ProgressDataType string

const (
	ProgressDataTypeAgent ProgressDataType = "agent_progress"
	ProgressDataTypeSkill ProgressDataType = "skill_progress"
	ProgressDataTypeHook  ProgressDataType = "hook_progress"
)

// ProgressTranscript represents an agent/skill progress entry in the conversation tree.
type ProgressTranscript struct {
	TreeNodeMeta `json:",inline" bson:",inline"`

	// Progress-specific fields
	ToolUseID       *string      `json:"toolUseID,omitempty" bson:"toolUseID,omitempty"`
	ParentToolUseID *string      `json:"parentToolUseID,omitempty" bson:"parentToolUseID,omitempty"`
	Data            ProgressData `json:"data" bson:"data"`
}

// ProgressData contains the progress payload.
type ProgressData struct {
	Type               ProgressDataType `json:"type" bson:"type"`
	Message            map[string]any   `json:"message,omitempty" bson:"message,omitempty"`
	NormalizedMessages []map[string]any `json:"normalizedMessages,omitempty" bson:"normalizedMessages,omitempty"`
	Prompt             *string          `json:"prompt,omitempty" bson:"prompt,omitempty"`
	AgentID            *string          `json:"agentId,omitempty" bson:"agentId,omitempty"`
	// Hook progress fields
	HookEvent *string `json:"hookEvent,omitempty" bson:"hookEvent,omitempty"`
	HookName  *string `json:"hookName,omitempty" bson:"hookName,omitempty"`
	Command   *string `json:"command,omitempty" bson:"command,omitempty"`
}
