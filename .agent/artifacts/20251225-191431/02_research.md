# Research Report

## Mode
General Research

## Request Summary

Add two CLI features to improve project management:
1. A `cops remove` command to unregister projects by removing them from global config and deleting local `.cops/` directory
2. Modify `cops add` to coordinate with the API server to obtain ProjectIDs instead of generating arbitrary UUIDs locally, with intelligent duplicate detection based on git repository information

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File | Reason |
|------|--------|
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go` | Contains existing `AddProject` and `RemoveProject` service methods to be modified |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go` | Current `cops add` command implementation pattern |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/handler.go` | CLI handler pattern showing how to add new commands |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/config/config_port.go` | ConfigPort interface - needs `DeleteLocalConfig` method |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/config/filesystem/filesystem_config.go` | Filesystem config adapter - needs delete implementation |
| `/Users/jayce/team-attention/cops/cli/internal/platform/util/gitutil/gitutil.go` | Existing git utilities for worktree/remote detection |
| `/Users/jayce/team-attention/cops/idl/protobuf/collector/v1/collector.proto` | Reference for protobuf service definition patterns |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/collector_port.go` | CollectorPort pattern for API communication |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/collector_client.go` | ConnectRPC client implementation pattern |
| `/Users/jayce/team-attention/cops/.agent/rules/idl/protobuf.md` | Protobuf naming conventions |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound.md` | Inbound handler structure rules |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-outbound.md` | Outbound adapter structure rules |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md` | Service implementation patterns |

## Technical Constraints

1. **No Project Registration Service Exists**: The API server currently lacks a project registration service. A new protobuf service (`project/v1/project.proto`) and corresponding API service must be created.

2. **CLI Uses `dig` for DI**: Unlike the API server which uses `fx`, the CLI uses `go.uber.org/dig` for dependency injection.

3. **Protobuf Naming Convention**: Request messages use `Req` suffix, response messages use `Res` suffix (not `Request`/`Response`).

4. **Git Utilities Already Exist**: `gitutil` package provides:
   - `IsGitRepo(dir string) bool`
   - `FindMainRepoPath(dir string) (string, error)`
   - `ListWorktrees(mainRepoPath string) ([]string, error)`
   - `GetCurrentBranch(repoPath string) (string, error)`
   - `GetRemoteURL(repoPath string) (string, error)`

5. **Config Structure**:
   - Global: `~/.cops/config.json` with `GlobalConfig{Projects: []*domain.Project}`
   - Local: `{project}/.cops/config.json` with `LocalConfig{ID: domain.ID}`

6. **Domain ID Type**: Use `domain.ID` (string alias) for project identifiers, created via `domain.NewID(string)`.

7. **API Server Port**: CLI config has both `Collector.URL` (default `http://localhost:8080`) and `API.URL` (default `http://localhost:8081`).

## Package Candidates

### Problem 1: User Confirmation Prompt

| Package | Context7 ID | Why Better Than Alternatives |
|---------|-------------|------------------------------|
| promptui | `/manifoldco/promptui` | Interactive prompts with validation, widely used in CLI tools |
| survey (AlecAivazis) | `/go-survey/survey` | Rich survey functionality but may be overkill for simple y/n |
| bufio (stdlib) | (stdlib) | No dependencies, simple for basic y/n confirmation |

**Recommendation**: Use `bufio` from stdlib for simple y/n confirmation - no external dependency needed for basic confirmation.

### Problem 2: ConnectRPC Client (Already in Use)

| Package | Context7 ID | Why Better Than Alternatives |
|---------|-------------|------------------------------|
| connectrpc | `/connectrpc/connect-go` | Already used in project, provides HTTP/2 + gRPC compatibility |

**Recommendation**: Continue using `connectrpc.com/connect` as already established in the codebase.

## Similar Implementations Found

### Example 1: Add Command Implementation
- **File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go:13-66`
- **Relevance**: Shows how to create a new cobra command with flags and call service methods

### Example 2: Tracking Service
- **File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go:52-163`
- **Relevance**: Shows `AddProject` method structure, config handling, and project ID generation (currently uses UUID - needs to be changed to API call)

### Example 3: RemoveProject Method (Partial - by ID only)
- **File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go:198-226`
- **Relevance**: Existing `RemoveProject(ctx, projectID)` method removes from global config only - needs extension for path-based removal and local config deletion

### Example 4: Collector ConnectRPC Client
- **File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/api/connectrpc/collector_client.go:25-112`
- **Relevance**: Pattern for creating ConnectRPC client, sending requests, and handling responses

### Example 5: Git Remote URL Extraction
- **File**: `/Users/jayce/team-attention/cops/cli/internal/platform/util/gitutil/gitutil.go:117-125`
- **Relevance**: `GetRemoteURL` function already exists and can be used for duplicate detection

### Example 6: Config Port Interface
- **File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/outbound/config/config_port.go:16-31`
- **Relevance**: Shows interface pattern for config operations - needs `DeleteLocalConfig(projectPath string) error` method

### Example 7: API Service Handler Pattern
- **File**: `/Users/jayce/team-attention/cops/api/internal/service/aggregation/inbound/grpc/connectrpc/handler.go`
- **Relevance**: Pattern for implementing ConnectRPC service handlers in API server

## Additional Information for Planning

### Current AddProject Flow (to be modified)
1. Expand and validate path
2. Check git status and find main repo path
3. Use directory name as project name
4. **Check local config for existing ProjectID** - If exists, use it
5. **Generate new UUID if not exists** - **This needs to change to API call**
6. Save local config with ID
7. Add to global config if not already registered

### New AddProject Flow (proposed)
1. Expand and validate path
2. Check git status, find main repo path, extract git remote URLs
3. Use directory name as project name
4. Check local config for existing ProjectID
5. **Call API RegisterProject with all collected info**
6. **Receive ProjectID from API (new or existing)**
7. Save ProjectID to local config
8. Add/update global config

### New RemoveProject Flow (proposed)
1. Expand and validate path
2. Display project information for confirmation
3. Prompt for user confirmation (y/n)
4. Delete `.cops/` directory from project path
5. Remove project entry from global config

### API Server Changes Required

1. **New Protobuf Service**: `idl/protobuf/project/v1/project.proto`
   - `RegisterProjectReq`: Contains path, name, git info (remote URLs, worktree status), existing local ID
   - `RegisterProjectRes`: Contains assigned ProjectID, synchronized metadata
   - `ProjectService.RegisterProject`: RPC method

2. **New API Service**: `api/internal/service/project/`
   - Service layer with project registration logic
   - Repository for project storage (MongoDB)
   - ConnectRPC handler implementing ProjectService

3. **Duplicate Detection Logic** (API Server):
   - Match by git remote URL (primary)
   - Match by existing local ProjectID (secondary)
   - Return existing project if match found

### CLI Changes Required

1. **New Remove Command**: `cli/internal/service/tracking/inbound/cli/cobra/remove.go`
   - Accept project path argument
   - Prompt for confirmation
   - Call service RemoveProjectByPath method

2. **Modified Add Command**: Update to use API for ProjectID

3. **New Project Registration Port**: `cli/internal/service/tracking/outbound/api/project_port.go`
   - Interface for project registration API calls

4. **New Project Registration Client**: `cli/internal/service/tracking/outbound/api/connectrpc/project_client.go`
   - ConnectRPC client implementing project registration

5. **Extended ConfigPort**: Add `DeleteLocalConfig(projectPath string) error` method

6. **Extended Filesystem Adapter**: Implement `DeleteLocalConfig` using `os.RemoveAll`

### GitInfo Structure (for API communication)

```go
type GitInfo struct {
    IsGitProject    bool     `json:"isGitProject"`
    IsWorktree      bool     `json:"isWorktree"`
    MainRepoPath    string   `json:"mainRepoPath,omitempty"`
    RemoteURL       string   `json:"remoteUrl,omitempty"`
    CurrentBranch   string   `json:"currentBranch,omitempty"`
}
```

### Error Handling

- API unreachable: Return clear error, do not fall back to UUID
- Project directory doesn't exist: Skip local config deletion gracefully
- Project not in global config: Skip removal gracefully

### Module Registration

For CLI, new outbound adapter must be registered in `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_tracking.go` using `dig.As` pattern.
