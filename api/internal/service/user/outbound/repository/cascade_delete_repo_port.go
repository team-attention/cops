package repository

import "context"

// CascadeDeleteRepositoryPort defines interface for cascade deletion operations.
// Used during account deletion to clean up related data.
type CascadeDeleteRepositoryPort interface {
	// DeleteProjectsByOrganization permanently deletes all projects for an organization.
	// Returns nil if successful or if no projects existed.
	// Returns error if database error occurs.
	DeleteProjectsByOrganization(ctx context.Context, organizationID string) error

	// DeleteEventsByOrganization permanently deletes all events for projects in an organization.
	// First queries projects to get project IDs, then deletes events matching those project IDs.
	// Returns nil if successful or if no events existed.
	// Returns error if database error occurs.
	DeleteEventsByOrganization(ctx context.Context, organizationID string) error
}
