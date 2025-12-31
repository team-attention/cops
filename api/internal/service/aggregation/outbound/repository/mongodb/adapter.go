package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bytedance/sonic"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/aggregation/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoSessionRecordRepository implements SessionRecordRepositoryPort using MongoDB.
type MongoSessionRecordRepository struct {
	logger     *slog.Logger
	collection *mongo.Collection
}

// NewMongoSessionRecordRepository creates a new MongoDB session record repository adapter.
func NewMongoSessionRecordRepository(l *slog.Logger, db *mongo.Database) *MongoSessionRecordRepository {
	return &MongoSessionRecordRepository{
		logger:     l.With(slog.String("name", "aggregation.repository.mongodb")),
		collection: db.Collection(mongoschema.RecordCollectionName),
	}
}

// SaveBatch saves a batch of records to MongoDB.
func (r *MongoSessionRecordRepository) SaveBatch(ctx context.Context, batch *repository.LogBatch) error {
	if len(batch.Records) == 0 {
		return nil
	}

	// Convert project_id to ObjectID
	projectObjID, err := bson.ObjectIDFromHex(batch.ProjectID)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}

	docs := make([]interface{}, len(batch.Records))
	for i, record := range batch.Records {
		docs[i] = toDocument(record, projectObjID)
	}

	result, err := r.collection.InsertMany(ctx, docs)
	if err != nil {
		r.logger.Error("failed to insert records",
			slog.String("projectId", batch.ProjectID),
			slog.Int("count", len(batch.Records)),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to insert records: %w", err)
	}

	r.logger.Debug("inserted records",
		slog.String("projectId", batch.ProjectID),
		slog.Int("count", len(result.InsertedIDs)),
	)

	return nil
}

func toDocument(record shareddomain.Record, projectObjID bson.ObjectID) bson.M {
	// Marshal record to JSON using record.MarshalJSON() (produces flat JSON)
	jsonBytes, err := record.MarshalJSON()
	if err != nil {
		// Log error and return minimal document
		slog.Error("failed to marshal record to JSON",
			slog.String("type", string(record.Type)),
			slog.Any("error", err),
		)
		return bson.M{
			mongoschema.RecordProjectIDField: projectObjID,
			mongoschema.RecordTypeField:      string(record.Type),
		}
	}

	// Unmarshal JSON into bson.M
	var doc bson.M
	if err := sonic.Unmarshal(jsonBytes, &doc); err != nil {
		// Log error and return minimal document
		slog.Error("failed to unmarshal JSON to bson.M",
			slog.String("type", string(record.Type)),
			slog.Any("error", err),
		)
		return bson.M{
			mongoschema.RecordProjectIDField: projectObjID,
			mongoschema.RecordTypeField:      string(record.Type),
		}
	}

	// Add projectId field with projectObjID
	doc[mongoschema.RecordProjectIDField] = projectObjID

	return doc
}

// Compile-time interface verification.
var _ repository.SessionRecordRepositoryPort = (*MongoSessionRecordRepository)(nil)
