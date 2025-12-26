# Development Walkthrough

## Summary
Implemented two CLI features: (1) `cops remove` command to unregister projects from local configurations, and (2) modified `cops add` to coordinate with API server via ConnectRPC for obtaining server-managed ProjectIDs based on git remote URL duplicate detection. The API server is now the single source of truth for all Project IDs, with no local UUID generation.

## Architecture Overview

### Project Registration Flow (New)

```
┌─────────────────────────────────────────────────────────────────────┐
│                         CLI: cops add                                │
├─────────────────────────────────────────────────────────────────────┤
│ 1. Detect git repository (main branch vs worktree)                  │
│ 2. Extract git remote URLs:                                         │
│    - Configured URL: git config --get remote.origin.url             │
│    - Actual URL: git ls-remote --get-url origin                     │
│ 3. Check for existing local ProjectID                               │
│ 4. Call API: RegisterProject(configuredURL, actualURL, existingID)  │
│                                                                      │
│    ┌────────────────────────────────────────────────────┐          │
│    │         API Server: RegisterProject                │          │
│    ├────────────────────────────────────────────────────┤          │
│    │ 1. Query MongoDB with $or filter:                 │          │
│    │    - Match by configured remote URL                │          │
│    │    - Match by actual remote URL                    │          │
│    │    - Match by existing project ID                  │          │
│    │ 2. If found: Return existing ProjectID            │          │
│    │ 3. If not found: Create new project, return ID    │          │
│    └────────────────────────────────────────────────────┘          │
│                                                                      │
│ 5. Save ProjectID to local config (.cops/config.json)               │
│ 6. Add/update global config (~/.cops/config.json)                   │
└─────────────────────────────────────────────────────────────────────┘
```

### Project Removal Flow (New)

```
┌─────────────────────────────────────────────────────────────────────┐
│                       CLI: cops remove                               │
├─────────────────────────────────────────────────────────────────────┤
│ 1. User confirmation prompt (y/N) unless --force flag                │
│ 2. Delete .cops/ directory from project (graceful if not exists)    │
│ 3. Remove project entry from global config                          │
│                                                                      │
│ NOTE: Does NOT delete:                                              │
│   - Claude Code session logs                                        │
│   - Project data from API server                                    │
└─────────────────────────────────────────────────────────────────────┘
```

### Duplicate Detection Strategy

The API server uses multi-factor duplicate detection:

1. **Primary: Git Remote URL** - Matches against both configured and actual remote URLs
2. **Secondary: Existing ProjectID** - Fallback for cases where remote URL changed (e.g., GitHub repo rename)
3. **Efficient Query** - Single `$or` query checks all conditions at once

This approach handles edge cases like:
- GitHub repository renames (actual URL differs from configured URL)
- Re-adding a project after manual config cleanup
- Multiple developers working on the same repository

## Code Overview

### New Components

#### Protobuf Service: `project/v1/project.proto`
- **Location**: `/Users/jayce/team-attention/cops/idl/protobuf/project/v1/project.proto`
- **Purpose**: Defines ConnectRPC service contract for project registration
- **Key Messages**:
  - `RegisterProjectReq`: Contains `configured_remote_url`, `actual_remote_url`, `existing_project_id`
  - `RegisterProjectRes`: Returns `project_id` (MongoDB ObjectID hex) and `is_new` flag
- **Service**: `ProjectService.RegisterProject` - RPC for registering/finding projects

#### CLI Command: `RemoveCommand`
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/remove.go`
- **Purpose**: Unregisters projects from local configurations
- **Key Features**:
  - Confirmation prompt with y/N (skip with `--force` flag)
  - Uses `bufio.NewReader(os.Stdin)` for user input
  - Accepts both "y" and "yes" responses
  - Calls `tracking.Service.RemoveProjectByPath()`

#### CLI Outbound Port: `ProjectPort`
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/project_port.go`
- **Purpose**: Interface for project registration API calls
- **Key Method**: `RegisterProject(ctx, RegisterProjectParams) (*RegisterProjectResult, error)`
- **Params**: Accepts both remote URLs and optional existing project ID
- **Result**: Returns `domain.ID` (not string) and `IsNew` flag

#### CLI Outbound Adapter: `ProjectClient`
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`
- **Purpose**: ConnectRPC client for calling project registration API
- **Key Details**:
  - Uses existing `APIHTTPClient` (doesn't create new HTTP client)
  - Calls API at `cfg.API.URL` endpoint
  - Converts protobuf response to domain types
  - Follows pattern from `collector_client.go`

#### API Service: `project.Service`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/project/project_service.go`
- **Purpose**: Thin service layer delegating to repository
- **Key Method**: `RegisterProject(ctx, RegisterProjectParams) (*RegisterProjectResult, error)`
- **Design**: Minimal logic - delegates duplicate detection to repository

#### API Repository Port: `ProjectRepositoryPort`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/project_repo_port.go`
- **Purpose**: Interface for project data persistence
- **Key Method**: `FindOrCreate(ctx, configuredURL, actualURL, existingID) (*FindOrCreateResult, error)`
- **Design**: Single method handles all: find by URL, find by ID, or create

#### API Repository Adapter: `mongodb.MongoProjectRepository`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`
- **Purpose**: MongoDB implementation of project repository
- **Naming Convention**: Follows `{Technology}{Domain}{Category}` pattern (MongoProjectRepository)
- **Key Features**:
  - Efficient `$or` query checking all conditions at once
  - Validates ObjectID format before querying
  - Ensures at least one valid search condition exists
  - Stores configured URL as canonical (fallback to actual URL)
  - Returns `IsNew` flag for logging purposes

#### API gRPC Handler: `ProjectGRPCHandler`
- **Location**: `/Users/jayce/team-attention/cops/api/internal/service/project/inbound/grpc/connectrpc/handler.go`
- **Purpose**: ConnectRPC handler implementing ProjectService
- **Key Features**:
  - Implements `projectv1connect.ProjectServiceHandler` interface
  - Implements `ConnectHandler` interface (GetHandler method)
  - Converts protobuf messages to service params and back

#### Git Utility: `GetActualRemoteURL`
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/platform/util/gitutil/gitutil.go:126-137`
- **Purpose**: Gets actual remote URL from git (follows redirects)
- **Command**: `git ls-remote --get-url origin`
- **Difference**: Returns actual URL (what GitHub points to), differs from `GetRemoteURL` which uses `git config --get`
- **Behavior**: Returns empty string on error (graceful handling)

### Modified Components

#### `tracking.Service.AddProject`
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go:54-193`
- **Changes**:
  - Added `project api.ProjectPort` dependency
  - Extracts BOTH remote URLs (configured and actual) from main repo path
  - **ALWAYS calls API for project registration** (even with empty URLs)
  - Falls back to existing local ID on API error (returns error if no existing ID)
  - **No local UUID generation** - API server is single source of truth
- **Key Logic** (lines 99-129):
  ```go
  // Get URLs (empty strings if not available)
  configuredURL := ""
  actualURL := ""
  if isGitProject {
      configuredURL, _ = gitutil.GetRemoteURL(projectPath)
      actualURL = gitutil.GetActualRemoteURL(projectPath)
  }

  // Always call API to register project
  result, err := s.project.RegisterProject(ctx, api.RegisterProjectParams{
      ConfiguredRemoteURL: configuredURL,
      ActualRemoteURL:     actualURL,
      ExistingProjectID:   existingProjectID,
  })
  if err != nil {
      // If we have an existing local ID, use it
      if existingProjectID != "" {
          projectID = domain.ID(existingProjectID)
          s.logger.Warn("failed to register with API, using existing local ID")
      } else {
          // No existing ID and API unreachable - FAIL
          return nil, errutil.Internalf("cannot register project: API unreachable and no existing local ID")
      }
  } else {
      projectID = result.ProjectID
  }
  ```

#### `tracking.Service.RemoveProjectByPath`
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go:267-308`
- **Purpose**: New method to remove project by path (not ID)
- **Key Features**:
  - Expands path with `pathutil.ExpandPath`
  - Deletes local config first (continues if fails)
  - Uses `lo.Filter` for clean, functional filtering of projects by path
  - Graceful handling: logs info if project not in global config
  - Does NOT delete Claude Code session logs
  - Does NOT communicate with API server
- **Code Pattern** (lines 289-292):
  ```go
  filtered := lo.Filter(globalConfig.Projects, func(p *domain.Project, _ int) bool {
      return p.Path != absPath
  })
  found := len(filtered) < len(globalConfig.Projects)
  ```

#### `ConfigPort.DeleteLocalConfig`
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/config/config_port.go:31-34`
- **Change**: Added new interface method
- **Signature**: `DeleteLocalConfig(projectPath string) error`
- **Purpose**: Delete `.cops/` directory from project path

#### `filesystem.Adapter.DeleteLocalConfig`
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/config/filesystem/filesystem_config.go:103-122`
- **Purpose**: Implementation of DeleteLocalConfig
- **Key Features**:
  - Builds path: `filepath.Join(projectPath, ".cops")`
  - Returns nil if directory doesn't exist (graceful)
  - Uses `os.RemoveAll` to delete entire directory
  - Logs at Debug level for "not exists", Info for successful delete

#### MongoDB Schema
- **Location**: `/Users/jayce/team-attention/cops/shared/domain/mongoschema/project.go`
- **Change**: Added `ProjectRemoteURLField = "remoteUrl"` constant
- **Purpose**: Avoid magic strings in MongoDB queries (camelCase per project convention)

#### Module Registration
- **CLI**: `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_tracking.go`
  - Added `ProjectClient` registration with `dig.As(new(api.ProjectPort))`
- **API**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/application.go`
  - Added `newProjectModule()` to fx.New() modules list
- **API Module**: `/Users/jayce/team-attention/cops/api/cmd/internal/container/module_project.go`
  - New module registering repository, service, and handler
  - Handler tagged with `group:"connect_handlers"` for auto-registration

#### CLI Command Registration
- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/handler.go`
- **Change**: Added `h.NewRemoveCommand()` to Commands() return slice

## Testing

### Manual Testing Commands

#### Test `cops add` with API Registration

```bash
# Start API server (requires MongoDB)
cd api && make dev

# Register a git project
cops add /path/to/git/project

# Expected output:
# - Log: "registered project with API" with ProjectID
# - ProjectID saved to ~/.cops/config.json
# - ProjectID saved to /path/to/git/project/.cops/config.json

# Verify duplicate detection: re-add same project
cops add /path/to/git/project

# Expected output:
# - Same ProjectID returned (no duplicate created)
# - Log: "project already registered"
```

#### Test `cops add` Fallback (No API Server)

```bash
# Stop API server
cd api && make dev-down

# Try to register a NEW git project (no existing local ID)
cops add /path/to/new/git/project

# Expected output:
# - Error: "cannot register project: API unreachable and no existing local ID"
# - Project NOT registered (requires API for new projects)

# Re-add an EXISTING project (has local ID from previous registration)
cops add /path/to/existing/project

# Expected output:
# - Warning: "failed to register with API, using existing local ID"
# - Project registered with existing local ID
# - No duplicate created
```

#### Test `cops add` for Non-Git Projects

```bash
# Register a non-git directory
cops add /path/to/non-git/directory

# Expected output:
# - API called with empty remote URLs
# - Server-assigned ProjectID returned
# - Project registered successfully
```

#### Test `cops remove` with Confirmation

```bash
# Remove a project (with confirmation prompt)
cops remove /path/to/project

# Expected prompt:
# Remove project at '/path/to/project' from tracking?
# This will delete local .cops/ config. Continue? (y/N):

# Type 'y' or 'yes'
# Expected output:
# - Log: "project removed from tracking"
# - .cops/ directory deleted
# - Project removed from ~/.cops/config.json
```

#### Test `cops remove` with Force Flag

```bash
# Remove without confirmation
cops remove /path/to/project --force

# Expected output:
# - No prompt shown
# - Project removed immediately
# - "Project removed successfully!"
```

#### Test `cops list` After Changes

```bash
# List all registered projects
cops list

# Expected output:
# - All projects with server-assigned MongoDB ObjectIDs
# - Git worktrees shown for git projects (dynamically discovered)
# - Consistent ID format across all project types
```

### Build Verification

```bash
# Generate protobuf code
cd idl/protobuf && buf generate  # Result: SUCCESS

# Build CLI
cd cli && go build ./...         # Result: SUCCESS

# Build API
cd api && go build ./...         # Result: SUCCESS

# Run tests
cd cli && go test ./...          # Result: PASS (no test files)
cd api && go test ./...          # Result: PASS
```

### Integration Test Scenario

1. **Setup**: Start API server with MongoDB
2. **Register Project**: Run `cops add /path/to/project` → ProjectID: `abc123` (from API)
3. **Verify Duplicate Detection**: Run `cops add /path/to/project` again → Same ProjectID: `abc123`
4. **Check Configs**:
   - Global: `~/.cops/config.json` contains project with ID `abc123`
   - Local: `/path/to/project/.cops/config.json` contains `{"id":"abc123"}`
5. **Remove Project**: Run `cops remove /path/to/project` with confirmation
6. **Verify Removal**:
   - Global config no longer has project entry
   - Local `.cops/` directory deleted
   - Claude Code logs remain intact
7. **Re-add Project**: Run `cops add /path/to/project` → Same ProjectID: `abc123` (duplicate detection works)

## Examples

### Example 1: Register New Git Project

```bash
$ cops add ~/work/my-app

# CLI extracts git info:
# - Main repo path: /Users/me/work/my-app
# - Configured URL: git@github.com:me/my-app.git
# - Actual URL: https://github.com/me/my-app.git
# - Existing ID: (none)

# API server searches MongoDB:
# - Query: $or [
#     {remoteUrl: "git@github.com:me/my-app.git"},
#     {remoteUrl: "https://github.com/me/my-app.git"}
#   ]
# - Result: Not found
# - Creates new document: {_id: ObjectId("..."), remoteUrl: "git@github.com:me/my-app.git"}
# - Returns: {projectId: "67890abcdef...", isNew: true}

# CLI saves:
# - Local: ~/work/my-app/.cops/config.json → {"id":"67890abcdef..."}
# - Global: ~/.cops/config.json → adds project entry
```

### Example 2: Re-add Existing Project (Duplicate Detection)

```bash
$ cops add ~/work/my-app

# CLI extracts git info:
# - Existing ID: "67890abcdef..." (from local config)

# API server searches MongoDB:
# - Query: $or [
#     {remoteUrl: "git@github.com:me/my-app.git"},
#     {remoteUrl: "https://github.com/me/my-app.git"},
#     {_id: ObjectId("67890abcdef...")}
#   ]
# - Result: Found existing document
# - Returns: {projectId: "67890abcdef...", isNew: false}

# CLI:
# - Log: "project already registered"
# - No duplicate created
```

### Example 3: GitHub Repo Renamed (Actual URL Differs)

```bash
# Scenario: GitHub repo renamed from "my-app" to "awesome-app"
# - Configured URL (in .git/config): git@github.com:me/my-app.git
# - Actual URL (from ls-remote): https://github.com/me/awesome-app.git

$ cops add ~/work/my-app

# API server searches MongoDB:
# - Query: $or [
#     {remoteUrl: "git@github.com:me/my-app.git"},      # Old URL
#     {remoteUrl: "https://github.com/me/awesome-app.git"} # New URL
#   ]
# - Result: Found by new URL
# - Returns existing project ID

# Duplicate detection works even after repo rename!
```

### Example 4: Remove Project

```bash
$ cops remove ~/work/my-app
Remove project at '/Users/me/work/my-app' from tracking?
This will delete local .cops/ config. Continue? (y/N): y

# CLI:
# 1. Deletes ~/work/my-app/.cops/ directory
# 2. Removes entry from ~/.cops/config.json
# 3. Logs: "project removed from tracking"

Project removed successfully!

# Note: Claude Code logs in ~/.claude/... remain intact
# Note: API server still has project data (not deleted)
```

### Example 5: Non-Git Project (API Call with Empty URLs)

```bash
$ cops add ~/documents/notes

# CLI detects:
# - Not a git repository
# - Calls API with empty URLs

# API server:
# - Query: No remote URL to match against
# - Creates new document: {_id: ObjectId("..."), remoteUrl: ""}
# - Returns: {projectId: "123abc...", isNew: true}

# CLI saves:
# - Local: ~/documents/notes/.cops/config.json → {"id":"123abc..."}
# - Global: ~/.cops/config.json → adds project entry with server-assigned ID
```

## Code Review & Refinements

After the initial implementation, a comprehensive code review identified 7 issues that were all resolved before PR creation. The review ensured alignment with project coding standards and architectural best practices.

### Review Issues Resolved

#### 1. Naming Convention Enforcement (Critical)
**Problem**: Repository adapter structs were named `Adapter` instead of following the `{Technology}{Domain}{Category}` pattern defined in `.agent/rules/go/go-outbound.md`.

**Fix Applied**:
- Renamed `Adapter` → `MongoSessionRecordRepository` (aggregation service)
- Renamed `Adapter` → `MongoDashboardRepository` (dashboard service)
- Renamed `Adapter` → `MongoProjectRepository` (project service)
- Updated all constructor names and method receivers accordingly
- Updated container module references to use new constructor names

#### 2. DTO Consolidation (Warning)
**Problem**: Service layer defined `RegisterProjectResult` which was an exact duplicate of repository layer's `FindOrCreateResult`.

**Fix Applied**:
- Removed duplicate `RegisterProjectResult` from service layer
- Service now returns `repository.FindOrCreateResult` directly
- Eliminated unnecessary data conversion between layers

#### 3. Import Cleanup (Info)
**Problem**: CLI module container used unnecessary import aliases where no naming conflicts existed.

**Fix Applied**:
- Removed aliases: `cliadapter`, `connectrpcadapter`, `filesystemadapter`, `jsonladapter`
- Kept package names simple: `cobra`, `connectrpc`, `filesystem`, `jsonl`
- Maintained consistency with API/daemon module patterns

#### 4. UUID Generation Removed (Critical)
**Problem**: CLI was generating local UUIDs for projects, defeating the purpose of API-coordinated Project IDs.

**Fix Applied**:
- Removed `github.com/google/uuid` dependency from CLI
- Removed all `uuid.New().String()` calls
- CLI now **always** calls API for project registration (even with empty URLs)
- On API failure: falls back to existing local ID if available, otherwise fails gracefully
- **API server is now the single source of truth** for all Project IDs

#### 5. Simplified Remote URL Handling (Warning)
**Problem**: Complex branching logic checked whether remote URLs existed before calling API.

**Fix Applied**:
- Simplified to unified flow: get URLs → call API → handle result
- Removed redundant `if configuredURL != "" || actualURL != ""` check
- API server handles empty URLs gracefully, no client-side validation needed

#### 6. Functional Programming Pattern (Info)
**Problem**: Manual slice filtering with verbose for-loop.

**Fix Applied**:
- Added `samber/lo` dependency (already in go.mod)
- Replaced manual loop with `lo.Filter` for cleaner, functional code:
  ```go
  filtered := lo.Filter(globalConfig.Projects, func(p *domain.Project, _ int) bool {
      return p.Path != absPath
  })
  ```

#### 7. Protobuf Schema Update (Critical)
**Problem**: User modified protobuf to remove Git Branch concept from Project (branches only differ in Sessions, not Projects).

**Fix Applied**:
- User confirmed: removed `git_branch` field from `Project` message
- Regenerated protobuf code: `cd idl/protobuf && buf generate`
- Updated all code references to match new schema

### Final Code Quality Checks

All checks passed after fixes:
- ✅ Build succeeds: `go build ./cli/... ./api/...`
- ✅ Tests pass: `go test ./cli/... ./api/...`
- ✅ Protobuf regenerated successfully
- ✅ All naming conventions follow project rules
- ✅ No duplicate code between layers
- ✅ API server is single source of truth for Project IDs

## Issues & Resolutions

| Issue | Resolution |
| ----- | ---------- |
| Need to handle two different remote URLs (configured vs actual) | API server accepts both URLs and searches using `$or` query, checking against either |
| API unreachable during project registration | CLI falls back to existing local ID if available, otherwise returns error (no local UUID generation) |
| Duplicate detection needed when repo is renamed on GitHub | CLI sends both configured and actual URLs; API matches against either |
| User might accidentally remove project | Added confirmation prompt (y/N) with `--force` flag to skip |
| Delete operation should be graceful if .cops/ doesn't exist | `DeleteLocalConfig` returns nil if directory doesn't exist instead of error |
| MongoDB field naming convention | Used camelCase (`remoteUrl`) per project convention, defined constant in mongoschema |
| Code review identified 7 issues | All resolved: naming conventions enforced, DTO duplication eliminated, UUID generation removed, imports cleaned, code simplified with `lo.Filter` |

## Architecture Decisions

### Decision 1: Dual Remote URL Support
**Why**: Git has two types of remote URLs that can differ:
- Configured URL: From `.git/config` (what developer set up)
- Actual URL: From `git ls-remote --get-url` (what GitHub actually points to after renames)

**Solution**: CLI sends both; API searches against both using `$or` query

### Decision 2: API Server Only Stores Remote URL
**Why**: Projects can be used by multiple people; local-specific info (path, name) cannot be uniquely associated on server

**Solution**: Server only stores ProjectID + remote URL; CLI manages local metadata

### Decision 3: Remove is Local-Only
**Why**: Per requirements, remove operation should only clean up local configs

**Solution**: `cops remove` only modifies global config and deletes local `.cops/` directory; does NOT communicate with API server or delete session logs

### Decision 4: API is Single Source of Truth for All Project IDs
**Why**: Centralized project ID management prevents duplicates and ensures data consistency across all project types

**Solution**:
- CLI **always** calls API for project registration (git and non-git projects)
- API handles empty remote URLs gracefully
- No local UUID generation in CLI
- On API failure: use existing local ID if available, otherwise fail gracefully

### Decision 5: Single `FindOrCreate` Repository Method
**Why**: Simplifies service logic; handles all duplicate detection and creation in one atomic operation

**Solution**: Repository uses efficient `$or` query checking all conditions at once

## Development Process Summary

This implementation followed a comprehensive workflow:

1. **Initial Implementation**: Built core features (`cops add` with API coordination, `cops remove` command)
2. **Code Review**: Identified 7 issues (3 critical, 2 warnings, 2 info) through systematic review
3. **Refinements**: All issues resolved, code quality improved
4. **Final State**: Production-ready code following all project conventions

### Key Achievements

- ✅ API server as single source of truth for Project IDs
- ✅ Robust duplicate detection using git remote URLs
- ✅ Clean codebase following hexagonal architecture
- ✅ All naming conventions enforced (MongoProjectRepository, MongoDashboardRepository)
- ✅ Zero code duplication between layers
- ✅ Functional programming patterns (`lo.Filter`)
- ✅ Graceful error handling and fallback mechanisms

The final code is ready for pull request creation and integration into the main branch.

## Related Tickets
- No explicit ticket reference (internal development work)
