# Research Report

## Mode
General Research

## Request Summary

Move JSONL parsing logic from Daemon to API Server. Currently, Daemon parses JSONL logs into structured `SessionRecord` objects before sending to API. After this change, Daemon will send raw JSONL text lines to API, and API will handle all parsing and validation logic.

**Current Flow:**
```
JSONL Log File -> Daemon (parse with sonic) -> SessionRecord objects -> gRPC -> API (convert & save)
                  ^-- sonic.Unmarshal                                         ^-- protobuf to domain
```

**Target Flow:**
```
JSONL Log File -> Daemon (read raw) -> Raw strings -> gRPC -> API (parse + save)
                                                              ^-- sonic.Unmarshal + convert + save
```

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File | Reason |
| ---- | ------ |
| `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go` | Core file containing `HandleFileChange()` with current JSONL parsing logic (sonic.Unmarshal at line 134), `AddRecordsForClaudeDir()`, `Flush()` |
| `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go` | Contains `SendLogs()` and `convertRecords()` which convert domain to protobuf |
| `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/watch.go` | Contains `LogBatch` struct definition in daemon domain |
| `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go` | API gRPC handler with `SendLogs()` and `convertToDomain()` |
| `/Users/jayce/team-attention/cops/api/internal/service/aggregation/aggregation_service.go` | API service layer calling repository |
| `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/port.go` | Repository port interface and `LogBatch` struct for API |
| `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go` | MongoDB adapter with `SaveBatch()` and `toDocument()` |
| `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto` | Protobuf schema to be modified |
| `/Users/jayce/team-attention/cops/shared/domain/session.go` | Shared `SessionRecord` struct definition |
| `/Users/jayce/team-attention/cops/shared/domain/message.go` | Shared `Message` and `MessageContent` with custom UnmarshalJSON |
| `/Users/jayce/team-attention/cops/shared/domain/content_block.go` | Content block types (text, tool_use, tool_result, thinking) |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md` | Service layer patterns |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound-grpc-connectrpc.md` | ConnectRPC handler patterns |
| `/Users/jayce/team-attention/cops/.agent/rules/idl/protobuf.md` | Protobuf conventions |

## Current Implementation Analysis

### 1. Daemon: JSONL Parsing Flow

**File: `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go`**

```
HandleFileChange(path, fromOffset)
    |
    v
os.Open(path) --> file.Seek(fromOffset) --> bufio.Scanner
    |
    v
for scanner.Scan() {
    line := scanner.Text()
    |
    v
    sonic.Unmarshal(line) --> shareddomain.SessionRecord  <-- PARSING HAPPENS HERE
    |
    v
    records = append(records, record)
}
    |
    v
return records, newOffset, nil
```

**Key Code Location (lines 125-143):**
```go
for scanner.Scan() {
    line := scanner.Text()
    newOffset += int64(len(line)) + 1 // +1 for newline

    if line == "" {
        continue
    }

    var record shareddomain.SessionRecord
    if err := sonic.Unmarshal([]byte(line), &record); err != nil {
        s.logger.Debug("failed to parse JSONL line",
            slog.String("path", path),
            slog.Any("error", err),
        )
        continue
    }

    records = append(records, record)
}
```

### 2. Daemon: Buffering Mechanism

**Current Buffer Structure:**
```go
// log_service.go line 29
bufferByProject map[shareddomain.ID][]shareddomain.SessionRecord
```

**Buffer Flow:**
```
AddRecordsForClaudeDir(claudeDir, records)
    |
    v
projectID := s.claudeDirToProject[claudeDir]
s.bufferByProject[projectID] = append(..., records...)
```

**Flush Flow (lines 163-215):**
```
Flush(ctx)
    |
    v
for projectID, records := range bufferedRecords {
    batch := domain.LogBatch{
        Records:   records,
        ProjectID: projectID,
    }
    |
    v
    s.apiClient.SendLogs(ctx, batch)
}
```

### 3. Daemon: API Client (Outbound)

**File: `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`**

```go
func (c *APIClient) SendLogs(ctx context.Context, batch domain.LogBatch) error {
    req := &aggregationv1.SendLogsReq{
        Batch: &aggregationv1.LogBatch{
            Records:   convertRecords(batch.Records),  // <-- Domain to Protobuf
            ProjectId: batch.ProjectID.String(),
        },
    }
    resp, err := c.client.SendLogs(ctx, connect.NewRequest(req))
    ...
}
```

**Conversion Functions (lines 61-137):**
- `convertRecords()` - converts `[]shareddomain.SessionRecord` to `[]*aggregationv1.SessionRecord`
- `convertSessionRecord()` - maps domain fields to protobuf fields
- `convertSessionType()` - maps domain enum to protobuf enum
- `convertMessage()` - maps domain Message to protobuf Message

### 4. API: gRPC Handler (Inbound)

**File: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go`**

```go
func (h *AggregationGRPCHandler) SendLogs(
    ctx context.Context,
    req *connect.Request[aggregationv1.SendLogsReq],
) (*connect.Response[aggregationv1.SendLogsRes], error) {
    pbBatch := req.Msg.GetBatch()

    batch := convertToDomain(pbBatch)  // <-- Protobuf to Domain
    result := h.svc.CollectLogs(ctx, batch)

    res := &aggregationv1.SendLogsRes{
        Success:        result.Success,
        ErrorMessage:   result.ErrorMessage,
        ProcessedCount: result.ProcessedCount,
    }
    return connect.NewResponse(res), nil
}
```

**Conversion Functions (lines 61-136):**
- `convertToDomain()` - converts protobuf LogBatch to repository.LogBatch
- `convertSessionType()` - maps protobuf enum to domain enum
- `convertMessage()` - maps protobuf Message to domain Message

### 5. API: Service Layer

**File: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/aggregation_service.go`**

```go
func (s *Service) CollectLogs(ctx context.Context, batch *repository.LogBatch) *CollectLogsResult {
    if len(batch.Records) == 0 {
        return &CollectLogsResult{Success: true, ProcessedCount: 0}
    }

    if err := s.repo.SaveBatch(ctx, batch); err != nil {
        return &CollectLogsResult{Success: false, ErrorMessage: err.Error()}
    }

    return &CollectLogsResult{Success: true, ProcessedCount: int32(len(batch.Records))}
}
```

### 6. API: Repository (Outbound)

**File: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go`**

```go
func (r *MongoSessionRecordRepository) SaveBatch(ctx context.Context, batch *repository.LogBatch) error {
    projectObjID, err := bson.ObjectIDFromHex(batch.ProjectID)

    docs := make([]interface{}, len(batch.Records))
    for i, record := range batch.Records {
        docs[i] = toDocument(record, projectObjID)
    }

    result, err := r.collection.InsertMany(ctx, docs)
    ...
}
```

### 7. Protobuf Schema

**File: `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto`**

```protobuf
message LogBatch {
  repeated SessionRecord records = 1;  // <-- TO BE CHANGED to repeated string jsonl
  string project_id = 2;
}

message SessionRecord {  // <-- TO BE REMOVED
  string uuid = 1;
  string parent_uuid = 2;
  string session_id = 3;
  SessionType type = 4;
  google.protobuf.Timestamp timestamp = 5;
  string cwd = 6;
  string git_branch = 7;
  string version = 8;
  string user_type = 9;
  bool is_sidechain = 10;
  bool is_meta = 11;
  string slug = 12;
  string request_id = 13;
  Message message = 14;
}
```

## Data Flow Diagram

```
                    CURRENT ARCHITECTURE
                    ===================

Daemon                                          API
======                                          ===

[JSONL File]
     |
     v
LogFsnotifyHandler.handleFileEvent()
     |
     v
log_service.HandleFileChange()
     |
     +----> bufio.Scanner.Scan()
     |           |
     |           v
     |      sonic.Unmarshal(line)  <-- JSONL PARSING
     |           |
     |           v
     |      []shareddomain.SessionRecord
     |
     v
log_service.AddRecordsForClaudeDir()
     |
     v
bufferByProject[projectID] = append(records)
     |
     | (every 5 seconds via flushTicker)
     v
log_service.Flush()
     |
     v
domain.LogBatch{Records, ProjectID}
     |
     v
api_client.SendLogs()
     |
     v
convertRecords() --> convertSessionRecord()  <-- Domain to Proto
     |
     v
aggregationv1.SendLogsReq{Batch}
     |
     | (gRPC/ConnectRPC)
     v
                                        handler.SendLogs()
                                             |
                                             v
                                        convertToDomain()  <-- Proto to Domain
                                             |
                                             v
                                        repository.LogBatch{Records, ProjectID}
                                             |
                                             v
                                        aggregation_service.CollectLogs()
                                             |
                                             v
                                        mongodb.SaveBatch()
                                             |
                                             v
                                        toDocument() --> InsertMany()
```

```
                    TARGET ARCHITECTURE
                    ==================

Daemon                                          API
======                                          ===

[JSONL File]
     |
     v
LogFsnotifyHandler.handleFileEvent()
     |
     v
log_service.HandleFileChange()
     |
     +----> bufio.Scanner.Scan()
     |           |
     |           v
     |      line := scanner.Text()  <-- RAW STRING ONLY
     |           |
     |           v
     |      []string (raw JSONL lines)
     |
     v
log_service.AddLinesForClaudeDir()  <-- RENAMED
     |
     v
bufferByProject[projectID] = append(lines)  <-- []string not []SessionRecord
     |
     | (every 5 seconds via flushTicker)
     v
log_service.Flush()
     |
     v
domain.LogBatch{Lines, ProjectID}  <-- Lines not Records
     |
     v
api_client.SendLogs()
     |
     v
(no conversion needed - just strings)
     |
     v
aggregationv1.SendLogsReq{Batch}
     |
     | (gRPC/ConnectRPC)
     v
                                        handler.SendLogs()
                                             |
                                             v
                                        parseJSONLLines()  <-- NEW: JSONL PARSING
                                             |
                                             v
                                        []shareddomain.SessionRecord
                                             |
                                             v
                                        repository.LogBatch{Records, ProjectID}
                                             |
                                             v
                                        aggregation_service.CollectLogs()
                                             |
                                             v
                                        mongodb.SaveBatch()
                                             |
                                             v
                                        toDocument() --> InsertMany()
```

## Package Candidates

### Problem 1: JSON Parsing

| Package | Context7 ID | Why Better Than Alternatives |
| ------- | ----------- | ---------------------------- |
| sonic | `/bytedance/sonic` | Already used in project, high performance, drop-in replacement for encoding/json |

**Note**: No new package needed - `github.com/bytedance/sonic` is already a dependency in both daemon and API.

### Problem 2: gRPC/Protobuf

| Package | Context7 ID | Why Better Than Alternatives |
| ------- | ----------- | ---------------------------- |
| connect-go | `/connectrpc/connect-go` | Already used in project, handles proto generation |

**Note**: No new package needed - ConnectRPC is already configured with buf.

## Technical Constraints

1. **Breaking Change**: This is a breaking change - old Daemon will not work with new API. Requires synchronized deployment.

2. **Buffer Type Change**: Daemon buffer changes from `map[ID][]SessionRecord` to `map[ID][]string`. This affects:
   - `log_service.go` buffer definition
   - `log_service.go` AddRecordsForClaudeDir (rename to AddLinesForClaudeDir)
   - `log_service.go` Flush method
   - `domain/watch.go` LogBatch struct

3. **Offset Tracking**: File offset calculation must remain in Daemon:
   - Offset = bytes read from file
   - Offset calculation: `newOffset += int64(len(line)) + 1` stays in `HandleFileChange()`
   - This is file I/O concern, not parsing concern

4. **Error Handling Strategy**: Fire & Forget
   - API logs parse errors at ERROR level
   - Invalid lines skipped, valid lines processed
   - No error returned to Daemon
   - Must maintain observability (log sample of problematic lines)

5. **Generated Code**: Must regenerate protobuf code after schema changes:
   ```bash
   cd idl/protobuf && buf generate
   ```

6. **Shared Domain Reuse**: The `shared/domain` package remains unchanged:
   - `SessionRecord`, `Message`, `MessageContent`, content blocks all stay
   - API will use these same types after parsing

## Similar Implementations Found

### Example 1: Current JSONL Parsing in Daemon
- **File**: `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go:125-143`
- **Relevance**: Shows exact parsing pattern to be moved to API

### Example 2: MessageContent Custom Unmarshaling
- **File**: `/Users/jayce/team-attention/cops/shared/domain/message.go:17-79`
- **Relevance**: Shows polymorphic JSON parsing for content blocks - this logic will be exercised by API after migration

### Example 3: Protobuf to Domain Conversion
- **File**: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go:61-136`
- **Relevance**: Shows current conversion pattern from protobuf - will be replaced with direct JSONL parsing

### Example 4: Domain to Document Mapping
- **File**: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go:66-126`
- **Relevance**: Shows how SessionRecord is converted to MongoDB document - remains unchanged

## Impact Analysis

### Files to Modify

| File | Change Type | Description |
| ---- | ----------- | ----------- |
| `/idl/protobuf/aggregation/v1/aggregation.proto` | Modify | Change `LogBatch.records` to `repeated string jsonl`, remove `SessionRecord` and related messages |
| `/daemon/internal/service/logwatcher/log_service.go` | Modify | Remove sonic.Unmarshal, return raw lines, change buffer type |
| `/daemon/internal/platform/domain/watch.go` | Modify | Change LogBatch.Records from `[]SessionRecord` to `Lines []string` |
| `/daemon/internal/service/logwatcher/outbound/api/api_client_port.go` | Modify | Update LogBatch parameter type |
| `/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go` | Modify | Remove convertRecords/convertSessionRecord/convertMessage, send raw strings |
| `/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go` | Modify | Add JSONL parsing logic, replace convertToDomain with parseJSONLLines |
| `/api/internal/service/aggregation/outbound/repository/port.go` | No Change | LogBatch.Records still uses `[]shareddomain.SessionRecord` |
| `/api/internal/service/aggregation/aggregation_service.go` | No Change | Interface unchanged |
| `/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go` | No Change | SaveBatch unchanged |

### Files to Regenerate

| File | Action |
| ---- | ------ |
| `/shared/gen/grpcstub/aggregation/v1/aggregation.pb.go` | Regenerate via `buf generate` |
| `/shared/gen/grpcstub/aggregation/v1/aggregationv1connect/aggregation.connect.go` | Regenerate via `buf generate` |
| `/web/src/gen/grpcstub/aggregation/v1/aggregation_pb.ts` | Regenerate via `buf generate` |

### Tests to Update

| Test File | Changes Needed |
| --------- | -------------- |
| `/daemon/internal/service/logwatcher/log_service_test.go` | Update to test raw line handling instead of SessionRecord parsing |
| `/shared/domain/message_test.go` | No changes - still validates parsing logic (now used by API) |

## Risks and Considerations

### 1. Breaking Change Deployment
- **Risk**: Old Daemon sending structured data to new API will fail
- **Mitigation**: Synchronized deployment of Daemon and API together
- **Rollback**: Both services must be rolled back together

### 2. Parse Error Handling Location Change
- **Risk**: Parse errors now happen on API side, may affect debugging
- **Mitigation**:
  - Log parse errors at ERROR level in API
  - Include sample of problematic line in log message
  - Include project_id in error logs for tracing

### 3. Memory Usage Pattern Change
- **Risk**: API now needs to allocate memory for parsing
- **Consideration**:
  - Current: Daemon parses, allocates SessionRecord
  - New: API receives strings, parses, allocates SessionRecord
  - Net effect: Similar total memory, different location
  - API may see increased memory usage with multiple concurrent daemons

### 4. Buffer Type Migration
- **Risk**: In-flight buffered records during upgrade
- **Mitigation**:
  - Flush all buffers before daemon shutdown
  - The `Stop()` method already calls `Flush()` before stopping

### 5. Removed Protobuf Messages
- **Risk**: Any code depending on removed protobuf messages will break
- **Files to check**:
  - `SessionRecord` message removed from proto
  - `Message` message removed from proto
  - `Usage` message removed from proto
  - `ContentBlock` and related messages removed from proto
  - `SessionType` enum removed from proto

### 6. Web TypeScript Generated Code
- **Risk**: Frontend may reference removed types
- **Files affected**: `/web/src/gen/grpcstub/aggregation/v1/aggregation_pb.ts`
- **Mitigation**: Check if web code uses these types (likely not, since web reads from MongoDB not gRPC)

## Additional Information for Planning

### Daemon Buffer Migration Details

**Current buffer structure:**
```go
// log_service.go line 29
bufferByProject map[shareddomain.ID][]shareddomain.SessionRecord
```

**New buffer structure:**
```go
// log_service.go (after change)
bufferByProject map[shareddomain.ID][]string  // raw JSONL lines
```

**Methods affected:**
1. `AddRecordsForClaudeDir()` -> rename to `AddLinesForClaudeDir()`
2. `Flush()` - update to iterate over strings, not SessionRecords

### API Handler New Parsing Logic

The API handler should implement parsing similar to current daemon logic:

```go
func parseJSONLLines(lines []string, projectID string) (*repository.LogBatch, []error) {
    var records []shareddomain.SessionRecord
    var parseErrors []error

    for _, line := range lines {
        if line == "" {
            continue
        }

        var record shareddomain.SessionRecord
        if err := sonic.Unmarshal([]byte(line), &record); err != nil {
            parseErrors = append(parseErrors, fmt.Errorf("failed to parse: %s", truncate(line, 100)))
            continue
        }

        records = append(records, record)
    }

    return &repository.LogBatch{
        Records:   records,
        ProjectID: projectID,
    }, parseErrors
}
```

### Error Logging Pattern

Per requirements, use ERROR level logging:

```go
if len(parseErrors) > 0 {
    h.logger.Error("failed to parse some JSONL lines",
        slog.String("projectId", projectID),
        slog.Int("failedCount", len(parseErrors)),
        slog.Int("totalCount", len(lines)),
        slog.String("sampleError", parseErrors[0].Error()),
    )
}
```

### Protobuf Schema After Change

```protobuf
// aggregation.proto (simplified - remove unused messages)
syntax = "proto3";

package aggregation.v1;

option go_package = "github.com/team-attention/cops/shared/gen/grpcstub/aggregation/v1;aggregationv1";

// LogBatch contains raw JSONL lines for batch sending.
message LogBatch {
  repeated string jsonl = 1;      // Raw JSONL lines (unparsed)
  string project_id = 2;
}

// SendLogsReq is the request for sending logs.
message SendLogsReq {
  LogBatch batch = 1;
}

// SendLogsRes is the response for sending logs.
message SendLogsRes {
  bool success = 1;
  string error_message = 2;
  int32 processed_count = 3;
}

// AggregationService handles log aggregation from daemons.
service AggregationService {
  // SendLogs sends a batch of raw JSONL lines to the API server.
  rpc SendLogs(SendLogsReq) returns (SendLogsRes);
}
```

**Messages to remove:**
- `SessionType` enum
- `Usage` message
- `ContentBlockType` enum
- `TextContentBlock`, `ToolUseContentBlock`, `ToolResultContentBlock`, `ThinkingContentBlock` messages
- `ContentBlock` message
- `Message` message
- `SessionRecord` message

### Dependency Order for Changes

1. **First**: Update protobuf schema (`aggregation.proto`)
2. **Second**: Run `buf generate` to regenerate Go/TS code
3. **Third**: Update Daemon domain (`watch.go` - LogBatch)
4. **Fourth**: Update Daemon service (`log_service.go`)
5. **Fifth**: Update Daemon API client (`api_client.go`)
6. **Sixth**: Update API handler (`handler.go`) - add parsing logic
7. **Seventh**: Update tests
8. **Eighth**: Run all tests to verify

---

**Research Complete**: Ready for Planning phase.
