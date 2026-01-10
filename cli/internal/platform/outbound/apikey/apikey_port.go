package apikey

import "context"

// APIKeyPort defines the interface for retrieving API keys.
// This is a platform-level adapter for obtaining API keys
// from various sources (filesystem, environment variables).
type APIKeyPort interface {
	// GetAPIKey returns the API key for authentication.
	// Priority: 1) COPS_API_KEY environment variable, 2) ~/.cops/auth.json file.
	// Returns error if no API key is available or the key is empty.
	GetAPIKey(ctx context.Context) (string, error)
}
