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

// OrganizationRepositoryPort defines interface for organization queries.
type OrganizationRepositoryPort interface {
	// GetUserOrganizations retrieves all organizations a user belongs to with their roles.
	// Queries organizations collection filtering by embedded members.userId.
	// Returns empty slice if user has no organizations.
	// Returns nil, error if database error occurs.
	GetUserOrganizations(ctx context.Context, userID string) ([]*UserOrganization, error)
}
