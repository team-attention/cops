# Implementation Plan

## Overview

Enhance the `cops add` command with parent directory Git repository detection and a bubbletea-based TUI for interactive project name customization and data sync preferences.

## Selected Packages

| Problem | Package | Context7 ID | Reason for Selection |
|---------|---------|-------------|----------------------|
| TUI Framework | bubbletea | `/charmbracelet/bubbletea` | User requirement; Elm-architecture MVU pattern |
| TUI Components | bubbles | `/charmbracelet/bubbles` | Official component library; provides textinput |
| TUI Styling | lipgloss | (charmbracelet) | Official styling for terminal UIs |
| Terminal Detection | golang.org/x/term | (stdlib extension) | Standard library extension for TTY detection |

## Architecture Decisions

### Decision 1: TUI File Location
**Choice**: Create `add_tui.go` in the same package as `add.go` (`cli/internal/service/tracking/inbound/cli/cobra/`)
**Rationale**: This is the only TUI command in the codebase; keeping it in the same package avoids unnecessary abstraction while maintaining cohesion with the add command.

### Decision 2: Git Detection Location
**Choice**: Move Git detection logic from service layer to TUI (pre-processing before service call)
**Rationale**: The TUI needs Git detection results to present the confirmation/selection UI. Service layer receives final decisions only.

### Decision 3: Multi-Step TUI State Management
**Choice**: Use a single model with step-based state machine (step 0: git selection, step 1: name input, step 2: sync selection)
**Rationale**: Simpler than nested models; all state in one place; easy to navigate forward/backward.

### Decision 4: Yes/No Selection Implementation
**Choice**: Custom cursor-based selection (not bubbles/list) for sync confirmation
**Rationale**: bubbles/list is overkill for 2 options; custom implementation is simpler and more readable.

### Decision 5: Project Name Field Addition
**Choice**: Add `Name` field to `AddProjectParams` struct
**Rationale**: Service layer needs user-provided name; cleanest way to pass it through existing interface.

## Implementation Steps

### Step 1: Add Dependencies

**Files to Create/Modify**:
- `cli/go.mod` (modify via go get)

**Commands**:
```bash
cd /Users/jayce/team-attention/cops/cli && go get github.com/charmbracelet/bubbletea@latest
cd /Users/jayce/team-attention/cops/cli && go get github.com/charmbracelet/bubbles@latest
cd /Users/jayce/team-attention/cops/cli && go get github.com/charmbracelet/lipgloss@latest
cd /Users/jayce/team-attention/cops/cli && go get golang.org/x/term@latest
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Dependencies added | `go mod tidy` | No errors, packages in go.sum | Happy path |

---

### Step 2: Add FindGitReposInParents Function to gitutil

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/cli/internal/platform/util/gitutil/gitutil.go` (modify)

**Functions**:

```go
// maxParentSearchDepth is the maximum number of parent directories to search.
const maxParentSearchDepth = 50

// FindGitReposInParents searches for Git repositories from dir up to (but not including) home directory.
// Returns a slice of Git root paths found (ordered from closest to farthest).
// Returns empty slice if no Git repos found.
// Stops searching when:
// - Home directory is reached (not included in results)
// - Filesystem root is reached
// - Max depth limit (50 levels) is exceeded
// - Permission denied on a parent directory
func FindGitReposInParents(dir string) ([]string, error) {
    // Implementation outline:
    // 1. Get home directory using os.UserHomeDir()
    // 2. Resolve symlinks in dir using filepath.EvalSymlinks()
    // 3. Initialize empty slice for results
    // 4. Walk up parent directories using filepath.Dir()
    // 5. At each level (before home dir), call IsGitRepo()
    // 6. If git repo found, append to results
    // 7. Stop at home directory, filesystem root, or max depth
    // 8. Return collected git repos (closest first)
}
```

**Detailed Implementation**:

```go
func FindGitReposInParents(dir string) ([]string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, fmt.Errorf("failed to get home directory: %w", err)
    }

    // Resolve symlinks for consistent comparison
    resolvedDir, err := filepath.EvalSymlinks(dir)
    if err != nil {
        // If symlink resolution fails, use original path
        resolvedDir = dir
    }

    absDir, err := filepath.Abs(resolvedDir)
    if err != nil {
        return nil, fmt.Errorf("failed to get absolute path: %w", err)
    }

    resolvedHome, err := filepath.EvalSymlinks(homeDir)
    if err != nil {
        resolvedHome = homeDir
    }

    var gitRepos []string
    currentDir := absDir
    depth := 0

    for depth < maxParentSearchDepth {
        // Stop if we've reached or passed the home directory
        if currentDir == resolvedHome || currentDir == homeDir {
            break
        }

        // Stop at filesystem root
        parentDir := filepath.Dir(currentDir)
        if parentDir == currentDir {
            break
        }

        // Check if parent directory is accessible
        if _, err := os.Stat(parentDir); err != nil {
            // Permission denied or other error - stop search
            break
        }

        // Check if parent is a git repo
        if IsGitRepo(parentDir) {
            gitRepos = append(gitRepos, parentDir)
        }

        currentDir = parentDir
        depth++
    }

    return gitRepos, nil
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Git repo in parent | `/home/user/project/subdir` with `.git` in `/home/user/project` | `["/home/user/project"]` | Happy path |
| Multiple git repos | Nested repos at depth 1 and 3 | Both paths, closest first | Multiple repos |
| No git repos | Directory with no parent git repos | Empty slice `[]` | No repos found |
| At home directory | `~` | Empty slice `[]` | Home dir boundary |
| Permission denied | Inaccessible parent | Partial results before error | Permission handling |
| Symlinks in path | Symlinked directory | Resolved paths | Symlink handling |
| Filesystem root | `/tmp` with no parent git | Empty slice `[]` | Root boundary |
| Deep nesting | 60 levels deep | Results up to depth 50 | Max depth limit |

---

### Step 3: Add Name Field to AddProjectParams

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go` (modify)

**Changes**:

1. Modify `AddProjectParams` struct (line 21-25):

```go
// AddProjectParams contains parameters for AddProject.
type AddProjectParams struct {
    Path  string
    Name  string // User-provided project name
    NoGit bool
    Sync  bool
}
```

2. Modify `AddProject` function (line 81):

Replace:
```go
// Use directory name as project name
name := filepath.Base(projectPath)
```

With:
```go
// Use provided name or fall back to directory name
name := params.Name
if name == "" {
    name = filepath.Base(projectPath)
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Name provided | `{Name: "my-project"}` | Project with name "my-project" | User-provided name |
| Name empty | `{Name: ""}` | Project with directory basename | Fallback to default |

---

### Step 4: Create TUI Model and Implementation

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui.go` (create)

**Complete Implementation**:

```go
package cobra

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "golang.org/x/term"

    "github.com/team-attention/cops/cli/internal/platform/util/gitutil"
)

// TUI step constants
const (
    stepGitSelection = iota
    stepNameInput
    stepSyncSelection
    stepCompleted
)

// addTUIResult contains the collected user inputs from the TUI.
type addTUIResult struct {
    ProjectPath  string
    ProjectName  string
    IsGitProject bool
    SyncPastLogs bool
    Cancelled    bool
}

// addModel is the bubbletea model for the add command TUI.
type addModel struct {
    // Current step in the TUI flow
    step int

    // Error state
    err error

    // Git detection results
    gitRepos       []string // Found git repos in parents (closest first)
    gitCursor      int      // Cursor for git repo selection
    selectedGitIdx int      // Selected git repo index (-1 for none/current dir)
    currentDir     string   // Original directory passed to command
    noGitFlag      bool     // --no-git flag value

    // Project name input
    nameInput textinput.Model

    // Sync selection
    syncCursor int // 0 = Yes, 1 = No

    // Final result
    result addTUIResult

    // Styling
    titleStyle   lipgloss.Style
    itemStyle    lipgloss.Style
    cursorStyle  lipgloss.Style
    helpStyle    lipgloss.Style
    errorStyle   lipgloss.Style
}

// newAddModel creates a new add TUI model.
func newAddModel(dir string, noGitFlag bool) addModel {
    ti := textinput.New()
    ti.Placeholder = "project-name"
    ti.CharLimit = 100
    ti.Width = 40

    m := addModel{
        step:           stepGitSelection,
        currentDir:     dir,
        noGitFlag:      noGitFlag,
        selectedGitIdx: -1,
        nameInput:      ti,
        titleStyle:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")),
        itemStyle:      lipgloss.NewStyle().PaddingLeft(2),
        cursorStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("86")),
        helpStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
        errorStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
    }

    return m
}

// Init implements tea.Model.
func (m addModel) Init() tea.Cmd {
    return m.detectGitRepos
}

// detectGitRepos is a command that searches for git repos in parent directories.
func (m addModel) detectGitRepos() tea.Msg {
    if m.noGitFlag {
        return gitDetectionMsg{repos: nil, err: nil}
    }

    // Check if current directory is a git repo
    absDir, err := filepath.Abs(m.currentDir)
    if err != nil {
        return gitDetectionMsg{repos: nil, err: err}
    }

    var repos []string

    // First check current directory
    if gitutil.IsGitRepo(absDir) {
        mainPath, err := gitutil.FindMainRepoPath(absDir)
        if err == nil {
            repos = append(repos, mainPath)
        }
    }

    // Then search parent directories
    parentRepos, err := gitutil.FindGitReposInParents(absDir)
    if err != nil {
        return gitDetectionMsg{repos: repos, err: nil} // Continue with what we found
    }

    // Combine and deduplicate
    for _, pr := range parentRepos {
        found := false
        for _, r := range repos {
            if r == pr {
                found = true
                break
            }
        }
        if !found {
            repos = append(repos, pr)
        }
    }

    return gitDetectionMsg{repos: repos, err: nil}
}

// gitDetectionMsg is sent when git detection completes.
type gitDetectionMsg struct {
    repos []string
    err   error
}

// Update implements tea.Model.
func (m addModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case gitDetectionMsg:
        if msg.err != nil {
            m.err = msg.err
            return m, tea.Quit
        }
        m.gitRepos = msg.repos
        return m.handleGitDetectionComplete()

    case tea.KeyMsg:
        switch m.step {
        case stepGitSelection:
            return m.updateGitSelection(msg)
        case stepNameInput:
            return m.updateNameInput(msg)
        case stepSyncSelection:
            return m.updateSyncSelection(msg)
        }
    }

    return m, nil
}

// handleGitDetectionComplete processes git detection results and transitions to next step.
func (m addModel) handleGitDetectionComplete() (tea.Model, tea.Cmd) {
    absDir, _ := filepath.Abs(m.currentDir)

    if m.noGitFlag || len(m.gitRepos) == 0 {
        // No git repos found or --no-git flag - proceed as non-git project
        m.result.ProjectPath = absDir
        m.result.IsGitProject = false
        m.step = stepNameInput
        m.nameInput.SetValue(filepath.Base(absDir))
        m.nameInput.Focus()
        return m, textinput.Blink
    }

    if len(m.gitRepos) == 1 {
        // Single git repo found - show confirmation
        m.step = stepGitSelection
        return m, nil
    }

    // Multiple git repos - show selection
    m.step = stepGitSelection
    return m, nil
}

// updateGitSelection handles input during git repo selection step.
func (m addModel) updateGitSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "ctrl+c", "esc":
        m.result.Cancelled = true
        return m, tea.Quit

    case "up", "k":
        if m.gitCursor > 0 {
            m.gitCursor--
        }

    case "down", "j":
        maxIdx := len(m.gitRepos) // +1 for "Use current directory" option
        if m.gitCursor < maxIdx {
            m.gitCursor++
        }

    case "enter":
        absDir, _ := filepath.Abs(m.currentDir)

        if m.gitCursor < len(m.gitRepos) {
            // Selected a git repo
            m.selectedGitIdx = m.gitCursor
            m.result.ProjectPath = m.gitRepos[m.gitCursor]
            m.result.IsGitProject = true
        } else {
            // Selected "Use current directory as non-git project"
            m.selectedGitIdx = -1
            m.result.ProjectPath = absDir
            m.result.IsGitProject = false
        }

        // Move to name input step
        m.step = stepNameInput
        m.nameInput.SetValue(filepath.Base(m.result.ProjectPath))
        m.nameInput.Focus()
        return m, textinput.Blink
    }

    return m, nil
}

// updateNameInput handles input during project name input step.
func (m addModel) updateNameInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "ctrl+c", "esc":
        m.result.Cancelled = true
        return m, tea.Quit

    case "enter":
        name := strings.TrimSpace(m.nameInput.Value())
        if name == "" {
            // Keep on this step - validation error shown in View
            return m, nil
        }
        m.result.ProjectName = name
        m.step = stepSyncSelection
        return m, nil
    }

    var cmd tea.Cmd
    m.nameInput, cmd = m.nameInput.Update(msg)
    return m, cmd
}

// updateSyncSelection handles input during sync selection step.
func (m addModel) updateSyncSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "ctrl+c", "esc":
        m.result.Cancelled = true
        return m, tea.Quit

    case "up", "k", "left", "h":
        if m.syncCursor > 0 {
            m.syncCursor--
        }

    case "down", "j", "right", "l":
        if m.syncCursor < 1 {
            m.syncCursor++
        }

    case "enter":
        m.result.SyncPastLogs = m.syncCursor == 0 // 0 = Yes
        m.step = stepCompleted
        return m, tea.Quit

    case "y", "Y":
        m.result.SyncPastLogs = true
        m.step = stepCompleted
        return m, tea.Quit

    case "n", "N":
        m.result.SyncPastLogs = false
        m.step = stepCompleted
        return m, tea.Quit
    }

    return m, nil
}

// View implements tea.Model.
func (m addModel) View() string {
    if m.err != nil {
        return m.errorStyle.Render(fmt.Sprintf("Error: %v\n", m.err))
    }

    var b strings.Builder

    switch m.step {
    case stepGitSelection:
        b.WriteString(m.viewGitSelection())
    case stepNameInput:
        b.WriteString(m.viewNameInput())
    case stepSyncSelection:
        b.WriteString(m.viewSyncSelection())
    }

    return b.String()
}

// viewGitSelection renders the git repo selection view.
func (m addModel) viewGitSelection() string {
    var b strings.Builder

    if len(m.gitRepos) == 0 {
        b.WriteString("Detecting Git repositories...\n")
        return b.String()
    }

    b.WriteString(m.titleStyle.Render("Git Repository Detected"))
    b.WriteString("\n\n")

    if len(m.gitRepos) == 1 {
        b.WriteString(fmt.Sprintf("Found Git repository at:\n  %s\n\n", m.gitRepos[0]))
        b.WriteString("Which project would you like to register?\n\n")
    } else {
        b.WriteString("Found multiple Git repositories:\n\n")
    }

    // Render git repo options
    for i, repo := range m.gitRepos {
        cursor := "  "
        if m.gitCursor == i {
            cursor = m.cursorStyle.Render("> ")
        }

        // Calculate levels up from current directory
        absDir, _ := filepath.Abs(m.currentDir)
        levelsUp := countLevelsUp(absDir, repo)
        levelStr := ""
        if levelsUp > 0 {
            levelStr = fmt.Sprintf(" (%d level(s) up)", levelsUp)
        }

        b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, repo, levelStr))
    }

    // Add "use current directory" option
    cursor := "  "
    if m.gitCursor == len(m.gitRepos) {
        cursor = m.cursorStyle.Render("> ")
    }
    absDir, _ := filepath.Abs(m.currentDir)
    b.WriteString(fmt.Sprintf("%sUse current directory as non-git project (%s)\n", cursor, absDir))

    b.WriteString("\n")
    b.WriteString(m.helpStyle.Render("up/down: navigate | enter: select | ctrl+c: cancel"))
    b.WriteString("\n")

    return b.String()
}

// viewNameInput renders the project name input view.
func (m addModel) viewNameInput() string {
    var b strings.Builder

    b.WriteString(m.titleStyle.Render("Project Name"))
    b.WriteString("\n\n")

    b.WriteString(fmt.Sprintf("Path: %s\n", m.result.ProjectPath))
    if m.result.IsGitProject {
        b.WriteString("Type: Git project\n")
    } else {
        b.WriteString("Type: Non-git project\n")
    }
    b.WriteString("\n")

    b.WriteString("Enter project name:\n")
    b.WriteString(m.nameInput.View())
    b.WriteString("\n")

    // Show validation error if name is empty and user pressed enter
    if strings.TrimSpace(m.nameInput.Value()) == "" {
        b.WriteString(m.errorStyle.Render("Project name cannot be empty"))
        b.WriteString("\n")
    }

    b.WriteString("\n")
    b.WriteString(m.helpStyle.Render("enter: continue | ctrl+c: cancel"))
    b.WriteString("\n")

    return b.String()
}

// viewSyncSelection renders the sync selection view.
func (m addModel) viewSyncSelection() string {
    var b strings.Builder

    b.WriteString(m.titleStyle.Render("Data Sync"))
    b.WriteString("\n\n")

    b.WriteString("Do you want to upload all past Claude Code logs for this project?\n")
    b.WriteString("Or only track logs from now on?\n\n")

    options := []string{
        "Yes - Upload all past logs",
        "No - Only track new logs",
    }

    for i, opt := range options {
        cursor := "  "
        if m.syncCursor == i {
            cursor = m.cursorStyle.Render("> ")
        }
        b.WriteString(fmt.Sprintf("%s%s\n", cursor, opt))
    }

    b.WriteString("\n")
    b.WriteString(m.helpStyle.Render("up/down: navigate | enter: select | y/n: quick select | ctrl+c: cancel"))
    b.WriteString("\n")

    return b.String()
}

// countLevelsUp counts how many directory levels repo is above dir.
func countLevelsUp(dir, repo string) int {
    levels := 0
    current := dir
    for current != repo && current != "/" && current != "." {
        parent := filepath.Dir(current)
        if parent == current {
            break
        }
        current = parent
        levels++
    }
    if current == repo {
        return levels
    }
    return 0
}

// runAddTUI runs the add command TUI and returns the result.
// Returns an error if the terminal is not interactive.
func runAddTUI(dir string, noGitFlag bool) (*addTUIResult, error) {
    // Check if running in an interactive terminal
    if !term.IsTerminal(int(os.Stdout.Fd())) {
        return nil, fmt.Errorf("cops add requires an interactive terminal. Use SSH with -t flag or run in a terminal emulator")
    }

    model := newAddModel(dir, noGitFlag)
    p := tea.NewProgram(model)

    finalModel, err := p.Run()
    if err != nil {
        return nil, fmt.Errorf("TUI error: %w", err)
    }

    result := finalModel.(addModel).result
    return &result, nil
}
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Single git repo | Parent has `.git` | Confirmation shown | Single repo flow |
| Multiple git repos | 2+ parent git repos | Selection menu | Multi-repo flow |
| No git repos | No parent git | Skip to name input | No git flow |
| --no-git flag | `noGitFlag=true` | Skip git detection | Flag handling |
| Name validation | Empty name + enter | Error message, stay on step | Validation |
| Valid name | "my-project" + enter | Proceed to sync step | Happy path |
| Sync yes | Select "Yes" | `SyncPastLogs=true` | Sync selection |
| Sync no | Select "No" | `SyncPastLogs=false` | Sync selection |
| Cancel at git step | Ctrl+C | `Cancelled=true` | Cancellation |
| Cancel at name step | Ctrl+C | `Cancelled=true` | Cancellation |
| Cancel at sync step | Ctrl+C | `Cancelled=true` | Cancellation |
| Non-TTY environment | Piped input | Error message | TTY detection |
| Quick select y | Press 'y' | `SyncPastLogs=true`, complete | Quick select |
| Quick select n | Press 'n' | `SyncPastLogs=false`, complete | Quick select |

---

### Step 5: Modify Add Command to Use TUI

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go` (modify)

**Complete Replacement**:

```go
package cobra

import (
    "context"
    "fmt"

    "github.com/spf13/cobra"

    "github.com/team-attention/cops/cli/internal/service/tracking"
)

// NewAddCommand creates the 'add' subcommand.
func (h *TrackingCLIHandler) NewAddCommand() *cobra.Command {
    var noGit bool

    cmd := &cobra.Command{
        Use:   "add [directory]",
        Short: "Add a project to tracking",
        Long: `Register a project directory for Claude Code session tracking.

This command launches an interactive TUI to configure the project:
1. Git repository detection - finds git repos in parent directories
2. Project name - customize the display name for the project
3. Data sync - choose whether to upload past Claude Code logs

If the directory is a git repository, the main repo path will be registered
(not the worktree path). This ensures the same project ID is used across
all worktrees.

Examples:
  cops add .                # Add current directory (interactive)
  cops add /path/to/project # Add specific directory (interactive)
  cops add . --no-git       # Treat as non-git project`,
        Args: cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            path := "."
            if len(args) > 0 {
                path = args[0]
            }

            // Run the TUI
            result, err := runAddTUI(path, noGit)
            if err != nil {
                return err
            }

            // Check if user cancelled
            if result.Cancelled {
                fmt.Println("Operation cancelled.")
                return nil
            }

            // Create params from TUI result
            params := tracking.AddProjectParams{
                Path:  result.ProjectPath,
                Name:  result.ProjectName,
                NoGit: !result.IsGitProject, // NoGit=true means non-git project
                Sync:  result.SyncPastLogs,
            }

            // Call service to register project
            project, err := h.svc.AddProject(context.Background(), params)
            if err != nil {
                return err
            }

            // Display success message
            fmt.Println("\nProject added successfully!")
            fmt.Printf("  ID:   %s\n", project.ID)
            fmt.Printf("  Name: %s\n", project.Name)
            fmt.Printf("  Path: %s\n", project.Path)
            fmt.Printf("  Git:  %t\n", project.IsGitProject)

            if result.SyncPastLogs {
                fmt.Println("  Sync: requested (will sync past logs)")
            }

            return nil
        },
    }

    cmd.Flags().BoolVar(&noGit, "no-git", false, "Treat directory as non-git project")
    // Note: --sync flag removed - now handled by TUI

    return cmd
}
```

**Key Changes from Original**:
1. Removed `var sync bool` and `--sync` flag
2. Replaced direct service call with TUI flow
3. Added handling for `result.Cancelled`
4. Added `Name` field to params
5. Updated help text to describe interactive flow
6. Display project name in success output

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Complete flow | Git repo, name, sync=yes | Project registered | Happy path |
| User cancels | Ctrl+C during TUI | "Operation cancelled." | Cancellation |
| --no-git flag | `cops add . --no-git` | Skip git detection in TUI | Flag handling |
| Non-TTY error | Piped input | Error about interactive terminal | TTY detection |
| Service error | API unreachable | Error from service | Error handling |

---

### Step 6: Update Imports in add.go

The new `add.go` requires the following imports (already included in Step 5):

```go
import (
    "context"
    "fmt"

    "github.com/spf13/cobra"

    "github.com/team-attention/cops/cli/internal/service/tracking"
)
```

No additional changes needed - the TUI functions are in the same package.

---

### Step 7: Run Tests and Verify Build

**Commands**:
```bash
cd /Users/jayce/team-attention/cops/cli && go build ./...
cd /Users/jayce/team-attention/cops/cli && go test ./...
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
|----------|-------|-----------------|----------------|
| Build succeeds | `go build ./...` | No errors | Compilation |
| Tests pass | `go test ./...` | All tests pass | Unit tests |

---

## Execution Order

1. **Step 1: Add Dependencies** (no dependencies)
   - Run `go get` commands to add bubbletea, bubbles, lipgloss, x/term

2. **Step 2: Add FindGitReposInParents** (no dependencies)
   - Add new function to gitutil.go

3. **Step 3: Add Name Field to AddProjectParams** (no dependencies)
   - Modify tracking_service.go struct and AddProject function

4. **Step 4: Create TUI Model** (depends on Steps 1, 2)
   - Create add_tui.go with complete TUI implementation

5. **Step 5: Modify Add Command** (depends on Steps 3, 4)
   - Replace add.go with TUI-integrated version

6. **Step 6: Update Imports** (depends on Step 5)
   - Verify imports are correct (already done in Step 5)

7. **Step 7: Run Tests and Verify Build** (depends on all steps)
   - Ensure everything compiles and tests pass

## Notes for Execute Agent

### Critical Implementation Details

1. **Package Import Path**: When importing bubbletea and bubbles, use:
   ```go
   tea "github.com/charmbracelet/bubbletea"
   "github.com/charmbracelet/bubbles/textinput"
   "github.com/charmbracelet/lipgloss"
   "golang.org/x/term"
   ```

2. **tea.Model Interface**: The `addModel` must implement:
   - `Init() tea.Cmd`
   - `Update(tea.Msg) (tea.Model, tea.Cmd)`
   - `View() string`

3. **Git Detection Command Pattern**: Use a method that returns `tea.Msg` as a command:
   ```go
   func (m addModel) detectGitRepos() tea.Msg {
       // ... detection logic
       return gitDetectionMsg{repos: repos, err: nil}
   }
   ```

4. **Handling --no-git with TUI**: When `noGitFlag` is true:
   - Skip git detection entirely
   - Go directly to name input step
   - Set `IsGitProject = false`

5. **Path Resolution**: Always resolve paths to absolute before storing:
   ```go
   absDir, err := filepath.Abs(m.currentDir)
   ```

6. **Error in Non-TTY**: Check BEFORE creating the tea.Program:
   ```go
   if !term.IsTerminal(int(os.Stdout.Fd())) {
       return nil, fmt.Errorf("requires interactive terminal")
   }
   ```

### File Locations Summary

| File | Action | Description |
|------|--------|-------------|
| `/Users/jayce/team-attention/cops/cli/go.mod` | modify (via go get) | Add dependencies |
| `/Users/jayce/team-attention/cops/cli/internal/platform/util/gitutil/gitutil.go` | modify | Add FindGitReposInParents |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/tracking_service.go` | modify | Add Name to params |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui.go` | create | TUI implementation |
| `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go` | modify | Integrate TUI |

### Breaking Changes

- The `--sync` / `-s` flag is removed
- All `cops add` invocations now require an interactive terminal
- Scripts that previously used `cops add --sync` will need to be updated

### Rollback Strategy

If issues occur:
1. Revert `add.go` to original (restore `--sync` flag)
2. Remove `add_tui.go`
3. Remove `Name` field from `AddProjectParams` in `tracking_service.go`
4. Optionally remove bubbletea dependencies (or leave them - they don't affect anything)
5. `FindGitReposInParents` can remain as it's additive and doesn't break existing code
