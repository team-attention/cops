package repository

import (
	"context"
	"time"

	shareddomain "github.com/team-attention/cops/shared/domain"
)

// TokenUsageSummary aggregates token usage statistics.
type TokenUsageSummary struct {
	TotalInputTokens         int64
	TotalOutputTokens        int64
	TotalCacheCreationTokens int64
	TotalCacheReadTokens     int64
}

// ProjectSummary contains project overview data.
type ProjectSummary struct {
	ID           string
	Name         string
	Path         string
	GitBranch    string
	SessionCount int32
	Usage        TokenUsageSummary
	LastActivity time.Time
}

// ProjectDetail contains full project information.
type ProjectDetail struct {
	ID           string
	Name         string
	Path         string
	GitBranch    string
	Worktrees    []string
	SessionCount int32
	Usage        TokenUsageSummary
	CreatedAt    time.Time
	LastActivity time.Time
}

// SessionSummary contains session overview data.
type SessionSummary struct {
	ID           string
	ProjectID    string
	GitBranch    string
	MessageCount int32
	Usage        TokenUsageSummary
	StartedAt    time.Time
	EndedAt      time.Time
}

// SessionDetail contains full session information with records.
type SessionDetail struct {
	ID        string
	ProjectID string
	GitBranch string
	CWD       string
	Version   string
	Usage     TokenUsageSummary
	StartedAt time.Time
	EndedAt   time.Time
	Records   []shareddomain.SessionRecord
}

// OverviewStats contains dashboard overview statistics.
type OverviewStats struct {
	TotalUsage     TokenUsageSummary
	ProjectCount   int32
	SessionCount   int32
	RecentProjects []ProjectSummary
	RecentSessions []SessionSummary
}

// ListProjectsParams contains parameters for listing projects.
type ListProjectsParams struct {
	Page     int32
	PageSize int32
	Search   string
	SortBy   string
	SortDesc bool
}

// ListSessionsParams contains parameters for listing sessions.
type ListSessionsParams struct {
	ProjectID string
	Page      int32
	PageSize  int32
	SortBy    string
	SortDesc  bool
}

// PaginatedProjects contains paginated project results.
type PaginatedProjects struct {
	Projects    []ProjectSummary
	CurrentPage int32
	PageSize    int32
	TotalPages  int32
	TotalCount  int64
}

// PaginatedSessions contains paginated session results.
type PaginatedSessions struct {
	Sessions    []SessionSummary
	CurrentPage int32
	PageSize    int32
	TotalPages  int32
	TotalCount  int64
}

// DashboardRepositoryPort defines the interface for dashboard data access.
type DashboardRepositoryPort interface {
	// GetOverviewStats retrieves dashboard overview statistics.
	GetOverviewStats(ctx context.Context) (*OverviewStats, error)

	// ListProjects retrieves a paginated list of projects.
	ListProjects(ctx context.Context, params ListProjectsParams) (*PaginatedProjects, error)

	// GetProject retrieves detailed project information.
	GetProject(ctx context.Context, projectID string) (*ProjectDetail, error)

	// ListSessions retrieves paginated sessions for a project.
	ListSessions(ctx context.Context, params ListSessionsParams) (*PaginatedSessions, error)

	// GetSession retrieves detailed session information with all records.
	GetSession(ctx context.Context, sessionID string) (*SessionDetail, error)
}
