package organization

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/team-attention/cops/api/internal/service/organization/outbound/repository"
	userrepo "github.com/team-attention/cops/api/internal/service/user/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
)

const (
	// SlugMinLength is the minimum length for organization slug.
	SlugMinLength = 3
	// SlugMaxLength is the maximum length for organization slug.
	SlugMaxLength = 63
)

// SlugPattern validates organization slug format.
// Lowercase alphanumeric with hyphens, no leading/trailing hyphens.
var SlugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// UpdateOrganizationResult contains the updated organization.
type UpdateOrganizationResult struct {
	Organization *domain.Organization
}

// GetOrganizationMembersResult contains members with details.
type GetOrganizationMembersResult struct {
	Members []*repository.MemberWithDetails
}

// LeaveOrganizationResult contains the result of leaving.
type LeaveOrganizationResult struct {
	Success            bool
	IsLastOrganization bool
}

// CreateOrganizationResult contains the created organization.
type CreateOrganizationResult struct {
	Organization *domain.Organization
}

// Service implements organization business logic.
type Service struct {
	logger            *slog.Logger
	orgRepo           repository.OrganizationRepositoryPort
	cascadeDeleteRepo userrepo.CascadeDeleteRepositoryPort
}

// NewService creates a new organization service.
func NewService(
	l *slog.Logger,
	orgRepo repository.OrganizationRepositoryPort,
	cascadeDeleteRepo userrepo.CascadeDeleteRepositoryPort,
) *Service {
	return &Service{
		logger:            l.With(slog.String("name", "organization.service")),
		orgRepo:           orgRepo,
		cascadeDeleteRepo: cascadeDeleteRepo,
	}
}

// CreateOrganization creates a new organization with the specified user as admin.
func (s *Service) CreateOrganization(ctx context.Context, userID, name, slug string) (*CreateOrganizationResult, error) {
	// Validate userID is not empty.
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	// Validate name is not empty.
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Validate slug format.
	trimmedSlug := strings.TrimSpace(strings.ToLower(slug))
	if len(trimmedSlug) < SlugMinLength {
		return nil, fmt.Errorf("slug must be at least %d characters", SlugMinLength)
	}
	if len(trimmedSlug) > SlugMaxLength {
		return nil, fmt.Errorf("slug must be at most %d characters", SlugMaxLength)
	}
	if !SlugPattern.MatchString(trimmedSlug) {
		return nil, fmt.Errorf("slug must contain only lowercase letters, numbers, and hyphens (no leading/trailing hyphens)")
	}

	// Check if slug is already taken.
	existingOrg, err := s.orgRepo.GetBySlug(ctx, trimmedSlug)
	if err != nil {
		s.logger.Error("failed to check slug uniqueness",
			slog.String("slug", trimmedSlug),
			slog.Any("error", err),
		)
		return nil, err
	}
	if existingOrg != nil {
		return nil, fmt.Errorf("slug is already taken")
	}

	// Create organization with user as admin member.
	org := &domain.Organization{
		Name: trimmedName,
		Slug: trimmedSlug,
		Members: []*domain.OrganizationMember{
			{
				UserID: domain.ID(userID),
				Role:   domain.MemberRoleAdmin,
			},
		},
	}

	created, err := s.orgRepo.Create(ctx, org)
	if err != nil {
		s.logger.Error("failed to create organization",
			slog.String("userID", userID),
			slog.String("name", trimmedName),
			slog.String("slug", trimmedSlug),
			slog.Any("error", err),
		)
		return nil, err
	}

	s.logger.Info("organization created successfully",
		slog.String("organizationID", string(created.ID)),
		slog.String("userID", userID),
	)

	return &CreateOrganizationResult{Organization: created}, nil
}

// UpdateOrganization updates an organization's name and slug.
// Requires admin role.
func (s *Service) UpdateOrganization(ctx context.Context, userID, organizationID, name, slug string) (*UpdateOrganizationResult, error) {
	// Validate userID is not empty.
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	// Validate organizationID is not empty.
	if organizationID == "" {
		return nil, fmt.Errorf("organizationID is required")
	}

	// Check user's role in organization using orgRepo.GetMemberRole.
	role, err := s.orgRepo.GetMemberRole(ctx, organizationID, userID)
	if err != nil {
		s.logger.Error("failed to get member role",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}

	if role == "" {
		return nil, fmt.Errorf("user is not a member of this organization")
	}

	if role != domain.MemberRoleAdmin {
		return nil, fmt.Errorf("admin role required")
	}

	// Validate name is not empty.
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Validate slug format.
	trimmedSlug := strings.TrimSpace(strings.ToLower(slug))
	if len(trimmedSlug) < SlugMinLength {
		return nil, fmt.Errorf("slug must be at least %d characters", SlugMinLength)
	}
	if len(trimmedSlug) > SlugMaxLength {
		return nil, fmt.Errorf("slug must be at most %d characters", SlugMaxLength)
	}
	if !SlugPattern.MatchString(trimmedSlug) {
		return nil, fmt.Errorf("slug must contain only lowercase letters, numbers, and hyphens (no leading/trailing hyphens)")
	}

	// Check if slug is already taken by another organization.
	existingOrg, err := s.orgRepo.GetBySlug(ctx, trimmedSlug)
	if err != nil {
		s.logger.Error("failed to check slug uniqueness",
			slog.String("slug", trimmedSlug),
			slog.Any("error", err),
		)
		return nil, err
	}
	if existingOrg != nil && string(existingOrg.ID) != organizationID {
		return nil, fmt.Errorf("slug is already taken")
	}

	// Call orgRepo.Update to update organization.
	updated, err := s.orgRepo.Update(ctx, organizationID, trimmedName, trimmedSlug)
	if err != nil {
		s.logger.Error("failed to update organization",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return nil, err
	}

	s.logger.Info("organization updated successfully",
		slog.String("organizationID", organizationID),
	)

	return &UpdateOrganizationResult{Organization: updated}, nil
}

// GetOrganizationMembers retrieves members with their details.
// Requires membership in the organization.
func (s *Service) GetOrganizationMembers(ctx context.Context, userID, organizationID string) (*GetOrganizationMembersResult, error) {
	// Validate userID is not empty.
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	// Validate organizationID is not empty.
	if organizationID == "" {
		return nil, fmt.Errorf("organizationID is required")
	}

	// Check user's role in organization using orgRepo.GetMemberRole.
	role, err := s.orgRepo.GetMemberRole(ctx, organizationID, userID)
	if err != nil {
		s.logger.Error("failed to get member role",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}

	if role == "" {
		return nil, fmt.Errorf("user is not a member of this organization")
	}

	// Call orgRepo.GetMembersWithDetails.
	members, err := s.orgRepo.GetMembersWithDetails(ctx, organizationID)
	if err != nil {
		s.logger.Error("failed to get members with details",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return nil, err
	}

	s.logger.Info("retrieved organization members successfully",
		slog.String("organizationID", organizationID),
		slog.Int("memberCount", len(members)),
	)

	return &GetOrganizationMembersResult{Members: members}, nil
}

// UpdateMemberRole changes a member's role.
// Requires admin role. Cannot demote the last admin.
func (s *Service) UpdateMemberRole(ctx context.Context, userID, organizationID, targetUserID, newRole string) error {
	// Validate inputs
	if userID == "" {
		return fmt.Errorf("userID is required")
	}
	if organizationID == "" {
		return fmt.Errorf("organizationID is required")
	}
	if targetUserID == "" {
		return fmt.Errorf("targetUserID is required")
	}

	// Validate newRole is either "admin" or "member".
	if newRole != string(domain.MemberRoleAdmin) && newRole != string(domain.MemberRoleMember) {
		return fmt.Errorf("role must be 'admin' or 'member'")
	}

	// Check requesting user's role in organization.
	role, err := s.orgRepo.GetMemberRole(ctx, organizationID, userID)
	if err != nil {
		s.logger.Error("failed to get requesting user role",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return err
	}

	if role != domain.MemberRoleAdmin {
		return fmt.Errorf("admin role required")
	}

	// Get target user's current role.
	targetRole, err := s.orgRepo.GetMemberRole(ctx, organizationID, targetUserID)
	if err != nil {
		s.logger.Error("failed to get target user role",
			slog.String("organizationID", organizationID),
			slog.String("targetUserID", targetUserID),
			slog.Any("error", err),
		)
		return err
	}

	if targetRole == "" {
		return fmt.Errorf("target user is not a member")
	}

	// If demoting from admin to member, check if this is the last admin.
	if targetRole == domain.MemberRoleAdmin && newRole == string(domain.MemberRoleMember) {
		count, err := s.orgRepo.CountAdmins(ctx, organizationID)
		if err != nil {
			s.logger.Error("failed to count admins",
				slog.String("organizationID", organizationID),
				slog.Any("error", err),
			)
			return err
		}
		if count == 1 {
			return fmt.Errorf("cannot demote the last admin")
		}
	}

	// Call orgRepo.UpdateMemberRole.
	err = s.orgRepo.UpdateMemberRole(ctx, organizationID, targetUserID, domain.MemberRole(newRole))
	if err != nil {
		s.logger.Error("failed to update member role",
			slog.String("organizationID", organizationID),
			slog.String("targetUserID", targetUserID),
			slog.String("newRole", newRole),
			slog.Any("error", err),
		)
		return err
	}

	s.logger.Info("member role updated successfully",
		slog.String("organizationID", organizationID),
		slog.String("targetUserID", targetUserID),
		slog.String("newRole", newRole),
	)

	return nil
}

// RemoveMember removes a member from the organization.
// Requires admin role. Cannot remove the last admin.
func (s *Service) RemoveMember(ctx context.Context, userID, organizationID, targetUserID string) error {
	// Validate inputs
	if userID == "" {
		return fmt.Errorf("userID is required")
	}
	if organizationID == "" {
		return fmt.Errorf("organizationID is required")
	}
	if targetUserID == "" {
		return fmt.Errorf("targetUserID is required")
	}

	// Check requesting user's role in organization.
	role, err := s.orgRepo.GetMemberRole(ctx, organizationID, userID)
	if err != nil {
		s.logger.Error("failed to get requesting user role",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return err
	}

	if role != domain.MemberRoleAdmin {
		return fmt.Errorf("admin role required")
	}

	// Get target user's current role.
	targetRole, err := s.orgRepo.GetMemberRole(ctx, organizationID, targetUserID)
	if err != nil {
		s.logger.Error("failed to get target user role",
			slog.String("organizationID", organizationID),
			slog.String("targetUserID", targetUserID),
			slog.Any("error", err),
		)
		return err
	}

	if targetRole == "" {
		return fmt.Errorf("target user is not a member")
	}

	// If target user is admin, check if this is the last admin.
	if targetRole == domain.MemberRoleAdmin {
		count, err := s.orgRepo.CountAdmins(ctx, organizationID)
		if err != nil {
			s.logger.Error("failed to count admins",
				slog.String("organizationID", organizationID),
				slog.Any("error", err),
			)
			return err
		}
		if count == 1 {
			return fmt.Errorf("cannot remove the last admin")
		}
	}

	// Call orgRepo.RemoveMember.
	err = s.orgRepo.RemoveMember(ctx, organizationID, targetUserID)
	if err != nil {
		s.logger.Error("failed to remove member",
			slog.String("organizationID", organizationID),
			slog.String("targetUserID", targetUserID),
			slog.Any("error", err),
		)
		return err
	}

	s.logger.Info("member removed successfully",
		slog.String("organizationID", organizationID),
		slog.String("targetUserID", targetUserID),
	)

	return nil
}

// LeaveOrganization removes the current user from the organization.
// If this is the user's last organization, cascade deletes all data.
func (s *Service) LeaveOrganization(ctx context.Context, userID, organizationID string) (*LeaveOrganizationResult, error) {
	// Validate userID is not empty.
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	// Validate organizationID is not empty.
	if organizationID == "" {
		return nil, fmt.Errorf("organizationID is required")
	}

	// Get user's role in organization.
	role, err := s.orgRepo.GetMemberRole(ctx, organizationID, userID)
	if err != nil {
		s.logger.Error("failed to get user role",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}

	if role == "" {
		return nil, fmt.Errorf("user is not a member of this organization")
	}

	// Get organization to check member count.
	org, err := s.orgRepo.GetByID(ctx, organizationID)
	if err != nil {
		s.logger.Error("failed to get organization",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return nil, err
	}

	if org == nil {
		return nil, fmt.Errorf("organization not found")
	}

	memberCount := len(org.Members)

	// If user is admin, check if they are the sole admin with other members.
	if role == domain.MemberRoleAdmin {
		adminCount, err := s.orgRepo.CountAdmins(ctx, organizationID)
		if err != nil {
			s.logger.Error("failed to count admins",
				slog.String("organizationID", organizationID),
				slog.Any("error", err),
			)
			return nil, err
		}
		if adminCount == 1 && memberCount > 1 {
			return nil, fmt.Errorf("cannot leave as the sole admin with other members")
		}
	}

	// Check if this is user's last organization.
	orgCount, err := s.orgRepo.GetUserOrganizationCount(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user organization count",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}

	isLastOrganization := orgCount == 1

	// If organization has only this user as member (sole member), delete the organization.
	if memberCount == 1 {
		// Cascade delete events
		err = s.cascadeDeleteRepo.DeleteEventsByOrganization(ctx, organizationID)
		if err != nil {
			s.logger.Error("failed to delete events",
				slog.String("organizationID", organizationID),
				slog.Any("error", err),
			)
			return nil, err
		}

		// Cascade delete projects
		err = s.cascadeDeleteRepo.DeleteProjectsByOrganization(ctx, organizationID)
		if err != nil {
			s.logger.Error("failed to delete projects",
				slog.String("organizationID", organizationID),
				slog.Any("error", err),
			)
			return nil, err
		}

		// Delete organization
		err = s.orgRepo.DeleteOrganization(ctx, organizationID)
		if err != nil {
			s.logger.Error("failed to delete organization",
				slog.String("organizationID", organizationID),
				slog.Any("error", err),
			)
			return nil, err
		}

		s.logger.Info("organization deleted as sole member left",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
		)
	} else {
		// Shared organization - just remove member
		err = s.orgRepo.RemoveMember(ctx, organizationID, userID)
		if err != nil {
			s.logger.Error("failed to remove member",
				slog.String("organizationID", organizationID),
				slog.String("userID", userID),
				slog.Any("error", err),
			)
			return nil, err
		}

		s.logger.Info("member removed from shared organization",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
		)
	}

	s.logger.Info("user left organization successfully",
		slog.String("organizationID", organizationID),
		slog.String("userID", userID),
		slog.Bool("isLastOrganization", isLastOrganization),
	)

	return &LeaveOrganizationResult{
		Success:            true,
		IsLastOrganization: isLastOrganization,
	}, nil
}
