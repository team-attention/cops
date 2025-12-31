package repository

import (
	"context"
	"time"

	"github.com/team-attention/cops/api/internal/platform/structure"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

// TokenUsageSummary aggregates token usage statistics.
type TokenUsageSummary struct {
	TotalInputTokens         int64
	TotalOutputTokens        int64
	TotalCacheCreationTokens int64
	TotalCacheReadTokens     int64
}

// ProjectAggregation contains computed/aggregated fields for a project.
// These are derived from session records, not stored with the project itself.
type ProjectAggregation struct {
	SessionCount int32
	Usage        TokenUsageSummary
	LastActivity time.Time
}

// ProjectSummary embeds ProjectAbstract with aggregation data.
// Used for list views and overview displays.
type ProjectSummary struct {
	shareddomain.ProjectAbstract
	GitBranch string
	ProjectAggregation
}

// ProjectDetail embeds full Project with aggregation data.
// Used for detailed project views.
type ProjectDetail struct {
	shareddomain.Project
	ProjectAggregation
}

// SessionBase contains common identification fields for a session.
// Used for embedding into SessionSummary and SessionDetail.
type SessionBase struct {
	ID        string
	ProjectID string
	GitBranch string
	StartedAt time.Time
	EndedAt   time.Time
}

// SessionSummary contains session overview data.
// Embeds SessionBase for common identification.
type SessionSummary struct {
	SessionBase
	MessageCount int32
	Usage        TokenUsageSummary
}

// SessionDetail contains full session information with records.
// Embeds SessionBase for common identification.
type SessionDetail struct {
	SessionBase
	Version string
	Usage   TokenUsageSummary
	Records []shareddomain.Record
}

// OverviewStats contains dashboard overview statistics.
type OverviewStats struct {
	TotalUsage     TokenUsageSummary
	ProjectCount   int32
	SessionCount   int32
	RecentProjects []ProjectSummary
	RecentSessions []SessionSummary
}

// ListProjectsQuery contains filter conditions for listing projects.
type ListProjectsQuery struct{}

// ListSessionsQuery contains filter conditions for listing sessions.
type ListSessionsQuery struct {
	ProjectID string
	SortBy    string
	SortDesc  bool
}

// ListProjectsParams is a type alias for paginated project queries.
type ListProjectsParams = structure.PaginationParams[ListProjectsQuery]

// ListSessionsParams is a type alias for paginated session queries.
type ListSessionsParams = structure.PaginationParams[ListSessionsQuery]

// PaginatedProjects is a type alias for paginated project results.
type PaginatedProjects = structure.PaginatedResult[ProjectSummary]

// PaginatedSessions is a type alias for paginated session results.
type PaginatedSessions = structure.PaginatedResult[SessionSummary]

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
