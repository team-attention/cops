# Implementation Plan

## Overview

Implement two CLI features: (1) `cops remove` command to unregister projects from local configs, and (2) modify `cops add` to coordinate with API server via ConnectRPC to get ProjectIDs based on git remote URL for duplicate detection.

## Selected Packages

| Problem | Package | Reason for Selection |
|---------|---------|---------------------|
| User Confirmation | bufio (stdlib) | No external dependency needed for simple y/n confirmation |
| ConnectRPC Client | connectrpc | Already used in project for API communication |
| HTTP Client | imroc/req/v3 | Already used (`APIHTTPClient` exists in httpclient package) |
| MongoDB Driver | mongo-go-driver v2 | Already used in project for data persistence |

## Architecture Decisions

### Decision 1: Protobuf Service Location
**Choice**: Create `idl/protobuf/project/v1/project.proto`
**Rationale**: Follows existing pattern (collector/v1, aggregation/v1, dashboard/v1)

### Decision 2: API Server Only Stores Remote URL
**Choice**: Server stores only ProjectID (MongoDB ObjectID) and git remote URL
**Rationale**: Project can be used by multiple people; local-specific info (path, name) cannot be uniquely associated on server

### Decision 3: Multi-Factor Duplicate Detection
**Choice**: Check by remote URLs first (both configured and actual), then by existing project ID as fallback
**Rationale**: Remote URLs can change if GitHub repo is renamed; existing project ID provides fallback for such cases

### Decision 4: Dual Remote URL Support
**Choice**: CLI sends BOTH configured remote URL and actual remote URL to API
**Rationale**: Git has two types of remote URLs that can differ (configured in `.git/config` vs actual from `git ls-remote`). Server should check against both.

### Decision 5: CLI Handles Local Information
**Choice**: CLI finds main branch path locally, sends remote URLs and optional existing ID to API
**Rationale**: API doesn't need worktree info; CLI manages local config storage

### Decision 6: Remove is Local-Only
**Choice**: `cops remove` only modifies local configs (global config + local .cops/ directory), no API communication
**Rationale**: Per requirements, does NOT delete project data from server. Does NOT delete Claude Code logs.

### Decision 7: HTTP Client Reuse
**Choice**: Use existing `APIHTTPClient` (uses imroc/req/v3)
**Rationale**: Already configured with proper timeout; has `StandardHTTPClient()` for ConnectRPC

### Decision 8: Non-Git Fallback
**Choice**: Non-git projects use local UUID generation (no API call)
**Rationale**: API requires remote URL; without it, no benefit to server registration

### Decision 9: Single Upsert-Like Repository Method
**Choice**: Use single `FindOrCreate` method instead of separate `FindByRemoteURL` and `Create`
**Rationale**: Simplifies service logic; handles all duplicate detection and creation in one atomic operation

---

## Implementation Steps

### Step 1: Create Project Protobuf Service

**File**: `/Users/jayce/team-attention/cops/idl/protobuf/project/v1/project.proto` (create)

**Structure**:
```protobuf
message RegisterProjectReq {
  string configured_remote_url = 1;  // From git config (git config --get remote.origin.url)
  string actual_remote_url = 2;      // From git ls-remote (git ls-remote --get-url origin)
  string existing_project_id = 3;    // Optional - from local config if available
}

message RegisterProjectRes {
  string project_id = 1;  // MongoDB ObjectID hex
  bool is_new = 2;        // For logging
}

service ProjectService {
  rpc RegisterProject(RegisterProjectReq) returns (RegisterProjectRes);
}
```

**Key details**:
- go_package: `github.com/team-attention/cops/shared/gen/grpcstub/project/v1;projectv1`
- Request includes both remote URLs for robust matching
- `existing_project_id` is optional (empty string if not available locally)
- Minimal response: only `project_id` and `is_new`

---

### Step 2: Generate Protobuf Code

**Command**: `cd /Users/jayce/team-attention/cops/idl/protobuf && buf generate`

**Generated files**:
- `shared/gen/grpcstub/project/v1/project.pb.go`
- `shared/gen/grpcstub/project/v1/projectv1connect/project.connect.go`

---

### Step 3: Add DeleteLocalConfig to ConfigPort Interface

**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/config/config_port.go` (modify)

**Change**: Add method to interface:
```go
DeleteLocalConfig(projectPath string) error
```

---

### Step 4: Implement DeleteLocalConfig

**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/config/filesystem/filesystem_config.go` (modify)

**Implementation approach**:
```
// DeleteLocalConfig:
// 1. Build path: filepath.Join(projectPath, ".cops")
// 2. Check if exists with os.Stat
// 3. If not exists -> return nil (graceful, not an error)
// 4. If exists -> os.RemoveAll to delete entire directory
// 5. Log result
```

**Critical details**:
- Use existing `localConfigDirName` constant (`.cops`)
- Return nil if directory doesn't exist (graceful handling)
- Log at Debug level for "not exists", Info for successful delete

---

### Step 5: Create ProjectPort Interface

**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/project_port.go` (create)

**Structure**:
```go
type RegisterProjectParams struct {
    ConfiguredRemoteURL string  // From git config
    ActualRemoteURL     string  // From git ls-remote
    ExistingProjectID   string  // Optional - from local config
}

type RegisterProjectResult struct {
    ProjectID domain.ID
    IsNew     bool
}

type ProjectPort interface {
    RegisterProject(ctx context.Context, params RegisterProjectParams) (*RegisterProjectResult, error)
}
```

**Key details**:
- Accepts both remote URLs for robust matching
- `ExistingProjectID` can be empty string if not available
- Returns `domain.ID` (not string) for consistency with existing code

---

### Step 6: Implement ProjectClient (ConnectRPC)

**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/project_client.go` (create)

**Implementation approach**:
```
// NewProjectClient:
// 1. Accept: *slog.Logger, *config.Config, *httpclient.APIHTTPClient
// 2. Create projectv1connect.NewProjectServiceClient with httpClient.StandardHTTPClient()
// 3. Use cfg.API.URL as base URL

// RegisterProject:
// 1. Create connect.NewRequest with all three fields:
//    - ConfiguredRemoteUrl
//    - ActualRemoteUrl
//    - ExistingProjectId (can be empty string)
// 2. Call client.RegisterProject
// 3. On error: log and return error
// 4. On success: convert resp.Msg.ProjectId to domain.ID, return result
```

**Critical details**:
- Use existing `APIHTTPClient` (NOT create new HTTP client)
- Follow pattern from `collector_client.go`
- Add compile-time interface check: `var _ api.ProjectPort = (*ProjectClient)(nil)`

---

### Step 7: Create API Server Project Service

**File**: `/Users/jayce/team-attention/cops/api/internal/service/project/project_service.go` (create)

**Implementation approach**:
```
// RegisterProject(ctx, configuredURL, actualURL, existingID string):
// 1. Call repo.FindOrCreate(ctx, configuredURL, actualURL, existingID)
// 2. Return result with ProjectID and IsNew
```

**Structure**:
```go
type RegisterProjectParams struct {
    ConfiguredRemoteURL string
    ActualRemoteURL     string
    ExistingProjectID   string
}

type RegisterProjectResult struct {
    ProjectID string
    IsNew     bool
}

type Service struct {
    logger *slog.Logger
    repo   repository.ProjectRepositoryPort
}
```

**Key details**:
- Service is thin - just delegates to repository's FindOrCreate
- All duplicate detection logic lives in repository

---

### Step 8: Create Project Repository Port

**File**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/project_repo_port.go` (create)

**Structure**:
```go
type FindOrCreateResult struct {
    ProjectID string
    IsNew     bool
}

type ProjectRepositoryPort interface {
    // FindOrCreate finds existing project or creates new one.
    // Search order:
    // 1. By remote URL (either configured or actual)
    // 2. By existing project ID (if provided)
    // 3. Create new if not found
    FindOrCreate(ctx context.Context, configuredURL, actualURL, existingID string) (*FindOrCreateResult, error)
}
```

**Key details**:
- Single method handles all: find by URL, find by ID, or create
- Returns `(result, error)` - never returns nil result on success

---

### Step 9: Add RemoteURL Field to MongoDB Schema

**File**: `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go` (modify)

**Change**: Add constant:
```go
ProjectRemoteURLField = "remoteUrl"  // camelCase per project convention
```

---

### Step 10: Implement MongoDB Project Repository

**File**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go` (create)

**Implementation approach**:
```
// FindOrCreate(ctx, configuredURL, actualURL, existingID):
// 1. Build $or conditions array with all search criteria:
//    conditions := []bson.M{}
//    - If configuredURL != "": append { remoteUrl: configuredURL }
//    - If actualURL != "": append { remoteUrl: actualURL }
//    - If existingID is valid ObjectID: append { _id: ObjectIDFromHex(existingID) }
//
// 2. Validate at least one condition exists:
//    - If len(conditions) == 0: return error (no search criteria provided)
//
// 3. Execute single findOne with $or filter:
//    - Query: { $or: conditions }
//    - If found -> return (doc.ID, isNew=false)
//
// 4. If not found, create new document:
//    - remoteUrl = configuredURL (prefer configured)
//    - If configuredURL is empty, use actualURL
//    - Insert new document with { remoteUrl: chosen_url }
//    - Return (newID, isNew=true)
```

**Critical details**:
- Use `mongoschema.ProjectCollectionName` for collection
- Use `mongoschema.ProjectRemoteURLField` ("remoteUrl" - camelCase)
- Use single `$or` query to search ALL criteria at once (efficient)
- Validate ObjectID format before including in query: `primitive.ObjectIDFromHex(existingID)` - skip if invalid
- Ensure at least one valid search condition exists before querying
- Store `configuredURL` as the canonical remoteUrl (fallback to actualURL if empty)
- Follow pattern from `dashboard_repo.go`

---

### Step 11: Create Project gRPC Handler

**File**: `/Users/jayce/team-attention/cops/api/internal/service/project/inbound/grpc/connectrpc/handler.go` (create)

**Implementation approach**:
```
// GetHandler:
// Return projectv1connect.NewProjectServiceHandler(h, opts...)

// RegisterProject:
// 1. Extract from req.Msg:
//    - ConfiguredRemoteUrl
//    - ActualRemoteUrl
//    - ExistingProjectId
// 2. Call svc.RegisterProject(ctx, params)
// 3. Return response with ProjectId and IsNew
```

**Critical details**:
- Implement `projectv1connect.ProjectServiceHandler` interface
- Implement `ConnectHandler` interface (GetHandler method)
- Follow pattern from existing handlers (aggregation, dashboard)

---

### Step 12: Create Project Module

**File**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_project.go` (create)

**Structure**:
```go
func newProjectModule() fx.Option {
    return fx.Module("project",
        // Repository: mongodb.NewAdapter -> ProjectRepositoryPort
        // Service: project.NewService
        // Handler: connectrpc.NewProjectGRPCHandler -> ConnectHandler (group:"connect_handlers")
    )
}
```

**Key details**:
- Follow pattern from `module_dashboard.go`
- Handler must be tagged with `group:"connect_handlers"` for auto-registration

---

### Step 13: Register Project Module

**File**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/application.go` (modify)

**Change**: Add `newProjectModule()` to fx.New() modules list

---

### Step 14: Add RemoveProjectByPath to Tracking Service

**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go` (modify)

**Implementation approach**:
```
// RemoveProjectByPath(ctx, params):
// 1. Expand path with pathutil.ExpandPath
// 2. Load global config
// 3. Filter out project with matching path
// 4. Delete local config (.cops/ directory) - graceful if not exists
// 5. If project was found in global config:
//    - Save updated global config
//    - Log info: "project removed from tracking"
// 6. If not found in global config:
//    - Log info: "project not in global config, local config deleted if existed"
```

**Critical details**:
- Delete local config BEFORE updating global config (ensures partial cleanup on failure)
- Continue with global config update even if local delete fails (log warning)
- Match by absolute path
- Does NOT delete Claude Code session logs - only removes from tracking
- Does NOT communicate with API server

---

### Step 15: Modify AddProject for API Registration

**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go` (modify)

**Changes needed**:

1. Add `project api.ProjectPort` field to Service struct
2. Add `project api.ProjectPort` parameter to NewService
3. Modify AddProject logic:

```
// AddProject modified logic:
// 1. Expand path
// 2. Detect git project, find main repo path (handles worktrees)
// 3. Get BOTH remote URLs from main repo path:
//    - configuredURL: gitutil.GetRemoteURL(mainPath)  // git config --get
//    - actualURL: gitutil.GetActualRemoteURL(mainPath) // git ls-remote --get-url
// 4. Check if local config exists -> get existingProjectID (or empty string)
// 5. Determine projectID:
//    IF isGitProject AND (configuredURL != "" OR actualURL != ""):
//        - Call s.project.RegisterProject(ctx, params) with both URLs and existingID
//        - Use returned ProjectID
//    ELSE (non-git or no remote):
//        - Use existing local ID if available
//        - Otherwise generate new UUID
// 6. Save local config with projectID
// 7. Update global config (add or update by ID match)
```

**Critical details**:
- Need to add `GetActualRemoteURL` to gitutil (uses `git ls-remote --get-url origin`)
- Keep `uuid` import for non-git fallback
- Only call API if git project WITH at least one remote URL
- Worktree detection stays in CLI, only remote URLs and optional existingID sent to API

---

### Step 16: Create Remove Command

**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/remove.go` (create)

**Implementation approach**:
```
// NewRemoveCommand:
// - Use: "remove <project-path>"
// - Args: cobra.ExactArgs(1)
// - Flag: --force/-f (skip confirmation)

// RunE logic:
// 1. If not --force:
//    - Print confirmation prompt
//    - Read response with bufio.NewReader(os.Stdin)
//    - If not "y" or "yes": print "Cancelled." and return nil
// 2. Call h.svc.RemoveProjectByPath(ctx, params)
// 3. Print "Project removed successfully!"
```

**Critical details**:
- Use `bufio.NewReader(os.Stdin).ReadString('\n')` for input
- Normalize response: `strings.TrimSpace(strings.ToLower(response))`
- Accept both "y" and "yes"

---

### Step 17: Register Remove Command

**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/handler.go` (modify)

**Change**: Add `h.NewRemoveCommand()` to Commands() return slice

---

### Step 18: Register ProjectClient in Container

**File**: `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_tracking.go` (modify)

**Change**: Add ProjectClient registration:
```go
c.Provide(connectrpcadapter.NewProjectClient, dig.As(new(api.ProjectPort)))
```

---

### Step 19: Verify APIHTTPClient Registration

**File**: `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_platform.go` (verify)

**Action**: Ensure `httpclient.InitAPIHTTPClient` is already provided. If not, add it.

---

### Step 20: Add GetActualRemoteURL to gitutil

**File**: `/Users/jayce/team-attention/cops/cli/internal/platform/util/gitutil/gitutil.go` (modify)

**Implementation approach**:
```
// GetActualRemoteURL returns the actual remote URL (what GitHub points to).
// Uses: git ls-remote --get-url origin
// This can differ from configured URL if the repo was renamed on GitHub.
//
// Implementation:
// 1. Run: git ls-remote --get-url origin
// 2. Return trimmed stdout
// 3. On error, return empty string (not an error)
```

**Critical details**:
- Different from `GetRemoteURL` which uses `git config --get remote.origin.url`
- `git ls-remote --get-url` follows redirects and returns actual URL
- Return empty string on error (graceful handling)

---

## Execution Order

1. **Step 1**: Create protobuf definition
2. **Step 2**: Generate protobuf code (depends on 1)
3. **Step 9**: Add RemoteURL field to MongoDB schema
4. **Step 20**: Add GetActualRemoteURL to gitutil
5. **Step 3**: Add DeleteLocalConfig to ConfigPort
6. **Step 4**: Implement DeleteLocalConfig (depends on 3)
7. **Step 5**: Create ProjectPort interface
8. **Step 6**: Implement ProjectClient (depends on 2, 5)
9. **Step 8**: Create Project Repository Port
10. **Step 10**: Implement MongoDB Repository (depends on 8, 9)
11. **Step 7**: Create Project Service (depends on 8)
12. **Step 11**: Create gRPC Handler (depends on 2, 7)
13. **Step 12**: Create Project Module (depends on 10, 7, 11)
14. **Step 13**: Register Project Module (depends on 12)
15. **Step 14**: Add RemoveProjectByPath (depends on 3, 4)
16. **Step 15**: Modify AddProject (depends on 5, 6, 20)
17. **Step 16**: Create Remove Command (depends on 14)
18. **Step 17**: Register Remove Command (depends on 16)
19. **Step 19**: Verify APIHTTPClient registration
20. **Step 18**: Register ProjectClient (depends on 6, 19)

---

## Notes for Execute Agent

- **Proto Generation**: Run `buf generate` immediately after creating proto file, before writing Go code that imports generated types
- **Go Modules**: Run `go mod tidy` in `cli/` and `api/` after implementation
- **Pattern Following**: Reference existing implementations:
  - CLI client: `collector_client.go`
  - API handler: `aggregation/inbound/grpc/connectrpc/handler.go`
  - Repository: `dashboard/outbound/repository/mongodb/dashboard_repo.go`
  - Module: `module_dashboard.go`
- **MongoDB Field Naming**: Use camelCase (`remoteUrl`) not snake_case
- **UUID Fallback**: Keep `uuid` import in tracking service for non-git projects
- **Graceful Delete**: `DeleteLocalConfig` should succeed even if directory doesn't exist
- **Dual URL Matching**: Repository must check BOTH configured and actual URLs when finding existing projects
- **Remove Command Scope**: Only deletes from global config and local .cops/ directory - does NOT delete Claude Code logs or API data
