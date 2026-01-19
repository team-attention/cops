package cobra

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/team-attention/cops/cli/internal/service/tracking"
	"github.com/team-attention/cops/shared/domain"
)

// Update implements tea.Model.
func (m addModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case localConfigDetectionMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.localConfig = msg.localConfig

		// Case 1: No local config found → go directly to registration flow
		if msg.localConfig == nil {
			m.step = stepOrgSelection
			return m, m.fetchOrganizations
		}

		// Case 2 & 3: Local config exists → check OrganizationID first
		if msg.localConfig.OrganizationID == "" {
			// Invalid config: missing required OrganizationID
			m.localConfigInvalid = true
			return m, nil // Stay on step, show invalid config view
		}

		// Local config found with OrganizationID, verify on server
		m.serverChecking = true
		return m, m.checkServerProject

	case serverProjectCheckMsg:
		// Server check completed
		m.serverChecking = false
		if msg.err != nil {
			// Auth error - set error and quit
			m.serverCheckErr = msg.err
			m.err = msg.err
			return m, tea.Quit
		}
		// Store server project result
		m.serverProject = msg.serverProject
		// Stay on stepLocalConfigDetection for user selection
		// View will render different options based on serverProject.Found
		return m, nil

	case connectExistingProjectMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.result.OrganizationID = m.serverProject.OrganizationID
		m.step = stepCompleted
		return m, tea.Quit

	case resetConfigMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.localConfig = nil
		m.localConfigInvalid = false
		// Go directly to registration flow (skip parent detection)
		m.step = stepOrgSelection
		return m, m.fetchOrganizations

	case orgFetchMsg:
		// 1. Handle error from organization fetch
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		// 2. Store fetched organizations in model
		m.organizations = msg.organizations
		// 3. Handle edge case: no organizations found
		if len(msg.organizations) == 0 {
			m.err = fmt.Errorf("no organizations found. Please create an organization first")
			return m, tea.Quit
		}
		// 4. Always show organization selection UI (removed auto-skip for single org)
		//    User must explicitly select/confirm their organization
		return m, nil

	case gitDetectionMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.gitRepos = msg.repos
		return m.handleGitDetectionComplete()

	case tea.KeyMsg:
		switch m.step {
		case stepLocalConfigDetection:
			return m.updateLocalConfigSelection(msg)
		case stepOrgSelection:
			return m.updateOrgSelection(msg)
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

// updateOrgSelection handles input during organization selection step.
func (m addModel) updateOrgSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.result.Cancelled = true
		return m, tea.Quit

	case "up", "k":
		if m.orgCursor > 0 {
			m.orgCursor--
		}

	case "down", "j":
		if m.orgCursor < len(m.organizations)-1 {
			m.orgCursor++
		}

	case "enter":
		// Select organization and proceed to git detection
		m.selectedOrgID = string(m.organizations[m.orgCursor].ID)
		m.selectedOrgName = m.organizations[m.orgCursor].Name
		m.result.OrganizationID = m.selectedOrgID
		m.step = stepGitSelection
		return m, m.detectGitRepos
	}

	return m, nil
}

// updateLocalConfigSelection handles input during local config detection step.
func (m addModel) updateLocalConfigSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle invalid config case (missing OrganizationID)
	if m.localConfigInvalid {
		maxCursor := 1 // 2 options: Reset & Register, Cancel
		switch msg.String() {
		case "ctrl+c", "esc":
			m.result.Cancelled = true
			return m, tea.Quit
		case "up", "k":
			if m.localConfigCursor > 0 {
				m.localConfigCursor--
			}
		case "down", "j":
			if m.localConfigCursor < maxCursor {
				m.localConfigCursor++
			}
		case "enter":
			switch m.localConfigCursor {
			case 0:
				// Reset & Register
				return m, m.resetLocalConfig()
			case 1:
				// Cancel
				m.result.Cancelled = true
				return m, tea.Quit
			}
		}
		return m, nil
	}

	// Calculate max cursor based on serverProject.Found and AlreadyRegistered
	// Case A: serverProject.Found && !AlreadyRegistered: 3 options (Connect, Register new, Cancel)
	// Case B: serverProject.Found && AlreadyRegistered: 2 options (Overwrite, Cancel)
	// Case C: !serverProject.Found: 2 options (Reset & Register, Cancel)
	maxCursor := 1 // Default: 2 options (0, 1)
	if m.serverProject != nil && m.serverProject.Found && !m.localConfig.AlreadyRegistered {
		maxCursor = 2 // 3 options for Case A
	}

	switch msg.String() {
	case "ctrl+c", "esc":
		m.result.Cancelled = true
		return m, tea.Quit

	case "up", "k":
		if m.localConfigCursor > 0 {
			m.localConfigCursor--
		}

	case "down", "j":
		if m.localConfigCursor < maxCursor {
			m.localConfigCursor++
		}

	case "enter":
		if m.serverProject != nil && m.serverProject.Found {
			if m.localConfig.AlreadyRegistered {
				// Case B: Already registered - 2 options (Overwrite, Cancel)
				switch m.localConfigCursor {
				case 0:
					// Overwrite - reset local config first, then proceed to registration flow
					return m, m.resetLocalConfig()
				case 1:
					// Cancel
					m.result.Cancelled = true
					return m, tea.Quit
				}
			} else {
				// Case A: Not registered - 3 options (Connect, Register new, Cancel)
				switch m.localConfigCursor {
				case 0:
					// Connect to existing project
					return m, m.connectToExistingProject()
				case 1:
					// Register as new project - proceed to org selection
					m.step = stepOrgSelection
					return m, m.fetchOrganizations
				case 2:
					// Cancel
					m.result.Cancelled = true
					return m, tea.Quit
				}
			}
		} else {
			// Case C: Server project not found - 2 options
			switch m.localConfigCursor {
			case 0:
				// Reset & Register as new project
				return m, m.resetLocalConfig()
			case 1:
				// Cancel
				m.result.Cancelled = true
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

// connectExistingProjectMsg is sent when connect operation completes.
type connectExistingProjectMsg struct {
	project *domain.Project
	err     error
}

// connectToExistingProject returns a command that adds the project to global config
// using the organization from the server project (which was verified to match localConfig.OrganizationID).
func (m addModel) connectToExistingProject() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		project, err := m.service.AddProject(ctx, tracking.AddProjectParams{
			Path:           m.localConfig.ProjectPath,
			OrganizationID: m.serverProject.OrganizationID, // From server, validated against local
			Sync:           false,
		})
		return connectExistingProjectMsg{project: project, err: err}
	}
}
