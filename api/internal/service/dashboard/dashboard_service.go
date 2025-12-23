package dashboard

import (
	"context"
	"log/slog"

	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
)

// Service implements dashboard business logic.
type Service struct {
	logger *slog.Logger
	repo   repository.DashboardRepositoryPort
}

// NewService creates a new dashboard service.
func NewService(l *slog.Logger, repo repository.DashboardRepositoryPort) *Service {
	return &Service{
		logger: l.With(slog.String("name", "dashboard.service")),
		repo:   repo,
	}
}

// GetOverview retrieves dashboard overview statistics.
func (s *Service) GetOverview(ctx context.Context) (*repository.OverviewStats, error) {
	stats, err := s.repo.GetOverviewStats(ctx)
	if err != nil {
		s.logger.Error("failed to get overview stats", slog.Any("error", err))
		return nil, err
	}
	return stats, nil
}

// ListProjects retrieves a paginated list of projects.
func (s *Service) ListProjects(ctx context.Context, params repository.ListProjectsParams) (*repository.PaginatedProjects, error) {
	// Apply defaults
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	projects, err := s.repo.ListProjects(ctx, params)
	if err != nil {
		s.logger.Error("failed to list projects",
			slog.Int("page", int(params.Page)),
			slog.Int("pageSize", int(params.PageSize)),
			slog.Any("error", err),
		)
		return nil, err
	}

	return projects, nil
}

// GetProject retrieves detailed project information.
func (s *Service) GetProject(ctx context.Context, projectID string) (*repository.ProjectDetail, error) {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		s.logger.Error("failed to get project",
			slog.String("projectID", projectID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return project, nil
}

// ListSessions retrieves paginated sessions for a project.
func (s *Service) ListSessions(ctx context.Context, params repository.ListSessionsParams) (*repository.PaginatedSessions, error) {
	// Apply defaults
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	sessions, err := s.repo.ListSessions(ctx, params)
	if err != nil {
		s.logger.Error("failed to list sessions",
			slog.String("projectID", params.ProjectID),
			slog.Int("page", int(params.Page)),
			slog.Int("pageSize", int(params.PageSize)),
			slog.Any("error", err),
		)
		return nil, err
	}

	return sessions, nil
}

// GetSession retrieves detailed session information with all records.
func (s *Service) GetSession(ctx context.Context, sessionID string) (*repository.SessionDetail, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		s.logger.Error("failed to get session",
			slog.String("sessionID", sessionID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return session, nil
}
