package cobra

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Update implements tea.Model.
func (m addModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case parentDetectionMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.parentProject = msg.parent
		if msg.parent == nil {
			// No parent found, proceed to org selection
			m.step = stepOrgSelection
			return m, m.fetchOrganizations
		}
		// Parent found, stay on stepParentDetection for user confirmation
		return m, nil

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
		case stepParentDetection:
			return m.updateParentSelection(msg)
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

// updateParentSelection handles input during parent project detection step.
func (m addModel) updateParentSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc", "n", "N":
		// User cancelled or said No
		m.result.Cancelled = true
		return m, tea.Quit

	case "up", "k":
		if m.parentCursor > 0 {
			m.parentCursor--
		}

	case "down", "j":
		if m.parentCursor < 1 {
			m.parentCursor++
		}

	case "enter":
		if m.parentCursor == 0 {
			// Yes - proceed to org selection
			m.step = stepOrgSelection
			return m, m.fetchOrganizations
		} else {
			// No - cancel
			m.result.Cancelled = true
			return m, tea.Quit
		}

	case "y", "Y":
		// Yes - proceed to org selection
		m.step = stepOrgSelection
		return m, m.fetchOrganizations
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
