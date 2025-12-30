# Requirements

## Request Summary

Implement custom JSON marshaling and unmarshaling methods for the `Record` struct in `/Users/jayce/team-attention/cops/shared/domain/record.go`. The Record struct currently has a `Type` field (RecordType) and a `Data` field (any). The custom marshaling will enable type-safe serialization/deserialization of JSONL log files from Claude Code, where each line contains a record with a specific type and corresponding data structure.

## Acceptance Criteria

- [ ] Implement `MarshalJSON()` method on Record struct that serializes the Data field inline with the Type field (flat JSON structure)
- [ ] Implement `UnmarshalJSON()` method on Record struct that deserializes JSON into the appropriate concrete type based on the Type field value
- [ ] Support three record types with their mappings:
  - `"file-history-snapshot"` → FileHistorySnapshotRecord
  - `"user"` → UserRecord
  - `"assistant"` → AssistantRecord
- [ ] Implement a type registry pattern to make adding new record types extensible without modifying core unmarshaling logic
- [ ] Handle unknown record types by storing data as `map[string]any` and logging an error (permissive approach)
- [ ] Handle schema mismatches permissively by unmarshaling available fields and leaving missing fields as zero values
- [ ] All JSON fields should be flattened at the top level (matching the `bson:",inline"` tag behavior)
- [ ] Maintain compatibility with existing JSONL files that have the current structure
- [ ] Add unit tests using Ginkgo (BDD test framework) and Gomega (matcher library) covering:
  - Successful marshaling/unmarshaling for all three record types using actual JSONL example files
  - Unknown type handling (map storage + error logging)
  - Schema mismatch scenarios (partial data)
  - Round-trip serialization (marshal then unmarshal produces equivalent result)
  - Tests must use the actual JSONL example files located in `/Users/jayce/team-attention/cops/shared/domain/`
  - Tests must follow BDD style with Describe/Context/It blocks

## Scope

### In Scope
- Custom MarshalJSON method for Record struct
- Custom UnmarshalJSON method for Record struct
- Type registry pattern implementation for extensibility
- Error handling for unknown types (map[string]any storage + logging)
- Permissive unmarshaling for schema mismatches
- Unit tests for marshaling/unmarshaling logic using Ginkgo and Gomega
- Support for three existing record types: file-history-snapshot, user, assistant

### Out of Scope
- Modifying existing record type structs (FileHistorySnapshotRecord, UserRecord, AssistantRecord)
- Changing the JSONL file format or structure
- Adding new record types (only infrastructure for future additions)
- Modifying Record struct fields (Type and Data remain unchanged)
- Database/BSON marshaling changes (only JSON marshaling)
- Migration of existing JSONL files

## Constraints

- Must preserve the flat JSON structure where record-specific fields appear at the top level alongside the "type" field
- Must not break existing code that reads/writes Record structs
- Error logging must be clear enough for debugging unknown types or schema issues
- Type registry should be initialized at package level to avoid runtime overhead
- Implementation must work with the existing `bson:",inline"` tag behavior for MongoDB compatibility
- Tests must be written using Ginkgo (BDD test framework) and Gomega (matcher library)
- Test code must follow BDD conventions with Describe/Context/It blocks for better readability and maintainability

## Additional Context

### Current Code Structure

**File**: `/Users/jayce/team-attention/cops/shared/domain/record.go`

```go
type RecordType string

const (
    RecordTypeFileHistorySnapshot RecordType = "file-history-snapshot"
    RecordTypeUser                RecordType = "user"
    RecordTypeMessage             RecordType = "assistant"
)

type Record struct {
    Type RecordType `json:"type" bson:"type"`
    Data any        `bson:",inline"`
}
```

### Existing Record Type Structs

1. **AssistantRecord** (`/Users/jayce/team-attention/cops/shared/domain/record_assistant.go`)
   - Embeds MessageMetadata inline
   - Contains RequestID (string) and Message (AssistantMessage)

2. **FileHistorySnapshotRecord** (`/Users/jayce/team-attention/cops/shared/domain/record_file_history_snapshot.go`)
   - Contains MessageID (string), Snapshot (FileHistorySnapshot), IsSnapshotUpdate (bool)

3. **UserRecord** (`/Users/jayce/team-attention/cops/shared/domain/record_user.go`)
   - Embeds MessageMetadata inline
   - Contains Message (UserMessage), IsMeta (bool), ThinkingMetadata (*UserRecordThinkingMetadata), Todos ([]*UserRecordTodo)

### Example JSONL Files

Test code MUST use the actual JSONL example files located in `/Users/jayce/team-attention/cops/shared/domain/`:

1. **`record_assistant.jsonl`** - Example records for assistant type
   - Contains actual assistant message records with full MessageMetadata, RequestID, and Message structures
   - Example structure: `{"parentUuid":"...", "isSidechain":false, "userType":"external", "message":{...}, "requestId":"...", "type":"assistant", "uuid":"...", "timestamp":"..."}`

2. **`record_user.jsonl`** - Example records for user type
   - Contains actual user message records with MessageMetadata, IsMeta, ThinkingMetadata, and Todos
   - Example structure: `{"parentUuid":null, "isSidechain":false, "userType":"external", "type":"user", "message":{...}, "isMeta":true, "uuid":"...", "timestamp":"...", "thinkingMetadata":{...}, "todos":[]}`

3. **`record_file_history_snapshot.jsonl`** - Example records for file-history-snapshot type
   - Contains actual file history snapshot records with MessageID, Snapshot, and IsSnapshotUpdate
   - Example structure: `{"type":"file-history-snapshot", "messageId":"...", "snapshot":{...}, "isSnapshotUpdate":false}`

These files contain real-world examples from Claude Code sessions and should be used in unit tests to ensure the marshaling/unmarshaling implementation works correctly with actual data.

### Design Pattern

Implement a type registry pattern where:
- A package-level registry maps RecordType to a factory function that returns a pointer to the appropriate struct
- UnmarshalJSON looks up the type in the registry and creates the correct struct instance
- New record types can be registered by adding an entry to the registry without modifying unmarshal logic
- Unknown types default to map[string]any storage with error logging

### Related Files

**Implementation Files:**
- `/Users/jayce/team-attention/cops/shared/domain/record.go` - Main Record struct (modification target)
- `/Users/jayce/team-attention/cops/shared/domain/record_assistant.go` - AssistantRecord definition
- `/Users/jayce/team-attention/cops/shared/domain/record_file_history_snapshot.go` - FileHistorySnapshotRecord definition
- `/Users/jayce/team-attention/cops/shared/domain/record_user.go` - UserRecord definition
- `/Users/jayce/team-attention/cops/shared/domain/common.go` - MessageMetadata and other common types

**Test Data Files (must be used in unit tests):**
- `/Users/jayce/team-attention/cops/shared/domain/record_assistant.jsonl` - Real assistant record examples
- `/Users/jayce/team-attention/cops/shared/domain/record_file_history_snapshot.jsonl` - Real file history snapshot examples
- `/Users/jayce/team-attention/cops/shared/domain/record_user.jsonl` - Real user record examples

**Testing Framework:**
- Ginkgo - BDD test framework for Go (https://onsi.github.io/ginkgo/)
- Gomega - Matcher/assertion library for Go (https://onsi.github.io/gomega/)
- Tests should use Describe/Context/It blocks following BDD conventions
- Use Gomega matchers (e.g., `Expect(...).To(Equal(...))`) for assertions

## Questions Resolved

| Question | Answer |
|----------|--------|
| Should all records have an explicit "type" field in JSON? | Yes, ALL JSONL files have the "type" field, including assistant records. Expect all records to have an explicit "type" field. |
| How should we handle unknown record types? | Store as map[string]any and log an error. This allows processing to continue while capturing issues for debugging. |
| Are there future record types planned? Should we design for extensibility? | Yes, design for extensibility using a type registry pattern to make adding new types easy without modifying core logic. |
| What should happen if Type and Data don't match during unmarshaling? | Unmarshal what we can, leave missing fields as zero values. Use a permissive approach rather than strict validation. |
| Should MessageMetadata fields be flattened in JSON? | Yes, preserve the flattened structure where embedded struct fields appear at the top level, matching the `bson:",inline"` tag behavior. |
| Do we need backward compatibility with different JSONL structures? | Yes, maintain compatibility with existing JSONL files that conform to the current structure shown in the examples. |
