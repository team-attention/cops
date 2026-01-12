package domain

import (
	"encoding/json"
	"log/slog"
	"time"
)

type RecordType string

const (
	RecordTypeFileHistorySnapshot RecordType = "file-history-snapshot"
	RecordTypeUser                RecordType = "user"
	RecordTypeMessage             RecordType = "assistant"
)

type RecordUserType string

const (
	RecordUserTypeExternal RecordUserType = "external"
)

type MessageMetadata struct {
	// ParentUUID null or string
	ParentUUID  *string        `json:"parentUuid" bson:"parentUuid"`
	IsSidechain bool           `json:"isSidechain" bson:"isSidechain"`
	UserType    RecordUserType `json:"userType" bson:"userType"`
	SessionID   string         `json:"sessionId" bson:"sessionId"`
	Version     string         `json:"version" bson:"version"`
	GitBranch   string         `json:"gitBranch" bson:"gitBranch"`
	UUID        string         `json:"uuid" bson:"uuid"`
	Timestamp   time.Time      `json:"timestamp" bson:"timestamp"`

	// CWD not required
	// CWD string `json:"cwd" bson:"cwd"`
}

type Record struct {
	Type RecordType `json:"type" bson:"type"`
	Data any        `bson:",inline"`
}

// recordTypeFactory is a function that returns a pointer to a new instance of the record data type.
type recordTypeFactory func() any

// recordTypeRegistry maps RecordType values to their corresponding factory functions.
// This registry enables extensible unmarshaling without modifying core logic.
var recordTypeRegistry = map[RecordType]recordTypeFactory{
	RecordTypeFileHistorySnapshot: func() any { return &FileHistorySnapshotRecord{} },
	RecordTypeUser:                func() any { return &UserRecord{} },
	RecordTypeMessage:             func() any { return &AssistantRecord{} },
}

// MarshalJSON serializes Record to JSON with flattened Data fields.
// The Data field's contents are merged at the top level alongside the "type" field.
// This produces flat JSON matching the JSONL format: {"type":"...", ...data fields...}
func (r Record) MarshalJSON() ([]byte, error) {
	// If Data is nil, marshal only the Type field as {"type":"..."}
	if r.Data == nil {
		return json.Marshal(map[string]any{"type": r.Type})
	}

	// Marshal the Data field to JSON bytes
	dataBytes, err := json.Marshal(r.Data)
	if err != nil {
		return nil, err
	}

	// Unmarshal Data JSON into map[string]json.RawMessage to preserve field values
	var dataMap map[string]json.RawMessage
	if err := json.Unmarshal(dataBytes, &dataMap); err != nil {
		return nil, err
	}

	// Add "type" field to the map with the Record's Type value
	typeBytes, err := json.Marshal(r.Type)
	if err != nil {
		return nil, err
	}
	dataMap["type"] = typeBytes

	// Marshal the combined map to produce final flat JSON
	return json.Marshal(dataMap)
}

// UnmarshalJSON deserializes flat JSON into Record with the appropriate typed Data.
// It reads the "type" field to determine which concrete type to use for Data,
// then unmarshals the entire JSON into that type (since record-specific fields
// are at the top level alongside "type").
// Unknown types are stored as map[string]any and logged as errors.
// Schema mismatches are handled permissively (missing fields become zero values).
func (r *Record) UnmarshalJSON(data []byte) error {
	// Define a temporary struct to extract just the Type field
	type typeExtractor struct {
		Type RecordType `json:"type"`
	}

	// Unmarshal data into typeExtractor to get the record type
	var extractor typeExtractor
	if err := json.Unmarshal(data, &extractor); err != nil {
		return err
	}

	// Set r.Type from extracted type
	r.Type = extractor.Type

	// Look up the type in recordTypeRegistry
	factory, found := recordTypeRegistry[r.Type]
	if found {
		// Call factory function to create new instance of the typed struct
		typedData := factory()

		// Unmarshal full data into the typed struct (permissive - ignores unknown fields)
		if err := json.Unmarshal(data, typedData); err != nil {
			// Log warning and fall through to map storage
			slog.Error("failed to unmarshal record data into typed struct",
				"type", r.Type,
				"error", err)
		} else {
			// Set r.Data to the typed struct pointer
			r.Data = typedData
			return nil
		}
	} else {
		// Log error for unknown type
		slog.Error("unknown record type encountered",
			"type", r.Type)
	}

	// Type not found in registry OR typed unmarshal failed
	// Create map[string]any to store raw data
	var mapData map[string]any
	if err := json.Unmarshal(data, &mapData); err != nil {
		return err
	}

	// Set r.Data to the map
	r.Data = mapData
	return nil
}
