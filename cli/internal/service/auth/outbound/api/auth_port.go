package api

import "context"

// DeviceCodeResult contains device code flow data.
type DeviceCodeResult struct {
	DeviceCode      string
	UserCode        string
	VerificationURL string
	ExpiresIn       int
	Interval        int
}

// PollResult contains poll result.
type PollResult struct {
	Pending      bool
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
}

// TokenResult contains new tokens.
type TokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
}

// AuthAPIPort defines the interface for auth API operations.
type AuthAPIPort interface {
	// DeviceCode initiates device flow.
	DeviceCode(ctx context.Context) (*DeviceCodeResult, error)

	// DevicePoll polls for device code completion.
	DevicePoll(ctx context.Context, deviceCode string) (*PollResult, error)

	// RefreshToken refreshes the access token.
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResult, error)
}
