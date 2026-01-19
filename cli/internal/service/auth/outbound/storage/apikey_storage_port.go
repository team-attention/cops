package storage

import "context"

// APIKeyStoragePort defines the interface for API key storage operations.
type APIKeyStoragePort interface {
	// SaveAPIKey stores the API key to local storage (~/.cops/auth.json).
	SaveAPIKey(ctx context.Context, apiKey string) error

	// HasAPIKey checks if an API key exists in local storage.
	HasAPIKey(ctx context.Context) (bool, error)
}
