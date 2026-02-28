// Package session provides provider-agnostic session models for AI agent sessions.
//
// # Design Goals
//
// This package abstracts session data from Claude Code-specific structures,
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
//	Session (discriminated union)
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
package session

import (
	"encoding/json"
	"log/slog"
	"time"
)

// SessionType represents the type discriminator for v2 session entries.
type SessionType string

const (
	// SessionTypeHuman represents user input in conversation tree.
	// Contains only text and image content blocks (no tool_result).
	SessionTypeHuman SessionType = "human"

	// SessionTypeAgent represents AI agent response in conversation tree.
	// Contains text, thinking, and tool_call references (actual tool calls are in ToolExecution).
	SessionTypeAgent SessionType = "agent"

	// SessionTypeToolExecution represents a tool call with its result.
	// Independent type that links to the originating agent via SourceAgentUUID.
	SessionTypeToolExecution SessionType = "tool_execution"

	// SessionTypeSystem represents system-level metadata.
	// Uses Subtype to distinguish: turn_duration, summary, file_snapshot.
	SessionTypeSystem SessionType = "system"

	// SessionTypeProgress represents real-time progress updates during tool execution.
	// Links to ToolExecution via ToolExecutionID.
	SessionTypeProgress SessionType = "progress"
)

// TreeNodeMeta contains common fields for conversation tree nodes.
// These types form a tree structure linked by ParentUUID.
type TreeNodeMeta struct {
	ParentUUID         *string   `json:"parentUuid,omitempty" bson:"parentUuid,omitempty"`
	UUID               string    `json:"uuid" bson:"uuid"`
	SessionID          string    `json:"sessionId" bson:"sessionId"`
	AgentID            *string   `json:"agentId,omitempty" bson:"agentId,omitempty"`                     // SubAgent identifier (nil for Main session)
	SpawnedByToolUseID *string   `json:"spawnedByToolUseId,omitempty" bson:"spawnedByToolUseId,omitempty"` // Tool Use ID that spawned this SubAgent
	Timestamp          time.Time `json:"timestamp" bson:"timestamp"`
	Provider           string    `json:"provider" bson:"provider"` // "claude_code" | "gemini_cli" | "open_code"
	Version            string    `json:"version,omitempty" bson:"version,omitempty"`
	GitBranch          string    `json:"gitBranch,omitempty" bson:"gitBranch,omitempty"`
	IsSidechain        bool      `json:"isSidechain,omitempty" bson:"isSidechain,omitempty"`
}

// Session represents a single entry with polymorphic data.
type Session struct {
	Type SessionType `json:"type" bson:"type"`
	Data any         `bson:"-"` // Handled via mongoschema MarshalBSON/UnmarshalBSON
}

// sessionTypeFactory creates new instances of session data types.
type sessionTypeFactory func() any

// sessionTypeRegistry maps SessionType to factory functions.
var sessionTypeRegistry = map[SessionType]sessionTypeFactory{
	SessionTypeHuman:         func() any { return &HumanMessage{} },
	SessionTypeAgent:         func() any { return &AgentMessage{} },
	SessionTypeToolExecution: func() any { return &ToolExecution{} },
	SessionTypeSystem:        func() any { return &SystemMessage{} },
	SessionTypeProgress:      func() any { return &Progress{} },
}

// MarshalJSON flattens Data fields alongside "type" field.
func (s Session) MarshalJSON() ([]byte, error) {
	if s.Data == nil {
		return json.Marshal(map[string]any{"type": s.Type})
	}

	dataBytes, err := json.Marshal(s.Data)
	if err != nil {
		return nil, err
	}

	var dataMap map[string]json.RawMessage
	if err := json.Unmarshal(dataBytes, &dataMap); err != nil {
		return nil, err
	}

	typeBytes, err := json.Marshal(s.Type)
	if err != nil {
		return nil, err
	}
	dataMap["type"] = typeBytes

	return json.Marshal(dataMap)
}

// UnmarshalJSON dispatches to correct concrete type based on "type" field.
func (s *Session) UnmarshalJSON(data []byte) error {
	type typeExtractor struct {
		Type SessionType `json:"type"`
	}

	var extractor typeExtractor
	if err := json.Unmarshal(data, &extractor); err != nil {
		return err
	}

	s.Type = extractor.Type

	factory, found := sessionTypeRegistry[s.Type]
	if found {
		typedData := factory()
		if err := json.Unmarshal(data, typedData); err != nil {
			slog.Error("failed to unmarshal session data",
				"type", s.Type, "error", err)
		} else {
			s.Data = typedData
			return nil
		}
	} else {
		slog.Error("unknown session type", "type", s.Type)
	}

	// Fallback to map for unknown types
	var mapData map[string]any
	if err := json.Unmarshal(data, &mapData); err != nil {
		return err
	}
	s.Data = mapData
	return nil
}
