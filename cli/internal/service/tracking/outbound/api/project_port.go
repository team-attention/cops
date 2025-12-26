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
}

// RegisterProjectResult contains the result of project registration.
type RegisterProjectResult struct {
	ProjectID domain.ID
	IsNew     bool
}

// ProjectPort defines the interface for project API operations.
type ProjectPort interface {
	// RegisterProject registers a project or returns existing project ID if already registered.
	// Performs duplicate detection using remote URLs and optional existing project ID.
	RegisterProject(ctx context.Context, params RegisterProjectParams) (*RegisterProjectResult, error)
}
