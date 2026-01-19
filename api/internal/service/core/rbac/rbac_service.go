package rbac

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository"
)

// Service implements RBAC business logic.
// ONLY checks organization membership - no project-level checks.
type Service struct {
	logger     *slog.Logger
	memberRepo repository.OrganizationMemberRepositoryPort
}

// NewService creates a new RBAC service.
func NewService(
	l *slog.Logger,
	memberRepo repository.OrganizationMemberRepositoryPort,
) *Service {
	return &Service{
		logger:     l.With(slog.String("name", "rbac.service")),
		memberRepo: memberRepo,
	}
}

// CanAccessOrganization checks if a user is a member of an organization.
// Returns true if user is a member (any role: admin or member).
// Returns false, nil if user is not a member.
// Returns false, error if a system error occurs.
func (s *Service) CanAccessOrganization(ctx context.Context, userID, organizationID string) (bool, error) {
	if userID == "" {
		s.logger.Warn("userID is required")
		return false, fmt.Errorf("userID is required")
	}

	if organizationID == "" {
		s.logger.Warn("organizationID is required")
		return false, fmt.Errorf("organizationID is required")
	}

	isMember, err := s.memberRepo.IsMember(ctx, userID, organizationID)
	if err != nil {
		s.logger.Error("failed to check membership",
			slog.String("userID", userID),
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return false, err
	}

	if !isMember {
		s.logger.Info("access denied: user is not member of organization",
			slog.String("userID", userID),
			slog.String("organizationID", organizationID),
		)
	} else {
		s.logger.Debug("access granted",
			slog.String("userID", userID),
			slog.String("organizationID", organizationID),
		)
	}

	return isMember, nil
}
