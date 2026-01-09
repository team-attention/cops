package domain

import (
	"encoding/json"
	"log/slog"
	"time"
)

// EventType represents the type of Claude Code event.
type EventType string

const (
	EventTypeSessionStart     EventType = "session_start"
	EventTypePostToolUse      EventType = "post_tool_use"
	EventTypeNotification     EventType = "notification"
	EventTypeUserPromptSubmit EventType = "user_prompt_submit"
	EventTypeStop             EventType = "stop"
	EventTypeSubagentStop     EventType = "subagent_stop"
	EventTypeSessionEnd       EventType = "session_end"
)

// Event represents a Claude Code event with type-specific data.
type Event struct {
	Type EventType `json:"type" bson:"type"`
	Data any       `bson:",inline"`
}

// EventBase contains common fields shared by all event types.
type EventBase struct {
	SessionID ID        `json:"sessionId" bson:"-"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}

// SessionStartEvent represents the start of a Claude Code session.
type SessionStartEvent struct {
	EventBase
	SessionType *string                `json:"session_type,omitempty" bson:"sessionType,omitempty"`
	Tools       []string               `json:"tools,omitempty" bson:"tools,omitempty"`
	McpServers  []string               `json:"mcp_servers,omitempty" bson:"mcpServers,omitempty"`
	Model       *string                `json:"model,omitempty" bson:"model,omitempty"`
	PermMode    *string                `json:"perm_mode,omitempty" bson:"permMode,omitempty"`
	MaxTurns    *int                   `json:"max_turns,omitempty" bson:"maxTurns,omitempty"`
	RawData     map[string]any `json:"-" bson:"rawData,omitempty"`
}

// PostToolUseEvent represents a tool use completion event.
type PostToolUseEvent struct {
	EventBase
	ToolName   string                 `json:"tool_name" bson:"toolName"`
	ToolID     string                 `json:"tool_id" bson:"toolId"`
	Success    bool                   `json:"success" bson:"success"`
	Error      *string                `json:"error,omitempty" bson:"error,omitempty"`
	DurationMs *int64                 `json:"duration_ms,omitempty" bson:"durationMs,omitempty"`
	RawData    map[string]interface{} `json:"-" bson:"rawData,omitempty"`
}

// NotificationEvent represents a notification event.
type NotificationEvent struct {
	EventBase
	Level    string                 `json:"level" bson:"level"`
	Message  string                 `json:"message" bson:"message"`
	Category *string                `json:"category,omitempty" bson:"category,omitempty"`
	RawData  map[string]interface{} `json:"-" bson:"rawData,omitempty"`
}

// UserPromptSubmitEvent represents a user prompt submission event.
type UserPromptSubmitEvent struct {
	EventBase
	PromptLength *int                   `json:"prompt_length,omitempty" bson:"promptLength,omitempty"`
	HasImages    *bool                  `json:"has_images,omitempty" bson:"hasImages,omitempty"`
	RawData      map[string]interface{} `json:"-" bson:"rawData,omitempty"`
}

// StopEvent represents a stop/completion event.
type StopEvent struct {
	EventBase
	StopReason   *string                `json:"stop_reason,omitempty" bson:"stopReason,omitempty"`
	TotalTurns   *int                   `json:"total_turns,omitempty" bson:"totalTurns,omitempty"`
	InputTokens  *int64                 `json:"input_tokens,omitempty" bson:"inputTokens,omitempty"`
	OutputTokens *int64                 `json:"output_tokens,omitempty" bson:"outputTokens,omitempty"`
	RawData      map[string]interface{} `json:"-" bson:"rawData,omitempty"`
}

// SubagentStopEvent represents a subagent stop event.
type SubagentStopEvent struct {
	EventBase
	SubagentID   *string                `json:"subagent_id,omitempty" bson:"subagentId,omitempty"`
	StopReason   *string                `json:"stop_reason,omitempty" bson:"stopReason,omitempty"`
	InputTokens  *int64                 `json:"input_tokens,omitempty" bson:"inputTokens,omitempty"`
	OutputTokens *int64                 `json:"output_tokens,omitempty" bson:"outputTokens,omitempty"`
	RawData      map[string]interface{} `json:"-" bson:"rawData,omitempty"`
}

// SessionEndEvent represents the end of a Claude Code session.
type SessionEndEvent struct {
	EventBase
	ExitCode          *int                   `json:"exit_code,omitempty" bson:"exitCode,omitempty"`
	TotalDurationMs   *int64                 `json:"total_duration_ms,omitempty" bson:"totalDurationMs,omitempty"`
	TotalInputTokens  *int64                 `json:"total_input_tokens,omitempty" bson:"totalInputTokens,omitempty"`
	TotalOutputTokens *int64                 `json:"total_output_tokens,omitempty" bson:"totalOutputTokens,omitempty"`
	RawData           map[string]interface{} `json:"-" bson:"rawData,omitempty"`
}

// eventTypeFactory is a function that returns a pointer to a new instance of the event data type.
type eventTypeFactory func() any

// eventTypeRegistry maps EventType values to their corresponding factory functions.
// This registry enables extensible unmarshaling without modifying core logic.
var eventTypeRegistry = map[EventType]eventTypeFactory{
	EventTypeSessionStart:     func() any { return &SessionStartEvent{} },
	EventTypePostToolUse:      func() any { return &PostToolUseEvent{} },
	EventTypeNotification:     func() any { return &NotificationEvent{} },
	EventTypeUserPromptSubmit: func() any { return &UserPromptSubmitEvent{} },
	EventTypeStop:             func() any { return &StopEvent{} },
	EventTypeSubagentStop:     func() any { return &SubagentStopEvent{} },
	EventTypeSessionEnd:       func() any { return &SessionEndEvent{} },
}

// MarshalJSON serializes Event to JSON with flattened Data fields.
// The Data field's contents are merged at the top level alongside the "type" field.
// This produces flat JSON matching the event format: {"type":"...", ...data fields...}
func (e Event) MarshalJSON() ([]byte, error) {
	// If Data is nil, marshal only the Type field as {"type":"..."}
	if e.Data == nil {
		return json.Marshal(map[string]any{"type": e.Type})
	}

	// Marshal the Data field to JSON bytes
	dataBytes, err := json.Marshal(e.Data)
	if err != nil {
		return nil, err
	}

	// Unmarshal Data JSON into map[string]json.RawMessage to preserve field values
	var dataMap map[string]json.RawMessage
	if err := json.Unmarshal(dataBytes, &dataMap); err != nil {
		return nil, err
	}

	// Add "type" field to the map with the Event's Type value
	typeBytes, err := json.Marshal(e.Type)
	if err != nil {
		return nil, err
	}
	dataMap["type"] = typeBytes

	// Marshal the combined map to produce final flat JSON
	return json.Marshal(dataMap)
}

// UnmarshalJSON deserializes flat JSON into Event with the appropriate typed Data.
// It reads the "type" field to determine which concrete type to use for Data,
// then unmarshals the entire JSON into that type (since event-specific fields
// are at the top level alongside "type").
// Unknown types are stored as map[string]any and logged as errors.
// Schema mismatches are handled permissively (missing fields become zero values).
func (e *Event) UnmarshalJSON(data []byte) error {
	// Define a temporary struct to extract just the Type field
	type typeExtractor struct {
		Type EventType `json:"type"`
	}

	// Unmarshal data into typeExtractor to get the event type
	var extractor typeExtractor
	if err := json.Unmarshal(data, &extractor); err != nil {
		return err
	}

	// Set e.Type from extracted type
	e.Type = extractor.Type

	// Look up the type in eventTypeRegistry
	factory, found := eventTypeRegistry[e.Type]
	if found {
		// Call factory function to create new instance of the typed struct
		typedData := factory()

		// Unmarshal full data into the typed struct (permissive - ignores unknown fields)
		if err := json.Unmarshal(data, typedData); err != nil {
			// Log warning and fall through to map storage
			slog.Error("failed to unmarshal event data into typed struct",
				"type", e.Type,
				"error", err)
		} else {
			// Set e.Data to the typed struct pointer
			e.Data = typedData
			return nil
		}
	} else {
		// Log error for unknown type
		slog.Error("unknown event type encountered",
			"type", e.Type)
	}

	// Type not found in registry OR typed unmarshal failed
	// Create map[string]any to store raw data
	var mapData map[string]any
	if err := json.Unmarshal(data, &mapData); err != nil {
		return err
	}

	// Set e.Data to the map
	e.Data = mapData
	return nil
}
