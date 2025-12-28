# Pre-PR Code Review

## Review Summary
- **Status: PASS** (with minor notes)
- **Files Reviewed**: 12
- **Issues Found**: 2 (Critical: 0, Warning: 1, Info: 1)

## Plan vs Implementation Analysis

The implementation successfully moves JSONL parsing from Daemon to API Server as specified in the plan. All core requirements have been met.

## Files Reviewed

### `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto`

**Change Made**: Updated `LogBatch` message to use `repeated string jsonl` instead of `repeated SessionRecord records`, updated service comment.

**Deviation from Plan**: The plan specified removing all unused protobuf messages (SessionRecord, Message, Usage, ContentBlock types, enums - lines 9-103). These messages were kept.

**Verdict**: ACCEPTABLE

**Reasoning**: Keeping unused messages is not harmful - they don't affect runtime behavior. The generated code is larger than necessary, but this is a cosmetic issue. The critical change (LogBatch using strings) was implemented correctly. Removing dead code could be done in a follow-up cleanup.

---

### `/Users/jayce/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregation.pb.go`

**Status**: Correctly regenerated via `buf generate`

The generated code correctly reflects:
- `LogBatch.Jsonl []string` instead of `Records []*SessionRecord`
- Updated dependency indices

---

### `/Users/jayce/team-attention/cops/shared/gen/grpcstub/aggregation/v1/aggregationv1connect/aggregation.connect.go`

**Status**: Correctly regenerated via `buf generate`

Service comments updated to reflect "raw JSONL lines" instead of "session records".

---

### `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/aggregation/v1/aggregation_pb.ts`

**Status**: Correctly regenerated via `buf generate`

TypeScript types updated with `jsonl: string[]` instead of `records: SessionRecord[]`.

---

### `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/watch.go`

**Status**: Matches plan exactly

```go
// Before
type LogBatch struct {
    Records   []shareddomain.SessionRecord
    ProjectID shareddomain.ID
}

// After
type LogBatch struct {
    Lines     []string
    ProjectID shareddomain.ID
}
```

---

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go`

**Status**: Matches plan exactly

All changes implemented correctly:
- Removed `sonic` import
- Changed `bufferByProject` type from `map[shareddomain.ID][]shareddomain.SessionRecord` to `map[shareddomain.ID][]string`
- `HandleFileChange` now returns `[]string` instead of `[]shareddomain.SessionRecord`
- Removed JSONL parsing logic (sonic.Unmarshal)
- Renamed `AddRecordsForClaudeDir` to `AddLinesForClaudeDir`
- Updated `Flush` method to use `Lines` field

---

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/inbound/worker/fsnotify/handler.go`

**Status**: Matches plan exactly

- Updated variable name from `records` to `lines`
- Changed method call from `AddRecordsForClaudeDir` to `AddLinesForClaudeDir`

---

### `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go`

**Status**: Matches plan exactly

- Removed unused imports (`timestamppb`, `shareddomain`)
- Updated `SendLogs` to use `batch.Lines` instead of `convertRecords(batch.Records)`
- Removed all conversion functions (`convertRecords`, `convertSessionRecord`, `convertSessionType`, `convertMessage`)

---

### `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go`

**Status**: Matches plan exactly

- Added `fmt` and `sonic` imports
- Replaced `convertToDomain(pbBatch)` with `h.parseJSONLLines(pbBatch.GetJsonl(), pbBatch.GetProjectId())`
- Added error logging for parse failures (Fire & Forget pattern)
- Added `parseJSONLLines` method implementing JSONL parsing
- Removed old conversion functions (`convertToDomain`, `convertSessionType`, `convertMessage`)

---

### `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler_test.go`

**Status**: Matches plan exactly

Test file created with all specified test cases:
- `TestParseJSONLLines_ValidLines` - Happy path
- `TestParseJSONLLines_InvalidLines` - Error handling
- `TestParseJSONLLines_EmptyLines` - Empty filter
- `TestParseJSONLLines_MessageContent` - Text content parsing
- `TestParseJSONLLines_ContentBlocks` - Block content parsing

---

## Warnings

### Warning 1: Unused Protobuf Messages Not Removed

**File**: `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto`

**Description**: The plan specified removing all unused protobuf messages (SessionType enum, Usage, ContentBlockType, TextContentBlock, ToolUseContentBlock, ToolResultContentBlock, ThinkingContentBlock, ContentBlock, Message, SessionRecord) but they remain in the file.

**Impact**: Low - No runtime impact, just larger generated code size.

**Recommendation**: Consider removing in a follow-up cleanup commit if desired.

---

## Info

### Info 1: Unrelated Change Detected

**File**: `/Users/jayce/team-attention/cops/.agent/agents/clarify.md`

**Description**: This file has an unrelated change (adding "ultrathink." to a line). This appears to be unintentional and should be reverted before committing.

**Recommendation**: Revert this change with:
```bash
git checkout -- .agent/agents/clarify.md
```

---

## Test Verification

- [x] All tests pass: `go test ./daemon/... ./api/... ./shared/...`
- [x] Build succeeds: `go build ./shared/... ./daemon/... ./api/...`
- [x] New test file created: `api/internal/service/aggregation/inbound/grpc/connectrpc/handler_test.go`

## Test Results Summary

```
ok  github.com/team-attention/cops/daemon/internal/service/logwatcher    0.212s
ok  github.com/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc    0.213s
ok  github.com/team-attention/cops/shared/domain    0.479s
```

## Approval Notes

The implementation correctly achieves the primary objective of moving JSONL parsing from Daemon to API Server:

1. **Daemon Changes**: No longer parses JSONL - collects raw strings and transmits them
2. **API Changes**: Receives raw JSONL strings and parses them using `sonic.Unmarshal`
3. **Error Handling**: Implements Fire & Forget pattern - logs parse errors but always returns success
4. **Protocol Changes**: `LogBatch.jsonl` now contains raw strings instead of parsed records

The code quality is good:
- Follows project rules and hexagonal architecture
- Uses existing `sonic` package for JSON parsing
- Proper error logging with structured fields
- Test coverage for the new parsing logic

**Ready for PR creation** after reverting the unrelated `.agent/agents/clarify.md` change.
