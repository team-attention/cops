package api

import "context"

// TokenResult contains new tokens from refresh operation.
type TokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
}

// AuthAPIPort defines the interface for auth API operations.
type AuthAPIPort interface {
	// RefreshToken exchanges a refresh token for a new token pair.
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResult, error)
}
