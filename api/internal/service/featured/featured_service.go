package featured

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/service/featured/outbound/repository"
	orgrepo "github.com/team-attention/cops/api/internal/service/organization/outbound/repository"
)

// FeaturedMember assembles per-member stats with display names.
type FeaturedMember struct {
	UserID         string
	UserName       string
	ProjectID      string
	ProjectName    string
	PromptCount    int32
	Usage          repository.TokenUsageSummary
	ToolCalls      map[string]int32
	LastActivityAt time.Time
}

// GetFeaturedBoardResult is the assembled response from GetFeaturedBoard.
type GetFeaturedBoardResult struct {
	OrganizationName string
	Members          []*FeaturedMember
}

// FeaturedBoardService implements featured board business logic.
type FeaturedBoardService struct {
	logger        *slog.Logger
	orgRepo       orgrepo.OrganizationRepositoryPort
	dashboardRepo repository.FeaturedBoardRepositoryPort
}

// NewFeaturedBoardService creates a new featured board service.
func NewFeaturedBoardService(
	l *slog.Logger,
	orgRepo orgrepo.OrganizationRepositoryPort,
	dashboardRepo repository.FeaturedBoardRepositoryPort,
) *FeaturedBoardService {
	return &FeaturedBoardService{
		logger:        l.With(slog.String("name", "featured.service")),
		orgRepo:       orgRepo,
		dashboardRepo: dashboardRepo,
	}
}

// GetFeaturedBoard assembles the featured board for the given organization slug.
// sinceUnix is a Unix timestamp (seconds); if zero or negative, defaults to 24 hours ago.
func (s *FeaturedBoardService) GetFeaturedBoard(
	ctx context.Context,
	orgSlug string,
	sinceUnix int64,
) (*GetFeaturedBoardResult, error) {
	if orgSlug == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("org_slug is required"))
	}

	// 1. Get organization by slug
	org, err := s.orgRepo.GetBySlug(ctx, orgSlug)
	if err != nil {
		s.logger.Error("failed to get organization by slug",
			slog.String("orgSlug", orgSlug),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get organization"))
	}
	if org == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization not found: %s", orgSlug))
	}

	// 2. Get org members list
	members := org.Members
	if len(members) == 0 {
		return &GetFeaturedBoardResult{
			OrganizationName: org.Name,
			Members:          []*FeaturedMember{},
		}, nil
	}

	memberIDs := make([]string, 0, len(members))
	for _, m := range members {
		memberIDs = append(memberIDs, string(m.UserID))
	}

	// Get member names from org repo
	membersWithDetails, err := s.orgRepo.GetMembersWithDetails(ctx, string(org.ID))
	if err != nil {
		s.logger.Error("failed to get members with details",
			slog.String("orgID", string(org.ID)),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get member details"))
	}

	memberNameMap := make(map[string]string, len(membersWithDetails))
	for _, m := range membersWithDetails {
		memberNameMap[m.UserID] = m.Name
	}

	// 3. For each member, get their earliest registered project (registered_at ASC, limit 1)
	memberProjectIDs, err := s.dashboardRepo.GetEarliestProjectForMembers(ctx, string(org.ID), memberIDs)
	if err != nil {
		s.logger.Error("failed to get earliest projects for members",
			slog.String("orgID", string(org.ID)),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get member projects"))
	}

	// Collect all unique project IDs for name lookup
	uniqueProjectIDs := make([]string, 0, len(memberProjectIDs))
	seen := make(map[string]bool)
	for _, projectID := range memberProjectIDs {
		if !seen[projectID] {
			seen[projectID] = true
			uniqueProjectIDs = append(uniqueProjectIDs, projectID)
		}
	}

	// Get project names
	projectNameMap, err := s.dashboardRepo.GetProjectNames(ctx, uniqueProjectIDs)
	if err != nil {
		s.logger.Error("failed to get project names",
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get project names"))
	}

	// Filter memberIDs to only those who have a mapped project.
	// Members without a project produce zero-stat cards, so exclude them here.
	filteredMemberIDs := make([]string, 0, len(memberProjectIDs))
	for _, id := range memberIDs {
		if _, ok := memberProjectIDs[id]; ok {
			filteredMemberIDs = append(filteredMemberIDs, id)
		}
	}

	// Determine since time
	var since time.Time
	if sinceUnix > 0 {
		since = time.Unix(sinceUnix, 0)
	} else {
		since = time.Now().UTC().Add(-24 * time.Hour)
	}

	// 4. Get featured board stats for members with projects only
	stats, err := s.dashboardRepo.GetFeaturedBoardStats(ctx, memberProjectIDs, filteredMemberIDs, since)
	if err != nil {
		s.logger.Error("failed to get featured board stats",
			slog.String("orgID", string(org.ID)),
			slog.Any("error", err),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get featured board stats"))
	}

	// 5. Assemble response
	featuredMembers := make([]*FeaturedMember, 0, len(stats))
	for _, stat := range stats {
		fm := &FeaturedMember{
			UserID:         stat.UserID,
			UserName:       memberNameMap[stat.UserID],
			ProjectID:      stat.ProjectID,
			ProjectName:    projectNameMap[stat.ProjectID],
			PromptCount:    stat.PromptCount,
			Usage:          stat.Usage,
			ToolCalls:      stat.ToolCalls,
			LastActivityAt: stat.LastActivityAt,
		}
		featuredMembers = append(featuredMembers, fm)
	}

	return &GetFeaturedBoardResult{
		OrganizationName: org.Name,
		Members:          featuredMembers,
	}, nil
}
