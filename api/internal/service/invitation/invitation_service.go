package invitation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/team-attention/cops/api/internal/platform/outbound/email"
	"github.com/team-attention/cops/api/internal/platform/setup/config"
	"github.com/team-attention/cops/api/internal/service/invitation/outbound/repository"
	orgrepo "github.com/team-attention/cops/api/internal/service/organization/outbound/repository"
	userrepo "github.com/team-attention/cops/api/internal/service/user/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
)

// CreateInvitationResult contains the created invitation.
type CreateInvitationResult struct {
	Invitation *domain.Invitation
}

// ListInvitationsResult contains pending invitations.
type ListInvitationsResult struct {
	Invitations []*domain.Invitation
}

// GetInvitationResult contains invitation with organization details.
type GetInvitationResult struct {
	Invitation       *domain.Invitation
	OrganizationName string
}

// AcceptInvitationResult contains the result of accepting.
type AcceptInvitationResult struct {
	Success        bool
	OrganizationID string
}

// Service implements invitation business logic.
type Service struct {
	logger       *slog.Logger
	inviteRepo   repository.InvitationRepositoryPort
	orgRepo      orgrepo.OrganizationRepositoryPort
	userRepo     userrepo.UserRepositoryPort
	emailService email.EmailServicePort
	webBaseURL   string
}

// NewService creates a new invitation service.
func NewService(
	l *slog.Logger,
	inviteRepo repository.InvitationRepositoryPort,
	orgRepo orgrepo.OrganizationRepositoryPort,
	userRepo userrepo.UserRepositoryPort,
	emailService email.EmailServicePort,
	cfg *config.Config,
) *Service {
	return &Service{
		logger:       l.With(slog.String("name", "invitation.service")),
		inviteRepo:   inviteRepo,
		orgRepo:      orgRepo,
		userRepo:     userRepo,
		emailService: emailService,
		webBaseURL:   cfg.DeviceCode.WebBaseURL,
	}
}

// CreateInvitation creates a new invitation and sends email.
func (s *Service) CreateInvitation(ctx context.Context, userID, organizationID, inviteeEmail string) (*CreateInvitationResult, error) {
	// Validate inputs
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}
	if organizationID == "" {
		return nil, fmt.Errorf("organizationID is required")
	}
	if inviteeEmail == "" {
		return nil, fmt.Errorf("email is required")
	}

	// Normalize email
	normalizedEmail := strings.ToLower(strings.TrimSpace(inviteeEmail))

	// Check user is admin in organization
	role, err := s.orgRepo.GetMemberRole(ctx, organizationID, userID)
	if err != nil {
		s.logger.Error("failed to get member role",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}
	if role != domain.MemberRoleAdmin {
		return nil, fmt.Errorf("admin role required")
	}

	// Check if invitee is already a member
	members, err := s.orgRepo.GetMembersWithDetails(ctx, organizationID)
	if err != nil {
		s.logger.Error("failed to get members",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return nil, err
	}
	for _, m := range members {
		if strings.EqualFold(m.Email, normalizedEmail) {
			return nil, fmt.Errorf("user is already a member")
		}
	}

	// Check if invitee is inviting themselves
	inviter, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get inviter",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}
	if inviter != nil && strings.EqualFold(inviter.Email, normalizedEmail) {
		return nil, fmt.Errorf("cannot invite yourself")
	}

	// Check if pending invitation already exists
	existing, err := s.inviteRepo.GetByEmailAndOrg(ctx, normalizedEmail, organizationID)
	if err != nil {
		s.logger.Error("failed to check existing invitation",
			slog.String("email", normalizedEmail),
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("invitation already sent")
	}

	// Generate secure random token
	token, err := generateToken()
	if err != nil {
		s.logger.Error("failed to generate token",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to generate invitation token")
	}

	// Create invitation
	invitation := &domain.Invitation{
		OrganizationID: domain.ID(organizationID),
		Email:          normalizedEmail,
		Token:          token,
		Status:         domain.InvitationStatusPending,
		InvitedByID:    domain.ID(userID),
		CreatedAt:      time.Now(),
	}

	created, err := s.inviteRepo.Create(ctx, invitation)
	if err != nil {
		s.logger.Error("failed to create invitation",
			slog.String("email", normalizedEmail),
			slog.Any("error", err),
		)
		return nil, err
	}

	// Get organization name for email
	org, err := s.orgRepo.GetByID(ctx, organizationID)
	if err != nil {
		s.logger.Error("failed to get organization",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		// Don't fail if we can't get org name, just use a generic name
	}

	orgName := "an organization"
	if org != nil {
		orgName = org.Name
	}

	// Send invitation email
	inviteURL := fmt.Sprintf("%s/invite/%s", s.webBaseURL, token)
	emailParams := email.SendEmailParams{
		To:      normalizedEmail,
		Subject: fmt.Sprintf("You've been invited to join %s on C-Ops", orgName),
		TextBody: fmt.Sprintf(`You've been invited to join %s on C-Ops.

Click the link below to accept the invitation:
%s

If you didn't expect this invitation, you can ignore this email.`, orgName, inviteURL),
		HTMLBody: fmt.Sprintf(`<p>You've been invited to join <strong>%s</strong> on C-Ops.</p>
<p><a href="%s">Click here to accept the invitation</a></p>
<p>Or copy and paste this link: %s</p>
<p>If you didn't expect this invitation, you can ignore this email.</p>`, orgName, inviteURL, inviteURL),
	}

	if err := s.emailService.Send(ctx, emailParams); err != nil {
		s.logger.Warn("failed to send invitation email",
			slog.String("email", normalizedEmail),
			slog.Any("error", err),
		)
		// Don't fail the invitation creation if email fails
	}

	s.logger.Info("invitation created successfully",
		slog.String("invitationID", string(created.ID)),
		slog.String("email", normalizedEmail),
		slog.String("organizationID", organizationID),
	)

	return &CreateInvitationResult{Invitation: created}, nil
}

// ListInvitations retrieves pending invitations for an organization.
func (s *Service) ListInvitations(ctx context.Context, userID, organizationID string) (*ListInvitationsResult, error) {
	// Validate inputs
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}
	if organizationID == "" {
		return nil, fmt.Errorf("organizationID is required")
	}

	// Check user is admin in organization
	role, err := s.orgRepo.GetMemberRole(ctx, organizationID, userID)
	if err != nil {
		s.logger.Error("failed to get member role",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}
	if role != domain.MemberRoleAdmin {
		return nil, fmt.Errorf("admin role required")
	}

	// Get pending invitations
	invitations, err := s.inviteRepo.ListByOrganization(ctx, organizationID)
	if err != nil {
		s.logger.Error("failed to list invitations",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return &ListInvitationsResult{Invitations: invitations}, nil
}

// RevokeInvitation cancels a pending invitation.
func (s *Service) RevokeInvitation(ctx context.Context, userID, invitationID string) error {
	// Validate inputs
	if userID == "" {
		return fmt.Errorf("userID is required")
	}
	if invitationID == "" {
		return fmt.Errorf("invitationID is required")
	}

	// Get invitation
	invitation, err := s.inviteRepo.GetByID(ctx, invitationID)
	if err != nil {
		s.logger.Error("failed to get invitation",
			slog.String("invitationID", invitationID),
			slog.Any("error", err),
		)
		return err
	}
	if invitation == nil {
		return fmt.Errorf("invitation not found")
	}

	// Check user is admin in invitation's organization
	role, err := s.orgRepo.GetMemberRole(ctx, string(invitation.OrganizationID), userID)
	if err != nil {
		s.logger.Error("failed to get member role",
			slog.String("organizationID", string(invitation.OrganizationID)),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return err
	}
	if role != domain.MemberRoleAdmin {
		return fmt.Errorf("admin role required")
	}

	// Check invitation is still pending
	if invitation.Status != domain.InvitationStatusPending {
		return fmt.Errorf("invitation already processed")
	}

	// Delete invitation
	if err := s.inviteRepo.Delete(ctx, invitationID); err != nil {
		s.logger.Error("failed to delete invitation",
			slog.String("invitationID", invitationID),
			slog.Any("error", err),
		)
		return err
	}

	s.logger.Info("invitation revoked successfully",
		slog.String("invitationID", invitationID),
	)

	return nil
}

// GetInvitationByToken retrieves invitation details for the acceptance page.
func (s *Service) GetInvitationByToken(ctx context.Context, token string) (*GetInvitationResult, error) {
	// Validate token
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}

	// Get invitation
	invitation, err := s.inviteRepo.GetByToken(ctx, token)
	if err != nil {
		s.logger.Error("failed to get invitation by token",
			slog.Any("error", err),
		)
		return nil, err
	}
	if invitation == nil {
		return nil, fmt.Errorf("invitation not found")
	}

	// Check invitation is still pending
	if invitation.Status != domain.InvitationStatusPending {
		return nil, fmt.Errorf("invitation no longer valid")
	}

	// Get organization name
	org, err := s.orgRepo.GetByID(ctx, string(invitation.OrganizationID))
	if err != nil {
		s.logger.Error("failed to get organization",
			slog.String("organizationID", string(invitation.OrganizationID)),
			slog.Any("error", err),
		)
		return nil, err
	}

	orgName := ""
	if org != nil {
		orgName = org.Name
	}

	return &GetInvitationResult{
		Invitation:       invitation,
		OrganizationName: orgName,
	}, nil
}

// AcceptInvitation adds user to organization if email matches.
func (s *Service) AcceptInvitation(ctx context.Context, userID, userEmail, token string) (*AcceptInvitationResult, error) {
	// Validate inputs
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}
	if userEmail == "" {
		return nil, fmt.Errorf("userEmail is required")
	}
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}

	// Get invitation by token
	invitation, err := s.inviteRepo.GetByToken(ctx, token)
	if err != nil {
		s.logger.Error("failed to get invitation by token",
			slog.Any("error", err),
		)
		return nil, err
	}
	if invitation == nil {
		return nil, fmt.Errorf("invitation not found")
	}

	// Check invitation is still pending
	if invitation.Status != domain.InvitationStatusPending {
		return nil, fmt.Errorf("invitation no longer valid")
	}

	// Verify user email matches invitation email (case-insensitive)
	normalizedUserEmail := strings.ToLower(strings.TrimSpace(userEmail))
	normalizedInviteEmail := strings.ToLower(strings.TrimSpace(invitation.Email))
	if normalizedUserEmail != normalizedInviteEmail {
		return nil, fmt.Errorf("email mismatch: invitation was sent to %s", invitation.Email)
	}

	organizationID := string(invitation.OrganizationID)

	// Check if user is already a member
	role, err := s.orgRepo.GetMemberRole(ctx, organizationID, userID)
	if err != nil {
		s.logger.Error("failed to check membership",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}

	if role != "" {
		// Already a member, just mark invitation as accepted
		if err := s.inviteRepo.UpdateStatus(ctx, string(invitation.ID), domain.InvitationStatusAccepted); err != nil {
			s.logger.Error("failed to update invitation status",
				slog.String("invitationID", string(invitation.ID)),
				slog.Any("error", err),
			)
			return nil, err
		}

		return &AcceptInvitationResult{
			Success:        true,
			OrganizationID: organizationID,
		}, nil
	}

	// Add user to organization as member
	if err := s.orgRepo.AddMember(ctx, organizationID, userID, domain.MemberRoleMember); err != nil {
		s.logger.Error("failed to add member",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}

	// Update invitation status to accepted
	if err := s.inviteRepo.UpdateStatus(ctx, string(invitation.ID), domain.InvitationStatusAccepted); err != nil {
		s.logger.Error("failed to update invitation status",
			slog.String("invitationID", string(invitation.ID)),
			slog.Any("error", err),
		)
		return nil, err
	}

	s.logger.Info("invitation accepted successfully",
		slog.String("invitationID", string(invitation.ID)),
		slog.String("userID", userID),
		slog.String("organizationID", organizationID),
	)

	return &AcceptInvitationResult{
		Success:        true,
		OrganizationID: organizationID,
	}, nil
}

// generateToken generates a secure random token.
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
