package repository

import (
	"context"
	"time"

	dashboardrepo "github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
)

// TokenUsageSummary is an alias for the dashboard repository's TokenUsageSummary.
type TokenUsageSummary = dashboardrepo.TokenUsageSummary

// FeaturedMemberStats contains aggregated metrics for a single member.
type FeaturedMemberStats struct {
	UserID         string
	ProjectID      string
	PromptCount    int32
	Usage          TokenUsageSummary
	ToolCalls      map[string]int32
	LastActivityAt time.Time
}

// FeaturedBoardRepositoryPort defines the interface for featured board data access.
type FeaturedBoardRepositoryPort interface {
	// GetEarliestProjectForMembers returns the earliest registered project (by registeredAt ASC)
	// for each member within the given organization.
	// Returns a map of UserID -> ProjectID for members that have at least one project in the org.
	GetEarliestProjectForMembers(ctx context.Context, organizationID string, memberIDs []string) (map[string]string, error)

	// GetProjectNames returns a map of ProjectID -> ProjectName for the given project IDs.
	GetProjectNames(ctx context.Context, projectIDs []string) (map[string]string, error)

	// GetFeaturedBoardStats retrieves per-member aggregated metrics.
	// projectIDs maps each member's UserID to their earliest registered ProjectID.
	// memberIDs is the list of user IDs to include.
	// since filters activity to only include records after this time.
	GetFeaturedBoardStats(ctx context.Context, projectIDs map[string]string, memberIDs []string, since time.Time) ([]*FeaturedMemberStats, error)
}
