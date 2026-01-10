package apikey

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/team-attention/cops/api/internal/platform/util/apikeyutil"
	"github.com/team-attention/cops/api/internal/service/apikey/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
)

// IssueAPIKeyParams contains parameters for issuing a new API key.
type IssueAPIKeyParams struct {
	UserID        string
	Name          string
	ExpiresInDays int32
}

// IssueAPIKeyResult contains the result of issuing a new API key.
type IssueAPIKeyResult struct {
	APIKey  string // Plain-text key (only returned once)
	KeyInfo *domain.APIKey
}

// ListAPIKeysParams contains parameters for listing API keys.
type ListAPIKeysParams struct {
	UserID         string
	IncludeRevoked bool
}

// RevokeAPIKeyParams contains parameters for revoking an API key.
type RevokeAPIKeyParams struct {
	UserID string
	KeyID  string
}

// ValidateAPIKeyResult contains the result of validating an API key.
type ValidateAPIKeyResult struct {
	Valid        bool
	UserID       string
	ErrorMessage string
}

// Service implements API key business logic.
type Service struct {
	logger *slog.Logger
	repo   repository.APIKeyRepositoryPort
}

// NewService creates a new API key service.
func NewService(l *slog.Logger, repo repository.APIKeyRepositoryPort) *Service {
	return &Service{
		logger: l.With(slog.String("name", "apikey.service")),
		repo:   repo,
	}
}

// IssueAPIKey creates a new API key for a user.
func (s *Service) IssueAPIKey(ctx context.Context, params IssueAPIKeyParams) (*IssueAPIKeyResult, error) {
	// Generate new API key
	plainKey, err := apikeyutil.GenerateAPIKey()
	if err != nil {
		s.logger.Error("failed to generate API key",
			slog.String("userID", params.UserID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Hash key
	keyHash := apikeyutil.HashAPIKey(plainKey)

	// Extract prefix for identification
	keyPrefix := apikeyutil.ExtractPrefix(plainKey)

	// Calculate expiresAt from expiresInDays if provided
	var expiresAt *time.Time
	if params.ExpiresInDays > 0 {
		expiry := time.Now().Add(time.Duration(params.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &expiry
	}

	// Create API key domain object
	apiKey := &domain.APIKey{
		UserID:    domain.ID(params.UserID),
		Name:      params.Name,
		KeyPrefix: keyPrefix,
		KeyHash:   keyHash,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	// Store in repository
	createdKey, err := s.repo.Create(ctx, apiKey)
	if err != nil {
		s.logger.Error("failed to store API key",
			slog.String("userID", params.UserID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to store API key: %w", err)
	}

	s.logger.Info("API key issued successfully",
		slog.String("userID", params.UserID),
		slog.String("keyID", string(createdKey.ID)),
		slog.String("keyPrefix", keyPrefix),
	)

	return &IssueAPIKeyResult{
		APIKey:  plainKey,
		KeyInfo: createdKey,
	}, nil
}

// ListAPIKeys retrieves all API keys for a user.
func (s *Service) ListAPIKeys(ctx context.Context, params ListAPIKeysParams) ([]*domain.APIKey, error) {
	apiKeys, err := s.repo.ListByUser(ctx, params.UserID, params.IncludeRevoked)
	if err != nil {
		s.logger.Error("failed to list API keys",
			slog.String("userID", params.UserID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	return apiKeys, nil
}

// RevokeAPIKey revokes an API key.
func (s *Service) RevokeAPIKey(ctx context.Context, params RevokeAPIKeyParams) error {
	// Get key by ID to verify ownership
	key, err := s.repo.GetByID(ctx, params.KeyID)
	if err != nil {
		s.logger.Error("failed to get API key for revocation",
			slog.String("keyID", params.KeyID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to get API key: %w", err)
	}

	if key == nil {
		return fmt.Errorf("API key not found")
	}

	// Verify key belongs to user
	if string(key.UserID) != params.UserID {
		s.logger.Warn("unauthorized API key revocation attempt",
			slog.String("userID", params.UserID),
			slog.String("keyID", params.KeyID),
			slog.String("keyOwner", string(key.UserID)),
		)
		return fmt.Errorf("API key not found")
	}

	// Revoke key
	if err := s.repo.Revoke(ctx, params.KeyID); err != nil {
		s.logger.Error("failed to revoke API key",
			slog.String("keyID", params.KeyID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	s.logger.Info("API key revoked successfully",
		slog.String("userID", params.UserID),
		slog.String("keyID", params.KeyID),
	)

	return nil
}

// ValidateAPIKey validates an API key and returns the associated user ID.
func (s *Service) ValidateAPIKey(ctx context.Context, apiKey string) (*ValidateAPIKeyResult, error) {
	// Hash the provided key
	keyHash := apikeyutil.HashAPIKey(apiKey)

	// Lookup by hash in repository
	key, err := s.repo.GetByHash(ctx, keyHash)
	if err != nil {
		s.logger.Error("failed to validate API key",
			slog.Any("error", err),
		)
		return &ValidateAPIKeyResult{
			Valid:        false,
			ErrorMessage: "failed to validate API key",
		}, nil
	}

	if key == nil {
		return &ValidateAPIKeyResult{
			Valid:        false,
			ErrorMessage: "invalid API key",
		}, nil
	}

	// Verify key is not revoked
	if key.RevokedAt != nil {
		return &ValidateAPIKeyResult{
			Valid:        false,
			ErrorMessage: "API key has been revoked",
		}, nil
	}

	// Verify key is not expired
	if key.IsExpired() {
		return &ValidateAPIKeyResult{
			Valid:        false,
			ErrorMessage: "API key has expired",
		}, nil
	}

	// Update lastUsedAt timestamp (non-blocking, ignore errors)
	go func() {
		_ = s.repo.UpdateLastUsedAt(context.Background(), string(key.ID))
	}()

	return &ValidateAPIKeyResult{
		Valid:  true,
		UserID: string(key.UserID),
	}, nil
}
