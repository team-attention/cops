# Development Walkthrough: Move JSONL Parsing from Daemon to API

## Summary

This implementation moves JSONL parsing logic from the Daemon service to the API server. Previously, the Daemon parsed JSONL log files into structured `SessionRecord` objects before sending them via gRPC. Now, the Daemon sends raw JSONL text lines, and the API server handles all parsing and validation.

## Architecture Change

### Before
```
JSONL File → Daemon (parse with sonic) → SessionRecord objects → gRPC → API (convert & save)
            ^-- sonic.Unmarshal                                         ^-- protobuf to domain
```

### After
```
JSONL File → Daemon (read raw) → Raw strings → gRPC → API (parse + save)
                                                       ^-- sonic.Unmarshal + convert + save
```

## Code Overview

### Modified Components

#### 1. Protobuf Schema - `idl/protobuf/aggregation/v1/aggregation.proto`
- **Location**: `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto`
- **Changes**:
  - Modified `LogBatch` message: replaced `repeated SessionRecord records` with `repeated string jsonl`
  - Updated service comments to reflect raw JSONL transmission
  - **Note**: Kept unused protobuf messages (`SessionRecord`, `Message`, etc.) as they may be used in the future
- **Impact**: Breaking change - old Daemon will not work with new API

#### 2. Daemon Domain Model - `daemon/internal/platform/domain/watch.go`
- **Location**: `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/watch.go`
- **Changes**:
  - Changed `LogBatch.Records` from `[]shareddomain.SessionRecord` to `Lines []string`
  - Updated comment to reflect raw JSONL lines (unparsed)

#### 3. Daemon Service - `daemon/internal/service/logwatcher/log_service.go`
- **Location**: `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go`
- **Key Changes**:
  - **Removed**: `sonic` import (no longer needed for parsing)
  - **Buffer type change**: `bufferByProject` changed from `map[shareddomain.ID][]shareddomain.SessionRecord` to `map[shareddomain.ID][]string`
  - **HandleFileChange method**:
    - Return type changed from `[]shareddomain.SessionRecord` to `[]string`
    - Removed JSON parsing logic (lines 130-138)
    - Now returns raw text lines directly
  - **AddRecordsForClaudeDir → AddLinesForClaudeDir**: Renamed method to reflect new behavior
  - **Flush method**: Updated to work with string slices instead of SessionRecord slices
- **Why**: Daemon no longer needs to parse JSONL - it just reads and buffers raw text

#### 4. Daemon API Client - `daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`
- **Location**: `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`
- **Key Changes**:
  - **Removed imports**: `timestamppb`, `shareddomain` (no longer needed)
  - **SendLogs method**: Now sends `batch.Lines` as `Jsonl` field (simple string array)
  - **Deleted functions**: Removed all conversion functions (84 lines deleted):
    - `convertRecords()`
    - `convertSessionRecord()`
    - `convertSessionType()`
    - `convertMessage()`
- **Why**: No conversion needed - just send raw strings to API

#### 5. Daemon Inbound Handler - `daemon/internal/service/logwatcher/inbound/worker/fsnotify/handler.go`
- **Location**: `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/inbound/worker/fsnotify/handler.go`
- **Changes**:
  - Variable renamed: `records` → `lines`
  - Method call updated: `AddRecordsForClaudeDir` → `AddLinesForClaudeDir`

#### 6. API Handler - `api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go`
- **Key Changes**:
  - **Added imports**: `fmt`, `sonic`
  - **SendLogs method**:
    - Calls new `parseJSONLLines()` instead of `convertToDomain()`
    - Implements Fire & Forget error handling (logs parse errors at ERROR level but doesn't fail request)
    - Logs sample error with metadata (projectId, failedCount, totalCount)
  - **New method - parseJSONLLines()**:
    - Parses raw JSONL lines into `shareddomain.SessionRecord` using `sonic.Unmarshal`
    - Skips empty lines
    - Collects parse errors with truncated line content (100 chars max)
    - Returns both parsed batch and error list
  - **Deleted functions**: Removed all old conversion functions (75 lines deleted):
    - `convertToDomain()`
    - `convertSessionType()`
    - `convertMessage()`
- **Why**: API now handles JSONL parsing, which was previously done in Daemon

#### 7. Generated Code - `shared/gen/grpcstub/aggregation/v1/`
- **Location**:
  - `/Users/jayce/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregation.pb.go`
  - `/Users/jayce/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregationv1connect/aggregation.connect.go`
- **Changes**: Auto-generated from updated protobuf schema via `buf generate`
- **Key updates**:
  - `LogBatch` struct now has `Jsonl []string` field instead of `Records []*SessionRecord`
  - Generated `GetJsonl()` method

### Unchanged Components (No Modifications Required)

#### API Service Layer
- **File**: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/aggregation_service.go`
- **Reason**: Service layer interface unchanged - still receives `repository.LogBatch` with `[]SessionRecord`

#### API Repository Layer
- **Files**:
  - `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/port.go`
  - `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go`
- **Reason**: Repository still works with domain models (`SessionRecord`), unaffected by wire format change

#### Shared Domain Models
- **Files**:
  - `/Users/jayce/team-attention/cops/shared/domain/session.go`
  - `/Users/jayce/team-attention/cops/shared/domain/message.go`
  - `/Users/jayce/team-attention/cops/shared/domain/content_block.go`
- **Reason**: Domain models unchanged - still used by API after parsing

## Technical Details

### JSONL Parsing Logic Migration

**Old location** (Daemon - `log_service.go:130-138`):
```go
var record shareddomain.SessionRecord
if err := sonic.Unmarshal([]byte(line), &record); err != nil {
    s.logger.Debug("failed to parse JSONL line",
        slog.String("path", path),
        slog.Any("error", err),
    )
    continue
}
records = append(records, record)
```

**New location** (API - `handler.go:parseJSONLLines()`):
```go
var record shareddomain.SessionRecord
if err := sonic.Unmarshal([]byte(line), &record); err != nil {
    parseErrors = append(parseErrors, fmt.Errorf("parse error: %s (line: %.100s...)", err.Error(), line))
    continue
}
records = append(records, record)
```

**Key difference**: API logs errors at ERROR level (per requirements), while Daemon previously logged at Debug level.

### Fire & Forget Error Handling

The API implements Fire & Forget pattern for parse errors:
- Accepts all batches regardless of parse success
- Logs parse errors at ERROR level with metadata
- Processes valid lines even if some fail
- Never returns error to Daemon for parse failures

Example log output:
```
failed to parse some JSONL lines projectId=abc123 failedCount=2 totalCount=10 sampleError="parse error: invalid character..."
```

### Buffer Type Migration

**Daemon buffer changed**:
```go
// Before
bufferByProject map[shareddomain.ID][]shareddomain.SessionRecord

// After
bufferByProject map[shareddomain.ID][]string
```

This affects:
- Memory allocation location (now in API instead of Daemon)
- Buffering behavior (raw strings lighter than parsed objects)

## Testing

### Unit Tests Added

Created new test suite using Ginkgo/Gomega BDD pattern:
- **Suite file**: `api/internal/service/aggregation/inbound/grpc/connectrpc/connectrpc_suite_test.go`
- **Test file**: `api/internal/service/aggregation/inbound/grpc/connectrpc/handler_test.go`

**Test Scenarios**:
1. Valid JSONL lines parsing (all lines valid)
2. Invalid JSONL lines (skips invalid, parses valid)
3. Empty lines handling (skips empty)
4. Text content parsing (string content)
5. Block content parsing (array content)

### Test Results

All tests passing:
```bash
✓ go test ./api/... -v          # 5 specs passed
✓ go test ./daemon/... -v       # 3 specs passed
✓ go test ./shared/... -v       # 43 specs passed
✓ go build ./api/... ./daemon/... ./shared/...
```

### Verification Commands Run
```bash
cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate  # SUCCESS
go build ./shared/...                                              # SUCCESS
go build ./daemon/...                                              # SUCCESS
go build ./api/...                                                 # SUCCESS
go test ./daemon/... -v                                            # 3 PASS
go test ./api/... -v                                               # 5 PASS
go test ./shared/... -v                                            # 43 PASS
```

## Breaking Changes

**CRITICAL: This is a breaking change requiring synchronized deployment.**

### What Changed
- Protobuf schema: `LogBatch.records` → `LogBatch.jsonl`
- Wire format: Structured objects → Raw strings
- Old Daemon will not work with new API and vice versa

### Deployment Requirements
1. Deploy new API and new Daemon together in same release
2. Rolling updates not supported - requires full restart
3. No backward compatibility maintained (per requirements)

### Rollback Considerations
- Both services must be rolled back together
- Database schema unchanged (no migration needed)
- In-flight buffered logs will be lost during restart

## Issues & Resolutions

| Issue | Resolution |
| ----- | ---------- |
| Breaking change in protobuf schema | Accepted as requirement - synchronized deployment needed |
| Parse errors now happen on API side | Implemented Fire & Forget pattern with ERROR-level logging |
| Memory allocation moved to API | Similar total memory usage, just different location |
| Loss of intermediate parse validation in Daemon | Trade-off accepted - API validates all input |

## Related Files

### Modified Files (11)
1. `idl/protobuf/aggregation/v1/aggregation.proto` - Schema change
2. `daemon/internal/platform/domain/watch.go` - Domain model
3. `daemon/internal/service/logwatcher/log_service.go` - Core parsing removal
4. `daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go` - Conversion removal
5. `daemon/internal/service/logwatcher/inbound/worker/fsnotify/handler.go` - Method call update
6. `api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go` - Parsing addition
7. `api/go.mod` / `api/go.sum` - Test dependencies (Ginkgo/Gomega)
8. `shared/gen/grpcstub/aggregation/v1/aggregation.pb.go` - Generated
9. `shared/gen/grpcstub/aggregation/v1/aggregationv1connect/aggregation.connect.go` - Generated
10. `web/src/gen/grpcstub/aggregation/v1/aggregation_pb.ts` - Generated (TypeScript)

### Test Files Added (2)
1. `api/internal/service/aggregation/inbound/grpc/connectrpc/connectrpc_suite_test.go`
2. `api/internal/service/aggregation/inbound/grpc/connectrpc/handler_test.go`

### Unchanged Files (No modifications needed)
- `shared/domain/session.go` - Domain models
- `shared/domain/message.go` - Message models
- `api/internal/service/aggregation/aggregation_service.go` - Service layer
- `api/internal/service/aggregation/outbound/repository/*.go` - Repository layer

## Impact Summary

### Lines Changed
- **Deleted**: ~240 lines (conversion functions removed)
- **Added**: ~95 lines (parsing logic, tests)
- **Net change**: -145 lines

### Complexity Reduction
- Removed duplicate conversion logic between Daemon and API
- Simplified Daemon responsibility (just read files, no parsing)
- Centralized JSONL parsing in single location (API)

### Performance Considerations
- Daemon now lighter (no parsing overhead)
- API handles more processing (parsing from multiple daemons)
- Network payload similar (raw strings vs protobuf objects)
- Memory usage redistributed (from Daemon to API)
