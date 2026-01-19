package api

import "context"

// DeviceCodeResult contains device code flow initiation response.
type DeviceCodeResult struct {
	DeviceCode      string
	UserCode        string
	VerificationURL string
	ExpiresIn       int
	Interval        int
}

// DevicePollResult contains device code polling response.
type DevicePollResult struct {
	Pending      bool
	AccessToken  string
	RefreshToken string
	ExpiresAt    int64
}

// AuthAPIPort defines the interface for auth API calls.
type AuthAPIPort interface {
	// DeviceCode initiates device flow authentication.
	DeviceCode(ctx context.Context) (*DeviceCodeResult, error)

	// DevicePoll polls for device code approval status.
	DevicePoll(ctx context.Context, deviceCode string) (*DevicePollResult, error)

	// IssueAPIKey creates a new API key using the access token.
	// Returns the plain-text API key (only returned once).
	IssueAPIKey(ctx context.Context, accessToken string, name string) (string, error)
}
