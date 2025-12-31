# Implementation Plan: Migrate from SessionRecord to Record Model

## Overview

This plan migrates the codebase from the old `SessionRecord` model (defined in `session.go` and `message.go`) to the new `Record` model (defined in `record.go`, `record_user.go`, `record_assistant.go`, `record_file_history_snapshot.go`). The new `Record` model uses custom JSON marshaling/unmarshaling to produce flattened JSON format matching the JSONL structure. All usages of `SessionRecord`, `Message`, `Usage`, `SessionType`, and related types will be replaced. After migration, `session.go` and `message.go` will be deleted.

The migration involves:
1. Updating protobuf definitions to match the new Record model
2. Regenerating gRPC stubs
3. Updating parser port interface and implementation
4. Updating MongoDB adapter to handle the new Record model with flattened storage
5. Updating MongoDB schema constants
6. Updating service layers and handlers
7. Updating all tests
8. Deleting old domain files

## Package Changes

| Action | Problem | Package | Reason |
| :----- | :------ | :------ | :----- |
| None | N/A | N/A | No new packages required; all functionality can be implemented with existing dependencies |

---

## Step 1: Update Protobuf Definitions

**Files to Read**:
- `.agent/rules/idl/protobuf.md`: Protobuf naming conventions and code generation commands
- `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto`: Current aggregation proto definitions
- `/Users/jayce/team-attention/cops/idl/protobuf/dashboard/v1/dashboard.proto`: Current dashboard proto definitions
- `/Users/jayce/team-attention/cops/shared/domain/record.go`: New Record model structure
- `/Users/jayce/team-attention/cops/shared/domain/record_user.go`: UserRecord structure
- `/Users/jayce/team-attention/cops/shared/domain/record_assistant.go`: AssistantRecord structure

### `idl/protobuf/aggregation/v1/aggregation.proto`

**Description**:
Replace the existing `SessionRecord` and related messages with a new `Record` message that matches the flattened JSONL structure. The new proto will use `oneof` for type-specific data.

```protobuf
syntax = "proto3";

package aggregation.v1;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1;aggregationv1";

// RecordType represents the type discriminator for records.
enum RecordType {
  RECORD_TYPE_UNSPECIFIED = 0;
  RECORD_TYPE_USER = 1;
  RECORD_TYPE_ASSISTANT = 2;
  RECORD_TYPE_FILE_HISTORY_SNAPSHOT = 3;
}

// MessageMetadata contains common metadata fields for user and assistant records.
message MessageMetadata {
  // 1. Define parent_uuid as optional string (google.protobuf.StringValue or just string with empty check)
  string parent_uuid = 1;
  // 2. Define is_sidechain as bool
  bool is_sidechain = 2;
  // 3. Define user_type as string
  string user_type = 3;
  // 4. Define session_id as string
  string session_id = 4;
  // 5. Define version as string
  string version = 5;
  // 6. Define git_branch as string
  string git_branch = 6;
  // 7. Define uuid as string
  string uuid = 7;
  // 8. Define timestamp as google.protobuf.Timestamp
  google.protobuf.Timestamp timestamp = 8;
}

// UserMessage contains user message content.
message UserMessage {
  // 1. Define role as string
  string role = 1;
  // 2. Define content as string (for simplicity; complex content stored as JSON string)
  string content = 2;
}

// UserRecordThinkingMetadataTrigger contains trigger information.
message UserRecordThinkingMetadataTrigger {
  // 1. Define start as int32
  int32 start = 1;
  // 2. Define end as int32
  int32 end = 2;
  // 3. Define text as string
  string text = 3;
}

// UserRecordThinkingMetadata contains thinking metadata.
message UserRecordThinkingMetadata {
  // 1. Define level as string
  string level = 1;
  // 2. Define disabled as bool
  bool disabled = 2;
  // 3. Define triggers as repeated UserRecordThinkingMetadataTrigger
  repeated UserRecordThinkingMetadataTrigger triggers = 3;
}

// UserRecordTodo contains todo item.
message UserRecordTodo {
  // 1. Define content as string
  string content = 1;
  // 2. Define status as string
  string status = 2;
  // 3. Define active_form as string
  string active_form = 3;
}

// UserRecordData contains user-specific record data.
message UserRecordData {
  // 1. Embed MessageMetadata
  MessageMetadata metadata = 1;
  // 2. Define message as UserMessage
  UserMessage message = 2;
  // 3. Define is_meta as bool
  bool is_meta = 3;
  // 4. Define thinking_metadata as optional UserRecordThinkingMetadata
  UserRecordThinkingMetadata thinking_metadata = 4;
  // 5. Define todos as repeated UserRecordTodo
  repeated UserRecordTodo todos = 5;
}

// AssistantMessageContent contains assistant message content block.
message AssistantMessageContent {
  // 1. Define type as string (text, thinking, tool_use)
  string type = 1;
  // 2. Define text as optional string (for text type)
  string text = 2;
  // 3. Define thinking as optional string (for thinking type)
  string thinking = 3;
  // 4. Define tool_use_id as optional string (for tool_use type)
  string tool_use_id = 4;
  // 5. Define tool_use_name as optional string (for tool_use type)
  string tool_use_name = 5;
  // 6. Define tool_use_input_json as optional string (for tool_use type, JSON encoded)
  string tool_use_input_json = 6;
}

// AssistantMessageUsage contains token usage information.
message AssistantMessageUsage {
  // 1. Define input_tokens as int32
  int32 input_tokens = 1;
  // 2. Define output_tokens as int32
  int32 output_tokens = 2;
  // 3. Define cache_creation_input_tokens as int32
  int32 cache_creation_input_tokens = 3;
  // 4. Define cache_read_input_tokens as int32
  int32 cache_read_input_tokens = 4;
  // 5. Define service_tier as string
  string service_tier = 5;
}

// AssistantMessage contains assistant message.
message AssistantMessage {
  // 1. Define model as string
  string model = 1;
  // 2. Define id as string
  string id = 2;
  // 3. Define type as string
  string type = 3;
  // 4. Define role as string
  string role = 4;
  // 5. Define content as repeated AssistantMessageContent
  repeated AssistantMessageContent content = 5;
  // 6. Define stop_reason as optional string
  string stop_reason = 6;
  // 7. Define stop_sequence as optional int32
  int32 stop_sequence = 7;
  // 8. Define usage as AssistantMessageUsage
  AssistantMessageUsage usage = 8;
}

// AssistantRecordData contains assistant-specific record data.
message AssistantRecordData {
  // 1. Embed MessageMetadata
  MessageMetadata metadata = 1;
  // 2. Define request_id as string
  string request_id = 2;
  // 3. Define message as AssistantMessage
  AssistantMessage message = 3;
}

// FileHistorySnapshotTrackedBackup contains file backup information.
message FileHistorySnapshotTrackedBackup {
  // 1. Define backup_file_name as optional string
  string backup_file_name = 1;
  // 2. Define version as int32
  int32 version = 2;
  // 3. Define backup_time as google.protobuf.Timestamp
  google.protobuf.Timestamp backup_time = 3;
}

// FileHistorySnapshot contains snapshot data.
message FileHistorySnapshot {
  // 1. Define message_id as string
  string message_id = 1;
  // 2. Define tracked_file_backups as map<string, FileHistorySnapshotTrackedBackup>
  map<string, FileHistorySnapshotTrackedBackup> tracked_file_backups = 2;
}

// FileHistorySnapshotRecordData contains file-history-snapshot-specific record data.
message FileHistorySnapshotRecordData {
  // 1. Define message_id as string
  string message_id = 1;
  // 2. Define snapshot as FileHistorySnapshot
  FileHistorySnapshot snapshot = 2;
  // 3. Define is_snapshot_update as bool
  bool is_snapshot_update = 3;
}

// Record represents a single Claude Code JSONL entry with type-specific data.
message Record {
  // 1. Define type as RecordType enum
  RecordType type = 1;
  // 2. Define data as oneof with user_data, assistant_data, file_history_snapshot_data
  oneof data {
    UserRecordData user_data = 2;
    AssistantRecordData assistant_data = 3;
    FileHistorySnapshotRecordData file_history_snapshot_data = 4;
  }
}

// LogBatch contains raw JSONL lines for batch sending.
message LogBatch {
  // 1. Keep jsonl as repeated string (raw JSONL lines)
  repeated string jsonl = 1;
  // 2. Keep project_id as string
  string project_id = 2;
}

// SendLogsReq is the request for sending logs.
message SendLogsReq {
  // 1. Define batch as LogBatch
  LogBatch batch = 1;
}

// SendLogsRes is the response for sending logs.
message SendLogsRes {
  // 1. Define success as bool
  bool success = 1;
  // 2. Define error_message as string
  string error_message = 2;
  // 3. Define processed_count as int32
  int32 processed_count = 3;
}

// AggregationService handles log aggregation from daemons.
service AggregationService {
  // SendLogs sends a batch of raw JSONL lines to the API server.
  rpc SendLogs(SendLogsReq) returns (SendLogsRes);
}
```

### `idl/protobuf/dashboard/v1/dashboard.proto`

**Description**:
Update the `SessionDetail.records` field to use the new `Record` message instead of `SessionRecord`. Update all references.

```protobuf
// In the imports section:
// 1. Keep import "aggregation/v1/aggregation.proto"

// In SessionDetail message:
// 1. Change field 9 from:
//    repeated aggregation.v1.SessionRecord records = 9;
//    to:
//    repeated aggregation.v1.Record records = 9;
```

**Test Scenarios**: N/A (proto files are not unit tested directly)

---

## Step 2: Regenerate gRPC Stubs

**Files to Read**:
- `.agent/rules/idl/protobuf.md`: Code generation commands

### Command

```bash
cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate
```

**Description**:
Run `buf generate` to regenerate all Go types and ConnectRPC handlers from the updated proto files.

**Test Scenarios**: N/A (code generation)

---

## Step 3: Update MongoDB Schema Constants

**Files to Read**:
- `/Users/jayce/team-attention/cops/shared/domain/mongoschema/session_record.go`: Current schema constants
- `/Users/jayce/team-attention/cops/shared/domain/record.go`: New Record model field names

### `shared/domain/mongoschema/session_record.go`

**Description**:
Update the schema constants to match the new Record model field names. The collection name remains `sessionRecords` for continuity.

```go
package mongoschema

const (
	// RecordCollectionName is the MongoDB collection name for records.
	// Name unchanged for continuity.
	RecordCollectionName = "sessionRecords"
)

// Record-level fields (root level of document)
// Naming pattern: Record<FieldName>Field
const (
	RecordIDField              = "_id"
	RecordTypeField            = "type"
	RecordParentUUIDField      = "parentUuid"
	RecordIsSidechainField     = "isSidechain"
	RecordUserTypeField        = "userType"
	RecordSessionIDField       = "sessionId"
	RecordVersionField         = "version"
	RecordGitBranchField       = "gitBranch"
	RecordUUIDField            = "uuid"
	RecordTimestampField       = "timestamp"
	RecordProjectIDField       = "projectId"
	RecordIsMetaField          = "isMeta"
	RecordMessageField         = "message"
	RecordThinkingMetadataField = "thinkingMetadata"
	RecordTodosField           = "todos"
	RecordRequestIDField       = "requestId"
	RecordMessageIDField       = "messageId"       // For FileHistorySnapshotRecord
	RecordSnapshotField        = "snapshot"
	RecordIsSnapshotUpdateField = "isSnapshotUpdate"
)

// Message-level fields (inside Record.Message object)
// Naming pattern: Message<FieldName>Field
const (
	MessageModelField      = "model"
	MessageIDField         = "id"
	MessageTypeField       = "type"
	MessageRoleField       = "role"
	MessageContentField    = "content"
	MessageStopReasonField = "stopReason"
	MessageStopSequenceField = "stopSequence"
	MessageUsageField      = "usage"
)

// Usage-level fields (inside Message.Usage object)
// Naming pattern: Usage<FieldName>Field
const (
	UsageInputTokensField         = "inputTokens"
	UsageOutputTokensField        = "outputTokens"
	UsageCacheCreationTokensField = "cacheCreationInputTokens"
	UsageCacheReadTokensField     = "cacheReadInputTokens"
	UsageServiceTierField         = "serviceTier"
)

// ThinkingMetadata-level fields (inside Record.ThinkingMetadata object)
// Naming pattern: ThinkingMetadata<FieldName>Field
const (
	ThinkingMetadataLevelField    = "level"
	ThinkingMetadataDisabledField = "disabled"
	ThinkingMetadataTriggersField = "triggers"
)

// Snapshot-level fields (inside Record.Snapshot object)
// Naming pattern: Snapshot<FieldName>Field
const (
	SnapshotMessageIDField          = "messageId"
	SnapshotTrackedFileBackupsField = "trackedFileBackups"
)

// Legacy aliases for backward compatibility during migration (to be removed after full migration)
const (
	SessionRecordCollectionName = RecordCollectionName
)
```

**Test Scenarios**: N/A (constants file)

---

## Step 4: Update Parser Port Interface

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/parser/parser_port.go`: Current interface
- `.agent/rules/go/go-outbound.md`: Outbound adapter guidelines

### `cli/internal/service/tracking/outbound/parser/parser_port.go`

**Description**:
Update the `ParserPort` interface to return `[]*domain.Record` instead of `[]*domain.SessionRecord`.

```go
package parser

import "github.com/team-attention/cops/shared/domain"

// ParserPort defines the interface for parsing session files.
type ParserPort interface {
	// ParseSessionFiles parses all JSONL files in a project's Claude directory.
	// Returns a slice of Record pointers containing all parsed records.
	// Implementation outline:
	// 1. Accept claudeProjectDir path string.
	// 2. Return []*domain.Record slice and error.
	ParseSessionFiles(claudeProjectDir string) ([]*domain.Record, error)
}
```

**Test Scenarios**: N/A (interface definition)

---

## Step 5: Update JSONL Parser Implementation

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/parser/jsonl/jsonl_parser.go`: Current implementation
- `/Users/jayce/team-attention/cops/shared/domain/record.go`: Record unmarshaling behavior
- `.agent/rules/go/go-outbound.md`: Outbound adapter guidelines

### `cli/internal/service/tracking/outbound/parser/jsonl/jsonl_parser.go`

**Description**:
Update the parser to use `domain.Record` and remove type filtering logic. Parse all record types including `file-history-snapshot`.

```go
package jsonl

import (
	"bufio"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/team-attention/cops/cli/internal/service/tracking/outbound/parser"
	"github.com/team-attention/cops/shared/domain"
)

// JSONLParser implements ParserPort for Claude Code JSONL files.
type JSONLParser struct {
	logger *slog.Logger
}

// NewJSONLParser creates a new JSONL parser.
func NewJSONLParser(l *slog.Logger) *JSONLParser {
	// Implementation outline:
	// 1. Create JSONLParser instance with logger.
	// 2. Bind logger with name "tracking.parser.jsonl".
	// 3. Return pointer to JSONLParser.
}

// ParseSessionFiles parses all JSONL files in a project's Claude directory.
func (p *JSONLParser) ParseSessionFiles(claudeProjectDir string) ([]*domain.Record, error) {
	// Implementation outline:
	// 1. Initialize empty records slice.
	// 2. Check if directory exists using os.Stat.
	//    a. If not exists, return empty slice (no error).
	// 3. Glob for *.jsonl files in directory.
	// 4. Log debug message with file count.
	// 5. For each file:
	//    a. Call parseFile to get file records.
	//    b. If error, log warning and continue.
	//    c. Append file records to main slice.
	// 6. Return records slice.
}

func (p *JSONLParser) parseFile(filePath string) ([]*domain.Record, error) {
	// Implementation outline:
	// 1. Open file.
	// 2. Defer file close.
	// 3. Initialize empty records slice.
	// 4. Create scanner with large buffer (10MB max).
	// 5. For each line:
	//    a. Trim whitespace.
	//    b. Skip empty lines.
	//    c. Unmarshal JSON into domain.Record using sonic.Unmarshal.
	//    d. If error, log debug and continue (skip malformed).
	//    e. Append pointer to record to slice.
	//    NOTE: Do NOT filter by type - parse ALL record types.
	// 6. Check scanner error.
	// 7. Return records slice.
}

// Compile-time interface verification
var _ parser.ParserPort = (*JSONLParser)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid user record | `{"type":"user","sessionId":"s1",...}` | Record with Type=RecordTypeUser | Happy path |
| Valid assistant record | `{"type":"assistant","requestId":"r1",...}` | Record with Type=RecordTypeMessage | Happy path |
| Valid file-history-snapshot | `{"type":"file-history-snapshot",...}` | Record with Type=RecordTypeFileHistorySnapshot | Happy path (new) |
| Empty line | `""` | Skipped, not in result | Empty line branch |
| Malformed JSON | `{not valid}` | Skipped with debug log | Error handling |
| Empty directory | Non-existent path | Empty slice, no error | Directory check |
| Mixed valid/invalid | Multiple lines | Only valid records in result | Mixed content |

---

## Step 6: Update Aggregation Repository Port

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/port.go`: Current port
- `.agent/rules/go/go-outbound.md`: Port definition guidelines

### `api/internal/service/aggregation/outbound/repository/port.go`

**Description**:
Update `LogBatch` to use `[]domain.Record` instead of `[]domain.SessionRecord`.

```go
package repository

import (
	"context"

	shareddomain "github.com/team-attention/cops/shared/domain"
)

// LogBatch represents a batch of records from a daemon.
type LogBatch struct {
	// Records contains the parsed Record instances (all types).
	Records []shareddomain.Record
	// ProjectID is the project identifier for this batch.
	ProjectID string
}

// SessionRecordRepositoryPort defines the interface for record persistence.
// NOTE: Name kept for minimal interface change; consider renaming to RecordRepositoryPort in future.
type SessionRecordRepositoryPort interface {
	// SaveBatch saves a batch of records to storage.
	SaveBatch(ctx context.Context, batch *LogBatch) error
}
```

**Test Scenarios**: N/A (interface definition)

---

## Step 7: Update MongoDB Aggregation Adapter

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go`: Current adapter
- `/Users/jayce/team-attention/cops/shared/domain/record.go`: Record model
- `/Users/jayce/team-attention/cops/shared/domain/record_user.go`: UserRecord
- `/Users/jayce/team-attention/cops/shared/domain/record_assistant.go`: AssistantRecord
- `/Users/jayce/team-attention/cops/shared/domain/record_file_history_snapshot.go`: FileHistorySnapshotRecord
- `.agent/rules/go/go-outbound.md`: Adapter guidelines

### `api/internal/service/aggregation/outbound/repository/mongodb/adapter.go`

**Description**:
Update the adapter to handle the new `Record` model. Use the Record's `MarshalJSON` to produce flattened documents for MongoDB storage.

```go
package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bytedance/sonic"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/aggregation/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoSessionRecordRepository implements SessionRecordRepositoryPort using MongoDB.
type MongoSessionRecordRepository struct {
	logger     *slog.Logger
	collection *mongo.Collection
}

// NewMongoSessionRecordRepository creates a new MongoDB record repository adapter.
func NewMongoSessionRecordRepository(l *slog.Logger, db *mongo.Database) *MongoSessionRecordRepository {
	// Implementation outline:
	// 1. Return MongoSessionRecordRepository with:
	//    a. Logger bound with name "aggregation.repository.mongodb".
	//    b. Collection from db using mongoschema.RecordCollectionName.
}

// SaveBatch saves a batch of records to MongoDB.
func (r *MongoSessionRecordRepository) SaveBatch(ctx context.Context, batch *repository.LogBatch) error {
	// Implementation outline:
	// 1. If batch.Records is empty, return nil.
	// 2. Convert projectID to ObjectID.
	//    a. If invalid, return error.
	// 3. Create docs slice of interface{}.
	// 4. For each record in batch.Records:
	//    a. Call toDocument(record, projectObjID).
	//    b. Append to docs slice.
	// 5. Call collection.InsertMany with docs.
	// 6. If error, log error and return wrapped error.
	// 7. Log debug with inserted count.
	// 8. Return nil.
}

func toDocument(record shareddomain.Record, projectObjID bson.ObjectID) bson.M {
	// Implementation outline:
	// 1. Marshal record to JSON using record.MarshalJSON() (produces flat JSON).
	// 2. Unmarshal JSON into bson.M.
	// 3. Add projectId field with projectObjID.
	// 4. Return bson.M document.
	//
	// Note: Record.MarshalJSON automatically flattens type-specific fields
	// to top level, matching JSONL format.
}

// Compile-time interface verification.
var _ repository.SessionRecordRepositoryPort = (*MongoSessionRecordRepository)(nil)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Empty batch | Empty Records slice | Return nil immediately | Early return |
| Valid user records | LogBatch with UserRecords | InsertMany called with flat docs | Happy path |
| Valid assistant records | LogBatch with AssistantRecords | InsertMany called with flat docs | Happy path |
| Valid mixed records | LogBatch with all types | InsertMany called with all flat docs | Happy path |
| Invalid project ID | Invalid hex string | Return error | Error handling |
| MongoDB insert error | DB failure | Return wrapped error | Error handling |

---

## Step 8: Update Dashboard Repository Port

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/dashboard_repo_port.go`: Current port

### `api/internal/service/dashboard/outbound/repository/dashboard_repo_port.go`

**Description**:
Update `SessionDetail.Records` to use `[]domain.Record` instead of `[]domain.SessionRecord`.

```go
package repository

// ... (keep existing imports and types)

// SessionDetail contains full session information with records.
// Embeds SessionBase for common identification.
type SessionDetail struct {
	SessionBase
	CWD     string
	Version string
	Usage   TokenUsageSummary
	// Records changed from []shareddomain.SessionRecord to []shareddomain.Record
	Records []shareddomain.Record
}

// ... (keep rest of file unchanged)
```

**Test Scenarios**: N/A (struct definition)

---

## Step 9: Update Dashboard MongoDB Repository

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`: Current implementation
- `/Users/jayce/team-attention/cops/shared/domain/record.go`: Record model

### `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`

**Description**:
Update the `GetSession` method to reconstruct `domain.Record` from MongoDB documents instead of `domain.SessionRecord`.

```go
// In GetSession method, update the record reconstruction logic:

// GetSession retrieves detailed session information with all records.
func (r *MongoDashboardRepository) GetSession(ctx context.Context, sessionID string) (*repository.SessionDetail, error) {
	// Implementation outline:
	// 1. Find all records for session by sessionId field.
	// 2. Sort by timestamp ascending.
	// 3. Initialize detail as nil and records slice.
	// 4. For each cursor document:
	//    a. Initialize detail from first record if nil.
	//    b. Reconstruct domain.Record from document:
	//       i. Read "type" field to determine RecordType.
	//       ii. Marshal document back to JSON.
	//       iii. Unmarshal JSON into domain.Record (uses custom UnmarshalJSON).
	//    c. Append record to records slice.
	// 5. If detail is nil, return "session not found" error.
	// 6. Set detail.Records = records.
	// 7. Calculate aggregated usage and timestamps from records:
	//    a. For user records: no token usage.
	//    b. For assistant records: extract usage from AssistantRecord.Message.Usage.
	// 8. Return detail.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Session with user records | sessionId with user records | SessionDetail with UserRecord data | User record path |
| Session with assistant records | sessionId with assistant records | SessionDetail with AssistantRecord data and usage | Assistant record path |
| Session with mixed records | sessionId with all types | SessionDetail with all record types | Mixed path |
| Session not found | Non-existent sessionId | Error "session not found" | Not found error |
| MongoDB error | DB failure | Wrapped error | Error handling |

---

## Step 10: Update Dashboard gRPC Handler Converter

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go`: Current converter
- `/Users/jayce/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregation.pb.go`: Generated proto types (after Step 2)

### `api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go`

**Description**:
Update the converter to transform `domain.Record` to the new `aggregationv1.Record` proto message.

```go
package connectrpc

import (
	// ... existing imports
	shareddomain "github.com/team-attention/cops/shared/domain"
	aggregationv1 "github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1"
)

// toProtoSessionDetail converts repository session detail to protobuf.
func toProtoSessionDetail(s *repository.SessionDetail) *dashboardv1.SessionDetail {
	// Implementation outline:
	// 1. Convert Records slice using toProtoRecords.
	// 2. Build SessionDetail proto with all fields.
	// 3. Return proto.
}

// toProtoRecords converts domain Records to protobuf Records.
func toProtoRecords(records []shareddomain.Record) []*aggregationv1.Record {
	// Implementation outline:
	// 1. Create result slice with capacity.
	// 2. For each record:
	//    a. Call toProtoRecord.
	//    b. Append to result if not nil.
	// 3. Return result.
}

// toProtoRecord converts a single domain Record to protobuf.
func toProtoRecord(r shareddomain.Record) *aggregationv1.Record {
	// Implementation outline:
	// 1. Create aggregationv1.Record.
	// 2. Set Type based on r.Type:
	//    a. RecordTypeUser -> RECORD_TYPE_USER
	//    b. RecordTypeMessage -> RECORD_TYPE_ASSISTANT
	//    c. RecordTypeFileHistorySnapshot -> RECORD_TYPE_FILE_HISTORY_SNAPSHOT
	// 3. Type switch on r.Data:
	//    a. case *shareddomain.UserRecord:
	//       Set user_data field with converted UserRecordData.
	//    b. case *shareddomain.AssistantRecord:
	//       Set assistant_data field with converted AssistantRecordData.
	//    c. case *shareddomain.FileHistorySnapshotRecord:
	//       Set file_history_snapshot_data field with converted data.
	// 4. Return proto record.
}

// toProtoUserRecordData converts UserRecord to proto UserRecordData.
func toProtoUserRecordData(u *shareddomain.UserRecord) *aggregationv1.UserRecordData {
	// Implementation outline:
	// 1. Create UserRecordData proto.
	// 2. Set metadata from MessageMetadata fields.
	// 3. Set message from UserMessage.
	// 4. Set is_meta from IsMeta.
	// 5. Set thinking_metadata if not nil.
	// 6. Set todos if not empty.
	// 7. Return proto.
}

// toProtoAssistantRecordData converts AssistantRecord to proto AssistantRecordData.
func toProtoAssistantRecordData(a *shareddomain.AssistantRecord) *aggregationv1.AssistantRecordData {
	// Implementation outline:
	// 1. Create AssistantRecordData proto.
	// 2. Set metadata from MessageMetadata fields.
	// 3. Set request_id from RequestID.
	// 4. Set message from AssistantMessage.
	// 5. Return proto.
}

// toProtoFileHistorySnapshotRecordData converts FileHistorySnapshotRecord to proto.
func toProtoFileHistorySnapshotRecordData(f *shareddomain.FileHistorySnapshotRecord) *aggregationv1.FileHistorySnapshotRecordData {
	// Implementation outline:
	// 1. Create FileHistorySnapshotRecordData proto.
	// 2. Set message_id from MessageID.
	// 3. Set snapshot with converted FileHistorySnapshot.
	// 4. Set is_snapshot_update from IsSnapshotUpdate.
	// 5. Return proto.
}

// Remove old conversion functions:
// - toProtoSessionRecord (delete)
// - convertSessionType (delete)
// - convertMessage (delete)
// - convertContentBlocks (delete)
// - convertContentBlock (delete)
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| UserRecord conversion | domain.Record with UserRecord data | aggregationv1.Record with user_data | User path |
| AssistantRecord conversion | domain.Record with AssistantRecord data | aggregationv1.Record with assistant_data | Assistant path |
| FileHistorySnapshotRecord conversion | domain.Record with FileHistorySnapshotRecord data | aggregationv1.Record with file_history_snapshot_data | File history path |
| nil Data | domain.Record with nil Data | Proto with type only | Nil data path |

---

## Step 11: Update Aggregation gRPC Handler

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go`: Current handler

### `api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go`

**Description**:
Update `parseJSONLLines` to parse into `domain.Record` instead of `domain.SessionRecord`.

```go
package connectrpc

// parseJSONLLines parses raw JSONL lines into Record domain objects.
// Returns the parsed batch and any parse errors encountered.
func (h *AggregationGRPCHandler) parseJSONLLines(lines []string, projectID string) (*repository.LogBatch, []error) {
	// Implementation outline:
	// 1. Initialize records slice as []shareddomain.Record.
	// 2. Initialize parseErrors slice.
	// 3. For each line:
	//    a. Skip empty lines.
	//    b. Unmarshal into domain.Record using sonic.Unmarshal.
	//       Note: domain.Record.UnmarshalJSON handles type dispatch.
	//    c. If error, append to parseErrors and continue.
	//    d. Append record to records slice.
	// 4. Return LogBatch with records and parseErrors.
}
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| :------- | :---- | :-------------- | :------------- |
| Valid user JSONL | `{"type":"user",...}` | Record with UserRecord data | User parsing |
| Valid assistant JSONL | `{"type":"assistant",...}` | Record with AssistantRecord data | Assistant parsing |
| Valid file-history-snapshot | `{"type":"file-history-snapshot",...}` | Record with FileHistorySnapshotRecord data | File history parsing |
| Empty line | `""` | Skipped | Empty line |
| Invalid JSON | `{invalid}` | Error in parseErrors | Parse error |
| Mixed valid/invalid | Multiple lines | Valid records + errors | Mixed |

---

## Step 12: Update Test Files

**Files to Read**:
- `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler_test.go`: Aggregation handler tests
- `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service_test.go`: Log watcher tests
- `/Users/jayce/team-attention/cops/shared/domain/message_suite_test.go`: Message tests

### `api/internal/service/aggregation/inbound/grpc/connectrpc/handler_test.go`

**Description**:
Update tests to use `domain.Record` instead of `domain.SessionRecord`.

```go
package connectrpc_test

import (
	"github.com/bytedance/sonic"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	shareddomain "github.com/team-attention/cops/shared/domain"
)

var _ = Describe("JSONL Parsing", func() {
	Describe("parsing valid JSONL lines", func() {
		Context("when all lines are valid JSON", func() {
			It("parses all lines successfully", func() {
				// Implementation outline:
				// 1. Define test JSONL lines with type field.
				// 2. Parse each line into domain.Record.
				// 3. Assert correct number of records.
				// 4. Assert record types match expected.
			})
		})
	})

	Describe("parsing record types", func() {
		Context("when parsing user record", func() {
			It("returns Record with UserRecord data", func() {
				// Implementation outline:
				// 1. Define user JSONL line.
				// 2. Parse into domain.Record.
				// 3. Assert Type == RecordTypeUser.
				// 4. Assert Data is *UserRecord.
			})
		})

		Context("when parsing assistant record", func() {
			It("returns Record with AssistantRecord data", func() {
				// Implementation outline:
				// 1. Define assistant JSONL line.
				// 2. Parse into domain.Record.
				// 3. Assert Type == RecordTypeMessage.
				// 4. Assert Data is *AssistantRecord.
			})
		})

		Context("when parsing file-history-snapshot record", func() {
			It("returns Record with FileHistorySnapshotRecord data", func() {
				// Implementation outline:
				// 1. Define file-history-snapshot JSONL line.
				// 2. Parse into domain.Record.
				// 3. Assert Type == RecordTypeFileHistorySnapshot.
				// 4. Assert Data is *FileHistorySnapshotRecord.
			})
		})
	})

	// ... keep other test cases updated similarly
})
```

### `daemon/internal/service/logwatcher/log_service_test.go`

**Description**:
Update integration tests to use `domain.Record` instead of `domain.SessionRecord`.

```go
package logwatcher_test

// Implementation outline:
// 1. Replace all shareddomain.SessionRecord with shareddomain.Record.
// 2. Replace record.Type comparisons:
//    a. SessionTypeUser -> RecordTypeUser
//    b. SessionTypeAssistant -> RecordTypeMessage
// 3. Update record field access to use type assertions on record.Data:
//    a. For user records: record.Data.(*shareddomain.UserRecord)
//    b. For assistant records: record.Data.(*shareddomain.AssistantRecord)
// 4. Update message content access via typed data fields.
```

### `shared/domain/message_suite_test.go`

**Description**:
This file tests the old Message/MessageContent types. After migration, these types will be deleted, so this test file should be deleted entirely.

**Action**: Delete file `/Users/jayce/team-attention/cops/shared/domain/message_suite_test.go`

**Test Scenarios**: N/A (test file updates)

---

## Step 13: Delete Old Domain Files

**Files to Read**: N/A

### Files to Delete

1. `/Users/jayce/team-attention/cops/shared/domain/session.go`
2. `/Users/jayce/team-attention/cops/shared/domain/message.go`
3. `/Users/jayce/team-attention/cops/shared/domain/message_suite_test.go`

**Description**:
After all references to `SessionRecord`, `Message`, `Usage`, `SessionType`, and related types have been replaced, delete the old domain files.

```bash
rm /Users/jayce/team-attention/cops/shared/domain/session.go
rm /Users/jayce/team-attention/cops/shared/domain/message.go
rm /Users/jayce/team-attention/cops/shared/domain/message_suite_test.go
```

**Test Scenarios**: N/A (file deletion)

---

## Step 14: Verify Build and Tests

**Files to Read**: N/A

### Commands

```bash
# Build all modules from root
cd /Users/jayce/team-attention/cops && go build ./cli/... ./api/... ./daemon/... ./shared/...

# Run all tests
cd /Users/jayce/team-attention/cops && go test ./...

# Verify no references to old types remain
grep -r "SessionRecord" --include="*.go" /Users/jayce/team-attention/cops
grep -r "SessionType" --include="*.go" /Users/jayce/team-attention/cops
# Both should return no results (except in generated proto files if any)
```

**Test Scenarios**: N/A (verification step)

---

## Summary of Files to Modify

| File | Action |
| :--- | :----- |
| `idl/protobuf/aggregation/v1/aggregation.proto` | Update with new Record message |
| `idl/protobuf/dashboard/v1/dashboard.proto` | Update SessionDetail.records field |
| `shared/domain/mongoschema/session_record.go` | Update schema constants |
| `cli/internal/service/tracking/outbound/parser/parser_port.go` | Update interface |
| `cli/internal/service/tracking/outbound/parser/jsonl/jsonl_parser.go` | Update implementation |
| `api/internal/service/aggregation/outbound/repository/port.go` | Update LogBatch |
| `api/internal/service/aggregation/outbound/repository/mongodb/adapter.go` | Update adapter |
| `api/internal/service/dashboard/outbound/repository/dashboard_repo_port.go` | Update SessionDetail |
| `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go` | Update GetSession |
| `api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go` | Update converter |
| `api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go` | Update parseJSONLLines |
| `api/internal/service/aggregation/inbound/grpc/connectrpc/handler_test.go` | Update tests |
| `daemon/internal/service/logwatcher/log_service_test.go` | Update tests |
| `shared/domain/session.go` | Delete |
| `shared/domain/message.go` | Delete |
| `shared/domain/message_suite_test.go` | Delete |

## Execution Order

1. Step 1: Update protobuf definitions
2. Step 2: Regenerate gRPC stubs
3. Step 3: Update MongoDB schema constants
4. Step 4: Update parser port interface
5. Step 5: Update JSONL parser implementation
6. Step 6: Update aggregation repository port
7. Step 7: Update MongoDB aggregation adapter
8. Step 8: Update dashboard repository port
9. Step 9: Update dashboard MongoDB repository
10. Step 10: Update dashboard gRPC handler converter
11. Step 11: Update aggregation gRPC handler
12. Step 12: Update test files
13. Step 13: Delete old domain files
14. Step 14: Verify build and tests
