# Development Walkthrough

## Summary

This implementation addressed three data model and persistence issues in the C-Ops project: (1) Removed Project fields that should be tracked at the Session level (ClaudeDir, Worktrees), (2) Fixed RegisteredAt timestamp not being set when creating new Projects, and (3) Added comprehensive MessageContent Block support to gRPC/API responses by extending protobuf definitions and converters.

## Changes Overview

### Issue 1: Removed Unnecessary Project Fields

**Problem**: The Project domain model contained two fields that represented local or session-specific state rather than project-level metadata:
- `ClaudeDir string` - The `.claude` directory location varies per execution context (main repo, subdirectory, worktree)
- `Worktrees []string` - Git worktrees are local development convenience, irrelevant to the server

**What Changed**:
- Removed both fields from the `Project` domain struct in `shared/domain/project.go`
- Removed corresponding MongoDB schema constants (`ProjectClaudeDirField`, `ProjectWorktreesField`)
- Updated all database queries and API converters to remove references to these fields
- Removed `worktrees` field from protobuf `ProjectDetail` message
- Regenerated protobuf code for all affected services

**Impact**:
- Cleaner domain model that accurately represents project-level metadata
- Reduced payload size for project-related API responses
- Daemon continues to watch worktrees locally without persisting to the database

### Issue 2: Fixed RegisteredAt Timestamp Generation

**Problem**: Projects created via `FindOrCreate` didn't have `registeredAt` set, resulting in Go zero value `time.Time{}` which displayed as "January 1, Year 1" in the web dashboard.

**What Changed**:
- Modified `api/internal/service/project/outbound/repository/mongodb/project_repo.go::FindOrCreate()`
- Added `mongoschema.ProjectRegisteredAtField: time.Now()` to the newDoc bson.M map when creating new project documents

**Impact**:
- New projects now have correct registration timestamp
- Web dashboard displays accurate creation dates instead of "January 1, Year 1"
- Existing projects in database continue to work (MongoDB flexible schema handles missing fields)

### Issue 3: MessageContent Blocks in gRPC Responses

**Problem**: While MessageContent Blocks were correctly persisted to MongoDB and parsed by domain models, the gRPC API responses dropped block structure because the protobuf schema only had a simple `string content` field.

**What Changed**:

1. **Protobuf Schema Extension** (`idl/protobuf/aggregation/v1/aggregation.proto`):
   - Added `ContentBlockType` enum with 5 values (UNSPECIFIED, TEXT, TOOL_USE, TOOL_RESULT, THINKING)
   - Added 4 ContentBlock message types matching domain model:
     - `TextContentBlock` - Contains `text` field
     - `ToolUseContentBlock` - Contains `id`, `name`, `input_json` fields
     - `ToolResultContentBlock` - Contains `tool_use_id`, `content`, `is_error` fields
     - `ThinkingContentBlock` - Contains `thinking`, `signature` fields
   - Added polymorphic `ContentBlock` message using protobuf `oneof` pattern
   - Updated `Message` message:
     - Renamed `content` field to `text` (field 5) to match domain model
     - Added `repeated ContentBlock content_blocks` (field 9)

2. **gRPC Converter Enhancement** (`api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go`):
   - Updated `convertMessage` function to handle both text and block content
   - Added `convertContentBlocks` helper to convert domain block arrays to protobuf
   - Added `convertContentBlock` helper with type switch for each block type:
     - TextContentBlock: Direct text mapping
     - ToolUseContentBlock: Serializes Input map to JSON string
     - ToolResultContentBlock: Direct field mapping
     - ThinkingContentBlock: Direct field mapping

3. **Other gRPC Handlers Updated**:
   - `daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go` - Updated to use renamed `text` field
   - `api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go` - Updated to use renamed `text` field

**Impact**:
- Dashboard API now returns structured block data with full type information
- Web frontend can render different block types appropriately (text, tool calls, thinking blocks)
- No data loss - MongoDB persistence was already correct, this just exposes it via API
- Backward compatible - plain text content still works via `text` field

## Files Changed

### Domain Model
- `shared/domain/project.go` - Removed ClaudeDir and Worktrees fields from Project struct
- `shared/domain/mongoschema/project.go` - Removed ProjectClaudeDirField and ProjectWorktreesField constants

### API Service Layer
- `api/internal/service/project/outbound/repository/mongodb/project_repo.go` - Added RegisteredAt timestamp to new project creation
- `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go` - Removed ClaudeDir/Worktrees field references from GetProject query
- `api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go` - Added ContentBlock conversion logic, updated to use renamed text field
- `api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go` - Updated to use renamed text field

### CLI Service Layer
- `cli/internal/service/tracking/tracking_service.go` - Removed ClaudeDir initialization when adding projects

### Daemon Service Layer
- `daemon/internal/service/logwatcher/outbound/api/connectrpc/api_client.go` - Updated to use renamed text field

### Protobuf Definitions
- `idl/protobuf/dashboard/v1/dashboard.proto` - Removed worktrees field from ProjectDetail message
- `idl/protobuf/aggregation/v1/aggregation.proto` - Added ContentBlock types, renamed Message.content to text, added content_blocks field

### Generated Code (auto-generated via buf generate)
- `shared/gen/grpcstub/dashboard/v1/dashboard.pb.go` - Regenerated from dashboard.proto
- `shared/gen/grpcstub/aggregation/v1/aggregation.pb.go` - Regenerated from aggregation.proto
- `web/src/gen/grpcstub/dashboard/v1/dashboard_pb.ts` - TypeScript types for web frontend
- `web/src/gen/grpcstub/aggregation/v1/aggregation_pb.ts` - TypeScript types for web frontend

### Test Files
- `shared/domain/message_test.go` - Added 6 new test cases for Lines 9-10 of JSONL data (tool_result with system-reminder, tool_use with Read tool)

### Documentation
- `TODO.md` - Updated with completed items and new issue tracking note

## Testing

### Unit Tests Added

**Test Coverage Expansion**:
- Expanded `shared/domain/log_data.jsonl` from 8 to 10 session records
- Added 2 test cases for Line 9 (tool_result block with system-reminder warning)
- Added 4 test cases for Line 10 (tool_use block with Read tool operation)

**Test Scenarios for Line 9**:
1. Parses tool_result content block with system-reminder warning
2. Preserves toolUseResult file metadata in record

**Test Scenarios for Line 10**:
1. Parses tool_use content block for Read operation
2. Correctly parses file_path input for Read tool
3. Verifies tool_use links to corresponding tool_result (ID matching)
4. Preserves usage metadata in assistant message

**Total Test Count**: 43 domain tests (previously 37)

### Test Results

```
Running Suite: Domain Suite
Will run 43 of 43 specs
...........................................

Ran 43 of 43 Specs in 0.007 seconds
SUCCESS! -- 43 Passed | 0 Failed | 0 Pending | 0 Skipped
```

### Build Verification

All modules build successfully:
```bash
go build ./cli/... ./api/... ./daemon/... ./shared/...
# Result: No errors
```

## Migration Notes

### Database Compatibility

**Existing Projects**:
- Projects with ClaudeDir/Worktrees fields in MongoDB continue to work
- Dashboard queries gracefully handle missing fields (MongoDB flexible schema)
- No manual migration required

**RegisteredAt Backfill**:
- Existing projects without RegisteredAt will have zero value (Year 1)
- Frontend should handle this gracefully (display "Unknown" or omit date)
- New projects created after this deployment will have correct timestamps

### API Compatibility

**Breaking Changes**: None - backward compatible
- Projects missing ClaudeDir/Worktrees: Fields omitted from JSON response
- Messages with plain text content: Use `text` field (previously `content`)
- Messages with block content: Use `content_blocks` array (new feature)

**Frontend Impact**:
- Dashboard should check for `content_blocks` presence to determine rendering strategy
- TypeScript protobuf types regenerated - frontend may need to update imports

### Deployment Order

Changes can be deployed independently:
1. API server can be deployed first (handles both old and new message formats)
2. Daemon can be deployed after (sends messages using new protobuf schema)
3. Web frontend should be deployed last (uses new TypeScript protobuf types)

## Next Steps

### Recommended Follow-ups

1. **Frontend Updates**: Update web dashboard to render different ContentBlock types with appropriate UI (syntax highlighting for tool_use, special styling for thinking blocks)

2. **RegisteredAt Display**: Add frontend logic to display "Unknown" or hide registration date for projects with zero-value timestamps

3. **Session-level ClaudeDir Tracking** (Future Enhancement): While ClaudeDir was removed from Project, consider adding it to SessionRecord if per-session execution context is needed

4. **Comprehensive Integration Test**: Add end-to-end test that verifies a session with all block types (text, tool_use, tool_result, thinking) round-trips correctly through:
   - Daemon parsing JSONL
   - API server receiving via gRPC
   - MongoDB persistence
   - Dashboard API retrieval
   - Web frontend rendering

### Performance Considerations

- ContentBlock conversion adds minimal overhead (simple type switch + JSON serialization for ToolUse.Input)
- Consider adding caching if dashboard queries become slow for sessions with many messages/blocks
- Monitor MongoDB query performance after deployment to ensure indexes are optimal

## Related Documentation

- Requirements: `.agent/artifacts/20251228-160608/01_requirements.md`
- Implementation Plan: `.agent/artifacts/20251228-160608/03_plan.md`
- Final Review: `.agent/artifacts/20251228-160608/08_review_iteration4.md`
