# Implementation Plan

## Overview

This plan addresses three issues in the C-Ops codebase:
1. Remove `ClaudeDir` and `Worktrees` fields from the `Project` domain model (they are local/per-session state)
2. Fix `RegisteredAt` timestamp not being set when creating new projects
3. Fix MessageContent Blocks not appearing in gRPC responses (MongoDB persistence is correct; gRPC converter needs updating)

Estimated complexity: Medium. Changes span shared domain, API service, and protobuf layers.

## Selected Packages

No new packages are needed. All changes use existing Go standard library and project dependencies.

| Problem | Package | Context7 ID | Reason for Selection |
| ------- | ------- | ----------- | -------------------- |
| Timestamp | `time` (stdlib) | N/A | Standard library for time operations |
| JSON serialization | `bytedance/sonic` | Already in use | Existing project dependency |

## Architecture Decisions

### Decision 1: Remove Fields vs Deprecate

**Choice**: Remove fields entirely from domain model
**Rationale**: Per requirements, backward compatibility is not a concern for this project. Clean removal is preferred over deprecation. MongoDB's flexible schema handles missing fields gracefully.

### Decision 2: Protobuf Field Removal Strategy

**Choice**: Remove `worktrees` field from `ProjectDetail` message entirely
**Rationale**: The field is unused and represents local state. No clients depend on this field. Protobuf field number 4 will become reserved/unused (no explicit reservation needed for this project).

### Decision 3: MessageContent Blocks in gRPC

**Choice**: Serialize Blocks to JSON string for gRPC `content` field
**Rationale**: The current protobuf `Message.content` is a string field. Rather than adding complex ContentBlock protobuf types (which would require significant schema changes), we will serialize block content to JSON for the gRPC response. This maintains backward compatibility and matches how data is stored in MongoDB.

### Decision 4: RegisteredAt Backfill Strategy

**Choice**: No migration; handle gracefully at display time
**Rationale**: Per requirements, backfilling existing projects is out of scope. Zero values (Year 1) should be displayed as "Unknown" or handled by the frontend.

## Implementation Phases

### Phase 1: Remove Project Fields from Domain Model

**Goal**: Remove `ClaudeDir` and `Worktrees` fields from the central `Project` domain model

**Dependencies**: None (this is the foundational change)

#### Task 1.1: Remove fields from domain model

- **File**: `/Users/jayce/team-attention/cops/shared/domain/project.go:18-19`
- **Change**: Delete lines 18-19 containing `ClaudeDir` and `Worktrees` fields from `Project` struct
- **Reason**: These fields represent local/per-session state, not project-level metadata
- **Test**: Run `go build ./shared/...` to verify compilation

**Before**:
```go
type Project struct {
    ProjectAbstract
    IsGitProject bool      `json:"gitProject"`
    ClaudeDir    string    `json:"claudeDir"`           // DELETE THIS LINE
    Worktrees    []string  `json:"worktrees,omitempty"` // DELETE THIS LINE
    RegisteredAt time.Time `json:"registeredAt"`
}
```

**After**:
```go
type Project struct {
    ProjectAbstract
    IsGitProject bool      `json:"gitProject"`
    RegisteredAt time.Time `json:"registeredAt"`
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Struct compiles | N/A | No compilation errors | N/A |
| JSON serialization | Project without ClaudeDir/Worktrees | Valid JSON without these fields | Happy path |

#### Task 1.2: Remove MongoDB schema constants

- **File**: `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go:17,20`
- **Change**: Delete `ProjectClaudeDirField` constant (line 17) and `ProjectWorktreesField` constant (line 20)
- **Reason**: No longer needed since fields are removed from domain model
- **Test**: Run `go build ./shared/...` to verify compilation

**Delete these lines**:
```go
ProjectClaudeDirField    = "claudeDir"    // DELETE
ProjectWorktreesField    = "worktrees"    // DELETE
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Constants removed | N/A | No compilation errors | N/A |

#### Task 1.3: Update dashboard repository

- **File**: `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go:278-279`
- **Change**: Remove the two lines setting `ClaudeDir` and `Worktrees` fields in the `Project` struct initialization
- **Reason**: Fields no longer exist on domain model
- **Test**: Run `go build ./api/...` to verify compilation

**Delete these lines**:
```go
ClaudeDir:    mongoutil.Get[string](doc, mongoschema.ProjectClaudeDirField),   // DELETE
Worktrees:    mongoutil.GetSlice[string](doc, mongoschema.ProjectWorktreesField), // DELETE
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Repository compiles | N/A | No compilation errors | N/A |
| GetProject returns valid data | Existing project document | Project without ClaudeDir/Worktrees | Happy path |

#### Task 1.4: Update gRPC converter

- **File**: `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go:39`
- **Change**: Remove line 39 `Worktrees: p.Project.Worktrees,`
- **Reason**: Field no longer exists on domain model
- **Test**: Run `go build ./api/...` to verify compilation

**Delete this line**:
```go
Worktrees:    p.Project.Worktrees,  // DELETE
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Converter compiles | N/A | No compilation errors | N/A |

### Phase 2: Update Protobuf Definition

**Goal**: Remove `worktrees` field from protobuf and regenerate code

**Dependencies**: Phase 1 must be complete (converter no longer references Worktrees)

#### Task 2.1: Remove worktrees from protobuf

- **File**: `/Users/jayce/team-attention/cops/idl/protobuf/dashboard/v1/dashboard.proto:81-82`
- **Change**: Delete lines 81-82 containing `repeated string worktrees = 4;` and its comment
- **Reason**: Field is local state, should not be in API contract
- **Test**: Run `buf lint` to verify proto validity

**Delete these lines**:
```protobuf
// List of git worktrees
repeated string worktrees = 4;  // DELETE BOTH LINES
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Proto is valid | buf lint | No errors | N/A |

#### Task 2.2: Regenerate protobuf code

- **File**: Generated files in `/Users/jayce/team-attention/cops/shared/gen/grpcstub/`
- **Change**: Run `cd idl/protobuf && buf generate`
- **Reason**: Regenerate Go code to match updated proto definitions
- **Test**: Run `go build ./...` from workspace root

**Command**:
```bash
cd idl/protobuf && buf generate
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Code generation succeeds | buf generate | Exit code 0 | N/A |
| All modules compile | go build ./... | No errors | N/A |

### Phase 3: Fix RegisteredAt Timestamp

**Goal**: Set `registeredAt` field when creating new projects

**Dependencies**: None (can be done in parallel with Phase 1-2)

#### Task 3.1: Add RegisteredAt to new project document

- **File**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go:92-96`
- **Change**: Add `mongoschema.ProjectRegisteredAtField: time.Now()` to the `newDoc` bson.M map
- **Reason**: New projects should have creation timestamp recorded
- **Test**: Create new project and verify RegisteredAt is set

**Before**:
```go
newDoc := bson.M{
    mongoschema.ProjectRemoteURLField:    remoteURL,
    mongoschema.ProjectNameField:         params.Name,
    mongoschema.ProjectIsGitProjectField: params.IsGitProject,
}
```

**After**:
```go
newDoc := bson.M{
    mongoschema.ProjectRemoteURLField:    remoteURL,
    mongoschema.ProjectNameField:         params.Name,
    mongoschema.ProjectIsGitProjectField: params.IsGitProject,
    mongoschema.ProjectRegisteredAtField: time.Now(),
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| New project has timestamp | Create project via FindOrCreate | RegisteredAt != zero value | Happy path |
| Existing project unchanged | FindOrCreate with existing URL | Original document unmodified | Existing project branch |

#### Task 3.2: Verify time import exists

- **File**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`
- **Change**: Verify `time` package is imported (add if missing)
- **Reason**: `time.Now()` requires the time package
- **Test**: File compiles without error

**Verify import exists**:
```go
import (
    // ... existing imports
    "time"  // Ensure this exists
)
```

### Phase 4: Add ContentBlock Types to Protobuf and Update gRPC Converter

**Goal**: Add structured ContentBlock types to protobuf so block data is preserved with full type information in gRPC responses

**Dependencies**: Must run `buf generate` after Phase 2 completes (can share the regeneration step)

**Context**:
- MongoDB serialization/deserialization is already correct
- The current protobuf `Message` only has `string content` field (field 5)
- Domain model has 4 block types: `text`, `tool_use`, `tool_result`, `thinking`
- We need to add these as proper protobuf types to preserve structure

#### Task 4.1: Add ContentBlock types to aggregation.proto

- **File**: `/Users/jayce/team-attention/cops/idl/protobuf/aggregation/v1/aggregation.proto`
- **Change**: Add ContentBlock message types and update Message to include them
- **Reason**: Preserve block structure in gRPC responses instead of losing type information
- **Test**: Run `buf lint` to verify proto validity

**Add these message definitions before the existing `Message` message (around line 28)**:

```protobuf
// ContentBlockType represents the type of content block.
enum ContentBlockType {
  CONTENT_BLOCK_TYPE_UNSPECIFIED = 0;
  CONTENT_BLOCK_TYPE_TEXT = 1;
  CONTENT_BLOCK_TYPE_TOOL_USE = 2;
  CONTENT_BLOCK_TYPE_TOOL_RESULT = 3;
  CONTENT_BLOCK_TYPE_THINKING = 4;
}

// TextContentBlock represents a text content block.
message TextContentBlock {
  string text = 1;
}

// ToolUseContentBlock represents a tool use content block.
message ToolUseContentBlock {
  string id = 1;
  string name = 2;
  string input_json = 3;  // JSON string for flexible input structure
}

// ToolResultContentBlock represents a tool result content block.
message ToolResultContentBlock {
  string tool_use_id = 1;
  string content = 2;
  bool is_error = 3;
}

// ThinkingContentBlock represents a thinking content block.
message ThinkingContentBlock {
  string thinking = 1;
  string signature = 2;
}

// ContentBlock represents a polymorphic content block using oneof.
message ContentBlock {
  ContentBlockType type = 1;
  oneof block {
    TextContentBlock text = 2;
    ToolUseContentBlock tool_use = 3;
    ToolResultContentBlock tool_result = 4;
    ThinkingContentBlock thinking = 5;
  }
}
```

**Modify the existing `Message` message to rename content to text and add content_blocks field**:

```protobuf
// Message contains the role and content of a session message.
message Message {
  string id = 1;
  string type = 2;
  string role = 3;
  string model = 4;
  string text = 5;                           // RENAME from 'content' to match domain model Text field
  string stop_reason = 6;
  string stop_sequence = 7;
  Usage usage = 8;
  repeated ContentBlock content_blocks = 9;  // ADD THIS FIELD for block content (matches domain model Blocks field)
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Proto compiles | buf lint | No errors | N/A |
| Proto generates | buf generate | Exit code 0 | N/A |

#### Task 4.2: Regenerate protobuf code

- **File**: Generated files in `/Users/jayce/team-attention/cops/shared/gen/grpcstub/aggregation/v1/`
- **Change**: Run `cd idl/protobuf && buf generate`
- **Reason**: Generate Go types for new ContentBlock messages
- **Test**: Run `go build ./...` from workspace root

**Note**: This can be combined with Task 2.2 if running phases sequentially.

#### Task 4.3: Update gRPC converter to populate content_blocks

- **File**: `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go:134-136`
- **Change**: Modify `convertMessage` function to populate `ContentBlocks` field when `IsBlocks` is true
- **Reason**: Map domain ContentBlock types to protobuf ContentBlock types
- **Test**: Call GetSession API with session containing blocks, verify content_blocks field is populated

**Before**:
```go
if m.Content != nil && !m.Content.IsBlocks && m.Content.Text != nil {
    msg.Content = *m.Content.Text
}
```

**After**:
```go
if m.Content != nil {
    if m.Content.IsBlocks {
        msg.ContentBlocks = convertContentBlocks(m.Content.Blocks)
    } else if m.Content.Text != nil {
        msg.Text = *m.Content.Text  // CHANGED: msg.Content -> msg.Text to match renamed proto field
    }
}
```

**Add new helper function `convertContentBlocks`**:

```go
// convertContentBlocks converts domain ContentBlocks to protobuf ContentBlocks.
func convertContentBlocks(blocks []shareddomain.ContentBlock) []*aggregationv1.ContentBlock {
    if len(blocks) == 0 {
        return nil
    }

    result := make([]*aggregationv1.ContentBlock, 0, len(blocks))
    for _, block := range blocks {
        pb := convertContentBlock(block)
        if pb != nil {
            result = append(result, pb)
        }
    }
    return result
}

// convertContentBlock converts a single domain ContentBlock to protobuf.
func convertContentBlock(block shareddomain.ContentBlock) *aggregationv1.ContentBlock {
    switch b := block.(type) {
    case *shareddomain.TextContentBlock:
        return &aggregationv1.ContentBlock{
            Type: aggregationv1.ContentBlockType_CONTENT_BLOCK_TYPE_TEXT,
            Block: &aggregationv1.ContentBlock_Text{
                Text: &aggregationv1.TextContentBlock{
                    Text: b.Text,
                },
            },
        }
    case *shareddomain.ToolUseContentBlock:
        inputJSON := ""
        if b.Input != nil {
            if bytes, err := sonic.Marshal(b.Input); err == nil {
                inputJSON = string(bytes)
            }
        }
        return &aggregationv1.ContentBlock{
            Type: aggregationv1.ContentBlockType_CONTENT_BLOCK_TYPE_TOOL_USE,
            Block: &aggregationv1.ContentBlock_ToolUse{
                ToolUse: &aggregationv1.ToolUseContentBlock{
                    Id:        b.ID,
                    Name:      b.Name,
                    InputJson: inputJSON,
                },
            },
        }
    case *shareddomain.ToolResultContentBlock:
        return &aggregationv1.ContentBlock{
            Type: aggregationv1.ContentBlockType_CONTENT_BLOCK_TYPE_TOOL_RESULT,
            Block: &aggregationv1.ContentBlock_ToolResult{
                ToolResult: &aggregationv1.ToolResultContentBlock{
                    ToolUseId: b.ToolUseID,
                    Content:   b.Content,
                    IsError:   b.IsError,
                },
            },
        }
    case *shareddomain.ThinkingContentBlock:
        return &aggregationv1.ContentBlock{
            Type: aggregationv1.ContentBlockType_CONTENT_BLOCK_TYPE_THINKING,
            Block: &aggregationv1.ContentBlock_Thinking{
                Thinking: &aggregationv1.ThinkingContentBlock{
                    Thinking:  b.Thinking,
                    Signature: b.Signature,
                },
            },
        }
    default:
        return nil
    }
}
```

**Add import**:
```go
import (
    // ... existing imports
    "github.com/bytedance/sonic"
)
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Text content | Content.IsBlocks=false, Text="hello" | msg.Text = "hello", ContentBlocks = nil | Text branch |
| Text block | Content.IsBlocks=true, TextContentBlock | msg.ContentBlocks[0].Type = TEXT | Text block |
| Tool use block | Content.IsBlocks=true, ToolUseContentBlock | msg.ContentBlocks[0].Type = TOOL_USE, InputJson populated | Tool use block |
| Tool result block | Content.IsBlocks=true, ToolResultContentBlock | msg.ContentBlocks[0].Type = TOOL_RESULT | Tool result block |
| Thinking block | Content.IsBlocks=true, ThinkingContentBlock | msg.ContentBlocks[0].Type = THINKING | Thinking block |
| Mixed blocks | Content.IsBlocks=true, multiple block types | All blocks converted correctly | Multiple blocks |
| Nil content | Content = nil | msg.Text = "", ContentBlocks = nil | Nil check |

## Execution Order

1. **Task 1.1** - Remove domain model fields (shared/domain/project.go) - No dependencies
2. **Task 1.2** - Remove MongoDB constants (shared/domain/mongoschema/project.go) - Depends on 1.1
3. **Task 1.3** - Update dashboard repository (api/.../dashboard_repo.go) - Depends on 1.2
4. **Task 1.4** - Update gRPC converter to remove Worktrees (api/.../converter.go) - Depends on 1.1
5. **Task 2.1** - Remove worktrees from dashboard.proto - Depends on 1.4
6. **Task 4.1** - Add ContentBlock types to aggregation.proto - No dependencies (can run with 2.1)
7. **Task 2.2 + 4.2** - Regenerate ALL protobuf code (single `buf generate`) - Depends on 2.1 and 4.1
8. **Task 4.3** - Update gRPC converter to populate content_blocks - Depends on 4.2
9. **Task 3.1** - Add RegisteredAt to FindOrCreate - No dependencies (parallel with all above)
10. **Task 3.2** - Add time import if missing - Part of 3.1

**Parallelization opportunities**:
- Phase 3 (Tasks 3.1-3.2) can run in parallel with all other phases
- Task 2.1 and Task 4.1 (both proto changes) can run in parallel, then single buf generate
- Task 4.3 must wait for buf generate to complete

## Testing Strategy

### Unit Tests

1. **Domain model tests**: Run existing tests in `shared/domain/message_test.go` - should pass unchanged
2. **Build verification**: Run `go build ./...` after each phase to catch compilation errors

```bash
# After Phase 1
go build ./shared/...

# After Phase 2
go build ./...

# After Phases 3-4 (both modify api module)
go build ./api/...
```

### Integration Tests

1. **MessageContent round-trip test**: Verify blocks survive marshal -> MongoDB -> unmarshal cycle
   - Use test data from `shared/domain/log_data.jsonl`
   - Lines 1, 5, 7, 8 contain various block types

2. **Project creation test**: Verify new projects have RegisteredAt set
   - Create project via daemon registration flow
   - Query project via dashboard API
   - Verify RegisteredAt is within last minute

### Manual Verification

1. **Dashboard UI**: After deployment, verify:
   - Project list displays correctly without ClaudeDir/Worktrees columns
   - Project detail page shows creation date (not "January 1, Year 1")
   - Session detail shows message content (including blocks as JSON)

2. **gRPC API**: Use `grpcurl` or similar to call:
   - `GetProject` - verify no `worktrees` field in response
   - `GetSession` - verify `content` field includes block data as JSON

## Rollback Plan

### If Phase 1-2 fails (Remove Project Fields)

1. Revert changes to:
   - `shared/domain/project.go`
   - `shared/domain/mongoschema/project.go`
   - `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go`
   - `api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go`
   - `idl/protobuf/dashboard/v1/dashboard.proto`
2. Regenerate protobuf: `cd idl/protobuf && buf generate`
3. Rebuild: `go build ./...`

### If Phase 3 fails (RegisteredAt)

1. Revert `api/internal/service/project/outbound/repository/mongodb/project_repo.go`
2. Existing projects unaffected (zero-value behavior continues)

### If Phase 4 fails (MessageContent Blocks)

1. Revert `idl/protobuf/aggregation/v1/aggregation.proto` (remove ContentBlock types and content_blocks field)
2. Revert `api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go`
3. Regenerate protobuf: `cd idl/protobuf && buf generate`
4. Rebuild: `go build ./...`
5. Blocks will be dropped from gRPC responses (back to original behavior)
6. MongoDB storage unaffected

## Acceptance Criteria Mapping

| Criterion | Implementation Task |
| --------- | ------------------- |
| AC1: Project no longer contains ClaudeDir | Task 1.1 |
| AC2: Project no longer contains Worktrees | Task 1.1 |
| AC3: DB queries/API responses updated | Tasks 1.2, 1.3, 1.4, 2.1, 2.2 |
| AC4: RegisteredAt set on new projects | Task 3.1 |
| AC5: Existing projects handled gracefully | No change needed (MongoDB returns zero value, UI should handle) |
| AC6: MessageContent.Blocks persisted | No change needed (MongoDB serialization already correct) |
| AC7: MessageContent.Blocks reconstructed | No change needed (MongoDB deserialization already correct) |
| AC8: All existing tests pass | Testing Strategy - run after each phase |
| AC9: Integration test for Blocks | Tasks 4.1-4.3 enable structured blocks in gRPC responses; Testing Strategy verifies with log_data.jsonl |

## Notes for Execute Agent

1. **Order matters**: Complete Phase 1 before Phase 2; complete proto changes (Tasks 2.1, 4.1) before buf generate
2. **Single buf generate**: Run `cd idl/protobuf && buf generate` once after ALL proto changes (Tasks 2.1 and 4.1)
3. **Add time import if missing**: When adding `time.Now()` in Task 3.1, add `"time"` to imports if not present
4. **Add sonic import in converter**: When adding `convertContentBlock` in Task 4.3, add `"github.com/bytedance/sonic"` to imports for serializing ToolUse.Input
5. **No ProjectWithWorktrees changes**: The `ProjectWithWorktrees` type in `shared/domain/project.go:23-28` should NOT be modified - it's used locally by CLI and embeds Project, which will automatically lose the removed fields
6. **Watch for compilation order**: Build `shared` module first, then `api` module
7. **Test data location**: Block content test data is in `shared/domain/log_data.jsonl`
8. **MongoDB code is correct**: Do NOT modify `adapter.go` or `dashboard_repo.go` for MessageContent - only protobuf and gRPC converter need updating
9. **Protobuf oneof pattern**: The `ContentBlock.Block` uses protobuf `oneof` - generated Go code will have `ContentBlock_Text`, `ContentBlock_ToolUse`, etc. wrapper types
