# Development Walkthrough

## Summary
Added `cops remove` command and refactored project registration to use API server coordination, enabling centralized project ID management and support for dual remote URLs (configured vs actual).

## Code Overview

### New Components

#### `cops remove` CLI Command
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/remove.go`
- **Purpose**: Remove projects from tracking with user confirmation
- **Key Features**:
  - Interactive confirmation prompt (unless `--force` flag used)
  - Removes both global config (`~/.cops/config.json`) and local config (`.cops/` directory)
  - Clear user messaging about what will and won't be deleted
- **CLI Usage**:
  ```bash
  cops add .                # Remove current directory
  cops remove /path/to/project # Remove specific directory
  cops remove . --force        # Remove without confirmation
  ```

#### Project Service (API Server)
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/project/project_service.go`
- **Purpose**: Centralized project registration business logic
- **Key Methods**:
  - `RegisterProject(ctx, params)`: Orchestrates project registration via repository

#### Project Repository (MongoDB)
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`
- **Purpose**: MongoDB adapter for project persistence with intelligent duplicate detection
- **Key Method**:
  - `FindOrCreate(ctx, configuredURL, actualURL, existingID)`: Searches by remote URLs OR existing ID, creates if not found
- **Search Strategy**:
  1. Search by configured remote URL
  2. Search by actual remote URL (if different from configured)
  3. Search by existing project ID (if provided)
  4. Create new project if none found
- **Duplicate Prevention**: Uses MongoDB `$or` query to match any of the search criteria

#### Project gRPC Handler (ConnectRPC)
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/project/inbound/grpc/connectrpc/handler.go`
- **Purpose**: Exposes ProjectService.RegisterProject via ConnectRPC
- **RPC**: `RegisterProject(RegisterProjectReq) returns (RegisterProjectRes)`

#### Project Client (CLI Outbound Adapter)
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`
- **Purpose**: ConnectRPC client adapter for calling API server's ProjectService
- **Implements**: `api.ProjectPort` interface
- **Key Method**:
  - `RegisterProject(ctx, params)`: Calls API server's RegisterProject RPC

#### Protobuf Definitions
- **Location**: `/Users/jayce/team-attention/cops/idl/protobuf/project/v1/project.proto`
- **Purpose**: Project service API contract
- **Messages**:
  - `RegisterProjectReq`: Contains `configured_remote_url`, `actual_remote_url`, `existing_project_id`
  - `RegisterProjectRes`: Returns `project_id` (MongoDB ObjectID hex) and `is_new` flag
- **Service**: `ProjectService` with `RegisterProject` RPC

### Modified Components

#### Tracking Service (`AddProject`)
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`
- **Changes**:
  - Replaced local UUID generation with API server coordination
  - Added Git remote URL detection (both configured and actual URLs)
  - Calls `s.project.RegisterProject()` with dual URLs and optional existing project ID
  - Fallback to existing local ID if API unreachable
  - Fails if API unreachable AND no existing local ID
- **New Behavior**:
  - API server is single source of truth for project IDs
  - Supports renamed GitHub repositories (configured URL != actual URL)
  - Graceful degradation when API server unavailable (uses cached local ID)

#### Tracking Service (`RemoveProjectByPath`)
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`
- **Changes**: New method implementing project removal logic
- **Responsibilities**:
  1. Resolve project path to absolute path
  2. Load global config and verify project exists
  3. Remove project from global config
  4. Save updated global config
  5. Delete local `.cops/` directory

#### Repository Naming Standardization
- **Affected Files**: All MongoDB repository adapters in `api/internal/service/*/outbound/repository/mongodb/`
- **Change**: Renamed all repository structs to follow `Mongo{Domain}Repository` pattern
- **Examples**:
  - `aggregationRepo` → `MongoAggregationRepository`
  - `dashboardRepo` → `MongoDashboardRepository`
  - `projectRepo` → `MongoProjectRepository`
- **Rationale**: Consistent naming convention across codebase

#### Config Port Extensions
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/config/config_port.go`
- **Added Methods**:
  - `LoadLocalConfig(projectPath)`: Load project-specific `.cops/config.json`
  - `SaveLocalConfig(projectPath, cfg)`: Save project-specific `.cops/config.json`
  - `LoadGlobalConfig()`: Load global `~/.cops/config.json`
  - `SaveGlobalConfig(cfg)`: Save global `~/.cops/config.json`
- **Rationale**: Separate global and local config operations

#### Git Utility Improvements
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/platform/util/gitutil/gitutil.go`
- **Added Functions**:
  - `GetRemoteURL(projectPath)`: Returns configured remote URL from `git config --get remote.origin.url`
  - `GetActualRemoteURL(projectPath)`: Returns actual remote URL from `git ls-remote --get-url origin`
- **Purpose**: Support dual URL detection for renamed repositories

#### Dependency Injection Updates
- **Location**: `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_tracking.go`
- **Changes**:
  - Registered `connectrpc.NewProjectClient` with `dig.As(new(api.ProjectPort))`
  - Simplified import aliases (removed unnecessary `connect_rpc` alias)
- **Location**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_project.go`
- **Changes**: New module registering Project service, repository, and handler

## Testing

- **Build Verification**:
  ```bash
  go build ./cli/... ./api/... ./daemon/... ./shared/...
  # Result: PASS (no compilation errors)
  ```

- **Manual Testing Commands**:
  ```bash
  # Test project addition with API coordination
  cops add .

  # Test project removal with confirmation
  cops remove .

  # Test project removal without confirmation
  cops remove . --force

  # Test listing projects
  cops list
  ```

- **Test Coverage**: No unit tests added (implementation-focused task)

## Issues & Resolutions

| Issue | Resolution |
| ----- | ---------- |
| Project IDs were generated locally, causing duplicates | API server now generates and stores project IDs as single source of truth |
| GitHub repository renames broke project matching | Added dual URL support: configured URL (from git config) and actual URL (from git ls-remote) |
| No way to remove registered projects | Implemented `cops remove` command with confirmation prompt and `--force` flag |
| Repository naming inconsistent across API services | Standardized all MongoDB repositories to `Mongo{Domain}Repository` pattern |
| DTO duplication between service and repository layers | Consolidated DTOs - repository layer returns `FindOrCreateResult` used directly by service |

## Architecture Decisions

### API Server as Project Registry
**Decision**: API server coordinates project registration and assigns project IDs

**Rationale**:
- Single source of truth prevents duplicate project IDs
- Centralized duplicate detection using remote URLs
- Enables future cross-CLI project sharing

**Implementation**:
1. CLI calls API's `ProjectService.RegisterProject` with remote URLs and optional existing ID
2. API searches MongoDB by URLs or existing ID
3. API returns existing ID or creates new project
4. CLI caches project ID locally in `.cops/config.json`

**Fallback Behavior**:
- If API unreachable but local ID exists, CLI uses cached ID
- If API unreachable and no local ID, operation fails with clear error

### Dual Remote URL Support
**Decision**: Track both configured and actual remote URLs

**Rationale**:
- GitHub repositories can be renamed, changing the actual URL
- Git config may still reference old URL (`git config --get remote.origin.url`)
- Git fetch still works with old URL (`git ls-remote --get-url origin` returns new URL)

**Implementation**:
1. `GetRemoteURL()` reads configured URL from git config
2. `GetActualRemoteURL()` resolves actual URL via `git ls-remote`
3. Both URLs sent to API server for duplicate detection
4. MongoDB query uses `$or` to match either URL

**Example Scenario**:
```bash
# Repository renamed from github.com/user/old-name to github.com/user/new-name
git config --get remote.origin.url        # github.com/user/old-name
git ls-remote --get-url origin            # github.com/user/new-name

# API server matches project by either URL
```

### Repository Pattern Consolidation
**Decision**: Use repository DTOs directly in service layer, avoid intermediate mapping

**Rationale**:
- Repository `FindOrCreateResult` contains exactly what service needs
- No additional computation or transformation required
- Reduces boilerplate code

**Before** (typical pattern):
```go
// Repository returns DTO
type RepositoryDTO struct { ... }

// Service maps to different DTO
type ServiceResult struct { ... }
func (s *Service) Method() (*ServiceResult, error) {
    repoResult := s.repo.Method()
    return mapToServiceResult(repoResult), nil
}
```

**After** (consolidated):
```go
// Repository returns shared DTO
type FindOrCreateResult struct { ... }

// Service uses repository DTO directly
func (s *Service) RegisterProject() (*repository.FindOrCreateResult, error) {
    return s.repo.FindOrCreate(), nil
}
```

### MongoDB $or Query Strategy
**Decision**: Single MongoDB query with `$or` condition for all search criteria

**Rationale**:
- Efficient: One database round-trip instead of multiple queries
- Clear: All matching logic in one place
- Flexible: Easy to add more search criteria

**Implementation**:
```go
conditions := []bson.M{}
if configuredURL != "" {
    conditions = append(conditions, bson.M{"remoteURL": configuredURL})
}
if actualURL != "" && actualURL != configuredURL {
    conditions = append(conditions, bson.M{"remoteURL": actualURL})
}
if existingID != "" {
    if objectID, err := bson.ObjectIDFromHex(existingID); err == nil {
        conditions = append(conditions, bson.M{"_id": objectID})
    }
}

filter := bson.M{"$or": conditions}
err := r.projectsColl.FindOne(ctx, filter).Decode(&doc)
```

## Related Tickets
- No external ticket references (internal refactoring and feature addition)
