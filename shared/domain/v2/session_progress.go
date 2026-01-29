package session

// ProgressType represents the type of progress update.
type ProgressType string

const (
	ProgressTypeAgent ProgressType = "agent_progress"
	ProgressTypeHook  ProgressType = "hook_progress"
	ProgressTypeBash  ProgressType = "bash_progress"
	ProgressTypeMcp   ProgressType = "mcp_progress"
)

// Progress represents real-time progress updates during tool execution.
// Links to the associated ToolExecution via ToolExecutionID.
type Progress struct {
	TreeNodeMeta `bson:",inline"`

	// ToolExecutionID references the ToolExecution this progress is for.
	// Corresponds to toolUseID in v1.
	ToolExecutionID string `json:"toolExecutionId,omitempty" bson:"toolExecutionId,omitempty"`

	// ParentToolExecutionID references parent tool execution for nested operations.
	ParentToolExecutionID string `json:"parentToolExecutionId,omitempty" bson:"parentToolExecutionId,omitempty"`

	// Data contains the progress payload.
	Data ProgressData `json:"data" bson:"data"`
}

// ProgressData contains the progress payload.
type ProgressData struct {
	Type ProgressType `json:"type" bson:"type"`

	// Message contains provider-specific progress message.
	Message map[string]any `json:"message,omitempty" bson:"message,omitempty"`

	// NormalizedMessages contains normalized progress messages.
	NormalizedMessages []map[string]any `json:"normalizedMessages,omitempty" bson:"normalizedMessages,omitempty"`

	// Prompt is the user prompt for subagent progress.
	Prompt *string `json:"prompt,omitempty" bson:"prompt,omitempty"`

	// AgentID is the subagent ID for agent progress.
	AgentID *string `json:"agentId,omitempty" bson:"agentId,omitempty"`

	// Hook progress fields
	HookEvent *string `json:"hookEvent,omitempty" bson:"hookEvent,omitempty"`
	HookName  *string `json:"hookName,omitempty" bson:"hookName,omitempty"`
	Command   *string `json:"command,omitempty" bson:"command,omitempty"`
}
