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
	ProjectPath string               // Original project path from GlobalConfig
	ClaudeDir   string               // ~/.claude/projects/{encoded-path}
	Type        WatchTargetType      // Type of watch target
	ProjectID   shareddomain.ID      // Project ID from local config
}

// FilePosition tracks read position for incremental file reading.
type FilePosition struct {
	Path      string    // File path
	Offset    int64     // Last read byte offset
	UpdatedAt time.Time // Last update time
}

// LogBatch contains raw JSONL lines for API transmission.
type LogBatch struct {
	Lines     []string        // Raw JSONL lines (unparsed)
	ProjectID shareddomain.ID // Project ID (for aggregation API)
}
