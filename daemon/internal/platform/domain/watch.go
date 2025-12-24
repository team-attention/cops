package domain

import (
	"time"

	shareddomain "github.com/team-attention/cops/shared/domain"
)

// WatchTargetType represents the type of watch target.
type WatchTargetType string

const (
	// WatchTargetRoot is the main project directory.
	WatchTargetRoot WatchTargetType = "root"
	// WatchTargetWorktree is a Git worktree directory.
	WatchTargetWorktree WatchTargetType = "worktree"
	// WatchTargetSubdirectory is a subdirectory within the project.
	WatchTargetSubdirectory WatchTargetType = "subdirectory"
)

// WatchTarget represents a directory to watch for Claude Code logs.
type WatchTarget struct {
	ProjectPath string          // Original project path from GlobalConfig
	ClaudeDir   string          // ~/.claude/projects/{encoded-path}
	Type        WatchTargetType // Type of watch target
}

// FilePosition tracks read position for incremental file reading.
type FilePosition struct {
	Path      string    // File path
	Offset    int64     // Last read byte offset
	UpdatedAt time.Time // Last update time
}

// LogBatch contains multiple session records for API transmission.
type LogBatch struct {
	Records      []shareddomain.SessionRecord // Session records from shared domain
	DaemonID     string                       // Daemon instance ID
	CreatedAt    time.Time                    // Batch creation time
	ProjectID    string                       // Project ID (for collector API)
	ProjectName  string                       // Project name (for collector API)
	ProjectPath  string                       // Project path (for collector API)
	IsGitProject bool                         // Whether project is git repo
}
