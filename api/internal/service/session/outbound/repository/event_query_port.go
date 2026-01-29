package repository

import (
	"context"

	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ChangeEventIterator abstracts the MongoDB ChangeStream to maintain port abstraction.
// This allows the Session Service to iterate over change events without depending
// on the concrete mongo.ChangeStream type.
type ChangeEventIterator interface {
	// Next advances the iterator to the next change event.
	// Returns true if there is a next event, false if the iterator is exhausted or an error occurred.
	Next(ctx context.Context) bool

	// Decode unmarshals the current change event into the provided value.
	// Must be called after Next returns true.
	Decode(val any) error

	// ResumeToken returns the current resume token for checkpointing.
	// Can be used to resume the stream from this position after restart.
	ResumeToken() bson.Raw

	// Err returns any error that occurred during iteration.
	// Should be checked after Next returns false.
	Err() error

	// Close closes the iterator and releases resources.
	Close(ctx context.Context) error
}

// EventQueryPort defines the interface for querying and managing events from Session Service.
type EventQueryPort interface {
	// WatchInserts opens a Change Stream watching for insert operations on events collection.
	// Returns a ChangeEventIterator that emits change events for new inserts.
	// Accepts optional resume token to resume from previous position.
	WatchInserts(ctx context.Context, resumeToken bson.Raw) (ChangeEventIterator, error)

	// FindByBatchID retrieves all events with the given batchID.
	// Used to process events in batch groups.
	FindByBatchID(ctx context.Context, batchID string) ([]*domain.Event, error)

	// DeleteByIDs removes events with the given IDs.
	// Called after successful conversion to Session records.
	DeleteByIDs(ctx context.Context, ids []domain.ID) error

	// IncrementRetryCount increments retryCount for a single event.
	// Called when event processing fails.
	IncrementRetryCount(ctx context.Context, id domain.ID) error
}
