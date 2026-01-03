package repository

import "context"

// OrganizationMemberRepositoryPort defines interface for organization membership queries.
type OrganizationMemberRepositoryPort interface {
	// IsMember checks if a user is a member of an organization.
	// Returns true if membership exists (any role: admin or member).
	// Returns false if no membership found.
	// Returns false, error if database error occurs.
	IsMember(ctx context.Context, userID, organizationID string) (bool, error)
}
