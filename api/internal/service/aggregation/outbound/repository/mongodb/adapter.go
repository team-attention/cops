package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bytedance/sonic"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/platform/util/errutil"
	"github.com/team-attention/cops/api/internal/service/aggregation/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoSessionRecordRepository implements SessionRecordRepositoryPort using MongoDB.
type MongoSessionRecordRepository struct {
	logger       *slog.Logger
	recordsColl  *mongo.Collection
	projectsColl *mongo.Collection
}

// NewMongoSessionRecordRepository creates a new MongoDB session record repository adapter.
func NewMongoSessionRecordRepository(l *slog.Logger, db *mongo.Database) *MongoSessionRecordRepository {
	return &MongoSessionRecordRepository{
		logger:       l.With(slog.String("name", "aggregation.repository.mongodb")),
		recordsColl:  db.Collection(mongoschema.RecordCollectionName),
		projectsColl: db.Collection(mongoschema.ProjectCollectionName),
	}
}

// SaveBatch saves a batch of records to MongoDB.
// Validates project belongs to organization before saving.
func (r *MongoSessionRecordRepository) SaveBatch(ctx context.Context, batch *repository.LogBatch) error {
	if len(batch.Records) == 0 {
		return nil
	}

	// Convert projectID and organizationID to ObjectID
	projectObjID, err := bson.ObjectIDFromHex(batch.ProjectID)
	if err != nil {
		return errutil.BadRequest("invalid project ID")
	}

	orgObjID, err := bson.ObjectIDFromHex(batch.OrganizationID)
	if err != nil {
		return errutil.BadRequest("invalid organization ID")
	}

	// Validate project belongs to organization
	filter := bson.M{
		mongoschema.ProjectIDField:             projectObjID,
		mongoschema.ProjectOrganizationIDField: orgObjID,
	}

	var project bson.M
	err = r.projectsColl.FindOne(ctx, filter).Decode(&project)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errutil.NotFound("project not found in organization")
		}
		r.logger.Error("failed to verify project ownership",
			slog.String("projectId", batch.ProjectID),
			slog.String("organizationId", batch.OrganizationID),
			slog.Any("error", err),
		)
		return errutil.Internal("failed to verify project ownership")
	}

	// Prepare documents for insertion
	docs := make([]interface{}, len(batch.Records))
	for i, record := range batch.Records {
		docs[i] = toDocument(record, projectObjID)
	}

	result, err := r.recordsColl.InsertMany(ctx, docs)
	if err != nil {
		r.logger.Error("failed to insert records",
			slog.String("projectId", batch.ProjectID),
			slog.String("organizationId", batch.OrganizationID),
			slog.Int("count", len(batch.Records)),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to insert records: %w", err)
	}

	r.logger.Debug("inserted records",
		slog.String("projectId", batch.ProjectID),
		slog.String("organizationId", batch.OrganizationID),
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
