# Development Walkthrough

## Summary

Enhanced the `cops add` command with two major features: (1) Parent directory Git repository detection that searches up to the home directory with user confirmation, and (2) an interactive TUI (bubbletea) that guides users through project name customization and data sync preferences, replacing the previous flag-based interface.

## Code Overview

### New Components

#### `FindGitReposInParents` - Git Utility Function

- **Location**: `/Users/jayce/team-attention/cops/cli/internal/platform/util/gitutil/gitutil.go:141-201`
- **Purpose**: Searches for Git repositories in parent directories up to (but not including) the home directory
- **Key Features**:
  - Returns slice of Git root paths ordered from closest to farthest
  - Stops at home directory boundary (never searches beyond `~`)
  - Handles symlinks using `filepath.EvalSymlinks()` for consistent path comparison
  - Gracefully handles permission errors by stopping search
  - No arbitrary depth limit - naturally terminates at home or filesystem root

**Implementation Details**:
```go
// Key termination conditions in infinite loop:
for {
    // Stop at home directory
    if currentDir == resolvedHome || currentDir == homeDir {
        break
    }

    // Stop at filesystem root
    parentDir := filepath.Dir(currentDir)
    if parentDir == currentDir {
        break
    }

    // Stop on permission errors
    if _, err := os.Stat(parentDir); err != nil {
        break
    }

    // Check and collect git repos
    if IsGitRepo(parentDir) {
        gitRepos = append(gitRepos, parentDir)
    }

    currentDir = parentDir
}
```

#### Interactive TUI - Multi-File MVU Architecture

The TUI implementation follows the bubbletea Model-View-Update (MVU) pattern, split into 3 focused files:

##### 1. `add_tui.go` - Model & Entry Point (163 lines)

- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui.go`
- **Purpose**: Model definition, initialization, and TUI entry point
- **Key Components**:
  - `addTUIResult` struct - Contains final user selections (path, name, isGit, sync)
  - `addModel` struct - Bubbletea model with state machine (step-based flow)
  - `newAddModel()` - Model constructor with lipgloss styling setup
  - `Init()` method - Triggers async git detection on startup
  - `detectGitRepos()` - Async command that searches for Git repos
  - `runAddTUI()` - Entry point with TTY detection

**State Machine**:
```go
const (
    stepGitSelection   = iota  // Select which Git repo to use (if found)
    stepNameInput              // Enter project name
    stepSyncSelection          // Choose whether to sync past logs
    stepCompleted              // TUI complete, return to command
)
```

**TTY Detection**:
```go
func runAddTUI(dir string, noGitFlag bool) (*addTUIResult, error) {
    // Prevent TUI in non-interactive environments (CI, pipes, etc.)
    if !term.IsTerminal(int(os.Stdout.Fd())) {
        return nil, fmt.Errorf("cops add requires an interactive terminal...")
    }
    // ...
}
```

##### 2. `add_tui_update.go` - Update Logic (162 lines)

- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go`
- **Purpose**: Message handling and state transitions
- **Key Methods**:
  - `Update(msg tea.Msg)` - Main message dispatcher
  - `handleGitDetectionComplete()` - Processes git detection results, decides next step
  - `updateGitSelection(msg tea.KeyMsg)` - Handles arrow keys, enter, ctrl+c for git selection
  - `updateNameInput(msg tea.KeyMsg)` - Handles text input and validation
  - `updateSyncSelection(msg tea.KeyMsg)` - Handles yes/no selection with shortcuts (y/n keys)

**Git Detection Flow**:
```go
func (m addModel) handleGitDetectionComplete() (tea.Model, tea.Cmd) {
    if m.noGitFlag || len(m.gitRepos) == 0 {
        // No git - skip to name input
        m.step = stepNameInput
        return m, textinput.Blink
    }

    // Git found - show selection UI
    m.step = stepGitSelection
    return m, nil
}
```

**Cancellation Handling**:
- All steps handle `ctrl+c` and `esc` by setting `result.Cancelled = true`
- Command checks cancellation and exits cleanly without registering project

##### 3. `add_tui_view.go` - View Logic (160 lines)

- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go`
- **Purpose**: Rendering logic for all TUI screens
- **Key Methods**:
  - `View()` - Main view dispatcher
  - `viewGitSelection()` - Renders Git repo selection with "levels up" indicator
  - `viewNameInput()` - Renders text input with validation feedback
  - `viewSyncSelection()` - Renders yes/no options for sync preference
  - `countLevelsUp(dir, repo int)` - Helper to calculate directory depth

**Visual Example** (Git Selection):
```
Git Repository Detected

Found Git repository at:
  /Users/jayce/team-attention/cops

Which project would you like to register?

> /Users/jayce/team-attention/cops
  Use current directory as non-git project (/Users/jayce/team-attention/cops/cli)

up/down: navigate | enter: select | ctrl+c: cancel
```

### Modified Components

#### `NewAddCommand` - CLI Command Handler

- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go`
- **Changes**:
  - **Removed**: `--sync` / `-s` flag (breaking change)
  - **Added**: TUI integration via `runAddTUI()`
  - **Added**: Cancellation handling
  - **Added**: Display of project name in success output
  - **Updated**: Help text to describe interactive flow

**Before**:
```go
var sync bool
params := tracking.AddProjectParams{
    Path:  path,
    NoGit: noGit,
    Sync:  sync,
}
```

**After**:
```go
// Run the TUI
result, err := runAddTUI(path, noGit)
if err != nil {
    return err
}

// Check cancellation
if result.Cancelled {
    fmt.Println("Operation cancelled.")
    return nil
}

// Use TUI results
params := tracking.AddProjectParams{
    Path:  result.ProjectPath,
    Name:  result.ProjectName,
    NoGit: !result.IsGitProject,
    Sync:  result.SyncPastLogs,
}
```

#### `AddProjectParams` - Service Parameters

- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go:20-26`
- **Changes**: Added `Name string` field

#### `AddProject` - Service Method

- **Location**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go:81-85`
- **Changes**: Use provided name or fall back to directory basename

**Before**:
```go
name := filepath.Base(projectPath)
```

**After**:
```go
// Use provided name or fall back to directory name
name := params.Name
if name == "" {
    name = filepath.Base(projectPath)
}
```

This maintains backward compatibility - if no name is provided, the original behavior is preserved.

## User Experience Improvements

### Before: Flag-Based Interface

```bash
$ cops add /path/to/project --sync
Project added successfully!
  ID:   proj_abc123
  Path: /path/to/project
  Git:  true
  Sync: completed
```

**Pain Points**:
- No parent directory Git detection (had to manually navigate to git root)
- No project name customization
- Had to know about `--sync` flag upfront
- No way to confirm detected Git repository

### After: Interactive TUI

**Step 1 - Git Detection** (if parent Git repo found):
```
Git Repository Detected

Found Git repository at:
  /Users/jayce/team-attention/cops

Which project would you like to register?

> /Users/jayce/team-attention/cops (2 level(s) up)
  Use current directory as non-git project (/Users/jayce/.../cli)

up/down: navigate | enter: select | ctrl+c: cancel
```

**Step 2 - Project Name**:
```
Project Name

Path: /Users/jayce/team-attention/cops
Type: Git project

Enter project name:
cops▊

enter: continue | ctrl+c: cancel
```

**Step 3 - Data Sync**:
```
Data Sync

Do you want to upload all past Claude Code logs for this project?
Or only track logs from now on?

> Yes - Upload all past logs
  No - Only track new logs

up/down: navigate | enter: select | y/n: quick select | ctrl+c: cancel
```

**Result**:
```
Project added successfully!
  ID:   proj_abc123
  Name: cops
  Path: /Users/jayce/team-attention/cops
  Git:  true
  Sync: requested (will sync past logs)
```

**Benefits**:
- Automatically finds parent Git repos (searches up to `~`)
- Shows how many levels up the Git repo is
- Allows choosing between multiple Git repos or non-git
- Customizable project name with sensible default
- Interactive sync decision with clear options
- Graceful cancellation at any step (Ctrl+C)
- Prevents accidental runs in non-TTY environments

## Architecture Decisions

### 1. MVU Pattern File Split

**Decision**: Split 467-line `add_tui.go` into 3 files (~160 lines each)

**Files**:
- `add_tui.go` - Model definition and initialization
- `add_tui_update.go` - All update/message handling
- `add_tui_view.go` - All rendering logic

**Rationale**: Follows bubbletea best practices; easier to maintain and understand; clear separation of concerns (Model-View-Update)

### 2. Removed Depth Limit

**Decision**: No `maxParentSearchDepth` constant; loop terminates naturally

**Rationale**:
- Home directory is the natural boundary (privacy and scope)
- Filesystem root provides absolute upper bound
- Permission errors stop search gracefully
- Arbitrary limits (like 50) are unnecessary and confusing

### 3. TUI in Same Package

**Decision**: Place TUI files in same package as `add.go` (`cobra/`)

**Rationale**: This is the only TUI command; keeping it colocated avoids premature abstraction

### 4. Breaking Change: Remove `--sync` Flag

**Decision**: Remove `--sync` flag entirely, force TUI interaction

**Rationale**: Ensures users make informed decision about sync; reduces flag complexity; aligns with interactive-first design

## Testing

### Manual Testing Performed

All acceptance criteria verified during implementation:

| Feature | Test Case | Result |
|---------|-----------|--------|
| Parent Git Detection | Run `cops add` in subdirectory of Git repo | Git repo detected and shown |
| Multiple Git Repos | Nested Git repos in parent chain | Selection menu displayed |
| No Git Repos | Non-git directory | Skips to name input |
| `--no-git` Flag | `cops add . --no-git` | Skips git detection entirely |
| Project Name Input | Edit default name | Custom name saved |
| Empty Name Validation | Press enter with empty name | Error shown, stays on step |
| Sync Selection | Choose "Yes" or "No" | Preference saved correctly |
| Cancellation | Press Ctrl+C at any step | "Operation cancelled" shown |
| Non-TTY Detection | Pipe input to command | Error about interactive terminal |

### Build Verification

```bash
cd /Users/jayce/team-attention/cops/cli
go build ./...  # Result: SUCCESS (0 errors)
```

### Dependency Installation

Added via `go get`:
- `github.com/charmbracelet/bubbletea@latest`
- `github.com/charmbracelet/bubbles@latest`
- `github.com/charmbracelet/lipgloss@latest`
- `golang.org/x/term@latest`

All dependencies added to `cli/go.mod` and `cli/go.sum`.

## Issues & Resolutions

No issues encountered during implementation. Initial plan was executed successfully with only minor refinements based on code review feedback.

## Code Review Iterations

### Iteration 1 (Initial Implementation)
- Implemented all features per plan
- All acceptance criteria met
- Status: NEEDS REVISION

### Iteration 2 (Refinements)
- Fixed minor issues from iteration 1
- Status: NEEDS REVISION

### Iteration 3 (Final)
- **Issue 1**: Removed unnecessary `maxParentSearchDepth` constant per user feedback
- **Resolution**: Simplified loop to terminate naturally at home directory or filesystem root
- **Issue 2**: Split 467-line TUI file per user feedback
- **Resolution**: Split into 3 files following MVU pattern (add_tui.go, add_tui_update.go, add_tui_view.go)
- Status: **PASS** ✓

## Breaking Changes

**`--sync` / `-s` flag removed**

- **Impact**: Scripts using `cops add --sync` will fail
- **Migration**: Remove flag from scripts; TUI will prompt for sync preference interactively
- **Reason**: Interactive TUI provides better UX and informed decision-making

**Interactive terminal now required**

- **Impact**: Cannot run `cops add` in CI/scripts without TTY
- **Workaround**: Use `ssh -t` for remote execution, or run in terminal emulator
- **Error Message**: "cops add requires an interactive terminal. Use SSH with -t flag or run in a terminal emulator"

## Files Changed

| File | Lines Changed | Type | Description |
|------|---------------|------|-------------|
| `cli/go.mod` | +4 deps | Modified | Added bubbletea ecosystem dependencies |
| `cli/go.sum` | +20 | Modified | Dependency checksums |
| `gitutil/gitutil.go` | +62 | Modified | Added `FindGitReposInParents` function |
| `tracking_service.go` | +5 | Modified | Added `Name` field to params |
| `cobra/add.go` | +23/-14 | Modified | Integrated TUI, removed `--sync` flag |
| `cobra/add_tui.go` | +163 | **New** | TUI model and initialization |
| `cobra/add_tui_update.go` | +162 | **New** | TUI update logic |
| `cobra/add_tui_view.go` | +160 | **New** | TUI rendering logic |

**Total**: 5 files modified, 3 files created, 569 lines added, 14 lines removed

## Future Enhancements

Potential improvements not in scope for this implementation:

1. **Non-interactive mode**: Add `--name` and `--sync-mode` flags for scripting support
2. **Git worktree visualization**: Show existing worktrees when registering Git project
3. **Config preview**: Show what will be written to `.cops/config.json` before confirming
4. **Project templates**: Pre-fill project name based on repository patterns (e.g., GitHub org/repo)
5. **Sync progress**: Real-time progress indicator when `SyncProject` is implemented

## Related Documentation

- Original requirements: `.agent/artifacts/20251227-231854/01_requirements.md`
- Research findings: `.agent/artifacts/20251227-231854/02_research.md`
- Implementation plan: `.agent/artifacts/20251227-231854/03_plan.md`
- Final review (PASS): `.agent/artifacts/20251227-231854/06_review_iteration3.md`
