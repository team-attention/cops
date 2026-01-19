// Package v2 provides provider-agnostic transcript models for AI agent sessions.
//
// # Design Goals
//
// This package abstracts transcript data from Claude Code-specific structures,
// enabling support for multiple AI providers (Claude, OpenAI, Gemini, etc.).
//
// # Key Abstractions
//
//  1. ToolExecution as independent type (not embedded in User/Agent messages)
//  2. Provider-agnostic content blocks (text, image, thinking)
//  3. Adapter pattern for provider-specific log conversion
//
// # Type Hierarchy
//
//	Transcript (discriminated union)
//	├── HumanMessage     - User input (text, image only)
//	├── AgentMessage     - AI response (text, thinking, tool_call refs)
//	├── ToolExecution    - Tool call + result (independent)
//	├── SystemMessage    - System metadata (turn_duration, summary, file_snapshot)
//	└── Progress         - Tool execution progress updates
//
// # Migration from v1
//
// Use claudecodeadapter.ClaudeCodeAdapter to convert v1 transcripts to v2.
//
// v1 -> v2 mapping:
//   - user (tool_result)     -> ToolExecution
//   - user (text/image)      -> HumanMessage
//   - assistant              -> AgentMessage + ToolExecution (tool_use)
//   - system                 -> SystemMessage (subtype=turn_duration)
//   - summary                -> SystemMessage (subtype=summary)
//   - file-history-snapshot  -> SystemMessage (subtype=file_snapshot)
//   - progress               -> Progress
package v2

import (
	"encoding/json"
	"log/slog"
	"time"
)

// TranscriptType represents the type discriminator for v2 transcript entries.
type TranscriptType string

const (
	// TranscriptTypeHuman represents user input in conversation tree.
	// Contains only text and image content blocks (no tool_result).
	TranscriptTypeHuman TranscriptType = "human"

	// TranscriptTypeAgent represents AI agent response in conversation tree.
	// Contains text, thinking, and tool_call references (actual tool calls are in ToolExecution).
	TranscriptTypeAgent TranscriptType = "agent"

	// TranscriptTypeToolExecution represents a tool call with its result.
	// Independent type that links to the originating agent via SourceAgentUUID.
	TranscriptTypeToolExecution TranscriptType = "tool_execution"

	// TranscriptTypeSystem represents system-level metadata.
	// Uses Subtype to distinguish: turn_duration, summary, file_snapshot.
	TranscriptTypeSystem TranscriptType = "system"

	// TranscriptTypeProgress represents real-time progress updates during tool execution.
	// Links to ToolExecution via ToolExecutionID.
	TranscriptTypeProgress TranscriptType = "progress"
)

// TreeNodeMeta contains common fields for conversation tree nodes.
// These types form a tree structure linked by ParentUUID.
type TreeNodeMeta struct {
	ParentUUID  *string   `json:"parentUuid,omitempty" bson:"parentUuid,omitempty"`
	UUID        string    `json:"uuid" bson:"uuid"`
	SessionID   string    `json:"sessionId" bson:"sessionId"`
	Timestamp   time.Time `json:"timestamp" bson:"timestamp"`
	Version     string    `json:"version,omitempty" bson:"version,omitempty"`
	CWD         string    `json:"cwd,omitempty" bson:"cwd,omitempty"`
	GitBranch   string    `json:"gitBranch,omitempty" bson:"gitBranch,omitempty"`
	Slug        string    `json:"slug,omitempty" bson:"slug,omitempty"`
	UserType    string    `json:"userType,omitempty" bson:"userType,omitempty"`
	IsSidechain bool      `json:"isSidechain,omitempty" bson:"isSidechain,omitempty"`
}

// Transcript represents a single entry with polymorphic data.
type Transcript struct {
	Type TranscriptType `json:"type" bson:"type"`
	Data any            `bson:",inline"`
}

// transcriptTypeFactory creates new instances of transcript data types.
type transcriptTypeFactory func() any

// transcriptTypeRegistry maps TranscriptType to factory functions.
var transcriptTypeRegistry = map[TranscriptType]transcriptTypeFactory{
	TranscriptTypeHuman:         func() any { return &HumanMessage{} },
	TranscriptTypeAgent:         func() any { return &AgentMessage{} },
	TranscriptTypeToolExecution: func() any { return &ToolExecution{} },
	TranscriptTypeSystem:        func() any { return &SystemMessage{} },
	TranscriptTypeProgress:      func() any { return &Progress{} },
}

// MarshalJSON flattens Data fields alongside "type" field.
func (t Transcript) MarshalJSON() ([]byte, error) {
	if t.Data == nil {
		return json.Marshal(map[string]any{"type": t.Type})
	}

	dataBytes, err := json.Marshal(t.Data)
	if err != nil {
		return nil, err
	}

	var dataMap map[string]json.RawMessage
	if err := json.Unmarshal(dataBytes, &dataMap); err != nil {
		return nil, err
	}

	typeBytes, err := json.Marshal(t.Type)
	if err != nil {
		return nil, err
	}
	dataMap["type"] = typeBytes

	return json.Marshal(dataMap)
}

// UnmarshalJSON dispatches to correct concrete type based on "type" field.
func (t *Transcript) UnmarshalJSON(data []byte) error {
	type typeExtractor struct {
		Type TranscriptType `json:"type"`
	}

	var extractor typeExtractor
	if err := json.Unmarshal(data, &extractor); err != nil {
		return err
	}

	t.Type = extractor.Type

	factory, found := transcriptTypeRegistry[t.Type]
	if found {
		typedData := factory()
		if err := json.Unmarshal(data, typedData); err != nil {
			slog.Error("failed to unmarshal transcript data",
				"type", t.Type, "error", err)
		} else {
			t.Data = typedData
			return nil
		}
	} else {
		slog.Error("unknown transcript type", "type", t.Type)
	}

	// Fallback to map for unknown types
	var mapData map[string]any
	if err := json.Unmarshal(data, &mapData); err != nil {
		return err
	}
	t.Data = mapData
	return nil
}
