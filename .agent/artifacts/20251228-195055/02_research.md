# Research Report

## Mode
General Research

## Request Summary
Research the C-Ops codebase to understand the current implementation before fixing the non-Git directory registration issue. This includes understanding the CLI registration flow, API validation, daemon session log routing, data models, and configuration management.

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File | Reason |
|------|--------|
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go` | Core registration logic - shows how AddProject handles git vs non-git projects |
| `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go:35-57` | Contains the validation that causes "no search criteria provided" error |
| `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/project_repo_port.go` | FindOrCreateParams - current search criteria structure |
| `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/configwatcher_service.go` | Daemon config watcher - needs hierarchical matching implementation |
| `/Users/jayce/team-attention/cops/daemon/internal/platform/domain/watch.go` | WatchTarget structure - may need extending for hierarchical matching |
| `/Users/jayce/team-attention/cops/shared/domain/project.go` | Domain model - Path field location analysis |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-struct.md` | Rules for pointer vs value types |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-hexagonal-layout.md` | Architecture rules for shared vs local models |

## Package Candidates

No new packages required for this fix. The issue is about modifying existing validation logic, not adding new dependencies.

## Technical Constraints
- Non-git projects have no remote URL, so the current search criteria (configuredURL, actualURL) are empty
- API validation at `project_repo.go:55-57` requires at least one search criterion
- API should NOT receive path - path is only used by Daemon for log routing
- For non-git projects, API should create new projects without deduplication (per requirements AC)
- Daemon needs hierarchical matching to route logs to most specific project
- Backward compatibility with existing git project registrations must be maintained

## Analysis of Current Implementation

### CLI Registration Flow (`cli/internal/service/tracking/`)

**File: `tracking_service.go`**

The `AddProject` function (lines 53-188) handles project registration:

1. **Path Resolution** (lines 55-79):
   - Expands and validates the path using `pathutil.ExpandPath`
   - Checks if it's a git repo using `gitutil.IsGitRepo`
   - If git repo, finds main repo path (not worktree) using `gitutil.FindMainRepoPath`
   - Sets `isGitProject = true` for git repos, `false` otherwise

2. **Name Resolution** (lines 82-85):
   - Uses provided name or falls back to directory basename

3. **Project ID Determination** (lines 88-98):
   - Checks if local config exists at `{projectPath}/.cops/config.json`
   - If exists, reads `existingProjectID` from it

4. **URL Collection** (lines 100-106):
   - URLs are only collected if `isGitProject == true`
   ```go
   if isGitProject {
       configuredURL, _ = gitutil.GetRemoteURL(projectPath)
       actualURL = gitutil.GetActualRemoteURL(projectPath)
   }
   ```
   - For non-git projects, both URLs are empty strings

5. **API Registration** (lines 108-132):
   - Calls `s.project.RegisterProject` with `RegisterProjectParams`:
     - `ConfiguredRemoteURL`: empty for non-git
     - `ActualRemoteURL`: empty for non-git
     - `ExistingProjectID`: only set if local config exists
     - `Name`: project name
     - `IsGitProject`: false for non-git
   - If API fails and no existing ID, registration fails completely

6. **Config Persistence** (lines 134-171):
   - Saves local config with projectID
   - Adds to global registry if not already present

**File: `outbound/api/project_port.go`**

```go
type RegisterProjectParams struct {
    ConfiguredRemoteURL string
    ActualRemoteURL     string
    ExistingProjectID   string
    Name                string
    IsGitProject        bool
}
```

**File: `outbound/api/connectrpc/project_client.go`**

Maps `RegisterProjectParams` to protobuf `RegisterProjectReq` fields.

### API Validation & Registration (`api/internal/service/project/`)

**File: `outbound/repository/mongodb/project_repo.go`**

The `FindOrCreate` function (lines 35-118) handles duplicate detection:

1. **Search Criteria Building** (lines 37-52):
   - Adds `configuredURL` to conditions if non-empty
   - Adds `actualURL` to conditions if non-empty and different from configured
   - Adds `existingID` to conditions if valid ObjectID

2. **Validation** (lines 54-57):
   ```go
   if len(conditions) == 0 {
       return nil, fmt.Errorf("no search criteria provided: at least one of configuredURL, actualURL, or existingID must be valid")
   }
   ```
   **THIS IS THE ROOT CAUSE OF THE ERROR**

3. **Document Creation** (lines 87-98):
   - When creating new project, uses `configuredURL` or `actualURL` for `remoteUrl` field
   - For non-git projects, this would be empty string

**File: `outbound/repository/project_repo_port.go`**

```go
type FindOrCreateParams struct {
    ConfiguredURL string
    ActualURL     string
    ExistingID    string
    Name          string
    IsGitProject  bool
}
```

**File: `project_service.go`**

Simple passthrough to repository - no additional validation.

### Daemon Session Log Routing (`daemon/internal/`)

**File: `service/logwatcher/log_service.go`**

The daemon tracks projects via `claudeDirToProject` mapping (line 26):
```go
claudeDirToProject map[string]shareddomain.ID
```

- `UpdateTargets` (lines 50-98): Updates which directories to watch
- `GetProjectIDForClaudeDir` (lines 207-212): Looks up project ID for a Claude directory
- Uses encoded path format: `/Users/jayce/project` -> `~/.claude/projects/-Users-jayce-project`

**File: `service/configwatcher/configwatcher_service.go`**

- `buildWatchTargets` (lines 96-157): Builds watch targets from global config
- For each project, creates `WatchTarget` with `ClaudeDir` pointing to Claude's encoded path
- Uses `pathutil.GetClaudeProjectDir(project.Path)` to get Claude directory
- **Currently does NOT implement hierarchical matching** - uses exact path match only

**File: `platform/util/pathutil/pathutil.go`**

```go
func GetClaudeProjectDir(projectPath string) string {
    encoded := EncodePathForClaude(projectPath)
    return filepath.Join(home, ".claude", "projects", encoded)
}
```

### Data Models

**File: `shared/domain/project.go`**

```go
type ProjectAbstract struct {
    ID   ID     `json:"id"`
    Name string `json:"name"`
    Path string `json:"path"`  // <-- Path is in shared model
}

type Project struct {
    ProjectAbstract
    IsGitProject bool      `json:"gitProject"`
    RegisteredAt time.Time `json:"registeredAt"`
}
```

**File: `shared/domain/mongoschema/project.go`**

```go
const (
    ProjectIDField           = "_id"
    ProjectNameField         = "name"
    ProjectPathField         = "path"
    ProjectIsGitProjectField = "isGitProject"
    ProjectRegisteredAtField = "registeredAt"
    ProjectGitBranchField    = "git_branch"
    ProjectRemoteURLField    = "remoteUrl"
)
```

**File: `idl/protobuf/project/v1/project.proto`**

```protobuf
message RegisterProjectReq {
  string configured_remote_url = 1;
  string actual_remote_url = 2;
  string existing_project_id = 3;
  string name = 4;
  bool is_git_project = 5;
}
```
Note: No `path` field - and **this is correct** as path is only used by Daemon.

### Configuration Management

**Global Config: `~/.cops/config.json`**
```json
{
  "projects": [
    {
      "id": "...",
      "name": "...",
      "path": "/absolute/path/to/project",
      "gitProject": true,
      "registeredAt": "..."
    }
  ]
}
```

**Local Config: `{projectPath}/.cops/config.json`**
```json
{
  "id": "mongodb-objectid-hex"
}
```

**Implementation: `cli/internal/service/tracking/outbound/config/filesystem/filesystem_config.go`**

- `LoadGlobalConfig`: Reads from `~/.cops/config.json`
- `SaveGlobalConfig`: Writes to `~/.cops/config.json`
- `LoadLocalConfig`: Reads from `{projectPath}/.cops/config.json`
- `SaveLocalConfig`: Writes to `{projectPath}/.cops/config.json`
- `LocalConfigExists`: Checks if local config exists

## Root Cause Analysis

The error "no search criteria provided" occurs because:

1. Non-git project has:
   - `configuredURL = ""` (empty)
   - `actualURL = ""` (empty)
   - `existingID = ""` (no local config on first registration)

2. All three search criteria are empty, triggering the validation error at `project_repo.go:55-57`

3. The current design assumes every project has either:
   - A git remote URL, OR
   - An existing project ID from a previous registration

4. Non-git projects on first registration have neither.

**Key Insight from Requirements**: For non-git projects, the API should simply **create a new project** without requiring any search criteria. Per the requirements AC:
- "API accepts registration requests with only `name` and `isGitProject=false` (no remote URLs required)"
- "API creates new projects for non-Git directories without requiring search criteria"
- "MongoDB repository handles non-Git projects by creating new documents when no `existingID` is provided"

This means no deduplication by path is needed at the API level - the only deduplication for non-git projects happens via local config (`existingID`).

## Path Field Usage Analysis

### Where Path is Currently Used

| Location | Usage | Should Path Be There? |
|----------|-------|----------------------|
| `shared/domain/project.go` | `ProjectAbstract.Path` field | **Debatable** - used for config serialization |
| `cli/internal/service/tracking/tracking_service.go` | Reads from domain model, saves to global config | Yes - CLI manages paths |
| `daemon/internal/service/configwatcher/configwatcher_service.go` | Reads `project.Path` from global config | Yes - Daemon needs paths for routing |
| `api/internal/service/dashboard/outbound/repository/mongodb/dashboard_repo.go` | Reads `ProjectPathField` from MongoDB | **Problem** - API reads path but doesn't store it |

### The Path Field Problem

1. **Shared Model** (`shared/domain/project.go`): Contains `Path` field
2. **MongoDB Schema** (`shared/domain/mongoschema/project.go`): Has `ProjectPathField = "path"` constant
3. **Dashboard Repo**: Tries to read `path` from MongoDB (lines 173, 234, 275, 578)
4. **Project Repo** (`project_repo.go`): Does NOT store path when creating projects

**Issue**: Path is in the shared model for config serialization (used by CLI and Daemon), but API's dashboard repo tries to read it from MongoDB where it's never stored.

### Recommended Architecture

Per user feedback and hexagonal architecture principles:

1. **Shared Model should be minimal** - only what's needed across components
2. **Path is only used locally** (CLI for config, Daemon for routing) - NOT transmitted to API
3. **Daemon should extend shared model** with path-specific logic

**Option A**: Keep Path in shared model for config serialization convenience, but:
- API should NOT expect it in MongoDB
- Daemon reads it from local config, not from API

**Option B**: Remove Path from shared model, have CLI and Daemon define their own extended models:
- More work but cleaner separation
- CLI model includes Path for config
- Daemon model includes Path for routing

For this fix, **Option A** is simpler and aligns with current architecture. The key fix is in API validation, not model restructuring.

## Corrected Solution Approach

Based on requirements AC and user feedback:

### 1. API Changes (Main Fix)

Modify `api/internal/service/project/outbound/repository/mongodb/project_repo.go` to:
- For non-git projects (`IsGitProject == false`): Skip search criteria validation and create new project directly
- For git projects: Continue using existing URL-based duplicate detection
- If `existingID` is provided (for either type): Always search for it first

**New Logic Flow**:
```
if existingID is valid:
    search by existingID
    if found: return existing

if isGitProject:
    if configuredURL or actualURL is provided:
        search by URLs
        if found: return existing
    else:
        return error (git project must have URL)

// No existing found or non-git project
create new project
```

### 2. No Protobuf Changes Needed

The current protobuf schema is sufficient:
- `name` and `is_git_project=false` are enough for non-git projects
- Path is NOT sent to API - it's only used by Daemon locally

### 3. CLI Changes

Add parent project detection (per requirements AC):
- Before registering, check if any parent directory is already in global config
- If found, prompt user: "This directory is already being tracked as a subdirectory of [parent]. Do you want to register it as a separate project?"
- User must explicitly confirm to proceed

### 4. Daemon Changes

Implement hierarchical project matching:
- When processing log files, find most specific matching project
- Use longest path prefix matching (similar to CIDR)
- Example: If `/home/user/repo` and `/home/user/repo/subdir` are both registered, logs in `/home/user/repo/subdir/` should route to the subdir project

## Similar Implementations Found

### Example 1: Git project registration flow
- **File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go:64-75`
- **Relevance**: Shows the git vs non-git branching logic

### Example 2: MongoDB FindOrCreate with conditions
- **File**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go:35-57`
- **Relevance**: Current validation logic that needs modification

### Example 3: Daemon watch target building
- **File**: `/Users/jayce/team-attention/cops/daemon/internal/service/configwatcher/configwatcher_service.go:96-157`
- **Relevance**: Needs hierarchical matching implementation

## Additional Information for Planning

### Files Requiring Changes (in order)

1. **API Repository** (main fix):
   - `api/internal/service/project/outbound/repository/mongodb/project_repo.go`
   - Modify `FindOrCreate` to handle non-git projects without search criteria

2. **CLI TUI** (parent detection):
   - `cli/internal/service/tracking/inbound/cli/cobra/add_tui.go`
   - Add parent project detection step
   - `cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go`
   - Handle parent detection result
   - `cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go`
   - Render parent detection prompt

3. **CLI Service** (support parent detection):
   - `cli/internal/service/tracking/tracking_service.go`
   - May need helper method for checking parent projects

4. **Daemon Config Watcher** (hierarchical matching):
   - `daemon/internal/service/configwatcher/configwatcher_service.go`
   - Modify `buildWatchTargets` or add hierarchical lookup
   - `daemon/internal/service/logwatcher/log_service.go`
   - Use hierarchical matching in `GetProjectIDForClaudeDir`

### No Changes Needed

- `idl/protobuf/project/v1/project.proto` - current schema is sufficient
- `shared/domain/project.go` - Path field stays for config serialization
- `shared/domain/mongoschema/project.go` - constants are fine

### Test Scenarios

1. Register non-git directory (first time) - should succeed with new project ID
2. Register same non-git directory again (local config exists) - should return existing project
3. Register non-git directory that was moved - should create new project
4. Register git repo - should still work via remote URL matching
5. Register subdirectory of existing project - should prompt user, allow if confirmed
6. Daemon logs routing with nested projects - should route to most specific match
