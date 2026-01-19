package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// UserRepositoryPort defines interface for user data retrieval.
type UserRepositoryPort interface {
	// GetByID retrieves a user by their ID.
	// Returns nil, nil if user not found.
	// Returns nil, error if database error occurs.
	GetByID(ctx context.Context, userID string) (*domain.User, error)

	// Delete permanently removes a user by their ID.
	// Returns nil if user was deleted successfully.
	// Returns error if database error occurs.
	Delete(ctx context.Context, userID string) error
}
