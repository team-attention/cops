package dashboard

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"

	"github.com/team-attention/cops/api/internal/service/core/rbac"
	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
)

// Service implements dashboard business logic.
// Injects RBAC service for authorization checks.
type Service struct {
	logger  *slog.Logger
	repo    repository.DashboardRepositoryPort
	rbacSvc *rbac.Service
}

// NewService creates a new dashboard service.
func NewService(l *slog.Logger, repo repository.DashboardRepositoryPort, rbacSvc *rbac.Service) *Service {
	return &Service{
		logger:  l.With(slog.String("name", "dashboard.service")),
		repo:    repo,
		rbacSvc: rbacSvc,
	}
}

// checkRBAC validates organization access for the user.
// Returns connect.Error if validation fails.
func (s *Service) checkRBAC(ctx context.Context, userID, organizationID string) error {
	if organizationID == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	if userID == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user not authenticated"))
	}

	canAccess, err := s.rbacSvc.CanAccessOrganization(ctx, userID, organizationID)
	if err != nil {
		s.logger.Error("failed to check access",
			slog.String("userID", userID),
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check access"))
	}

	if !canAccess {
		s.logger.Info("access denied to organization",
			slog.String("userID", userID),
			slog.String("organizationID", organizationID),
		)
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("access denied to organization"))
	}

	return nil
}

// GetOverview retrieves dashboard overview statistics for an organization.
func (s *Service) GetOverview(ctx context.Context, userID, organizationID string) (*repository.OverviewStats, error) {
	if err := s.checkRBAC(ctx, userID, organizationID); err != nil {
		return nil, err
	}

	stats, err := s.repo.GetOverviewStats(ctx, organizationID)
	if err != nil {
		s.logger.Error("failed to get overview stats",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return nil, err
	}
	return stats, nil
}

// ListProjects retrieves a paginated list of projects for an organization.
func (s *Service) ListProjects(ctx context.Context, userID string, params repository.ListProjectsParams) (*repository.PaginatedProjects, error) {
	if err := s.checkRBAC(ctx, userID, params.Query.OrganizationID); err != nil {
		return nil, err
	}

	// Apply defaults
	params.SetDefaults()

	projects, err := s.repo.ListProjects(ctx, params)
	if err != nil {
		s.logger.Error("failed to list projects",
			slog.String("organizationID", params.Query.OrganizationID),
			slog.Int("page", int(params.Page)),
			slog.Int("pageSize", int(params.PageSize)),
			slog.Any("error", err),
		)
		return nil, err
	}

	return projects, nil
}

// GetProject retrieves detailed project information.
// Validates RBAC and project belongs to organization.
func (s *Service) GetProject(ctx context.Context, userID, organizationID, projectID string) (*repository.ProjectDetail, error) {
	if err := s.checkRBAC(ctx, userID, organizationID); err != nil {
		return nil, err
	}

	project, err := s.repo.GetProject(ctx, organizationID, projectID)
	if err != nil {
		s.logger.Error("failed to get project",
			slog.String("organizationID", organizationID),
			slog.String("projectID", projectID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return project, nil
}

// ListSessions retrieves paginated sessions for a project.
// Validates RBAC and project belongs to organization.
func (s *Service) ListSessions(ctx context.Context, userID string, params repository.ListSessionsParams) (*repository.PaginatedSessions, error) {
	if err := s.checkRBAC(ctx, userID, params.Query.OrganizationID); err != nil {
		return nil, err
	}

	// Apply defaults
	params.SetDefaults()

	sessions, err := s.repo.ListSessions(ctx, params)
	if err != nil {
		s.logger.Error("failed to list sessions",
			slog.String("organizationID", params.Query.OrganizationID),
			slog.String("projectID", params.Query.ProjectID),
			slog.Int("page", int(params.Page)),
			slog.Int("pageSize", int(params.PageSize)),
			slog.Any("error", err),
		)
		return nil, err
	}

	return sessions, nil
}

// GetSession retrieves detailed session information with all records.
// Validates RBAC and session's project belongs to organization.
func (s *Service) GetSession(ctx context.Context, userID, organizationID, sessionID string) (*repository.SessionDetail, error) {
	if err := s.checkRBAC(ctx, userID, organizationID); err != nil {
		return nil, err
	}

	session, err := s.repo.GetSession(ctx, organizationID, sessionID)
	if err != nil {
		s.logger.Error("failed to get session",
			slog.String("organizationID", organizationID),
			slog.String("sessionID", sessionID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return session, nil
}
