package domain

import "time"

// ProjectAbstract contains the minimal identification fields for a project.
// Used for summary views and embedding into full Project.
type ProjectAbstract struct {
	ID   ID     `json:"id"`   // Unique identifier
	Name string `json:"name"` // Display name
	Path string `json:"path"` // Absolute path (main repo for git projects)
}

// Project represents a registered project for session tracking.
// Embeds ProjectAbstract for basic identification.
type Project struct {
	ProjectAbstract
	IsGitProject bool      `json:"gitProject"`   // true if git repo, false otherwise
	RegisteredAt time.Time `json:"registeredAt"` // When the project was registered
}

// ProjectWithWorktrees represents a project with dynamically discovered worktrees.
// Used for listing projects with their git worktree information.
type ProjectWithWorktrees struct {
	Project
	Worktrees []string `json:"worktrees,omitempty"` // Dynamically discovered worktree paths
}
