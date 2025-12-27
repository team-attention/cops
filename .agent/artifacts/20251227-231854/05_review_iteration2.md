# Pre-PR Code Review - Iteration 2

## Review Summary
- **Status**: FAIL
- **Feedback Items**: 2
- **Changes Recommended**: 2

---

## User Feedback Analysis

### Feedback 1: `maxParentSearchDepth = 50` Necessity

**File**: `/Users/jayce/team-attention/cops/cli/internal/platform/util/gitutil/gitutil.go:141-142`

**User Question**: "Why is this needed? Can't we just keep searching up until we reach the ~ path?"

**Analysis**:

The user's question is **VALID**. Let me evaluate the current implementation:

**Current Implementation**:
```go
const maxParentSearchDepth = 50

func FindGitReposInParents(dir string) ([]string, error) {
    // ...
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
        // ...
    }
}
```

**Why the depth limit is unnecessary**:

1. **Home directory check already exists**: Line 180-183 already breaks the loop when reaching the home directory
2. **Filesystem root check already exists**: Line 185-188 already breaks the loop when reaching the filesystem root (`/`)
3. **Redundant safety mechanism**: The depth limit of 50 is a defensive measure that serves no practical purpose because:
   - The maximum possible depth from any directory to `/` is already bounded by the filesystem
   - Home directory check will always trigger before reaching 50 levels in normal usage
   - Even deeply nested directories rarely exceed 20-30 levels
4. **Code simplicity**: Removing the depth limit makes the code cleaner and more readable

**Verdict**: VALID - The `maxParentSearchDepth` constant and depth counter are redundant defensive programming that complicate the code without providing real value.

**Recommendation**: Remove the depth limit and simplify the loop.

---

### Feedback 2: `add_tui.go` File Length and Readability

**File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui.go`

**Current State**:
- File length: 467 lines
- Contains: 1 struct definition, 1 message type, 15+ methods/functions
- All TUI logic in a single file

**Analysis**:

The user's concern is **VALID**. While the file is not excessively long by Go standards, there are opportunities to improve readability and organization.

**Current Structure Issues**:
1. **Mixed responsibilities**: Model definition, update logic, and view rendering are all in one file
2. **Large View methods**: Each `view*` method contains significant string building logic
3. **Helper function placement**: `countLevelsUp` is at the end of the file, disconnected from its usage

**Improvement Options**:

**Option A: Split by MVU Pattern (Recommended)**
Split into 3 files following the bubbletea Model-View-Update pattern:
- `add_tui.go` - Model definition, Init, and entry point (`runAddTUI`)
- `add_tui_update.go` - All Update methods (message handling)
- `add_tui_view.go` - All View methods (rendering)

**Option B: Split by Feature/Step**
Split by TUI step:
- `add_tui.go` - Core model and common logic
- `add_tui_git.go` - Git selection step (update + view)
- `add_tui_name.go` - Name input step (update + view)
- `add_tui_sync.go` - Sync selection step (update + view)

**Recommendation**: Option A is cleaner because:
- Follows bubbletea's natural MVU architecture
- Each file has a single responsibility (model, update, view)
- Easier to navigate: "Where's the rendering code?" -> `add_tui_view.go`
- More maintainable: Changes to update logic don't touch view file and vice versa

**Verdict**: VALID - File can be improved by splitting into 3 files following MVU pattern.

---

## Execution Plan for Execute Agent

### Task 1: Remove `maxParentSearchDepth` and Simplify Loop

**File**: `/Users/jayce/team-attention/cops/cli/internal/platform/util/gitutil/gitutil.go`

**Changes Required**:

1. **Delete lines 141-142** (constant definition):
   ```go
   // DELETE THESE LINES:
   // maxParentSearchDepth is the maximum number of parent directories to search.
   const maxParentSearchDepth = 50
   ```

2. **Update function documentation** (lines 144-151):
   ```go
   // FindGitReposInParents searches for Git repositories from dir up to (but not including) home directory.
   // Returns a slice of Git root paths found (ordered from closest to farthest).
   // Returns empty slice if no Git repos found.
   // Stops searching when:
   // - Home directory is reached (not included in results)
   // - Filesystem root is reached
   // - Permission denied on a parent directory
   ```
   Remove the line: `// - Max depth limit (50 levels) is exceeded`

3. **Simplify the loop** (lines 175-204):
   Replace:
   ```go
   var gitRepos []string
   currentDir := absDir
   depth := 0

   for depth < maxParentSearchDepth {
       // ... loop body ...
       depth++
   }
   ```
   With:
   ```go
   var gitRepos []string
   currentDir := absDir

   for {
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
   }
   ```

---

### Task 2: Split `add_tui.go` into MVU Files

**Source File**: `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui.go`

**Create 3 new files**:

#### File 1: `add_tui.go` (Keep - Model and Entry Point)

Keep only:
- Package declaration and imports (lines 1-15)
- TUI step constants (lines 17-23)
- `addTUIResult` struct (lines 25-32)
- `addModel` struct (lines 34-64)
- `newAddModel` function (lines 66-87)
- `Init` method (lines 89-92)
- `detectGitRepos` method (lines 94-137)
- `gitDetectionMsg` type (lines 139-143)
- `runAddTUI` function (lines 449-467)

#### File 2: `add_tui_update.go` (New - Update Logic)

Move to this file:
- Package declaration
- Import: `tea "github.com/charmbracelet/bubbletea"`, `"github.com/charmbracelet/bubbles/textinput"`, `"path/filepath"`, `"strings"`
- `Update` method (lines 145-168)
- `handleGitDetectionComplete` method (lines 170-193)
- `updateGitSelection` method (lines 195-236)
- `updateNameInput` method (lines 238-259)
- `updateSyncSelection` method (lines 261-295)

#### File 3: `add_tui_view.go` (New - View Logic)

Move to this file:
- Package declaration
- Import: `"fmt"`, `"strings"`, `"path/filepath"`
- `View` method (lines 297-315)
- `viewGitSelection` method (lines 317-367)
- `viewNameInput` method (lines 369-399)
- `viewSyncSelection` method (lines 401-429)
- `countLevelsUp` helper function (lines 431-447)

---

## Detailed File Contents for Execute Agent

### `/Users/jayce/team-attention/cops/cli/internal/platform/util/gitutil/gitutil.go`

**Final state of `FindGitReposInParents` function**:

```go
// FindGitReposInParents searches for Git repositories from dir up to (but not including) home directory.
// Returns a slice of Git root paths found (ordered from closest to farthest).
// Returns empty slice if no Git repos found.
// Stops searching when:
// - Home directory is reached (not included in results)
// - Filesystem root is reached
// - Permission denied on a parent directory
func FindGitReposInParents(dir string) ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.New("failed to get home directory")
	}

	// Resolve symlinks for consistent comparison
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		// If symlink resolution fails, use original path
		resolvedDir = dir
	}

	absDir, err := filepath.Abs(resolvedDir)
	if err != nil {
		return nil, errors.New("failed to get absolute path")
	}

	resolvedHome, err := filepath.EvalSymlinks(homeDir)
	if err != nil {
		resolvedHome = homeDir
	}

	var gitRepos []string
	currentDir := absDir

	for {
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
	}

	return gitRepos, nil
}
```

---

### TUI File Split

#### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui.go`

```go
package cobra

import (
	"fmt"
	"os"
	"path/filepath"

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
	titleStyle  lipgloss.Style
	itemStyle   lipgloss.Style
	cursorStyle lipgloss.Style
	helpStyle   lipgloss.Style
	errorStyle  lipgloss.Style
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

#### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go`

```go
package cobra

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

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
```

#### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go`

```go
package cobra

import (
	"fmt"
	"path/filepath"
	"strings"
)

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
```

---

## Verification Steps After Implementation

1. **Build verification**:
   ```bash
   go build ./cli/...
   ```

2. **Test the TUI** (manual):
   ```bash
   cd /some/nested/directory
   cops add .
   ```

3. **Verify file organization**:
   - `add_tui.go` - Model, Init, entry point (~150 lines)
   - `add_tui_update.go` - Update methods (~130 lines)
   - `add_tui_view.go` - View methods (~120 lines)

---

## Summary of Changes

| Item | Type | Description |
|------|------|-------------|
| Remove `maxParentSearchDepth` | Simplification | Remove unnecessary depth limit constant and counter |
| Split `add_tui.go` | Refactoring | Split into 3 files following MVU pattern |

---

## Final Status: **FAIL**

Changes recommended to address user feedback. Execute agent should:

1. Remove the `maxParentSearchDepth` constant and simplify the `FindGitReposInParents` function loop
2. Split `add_tui.go` into three files: `add_tui.go`, `add_tui_update.go`, `add_tui_view.go`
