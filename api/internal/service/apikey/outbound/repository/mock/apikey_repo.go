package mock

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
)

// APIKeyRepository is a mock implementation of APIKeyRepositoryPort.
type APIKeyRepository struct {
	CreateFunc           func(ctx context.Context, apiKey *domain.APIKey) (*domain.APIKey, error)
	GetByHashFunc        func(ctx context.Context, keyHash string) (*domain.APIKey, error)
	GetByIDFunc          func(ctx context.Context, keyID string) (*domain.APIKey, error)
	ListByUserFunc       func(ctx context.Context, userID string, includeRevoked bool) ([]*domain.APIKey, error)
	RevokeFunc           func(ctx context.Context, keyID string) error
	UpdateLastUsedAtFunc func(ctx context.Context, keyID string) error
}

func (m *APIKeyRepository) Create(ctx context.Context, apiKey *domain.APIKey) (*domain.APIKey, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, apiKey)
	}
	return nil, nil
}

func (m *APIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	if m.GetByHashFunc != nil {
		return m.GetByHashFunc(ctx, keyHash)
	}
	return nil, nil
}

func (m *APIKeyRepository) GetByID(ctx context.Context, keyID string) (*domain.APIKey, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, keyID)
	}
	return nil, nil
}

func (m *APIKeyRepository) ListByUser(ctx context.Context, userID string, includeRevoked bool) ([]*domain.APIKey, error) {
	if m.ListByUserFunc != nil {
		return m.ListByUserFunc(ctx, userID, includeRevoked)
	}
	return nil, nil
}

func (m *APIKeyRepository) Revoke(ctx context.Context, keyID string) error {
	if m.RevokeFunc != nil {
		return m.RevokeFunc(ctx, keyID)
	}
	return nil
}

func (m *APIKeyRepository) UpdateLastUsedAt(ctx context.Context, keyID string) error {
	if m.UpdateLastUsedAtFunc != nil {
		return m.UpdateLastUsedAtFunc(ctx, keyID)
	}
	return nil
}
