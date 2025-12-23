package domain

import "time"

// Project represents a registered project for session tracking.
type Project struct {
	ID           ID        `json:"id"`           // UUID
	Name         string    `json:"name"`         // Display name
	Path         string    `json:"path"`         // Absolute path (main repo for git projects)
	GitProject   bool      `json:"gitProject"`   // true if git repo, false otherwise
	ClaudeDir    string    `json:"claudeDir"`    // Claude project directory path
	RegisteredAt time.Time `json:"registeredAt"` // When the project was registered
}

// ProjectWithWorktrees extends Project with discovered worktree paths.
type ProjectWithWorktrees struct {
	Project
	Worktrees []string `json:"worktrees,omitempty"` // Worktree paths for git projects
}
