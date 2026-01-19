package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// UserOrganization represents a user's membership in an organization.
// Contains organization data plus the user's specific role.
type UserOrganization struct {
	Organization *domain.Organization
	Role         domain.MemberRole
}

// OrganizationWithMemberCount represents an organization with its member count.
// Used to determine if cascade deletion is needed.
type OrganizationWithMemberCount struct {
	Organization *domain.Organization
	MemberCount  int
}

// OrganizationRepositoryPort defines interface for organization queries.
type OrganizationRepositoryPort interface {
	// Create creates a new organization.
	// Participates in transaction if ctx contains mongo.SessionContext.
	// Returns created organization with generated ID.
	Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error)

	// GetUserOrganizations retrieves all organizations a user belongs to with their roles.
	// Queries organizations collection filtering by embedded members.userId.
	// Returns empty slice if user has no organizations.
	// Returns nil, error if database error occurs.
	GetUserOrganizations(ctx context.Context, userID string) ([]*UserOrganization, error)

	// GetUserOrganizationsWithMemberCount retrieves all organizations a user belongs to with member counts.
	// Used to determine which organizations need cascade deletion (sole member) vs membership removal.
	// Returns empty slice if user has no organizations.
	// Returns nil, error if database error occurs.
	GetUserOrganizationsWithMemberCount(ctx context.Context, userID string) ([]*OrganizationWithMemberCount, error)

	// RemoveUserFromOrganization removes a user from an organization's members array.
	// Returns nil if successful or if user was not a member.
	// Returns error if database error occurs.
	RemoveUserFromOrganization(ctx context.Context, organizationID, userID string) error

	// DeleteOrganization permanently deletes an organization by ID.
	// Returns nil if successful or if organization did not exist.
	// Returns error if database error occurs.
	DeleteOrganization(ctx context.Context, organizationID string) error
}
