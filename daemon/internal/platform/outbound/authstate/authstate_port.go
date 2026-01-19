package authstate

import "context"

// AuthStatePort defines the interface for accessing authentication state.
// This is a platform-level adapter that can be used by any service needing
// authenticated API access without depending on the auth service directly.
type AuthStatePort interface {
	// GetAccessToken returns a valid access token, refreshing if needed.
	// Returns error if not logged in or token refresh fails.
	GetAccessToken(ctx context.Context) (string, error)
}
