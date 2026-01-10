package mongodb

import (
	"context"
	"errors"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/event/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoEventRepository implements EventRepositoryPort using MongoDB.
type MongoEventRepository struct {
	logger     *slog.Logger
	eventsColl *mongo.Collection
}

// NewMongoEventRepository creates a new MongoDB event repository adapter.
func NewMongoEventRepository(l *slog.Logger, db *mongo.Database) *MongoEventRepository {
	return &MongoEventRepository{
		logger:     l.With(slog.String("name", "event.repository.mongodb")),
		eventsColl: db.Collection(mongoschema.EventCollectionName),
	}
}

// SaveEvent saves a single event to storage.
func (r *MongoEventRepository) SaveEvent(ctx context.Context, userID string, event *domain.Event) error {
	if userID == "" {
		return errors.New("userID is required")
	}

	var schema mongoschema.Event
	schema.FromDomain(userID, event)

	doc, err := schema.ToBSONDocument()
	if err != nil {
		r.logger.Error("failed to convert event to BSON document",
			slog.Any("error", err),
		)
		return err
	}

	_, err = r.eventsColl.InsertOne(ctx, doc)
	if err != nil {
		r.logger.Error("failed to save event",
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

// SaveEvents saves multiple events to storage in a batch.
func (r *MongoEventRepository) SaveEvents(ctx context.Context, userID string, events []*domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	if userID == "" {
		return errors.New("userID is required")
	}

	docs := make([]any, len(events))
	for i, event := range events {
		var schema mongoschema.Event
		schema.FromDomain(userID, event)

		doc, err := schema.ToBSONDocument()
		if err != nil {
			r.logger.Error("failed to convert event to BSON document",
				slog.Int("index", i),
				slog.Any("error", err),
			)
			return err
		}
		docs[i] = doc
	}

	_, err := r.eventsColl.InsertMany(ctx, docs)
	if err != nil {
		r.logger.Error("failed to save events",
			slog.Int("count", len(events)),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

// Interface verification
var _ repository.EventRepositoryPort = (*MongoEventRepository)(nil)
