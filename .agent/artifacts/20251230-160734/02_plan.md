# Implementation Plan: Custom JSON Marshaling for Record Struct

## Overview

This plan implements custom JSON marshaling and unmarshaling for the `Record` struct in `/Users/jayce/team-attention/cops/shared/domain/record.go`. The implementation enables type-safe serialization/deserialization of JSONL log files from Claude Code, where each line contains a record with a specific type and corresponding data structure.

The key challenge is that the JSON structure is flat (all fields at the top level alongside the "type" field), but the Go struct has a nested `Data` field. The custom marshaling will:
1. Flatten the `Data` field's contents into the top-level JSON during marshaling
2. Extract the correct typed struct from flat JSON during unmarshaling based on the "type" field

## Package Changes

None required. All necessary packages (`encoding/json`, `log/slog`) are part of the Go standard library.

## Implementation Steps

### Step 1: Implement Type Registry Pattern

**Files to Read**:
- `/Users/jayce/team-attention/cops/shared/domain/record.go`: Current Record struct and type definitions
- `/Users/jayce/team-attention/cops/shared/domain/record_assistant.go`: AssistantRecord struct definition
- `/Users/jayce/team-attention/cops/shared/domain/record_file_history_snapshot.go`: FileHistorySnapshotRecord struct definition
- `/Users/jayce/team-attention/cops/shared/domain/record_user.go`: UserRecord struct definition

#### `/Users/jayce/team-attention/cops/shared/domain/record.go`

**Description**:
Add a package-level type registry that maps RecordType to factory functions. This enables extensibility for adding new record types without modifying core unmarshaling logic.

```go
// recordTypeFactory is a function that returns a pointer to a new instance of the record data type.
type recordTypeFactory func() any

// recordTypeRegistry maps RecordType values to their corresponding factory functions.
// This registry enables extensible unmarshaling without modifying core logic.
var recordTypeRegistry = map[RecordType]recordTypeFactory{
	RecordTypeFileHistorySnapshot: func() any { return &FileHistorySnapshotRecord{} },
	RecordTypeUser:                func() any { return &UserRecord{} },
	RecordTypeMessage:             func() any { return &AssistantRecord{} },
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Registry contains file-history-snapshot | `RecordTypeFileHistorySnapshot` | `*FileHistorySnapshotRecord` instance | Registry lookup |
| Registry contains user | `RecordTypeUser` | `*UserRecord` instance | Registry lookup |
| Registry contains assistant | `RecordTypeMessage` | `*AssistantRecord` instance | Registry lookup |
| Registry lookup for unknown type | `"unknown-type"` | `nil` (not found) | Missing key |

---

### Step 2: Implement MarshalJSON Method

**Files to Read**:
- `/Users/jayce/team-attention/cops/shared/domain/record.go`: Current Record struct
- `/Users/jayce/team-attention/cops/shared/domain/record_assistant.jsonl`: Example assistant record JSON structure
- `/Users/jayce/team-attention/cops/shared/domain/record_user.jsonl`: Example user record JSON structure
- `/Users/jayce/team-attention/cops/shared/domain/record_file_history_snapshot.jsonl`: Example file-history-snapshot JSON structure

#### `/Users/jayce/team-attention/cops/shared/domain/record.go`

**Description**:
Implement `MarshalJSON` method that flattens the Data field's contents into the top-level JSON alongside the "type" field.

```go
// MarshalJSON serializes Record to JSON with flattened Data fields.
// The Data field's contents are merged at the top level alongside the "type" field.
// This produces flat JSON matching the JSONL format: {"type":"...", ...data fields...}
func (r Record) MarshalJSON() ([]byte, error) {
	// Implementation outline:
	// 1. If Data is nil, marshal only the Type field as {"type":"..."}
	// 2. Marshal the Data field to JSON bytes
	// 3. Unmarshal Data JSON into map[string]json.RawMessage to preserve field values
	// 4. Add "type" field to the map with the Record's Type value
	// 5. Marshal the combined map to produce final flat JSON
	// 6. Return the marshaled bytes
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Marshal AssistantRecord | Record{Type: "assistant", Data: &AssistantRecord{...}} | Flat JSON with "type":"assistant" and all AssistantRecord fields at top level | Happy path with embedded MessageMetadata |
| Marshal UserRecord | Record{Type: "user", Data: &UserRecord{...}} | Flat JSON with "type":"user" and all UserRecord fields at top level | Happy path with embedded MessageMetadata |
| Marshal FileHistorySnapshotRecord | Record{Type: "file-history-snapshot", Data: &FileHistorySnapshotRecord{...}} | Flat JSON with "type":"file-history-snapshot" and all fields at top level | Happy path without embedded struct |
| Marshal with nil Data | Record{Type: "user", Data: nil} | `{"type":"user"}` | Nil Data branch |
| Marshal with map[string]any Data | Record{Type: "unknown", Data: map[string]any{...}} | Flat JSON with "type" and map contents | Unknown type fallback |

---

### Step 3: Implement UnmarshalJSON Method

**Files to Read**:
- `/Users/jayce/team-attention/cops/shared/domain/record.go`: Current Record struct and registry
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logging conventions for error handling

#### `/Users/jayce/team-attention/cops/shared/domain/record.go`

**Description**:
Implement `UnmarshalJSON` method that extracts the type field, looks up the factory in the registry, creates the appropriate struct, and unmarshals the full JSON into it.

```go
// UnmarshalJSON deserializes flat JSON into Record with the appropriate typed Data.
// It reads the "type" field to determine which concrete type to use for Data,
// then unmarshals the entire JSON into that type (since record-specific fields
// are at the top level alongside "type").
// Unknown types are stored as map[string]any and logged as errors.
// Schema mismatches are handled permissively (missing fields become zero values).
func (r *Record) UnmarshalJSON(data []byte) error {
	// Implementation outline:
	// 1. Define a temporary struct to extract just the Type field:
	//    type typeExtractor struct { Type RecordType `json:"type"` }
	// 2. Unmarshal data into typeExtractor to get the record type
	// 3. If unmarshal fails, return error
	// 4. Set r.Type from extracted type
	// 5. Look up the type in recordTypeRegistry
	// 6. If type found in registry:
	//    a. Call factory function to create new instance of the typed struct
	//    b. Unmarshal full data into the typed struct (permissive - ignores unknown fields)
	//    c. If unmarshal fails, log warning and fall through to map storage
	//    d. Set r.Data to the typed struct pointer
	// 7. If type not found in registry OR typed unmarshal failed:
	//    a. Log error with slog.Error including the type value
	//    b. Create map[string]any to store raw data
	//    c. Unmarshal full data into the map
	//    d. Set r.Data to the map
	// 8. Return nil (permissive approach - always succeeds)
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Unmarshal assistant record | Valid assistant JSON from JSONL file | Record with Type="assistant", Data=*AssistantRecord with all fields populated | Happy path - registry hit |
| Unmarshal user record | Valid user JSON from JSONL file | Record with Type="user", Data=*UserRecord with all fields populated | Happy path - registry hit |
| Unmarshal file-history-snapshot | Valid snapshot JSON from JSONL file | Record with Type="file-history-snapshot", Data=*FileHistorySnapshotRecord | Happy path - registry hit |
| Unmarshal unknown type | `{"type":"new-type","field":"value"}` | Record with Type="new-type", Data=map[string]any{"type":"new-type","field":"value"} | Unknown type - map storage + error log |
| Unmarshal missing type | `{"field":"value"}` | Record with Type="", Data=map[string]any{...} | Empty type - map storage + error log |
| Unmarshal partial data | User JSON missing optional fields | Record with Data=*UserRecord with missing fields as zero values | Schema mismatch - permissive |
| Unmarshal extra fields | User JSON with extra unknown fields | Record with Data=*UserRecord (extra fields ignored) | Forward compatibility |
| Unmarshal invalid JSON | `{not valid json}` | Error returned | JSON parse error |

---

### Step 4: Implement Unit Tests with Ginkgo/Gomega

**Files to Read**:
- `/Users/jayce/team-attention/cops/shared/domain/message_suite_test.go`: Existing test suite setup pattern
- `/Users/jayce/team-attention/cops/shared/domain/message_test.go`: Existing BDD test structure and JSONL file reading pattern
- `/Users/jayce/team-attention/cops/shared/domain/record_assistant.jsonl`: Test data for assistant records
- `/Users/jayce/team-attention/cops/shared/domain/record_user.jsonl`: Test data for user records
- `/Users/jayce/team-attention/cops/shared/domain/record_file_history_snapshot.jsonl`: Test data for file-history-snapshot records

#### `/Users/jayce/team-attention/cops/shared/domain/record_test.go` (New File)

**Description**:
Create a comprehensive test file using Ginkgo/Gomega BDD framework. Tests will use actual JSONL files as test data sources.

```go
package domain_test

import (
	"bufio"
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/shared/domain"
)

var _ = Describe("Record", func() {
	// Test constants for JSONL file paths
	// const assistantTestDataFile = "record_assistant.jsonl"
	// const userTestDataFile = "record_user.jsonl"
	// const fileHistoryTestDataFile = "record_file_history_snapshot.jsonl"

	Describe("UnmarshalJSON", func() {
		Context("when unmarshaling assistant records", func() {
			// Implementation outline:
			// 1. Read record_assistant.jsonl file
			// 2. Parse each line as JSON
			// 3. Unmarshal into domain.Record
			// 4. Verify Type is RecordTypeMessage ("assistant")
			// 5. Verify Data is *AssistantRecord
			// 6. Verify specific fields (RequestID, Message.Model, MessageMetadata fields)

			It("parses assistant record with tool_use content from JSONL file", func() {
				// Implementation outline:
				// 1. Read line 2 from record_assistant.jsonl (contains tool_use)
				// 2. Unmarshal into domain.Record
				// 3. Assert Type equals RecordTypeMessage
				// 4. Assert Data is *AssistantRecord
				// 5. Assert RequestID is "req_011CWPJbziKELTTDQvopECdi"
				// 6. Assert MessageMetadata fields are populated (ParentUUID, SessionID, etc.)
				// 7. Assert Message.Content contains tool_use block
			})

			It("parses assistant record with thinking content from JSONL file", func() {
				// Implementation outline:
				// 1. Read line 3 from record_assistant.jsonl (contains thinking)
				// 2. Unmarshal into domain.Record
				// 3. Assert Type equals RecordTypeMessage
				// 4. Assert Data is *AssistantRecord
				// 5. Assert Message.Content contains thinking block
			})
		})

		Context("when unmarshaling user records", func() {
			// Implementation outline:
			// 1. Read record_user.jsonl file
			// 2. Parse each line as JSON
			// 3. Unmarshal into domain.Record
			// 4. Verify Type is RecordTypeUser ("user")
			// 5. Verify Data is *UserRecord

			It("parses user record with meta flag from JSONL file", func() {
				// Implementation outline:
				// 1. Read line 1 from record_user.jsonl (isMeta: true)
				// 2. Unmarshal into domain.Record
				// 3. Assert Type equals RecordTypeUser
				// 4. Assert Data is *UserRecord
				// 5. Assert IsMeta is true
				// 6. Assert MessageMetadata fields are populated
			})

			It("parses user record with thinkingMetadata from JSONL file", func() {
				// Implementation outline:
				// 1. Read line 4 from record_user.jsonl (has thinkingMetadata)
				// 2. Unmarshal into domain.Record
				// 3. Assert Data.ThinkingMetadata is not nil
				// 4. Assert ThinkingMetadata.Level is "high"
			})

			It("parses user record with todos from JSONL file", func() {
				// Implementation outline:
				// 1. Read line 8 from record_user.jsonl (has todos)
				// 2. Unmarshal into domain.Record
				// 3. Assert Data.Todos is not empty
				// 4. Assert todo content and status fields
			})
		})

		Context("when unmarshaling file-history-snapshot records", func() {
			// Implementation outline:
			// 1. Read record_file_history_snapshot.jsonl file
			// 2. Parse each line as JSON
			// 3. Unmarshal into domain.Record
			// 4. Verify Type is RecordTypeFileHistorySnapshot
			// 5. Verify Data is *FileHistorySnapshotRecord

			It("parses file-history-snapshot record from JSONL file", func() {
				// Implementation outline:
				// 1. Read line 1 from record_file_history_snapshot.jsonl
				// 2. Unmarshal into domain.Record
				// 3. Assert Type equals RecordTypeFileHistorySnapshot
				// 4. Assert Data is *FileHistorySnapshotRecord
				// 5. Assert MessageID is populated
				// 6. Assert Snapshot.MessageID matches
				// 7. Assert IsSnapshotUpdate is false
			})

			It("parses file-history-snapshot update record from JSONL file", func() {
				// Implementation outline:
				// 1. Read line 2 from record_file_history_snapshot.jsonl (isSnapshotUpdate: true)
				// 2. Unmarshal into domain.Record
				// 3. Assert IsSnapshotUpdate is true
				// 4. Assert Snapshot.TrackedFileBackups is not empty
			})
		})

		Context("when unmarshaling unknown record types", func() {
			It("stores unknown type as map[string]any and logs error", func() {
				// Implementation outline:
				// 1. Create JSON with unknown type: {"type":"future-type","customField":"value"}
				// 2. Unmarshal into domain.Record
				// 3. Assert no error returned (permissive)
				// 4. Assert Type equals "future-type"
				// 5. Assert Data is map[string]any
				// 6. Assert map contains "type" and "customField" keys
			})

			It("stores record with missing type as map[string]any", func() {
				// Implementation outline:
				// 1. Create JSON without type field: {"field":"value"}
				// 2. Unmarshal into domain.Record
				// 3. Assert no error returned (permissive)
				// 4. Assert Type is empty string
				// 5. Assert Data is map[string]any
			})
		})

		Context("when handling schema mismatches", func() {
			It("handles missing optional fields gracefully", func() {
				// Implementation outline:
				// 1. Create minimal user JSON with only required fields
				// 2. Unmarshal into domain.Record
				// 3. Assert Data is *UserRecord
				// 4. Assert optional fields (ThinkingMetadata, Todos) are nil/empty
			})

			It("ignores extra fields not in struct", func() {
				// Implementation outline:
				// 1. Create user JSON with extra unknown field
				// 2. Unmarshal into domain.Record
				// 3. Assert no error
				// 4. Assert Data is *UserRecord (extra field ignored)
			})
		})

		Context("when JSON is invalid", func() {
			It("returns error for malformed JSON", func() {
				// Implementation outline:
				// 1. Create invalid JSON: "{not valid"
				// 2. Unmarshal into domain.Record
				// 3. Assert error is returned
			})
		})
	})

	Describe("MarshalJSON", func() {
		Context("when marshaling typed records", func() {
			It("produces flat JSON for AssistantRecord", func() {
				// Implementation outline:
				// 1. Create Record with Type=RecordTypeMessage, Data=&AssistantRecord{...}
				// 2. Marshal to JSON
				// 3. Assert no error
				// 4. Parse result as map[string]any
				// 5. Assert "type" field equals "assistant"
				// 6. Assert AssistantRecord fields present at top level (requestId, message, etc.)
				// 7. Assert MessageMetadata fields present at top level (parentUuid, sessionId, etc.)
			})

			It("produces flat JSON for UserRecord", func() {
				// Implementation outline:
				// 1. Create Record with Type=RecordTypeUser, Data=&UserRecord{...}
				// 2. Marshal to JSON
				// 3. Assert "type" field equals "user"
				// 4. Assert UserRecord fields at top level (message, isMeta, etc.)
			})

			It("produces flat JSON for FileHistorySnapshotRecord", func() {
				// Implementation outline:
				// 1. Create Record with Type=RecordTypeFileHistorySnapshot, Data=&FileHistorySnapshotRecord{...}
				// 2. Marshal to JSON
				// 3. Assert "type" field equals "file-history-snapshot"
				// 4. Assert FileHistorySnapshotRecord fields at top level (messageId, snapshot, isSnapshotUpdate)
			})
		})

		Context("when marshaling nil Data", func() {
			It("produces JSON with only type field", func() {
				// Implementation outline:
				// 1. Create Record with Type=RecordTypeUser, Data=nil
				// 2. Marshal to JSON
				// 3. Assert result equals {"type":"user"}
			})
		})

		Context("when marshaling map[string]any Data", func() {
			It("produces flat JSON for unknown type stored as map", func() {
				// Implementation outline:
				// 1. Create Record with Type="unknown", Data=map[string]any{"field":"value"}
				// 2. Marshal to JSON
				// 3. Assert result contains "type":"unknown" and "field":"value" at top level
			})
		})
	})

	Describe("Round-trip serialization", func() {
		Context("with real JSONL data", func() {
			It("preserves assistant record through marshal/unmarshal cycle", func() {
				// Implementation outline:
				// 1. Read line from record_assistant.jsonl
				// 2. Unmarshal into domain.Record
				// 3. Marshal back to JSON
				// 4. Unmarshal again into new domain.Record
				// 5. Assert both Records have equal Type
				// 6. Assert both Records have same Data type (*AssistantRecord)
				// 7. Compare key fields (RequestID, Message.ID)
			})

			It("preserves user record through marshal/unmarshal cycle", func() {
				// Implementation outline:
				// 1. Read line from record_user.jsonl
				// 2. Perform round-trip marshal/unmarshal
				// 3. Assert Records are equivalent
			})

			It("preserves file-history-snapshot record through marshal/unmarshal cycle", func() {
				// Implementation outline:
				// 1. Read line from record_file_history_snapshot.jsonl
				// 2. Perform round-trip marshal/unmarshal
				// 3. Assert Records are equivalent
			})
		})

		Context("with all JSONL files", func() {
			It("preserves all assistant records through round-trip", func() {
				// Implementation outline:
				// 1. Read all lines from record_assistant.jsonl
				// 2. For each line, perform round-trip
				// 3. Assert no errors and data preserved
			})

			It("preserves all user records through round-trip", func() {
				// Implementation outline:
				// 1. Read all lines from record_user.jsonl
				// 2. For each line, perform round-trip
				// 3. Assert no errors and data preserved
			})

			It("preserves all file-history-snapshot records through round-trip", func() {
				// Implementation outline:
				// 1. Read all lines from record_file_history_snapshot.jsonl
				// 2. For each line, perform round-trip
				// 3. Assert no errors and data preserved
			})
		})
	})

	Describe("Integration with existing JSONL reading", func() {
		It("can parse entire assistant JSONL file", func() {
			// Implementation outline:
			// 1. Open record_assistant.jsonl
			// 2. Create scanner
			// 3. For each line:
			//    a. Unmarshal into domain.Record
			//    b. Assert no error
			//    c. Assert Type is RecordTypeMessage
			//    d. Assert Data is *AssistantRecord
			// 4. Assert correct number of records parsed (4 lines)
		})

		It("can parse entire user JSONL file", func() {
			// Implementation outline:
			// 1. Open record_user.jsonl
			// 2. Parse all lines into domain.Record slice
			// 3. Assert correct number of records (8 lines)
			// 4. Assert all have Type RecordTypeUser
		})

		It("can parse entire file-history-snapshot JSONL file", func() {
			// Implementation outline:
			// 1. Open record_file_history_snapshot.jsonl
			// 2. Parse all lines into domain.Record slice
			// 3. Assert correct number of records (7 lines)
			// 4. Assert all have Type RecordTypeFileHistorySnapshot
		})
	})
})
```

**Test Scenarios Summary**:

| Category | Test Count | Coverage |
| :------- | :--------- | :------- |
| UnmarshalJSON - Assistant | 2 | Happy path with different content types |
| UnmarshalJSON - User | 3 | Meta flag, thinkingMetadata, todos |
| UnmarshalJSON - FileHistorySnapshot | 2 | Initial snapshot, snapshot update |
| UnmarshalJSON - Unknown types | 2 | Unknown type, missing type |
| UnmarshalJSON - Schema mismatches | 2 | Missing optional, extra fields |
| UnmarshalJSON - Invalid JSON | 1 | Error handling |
| MarshalJSON - Typed records | 3 | All three record types |
| MarshalJSON - Edge cases | 2 | Nil data, map data |
| Round-trip - Individual | 3 | One per record type |
| Round-trip - All records | 3 | Full file verification |
| Integration | 3 | Full JSONL file parsing |

---

## Implementation Sequence

Execute the steps in this order:

1. **Step 1**: Add type registry to `record.go`
   - Add `recordTypeFactory` type alias
   - Add `recordTypeRegistry` package-level variable

2. **Step 2**: Implement `MarshalJSON` method
   - Handles nil Data case
   - Flattens typed struct fields
   - Flattens map[string]any fields

3. **Step 3**: Implement `UnmarshalJSON` method
   - Extracts type field
   - Looks up factory in registry
   - Creates typed struct or falls back to map
   - Logs errors for unknown types

4. **Step 4**: Create test file with Ginkgo/Gomega
   - Follow existing test patterns in `message_test.go`
   - Use actual JSONL files as test data
   - Cover all branches and edge cases

5. **Verification**: Run tests
   - Execute `go test ./shared/domain/... -v` from project root
   - Ensure all tests pass
   - Verify no regressions in existing `message_test.go` tests

## Quality Checklist

- [x] Every function has a concrete signature
- [x] Detailed algorithm explanation included as comments in function bodies
- [x] Every function has test scenarios covering all branches
- [x] No "or" statements leaving choices to Execute Agent
- [x] All packages are selected (standard library only)
- [x] Execution order is clear and dependencies are explicit
