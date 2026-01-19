package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// UserRepositoryPort defines interface for user data persistence.
type UserRepositoryPort interface {
	// Create creates a new user with embedded accounts.
	Create(ctx context.Context, user *domain.User) (*domain.User, error)

	// GetByID retrieves user by ID.
	GetByID(ctx context.Context, userID string) (*domain.User, error)

	// FindByAccountProvider finds user by OAuth provider account.
	// Searches within the embedded accounts array.
	FindByAccountProvider(ctx context.Context, provider domain.AccountProvider, providerID string) (*domain.User, error)
}
