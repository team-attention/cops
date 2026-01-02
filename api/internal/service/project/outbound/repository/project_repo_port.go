package repository

import "context"

// FindOrCreateParams contains parameters for FindOrCreate operation.
type FindOrCreateParams struct {
	ConfiguredURL  string
	ActualURL      string
	ExistingID     string
	Name           string
	IsGitProject   bool
	OrganizationID string
}

// FindOrCreateResult contains the result of find-or-create operation.
type FindOrCreateResult struct {
	ProjectID      string
	IsNew          bool
	Name           string
	IsGitProject   bool
	OrganizationID string
}

// ProjectRepositoryPort defines the interface for project data persistence.
type ProjectRepositoryPort interface {
	// FindOrCreate finds existing project or creates new one.
	// Search order:
	// 1. By remote URL (either configured or actual)
	// 2. By existing project ID (if provided)
	// 3. Create new if not found
	FindOrCreate(ctx context.Context, params FindOrCreateParams) (*FindOrCreateResult, error)
}
