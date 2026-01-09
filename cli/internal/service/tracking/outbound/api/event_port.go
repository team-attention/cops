package api

import "context"

// EventAPIPort defines the interface for event API operations.
type EventAPIPort interface {
	// SendEvents sends a batch of raw JSON event strings to the API server.
	// Uses API key for Bearer authentication.
	// Returns error on network failure or server error.
	SendEvents(ctx context.Context, apiKey string, events []string) error
}
