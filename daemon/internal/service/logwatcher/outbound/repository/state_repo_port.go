package repository

import (
	"context"

	"github.com/team-attention/cops/daemon/internal/platform/domain"
)

// StateRepositoryPort defines persistence operations for daemon state.
type StateRepositoryPort interface {
	LoadFilePositions(ctx context.Context) (map[string]*domain.FilePosition, error)
	SaveFilePosition(ctx context.Context, pos *domain.FilePosition) error
	DeleteFilePosition(ctx context.Context, path string) error
	Close() error
}
