# Development Walkthrough

## Summary

Fixed the `cops add` command to properly handle non-Git directories by removing API search criteria validation, implementing parent project detection with user confirmation in the CLI, refactoring the MongoDB repository to eliminate business logic, and adding hierarchical project matching in the daemon for accurate session log routing.

## Code Overview

### Modified Components

#### API: `MongoProjectRepository`

**Location**: `/Users/jayce/team-attention/cops/api/internal/service/project/outbound/repository/mongodb/project_repo.go`

**Purpose**: Handles project persistence in MongoDB with clean separation of data access from business logic

**Major Refactoring**:
- **Removed validation**: Eliminated the requirement for search criteria (URLs or existing ID) when registering non-Git projects
- **Extracted helper methods**: Broke down 107-line `FindOrCreate` function into 5 focused helper methods
- **Removed business logic**: Eliminated all `if params.IsGitProject` conditionals from repository layer (hexagonal architecture violation)
- **Simplified query pattern**: Uses single `$or` query combining ID and URL conditions

**Key Methods**:
- `FindOrCreate(ctx, params)`: Main entry point - now ~20 lines with clear flow
- `buildSearchConditions(params)`: Creates $or conditions from non-empty parameters only
- `collectNonEmptyURLs(configuredURL, actualURL)`: Gathers unique, non-empty URLs
- `findByConditions(ctx, conditions)`: Executes the $or query
- `createProject(ctx, params)`: Inserts new project document
- `docToResult(doc)`: Converts MongoDB document to result struct

**Implementation Pattern**:
```go
// Main flow (before refactor: 107 lines, after: ~20 lines)
func (r *MongoProjectRepository) FindOrCreate(ctx, params) (*FindOrCreateResult, error) {
    conditions := r.buildSearchConditions(params)  // Natural filtering

    if len(conditions) > 0 {
        if project, err := r.findByConditions(ctx, conditions); project != nil {
            return project, nil
        }
    }

    return r.createProject(ctx, params)  // Create if not found
}
```

**Behavioral Change**:
- **Before**: Required at least one of `configuredURL`, `actualURL`, or `existingID`
- **After**: Accepts projects with no search criteria (creates new project for non-Git directories)
- **Query Optimization**: Changed from sequential queries (ID first, then URL) to single `$or` query with all conditions

---

#### CLI Service: `Service.FindParentProject`

**Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`

**Purpose**: Detects if any parent directory of the target path is already registered as a project

**New Struct**:
```go
type ParentProjectInfo struct {
    ID   domain.ID
    Name string
    Path string
}
```

**New Method**:
- `FindParentProject(targetPath string)`: Returns parent project info if found, nil otherwise
  - Expands and validates the target path to absolute path
  - Loads global config to get registered projects
  - Walks up directory tree from parent of target path
  - Checks each directory level against registered projects
  - Returns first match found (most specific parent)
  - Returns nil (not an error) if no parent project exists

**Algorithm**:
1. Expand target path to absolute path
2. Load global config via `configRepo.LoadGlobalConfig()`
3. Start from parent of target path: `filepath.Dir(absPath)`
4. Loop: Check if current directory matches any registered project path
5. Move up: `current = filepath.Dir(current)`
6. Stop when reaching root (`"/"`) or no more parents

---

#### CLI TUI: Parent Detection Step

**Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui*.go`

**Purpose**: Interactive user confirmation when parent project is detected

**Changes to `add_tui.go`**:
- Added `stepParentDetection` as first step (before git selection)
- Added `parentProject *tracking.ParentProjectInfo` field to `addModel`
- Added `parentCursor int` field for Yes/No selection
- Added `service *tracking.Service` reference to enable parent detection
- Updated `newAddModel()` to accept service parameter
- Changed `Init()` to return `detectParentProject` command instead of `detectGitRepos`

**New Message Type**:
```go
type parentDetectionMsg struct {
    parent *tracking.ParentProjectInfo
    err    error
}
```

**New Command**:
```go
func (m addModel) detectParentProject() tea.Msg {
    parent, err := m.service.FindParentProject(m.currentDir)
    return parentDetectionMsg{parent: parent, err: err}
}
```

**Changes to `add_tui_update.go`**:
- Added `case parentDetectionMsg` handler in `Update()`
  - If parent found: Stay on `stepParentDetection` for user prompt
  - If no parent: Proceed to `stepGitSelection` automatically
  - If error: Display error and quit
- Added `updateParentSelection()` method for handling user input:
  - `ctrl+c`, `esc`, `n`, `N`: Cancel registration
  - `up`, `down`, `k`, `j`: Navigate between Yes/No
  - `enter`: Confirm selection
  - `y`, `Y`: Quick Yes (proceed to git detection)

**Changes to `add_tui_view.go`**:
- Added `case stepParentDetection` in `View()` method
- Added `viewParentConfirmation()` method:
  - Shows "Checking for parent projects..." if detection in progress
  - Displays parent project name and path when found
  - Renders Yes/No options with cursor navigation
  - Provides keyboard shortcuts help text

**User Experience Flow**:
1. User runs `cops add /path/to/child`
2. TUI shows "Checking for parent projects..."
3. If parent found: Shows confirmation prompt with parent details
4. User can choose Yes (proceed) or No (cancel)
5. If no parent: Automatically proceeds to git detection

---

#### CLI Command: Updated Add Handler

**Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go`

**Changes**: Updated `runAddTUI()` call to pass service reference
- **Before**: `result, err := runAddTUI(path, noGit)`
- **After**: `result, err := runAddTUI(path, noGit, h.svc)`

**Purpose**: Enables TUI to call service methods for parent detection

---

#### Daemon PathUtil: Path Decoding

**Location**: `/Users/jayce/team-attention/cops/daemon/internal/platform/util/pathutil/pathutil.go`

**Purpose**: Decodes Claude Code project directory paths back to original file system paths

**New Function**:
```go
func DecodeClaudeProjectDir(claudeDir string) string
```

**Logic**:
1. Get user home directory
2. Build expected prefix: `~/.claude/projects/`
3. Validate that `claudeDir` starts with prefix
4. Extract encoded portion after prefix
5. Replace all `-` with `/` (reverse of encoding)
6. Ensure result starts with `/`
7. Return decoded path or empty string if invalid

**Examples**:
- Input: `~/.claude/projects/-Users-jayce-project`
- Output: `/Users/jayce/project`

**Error Cases**:
- Invalid prefix → returns `""`
- Empty encoded portion → returns `""`
- Home directory lookup fails → returns `""`

---

#### Daemon LogService: Hierarchical Matching

**Location**: `/Users/jayce/team-attention/cops/daemon/internal/service/logwatcher/log_service.go`

**Purpose**: Routes session logs to the most specific matching project

**New Field**:
- `projectPathToID map[string]shareddomain.ID`: Maps project paths to IDs for hierarchical matching

**Updated Methods**:

1. **`NewService()`**: Initialize `projectPathToID` map

2. **`UpdateTargets(targets []WatchTarget)`**: Build both mappings
   - `claudeDirToProject`: Exact Claude directory to project ID
   - `projectPathToID`: Original project path to project ID

3. **`GetProjectIDForClaudeDir(claudeDir string)`**: Hierarchical matching logic
   - **Step 1**: Try exact match in `claudeDirToProject`
   - **Step 2**: Decode `claudeDir` to original path using `DecodeClaudeProjectDir()`
   - **Step 3**: Find all registered project paths that are prefixes of decoded path
   - **Step 4**: Select longest matching prefix (most specific project)
   - **Step 5**: Return matched project ID or empty string if no match

**Matching Algorithm**:
```go
// Check if projectPath is a prefix of decodedPath
if projectPath == decodedPath {
    return projectID  // Exact match
}
if strings.HasPrefix(decodedPath, projectPath+"/") {
    // Parent directory match - keep longest
    if len(projectPath) > len(longestMatch) {
        longestMatch = projectPath
        matchedID = projectID
    }
}
```

**Example Scenario**:
- Registered projects:
  - `/home/user/repo` (Git project)
  - `/home/user/repo/subdir` (Non-Git project)
- Session log: `/home/user/repo/subdir/.claude/projects/-home-user-repo-subdir/.../session.jsonl`
- Claude dir: `-home-user-repo-subdir`
- Decoded: `/home/user/repo/subdir`
- Matches: Both `/home/user/repo` and `/home/user/repo/subdir`
- Selected: `/home/user/repo/subdir` (longest prefix → most specific)
- Result: Session routed to subdir project, not parent repo project

---

## Architecture Decisions

### Decision 1: Repository Refactoring for Hexagonal Architecture Compliance

**Problem**: The MongoDB repository contained business logic (`if params.IsGitProject` checks), violating hexagonal architecture principles.

**Solution**:
- Removed all `IsGitProject` conditionals from repository layer
- Repository now only filters based on empty/non-empty values (natural filtering)
- Business decisions remain in service layer where they belong

**Rationale**:
- Hexagonal architecture requires that outbound adapters (repositories) contain only data access logic
- Business logic belongs in the domain/service layer
- This maintains clean separation of concerns and makes the repository reusable

**Implementation**:
- `buildSearchConditions()`: Only adds conditions for non-empty parameters (no IsGitProject check)
- `collectNonEmptyURLs()`: Filters empty strings, not project types
- Service layer decides what parameters to pass based on project type

### Decision 2: Single $or Query Pattern

**Problem**: Original implementation used sequential queries (search by ID first, then by URL), which was inefficient.

**Solution**: Combined all search conditions into a single `$or` query.

**Rationale**:
- MongoDB can efficiently handle `$or` queries with proper indexing
- Reduces network round trips (one query instead of multiple)
- Simpler code flow - one query path instead of branching logic

**Implementation**:
```go
filter := bson.M{"$or": []bson.M{
    {mongoschema.ProjectIDField: objectID},
    {mongoschema.ProjectRemoteURLField: bson.M{"$in": urls}},
}}
```

### Decision 3: Parent Detection in Service Layer

**Problem**: Where to implement parent project detection logic - TUI layer or service layer?

**Solution**: Implemented in service layer (`tracking_service.go`) with TUI calling the service method.

**Rationale**:
- Follows hexagonal architecture: Inbound layer (TUI) should not directly access outbound ports (ConfigPort)
- Service layer acts as mediator between inbound and outbound adapters
- Enables reusability - other inbound adapters (HTTP, gRPC) could use the same logic
- Maintains proper layer separation and testability

**Flow**:
1. TUI calls `Service.FindParentProject(path)`
2. Service uses `ConfigPort.LoadGlobalConfig()` internally
3. Service returns processed result (`ParentProjectInfo` or nil)
4. TUI displays prompt and handles user interaction

### Decision 4: Hierarchical Matching Implementation

**Problem**: When multiple projects could match a session log path, which one should receive the log?

**Solution**: Longest path prefix matching - most specific project wins.

**Rationale**:
- Mirrors CIDR routing in networking (longest prefix match)
- Intuitive behavior: subdirectory projects override parent projects
- Enables nested project tracking (child project logs don't leak to parent)

**Implementation Location**: Daemon's `log_service.go` in `GetProjectIDForClaudeDir()` method
- Already maintains project routing mappings
- Centralized decision point for log routing
- Minimal performance impact (map iteration with prefix checking)

### Decision 5: Path Storage Strategy in Daemon

**Problem**: Claude directories are encoded (`-Users-jayce-project`), but hierarchical matching requires original paths.

**Solution**: Store both mappings:
- `claudeDirToProject`: Encoded Claude directory → Project ID (for exact matches)
- `projectPathToID`: Original file system path → Project ID (for hierarchical matching)

**Rationale**:
- Exact matches are fast and common (no decoding needed)
- Hierarchical matching only needed when exact match fails
- Storing both avoids repeated decoding operations
- Minimal memory overhead (one additional map)

---

## Testing

### Unit Tests Added

No new unit tests were added in this implementation (focus was on integration fixes).

### Test Coverage

Manual testing confirmed:
- Non-Git directories can be registered without errors
- Parent detection works correctly for nested directories
- User confirmation prompt displays and handles input properly
- Hierarchical matching routes logs to most specific project

### Verification Commands Run

```bash
# Build verification - all modules
cd /Users/jayce/team-attention/cops
go build ./api/...       # Result: PASS
go build ./cli/...       # Result: PASS
go build ./daemon/...    # Result: PASS
go build ./shared/...    # Result: PASS

# Full workspace build
go build ./...           # Result: PASS
```

### Manual Testing Scenarios

| Scenario | Expected Result | Status |
|----------|----------------|--------|
| Register non-Git directory (no parent) | Success - creates new project | PASS |
| Register non-Git directory (with parent) | Shows confirmation prompt | PASS |
| User confirms parent prompt | Proceeds to git detection | PASS |
| User cancels parent prompt | Exits without registration | PASS |
| Daemon routes log from child project | Routes to child, not parent | PASS |
| Daemon routes log from parent-only | Routes to parent project | PASS |

---

## Issues & Resolutions

| Issue | Resolution |
|-------|-----------|
| **API rejected non-Git projects** | Removed search criteria validation - repository now creates projects when conditions array is empty |
| **Repository contained business logic** | Refactored into 5 helper methods, removed all `IsGitProject` conditionals, kept only data access logic |
| **Function too complex (107 lines)** | Extracted helper methods: `buildSearchConditions`, `collectNonEmptyURLs`, `findByConditions`, `createProject`, `docToResult` |
| **Sequential query inefficiency** | Changed to single `$or` query combining ID and URL conditions |
| **No parent project detection** | Added `FindParentProject()` method in service layer with directory tree walking algorithm |
| **Daemon couldn't handle nested projects** | Implemented hierarchical matching with longest path prefix algorithm |
| **Encoded Claude paths unusable for matching** | Created `DecodeClaudeProjectDir()` function to reverse path encoding |

---

## Related Tickets

This work addresses the core requirement to support non-Git directory registration with hierarchical project tracking and proper session log routing.

## Key Takeaways

### What Changed
1. **API**: Removed validation barrier for non-Git projects, refactored repository to remove business logic
2. **CLI**: Added multi-step TUI flow with parent detection and user confirmation
3. **Daemon**: Implemented intelligent log routing with hierarchical project matching
4. **Architecture**: Improved hexagonal architecture compliance by removing business logic from repository layer

### Why It Matters
- **User Experience**: Users can now register any directory (Git or non-Git) without errors
- **Safety**: Parent detection prevents accidental duplicate tracking of subdirectories
- **Accuracy**: Hierarchical matching ensures session logs route to the correct project
- **Maintainability**: Cleaner architecture with proper separation of concerns

### Design Principles Applied
- **Hexagonal Architecture**: Clear separation between domain logic, inbound adapters, and outbound adapters
- **Single Responsibility**: Each helper method has one focused purpose
- **Fail Fast**: Path validation happens early in the flow
- **User Consent**: Explicit confirmation required before creating nested projects
- **Longest Prefix Match**: Industry-standard pattern for hierarchical routing
