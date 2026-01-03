package user

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/team-attention/cops/api/internal/service/user/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
)

// GetMeResult contains the user data and their organizations.
type GetMeResult struct {
	User          *domain.User
	Organizations []*repository.UserOrganization
}

// Service implements user business logic.
type Service struct {
	logger   *slog.Logger
	userRepo repository.UserRepositoryPort
	orgRepo  repository.OrganizationRepositoryPort
}

// NewService creates a new user service.
func NewService(
	l *slog.Logger,
	userRepo repository.UserRepositoryPort,
	orgRepo repository.OrganizationRepositoryPort,
) *Service {
	return &Service{
		logger:   l.With(slog.String("name", "user.service")),
		userRepo: userRepo,
		orgRepo:  orgRepo,
	}
}

// GetMe retrieves the authenticated user's information and organizations.
func (s *Service) GetMe(ctx context.Context, userID string) (*GetMeResult, error) {
	// 1. Validate userID is not empty.
	if userID == "" {
		s.logger.Warn("empty userID provided to GetMe")
		return nil, fmt.Errorf("userID is required")
	}

	// 2. Call userRepo.GetByID to fetch user data.
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		// If error, log error and return nil, error.
		s.logger.Error("failed to get user by ID",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}

	// If user is nil (not found), log info and return error "user not found".
	if user == nil {
		s.logger.Info("user not found",
			slog.String("userID", userID),
		)
		return nil, fmt.Errorf("user not found")
	}

	// 3. Call orgRepo.GetUserOrganizations to fetch user's organizations.
	organizations, err := s.orgRepo.GetUserOrganizations(ctx, userID)
	if err != nil {
		// If error, log error and return nil, error.
		s.logger.Error("failed to get user organizations",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}

	// 4. Log successful retrieval with user ID and organization count.
	s.logger.Info("successfully retrieved user data",
		slog.String("userID", userID),
		slog.Int("organizationCount", len(organizations)),
	)

	// 5. Return GetMeResult with user and organizations.
	return &GetMeResult{
		User:          user,
		Organizations: organizations,
	}, nil
}
