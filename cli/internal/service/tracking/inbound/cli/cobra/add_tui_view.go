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
	case stepLocalConfigDetection:
		b.WriteString(m.viewLocalConfigDetection())
	case stepOrgSelection:
		b.WriteString(m.viewOrgSelection())
	case stepGitSelection:
		b.WriteString(m.viewGitSelection())
	case stepNameInput:
		b.WriteString(m.viewNameInput())
	case stepSyncSelection:
		b.WriteString(m.viewSyncSelection())
	}

	return b.String()
}

// viewOrgSelection renders the organization selection view.
func (m addModel) viewOrgSelection() string {
	var b strings.Builder

	// 1. Show loading state when organizations not yet fetched
	if len(m.organizations) == 0 {
		b.WriteString("Fetching organizations...\n")
		return b.String()
	}

	// 2. Render title
	b.WriteString(m.titleStyle.Render("Select Organization"))
	b.WriteString("\n\n")

	// 3. Show contextual instruction based on organization count
	if len(m.organizations) == 1 {
		// Single org: user confirms rather than chooses
		b.WriteString("Confirm organization for this project:\n\n")
	} else {
		// Multiple orgs: user chooses from list
		b.WriteString("Choose which organization to add this project to:\n\n")
	}

	// 4. Render organization list with cursor
	for i, org := range m.organizations {
		cursor := "  "
		if m.orgCursor == i {
			cursor = m.cursorStyle.Render("> ")
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, org.Name))
	}

	// 5. Render help text
	b.WriteString("\n")
	b.WriteString(m.helpStyle.Render("up/down: navigate | enter: select | ctrl+c: cancel"))
	b.WriteString("\n")

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

// viewLocalConfigDetection renders the local config detection view.
func (m addModel) viewLocalConfigDetection() string {
	var b strings.Builder

	// 1. Initial loading state
	if m.localConfig == nil && !m.localConfigInvalid {
		b.WriteString("Checking for existing configuration...\n")
		return b.String()
	}

	// 2. Server verification in progress
	if m.serverChecking {
		b.WriteString("Verifying project on server...\n")
		b.WriteString(fmt.Sprintf("  Project ID:      %s\n", m.localConfig.ProjectID))
		b.WriteString(fmt.Sprintf("  Organization ID: %s\n", m.localConfig.OrganizationID))
		return b.String()
	}

	// 3. Invalid config: missing OrganizationID
	if m.localConfigInvalid {
		b.WriteString(m.titleStyle.Render("Invalid Local Configuration"))
		b.WriteString("\n\n")
		b.WriteString(m.errorStyle.Render("Error: "))
		b.WriteString("Local configuration is missing required 'organizationId' field.\n\n")
		b.WriteString("Found configuration:\n")
		b.WriteString(fmt.Sprintf("  Project ID:      %s\n", m.localConfig.ProjectID))
		b.WriteString(fmt.Sprintf("  Organization ID: %s\n", m.warningStyle.Render("(missing)")))
		b.WriteString("\n")
		b.WriteString("This configuration file is invalid and cannot be used.\n\n")
		b.WriteString("What would you like to do?\n\n")

		options := []string{"Reset & Register as new project", "Cancel"}
		for i, opt := range options {
			cursor := "  "
			if m.localConfigCursor == i {
				cursor = m.cursorStyle.Render("> ")
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, opt))
		}
		b.WriteString("\n")
		b.WriteString(m.helpStyle.Render("up/down: navigate | enter: select | ctrl+c: cancel"))
		return b.String()
	}

	// 4. Server project FOUND - check if already registered or not
	if m.serverProject != nil && m.serverProject.Found {
		if m.localConfig.AlreadyRegistered {
			// Case: Already registered in ~/.cops/config.json
			b.WriteString(m.titleStyle.Render("Project Already Registered"))
			b.WriteString("\n\n")

			// Show project info
			b.WriteString("This project is already registered:\n")
			b.WriteString(fmt.Sprintf("  Project ID:      %s\n", m.localConfig.ProjectID))
			b.WriteString(fmt.Sprintf("  Project Name:    %s\n", m.serverProject.Name))
			b.WriteString(fmt.Sprintf("  Organization ID: %s\n", m.serverProject.OrganizationID))
			b.WriteString("\n")

			// Ask what to do
			b.WriteString("Do you want to overwrite the existing registration?\n\n")

			// Render 2 options with cursor
			options := []string{
				"Yes - Overwrite and register again",
				"No - Cancel",
			}

			for i, opt := range options {
				cursor := "  "
				if m.localConfigCursor == i {
					cursor = m.cursorStyle.Render("> ")
				}
				b.WriteString(fmt.Sprintf("%s%s\n", cursor, opt))
			}

			b.WriteString("\n")
			b.WriteString(m.helpStyle.Render("up/down: navigate | enter: select | ctrl+c: cancel"))
			b.WriteString("\n")
			return b.String()
		}

		// Case: Not registered yet (e.g., pulled from GitHub)
		b.WriteString(m.titleStyle.Render("Existing Project Found on Server"))
		b.WriteString("\n\n")

		// Show project info from server
		b.WriteString("Local configuration matches server project:\n")
		b.WriteString(fmt.Sprintf("  Project ID:      %s\n", m.localConfig.ProjectID))
		b.WriteString(fmt.Sprintf("  Project Name:    %s\n", m.serverProject.Name))
		b.WriteString(fmt.Sprintf("  Organization ID: %s\n", m.serverProject.OrganizationID))
		b.WriteString("\n")

		// Ask what to do
		b.WriteString("What would you like to do?\n\n")

		// Render 3 options with cursor
		options := []string{
			fmt.Sprintf("Connect to this project (%s)", m.serverProject.Name),
			"Register as new project",
			"Cancel",
		}

		for i, opt := range options {
			cursor := "  "
			if m.localConfigCursor == i {
				cursor = m.cursorStyle.Render("> ")
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, opt))
		}

		b.WriteString("\n")
		b.WriteString(m.helpStyle.Render("up/down: navigate | enter: select | ctrl+c: cancel"))
		b.WriteString("\n")
		return b.String()
	}

	// 5. Server project NOT FOUND
	b.WriteString(m.titleStyle.Render("Invalid Local Configuration"))
	b.WriteString("\n\n")
	b.WriteString(m.errorStyle.Render("Error: "))
	b.WriteString("Local configuration references a project that does not exist on the server.\n\n")
	b.WriteString("Found configuration:\n")
	b.WriteString(fmt.Sprintf("  Project ID:      %s\n", m.localConfig.ProjectID))
	b.WriteString(fmt.Sprintf("  Organization ID: %s\n", m.localConfig.OrganizationID))
	b.WriteString("\n")
	b.WriteString(m.warningStyle.Render("Warning: "))
	b.WriteString("The referenced project may have been deleted or was never registered.\n\n")
	b.WriteString("What would you like to do?\n\n")

	options := []string{"Reset & Register as new project", "Cancel"}
	for i, opt := range options {
		cursor := "  "
		if m.localConfigCursor == i {
			cursor = m.cursorStyle.Render("> ")
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, opt))
	}
	b.WriteString("\n")
	b.WriteString(m.helpStyle.Render("up/down: navigate | enter: select | ctrl+c: cancel"))
	return b.String()
}
