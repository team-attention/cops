# Requirements

## Request Summary

Enhance the `cops add` command with two improvements: (1) Add parent directory Git repository detection with user confirmation when a Git repo is found in parent directories up to the home directory, and (2) Replace the current flag-based interface with an interactive TUI using bubbletea that guides users through project name customization and data sync preferences.

## Acceptance Criteria

### Feature 1: Parent Directory Git Detection

- [ ] AC1.1: When `cops add` is run, search for `.git` in parent directories up to (but not including) the home directory (`~`)
- [ ] AC1.2: If a Git repository is found in a parent directory, show a confirmation prompt with the detected Git root path and number of levels up
- [ ] AC1.3: If nested Git repositories exist (multiple `.git` directories in the parent chain), present a selection menu allowing the user to choose which Git repository to use
- [ ] AC1.4: If user confirms/selects a parent Git repo, register that Git root as the project (same behavior as current Git project registration)
- [ ] AC1.5: If user declines, fall back to treating the current directory as a non-Git project
- [ ] AC1.6: The search stops at the home directory boundary - do not search beyond `~`
- [ ] AC1.7: Maintain existing behavior for `--no-git` flag (skip Git detection entirely)

### Feature 2: Bubbletea TUI Multi-Step Process

- [ ] AC2.1: Replace current non-interactive command with a bubbletea-based TUI that always runs (unless interrupted)
- [ ] AC2.2: **Step 1 - Project Name**: Display an editable text input with default project name pre-filled
  - Default for Git projects: directory name where `.git` folder is located
  - Default for non-Git projects: current directory name
  - User can edit, accept, or cancel (Ctrl+C)
- [ ] AC2.3: **Step 2 - Data Sync**: Display a yes/no confirmation prompt: "Do you want to upload all past Claude Code logs for this project? Or only logs from now on?"
  - Yes = sync past logs (calls `SyncProject`)
  - No = only track new logs going forward
  - User can select or cancel (Ctrl+C)
- [ ] AC2.4: Remove the `--sync` flag from the command (breaking change - remove from flags, help text, and implementation)
- [ ] AC2.5: If user presses Ctrl+C at any TUI step, abort the entire operation with a clean exit (no project registered)
- [ ] AC2.6: After successful completion, display project summary (ID, Name, Path, Git status) as currently shown
- [ ] AC2.7: Add bubbletea dependency to the project using `go get` command

## Scope

### In Scope

- Implement recursive parent directory search for Git repositories (up to home directory)
- Add user confirmation UI when parent Git repo is detected
- Add multi-selection UI when nested Git repos are found
- Create bubbletea TUI with two steps: project name input and data sync confirmation
- Remove `--sync` flag and migrate to TUI-based sync selection
- Handle Ctrl+C gracefully in TUI with full operation abort
- Maintain existing `--no-git` flag functionality
- Validate project name input (non-empty, reasonable length)

### Out of Scope

- Implementing the actual `SyncProject` functionality (already marked as "not yet implemented")
- Changing the project registration API or data structures
- Modifying how Git worktrees are detected or handled
- Adding additional TUI steps beyond name and sync
- Supporting non-interactive mode (all operations require TUI interaction)
- Customizing Claude directory paths in the TUI

## Technical Requirements

### Git Detection Enhancement

**Function**: Extend `gitutil` package with parent directory search

```go
// FindGitReposInParents searches for Git repositories from dir up to (but not including) home directory
// Returns a slice of Git root paths found (ordered from closest to farthest)
// Returns empty slice if no Git repos found
func FindGitReposInParents(dir string) ([]string, error)
```

**Implementation details:**
- Start from the given directory
- Walk up parent directories using `filepath.Dir()`
- At each level, check if `.git` exists
- Stop when reaching home directory (use `os.UserHomeDir()` for comparison)
- Collect all Git roots found in order (closest first)
- Return slice of absolute paths

**Integration point**: `tracking_service.go` in `AddProject` method before the existing `IsGitRepo` check

### TUI Implementation

**Library**: `github.com/charmbracelet/bubbletea` (latest version)

**Additional libraries**:
- `github.com/charmbracelet/bubbles/textinput` - for project name input
- `github.com/charmbracelet/bubbles/list` or custom selection - for Git repo selection
- `github.com/charmbracelet/lipgloss` - for styling (optional, recommended)

**TUI Flow**:

1. **Pre-TUI Phase**: Git repository detection
   - Run parent directory search
   - If no Git repos found → proceed with non-Git flow
   - If one Git repo found → show confirmation prompt
   - If multiple Git repos found → show selection menu

2. **Step 1**: Project Name Input
   - Model: `textinput.Model`
   - Default value: directory base name (from selected Git root or current dir)
   - Validation: non-empty, max 100 characters
   - Submit: Enter key
   - Cancel: Ctrl+C / Esc

3. **Step 2**: Data Sync Confirmation
   - Model: Custom yes/no selection or list with 2 items
   - Options: "Yes - Upload all past logs" / "No - Only track new logs"
   - Submit: Enter key
   - Cancel: Ctrl+C / Esc

4. **Post-TUI Phase**: Project registration
   - Use collected inputs (name, sync preference)
   - Call existing registration logic
   - Display success message

**Integration point**: `cli/internal/service/tracking/inbound/cli/cobra/add.go` in `RunE` function

### Data Flow Changes

**Current flow:**
```
User runs command → Parse flags → Validate path → Git detection → Register project → Done
```

**New flow:**
```
User runs command → Validate path → Parent Git detection →
  [If Git found] → Confirmation/Selection UI →
  TUI Step 1 (Name) → TUI Step 2 (Sync) → Register project → Done
```

### Breaking Changes

- Remove `--sync` / `-s` flag (users must use TUI)
- All `cops add` invocations now require interactive terminal (no scripting support)

## Edge Cases & Error Handling

### Edge Case 1: Permission Denied on Parent Directories
**Scenario**: User runs `cops add` in a directory where parent directories are not readable
**Handling**: Gracefully handle permission errors, stop search at first inaccessible directory, use whatever Git repos were found before the error

### Edge Case 2: Home Directory is a Git Repo
**Scenario**: User's home directory itself contains `.git`
**Handling**: Do NOT include home directory in search results (stop before reaching it)

### Edge Case 3: Symlinks in Path
**Scenario**: Current directory path contains symlinks
**Handling**: Resolve symlinks using `filepath.EvalSymlinks()` before starting parent search to ensure consistent path comparison

### Edge Case 4: Non-Interactive Terminal
**Scenario**: Command run in a non-TTY environment (CI, script, ssh without -t)
**Handling**: Detect using `term.IsTerminal(int(os.Stdout.Fd()))`, show clear error: "Error: cops add requires an interactive terminal. Use SSH with -t flag or run in a terminal emulator."

### Edge Case 5: Very Long Directory Path
**Scenario**: Path is hundreds of directories deep
**Handling**: Set a reasonable max depth limit (e.g., 50 levels) to prevent infinite loops on misconfigured filesystems

### Edge Case 6: Invalid Project Name Input
**Scenario**: User enters empty string or only whitespace
**Handling**: Show validation error in TUI, keep user on Step 1 until valid input provided

### Edge Case 7: Git Repo at Filesystem Root
**Scenario**: There's a `.git` at `/` (unlikely but possible in containers)
**Handling**: Treat as valid Git repo, include in results if found before home directory check

### Edge Case 8: Ctrl+C During Project Registration
**Scenario**: User cancels after TUI completes but during API call
**Handling**: Best effort cleanup - if local config was written, attempt to delete it. Log warning about partial state.

## Constraints

- Must use bubbletea for TUI implementation (user requirement)
- Must maintain backward compatibility with `--no-git` flag
- Must not search beyond home directory for performance and privacy reasons
- TUI must be keyboard-navigable (no mouse required)
- Must handle terminal resize gracefully (bubbletea handles this by default)
- Parent directory search must complete in reasonable time (<500ms for typical depths)

## Additional Context

### Related Code Files

- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go` - Main AddProject logic
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go` - Cobra command definition
- `/Users/jayce/team-attention/cops/cli/internal/platform/util/gitutil/gitutil.go` - Git utility functions
- `/Users/jayce/team-attention/cops/shared/domain/project.go` - Project data model

### Current Project Name Logic

Currently in `tracking_service.go` line 81:
```go
name := filepath.Base(projectPath)
```

This will be replaced with user-provided name from TUI.

### Current Sync Flag Logic

Currently in `add.go` lines 14, 62:
```go
var sync bool
cmd.Flags().BoolVarP(&sync, "sync", "s", false, "Sync past records immediately")
```

These will be removed and replaced with TUI prompt.

### SyncProject Status

The `SyncProject` function (line 309 in tracking_service.go) currently returns:
```go
return errutil.Internalf("sync is not yet implemented")
```

The TUI will call this function when user selects "Yes", but implementation is out of scope for this work.

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| How far up should parent directory search go? | Stop at home directory (`~`), do not search beyond it |
| What is "Sub Directory" configuration? | Just register as Git project if found, keep similar to current process, add parent search |
| Should we ask for confirmation when parent Git repo is detected? | Yes, show confirmation prompt with detected path and levels up |
| What if there are nested Git repos? | Let user choose between them (show selection menu) |
| When should TUI appear? | Always show TUI |
| Should project name be editable? | Yes, show default value but make it editable |
| What does data sync step ask? | "Do you want to upload all past Claude Code logs for this project? Or only logs from now on?" (Yes/No) |
| Which TUI library to use? | bubbletea |
| Should --sync flag still work? | No, remove it completely (breaking change) |
| What happens on Ctrl+C during TUI? | Abort entire operation, do not register project |
