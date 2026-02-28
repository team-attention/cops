package mongodb

import (
	"context"
	"log/slog"

	"github.com/team-attention/cops/api/internal/platform/util/errutil"
	"github.com/team-attention/cops/api/internal/service/session/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// mongoChangeStreamIterator wraps *mongo.ChangeStream to implement ChangeEventIterator.
// This maintains port abstraction by not exposing the concrete MongoDB type.
type mongoChangeStreamIterator struct {
	cs *mongo.ChangeStream
}

// Next advances the iterator to the next change event.
func (it *mongoChangeStreamIterator) Next(ctx context.Context) bool {
	return it.cs.Next(ctx)
}

// Decode unmarshals the current change event into the provided value.
func (it *mongoChangeStreamIterator) Decode(val any) error {
	return it.cs.Decode(val)
}

// ResumeToken returns the current resume token for checkpointing.
func (it *mongoChangeStreamIterator) ResumeToken() bson.Raw {
	return it.cs.ResumeToken()
}

// Err returns any error that occurred during iteration.
func (it *mongoChangeStreamIterator) Err() error {
	return it.cs.Err()
}

// Close closes the iterator and releases resources.
func (it *mongoChangeStreamIterator) Close(ctx context.Context) error {
	return it.cs.Close(ctx)
}

// Interface verification for mongoChangeStreamIterator.
var _ repository.ChangeEventIterator = (*mongoChangeStreamIterator)(nil)

// MongoEventQuery implements EventQueryPort using MongoDB.
type MongoEventQuery struct {
	logger     *slog.Logger
	eventsColl *mongo.Collection
}

// NewMongoEventQuery creates a new MongoDB event query adapter.
func NewMongoEventQuery(l *slog.Logger, db *mongo.Database) *MongoEventQuery {
	return &MongoEventQuery{
		logger:     l.With(slog.String("name", "session.event_query.mongodb")),
		eventsColl: db.Collection(mongoschema.EventCollectionName),
	}
}

// WatchInserts opens a Change Stream watching for insert operations.
// Returns a ChangeEventIterator that wraps the underlying mongo.ChangeStream.
func (q *MongoEventQuery) WatchInserts(ctx context.Context, resumeToken bson.Raw) (repository.ChangeEventIterator, error) {
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{{Key: "operationType", Value: "insert"}}}},
	}

	opts := options.ChangeStream().SetFullDocument(options.FullDocument("updateLookup"))
	if len(resumeToken) > 0 {
		opts.SetResumeAfter(resumeToken)
	}

	cs, err := q.eventsColl.Watch(ctx, pipeline, opts)
	if err != nil {
		q.logger.Error("failed to open change stream", slog.Any("error", err))
		return nil, errutil.Wrap(errutil.ErrorTypeInternal, "failed to open change stream", err)
	}

	q.logger.Info("change stream opened")

	return &mongoChangeStreamIterator{cs: cs}, nil
}

// FindByBatchID retrieves all events with the given batchID that have not exceeded max retries.
func (q *MongoEventQuery) FindByBatchID(ctx context.Context, batchID string, maxRetries int) ([]*domain.Event, error) {
	filter := bson.M{
		mongoschema.EventBatchIDField:    batchID,
		mongoschema.EventRetryCountField: bson.M{"$lt": maxRetries},
		mongoschema.EventStatusField:     domain.EventStatusPending,
	}

	cursor, err := q.eventsColl.Find(ctx, filter)
	if err != nil {
		return nil, errutil.Wrap(errutil.ErrorTypeInternal, "failed to find events by batchID", err)
	}
	defer cursor.Close(ctx)

	var mongoEvents []*mongoschema.Event
	if err := cursor.All(ctx, &mongoEvents); err != nil {
		return nil, errutil.Wrap(errutil.ErrorTypeInternal, "failed to decode events", err)
	}

	events := make([]*domain.Event, len(mongoEvents))
	for i, e := range mongoEvents {
		events[i] = e.ToDomain()
	}

	return events, nil
}

// DeleteByIDs removes events with the given IDs.
func (q *MongoEventQuery) DeleteByIDs(ctx context.Context, ids []domain.ID) error {
	if len(ids) == 0 {
		return nil
	}

	objectIDs := make([]bson.ObjectID, len(ids))
	for i, id := range ids {
		objectIDs[i], _ = bson.ObjectIDFromHex(string(id))
	}

	filter := bson.M{mongoschema.EventIDField: bson.M{"$in": objectIDs}}

	result, err := q.eventsColl.DeleteMany(ctx, filter)
	if err != nil {
		return errutil.Wrap(errutil.ErrorTypeInternal, "failed to delete events", err)
	}

	q.logger.Debug("deleted events", slog.Int64("count", result.DeletedCount))

	return nil
}

// IncrementRetryCount increments retryCount for a single event.
func (q *MongoEventQuery) IncrementRetryCount(ctx context.Context, id domain.ID) error {
	objectID, _ := bson.ObjectIDFromHex(string(id))
	filter := bson.M{mongoschema.EventIDField: objectID}

	update := bson.M{
		"$inc": bson.M{mongoschema.EventRetryCountField: 1},
	}

	_, err := q.eventsColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return errutil.Wrap(errutil.ErrorTypeInternal, "failed to increment retry count", err)
	}

	return nil
}

// UpdateStatusByIDs updates the status of events with the given IDs.
func (q *MongoEventQuery) UpdateStatusByIDs(ctx context.Context, ids []domain.ID, status domain.EventStatus) error {
	if len(ids) == 0 {
		return nil
	}

	objectIDs := make([]bson.ObjectID, len(ids))
	for i, id := range ids {
		objectIDs[i], _ = bson.ObjectIDFromHex(string(id))
	}

	filter := bson.M{mongoschema.EventIDField: bson.M{"$in": objectIDs}}
	update := bson.M{"$set": bson.M{mongoschema.EventStatusField: status}}

	result, err := q.eventsColl.UpdateMany(ctx, filter, update)
	if err != nil {
		return errutil.Wrap(errutil.ErrorTypeInternal, "failed to update event status", err)
	}

	q.logger.Debug("updated event status",
		slog.String("status", string(status)),
		slog.Int64("count", result.ModifiedCount),
	)

	return nil
}

// Interface verification
var _ repository.EventQueryPort = (*MongoEventQuery)(nil)
