# Requirements

## Request Summary

Move JSONL parsing logic from Daemon to API Server. Currently, Daemon parses JSONL logs into structured `SessionRecord` objects before sending to API. After this change, Daemon will send raw JSONL text lines to API, and API will handle all parsing and validation logic.

## Current Architecture Analysis

**Current Flow:**
```
JSONL Log File → Daemon (parse) → SessionRecord objects → API (save to DB)
                  ↑ Parsing happens here
```

**Target Flow:**
```
JSONL Log File → Daemon (read raw) → Raw JSONL lines → API (parse + save)
                                                         ↑ Parsing happens here
```

**Current Implementation:**
- **Daemon** (`daemon/internal/service/logwatcher/log_service.go`):
  - Reads JSONL files line-by-line
  - Parses each line using `sonic.Unmarshal` into `shareddomain.SessionRecord`
  - Sends structured data via `api.APIClientPort.SendLogs()`

- **API** (`api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go`):
  - Receives structured `SessionRecord` objects via gRPC
  - Converts from protobuf to domain models
  - Saves to MongoDB via repository

## Acceptance Criteria

- [ ] **AC1**: Daemon reads JSONL log files and sends raw text lines (not parsed) to API
- [ ] **AC2**: API receives raw JSONL text lines from Daemon
- [ ] **AC3**: API parses JSONL lines into `SessionRecord` domain models before saving
- [ ] **AC4**: All existing JSONL parsing logic is removed from Daemon
- [ ] **AC5**: Protobuf schema updated to support raw text transmission
- [ ] **AC6**: All existing tests pass after refactoring
- [ ] **AC7**: Error handling for malformed JSONL occurs on API side
- [ ] **AC8**: No data loss during migration (existing buffered records handled)

## Scope

### In Scope
- Modify protobuf schema to accept raw JSONL text lines
- Update Daemon to send raw text instead of parsed objects
- Move JSONL parsing logic from Daemon to API
- Update API to parse incoming raw JSONL
- Update error handling to reflect new parsing location
- Update tests to reflect architectural changes

### Out of Scope
- Changing the JSONL file format itself
- Modifying MongoDB schema
- Changing flush/buffering mechanism (timing and batching remain same)
- Performance optimization beyond the scope of this refactor
- Dashboard or CLI changes

## Constraints

- Must maintain backward compatibility during deployment (consider rolling update scenario)
- Must not lose buffered logs during daemon restart
- Must preserve existing error logging and debugging capabilities
- Must follow project's Hexagonal Architecture patterns
- Must use `sonic` library for JSON parsing (consistent with existing code)

## Technical Questions for Clarification

### 1. Protobuf Schema Design

**Current**: `LogBatch` contains `repeated SessionRecord records`

**Option A - Send as structured array of strings:**
```protobuf
message LogBatch {
  repeated string raw_jsonl_lines = 1;  // Each line is unparsed JSONL
  string project_id = 2;
}
```

**Option B - Send as single multiline string:**
```protobuf
message LogBatch {
  string raw_jsonl_text = 1;  // Contains multiple lines separated by \n
  string project_id = 2;
}
```

**Question 1.1**: Which schema approach do you prefer? (Option A is recommended for line-level error handling)

**Question 1.2**: Should we maintain the existing `SessionRecord` message definition in proto for potential future use, or remove it entirely?

### 2. Error Handling Strategy

**Current**: Daemon logs parse errors and skips invalid lines silently

**Question 2.1**: Should API maintain the same behavior (log and skip invalid lines), or should it:
- Return partial success (X/Y lines processed)?
- Return error and reject the entire batch?
- Log but continue processing?

**Question 2.2**: How should we handle the case where ALL lines in a batch fail parsing? Should API return success=false or success=true with processedCount=0?

### 3. Backward Compatibility

**Question 3.1**: Do we need to support gradual rollout where old Daemon sends structured data and new API must accept both? Or can we do a synchronized deployment?

**Question 3.2**: If we need backward compatibility, should we:
- Add a new gRPC endpoint (`SendRawLogs`) and deprecate `SendLogs`?
- Use a discriminator field in the existing message?
- Accept this is a breaking change requiring synchronized deployment?

### 4. Buffering and Offset Tracking

**Current**: Daemon buffers parsed `SessionRecord` objects and tracks file offsets

**Question 4.1**: Should offset calculation remain in Daemon (since it knows bytes read), or move to API? (Recommend: keep in Daemon as it's file I/O concern)

**Question 4.2**: The buffering mechanism (`bufferByProject`) currently stores `[]SessionRecord`. Should we:
- Change to buffer `[]string` (raw JSONL lines)?
- Keep the same buffer structure and just change what gets sent?

### 5. Logging and Observability

**Question 5.1**: Should we add new metrics/logs to track:
- Parse success rate per batch?
- Parse errors with sample of problematic lines?
- Performance metrics (parse time on API side)?

**Question 5.2**: Current Daemon logs: "failed to parse JSONL line" at Debug level. Should API maintain Debug level or increase to Warn/Error when parsing fails?

### 6. Testing Strategy

**Question 6.1**: Should we add integration tests that verify end-to-end flow (Daemon → API → MongoDB), or rely on unit tests for each component?

**Question 6.2**: Do you want test coverage for specific malformed JSONL scenarios (truncated JSON, invalid UTF-8, etc.)?

## Additional Context

- Related Files:
  - Daemon: `/daemon/internal/service/logwatcher/log_service.go`
  - Daemon API Client: `/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`
  - API Handler: `/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go`
  - API Service: `/api/internal/service/aggregation/aggregation_service.go`
  - Protobuf: `/idl/protobuf/aggregation/v1/aggregation.proto`
  - Shared Domain: `/shared/domain/message.go`, `/shared/domain/session.go`

- Dependencies:
  - Uses `github.com/bytedance/sonic` for JSON parsing
  - Uses ConnectRPC for gRPC communication
  - MongoDB for persistence

- Performance Considerations:
  - Current implementation uses `bufio.Scanner` with 1MB buffer
  - Parsing currently happens in Daemon's file watcher goroutine
  - Moving parsing to API means API must handle concurrent parsing from multiple daemons

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| **1. Protobuf Schema** | Use `repeated string jsonl` field name |
| **2. Error Handling** | Fire & Forget - API logs errors but doesn't fail request |
| **3. Backward Compatibility** | No - remove old SessionRecord implementation completely |
| **4. Buffering** | Yes - buffer raw strings instead of parsed objects |
| **5. Logging Level** | Error level for parse failures |
| **6. Testing** | Unit tests only |

## Implementation Decisions

Based on the answers above:

1. **Protobuf Schema Change**:
   ```protobuf
   message LogBatch {
     repeated string jsonl = 1;  // Raw JSONL lines (unparsed)
     string project_id = 2;
   }
   ```
   - Remove `SessionRecord` message definition
   - This is a breaking change - old Daemon will not work with new API

2. **Error Handling**:
   - API accepts all batches (Fire & Forget)
   - Parse errors logged at ERROR level with problematic line samples
   - Invalid lines skipped, valid lines processed
   - No error response returned to Daemon

3. **Daemon Changes**:
   - Buffer `[]string` instead of `[]SessionRecord`
   - Remove all `sonic.Unmarshal` calls from log_service.go
   - Send raw text lines to API

4. **API Changes**:
   - Parse JSONL in handler using `sonic.Unmarshal`
   - Log parse errors at ERROR level
   - Continue processing valid lines even if some fail

---

**Ready for Research phase**: Requirements are now complete and clarified.
