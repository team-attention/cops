package mock

import (
	"context"

	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
)

// ProjectRepository implements repository.ProjectRepositoryPort for testing.
type ProjectRepository struct {
	// GetByIDFunc is the behavior to execute when GetByID is called.
	GetByIDFunc func(ctx context.Context, projectID string) (*domain.Project, error)
	// CallCount tracks the number of GetByID calls.
	CallCount int
	// ProjectIDs records all projectIDs queried.
	ProjectIDs []string
}

// GetByID implements repository.ProjectRepositoryPort.
func (m *ProjectRepository) GetByID(ctx context.Context, projectID string) (*domain.Project, error) {
	m.CallCount++

	m.ProjectIDs = append(m.ProjectIDs, projectID)

	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, projectID)
	}

	return nil, nil
}

// Compile-time interface verification.
var _ repository.ProjectRepositoryPort = (*ProjectRepository)(nil)
