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
	case stepParentDetection:
		b.WriteString(m.viewParentConfirmation())
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

	if len(m.organizations) == 0 {
		b.WriteString("Fetching organizations...\n")
		return b.String()
	}

	b.WriteString(m.titleStyle.Render("Select Organization"))
	b.WriteString("\n\n")
	b.WriteString("Choose which organization to add this project to:\n\n")

	for i, org := range m.organizations {
		cursor := "  "
		if m.orgCursor == i {
			cursor = m.cursorStyle.Render("> ")
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, org.Name))
	}

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

// viewParentConfirmation renders the parent project confirmation view.
func (m addModel) viewParentConfirmation() string {
	var b strings.Builder

	if m.parentProject == nil {
		b.WriteString("Checking for parent projects...\n")
		return b.String()
	}

	b.WriteString(m.titleStyle.Render("Parent Project Detected"))
	b.WriteString("\n\n")

	b.WriteString("This directory is already being tracked as a subdirectory of:\n")
	b.WriteString(fmt.Sprintf("  Project: %s\n", m.parentProject.Name))
	b.WriteString(fmt.Sprintf("  Path:    %s\n", m.parentProject.Path))
	b.WriteString("\n")

	b.WriteString("Do you want to register it as a separate project?\n\n")

	options := []string{
		"Yes - Register as separate project",
		"No - Cancel registration",
	}

	for i, opt := range options {
		cursor := "  "
		if m.parentCursor == i {
			cursor = m.cursorStyle.Render("> ")
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, opt))
	}

	b.WriteString("\n")
	b.WriteString(m.helpStyle.Render("up/down: navigate | enter: select | y/n: quick select | ctrl+c: cancel"))
	b.WriteString("\n")

	return b.String()
}
