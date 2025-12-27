# Research Report

## Mode
General Research

## Request Summary
Enhance the `cops add` command with two features: (1) Parent directory Git repository detection with user confirmation when Git repos are found in parent directories up to the home directory, and (2) Replace the current flag-based interface with an interactive TUI using bubbletea for project name customization and data sync preferences.

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File | Reason |
|------|--------|
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go` | Current add command implementation - will be heavily modified |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go` | Service layer with AddProject logic - needs parameter changes |
| `/Users/jayce/team-attention/cops/cli/internal/platform/util/gitutil/gitutil.go` | Git utilities - new function FindGitReposInParents needed here |
| `/Users/jayce/team-attention/cops/cli/internal/platform/util/pathutil/pathutil.go` | Path utilities - home directory detection available |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/handler.go` | Handler structure for CLI commands |
| `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_tracking.go` | DI container setup for tracking module |
| `/Users/jayce/team-attention/cops/shared/domain/project.go` | Project domain model |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-backend.md` | Go coding conventions |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-service.md` | Service layer patterns |
| `/Users/jayce/team-attention/cops/.agent/rules/go/go-inbound.md` | Inbound adapter patterns |

## Package Candidates

### Problem 1: TUI Framework

| Package | Context7 ID | Why Better Than Alternatives |
|---------|-------------|------------------------------|
| bubbletea | `/charmbracelet/bubbletea` | User requirement; Elm-architecture, mature, well-documented |
| bubbles | `/charmbracelet/bubbles` | Official component library for bubbletea; provides textinput, list components |
| lipgloss | `/charmbracelet/lipgloss` | Official styling library for terminal UIs; pairs with bubbletea |

### Problem 2: Terminal Detection

| Package | Context7 ID | Why Better Than Alternatives |
|---------|-------------|------------------------------|
| golang.org/x/term | (stdlib extension) | Standard library extension; IsTerminal() function for TTY detection |

## Technical Constraints

- Must use `dig` for dependency injection (not `fx`) - CLI uses stateless command execution pattern
- bubbletea/bubbles packages must be added via `go get` command
- TUI must integrate with existing Cobra command structure
- Must maintain backward compatibility with `--no-git` flag
- No existing TUI implementations in codebase - this will be the first

## Current Implementation Analysis

### 1. Current `cops add` Command (`add.go:1-67`)

**Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go`

**Current Flow**:
```
1. Parse command args (path defaults to ".")
2. Parse flags (--sync, --no-git)
3. Create AddProjectParams struct
4. Call h.svc.AddProject(ctx, params)
5. Print success message with project details
```

**Flags to modify**:
- Line 14: `var sync bool` - REMOVE (will be handled by TUI)
- Line 62: `cmd.Flags().BoolVarP(&sync, "sync", "s", false, ...)` - REMOVE
- Line 63: `cmd.Flags().BoolVar(&noGit, "no-git", false, ...)` - KEEP

**Current AddProjectParams** (tracking_service.go:21-25):
```go
type AddProjectParams struct {
    Path  string
    NoGit bool
    Sync  bool
}
```

**Needs to add**:
- `Name string` - user-provided project name from TUI

### 2. Git Utilities (`gitutil.go:1-140`)

**Location**: `/Users/jayce/team-attention/cops/cli/internal/platform/util/gitutil/gitutil.go`

**Existing functions**:
- `IsGitRepo(dir string) bool` (line 15-23) - checks if directory has `.git`
- `FindMainRepoPath(dir string) (string, error)` (line 28-70) - finds main repo from worktree
- `ListWorktrees(mainRepoPath string) ([]string, error)` (line 73-105)
- `GetCurrentBranch(repoPath string) (string, error)` (line 108-115)
- `GetRemoteURL(repoPath string) (string, error)` (line 119-126)
- `GetActualRemoteURL(repoPath string) string` (line 132-139)

**New function needed**:
```go
// FindGitReposInParents searches for Git repositories from dir up to (but not including) home directory
// Returns a slice of Git root paths found (ordered from closest to farthest)
// Returns empty slice if no Git repos found
func FindGitReposInParents(dir string) ([]string, error)
```

**Implementation approach**:
- Use `os.UserHomeDir()` to get home directory boundary
- Use `filepath.Dir()` to walk up parent directories
- At each level, call existing `IsGitRepo()` to check for `.git`
- Stop when current directory equals home directory
- Handle symlinks with `filepath.EvalSymlinks()` for consistent path comparison
- Set max depth limit (50 levels) for safety

### 3. Path Utilities (`pathutil.go:1-55`)

**Location**: `/Users/jayce/team-attention/cops/cli/internal/platform/util/pathutil/pathutil.go`

**Relevant existing functions**:
- `ExpandPath(path string) (string, error)` (line 10-19) - expands `~` and resolves absolute path
- `DefaultCopsConfigDir() (string, error)` (line 43-49) - returns `~/.cops`

**Can use `os.UserHomeDir()` directly for home directory boundary check**

### 4. Tracking Service (`tracking_service.go:1-312`)

**Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go`

**Current AddProject logic** (lines 52-190):
1. Expand and validate path (line 54-57)
2. Check if git project unless --no-git (line 63-78)
3. Use directory name as project name (line 81) - **NEEDS CHANGE**: use user-provided name
4. Determine project ID via API or local config (lines 84-126)
5. Save local config (lines 128-132)
6. Create project struct (lines 141-151)
7. Add to global registry (lines 153-173)
8. Sync if requested (lines 181-187)

**Changes needed**:
- Modify `AddProjectParams` to include `Name string`
- Replace `name := filepath.Base(projectPath)` with user-provided name
- Keep sync logic but trigger from TUI selection

### 5. Handler Structure (`handler.go:1-33`)

**Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/handler.go`

**Current structure**:
```go
type TrackingCLIHandler struct {
    logger *slog.Logger
    svc    *tracking.Service
}

func NewTrackingCLIHandler(l *slog.Logger, svc *tracking.Service) *TrackingCLIHandler
func (h *TrackingCLIHandler) Commands() []*cobra.Command
```

**No changes needed** - TUI will be integrated into the add command, not the handler

### 6. Project Domain Model (`project.go:1-29`)

**Location**: `/Users/jayce/team-attention/cops/shared/domain/project.go`

**Current structure**:
```go
type ProjectAbstract struct {
    ID   ID     `json:"id"`
    Name string `json:"name"`
    Path string `json:"path"`
}

type Project struct {
    ProjectAbstract
    IsGitProject bool      `json:"gitProject"`
    ClaudeDir    string    `json:"claudeDir"`
    Worktrees    []string  `json:"worktrees,omitempty"`
    RegisteredAt time.Time `json:"registeredAt"`
}
```

**No changes needed** - existing structure supports user-provided name

### 7. DI Container (`module_tracking.go:1-49`)

**Location**: `/Users/jayce/team-attention/cops/cli/cmd/internal/container/module_tracking.go`

**Current registration**:
- Uses `dig.As` and `dig.Group` for interface casting
- Registers outbound adapters, service, and CLI handler

**No changes needed** - TUI is internal to the add command, not a new injectable component

### 8. CLI go.mod (`go.mod:1-60`)

**Location**: `/Users/jayce/team-attention/cops/cli/go.mod`

**Current dependencies** - no bubbletea packages present

**Dependencies to add**:
```
github.com/charmbracelet/bubbletea
github.com/charmbracelet/bubbles
github.com/charmbracelet/lipgloss
golang.org/x/term
```

## Similar Implementations Found

### No existing TUI implementations in this codebase

The codebase currently uses:
- Cobra for CLI command handling
- Simple `fmt.Println` for output
- No interactive prompts or TUI components

This will be the first TUI implementation. Reference the bubbletea documentation and examples.

### Example Pattern: Existing Add Command Flow

- **File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go:32-59`
- **Relevance**: Shows current RunE pattern that will be modified to launch TUI

```go
RunE: func(cmd *cobra.Command, args []string) error {
    path := "."
    if len(args) > 0 {
        path = args[0]
    }

    params := tracking.AddProjectParams{
        Path:  path,
        NoGit: noGit,
        Sync:  sync,
    }

    project, err := h.svc.AddProject(context.Background(), params)
    // ... handle result
}
```

### Example Pattern: Git Detection

- **File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go:62-78`
- **Relevance**: Shows current git detection logic that needs to be enhanced with parent search

```go
if !params.NoGit && gitutil.IsGitRepo(absPath) {
    mainPath, err := gitutil.FindMainRepoPath(absPath)
    if err != nil {
        // fallback to non-git
    } else {
        projectPath = mainPath
        isGitProject = true
    }
}
```

## Additional Information for Planning

### Proposed File Structure for TUI

Create new files in the inbound adapter:
```
cli/internal/service/tracking/inbound/cli/cobra/
    handler.go      # existing - no changes
    add.go          # modify to launch TUI
    add_tui.go      # NEW: TUI model and logic
    list.go         # existing - no changes
    remove.go       # existing - no changes
```

Alternative structure (separate TUI package):
```
cli/internal/service/tracking/inbound/cli/
    cobra/
        handler.go
        add.go      # delegates to TUI
        list.go
        remove.go
    tui/            # NEW package
        add_tui.go  # TUI implementation
```

**Recommendation**: Use the first approach (same package) for simplicity since this is the only TUI command.

### TUI Model Structure (Proposed)

```go
type addModel struct {
    // State
    step       int  // 0=git selection, 1=name input, 2=sync selection
    quitting   bool
    err        error

    // Git detection results
    gitRepos   []string  // Found git repos in parents
    selectedGit int      // Index of selected git repo (-1 for none)

    // User inputs
    nameInput  textinput.Model
    syncChoice bool

    // Final result
    projectPath string
    projectName string
    isGitProject bool
}
```

### Integration Points Summary

1. **gitutil.go**: Add `FindGitReposInParents()` function
2. **tracking_service.go**:
   - Add `Name` field to `AddProjectParams`
   - Use `params.Name` instead of `filepath.Base(projectPath)`
3. **add.go**:
   - Remove `--sync` flag
   - Launch TUI in `RunE`
   - Collect TUI results and call service
4. **add_tui.go** (NEW):
   - Implement bubbletea Model interface
   - Multi-step form: git selection, name input, sync selection
5. **go.mod**: Add bubbletea, bubbles, lipgloss, x/term dependencies

### Edge Cases to Handle

1. **Non-TTY environment**: Check `term.IsTerminal()` before launching TUI
2. **Ctrl+C handling**: bubbletea handles this via `tea.Quit` message
3. **Empty project name**: Validate in TUI, show error, keep on step
4. **Permission denied**: Gracefully stop parent search, use found repos
5. **Home directory is git repo**: Stop search before reaching home
6. **Symlinks in path**: Resolve with `filepath.EvalSymlinks()` before comparison

### Architectural Notes

- The TUI should be self-contained within the CLI inbound adapter layer
- Service layer remains unaware of TUI - it just receives parameters
- Git detection logic moves from service layer to TUI (pre-processing before service call)
- Service layer will receive final decisions: path, name, isGitProject, sync preference

### Performance Considerations

- Parent directory search should complete in <500ms for typical depths
- Set max depth limit of 50 levels to prevent infinite loops
- Use `filepath.EvalSymlinks()` once at the start, not per iteration
- bubbletea rendering is already optimized with frame rate limiting
