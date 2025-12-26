package repository

import "context"

// FindOrCreateResult contains the result of find-or-create operation.
type FindOrCreateResult struct {
	ProjectID string
	IsNew     bool
}

// ProjectRepositoryPort defines the interface for project data persistence.
type ProjectRepositoryPort interface {
	// FindOrCreate finds existing project or creates new one.
	// Search order:
	// 1. By remote URL (either configured or actual)
	// 2. By existing project ID (if provided)
	// 3. Create new if not found
	FindOrCreate(ctx context.Context, configuredURL, actualURL, existingID string) (*FindOrCreateResult, error)
}
