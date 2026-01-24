package mongodb

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/team-attention/cops/api/internal/platform/domain"
	"github.com/team-attention/cops/api/internal/platform/domain/mongoschema"
	"github.com/team-attention/cops/api/internal/service/retry/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

// MongoRetryRepository implements RetryRepositoryPort using MongoDB.
type MongoRetryRepository struct {
	logger     *slog.Logger
	collection *mongo.Collection
}

// NewMongoRetryRepository creates a new MongoDB retry repository adapter.
// Returns concrete type; use fx.As in container module for interface conversion.
func NewMongoRetryRepository(l *slog.Logger, db *mongo.Database) *MongoRetryRepository {
	return &MongoRetryRepository{
		logger:     l.With(slog.String("name", "retry.repository.mongodb")),
		collection: db.Collection(mongoschema.FailedEventCollectionName),
	}
}

// FindRetryableEvents finds events eligible for retry processing.
func (r *MongoRetryRepository) FindRetryableEvents(ctx context.Context, maxRetries int, limit int) ([]*domain.FailedEvent, error) {
	filter := bson.M{
		mongoschema.FailedEventStatusField:     domain.FailedEventStatusFailed,
		mongoschema.FailedEventRetryCountField: bson.M{"$lt": maxRetries},
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: mongoschema.FailedEventCreatedAtField, Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		r.logger.Error("failed to find retryable events",
			slog.Any("error", err),
		)
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []*domain.FailedEvent
	for cursor.Next(ctx) {
		var schema mongoschema.FailedEvent
		if err := cursor.Decode(&schema); err != nil {
			r.logger.Warn("failed to decode failed event, skipping",
				slog.Any("error", err),
			)
			continue
		}
		events = append(events, schema.ToDomain())
	}

	if err := cursor.Err(); err != nil {
		r.logger.Error("cursor error while iterating failed events",
			slog.Any("error", err),
		)
		return events, err
	}

	return events, nil
}

// IncrementRetryCount atomically increments retry count for concurrent safety.
func (r *MongoRetryRepository) IncrementRetryCount(ctx context.Context, eventID shareddomain.ID) (*domain.FailedEvent, error) {
	objID, err := bson.ObjectIDFromHex(string(eventID))
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		mongoschema.FailedEventIDField:     objID,
		mongoschema.FailedEventStatusField: domain.FailedEventStatusFailed,
	}

	now := time.Now()
	update := bson.M{
		"$inc": bson.M{mongoschema.FailedEventRetryCountField: 1},
		"$set": bson.M{mongoschema.FailedEventLastRetryAtField: now},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var schema mongoschema.FailedEvent
	err = r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&schema)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.logger.Error("failed to increment retry count",
			slog.String("eventID", string(eventID)),
			slog.Any("error", err),
		)
		return nil, err
	}

	return schema.ToDomain(), nil
}

// MarkPermanentlyFailed updates event status to permanently_failed.
func (r *MongoRetryRepository) MarkPermanentlyFailed(ctx context.Context, eventID shareddomain.ID, reason string) error {
	objID, err := bson.ObjectIDFromHex(string(eventID))
	if err != nil {
		return err
	}

	filter := bson.M{mongoschema.FailedEventIDField: objID}
	update := bson.M{
		"$set": bson.M{
			mongoschema.FailedEventStatusField:        domain.FailedEventStatusPermanentlyFailed,
			mongoschema.FailedEventFailureReasonField: reason,
		},
	}

	_, err = r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("failed to mark event as permanently failed",
			slog.String("eventID", string(eventID)),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

// DeleteEvent removes a successfully processed event.
func (r *MongoRetryRepository) DeleteEvent(ctx context.Context, eventID shareddomain.ID) error {
	objID, err := bson.ObjectIDFromHex(string(eventID))
	if err != nil {
		return err
	}

	filter := bson.M{mongoschema.FailedEventIDField: objID}

	_, err = r.collection.DeleteOne(ctx, filter)
	if err != nil {
		r.logger.Error("failed to delete event",
			slog.String("eventID", string(eventID)),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

// SaveFailedEvent inserts a new failed event record.
func (r *MongoRetryRepository) SaveFailedEvent(ctx context.Context, event *domain.FailedEvent) error {
	var schema mongoschema.FailedEvent
	schema.FromDomain(event)
	schema.CreatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, &schema)
	if err != nil {
		r.logger.Error("failed to save failed event",
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

// Compile-time interface implementation check.
var _ repository.RetryRepositoryPort = (*MongoRetryRepository)(nil)
