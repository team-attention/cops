package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// MemberWithDetails represents a member with full user information.
type MemberWithDetails struct {
	UserID    string
	Email     string
	Name      string
	AvatarURL string
	Role      domain.MemberRole
}

// OrganizationRepositoryPort defines interface for organization management operations.
type OrganizationRepositoryPort interface {
	// Create persists a new organization.
	// Returns the created organization with generated ID.
	// Returns nil, error if database error occurs.
	Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error)

	// GetByID retrieves an organization by its ID.
	// Returns nil, nil if organization not found.
	// Returns nil, error if database error occurs.
	GetByID(ctx context.Context, organizationID string) (*domain.Organization, error)

	// GetBySlug retrieves an organization by its slug.
	// Returns nil, nil if organization not found.
	// Returns nil, error if database error occurs.
	GetBySlug(ctx context.Context, slug string) (*domain.Organization, error)

	// Update updates an organization's name and slug.
	// Returns the updated organization.
	// Returns error if database error occurs.
	Update(ctx context.Context, organizationID, name, slug string) (*domain.Organization, error)

	// GetMembersWithDetails retrieves all members with their user details.
	// Performs join with users collection to get email, name, avatar.
	// Returns empty slice if no members.
	// Returns nil, error if database error occurs.
	GetMembersWithDetails(ctx context.Context, organizationID string) ([]*MemberWithDetails, error)

	// UpdateMemberRole updates a member's role in the organization.
	// Returns error if database error occurs.
	UpdateMemberRole(ctx context.Context, organizationID, userID string, role domain.MemberRole) error

	// RemoveMember removes a member from the organization.
	// Returns error if database error occurs.
	RemoveMember(ctx context.Context, organizationID, userID string) error

	// CountAdmins returns the number of admin members in an organization.
	// Returns count, nil on success.
	// Returns 0, error if database error occurs.
	CountAdmins(ctx context.Context, organizationID string) (int, error)

	// GetUserOrganizationCount returns the number of organizations a user belongs to.
	// Returns count, nil on success.
	// Returns 0, error if database error occurs.
	GetUserOrganizationCount(ctx context.Context, userID string) (int, error)

	// GetMemberRole retrieves a user's role in an organization.
	// Returns empty string, nil if user is not a member.
	// Returns role, nil on success.
	// Returns "", error if database error occurs.
	GetMemberRole(ctx context.Context, organizationID, userID string) (domain.MemberRole, error)

	// DeleteOrganization permanently deletes an organization.
	// Returns error if database error occurs.
	DeleteOrganization(ctx context.Context, organizationID string) error
}
