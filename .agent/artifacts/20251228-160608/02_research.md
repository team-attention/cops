# Research Findings

## Mode
General Research

## Request Summary
Fix three data model and persistence issues: (1) Remove Project fields (ClaudeDir, Worktrees) that should be tracked at Session level or locally, (2) Fix RegisteredAt timestamp not being set on new Projects, and (3) Fix MessageContent Block data not being persisted to MongoDB.

---

## Issue 1: Remove Project Fields (ClaudeDir, Worktrees)

### Current State

#### Domain Model Definition
**File**: `/Users/jayce/team-attention/cops/shared/domain/project.go:15-21`

```go
type Project struct {
    ProjectAbstract
    IsGitProject bool      `json:"gitProject"`          // true if git repo, false otherwise
    ClaudeDir    string    `json:"claudeDir"`           // Claude project directory path
    Worktrees    []string  `json:"worktrees,omitempty"` // Worktree paths for git projects
    RegisteredAt time.Time `json:"registeredAt"`        // When the project was registered
}
```

#### MongoDB Schema Constants
**File**: `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go:17,20`

```go
const (
    // ...
    ProjectClaudeDirField    = "claudeDir"
    // ...
    ProjectWorktreesField    = "worktrees"
    // ...
)
```

#### ProjectWithWorktrees Type
**File**: `/Users/jayce/team-attention/cops/shared/domain/project.go:23-28`

The `ProjectWithWorktrees` struct embeds `Project` and adds a dynamically discovered `Worktrees` field. This is used for local display (CLI) and is separate from the `Project.Worktrees` field.

### Dependencies

#### 1. Dashboard Repository (API Server)
**File**: `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go:278-279`

```go
detail := &repository.ProjectDetail{
    Project: shareddomain.Project{
        // ...
        ClaudeDir:    mongoutil.Get[string](doc, mongoschema.ProjectClaudeDirField),
        Worktrees:    mongoutil.GetSlice[string](doc, mongoschema.ProjectWorktreesField),
        RegisteredAt: mongoutil.Get[time.Time](doc, mongoschema.ProjectRegisteredAtField),
    },
}
```

#### 2. gRPC Converter
**File**: `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go:39`

```go
func toProtoProjectDetail(p *repository.ProjectDetail) *dashboardv1.ProjectDetail {
    return &dashboardv1.ProjectDetail{
        // ...
        Worktrees:    p.Project.Worktrees,  // References domain.Project.Worktrees
        // ...
    }
}
```

#### 3. Protobuf Definition
**File**: `/Users/jayce/team-attention/cops/idl/protobuf/dashboard/v1/dashboard.proto:81-82`

```protobuf
message ProjectDetail {
    // ...
    repeated string worktrees = 4;  // List of git worktrees
    // ...
}
```

#### 4. CLI Tracking Service (Local Use Only)
**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go:155`

The CLI sets `ClaudeDir` when creating a Project locally. This is used by the daemon to watch for JSONL files but should NOT be persisted to the server.

#### 5. Daemon Config Watcher (Local Use Only)
**File**: `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/configwatcher_service.go:117`

The daemon reads `ClaudeDir` from the local project config and uses it to determine which directories to watch for JSONL log files.

### Impact Analysis

| Component | Impact | Action Required |
|-----------|--------|-----------------|
| `shared/domain/project.go` | Remove `ClaudeDir` and `Worktrees` fields from `Project` struct | HIGH - Core change |
| `shared/domain/mongoschema/project.go` | Remove `ProjectClaudeDirField` and `ProjectWorktreesField` constants | HIGH - Remove unused constants |
| `api/.../dashboard_repo.go` | Remove references to `ClaudeDir` and `Worktrees` fields | HIGH - Won't compile if fields removed |
| `api/.../converter.go` | Remove `Worktrees` mapping | HIGH - Won't compile if field removed |
| `idl/protobuf/dashboard/v1/dashboard.proto` | Remove `worktrees` field from `ProjectDetail` | MEDIUM - Requires buf generate |
| `shared/gen/grpcstub/dashboard/v1/dashboard.pb.go` | Auto-regenerated | N/A - Generated |
| CLI tracking service | Continue to use `ClaudeDir` locally (do NOT persist to server) | LOW - Local only |
| Daemon config watcher | Continue to use `ClaudeDir` from local config | LOW - Local only |

### Key Insight: ClaudeDir and Worktrees Are Local State

- **ClaudeDir**: Computed locally as `~/.claude/projects/{encoded-path}`. Different on each machine.
- **Worktrees**: Discovered dynamically via `git worktree list`. Changes as developers create/delete worktrees.

Neither should be stored in the central MongoDB database because:
1. They are machine-specific
2. They change frequently
3. The daemon already discovers them locally

---

## Issue 2: RegisteredAt Timestamp

### Current Implementation

**File**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go:92-96`

```go
// Not found, create new document
newDoc := bson.M{
    mongoschema.ProjectRemoteURLField:    remoteURL,
    mongoschema.ProjectNameField:         params.Name,
    mongoschema.ProjectIsGitProjectField: params.IsGitProject,
}
```

### Root Cause

The `RegisteredAt` field is **NOT included** in the `newDoc` when creating a new project. This results in:
1. MongoDB document has no `registeredAt` field
2. When read back, `mongoutil.Get[time.Time](doc, mongoschema.ProjectRegisteredAtField)` returns Go's zero value for `time.Time`
3. Zero value of `time.Time` is `0001-01-01 00:00:00 +0000 UTC`
4. This displays as "January 1, Year 1" in the web UI

### Fix Location

**File**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`

Add `mongoschema.ProjectRegisteredAtField: time.Now()` to the `newDoc` bson.M on line 92-96.

### Similar Patterns (for reference)

Looking at how timestamps are handled elsewhere:
- Session records include `timestamp` field in their documents (line 73 in adapter.go)
- The `mongoschema.ProjectRegisteredAtField` constant already exists (line 18 in project.go)

### Verification Points

1. Existing projects in DB will still have zero value - this is acceptable per requirements
2. New projects should have correct `registeredAt` timestamp
3. Dashboard should display the timestamp correctly (or show "Unknown" for legacy projects)

---

## Issue 3: MessageContent Blocks Persistence

### Domain Model Analysis

**File**: `/Users/jayce/team-attention/cops/shared/domain/message.go:8-94`

The `MessageContent` struct handles polymorphic content:

```go
type MessageContent struct {
    Text     *string        // When content is string (nil when IsBlocks=true)
    Blocks   []ContentBlock // When content is []ContentBlock (nil when IsBlocks=false)
    IsBlocks bool           // Internal discriminator
}
```

#### Custom Marshaling
**`MarshalJSON()` (lines 81-94)**:
- If `IsBlocks` is true, marshals `c.Blocks` as array
- If `IsBlocks` is false, marshals `*c.Text` as string
- Returns `null` for uninitialized content

#### Custom Unmarshaling
**`UnmarshalJSON()` (lines 17-79)**:
- First tries to parse as string
- If fails, parses as array of content blocks
- Handles 4 block types: text, tool_use, tool_result, thinking

### Content Block Types

**File**: `/Users/jayce/team-attention/cops/shared/domain/content_block.go`

| Type | Struct | Fields |
|------|--------|--------|
| `text` | `TextContentBlock` | Type, Text |
| `tool_use` | `ToolUseContentBlock` | Type, ID, Name, Input (map) |
| `tool_result` | `ToolResultContentBlock` | Type, ToolUseID, Content, IsError |
| `thinking` | `ThinkingContentBlock` | Type, Thinking, Signature |

### Serialization Flow (Write Path)

**File**: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go:106-117`

```go
if msg.Content != nil {
    contentBytes, err := sonic.Marshal(msg.Content)
    if err != nil {
        // Log warning but continue
        slog.Warn("failed to serialize message content", ...)
    } else {
        doc[mongoschema.SessionRecordMessageContentField] = string(contentBytes)
    }
}
```

This correctly:
1. Uses `sonic.Marshal()` which calls custom `MarshalJSON()`
2. Stores as JSON string in `messageContent` field

### Deserialization Flow (Read Path)

**File**: `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go:466-478`

```go
if content := mongoutil.Get[string](doc, mongoschema.SessionRecordMessageContentField); content != "" {
    var mc shareddomain.MessageContent
    if err := sonic.Unmarshal([]byte(content), &mc); err != nil {
        // Fallback: treat as legacy plain text (backward compatibility)
        msg.Content = &shareddomain.MessageContent{
            Text:     lo.ToPtr(content),
            IsBlocks: false,
        }
    } else {
        msg.Content = &mc
    }
}
```

This correctly:
1. Uses `sonic.Unmarshal()` which calls custom `UnmarshalJSON()`
2. Falls back to plain text for legacy data

### Test Coverage

**File**: `/Users/jayce/team-attention/cops/shared/domain/message_test.go`

Comprehensive test coverage exists (633 lines):
- String content parsing
- Block array parsing (text, tool_use, tool_result, thinking)
- Real JSONL data integration tests (8 records from `log_data.jsonl`)
- Round-trip serialization tests

### Sonic vs encoding/json Compatibility

The codebase uses `bytedance/sonic` for JSON operations. According to research:
- `sonic.Marshal()` correctly calls custom `MarshalJSON()` methods
- `sonic.Unmarshal()` correctly calls custom `UnmarshalJSON()` methods
- `MessageContent.UnmarshalJSON()` uses `encoding/json` internally, which is compatible

### Root Cause Hypothesis

Based on the code analysis, **the serialization appears to be implemented correctly**. The `sonic.Marshal(msg.Content)` on line 107 of adapter.go should correctly serialize block content.

Possible issues to verify:
1. **Data already in DB may be legacy**: Records saved before the fix may have plain text only
2. **IsBlocks discriminator**: The `IsBlocks` field is internal and NOT persisted in JSON. It's reconstructed during unmarshal based on content type.

### Current State Verification

The test data in `log_data.jsonl` contains examples with:
- Line 1: text block in array `[{"type":"text","text":"..."}]`
- Line 2: plain string content
- Lines 3-4: string content with XML/HTML
- Line 5: thinking block `[{"type":"thinking","thinking":"...","signature":"..."}]`
- Line 6: text block
- Line 7: tool_use block `[{"type":"tool_use","id":"...","name":"Skill","input":{...}}]`
- Line 8: tool_result block `[{"type":"tool_result","tool_use_id":"...","content":"..."}]`

---

## Files to Read Before Planning

| File | Reason |
|------|--------|
| `/Users/jayce/team-attention/cops/shared/domain/project.go` | Contains Project struct to modify |
| `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go` | MongoDB field constants to remove |
| `/Users/jayce/team-attention/cops/api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go:270-282` | GetProject method referencing ClaudeDir/Worktrees |
| `/Users/jayce/team-attention/cops/api/internal/service/dashboard/inbound/grpc/connectrpc/converter.go:34-45` | toProtoProjectDetail referencing Worktrees |
| `/Users/jayce/team-attention/cops/idl/protobuf/dashboard/v1/dashboard.proto:70-95` | ProjectDetail proto definition |
| `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go:92-96` | FindOrCreate missing RegisteredAt |
| `/Users/jayce/team-attention/cops/api/internal/service/aggregation/outbound/repository/mongodb/adapter.go:106-117` | MessageContent serialization |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-struct.md` | Struct definition rules |

---

## Implementation Recommendations

### Issue 1: Remove Project Fields

1. **Remove from domain model** (`shared/domain/project.go`):
   - Delete `ClaudeDir string` field
   - Delete `Worktrees []string` field

2. **Remove MongoDB constants** (`shared/domain/mongoschema/project.go`):
   - Delete `ProjectClaudeDirField` constant
   - Delete `ProjectWorktreesField` constant

3. **Update dashboard repo** (`api/.../dashboard_repo.go`):
   - Remove lines setting `ClaudeDir` and `Worktrees` in GetProject()

4. **Update gRPC converter** (`api/.../converter.go`):
   - Remove `Worktrees: p.Project.Worktrees` line

5. **Update protobuf** (`idl/protobuf/dashboard/v1/dashboard.proto`):
   - Remove `repeated string worktrees = 4;` from ProjectDetail
   - Run `cd idl/protobuf && buf generate`

6. **Keep CLI/Daemon local usage**:
   - CLI can still compute `ClaudeDir` locally for its own use
   - Daemon continues using `ClaudeDir` from local config
   - These are runtime values, not persisted to server

### Issue 2: Fix RegisteredAt Timestamp

1. **Modify FindOrCreate** (`api/.../project_repo.go`):
   ```go
   newDoc := bson.M{
       mongoschema.ProjectRemoteURLField:    remoteURL,
       mongoschema.ProjectNameField:         params.Name,
       mongoschema.ProjectIsGitProjectField: params.IsGitProject,
       mongoschema.ProjectRegisteredAtField: time.Now(),  // ADD THIS LINE
   }
   ```

2. **Add import** for `time` package if not already imported

### Issue 3: MessageContent Blocks Persistence

The current implementation appears correct. However, verify by:

1. **Add integration test** in `daemon/internal/service/logwatcher/log_service_test.go`:
   - Read test JSONL with block content
   - Verify blocks are parsed
   - Simulate round-trip through `sonic.Marshal` / `sonic.Unmarshal`

2. **Verify gRPC response**:
   - The `convertMessage()` function in converter.go currently only handles text content (line 134-136)
   - This may need updating to return block content in protobuf format

3. **Check protobuf Message definition**:
   - The `aggregation.v1.Message` proto may need fields for block content if they should be exposed via gRPC

---

## Technical Constraints

- **Backward compatibility**: Existing session records must continue to work (GetSession handles both legacy and new)
- **Zero downtime**: API and daemon can be deployed independently
- **Go workspace structure**: Changes span shared, api modules
- **MongoDB flexible schema**: No migration needed for field removal

---

## Additional Information for Planning

1. **CLI still needs ProjectWithWorktrees**: Keep this type for local `cops list` display, but it should NOT be the server-stored `Project.Worktrees`

2. **Generated protobuf must be regenerated**: After removing `worktrees` from proto, run `cd idl/protobuf && buf generate`

3. **Tests may need updating**: Any tests asserting on `ClaudeDir` or `Worktrees` fields need to be updated

4. **Consider adding test for RegisteredAt**: After fix, add test verifying new projects have non-zero RegisteredAt
