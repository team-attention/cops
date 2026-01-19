package project

import (
	"context"
	"log/slog"

	"github.com/team-attention/cops/api/internal/service/project/outbound/repository"
)

// RegisterProjectParams contains parameters for registering a project.
type RegisterProjectParams struct {
	ConfiguredRemoteURL string
	ActualRemoteURL     string
	ExistingProjectID   string
	Name                string
	IsGitProject        bool
	OrganizationID      string
}

// Service implements project business logic.
type Service struct {
	logger *slog.Logger
	repo   repository.ProjectRepositoryPort
}

// NewService creates a new project service.
func NewService(l *slog.Logger, repo repository.ProjectRepositoryPort) *Service {
	return &Service{
		logger: l.With(slog.String("name", "project.service")),
		repo:   repo,
	}
}

// RegisterProject registers a project or returns existing project ID if already registered.
func (s *Service) RegisterProject(ctx context.Context, params RegisterProjectParams) (*repository.FindOrCreateResult, error) {
	result, err := s.repo.FindOrCreate(ctx, repository.FindOrCreateParams{
		ConfiguredURL:  params.ConfiguredRemoteURL,
		ActualURL:      params.ActualRemoteURL,
		ExistingID:     params.ExistingProjectID,
		Name:           params.Name,
		IsGitProject:   params.IsGitProject,
		OrganizationID: params.OrganizationID,
	})
	if err != nil {
		s.logger.Error("failed to register project", slog.Any("error", err))
		return nil, err
	}

	s.logger.Info("project registered",
		slog.String("projectID", result.ProjectID),
		slog.Bool("isNew", result.IsNew),
		slog.String("name", result.Name),
		slog.Bool("isGitProject", result.IsGitProject),
		slog.String("organizationID", result.OrganizationID))

	return result, nil
}

// GetProjectByIDAndOrgResult contains the result of GetProjectByIDAndOrg operation.
type GetProjectByIDAndOrgResult struct {
	Found          bool
	ProjectID      string
	Name           string
	OrganizationID string
}

// GetProjectByIDAndOrg retrieves a project by its ID within a specific organization.
func (s *Service) GetProjectByIDAndOrg(ctx context.Context, projectID string, organizationID string) (*GetProjectByIDAndOrgResult, error) {
	result, err := s.repo.GetByIDAndOrg(ctx, projectID, organizationID)
	if err != nil {
		s.logger.Error("failed to get project by ID and org",
			slog.String("projectID", projectID),
			slog.String("organizationID", organizationID),
			slog.Any("error", err))
		return nil, err
	}

	return &GetProjectByIDAndOrgResult{
		Found:          result.Found,
		ProjectID:      result.ProjectID,
		Name:           result.Name,
		OrganizationID: result.OrganizationID,
	}, nil
}
