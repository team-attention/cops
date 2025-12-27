# Research Report

## Mode
General Research

## Request Summary

The CLI collects project name and git status during `cops add`, but the API server only stores `remoteUrl` in MongoDB. This task adds `name` and `isGitProject` fields to the protobuf schema, CLI API client, API server, and MongoDB repository to ensure all collected project metadata is properly persisted.

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File                                                                                         | Reason                                                              |
| -------------------------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| `/Users/jayce/team-attention/cops/idl/protobuf/project/v1/project.proto`                     | Current protobuf schema - will add `name` and `is_git_project` fields |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/project_port.go` | CLI ProjectPort interface - will add `Name` and `IsGitProject` to params/result |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/project_client.go` | CLI ConnectRPC client - will send new fields to API |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`          | CLI service - will pass name/isGitProject to API client (lines 109-113) |
| `/Users/jayce/team-attention/cops/api/internal/service/project/project_service.go`            | API service - will accept new fields in params |
| `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/project_repo_port.go` | API repository port - will add new params to FindOrCreate |
| `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go` | MongoDB repository - will store/return new fields |
| `/Users/jayce/team-attention/cops/api/internal/service/project/inbound/grpc/connectrpc/handler.go` | API gRPC handler - will parse new fields from request |
| `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`                       | MongoDB schema constants - already has `ProjectNameField` and `ProjectIsGitProjectField` |
| `/Users/jayce/team-attention/cops/.agent/rules/idl/protobuf.md`                               | Protobuf naming conventions (snake_case fields, Req/Res suffix) |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-backend.md`                              | Go parameter structure rules |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-platform-domain-mongoschema.md`          | MongoSchema field constant conventions |

## Current Implementation Analysis

### 1. Protobuf Schema (`/Users/jayce/team-attention/cops/idl/protobuf/project/v1/project.proto`)

**Current State (lines 9-29):**
```protobuf
message RegisterProjectReq {
  string configured_remote_url = 1;
  string actual_remote_url = 2;
  string existing_project_id = 3;
  // MISSING: name, is_git_project fields
}

message RegisterProjectRes {
  string project_id = 1;
  bool is_new = 2;
  // MISSING: name, is_git_project fields for response
}
```

**Required Changes:**
- Add `string name = 4;` to `RegisterProjectReq`
- Add `bool is_git_project = 5;` to `RegisterProjectReq`
- Add `string name = 3;` to `RegisterProjectRes` (for returning stored/generated name)
- Add `bool is_git_project = 4;` to `RegisterProjectRes`

**Field Numbering:**
- Request fields: 1-3 used, add new fields as 4 and 5
- Response fields: 1-2 used, add new fields as 3 and 4

### 2. CLI Project Port (`/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/project_port.go`)

**Current State (lines 10-27):**
```go
type RegisterProjectParams struct {
    ConfiguredRemoteURL string
    ActualRemoteURL     string
    ExistingProjectID   string
    // MISSING: Name, IsGitProject
}

type RegisterProjectResult struct {
    ProjectID domain.ID
    IsNew     bool
    // MISSING: Name, IsGitProject
}
```

**Required Changes:**
- Add `Name string` to `RegisterProjectParams`
- Add `IsGitProject bool` to `RegisterProjectParams`
- Add `Name string` to `RegisterProjectResult`
- Add `IsGitProject bool` to `RegisterProjectResult`

### 3. CLI ConnectRPC Client (`/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`)

**Current State (lines 39-61):**
```go
func (c *ProjectClient) RegisterProject(ctx context.Context, params api.RegisterProjectParams) (*api.RegisterProjectResult, error) {
    req := connect.NewRequest(&projectv1.RegisterProjectReq{
        ConfiguredRemoteUrl: params.ConfiguredRemoteURL,
        ActualRemoteUrl:     params.ActualRemoteURL,
        ExistingProjectId:   params.ExistingProjectID,
        // MISSING: Name, IsGitProject
    })
    // ...
    result := &api.RegisterProjectResult{
        ProjectID: domain.ID(resp.Msg.ProjectId),
        IsNew:     resp.Msg.IsNew,
        // MISSING: Name, IsGitProject
    }
}
```

**Required Changes:**
- Add `Name: params.Name` and `IsGitProject: params.IsGitProject` to request
- Add `Name: resp.Msg.Name` and `IsGitProject: resp.Msg.IsGitProject` to result

### 4. CLI Tracking Service (`/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`)

**Current State (lines 109-113):**
```go
result, err := s.project.RegisterProject(ctx, api.RegisterProjectParams{
    ConfiguredRemoteURL: configuredURL,
    ActualRemoteURL:     actualURL,
    ExistingProjectID:   existingProjectID,
    // MISSING: Name, IsGitProject
})
```

**Required Changes:**
- Add `Name: name` (from line 82-85)
- Add `IsGitProject: isGitProject` (from line 61)

### 5. API Service (`/Users/jayce/team-attention/cops/api/internal/service/project/project_service.go`)

**Current State (lines 11-15, 32-33):**
```go
type RegisterProjectParams struct {
    ConfiguredRemoteURL string
    ActualRemoteURL     string
    ExistingProjectID   string
    // MISSING: Name, IsGitProject
}

// In RegisterProject method:
result, err := s.repo.FindOrCreate(ctx, params.ConfiguredRemoteURL, params.ActualRemoteURL, params.ExistingProjectID)
// MISSING: passing Name, IsGitProject to repository
```

**Required Changes:**
- Add `Name string` and `IsGitProject bool` to `RegisterProjectParams`
- Pass new fields to `repo.FindOrCreate()`

### 6. API Repository Port (`/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/project_repo_port.go`)

**Current State (lines 5-18):**
```go
type FindOrCreateResult struct {
    ProjectID string
    IsNew     bool
    // MISSING: Name, IsGitProject
}

type ProjectRepositoryPort interface {
    FindOrCreate(ctx context.Context, configuredURL, actualURL, existingID string) (*FindOrCreateResult, error)
    // MISSING: name, isGitProject params
}
```

**Required Changes:**
- Add `Name string` and `IsGitProject bool` to `FindOrCreateResult`
- Update `FindOrCreate` signature to accept params struct instead of individual params (per go-backend.md rules - max 3 params)

### 7. MongoDB Repository (`/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`)

**Current State (lines 29-105):**
- Uses `ProjectRemoteURLField` for storage (line 88)
- Returns only `ProjectID` and `IsNew` in result
- Creates new doc with only `remoteUrl` field (lines 87-89)

**Required Changes:**
- Store `name` and `isGitProject` when creating new project
- Return `name` and `isGitProject` when finding existing project
- Accept new params via updated interface

### 8. API gRPC Handler (`/Users/jayce/team-attention/cops/api/internal/service/project/inbound/grpc/connectrpc/handler.go`)

**Current State (lines 35-60):**
```go
params := projectservice.RegisterProjectParams{
    ConfiguredRemoteURL: msg.GetConfiguredRemoteUrl(),
    ActualRemoteURL:     msg.GetActualRemoteUrl(),
    ExistingProjectID:   msg.GetExistingProjectId(),
    // MISSING: Name, IsGitProject
}
// ...
res := &projectv1.RegisterProjectRes{
    ProjectId: result.ProjectID,
    IsNew:     result.IsNew,
    // MISSING: Name, IsGitProject
}
```

**Required Changes:**
- Add `Name: msg.GetName()` and `IsGitProject: msg.GetIsGitProject()` to params
- Add `Name: result.Name` and `IsGitProject: result.IsGitProject` to response

### 9. MongoDB Schema Constants (`/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`)

**Current State (lines 12-22):**
```go
const (
    ProjectIDField           = "_id"
    ProjectNameField         = "name"           // Already exists!
    ProjectPathField         = "path"
    ProjectIsGitProjectField = "isGitProject"   // Already exists!
    ProjectClaudeDirField    = "claudeDir"
    ProjectRegisteredAtField = "registeredAt"
    ProjectGitBranchField    = "git_branch"
    ProjectWorktreesField    = "worktrees"
    ProjectRemoteURLField    = "remoteUrl"
)
```

**Note:** Field constants already exist for `name` and `isGitProject`. No changes needed here.

## Package Candidates

No new packages are required for this implementation. All necessary dependencies are already in use:
- `connectrpc.com/connect` - Already used for gRPC
- `go.mongodb.org/mongo-driver/v2` - Already used for MongoDB

## Technical Constraints

1. **Backward Compatibility**: New fields must be optional - existing MongoDB documents without `name`/`isGitProject` should still work
2. **Protobuf Field Numbers**: Must use next available numbers (4, 5 for request; 3, 4 for response)
3. **Go Parameter Limit**: Per `go-backend.md`, functions with >3 params should use a params struct
4. **Snake_case in Proto**: Per `protobuf.md`, field names must use snake_case (`is_git_project`, not `isGitProject`)
5. **Default Values**: API should provide sensible defaults if name is empty (e.g., extract from remote URL or use "Unknown Project")

## Similar Implementations Found

### Example 1: Existing Field Handling in Project Repository
- **File**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go:87-89`
- **Relevance**: Shows pattern for creating new MongoDB documents with field constants

```go
newDoc := bson.M{
    mongoschema.ProjectRemoteURLField: remoteURL,
}
```

### Example 2: MongoDB Schema Constants Pattern
- **File**: `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go:12-22`
- **Relevance**: Shows existing field constant naming pattern - `ProjectNameField` and `ProjectIsGitProjectField` already defined

### Example 3: Request/Response Mapping in gRPC Handler
- **File**: `/Users/jayce/team-attention/cops/api/internal/service/project/inbound/grpc/connectrpc/handler.go:40-58`
- **Relevance**: Shows pattern for mapping protobuf fields to service params and back to response

### Example 4: CLI Service Passing Params to API
- **File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go:109-113`
- **Relevance**: Shows where `name` (line 82) and `isGitProject` (line 61) are already available but not passed to API

## Backward Compatibility Considerations

1. **Existing MongoDB Documents**: Documents without `name`/`isGitProject` fields will return empty string and false respectively when queried - this is acceptable default behavior

2. **Protobuf Default Values**: In proto3, unset string fields default to empty string, unset bool fields default to false - this aligns with MongoDB behavior

3. **API Default Name Generation**: When `name` is empty for a new project, the API should generate a default name from the remote URL (e.g., extract repo name from `github.com/org/repo-name`)

4. **No Migration Required**: Per requirements, existing documents do not need to be migrated

## Risks and Edge Cases

1. **Empty Name Handling**: If CLI sends empty name (shouldn't happen in normal flow), API needs fallback logic
2. **Non-Git Projects**: When `isGitProject=false`, remote URLs will be empty - ensure MongoDB query handles this
3. **Response Name Mismatch**: If CLI sends a name but API generates a different default, CLI should use API response name
4. **Duplicate Detection**: Existing duplicate detection by remote URL remains unchanged - name/isGitProject are metadata only

## Implementation Order

1. **Protobuf Schema** - Add new fields to `RegisterProjectReq` and `RegisterProjectRes`
2. **Regenerate Stubs** - Run `cd idl/protobuf && buf generate`
3. **API Repository Port** - Update `FindOrCreateResult` and `FindOrCreate` signature
4. **API MongoDB Repository** - Store and return new fields
5. **API Service** - Update `RegisterProjectParams` and pass to repository
6. **API gRPC Handler** - Map new protobuf fields
7. **CLI Project Port** - Update `RegisterProjectParams` and `RegisterProjectResult`
8. **CLI ConnectRPC Client** - Send and receive new fields
9. **CLI Tracking Service** - Pass name and isGitProject to API client

## Additional Information for Planning

- The MongoDB schema constants (`ProjectNameField`, `ProjectIsGitProjectField`) already exist in `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`
- The domain model `domain.Project` already has `Name` and `IsGitProject` fields (see `/Users/jayce/team-attention/cops/shared/domain/project.go:9,17`)
- The CLI TUI already collects and passes these values to the tracking service (see `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go:54-59`)
- Per `go-backend.md`, the repository interface should use a params struct since it will have 5 parameters after the change
