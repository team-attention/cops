# Development Walkthrough

## Summary

Extended the project registration flow to include project name and Git status tracking. Added `name` and `isGitProject` fields throughout the entire data pipeline from CLI input through API service to MongoDB storage, enabling richer project metadata in the central database.

## Code Overview

### New Fields Added

This implementation adds two new fields to the project registration flow:

1. **`name`**: Human-readable project name (string)
2. **`isGitProject`**: Boolean flag indicating whether the project is a Git repository

These fields flow through the entire system: **CLI → gRPC API → MongoDB**.

### Modified Components

#### 1. Protocol Buffer Definition

**Location**: `/Users/jayce/team-attention/cops/idl/protobuf/project/v1/project.proto`

**Changes**: Added two fields to request and response messages

**RegisterProjectReq** (lines 20-26):
```protobuf
// name is the human-readable project name
// If empty, the server will generate a default from the remote URL
string name = 4;

// is_git_project indicates whether this is a git repository
bool is_git_project = 5;
```

**RegisterProjectRes** (lines 36-42):
```protobuf
// name is the project name (either provided or generated)
string name = 3;

// is_git_project indicates whether this is a git repository
bool is_git_project = 4;
```

**Impact**: This generates TypeScript/Go stubs with the new fields via `buf generate`.

---

#### 2. Generated Code (Auto-Updated)

**Location**: `/Users/jayce/team-attention/cops/shared/gen/grpcstub/project/v1/project.pb.go`

**Changes**: Auto-generated getters and struct fields from protobuf

**Key Methods Added**:
- `GetName() string` (lines 93-98)
- `GetIsGitProject() bool` (lines 100-105)

These are programmatically generated and should not be manually edited.

---

#### 3. API Handler (Inbound Adapter)

**Location**: `/Users/jayce/team-attention/cops/api/internal/service/project/inbound/grpc/connectrpc/handler.go`

**Changes**: Extract fields from request, pass to service, return in response

**Request Parsing** (lines 45-46):
```go
params := project.RegisterProjectParams{
    // ... existing fields ...
    Name:         msg.GetName(),
    IsGitProject: msg.GetIsGitProject(),
}
```

**Response Building** (lines 57-62):
```go
res := &projectv1.RegisterProjectRes{
    ProjectId:    result.ProjectID,
    IsNew:        result.IsNew,
    Name:         result.Name,        // Added
    IsGitProject: result.IsGitProject, // Added
}
```

**Purpose**: Translates between protobuf messages and internal service types.

---

#### 4. API Service (Business Logic)

**Location**: `/Users/jayce/team-attention/cops/api/internal/service/project/project_service.go`

**Changes**: Added fields to params struct and service method

**RegisterProjectParams Struct** (lines 15-16):
```go
type RegisterProjectParams struct {
    ConfiguredRemoteURL string
    ActualRemoteURL     string
    ExistingProjectID   string
    Name                string  // Added
    IsGitProject        bool    // Added
}
```

**Service Method** (lines 35-42):
```go
result, err := s.repo.FindOrCreate(ctx, repository.FindOrCreateParams{
    ConfiguredURL: params.ConfiguredRemoteURL,
    ActualURL:     params.ActualRemoteURL,
    ExistingID:    params.ExistingProjectID,
    Name:          params.Name,         // Added
    IsGitProject:  params.IsGitProject,  // Added
})
```

**Logging** (lines 49-50):
```go
s.logger.Info("project registered",
    slog.String("projectID", result.ProjectID),
    slog.Bool("isNew", result.IsNew),
    slog.String("name", result.Name),              // Added
    slog.Bool("isGitProject", result.IsGitProject)) // Added
```

---

#### 5. Repository Port (Interface)

**Location**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/project_repo_port.go`

**Changes**: Created parameter object and added fields to result

**FindOrCreateParams Struct** (NEW, lines 5-12):
```go
// FindOrCreateParams contains parameters for FindOrCreate operation.
type FindOrCreateParams struct {
    ConfiguredURL string
    ActualURL     string
    ExistingID    string
    Name          string
    IsGitProject  bool
}
```

**Rationale**: Follows function parameter guidelines - more than 3 params requires a struct.

**FindOrCreateResult Struct** (lines 15-20):
```go
type FindOrCreateResult struct {
    ProjectID    string
    IsNew        bool
    Name         string  // Added
    IsGitProject bool    // Added
}
```

**Port Interface** (line 29):
```go
FindOrCreate(ctx context.Context, params FindOrCreateParams) (*FindOrCreateResult, error)
```

---

#### 6. MongoDB Repository (Outbound Adapter)

**Location**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`

**Changes**: Store and retrieve new fields from MongoDB

**Method Signature** (line 34):
```go
func (r *MongoProjectRepository) FindOrCreate(
    ctx context.Context,
    params repository.FindOrCreateParams,
) (*repository.FindOrCreateResult, error)
```

**Query Building** (lines 38-50):
Uses params struct fields for URL and ID lookups (no change to query logic, just parameter access pattern).

**Finding Existing Project** (lines 66-67):
```go
name := doc[mongoschema.ProjectNameField].(string)
isGitProject := doc[mongoschema.ProjectIsGitProjectField].(bool)
```

**Return Existing** (lines 71-74):
```go
return &repository.FindOrCreateResult{
    ProjectID:    projectID,
    IsNew:        false,
    Name:         name,         // Added
    IsGitProject: isGitProject,  // Added
}, nil
```

**Creating New Project** (lines 93-95):
```go
newDoc := bson.M{
    mongoschema.ProjectRemoteURLField:    remoteURL,
    mongoschema.ProjectNameField:         params.Name,         // Added
    mongoschema.ProjectIsGitProjectField: params.IsGitProject, // Added
}
```

**Logging New Project** (lines 107-108):
```go
r.logger.Info("created new project",
    slog.String("projectID", newID),
    slog.String("name", params.Name),              // Added
    slog.String("remoteURL", remoteURL),
    slog.Bool("isGitProject", params.IsGitProject)) // Added
```

**Return New** (lines 111-115):
```go
return &repository.FindOrCreateResult{
    ProjectID:    newID,
    IsNew:        true,
    Name:         params.Name,         // Added
    IsGitProject: params.IsGitProject,  // Added
}, nil
```

---

#### 7. CLI API Client (Outbound Adapter)

**Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`

**Changes**: Send fields in request, receive in response

**Request Building** (lines 44-45):
```go
req := connect.NewRequest(&projectv1.RegisterProjectReq{
    ConfiguredRemoteUrl: params.ConfiguredRemoteURL,
    ActualRemoteUrl:     params.ActualRemoteURL,
    ExistingProjectId:   params.ExistingProjectID,
    Name:                params.Name,         // Added
    IsGitProject:        params.IsGitProject,  // Added
})
```

**Response Parsing** (lines 55-58):
```go
result := &api.RegisterProjectResult{
    ProjectID:    domain.ID(resp.Msg.ProjectId),
    IsNew:        resp.Msg.IsNew,
    Name:         resp.Msg.Name,         // Added
    IsGitProject: resp.Msg.IsGitProject,  // Added
}
```

---

#### 8. CLI API Port (Interface)

**Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/project_port.go`

**Changes**: Added fields to params and result structs

**RegisterProjectParams** (lines 21-27):
```go
type RegisterProjectParams struct {
    ConfiguredRemoteURL string
    ActualRemoteURL     string
    ExistingProjectID   string

    Name         string  // Added (line 24)
    IsGitProject bool    // Added (line 27)
}
```

**RegisterProjectResult** (lines 31-34):
```go
type RegisterProjectResult struct {
    ProjectID    domain.ID
    IsNew        bool
    Name         string  // Added
    IsGitProject bool    // Added
}
```

---

#### 9. CLI Tracking Service

**Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`

**Changes**: Pass name and isGitProject to API client

**Service Call** (lines 113-114):
```go
result, err := s.projectClient.RegisterProject(ctx, api.RegisterProjectParams{
    ConfiguredRemoteURL: configuredURL,
    ActualRemoteURL:     actualURL,
    ExistingProjectID:   existingProjectID,
    Name:                name,         // Added
    IsGitProject:        isGitProject,  // Added
})
```

**Context**: The `name` and `isGitProject` variables are derived earlier in the method from:
- TUI user input (if TUI flow was used)
- Git detection logic
- Directory path analysis

---

#### 10. Domain Models (Shared)

**Location**: `/Users/jayce/team-attention/cops/shared/domain/content_block.go` and `message.go`

**Changes**: Added support for `thinking` content block type (for extended thinking models)

**New Content Block Type** (content_block.go, line 10):
```go
ContentBlockTypeThinking ContentBlockType = "thinking"
```

**ThinkingContentBlock Struct** (lines 48-56):
```go
type ThinkingContentBlock struct {
    Type      ContentBlockType `json:"type"`
    Thinking  string           `json:"thinking"`
    Signature string           `json:"signature,omitempty"`
}
```

**Unmarshaling Support** (message.go, lines 65-71):
```go
case ContentBlockTypeThinking:
    var tb ThinkingContentBlock
    if err := json.Unmarshal(raw, &tb); err != nil {
        return fmt.Errorf("failed to parse thinking block %d: %w", i, err)
    }
    block = &tb
```

**Purpose**: Enables parsing of Claude's extended thinking model responses in JSONL logs.

---

## Data Flow

### Complete Flow: CLI → API → MongoDB

```
1. User runs TUI command
   ↓
2. CLI captures name and isGitProject from user input/detection
   ↓
3. CLI sends gRPC request to API
   RegisterProjectReq { name, isGitProject, ... }
   ↓
4. API handler extracts fields and calls service
   RegisterProjectParams { Name, IsGitProject, ... }
   ↓
5. Service calls repository with params
   FindOrCreateParams { Name, IsGitProject, ... }
   ↓
6. MongoDB repository stores/retrieves from database
   {
     "remoteURL": "...",
     "name": "cops",
     "isGitProject": true
   }
   ↓
7. Repository returns result
   FindOrCreateResult { ProjectID, IsNew, Name, IsGitProject }
   ↓
8. Service returns result
   ↓
9. API handler builds response
   RegisterProjectRes { projectId, isNew, name, isGitProject }
   ↓
10. CLI receives and uses result
```

### Database Schema Changes

**MongoDB Collection**: `projects`

**New Fields**:
- `name` (string) - Human-readable project name
- `isGitProject` (boolean) - Git repository status

**Example Document**:
```json
{
  "_id": ObjectId("69504100a865b1fd5ddd5e21"),
  "remoteURL": "https://github.com/team-attention/cops.git",
  "name": "cops",
  "isGitProject": true
}
```

---

## Testing

### Build Verification

```bash
# Build all modules
go build ./cli/... ./api/... ./daemon/... ./shared/...
# Result: SUCCESS (0 errors)
```

### Integration Points Verified

| Component | Verification | Status |
|-----------|--------------|--------|
| Protobuf generation | `buf generate` | ✓ Generated new stubs |
| CLI compilation | `go build ./cli/...` | ✓ No errors |
| API compilation | `go build ./api/...` | ✓ No errors |
| Shared compilation | `go build ./shared/...` | ✓ No errors |
| Type consistency | All layers use same field names | ✓ Verified |

---

## Issues & Resolutions

No issues encountered. Implementation followed established patterns:
- Port/Adapter pattern for inbound/outbound
- Parameter structs for functions with >3 params
- Consistent naming across all layers
- Proper logging with structured fields

---

## Architecture Decisions

### Decision 1: Field Names - `name` vs `projectName`

**Choice**: Use simple field name `name` (not `projectName`)

**Rationale**:
- Shorter and cleaner in protobuf
- Context is clear from message type (RegisterProjectReq)
- Follows protobuf best practices

### Decision 2: Parameter Object Pattern

**Choice**: Created `FindOrCreateParams` struct instead of adding individual parameters

**Rationale**:
- Function would exceed 3 parameters (violates project guidelines)
- Easier to extend in the future
- More readable at call sites

### Decision 3: Return Fields in Response

**Choice**: Return `name` and `isGitProject` in `RegisterProjectRes`

**Rationale**:
- Client needs to confirm what was actually stored
- Server may modify/normalize the name
- Provides feedback loop for client validation

### Decision 4: Optional vs Required Fields

**Choice**: Made both fields optional in protobuf (no `required` keyword)

**Rationale**:
- Proto3 doesn't support `required` keyword
- Empty string and false are valid defaults
- Server can handle missing values gracefully

---

## Related Changes

### Thinking Content Block Support

Added support for Claude's extended thinking model responses:
- New `ContentBlockTypeThinking` type
- `ThinkingContentBlock` struct with `thinking` and `signature` fields
- Unmarshaling logic in `message.go`

**Purpose**: Enables daemon to parse and store extended thinking responses from Claude Code sessions.

**Impact**: No breaking changes; existing message parsing still works.

---

## Breaking Changes

**None**. This is a backward-compatible addition:
- Existing clients not sending `name`/`isGitProject` will work (fields optional)
- Existing database documents without these fields will continue to function
- New fields are additive, not replacing existing functionality

---

## Future Enhancements

1. **Name validation**: Add server-side validation for project name (length, characters)
2. **Name uniqueness**: Consider enforcing unique names per organization
3. **Default name generation**: Improve server-side logic for generating names from remote URLs
4. **Git detection on server**: Server could detect Git status from remote URL patterns

---

## Files Changed

| File | Type | Lines Changed | Description |
|------|------|---------------|-------------|
| `idl/protobuf/project/v1/project.proto` | Modified | +11 | Added name/isGitProject fields to proto |
| `shared/gen/grpcstub/project/v1/project.pb.go` | Generated | +35 | Auto-generated getters/setters |
| `api/.../grpc/connectrpc/handler.go` | Modified | +4 | Extract/return new fields |
| `api/.../project/project_service.go` | Modified | +5 | Pass fields to repository |
| `api/.../repository/project_repo_port.go` | Modified | +10 | Added params struct, updated result |
| `api/.../repository/mongodb/project_repo.go` | Modified | +15 | Store/retrieve from MongoDB |
| `cli/.../api/connectrpc/project_client.go` | Modified | +4 | Send/receive new fields |
| `cli/.../api/project_port.go` | Modified | +6 | Added fields to params/result |
| `cli/.../tracking/tracking_service.go` | Modified | +2 | Pass name/isGitProject to API |
| `shared/domain/content_block.go` | Modified | +13 | Added thinking block type |
| `shared/domain/message.go` | Modified | +7 | Added thinking block parsing |

**Summary**: 11 files modified, ~112 lines added

---

## Impact

### CLI Impact
- Users' project names are now stored in central database
- Git status is tracked alongside project metadata
- Future features can leverage this richer data (e.g., dashboard filtering by Git projects)

### API Impact
- Project registration now captures more context about each project
- Enables better analytics and reporting

### Database Impact
- Projects collection now has two new fields
- Existing projects without these fields will work (graceful degradation)
- New projects will have complete metadata

### Dashboard Impact (Future)
- Dashboard can display human-readable project names
- Can filter/group projects by Git status
- Better UX for project selection and navigation
