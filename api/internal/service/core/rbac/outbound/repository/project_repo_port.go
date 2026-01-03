package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// ProjectRepositoryPort defines interface for project data access needed by RBAC.
type ProjectRepositoryPort interface {
	// GetByID retrieves a project by its ID.
	// Returns nil, nil if project not found.
	// Returns nil, error if database error occurs.
	GetByID(ctx context.Context, projectID string) (*domain.Project, error)
}
