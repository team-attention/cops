package rbac

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository"
)

// Service implements RBAC business logic.
type Service struct {
	logger      *slog.Logger
	projectRepo repository.ProjectRepositoryPort
	memberRepo  repository.OrganizationMemberRepositoryPort
}

// NewService creates a new RBAC service.
func NewService(
	l *slog.Logger,
	projectRepo repository.ProjectRepositoryPort,
	memberRepo repository.OrganizationMemberRepositoryPort,
) *Service {
	return &Service{
		logger:      l.With(slog.String("name", "rbac.service")),
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
	}
}

// CanAccess checks if a user can access a project.
// Returns true if user is a member of the project's organization.
// Returns false, nil if access is denied (project not found, not a member).
// Returns false, error if a system error occurs.
func (s *Service) CanAccess(ctx context.Context, userID, projectID string) (bool, error) {
	if userID == "" {
		s.logger.Warn("userID is required")
		return false, fmt.Errorf("userID is required")
	}

	if projectID == "" {
		s.logger.Warn("projectID is required")
		return false, fmt.Errorf("projectID is required")
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		s.logger.Error("failed to get project",
			slog.String("projectID", projectID),
			slog.Any("error", err),
		)
		return false, err
	}

	if project == nil {
		s.logger.Info("project not found",
			slog.String("projectID", projectID),
		)
		return false, nil
	}

	isMember, err := s.memberRepo.IsMember(ctx, userID, string(project.OrganizationID))
	if err != nil {
		s.logger.Error("failed to check membership",
			slog.String("userID", userID),
			slog.String("organizationID", string(project.OrganizationID)),
			slog.Any("error", err),
		)
		return false, err
	}

	if !isMember {
		s.logger.Info("access denied: user is not member of organization",
			slog.String("userID", userID),
			slog.String("projectID", projectID),
			slog.String("organizationID", string(project.OrganizationID)),
		)
	} else {
		s.logger.Debug("access granted",
			slog.String("userID", userID),
			slog.String("projectID", projectID),
			slog.String("organizationID", string(project.OrganizationID)),
		)
	}

	return isMember, nil
}
