package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ResumeTokenKey identifies which change stream this token belongs to.
const ResumeTokenKey = "session_service_events"

// ResumeTokenRepositoryPort defines the interface for Change Stream resume token persistence.
type ResumeTokenRepositoryPort interface {
	// GetResumeToken retrieves the stored resume token for the given key.
	// Returns nil if no token exists (first run scenario).
	GetResumeToken(ctx context.Context, key string) (bson.Raw, error)

	// SaveResumeToken persists the resume token for the given key.
	// Uses upsert to handle both insert and update cases.
	SaveResumeToken(ctx context.Context, key string, token bson.Raw) error
}
