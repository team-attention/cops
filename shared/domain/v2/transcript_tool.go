package v2

import "time"

// ToolResultStatus represents the execution status of a tool call.
type ToolResultStatus string

const (
	ToolResultStatusSuccess ToolResultStatus = "success"
	ToolResultStatusError   ToolResultStatus = "error"
	ToolResultStatusTimeout ToolResultStatus = "timeout"
	ToolResultStatusSkipped ToolResultStatus = "skipped" // User rejected or tool blocked
)

// ToolExecution represents a tool call with its result.
// This is an independent type that separates tool execution from message flow.
//
// In v1, tool calls were embedded:
//   - tool_use in assistant message
//   - tool_result in user message
//
// In v2, ToolExecution is a first-class entry that contains both call and result.
type ToolExecution struct {
	TreeNodeMeta `bson:",inline"`

	// ID is the unique identifier for this tool execution.
	// Corresponds to tool_use_id in Claude API.
	ID string `json:"id" bson:"id"`

	// ToolName is the name of the tool being executed.
	// e.g., "Bash", "Read", "Write", "Edit", "Glob", "Grep"
	ToolName string `json:"toolName" bson:"toolName"`

	// Input contains the tool parameters.
	Input map[string]any `json:"input" bson:"input"`

	// Result contains the execution result (nil if execution is pending).
	Result *ToolResult `json:"result,omitempty" bson:"result,omitempty"`

	// Metadata contains execution metadata (duration, approval mode, etc.).
	Metadata *ToolExecutionMetadata `json:"metadata,omitempty" bson:"metadata,omitempty"`

	// SourceAgentUUID references the agent message that initiated this tool call.
	SourceAgentUUID string `json:"sourceAgentUuid" bson:"sourceAgentUuid"`
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	// Status indicates the execution outcome.
	Status ToolResultStatus `json:"status" bson:"status"`

	// Content contains the result data.
	// Type depends on the tool (string, map, array, etc.).
	Content any `json:"content,omitempty" bson:"content,omitempty"`

	// Error contains error message if Status is error or timeout.
	Error *string `json:"error,omitempty" bson:"error,omitempty"`
}

// ToolExecutionMetadata contains additional metadata about tool execution.
type ToolExecutionMetadata struct {
	// DurationMs is the execution time in milliseconds.
	DurationMs *int `json:"durationMs,omitempty" bson:"durationMs,omitempty"`

	// ApprovalMode indicates how the tool was approved.
	// e.g., "auto", "manual", "allowed_by_hook"
	ApprovalMode string `json:"approvalMode,omitempty" bson:"approvalMode,omitempty"`

	// ExecutedAt is when the tool was executed.
	ExecutedAt *time.Time `json:"executedAt,omitempty" bson:"executedAt,omitempty"`

	// IsSubagent indicates if this tool was called by a subagent.
	IsSubagent bool `json:"isSubagent,omitempty" bson:"isSubagent,omitempty"`

	// SubagentID is the ID of the subagent that called this tool (if IsSubagent is true).
	SubagentID string `json:"subagentId,omitempty" bson:"subagentId,omitempty"`
}
