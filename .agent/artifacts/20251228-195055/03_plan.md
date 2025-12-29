# Implementation Plan

## Overview

Fix the non-Git directory registration issue by modifying API validation to allow projects without search criteria when `isGitProject=false`, adding parent project detection with user prompt in CLI, and implementing hierarchical project matching in Daemon for correct log routing.

## Selected Packages

| Problem | Package | Context7 ID | Reason for Selection |
| ------- | ------- | ----------- | -------------------- |
| No new packages required | - | - | All functionality achievable with existing dependencies |

## Architecture Decisions

### Decision 1: API Validation Strategy for Non-Git Projects
**Choice**: For non-Git projects (`IsGitProject == false`) without `existingID`, skip search criteria validation and directly create a new project. For Git projects, maintain existing URL-based duplicate detection.
**Rationale**: Per requirements, non-Git projects have no natural deduplication key (no remote URL). The only deduplication happens via local config (`existingID`). Creating new projects directly is the intended behavior.

### Decision 2: Parent Project Detection Location
**Choice**: Implement parent detection in the **Service layer** (`tracking_service.go`) as a new method. The TUI calls this service method and receives the result, then handles the user prompt based on the result.
**Rationale**: Following Hexagonal Architecture, the Inbound layer (TUI) should not directly access Outbound ports (ConfigPort). The Service layer acts as the mediator - it uses the ConfigPort to load global config and provides processed results to the TUI. This maintains proper layer separation.

**Flow**:
1. TUI calls `Service.FindParentProject(path string) (*ParentProjectInfo, error)`
2. Service uses `ConfigPort.LoadGlobalConfig()` internally
3. Service returns parent project info (or nil if none found)
4. TUI displays prompt based on result and handles user input

### Decision 3: Hierarchical Matching Implementation Location
**Choice**: Implement hierarchical matching in Daemon's `logwatcher/log_service.go` by modifying `GetProjectIDForClaudeDir` to use longest-path-prefix matching.
**Rationale**: The log service already maintains the `claudeDirToProject` mapping. Adding hierarchical matching here keeps the logic centralized where project routing decisions are made.

### Decision 4: Path Storage in Daemon Mapping
**Choice**: Store both `ClaudeDir` and `ProjectPath` in the mapping to enable hierarchical matching. Add a new `projectPathToID` map alongside existing `claudeDirToProject`.
**Rationale**: Claude directories are encoded paths (e.g., `-Users-jayce-project`), but hierarchical matching requires comparing original file system paths. We need to track both.

## Implementation Steps

### Step 1: Fix API Validation for Non-Git Projects

**File**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`

**Function Signature**:
```go
func (r *MongoProjectRepository) FindOrCreate(ctx context.Context, params repository.FindOrCreateParams) (*repository.FindOrCreateResult, error)
```

**Logic**:
1. If `existingID` is provided and valid ObjectID:
   - Search MongoDB by `_id` field
   - If found, return existing project (isNew=false)
   - If not found (ErrNoDocuments), continue to next step
   - If other error, return error
2. If `IsGitProject == true`:
   - Build $or conditions from configuredURL and actualURL (if non-empty)
   - If conditions exist, search by URL
   - If found, return existing project (isNew=false)
   - If not found, continue to creation
3. Create new project document:
   - Set remoteUrl (prefer configuredURL, fallback to actualURL, empty for non-Git)
   - Set name, isGitProject, registeredAt
   - Insert into MongoDB
   - Return new project (isNew=true)

**Key Change**: Remove the validation that requires at least one search criterion. Non-Git projects without existingID should proceed directly to step 3.

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Non-Git project, no existingID | `{IsGitProject: false, Name: "test"}` | New project created | Non-Git direct creation |
| Non-Git project with existingID | `{IsGitProject: false, ExistingID: "valid-id"}` | Existing project returned | ExistingID lookup |
| Non-Git project with invalid existingID | `{IsGitProject: false, ExistingID: "not-found-id"}` | New project created | ExistingID not found fallback |
| Git project with URL | `{IsGitProject: true, ConfiguredURL: "url"}` | Existing or new based on URL | URL-based dedup |
| Git project, URL not found | `{IsGitProject: true, ConfiguredURL: "new-url"}` | New project created | Git project creation |

---

### Step 2: Add Parent Detection Method to Service Layer

**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`

**New Struct**:
```go
type ParentProjectInfo struct {
    ID   domain.ID
    Name string
    Path string
}
```

**New Function Signature**:
```go
func (s *Service) FindParentProject(targetPath string) (*ParentProjectInfo, error)
```

**Logic**:
1. Expand and validate targetPath to absolute path
2. Load global config via `s.configRepo.LoadGlobalConfig()`
3. Walk up directory tree from parent of targetPath:
   - For each directory level, check if any registered project has matching Path
   - If match found, return ParentProjectInfo with project's ID, Name, Path
   - Continue to parent directory until reaching root ("/")
4. If no parent found, return nil (not an error)

---

### Step 3: Add Parent Detection Step to CLI TUI

**Files**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui.go`
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go`
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go`

**Changes to `add_tui.go`**:

New step constant:
```go
const (
    stepParentDetection = iota  // NEW: first step
    stepGitSelection
    stepNameInput
    stepSyncSelection
    stepCompleted
)
```

New field in addModel:
```go
type addModel struct {
    // ... existing fields ...
    parentProject *tracking.ParentProjectInfo  // Result from service
    parentCursor  int                          // 0=Yes, 1=No
    service       *tracking.Service            // Reference to service for parent detection
}
```

Updated function signatures:
```go
func newAddModel(dir string, noGitFlag bool, service *tracking.Service) addModel
func runAddTUI(dir string, noGitFlag bool, service *tracking.Service) (*addTUIResult, error)
```

New message type:
```go
type parentDetectionMsg struct {
    parent *tracking.ParentProjectInfo
    err    error
}
```

New command function:
```go
func (m addModel) detectParentProject() tea.Msg
```
**Logic**: Call `m.service.FindParentProject(m.currentDir)` and return result as parentDetectionMsg

**Changes to `add_tui_update.go`**:

New case in Update():
```go
case parentDetectionMsg:
```
**Logic**: Store parent in model, if nil proceed to stepGitSelection, else stay on stepParentDetection for user prompt

New handler function:
```go
func (m addModel) updateParentSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd)
```
**Logic**: Handle up/down for cursor, enter/y/n for selection. If "Yes" proceed to git detection, if "No" cancel.

**Changes to `add_tui_view.go`**:

New case in View():
```go
case stepParentDetection:
```
**Logic**: If parentProject is nil show "Checking...", else render confirmation prompt

New view function:
```go
func (m addModel) viewParentConfirmation() string
```
**Logic**: Display parent project name and path, show Yes/No options with cursor

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| No parent project | `/new/path` (no parent in config) | Skip to git detection | No parent found |
| Parent project exists, user confirms | `/parent/child` with `/parent` registered | Proceed to git detection | Parent confirmed |
| Parent project exists, user cancels | `/parent/child` with `/parent` registered | Exit with Cancelled=true | Parent rejected |
| Deep nested path | `/a/b/c/d` with `/a` registered | Show `/a` as parent | Parent detection walk |

---

### Step 4: Update Add Command to Pass Service to TUI

**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go`

**Verification Required**: Read this file first to understand current structure.

**Expected Changes**:

Update runAddTUI call:
```go
func runAddTUI(dir string, noGitFlag bool, service *tracking.Service) (*addTUIResult, error)
```

**Logic**: The handler already has access to the Service. Pass it to runAddTUI instead of (or in addition to) any existing parameters.

---

### Step 5: Implement Hierarchical Project Matching in Daemon

**File**: `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go`

**Updated struct**:
```go
type Service struct {
    // ... existing fields ...
    projectPathToID map[string]shareddomain.ID  // NEW: ProjectPath -> ProjectID mapping
}
```

**Updated NewService**:
```go
func NewService(...) *Service
```
**Logic**: Initialize `projectPathToID` as empty map

**Updated function**:
```go
func (s *Service) UpdateTargets(targets []domain.WatchTarget) error
```
**Logic**: In addition to existing claudeDirToProject mapping, also build projectPathToID mapping from targets (using t.ProjectPath -> t.ProjectID)

**Updated function**:
```go
func (s *Service) GetProjectIDForClaudeDir(claudeDir string) shareddomain.ID
```
**Logic**:
1. Try exact match in claudeDirToProject map
2. If not found, decode claudeDir to original path using pathutil.DecodeClaudeProjectDir()
3. If decode fails, return empty ID
4. Iterate projectPathToID map, find all paths that are prefixes of the decoded path
5. Select the longest matching prefix (most specific project)
6. Return the ProjectID for that match, or empty ID if no match

---

### Step 6: Add Path Decode Function to Daemon Pathutil

**File**: `/Users/jayce/team-attention/cops/daemon/internal/platform/util/pathutil/pathutil.go`

**New function signature**:
```go
func DecodeClaudeProjectDir(claudeDir string) string
```

**Logic**:
1. Get user home directory
2. Build expected prefix: `~/.claude/projects/`
3. If claudeDir doesn't start with prefix, return empty string
4. Extract the encoded portion after prefix
5. Replace all `-` with `/` (reverse of encoding)
6. Ensure result starts with `/`
7. Return decoded path

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Valid Claude dir | `~/.claude/projects/-Users-jayce-project` | `/Users/jayce/project` | Normal decode |
| Invalid prefix | `/some/other/path` | `""` (empty) | Invalid prefix |
| Empty encoded | `~/.claude/projects/` | `""` (empty) | Empty encoded part |

---

### Step 7: Hierarchical Matching Test Scenarios

| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Exact match | ClaudeDir for `/home/user/project` | ProjectID of `/home/user/project` | Exact match |
| Child directory log | ClaudeDir for `/home/user/project/subdir` with only `/home/user/project` registered | ProjectID of `/home/user/project` | Hierarchical fallback |
| Nested projects | ClaudeDir for `/a/b/c`, registered: `/a` and `/a/b/c` | ProjectID of `/a/b/c` (most specific) | Longest prefix match |
| No match | ClaudeDir for `/unregistered/path` | Empty ID | No match fallback |

## Execution Order

1. **Step 1: Fix API Validation** (no dependencies)
   - Most critical fix - unblocks non-Git project registration

2. **Step 5 + Step 6: Implement Hierarchical Matching in Daemon** (no dependencies)
   - Can be done in parallel with Step 1
   - Step 6 (decode function) must be done before Step 5

3. **Step 2: Add FindParentProject to Service** (no dependencies)
   - Can be done in parallel with Steps 1 and 5

4. **Step 4: Read Add Command Structure** (no dependencies)
   - Verification step before TUI changes

5. **Step 3: Add Parent Detection to TUI** (depends on Steps 2 and 4)
   - Requires service method from Step 2
   - Requires understanding of add.go structure from Step 4

## Notes for Execute Agent

1. **API Changes (Step 1)**:
   - Remove the validation at lines 54-57 that returns error for empty conditions
   - Restructure to check existingID first, then URL for git projects, then create

2. **Service Layer (Step 2)**:
   - ParentProjectInfo struct should be exported (capital P)
   - Method uses existing configRepo dependency, no new dependencies needed

3. **TUI Changes (Steps 3-4)**:
   - Import tracking package for ParentProjectInfo type
   - Step constants shift: stepGitSelection becomes 1 instead of 0
   - Init() should return detectParentProject command instead of detectGitRepos

4. **Daemon Changes (Steps 5-6)**:
   - Import `strings` for HasPrefix in hierarchical matching
   - Path prefix matching must check for separator to avoid `/a/bc` matching `/a/b`
   - Use `strings.HasPrefix(path, prefix + "/")` or exact equality check

5. **Build Verification**:
   - After each step, run `go build ./...` in the relevant module
   - API: `cd api && go build ./...`
   - CLI: `cd cli && go build ./...`
   - Daemon: `cd daemon && go build ./...`
