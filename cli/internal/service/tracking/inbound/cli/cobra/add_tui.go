package cobra

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/team-attention/cops/cli/internal/platform/util/gitutil"
	"github.com/team-attention/cops/cli/internal/service/tracking"
	"github.com/team-attention/cops/cli/internal/service/user"
	"github.com/team-attention/cops/shared/domain"
)

// TUI step constants
const (
	stepParentDetection = iota
	stepOrgSelection
	stepGitSelection
	stepNameInput
	stepSyncSelection
	stepCompleted
)

// addTUIResult contains the collected user inputs from the TUI.
type addTUIResult struct {
	ProjectPath    string
	ProjectName    string
	IsGitProject   bool
	SyncPastLogs   bool
	Cancelled      bool
	OrganizationID string
}

// addModel is the bubbletea model for the add command TUI.
type addModel struct {
	// Current step in the TUI flow
	step int

	// Error state
	err error

	// Parent detection results
	parentProject *tracking.ParentProjectInfo // Result from service
	parentCursor  int                          // 0=Yes, 1=No
	service       *tracking.Service            // Reference to service for parent detection

	// Organization selection
	organizations   []*domain.Organization // List of user's organizations
	orgCursor       int                    // Cursor for organization selection
	selectedOrgID   string                 // Selected organization ID
	selectedOrgName string                 // Selected organization name (for display)
	userSvc         *user.Service          // Reference to user service for fetching orgs

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
func newAddModel(
	dir string,
	noGitFlag bool,
	service *tracking.Service,
	userSvc *user.Service,
) addModel {
	ti := textinput.New()
	ti.Placeholder = "project-name"
	ti.CharLimit = 100
	ti.Width = 40

	m := addModel{
		step:           stepParentDetection,
		currentDir:     dir,
		noGitFlag:      noGitFlag,
		service:        service,
		userSvc:        userSvc,
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
	return m.detectParentProject
}

// parentDetectionMsg is sent when parent detection completes.
type parentDetectionMsg struct {
	parent *tracking.ParentProjectInfo
	err    error
}

// detectParentProject is a command that checks for parent projects.
func (m addModel) detectParentProject() tea.Msg {
	parent, err := m.service.FindParentProject(m.currentDir)
	return parentDetectionMsg{parent: parent, err: err}
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

// orgFetchMsg is sent when organization fetching completes.
type orgFetchMsg struct {
	organizations []*domain.Organization
	err           error
}

// fetchOrganizations is a command that fetches user organizations.
func (m addModel) fetchOrganizations() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	orgs, err := m.userSvc.GetMyOrganizations(ctx)
	return orgFetchMsg{organizations: orgs, err: err}
}

// runAddTUI runs the add command TUI and returns the result.
// Returns an error if the terminal is not interactive.
func runAddTUI(
	dir string,
	noGitFlag bool,
	service *tracking.Service,
	userSvc *user.Service,
) (*addTUIResult, error) {
	// Check if running in an interactive terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return nil, fmt.Errorf("cops add requires an interactive terminal. Use SSH with -t flag or run in a terminal emulator")
	}

	model := newAddModel(dir, noGitFlag, service, userSvc)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("TUI error: %w", err)
	}

	result := finalModel.(addModel).result
	return &result, nil
}
