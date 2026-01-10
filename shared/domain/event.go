package domain

import (
	"encoding/json"
	"log/slog"
)

// EventType represents the type of Claude Code hook event.
// Values match Claude Code's hook_event_name field (PascalCase).
type EventType string

const (
	EventTypePreToolUse       EventType = "PreToolUse"
	EventTypePostToolUse      EventType = "PostToolUse"
	EventTypeNotification     EventType = "Notification"
	EventTypeUserPromptSubmit EventType = "UserPromptSubmit"
	EventTypeStop             EventType = "Stop"
	EventTypeSubagentStop     EventType = "SubagentStop"
	EventTypePreCompact       EventType = "PreCompact"
	EventTypeSessionStart     EventType = "SessionStart"
	EventTypeSessionEnd       EventType = "SessionEnd"
)

// Event represents a Claude Code hook event with type-specific data.
type Event struct {
	Type EventType `json:"hook_event_name" bson:"hookEventName"`
	Data any       `bson:",inline"`
}

// HookEventBase contains common fields for all hook events.
// Field names match Claude Code's hook input schema (snake_case).
type HookEventBase struct {
	SessionID      string `json:"session_id" bson:"sessionId"`
	PermissionMode string `json:"permission_mode" bson:"permissionMode"`
	// TranscriptPath string `json:"transcript_path" bson:"transcriptPath"` // Excluded: local file path, not needed on server
	// Cwd            string `json:"cwd" bson:"cwd"`                         // Excluded: local file path, not needed on server
}

// PreToolUseEvent represents a pre-tool-use hook event.
// Triggered before Claude executes a tool.
type PreToolUseEvent struct {
	HookEventBase
	ToolName  string         `json:"tool_name" bson:"toolName"`
	ToolInput map[string]any `json:"tool_input" bson:"toolInput"`
	ToolUseID string         `json:"tool_use_id" bson:"toolUseId"`
}

// PostToolUseEvent represents a post-tool-use hook event.
// Triggered after Claude executes a tool.
type PostToolUseEvent struct {
	HookEventBase
	ToolName     string         `json:"tool_name" bson:"toolName"`
	ToolInput    map[string]any `json:"tool_input" bson:"toolInput"`
	ToolResponse map[string]any `json:"tool_response" bson:"toolResponse"`
	ToolUseID    string         `json:"tool_use_id" bson:"toolUseId"`
}

// NotificationEvent represents a notification hook event.
// Triggered when Claude Code sends notifications.
type NotificationEvent struct {
	HookEventBase
	Message          string `json:"message" bson:"message"`
	NotificationType string `json:"notification_type" bson:"notificationType"`
}

// UserPromptSubmitEvent represents a user prompt submission hook event.
// Triggered when a user submits a prompt.
type UserPromptSubmitEvent struct {
	HookEventBase
	Prompt string `json:"prompt" bson:"prompt"`
}

// StopEvent represents a stop hook event.
// Triggered when the main Claude Code agent finishes responding.
type StopEvent struct {
	HookEventBase
	StopHookActive bool `json:"stop_hook_active" bson:"stopHookActive"`
}

// SubagentStopEvent represents a subagent stop hook event.
// Triggered when a Claude Code subagent finishes responding.
type SubagentStopEvent struct {
	HookEventBase
	StopHookActive bool   `json:"stop_hook_active" bson:"stopHookActive"`
	AgentID        string `json:"agent_id" bson:"agentId"`
	// AgentTranscriptPath string `json:"agent_transcript_path" bson:"agentTranscriptPath"` // Excluded: local file path, not needed on server
}

// PreCompactEvent represents a pre-compact hook event.
// Triggered before Claude Code runs a compact operation.
type PreCompactEvent struct {
	HookEventBase
	Trigger            string `json:"trigger" bson:"trigger"`
	CustomInstructions string `json:"custom_instructions" bson:"customInstructions"`
}

// SessionStartEvent represents a session start hook event.
// Triggered when Claude Code starts or resumes a session.
type SessionStartEvent struct {
	HookEventBase
	Source string `json:"source" bson:"source"`
}

// SessionEndEvent represents a session end hook event.
// Triggered when a Claude Code session ends.
type SessionEndEvent struct {
	HookEventBase
	Reason string `json:"reason" bson:"reason"`
}

// eventTypeFactory is a function that returns a pointer to a new instance of the event data type.
type eventTypeFactory func() any

// eventTypeRegistry maps EventType values to their corresponding factory functions.
var eventTypeRegistry = map[EventType]eventTypeFactory{
	EventTypePreToolUse:       func() any { return &PreToolUseEvent{} },
	EventTypePostToolUse:      func() any { return &PostToolUseEvent{} },
	EventTypeNotification:     func() any { return &NotificationEvent{} },
	EventTypeUserPromptSubmit: func() any { return &UserPromptSubmitEvent{} },
	EventTypeStop:             func() any { return &StopEvent{} },
	EventTypeSubagentStop:     func() any { return &SubagentStopEvent{} },
	EventTypePreCompact:       func() any { return &PreCompactEvent{} },
	EventTypeSessionStart:     func() any { return &SessionStartEvent{} },
	EventTypeSessionEnd:       func() any { return &SessionEndEvent{} },
}

// MarshalJSON serializes Event to JSON with flattened Data fields.
// The Data field's contents are merged at the top level alongside the "hook_event_name" field.
func (e Event) MarshalJSON() ([]byte, error) {
	if e.Data == nil {
		return json.Marshal(map[string]any{"hook_event_name": e.Type})
	}

	dataBytes, err := json.Marshal(e.Data)
	if err != nil {
		return nil, err
	}

	var dataMap map[string]json.RawMessage
	if err := json.Unmarshal(dataBytes, &dataMap); err != nil {
		return nil, err
	}

	typeBytes, err := json.Marshal(e.Type)
	if err != nil {
		return nil, err
	}
	dataMap["hook_event_name"] = typeBytes

	return json.Marshal(dataMap)
}

// UnmarshalJSON deserializes flat JSON into Event with the appropriate typed Data.
// It reads the "hook_event_name" field (or "type" for backward compatibility) to determine
// which concrete type to use for Data.
func (e *Event) UnmarshalJSON(data []byte) error {
	// Extract event type from either hook_event_name or type field
	type typeExtractor struct {
		HookEventName string    `json:"hook_event_name"`
		Type          EventType `json:"type"`
	}

	var extractor typeExtractor
	if err := json.Unmarshal(data, &extractor); err != nil {
		return err
	}

	// Determine event type: prefer hook_event_name, fallback to type
	if extractor.HookEventName != "" {
		e.Type = EventType(extractor.HookEventName)
	} else if extractor.Type != "" {
		e.Type = extractor.Type
	}

	// Look up the type in eventTypeRegistry
	factory, found := eventTypeRegistry[e.Type]
	if found {
		typedData := factory()

		if err := json.Unmarshal(data, typedData); err != nil {
			slog.Error("failed to unmarshal event data into typed struct",
				"type", e.Type,
				"error", err)
		} else {
			e.Data = typedData
			return nil
		}
	} else if e.Type != "" {
		slog.Error("unknown event type encountered",
			"type", e.Type)
	}

	// Type not found in registry OR typed unmarshal failed
	var mapData map[string]any
	if err := json.Unmarshal(data, &mapData); err != nil {
		return err
	}

	e.Data = mapData
	return nil
}
