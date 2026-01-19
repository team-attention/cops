package api

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// RegisterProjectParams contains parameters for registering a project with the API server.
type RegisterProjectParams struct {
	// ConfiguredRemoteURL is from git config (git config --get remote.origin.url)
	ConfiguredRemoteURL string

	// ActualRemoteURL is from git ls-remote (git ls-remote --get-url origin)
	// This may differ from configured URL if the GitHub repo was renamed
	ActualRemoteURL string

	// ExistingProjectID is optional - from local config if available
	// Used as fallback for finding existing projects
	ExistingProjectID string

	// Name is the human-readable project name
	Name string

	// IsGitProject indicates whether this is a git repository
	IsGitProject bool

	// OrganizationID is the organization this project belongs to
	OrganizationID string
}

// RegisterProjectResult contains the result of project registration.
type RegisterProjectResult struct {
	ProjectID    domain.ID
	IsNew        bool
	Name         string
	IsGitProject bool
}

// GetProjectByIDAndOrgResult contains the result of GetProjectByIDAndOrg API call.
type GetProjectByIDAndOrgResult struct {
	Found          bool
	ProjectID      string
	Name           string
	OrganizationID string
}

// ProjectPort defines the interface for project API operations.
type ProjectPort interface {
	// RegisterProject registers a project or returns existing project ID if already registered.
	// Performs duplicate detection using remote URLs and optional existing project ID.
	// Requires valid access token for authentication.
	RegisterProject(ctx context.Context, accessToken string, params RegisterProjectParams) (*RegisterProjectResult, error)

	// GetProjectByIDAndOrg checks if a project exists on the server by ID within a specific organization.
	// Requires valid access token for authentication.
	GetProjectByIDAndOrg(ctx context.Context, accessToken string, projectID string, organizationID string) (*GetProjectByIDAndOrgResult, error)
}
