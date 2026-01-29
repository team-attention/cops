package domain

import (
	"encoding/json"
	"log/slog"
	"time"
)

// TranscriptType represents the type discriminator for JSONL transcript entries.
type TranscriptType string

const (
	// TranscriptTypeUser represents user input in conversation tree.
	// Contains TreeNodeMeta (sessionId, uuid, timestamp).
	TranscriptTypeUser TranscriptType = "user"

	// TranscriptTypeAssistant represents AI assistant response in conversation tree.
	// Contains TreeNodeMeta (sessionId, uuid, timestamp).
	TranscriptTypeAssistant TranscriptType = "assistant"

	// TranscriptTypeSystem represents system-level metadata (e.g., turn duration).
	// Contains TreeNodeMeta (sessionId, uuid, timestamp).
	TranscriptTypeSystem TranscriptType = "system"

	// TranscriptTypeSummary represents conversation branch summary for context management.
	// Does NOT contain TreeNodeMeta. References conversation node via leafUuid.
	// Not included in session aggregation.
	TranscriptTypeSummary TranscriptType = "summary"

	// TranscriptTypeFileHistorySnapshot represents file change tracking entry.
	// Does NOT contain TreeNodeMeta. References message via messageId.
	// Contains snapshot.timestamp but no root-level timestamp.
	// Not included in session aggregation.
	TranscriptTypeFileHistorySnapshot TranscriptType = "file-history-snapshot"

	// TranscriptTypeProgress represents real-time progress updates (e.g., bash execution).
	// Contains TreeNodeMeta (sessionId, uuid, timestamp).
	TranscriptTypeProgress TranscriptType = "progress"
)

// TreeNodeMeta contains common fields for conversation tree nodes (user, assistant, system).
// These types form a tree structure linked by parentUuid.
type TreeNodeMeta struct {
	ParentUUID  *string   `json:"parentUuid" bson:"parentUuid"`
	UUID        string    `json:"uuid" bson:"uuid"`
	SessionID   string    `json:"sessionId" bson:"sessionId"`
	AgentID     *string   `json:"agentId,omitempty" bson:"agentId,omitempty"` // SubAgent identifier (nil for Main session)
	Timestamp   time.Time `json:"timestamp" bson:"timestamp"`
	Version     string    `json:"version" bson:"version"`
	CWD         string    `json:"cwd,omitempty" bson:"cwd,omitempty"`
	GitBranch   string    `json:"gitBranch,omitempty" bson:"gitBranch,omitempty"`
	Slug        string    `json:"slug,omitempty" bson:"slug,omitempty"`
	UserType    string    `json:"userType,omitempty" bson:"userType,omitempty"`
	IsSidechain bool      `json:"isSidechain" bson:"isSidechain"`
}

// Transcript represents a single JSONL entry with polymorphic data.
type Transcript struct {
	Type TranscriptType `json:"type" bson:"type"`
	Data any            `bson:",inline"`
}

// transcriptTypeFactory creates new instances of transcript data types.
type transcriptTypeFactory func() any

// transcriptTypeRegistry maps TranscriptType to factory functions.
var transcriptTypeRegistry = map[TranscriptType]transcriptTypeFactory{
	TranscriptTypeUser:                func() any { return &UserTranscript{} },
	TranscriptTypeAssistant:           func() any { return &AssistantTranscript{} },
	TranscriptTypeSystem:              func() any { return &SystemTranscript{} },
	TranscriptTypeSummary:             func() any { return &SummaryTranscript{} },
	TranscriptTypeFileHistorySnapshot: func() any { return &FileHistorySnapshotTranscript{} },
	TranscriptTypeProgress:            func() any { return &ProgressTranscript{} },
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
