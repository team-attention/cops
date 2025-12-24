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
	GitProject   bool      `json:"gitProject"`             // true if git repo, false otherwise
	ClaudeDir    string    `json:"claudeDir"`              // Claude project directory path
	GitBranch    string    `json:"gitBranch,omitempty"`    // Current git branch
	Worktrees    []string  `json:"worktrees,omitempty"`    // Worktree paths for git projects
	RegisteredAt time.Time `json:"registeredAt"`           // When the project was registered
}
