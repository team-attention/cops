# Development Walkthrough

## Summary

Implemented custom JSON marshaling and unmarshaling for the `Record` struct in the C-Ops shared domain package, enabling type-safe serialization and deserialization of Claude Code JSONL log files. The implementation uses a type registry pattern for extensibility and handles all three record types (assistant, user, file-history-snapshot) with permissive error handling for unknown types and schema mismatches.

## Code Overview

### Modified Components

#### `Record` struct with custom JSON marshaling
- **Location**: `/Users/jayce/team-attention/cops/shared/domain/record.go`
- **Changes**: Added type registry pattern and custom JSON marshaling methods
- **Key Components**:
  - **Type Registry** (lines 43-52):
    - `recordTypeFactory` type alias for factory functions
    - `recordTypeRegistry` maps `RecordType` to factory functions that create typed struct instances
    - Enables adding new record types without modifying core unmarshaling logic

  - **`MarshalJSON()` method** (lines 54-84):
    - Flattens the `Data` field's contents to the top level of JSON output
    - Produces flat structure: `{"type":"...", ...data fields...}`
    - Handles nil `Data` by returning only the type field
    - Algorithm:
      1. Return `{"type":"..."}` if Data is nil
      2. Marshal Data to JSON bytes
      3. Unmarshal into `map[string]json.RawMessage` to preserve field values
      4. Add "type" field to the map
      5. Marshal combined map to produce final flat JSON

  - **`UnmarshalJSON()` method** (lines 86-140):
    - Extracts type field and creates appropriate typed struct instance
    - Falls back to `map[string]any` storage for unknown types
    - Uses permissive unmarshaling (ignores unknown fields, missing fields become zero values)
    - Logs errors using `slog.Error` with structured fields
    - Algorithm:
      1. Extract type field using temporary `typeExtractor` struct
      2. Set `r.Type` from extracted value
      3. Look up factory function in `recordTypeRegistry`
      4. If found: Create typed instance, unmarshal into it, set as `r.Data`
      5. If not found or unmarshal fails: Log error, unmarshal into `map[string]any`, set as `r.Data`
      6. Always return nil (permissive approach)

#### `MessageMetadata` struct
- **Location**: `/Users/jayce/team-attention/cops/shared/domain/record.go` (lines 23-36)
- **Changes**: None to structure, but this is embedded in UserRecord and AssistantRecord
- **Fields**:
  - `ParentUUID *string` - Optional parent message reference (pointer for null support)
  - `IsSidechain bool` - Indicates if message is part of a sidechain
  - `UserType RecordUserType` - User type (e.g., "external")
  - `SessionID string` - Session identifier
  - `Version string` - Claude Code version
  - `GitBranch string` - Git branch name
  - `UUID string` - Unique message identifier
  - `Timestamp time.Time` - Message timestamp

### New Components

#### `AssistantRecord`
- **Location**: `/Users/jayce/team-attention/cops/shared/domain/record_assistant.go`
- **Purpose**: Represents assistant (Claude) message records from JSONL logs
- **Structure**:
  - Embeds `MessageMetadata` inline (fields appear at top level in JSON)
  - `RequestID string` - API request identifier
  - `Message AssistantMessage` - Full assistant message with content blocks
- **Supporting Types**:
  - `AssistantMessage`: Contains model, ID, type, role, content blocks, stop reason, usage stats
  - `AssistantMessageContent`: Discriminated union supporting text, thinking, and tool_use content types
  - `AssistantMessageToolUseContent`: Tool invocation details (ID, name, input)
  - `AssistantMessageUsage`: Token usage statistics including cache metrics

#### `UserRecord`
- **Location**: `/Users/jayce/team-attention/cops/shared/domain/record_user.go`
- **Purpose**: Represents user message records from JSONL logs
- **Structure**:
  - Embeds `MessageMetadata` inline
  - `Message UserMessage` - User message with role and content
  - `IsMeta bool` - Indicates if this is a meta message
  - `ThinkingMetadata *UserRecordThinkingMetadata` - Optional thinking configuration (pointer for null support)
  - `Todos []*UserRecordTodo` - Optional task list items
- **Supporting Types**:
  - `UserMessage`: Simple role + content structure
  - `UserRecordThinkingMetadata`: Thinking level, disabled flag, triggers
  - `UserRecordThinkingMetadataTrigger`: Text range that triggered thinking
  - `UserRecordTodo`: Task content, status, active form

#### `FileHistorySnapshotRecord`
- **Location**: `/Users/jayce/team-attention/cops/shared/domain/record_file_history_snapshot.go`
- **Purpose**: Represents file history snapshot records from JSONL logs
- **Structure**:
  - `MessageID string` - Associated message identifier
  - `Snapshot FileHistorySnapshot` - File backup snapshot data
  - `IsSnapshotUpdate bool` - Indicates if this is an update to existing snapshot
- **Supporting Types**:
  - `FileHistorySnapshot`: Contains message ID and map of tracked file backups
  - `FileHistorySnapshotTrackedBackups`: Backup metadata (filename, version, time)

## Type Registry Pattern

The implementation uses a registry-based approach for extensibility:

```go
// Factory function type
type recordTypeFactory func() any

// Package-level registry mapping types to factory functions
var recordTypeRegistry = map[RecordType]recordTypeFactory{
    RecordTypeFileHistorySnapshot: func() any { return &FileHistorySnapshotRecord{} },
    RecordTypeUser:                func() any { return &UserRecord{} },
    RecordTypeMessage:             func() any { return &AssistantRecord{} },
}
```

**Benefits**:
1. Adding new record types only requires adding an entry to the registry
2. No modifications to `UnmarshalJSON` logic needed for new types
3. Clear separation between type definitions and unmarshaling logic
4. Factory pattern ensures fresh instances for each unmarshal operation

## How to Use

### Reading JSONL Files

```go
import (
    "bufio"
    "encoding/json"
    "os"
    "github.com/team-attention/cops/shared/domain"
)

// Read and parse JSONL file
file, _ := os.Open("session_log.jsonl")
defer file.Close()

scanner := bufio.NewScanner(file)
for scanner.Scan() {
    var record domain.Record
    if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
        // Handle error
        continue
    }

    // Type-switch on Data to access typed fields
    switch data := record.Data.(type) {
    case *domain.AssistantRecord:
        fmt.Printf("Assistant message: %s\n", data.Message.ID)
        // Access MessageMetadata fields directly
        fmt.Printf("Session: %s, UUID: %s\n", data.SessionID, data.UUID)

    case *domain.UserRecord:
        fmt.Printf("User message: %s\n", data.Message.Content)
        if data.ThinkingMetadata != nil {
            fmt.Printf("Thinking level: %s\n", data.ThinkingMetadata.Level)
        }

    case *domain.FileHistorySnapshotRecord:
        fmt.Printf("Snapshot for message: %s\n", data.MessageID)

    case map[string]any:
        // Unknown type - fallback storage
        fmt.Printf("Unknown record type: %s\n", record.Type)
    }
}
```

### Writing JSONL Files

```go
// Create typed record
record := domain.Record{
    Type: domain.RecordTypeUser,
    Data: &domain.UserRecord{
        MessageMetadata: domain.MessageMetadata{
            UUID:      "msg-123",
            SessionID: "session-456",
            Timestamp: time.Now(),
            // ... other fields
        },
        Message: domain.UserMessage{
            Role:    domain.UserMessageRoleUser,
            Content: "Hello, Claude!",
        },
        IsMeta: false,
    },
}

// Marshal to JSON
jsonBytes, err := json.Marshal(record)
// Result: {"type":"user","uuid":"msg-123","sessionId":"session-456",...,"message":{...},"isMeta":false}

// Write to JSONL file
file.Write(jsonBytes)
file.Write([]byte("\n"))
```

### Adding New Record Types

1. Define the record struct in a new file (e.g., `record_new_type.go`):
```go
type NewTypeRecord struct {
    MessageMetadata `bson:"inline"`  // If embedding metadata
    CustomField     string           `json:"customField" bson:"customField"`
}
```

2. Add constant to `RecordType`:
```go
const RecordTypeNewType RecordType = "new-type"
```

3. Register in `recordTypeRegistry`:
```go
var recordTypeRegistry = map[RecordType]recordTypeFactory{
    // ... existing entries
    RecordTypeNewType: func() any { return &NewTypeRecord{} },
}
```

No changes to `MarshalJSON` or `UnmarshalJSON` required!

## Testing

### Test Files Created
- **Location**: `/Users/jayce/team-attention/cops/shared/domain/record_test.go`
- **Lines of Code**: 783 lines
- **Framework**: Ginkgo v2 (BDD) + Gomega (matchers)

### Test Data Files
Real JSONL examples from Claude Code sessions:
- `record_assistant.jsonl` - 4 assistant message records
- `record_user.jsonl` - 8 user message records
- `record_file_history_snapshot.jsonl` - 7 file history snapshot records

### Test Coverage

| Category | Tests | Description |
|----------|-------|-------------|
| UnmarshalJSON - Assistant | 2 | Tool use content, thinking content |
| UnmarshalJSON - User | 3 | Meta flag, thinking metadata, todos |
| UnmarshalJSON - FileHistorySnapshot | 2 | Initial snapshot, snapshot update |
| UnmarshalJSON - Unknown types | 2 | Unknown type handling, missing type |
| UnmarshalJSON - Schema mismatches | 2 | Missing optional fields, extra fields |
| UnmarshalJSON - Error handling | 1 | Malformed JSON |
| MarshalJSON - Typed records | 3 | All three record types |
| MarshalJSON - Edge cases | 2 | Nil data, map data |
| Round-trip serialization | 6 | Individual + batch for all types |
| Integration | 3 | Full JSONL file parsing |

**Total**: 26 test cases covering all branches

### Verification Commands Run

```bash
# Build verification
cd /Users/jayce/team-attention/cops/shared/domain && go build .
# Result: PASS (no errors)

# Test execution
go test ./shared/domain/... -v
# Result: PASS - All Record tests pass (46 specs)
```

## Design Decisions

### 1. Why Type Registry Pattern?

**Decision**: Use map-based registry with factory functions instead of type switch

**Rationale**:
- Extensibility: New types can be added without modifying unmarshaling code
- Open-closed principle: Open for extension, closed for modification
- Centralized type mapping: All type associations in one place
- Factory pattern ensures fresh instances for each unmarshal

### 2. Why Permissive Unmarshaling?

**Decision**: Unknown types stored as `map[string]any` with error logging, not hard failure

**Rationale**:
- Forward compatibility: Future Claude Code versions can add new record types
- Graceful degradation: System continues processing even with unknown records
- Debugging support: Error logs capture unknown types for investigation
- Schema evolution: Allows partial schema mismatches (missing fields become zero values)

### 3. Why Flatten JSON Structure?

**Decision**: Merge Data fields to top level alongside "type" field

**Rationale**:
- Matches existing JSONL format from Claude Code
- Consistency with BSON `bson:",inline"` tag behavior for MongoDB
- Simpler JSON structure for debugging and manual inspection
- Backward compatibility with existing log files

### 4. Pointer vs Value Types for Struct Fields

**Decision**: Follow strict rules from `.agent/rules/go/go-struct.md`

**Applied Rules**:
- **Optional primitives**: Use pointers (`*string`, `*int`) with `omitempty`
  - Example: `ParentUUID *string`, `StopReason *string`, `BackupFileName *string`
- **Required primitives**: Use values (`string`, `int`, `bool`)
  - Example: `UUID string`, `IsMeta bool`, `Version int`
- **Optional structs**: Use pointers with `omitempty`
  - Example: `ThinkingMetadata *UserRecordThinkingMetadata`
- **Slices/maps**: Use values (nil equivalent to empty)
  - Example: `Content []*AssistantMessageContent`, `Todos []*UserRecordTodo`

**Rationale**:
- Explicit null vs absent distinction in JSON
- Type safety: Compiler catches missing nil checks
- Clear API: Callers know which fields can be nil

## Issues & Resolutions

| Issue | Resolution |
|-------|------------|
| Flattening embedded struct fields during marshaling | Marshal Data to JSON, unmarshal to map, add "type" field, re-marshal combined map |
| Type-safe unmarshaling without type switch | Implement type registry with factory functions that return typed struct instances |
| Handling unknown/future record types | Store as `map[string]any` with `slog.Error` logging for debugging while allowing processing to continue |
| Testing with real-world data | Use actual JSONL example files from Claude Code sessions as test data sources |
| MessageMetadata appearing at top level | Embedded structs naturally flatten with JSON tags; no special handling needed |

## Files Cleaned Up

### Deleted Files
- `shared/domain/record_file_history_snapshot_type.go` - Consolidated into `record_file_history_snapshot.go`
- `shared/domain/record_message_type.go` - Consolidated into `record_assistant.go`
- `shared/domain/record_user_type.go` - Consolidated into `record_user.go`
- `shared/domain/log_data.jsonl` - Replaced with type-specific JSONL files

**Rationale**: Original structure split type definitions across multiple files. Consolidation improves maintainability by keeping related types together in single files (one file per record type).

## Related Documentation

- **Requirements**: `.agent/artifacts/20251230-160734/01_requirements.md`
- **Implementation Plan**: `.agent/artifacts/20251230-160734/02_plan.md`
- **Code Review**: `.agent/artifacts/20251230-160734/03_review.md`
- **Project Rules**: `.agent/rules/go/go-struct.md`, `.agent/rules/go/go-backend.md`

## Future Enhancements

1. **Validation**: Add optional strict validation mode that returns errors for unknown types
2. **Schema Versioning**: Support multiple schema versions per record type
3. **Performance**: Optimize marshaling by caching reflection results
4. **Streaming**: Add streaming JSONL reader/writer utilities
5. **Migration Tools**: Create utilities to migrate old JSONL files to new formats
