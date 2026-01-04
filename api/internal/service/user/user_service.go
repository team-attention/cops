package user

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/team-attention/cops/api/internal/service/user/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
)

const (
	// DeleteConfirmationPhrase is the required confirmation phrase for account deletion.
	DeleteConfirmationPhrase = "DELETE"
)

// GetMeResult contains the user data and their organizations.
type GetMeResult struct {
	User          *domain.User
	Organizations []*repository.UserOrganization
}

// DeleteAccountResult contains the result of account deletion.
type DeleteAccountResult struct {
	Success bool
	Message string
}

// Service implements user business logic.
type Service struct {
	logger            *slog.Logger
	userRepo          repository.UserRepositoryPort
	orgRepo           repository.OrganizationRepositoryPort
	cascadeDeleteRepo repository.CascadeDeleteRepositoryPort
}

// NewService creates a new user service.
func NewService(
	l *slog.Logger,
	userRepo repository.UserRepositoryPort,
	orgRepo repository.OrganizationRepositoryPort,
	cascadeDeleteRepo repository.CascadeDeleteRepositoryPort,
) *Service {
	return &Service{
		logger:            l.With(slog.String("name", "user.service")),
		userRepo:          userRepo,
		orgRepo:           orgRepo,
		cascadeDeleteRepo: cascadeDeleteRepo,
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

// DeleteAccount permanently deletes the authenticated user's account and related data.
func (s *Service) DeleteAccount(ctx context.Context, userID, confirmationPhrase string) (*DeleteAccountResult, error) {
	// 1. Validate confirmationPhrase equals DeleteConfirmationPhrase.
	//    a. If not, log warn with userID and return error "confirmation phrase must be 'DELETE'".
	if confirmationPhrase != DeleteConfirmationPhrase {
		s.logger.Warn("invalid confirmation phrase for account deletion",
			slog.String("userID", userID),
		)
		return nil, fmt.Errorf("confirmation phrase must be 'DELETE'")
	}

	// 2. Validate userID is not empty.
	//    a. If empty, return error "userID is required".
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	// 3. Call userRepo.GetByID to verify user exists.
	user, err := s.userRepo.GetByID(ctx, userID)
	//    a. If error, log error and return nil, error.
	if err != nil {
		s.logger.Error("failed to get user for deletion",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}
	//    b. If user is nil, log info and return nil, error "user not found".
	if user == nil {
		s.logger.Info("user not found for deletion",
			slog.String("userID", userID),
		)
		return nil, fmt.Errorf("user not found")
	}

	// 4. Call orgRepo.GetUserOrganizationsWithMemberCount to get all user's organizations with member counts.
	orgsWithCount, err := s.orgRepo.GetUserOrganizationsWithMemberCount(ctx, userID)
	//    a. If error, log error and return nil, error.
	if err != nil {
		s.logger.Error("failed to get user organizations for deletion",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}

	// 5. Iterate through organizations:
	for _, orgWithCount := range orgsWithCount {
		orgID := string(orgWithCount.Organization.ID)

		//    a. If MemberCount == 1 (user is sole member):
		if orgWithCount.MemberCount == 1 {
			//       i. Call cascadeDeleteRepo.DeleteSessionRecordsByOrganization.
			err = s.cascadeDeleteRepo.DeleteSessionRecordsByOrganization(ctx, orgID)
			//          - If error, log error and return nil, error.
			if err != nil {
				s.logger.Error("failed to delete session records for organization",
					slog.String("userID", userID),
					slog.String("organizationID", orgID),
					slog.Any("error", err),
				)
				return nil, err
			}

			//       ii. Call cascadeDeleteRepo.DeleteProjectsByOrganization.
			err = s.cascadeDeleteRepo.DeleteProjectsByOrganization(ctx, orgID)
			//          - If error, log error and return nil, error.
			if err != nil {
				s.logger.Error("failed to delete projects for organization",
					slog.String("userID", userID),
					slog.String("organizationID", orgID),
					slog.Any("error", err),
				)
				return nil, err
			}

			//       iii. Call orgRepo.DeleteOrganization.
			err = s.orgRepo.DeleteOrganization(ctx, orgID)
			//          - If error, log error and return nil, error.
			if err != nil {
				s.logger.Error("failed to delete organization",
					slog.String("userID", userID),
					slog.String("organizationID", orgID),
					slog.Any("error", err),
				)
				return nil, err
			}

			//       iv. Log info about cascade deletion with orgID.
			s.logger.Info("cascade deleted organization (sole member)",
				slog.String("userID", userID),
				slog.String("organizationID", orgID),
			)
		} else {
			//    b. If MemberCount > 1 (shared organization):
			//       i. Call orgRepo.RemoveUserFromOrganization.
			err = s.orgRepo.RemoveUserFromOrganization(ctx, orgID, userID)
			//          - If error, log error and return nil, error.
			if err != nil {
				s.logger.Error("failed to remove user from organization",
					slog.String("userID", userID),
					slog.String("organizationID", orgID),
					slog.Any("error", err),
				)
				return nil, err
			}

			//       ii. Log info about membership removal with orgID.
			s.logger.Info("removed user membership from shared organization",
				slog.String("userID", userID),
				slog.String("organizationID", orgID),
				slog.Int("remainingMembers", orgWithCount.MemberCount-1),
			)
		}
	}

	// 6. Call userRepo.Delete to delete user profile.
	err = s.userRepo.Delete(ctx, userID)
	//    a. If error, log error and return nil, error.
	if err != nil {
		s.logger.Error("failed to delete user",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}

	// 7. Log info with userID about successful account deletion.
	s.logger.Info("successfully deleted user account",
		slog.String("userID", userID),
		slog.Int("organizationsProcessed", len(orgsWithCount)),
	)

	// 8. Return &DeleteAccountResult{Success: true, Message: "Account deleted successfully"}, nil.
	return &DeleteAccountResult{
		Success: true,
		Message: "Account deleted successfully",
	}, nil
}
