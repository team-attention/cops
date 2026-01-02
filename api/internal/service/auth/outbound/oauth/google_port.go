package oauth

import "context"

// GoogleUserInfo contains user profile from Google.
type GoogleUserInfo struct {
	ID            string
	Email         string
	Name          string
	Picture       string
	EmailVerified bool
}

// DeviceCodeResponse contains device code flow data.
type DeviceCodeResponse struct {
	DeviceCode      string
	UserCode        string
	VerificationURL string
	ExpiresIn       int
	Interval        int
}

// TokenResponse contains OAuth tokens from Google.
type TokenResponse struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// GoogleOAuthPort defines interface for Google OAuth operations.
type GoogleOAuthPort interface {
	// ExchangeCode exchanges authorization code for tokens (web flow).
	ExchangeCode(ctx context.Context, code, redirectURI string) (*TokenResponse, error)

	// GetUserInfo fetches user profile using access token.
	GetUserInfo(ctx context.Context, accessToken string) (*GoogleUserInfo, error)

	// InitiateDeviceFlow starts device code flow.
	InitiateDeviceFlow(ctx context.Context) (*DeviceCodeResponse, error)

	// PollDeviceCode polls for device code authorization.
	// Returns nil TokenResponse if authorization is still pending.
	PollDeviceCode(ctx context.Context, deviceCode string) (*TokenResponse, error)
}
