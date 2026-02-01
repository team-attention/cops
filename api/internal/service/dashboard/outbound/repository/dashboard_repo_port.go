package repository

import (
	"context"
	"time"

	"github.com/team-attention/cops/api/internal/platform/structure"
	shareddomain "github.com/team-attention/cops/shared/domain"
	session "github.com/team-attention/cops/shared/domain/v2"
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

// SessionDetail contains full session information with session entries.
// Embeds SessionBase for common identification.
type SessionDetail struct {
	SessionBase
	Version  string
	Usage    TokenUsageSummary
	Sessions []*session.Session
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
type ListProjectsQuery struct {
	OrganizationID string
}

// ListSessionsQuery contains filter conditions for listing sessions.
type ListSessionsQuery struct {
	OrganizationID string
	ProjectID      string
	SortBy         string
	SortDesc       bool
}

// GetSessionQuery contains filter conditions for retrieving a session.
type GetSessionQuery struct {
	OrganizationID string
	SessionID      string
	AgentID        *string // nil = all agents, "main" = Main session only, other = specific SubAgent
}

// GetSessionSegmentsParams contains filter conditions for retrieving session segments.
type GetSessionSegmentsParams struct {
	OrganizationID string
	SessionID      string
}

// SessionSegmentInfo contains segment summary information.
type SessionSegmentInfo struct {
	ID           string
	Label        string
	StartTime    time.Time
	EndTime      time.Time
	MessageCount int32
}

// SessionSegmentsResult contains all segments and time range for a session.
type SessionSegmentsResult struct {
	Segments       []*SessionSegmentInfo
	StartTime      time.Time
	EndTime        time.Time
	TotalDurationS int64
}

// ListProjectsParams is a type alias for paginated project queries.
type ListProjectsParams = structure.PaginationParams[ListProjectsQuery]

// ListSessionsParams is a type alias for paginated session queries.
type ListSessionsParams = structure.PaginationParams[ListSessionsQuery]

// GetSessionParams is a type alias for paginated session detail queries.
type GetSessionParams = structure.PaginationParams[GetSessionQuery]

// PaginatedProjects is a type alias for paginated project results.
type PaginatedProjects = structure.PaginatedResult[ProjectSummary]

// PaginatedSessions is a type alias for paginated session results.
type PaginatedSessions = structure.PaginatedResult[SessionSummary]

// PaginatedSessionDetail contains session detail with pagination metadata for transcripts.
type PaginatedSessionDetail struct {
	SessionDetail
	structure.PaginationMeta
}

// DashboardRepositoryPort defines the interface for dashboard data access.
type DashboardRepositoryPort interface {
	// GetOverviewStats retrieves dashboard overview statistics for an organization.
	GetOverviewStats(ctx context.Context, organizationID string) (*OverviewStats, error)

	// ListProjects retrieves a paginated list of projects for an organization.
	ListProjects(ctx context.Context, params ListProjectsParams) (*PaginatedProjects, error)

	// GetProject retrieves detailed project information.
	// Returns nil, errutil.NotFound if project not found or does not belong to organization.
	GetProject(ctx context.Context, organizationID, projectID string) (*ProjectDetail, error)

	// ListSessions retrieves paginated sessions for a project in an organization.
	// Returns empty result if project does not belong to organization.
	ListSessions(ctx context.Context, params ListSessionsParams) (*PaginatedSessions, error)

	// GetSession retrieves detailed session information with paginated transcripts.
	// Returns nil, errutil.NotFound if session not found or its project does not belong to organization.
	GetSession(ctx context.Context, params GetSessionParams) (*PaginatedSessionDetail, error)

	// GetSessionSegments retrieves lightweight segment summaries for timeline display.
	// Returns nil, errutil.NotFound if session not found or its project does not belong to organization.
	GetSessionSegments(ctx context.Context, params GetSessionSegmentsParams) (*SessionSegmentsResult, error)
}
