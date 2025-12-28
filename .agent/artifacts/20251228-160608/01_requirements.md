# Requirements

## Request Summary
Fix three data model and persistence issues in the C-Ops project: (1) Remove Project fields that should be tracked at the Session level (ClaudeDir, Worktrees), (2) Ensure RegisteredAt timestamp is properly set when creating new Projects, and (3) Fix MessageContent Block data not being persisted to MongoDB when saving Session Records.

## Acceptance Criteria

- [ ] Criterion 1: Project domain model no longer contains ClaudeDir field (this data should be tracked per session, not per project)
- [ ] Criterion 2: Project domain model no longer contains Worktrees field (this is local daemon state that should not be stored on the server)
- [ ] Criterion 3: All database queries, aggregations, and API responses that referenced ProjectClaudeDirField or ProjectWorktreesField are updated to remove these references
- [ ] Criterion 4: RegisteredAt field is set to current timestamp (time.Now()) when creating new Project records in the database
- [ ] Criterion 5: Existing projects in the database have their RegisteredAt field backfilled or handled gracefully (display logic shows appropriate value instead of "January 1, Year 1")
- [ ] Criterion 6: MessageContent.Blocks data is properly persisted to MongoDB when saving SessionRecord
- [ ] Criterion 7: MessageContent.Blocks data is properly reconstructed when reading SessionRecord from MongoDB
- [ ] Criterion 8: All existing tests continue to pass after the changes
- [ ] Criterion 9: Integration test demonstrates that Blocks are persisted and retrieved correctly (verify using existing test data from message_test.go)

## Scope

### In Scope
- **Item 1: Remove unnecessary Project fields**
  - Remove `ClaudeDir string` field from `shared/domain/project.go`
  - Remove `Worktrees []string` field from `shared/domain/project.go`
  - Remove `ProjectClaudeDirField` constant from `shared/domain/mongoschema/project.go`
  - Remove `ProjectWorktreesField` constant from `shared/domain/mongoschema/project.go`
  - Update `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go` to remove references to these fields
  - Update any API converters or gRPC handlers that reference these fields

- **Item 2: Fix RegisteredAt timestamp generation**
  - Modify `api/internal/service/project/outbound/repository/mongodb/project_repo.go::FindOrCreate()` to set `registeredAt` field when creating new project document (lines 92-96)
  - Add `time.Now()` timestamp to the newDoc bson.M map
  - Verify web dashboard displays correct registration date instead of "January 1, Year 1"

- **Item 3: Fix Session Record MessageContent Blocks persistence**
  - Analyze `api/internal/service/aggregation/outbound/repository/mongodb/adapter.go::toDocument()` function (lines 106-117)
  - Current implementation serializes msg.Content to JSON string, but may not preserve Block structure correctly
  - Verify that sonic.Marshal correctly handles the MessageContent custom MarshalJSON implementation
  - Update retrieval logic in `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go::GetSession()` (lines 467-478) to properly deserialize Blocks
  - Test with actual session records containing all Block types (TextContentBlock, ToolUseContentBlock, ToolResultContentBlock, ThinkingContentBlock)

### Out of Scope
- **Item 1: Session-level ClaudeDir tracking** - While ClaudeDir was removed from Project, adding it to SessionRecord is out of scope for this issue (future enhancement)
- **Item 2: Daemon worktree monitoring** - The daemon should continue watching worktrees locally, but architectural changes to how this is done are out of scope
- **Item 3: Database migration scripts** - Manual MongoDB migration to backfill RegisteredAt for existing projects is out of scope (acceptable to handle at query time)
- **Item 4: API endpoint changes** - No new API endpoints or major API contract changes
- **Item 5: Web dashboard UI changes** - Beyond displaying correct data, no UI/UX improvements to the dashboard

## Constraints
- **Backward compatibility**: Existing session records in the database must continue to work (GetSession must handle both legacy plain text content and new Block-based content)
- **Zero downtime**: Changes should not require coordinated deployment (API server and daemon can be deployed independently)
- **Test coverage**: All domain model changes must maintain or improve existing test coverage (domain tests in shared/domain/*_test.go must pass)
- **Go workspace structure**: Changes must respect the multi-module workspace (shared, api, daemon, cli modules)
- **MongoDB schema flexibility**: Use MongoDB's flexible schema to handle missing fields gracefully (no hard schema migrations)

## Additional Context

### Why ClaudeDir and Worktrees Don't Belong on Project
- **ClaudeDir is execution-context specific**: The same Git project can be executed from different directories (main repo, subdirectory, worktree). The `.claude` directory location varies per execution context, not per project.
- **Worktrees are local state**: Git worktrees are a local development convenience. They exist on the developer's machine and are irrelevant to the server. The daemon can discover and watch them locally without persisting to the database.
- **Example**: A project at `/repos/cops` might have worktrees at `/repos/cops-worktree1` and `/repos/cops-worktree2`. Each worktree might have its own `.claude` directory. This is local state that varies by execution, not project-level metadata.

### MessageContent Persistence Context
- **Domain model is correct**: Tests in `shared/domain/message_test.go` prove that MessageContent correctly parses all Block types from JSON
- **Custom JSON serialization**: MessageContent implements custom MarshalJSON and UnmarshalJSON to handle polymorphic content (string vs []ContentBlock)
- **Storage format**: Currently stored as JSON string in MongoDB field `messageContent`
- **Round-trip requirement**: Data must survive marshal → store → retrieve → unmarshal cycle without data loss
- **Test data available**: File `shared/domain/log_data.jsonl` contains 8 real session records with various Block types to test against

### RegisteredAt Timestamp Context
- **Current behavior**: Projects created via `FindOrCreate` don't have `registeredAt` set, resulting in Go zero value (time.Time{}) which displays as "January 1, Year 1" in web UI
- **Expected behavior**: New projects should have `registeredAt` set to creation time
- **Affected code**: `api/internal/service/project/outbound/repository/mongodb/project_repo.go:92-96`
- **Display locations**: Dashboard project list and project detail pages

## Questions Resolved

| Question | Answer |
| --- | --- |
| Should we migrate existing projects to backfill RegisteredAt? | No - handle gracefully at query/display time. If RegisteredAt is zero value, can display "Unknown" or omit from UI. Focus on fixing new projects. |
| How should MessageContent Blocks be stored in MongoDB? | Continue storing as JSON string in `messageContent` field. The issue is ensuring the serialization preserves Block structure correctly. |
| Should ClaudeDir be added to SessionRecord instead? | Out of scope for this issue. For now, just remove from Project. Future enhancement can add to SessionRecord if needed. |
| What happens to existing session records without Block data? | Backward compatibility required - GetSession must handle both legacy plain text (line 471-474 in dashboard_repo.go already does this) and new Block-based content. |
| Do we need to update protobuf definitions? | Check if Project protobuf in `idl/protobuf/project/v1/project.proto` includes ClaudeDir/Worktrees fields. If yes, remove them and regenerate with `buf generate`. |
