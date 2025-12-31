# Development Walkthrough

## Summary

Migrated the C-Ops codebase from the legacy `SessionRecord` model (defined in `session.go` and `message.go`) to the new `Record` model with custom JSON marshaling. This migration involved updating 23 files, removing 184 lines of legacy code, and establishing a flattened JSONL storage format that matches Claude Code's native output. The new model supports three record types (user, assistant, file-history-snapshot) with type-safe data access through discriminated unions.

## Code Overview

### Deleted Components

#### Legacy Domain Models
- **Location**: `shared/domain/session.go` (deleted)
- **Purpose**: Old `SessionRecord` struct with nested `type` and `data` fields
- **Reason for Deletion**: Replaced by new `Record` model with custom JSON marshaling that automatically flattens fields

- **Location**: `shared/domain/message.go` (deleted)
- **Purpose**: Old `Message` and `MessageContent` types with complex nested structure
- **Reason for Deletion**: Replaced by type-specific records (`UserRecord`, `AssistantRecord`) embedded in new `Record.Data` field

- **Location**: `shared/domain/message_suite_test.go` (deleted)
- **Purpose**: Tests for legacy `Message` types
- **Reason for Deletion**: No longer needed after model deletion

### Modified Components

#### MongoDB Schema Constants
- **Location**: `shared/domain/mongoschema/session_record.go`
- **Changes**: Updated all field name constants to match the new flattened Record model structure
- **Key Updates**:
  - Added `RecordTypeField`, `RecordParentUUIDField`, `RecordIsSidechainField`, etc. (flattened field names)
  - Removed references to nested `data` object
  - Added composite path constants for nested queries (e.g., `MessageUsageInputTokensPath`)
  - Kept collection name as `sessionRecords` for continuity
- **Pattern**: Field naming follows `Record<FieldName>Field` for root-level fields, `Message<FieldName>Field` for message-level fields

#### Protobuf Definitions
- **Location**: `idl/protobuf/aggregation/v1/aggregation.proto`
- **Changes**: Complete restructure to match the new Record model
- **Key Updates**:
  - Replaced `SessionRecord` message with new `Record` message using `oneof` for type discrimination
  - Added `RecordType` enum with `RECORD_TYPE_USER`, `RECORD_TYPE_ASSISTANT`, `RECORD_TYPE_FILE_HISTORY_SNAPSHOT`
  - Created separate messages for each record type: `UserRecordData`, `AssistantRecordData`, `FileHistorySnapshotRecordData`
  - Flattened structure to match JSONL format (no nested `type`/`data` wrapper)
  - Removed `cwd` field from `MessageMetadata` as it's not in the new model

- **Location**: `idl/protobuf/dashboard/v1/dashboard.proto`
- **Changes**: Updated `SessionDetail.records` field type
- **Before**: `repeated aggregation.v1.SessionRecord records = 9;`
- **After**: `repeated aggregation.v1.Record records = 9;`

#### Generated gRPC Stubs
- **Location**: `shared/gen/grpcstub/aggregation/v1/aggregation.pb.go`
- **Changes**: Regenerated from updated protobuf definitions using `buf generate`
- **Impact**: 1242 lines changed (complete restructure of generated types)

- **Location**: `shared/gen/grpcstub/dashboard/v1/dashboard.pb.go`
- **Changes**: Updated to reference new `Record` type instead of `SessionRecord`

- **Location**: `web/src/gen/grpcstub/aggregation/v1/aggregation_pb.ts` and `dashboard/v1/dashboard_pb.ts`
- **Changes**: TypeScript definitions regenerated for web dashboard

#### CLI Parser Interface and Implementation
- **Location**: `cli/internal/service/tracking/outbound/parser/parser_port.go`
- **Changes**: Updated interface signature
- **Before**: `ParseSessionFiles(claudeProjectDir string) ([]*domain.SessionRecord, error)`
- **After**: `ParseSessionFiles(claudeProjectDir string) ([]*domain.Record, error)`

- **Location**: `cli/internal/service/tracking/outbound/parser/jsonl/jsonl_parser.go`
- **Changes**: Updated parser to use new `Record` model
- **Key Updates**:
  - Changed unmarshaling to use `domain.Record` instead of `domain.SessionRecord`
  - Removed type filtering logic - now parses ALL record types including `file-history-snapshot`
  - Leverages `Record.UnmarshalJSON()` which automatically dispatches to correct type based on `type` field
  - Comment added: "Parse ALL record types (user, assistant, file-history-snapshot)"

#### API Aggregation Repository
- **Location**: `api/internal/service/aggregation/outbound/repository/port.go`
- **Changes**: Updated `LogBatch` struct
- **Before**: `Records []shareddomain.SessionRecord`
- **After**: `Records []shareddomain.Record`

- **Location**: `api/internal/service/aggregation/outbound/repository/mongodb/adapter.go`
- **Changes**: Simplified MongoDB storage using Record's custom marshaling
- **Key Implementation**:
  ```go
  func toDocument(record shareddomain.Record, projectObjID bson.ObjectID) bson.M {
      // 1. Marshal record to JSON using record.MarshalJSON() (produces flat JSON)
      jsonBytes, err := record.MarshalJSON()

      // 2. Unmarshal JSON into bson.M
      var doc bson.M
      sonic.Unmarshal(jsonBytes, &doc)

      // 3. Add projectId field
      doc[mongoschema.RecordProjectIDField] = projectObjID

      return doc
  }
  ```
- **Why This Works**: `Record.MarshalJSON()` automatically flattens type-specific fields to root level, matching JSONL format

#### Dashboard Repository
- **Location**: `api/internal/service/dashboard/outbound/repository/dashboard_repo_port.go`
- **Changes**: Updated `SessionDetail.Records` type
- **Before**: `Records []shareddomain.SessionRecord`
- **After**: `Records []shareddomain.Record`

- **Location**: `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`
- **Changes**: Updated `GetSession` to reconstruct `Record` from MongoDB documents
- **Key Implementation**:
  - Read `type` field from MongoDB document to determine `RecordType`
  - Marshal document back to JSON
  - Unmarshal into `domain.Record` which uses custom `UnmarshalJSON` to dispatch to correct type
  - Extract usage data from `AssistantRecord.Message.Usage` for token aggregation

#### gRPC Handlers and Converters
- **Location**: `api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go`
- **Changes**: Updated `parseJSONLLines` method
- **Key Updates**:
  - Changed slice type from `[]shareddomain.SessionRecord` to `[]shareddomain.Record`
  - Unmarshaling uses `sonic.Unmarshal` into `domain.Record` which handles type dispatch automatically

- **Location**: `api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go`
- **Changes**: Complete rewrite of conversion logic to map `domain.Record` to proto `Record`
- **Removed Functions**:
  - `toProtoSessionRecord()` (old conversion logic)
  - `convertSessionType()`, `convertMessage()`, `convertContentBlocks()` (old helpers)
- **New Functions**:
  - `toProtoRecords()` - converts slice of domain records
  - `toProtoRecord()` - dispatches based on record type using type switch
  - `toProtoUserRecordData()` - converts `UserRecord` to proto
  - `toProtoAssistantRecordData()` - converts `AssistantRecord` to proto
  - `toProtoFileHistorySnapshotRecordData()` - converts `FileHistorySnapshotRecord` to proto
- **Type Dispatch Pattern**:
  ```go
  switch data := r.Data.(type) {
  case *shareddomain.UserRecord:
      protoRecord.Data = &aggregationv1.Record_UserData{...}
  case *shareddomain.AssistantRecord:
      protoRecord.Data = &aggregationv1.Record_AssistantData{...}
  case *shareddomain.FileHistorySnapshotRecord:
      protoRecord.Data = &aggregationv1.Record_FileHistorySnapshotData{...}
  }
  ```

#### Test Updates
- **Location**: `api/internal/service/aggregation/inbound/grpc/connectrpc/handler_test.go`
- **Changes**: Updated all tests to use `domain.Record` instead of `domain.SessionRecord`
- **Pattern**: Tests now verify correct `RecordType` and type assertion on `Record.Data`

- **Location**: `daemon/internal/service/logwatcher/log_service_test.go`
- **Changes**: Updated integration tests
- **Key Updates**:
  - Replaced `shareddomain.SessionRecord` with `shareddomain.Record`
  - Updated type comparisons: `SessionTypeUser` → `RecordTypeUser`, `SessionTypeAssistant` → `RecordTypeMessage`
  - Added type assertions to access record data: `record.Data.(*shareddomain.UserRecord)`

## Key Design Decisions

### 1. Custom JSON Marshaling Strategy

**Decision**: Implement custom `MarshalJSON` and `UnmarshalJSON` on the `Record` type to automatically flatten/unflatten type-specific fields.

**Rationale**:
- Claude Code JSONL files store all fields at root level with `type` as discriminator
- MongoDB should mirror this flattened structure for query simplicity
- Custom marshaling centralizes flattening logic in one place instead of manual flattening in every adapter

**Implementation**: `Record.MarshalJSON()` in `shared/domain/record.go` flattens `Data` fields to root level; `Record.UnmarshalJSON()` reads `type` field and dispatches to type-specific unmarshal using registry pattern.

### 2. Type Registry Pattern

**Decision**: Use a registry map to associate `RecordType` values with factory functions.

**Implementation**:
```go
var recordTypeRegistry = map[RecordType]recordTypeFactory{
    RecordTypeFileHistorySnapshot: func() any { return &FileHistorySnapshotRecord{} },
    RecordTypeUser:                func() any { return &UserRecord{} },
    RecordTypeMessage:             func() any { return &AssistantRecord{} },
}
```

**Benefits**:
- Extensible: new record types can be added by registering in the map
- Type-safe: factory function ensures correct type instantiation
- Centralized: unmarshaling logic doesn't need modification for new types

### 3. Removed CWD Field

**Decision**: Remove `CWD` field from `MessageMetadata` struct.

**Rationale**: Reviewing Claude Code's actual JSONL output and the new Record model implementation revealed that `cwd` is not a standard field in the metadata. The field was present in the old `SessionRecord` model but is not part of the canonical Record structure.

**Impact**: Protobuf definitions, generated code, and all usages updated to remove `cwd` references.

### 4. Removed Type Filtering in Parser

**Decision**: Parse ALL record types including `file-history-snapshot`, removing the old filter that only parsed `user` and `assistant` records.

**Rationale**:
- `file-history-snapshot` records provide valuable context about file backup state
- Filtering was arbitrary and limited system visibility
- Storage cost is minimal, and filtering can be done at query time if needed

**Implementation**: JSONL parser now appends all successfully parsed records regardless of type.

### 5. MongoDB Field Naming Conventions

**Decision**: Use structured naming pattern for schema constants.

**Pattern**:
- Root-level fields: `Record<FieldName>Field` (e.g., `RecordTypeField`, `RecordSessionIDField`)
- Nested object fields: `<Object><FieldName>Field` (e.g., `MessageModelField`, `UsageInputTokensField`)
- Composite paths: `<Object><Nested><Field>Path` (e.g., `MessageUsageInputTokensPath`)

**Rationale**: Clear namespacing prevents naming collisions and makes MongoDB query paths explicit and self-documenting.

### 6. Legacy Alias Cleanup

**Decision**: Remove legacy `SessionRecordCollectionName` alias constant from schema file.

**Note**: While some interface and struct names still contain "SessionRecord" (e.g., `SessionRecordRepositoryPort`, `MongoSessionRecordRepository`), these are documented as candidates for future renaming. The requirements document explicitly stated that interface names could be kept for minimal change during migration.

## Testing

### Unit Tests Updated
- **File**: `api/internal/service/aggregation/inbound/grpc/connectrpc/handler_test.go`
- **Coverage**: Tests for parsing user records, assistant records, and file-history-snapshot records
- **Key Test Cases**:
  - Valid user record parsing → `Record` with `RecordTypeUser` and `*UserRecord` data
  - Valid assistant record parsing → `Record` with `RecordTypeMessage` and `*AssistantRecord` data
  - Valid file-history-snapshot parsing → `Record` with `RecordTypeFileHistorySnapshot` and `*FileHistorySnapshotRecord` data
  - Empty lines → skipped
  - Malformed JSON → error in `parseErrors` slice
  - Mixed valid/invalid → valid records parsed, errors collected

### Integration Tests Updated
- **File**: `daemon/internal/service/logwatcher/log_service_test.go`
- **Changes**: Updated to use `domain.Record` with type assertions for data access
- **Pattern**: Tests verify record type, then assert on `record.Data.(*shareddomain.UserRecord)` to access typed fields

### Verification Commands Run

```bash
# Generate protobuf code
cd idl/protobuf && buf generate
# Result: SUCCESS - Generated 4 files (aggregation.pb.go, dashboard.pb.go, TS definitions)

# Build all modules
go build ./cli/... ./api/... ./daemon/... ./shared/...
# Result: SUCCESS - All modules compile without errors

# Run all tests
go test ./...
# Result: SUCCESS - All tests pass

# Verify no legacy type references remain
grep -r "SessionRecord" --include="*.go" .
# Result: Only references in generated proto files and intentionally preserved interface/struct names
```

## Issues & Resolutions

| Issue | Resolution |
| ----- | ---------- |
| Proto field naming used `cwd` which doesn't exist in new model | Removed `cwd` field from `MessageMetadata` in protobuf definitions and regenerated stubs |
| MongoDB adapter needed to flatten Record for storage | Leveraged `Record.MarshalJSON()` which automatically produces flat JSON matching JSONL format |
| Dashboard repo needed to reconstruct Record from flat MongoDB documents | Used `Record.UnmarshalJSON()` which reads `type` field and dispatches to correct type constructor via registry |
| Legacy schema constants used `SessionRecord` prefix | Updated all constants to use `Record` prefix with structured naming pattern |
| Parser was filtering out `file-history-snapshot` records | Removed type filtering logic; parser now accepts all record types |
| Type assertions needed for accessing Record.Data | Added type switches in converters and type assertions in tests: `data.(*shareddomain.UserRecord)` |

## Data Model Comparison

### Old Model (SessionRecord)

```json
{
  "type": "user",
  "data": {
    "sessionId": "abc123",
    "message": {...},
    "cwd": "/path/to/project"
  }
}
```

**Storage**: Nested structure with separate `type` and `data` fields

### New Model (Record)

```json
{
  "type": "user",
  "sessionId": "abc123",
  "message": {...}
}
```

**Storage**: Flattened structure with `type` as discriminator and all fields at root level

**Benefits**:
- Simpler MongoDB queries (no need to reference `data.sessionId`)
- Matches Claude Code's native JSONL format exactly
- Custom marshaling handles flattening/unflattening transparently

## Related Documentation

- **Record Model Implementation**: `shared/domain/record.go`, `record_user.go`, `record_assistant.go`, `record_file_history_snapshot.go`
- **Example JSONL Data**: `shared/domain/record_user.jsonl`, `record_assistant.jsonl`
- **MongoDB Schema**: `shared/domain/mongoschema/session_record.go`
- **Protobuf Definitions**: `idl/protobuf/aggregation/v1/aggregation.proto`
- **Project Architecture**: `CLAUDE.md` (system overview and build commands)
