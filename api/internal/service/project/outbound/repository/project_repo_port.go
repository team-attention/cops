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

// GetByIDAndOrgResult contains the result of GetByIDAndOrg operation.
type GetByIDAndOrgResult struct {
	Found          bool
	ProjectID      string
	Name           string
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

	// GetByIDAndOrg retrieves a project by its ID within a specific organization.
	// Returns GetByIDAndOrgResult with Found=false if project doesn't exist or belongs to different org.
	GetByIDAndOrg(ctx context.Context, projectID string, organizationID string) (*GetByIDAndOrgResult, error)
}
