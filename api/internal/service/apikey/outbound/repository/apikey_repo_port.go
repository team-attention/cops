package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// APIKeyRepositoryPort defines interface for API key persistence.
type APIKeyRepositoryPort interface {
	// Create stores a new API key and returns the created key with ID.
	Create(ctx context.Context, apiKey *domain.APIKey) (*domain.APIKey, error)

	// GetByHash retrieves an API key by its hash value.
	// Returns nil, nil if not found.
	GetByHash(ctx context.Context, keyHash string) (*domain.APIKey, error)

	// GetByID retrieves an API key by its ID.
	// Returns nil, nil if not found.
	GetByID(ctx context.Context, keyID string) (*domain.APIKey, error)

	// ListByUser retrieves all API keys for a user.
	// If includeRevoked is false, only returns active keys.
	ListByUser(ctx context.Context, userID string, includeRevoked bool) ([]*domain.APIKey, error)

	// Revoke marks an API key as revoked by setting revokedAt.
	// Returns error if key not found.
	Revoke(ctx context.Context, keyID string) error

	// UpdateLastUsedAt updates the lastUsedAt timestamp.
	UpdateLastUsedAt(ctx context.Context, keyID string) error
}
