package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/platform/util/errutil"
	"github.com/team-attention/cops/api/internal/service/event/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoEventRepository implements EventRepositoryPort using MongoDB.
type MongoEventRepository struct {
	logger       *slog.Logger
	eventsColl   *mongo.Collection
	projectsColl *mongo.Collection
}

// NewMongoEventRepository creates a new MongoDB event repository adapter.
func NewMongoEventRepository(l *slog.Logger, db *mongo.Database) *MongoEventRepository {
	return &MongoEventRepository{
		logger:       l.With(slog.String("name", "event.repository.mongodb")),
		eventsColl:   db.Collection(mongoschema.EventCollectionName),
		projectsColl: db.Collection(mongoschema.ProjectCollectionName),
	}
}

// SaveEventBatch saves a batch of event data to the events collection.
func (r *MongoEventRepository) SaveEventBatch(ctx context.Context, batch *repository.EventBatch) (int, error) {
	if len(batch.DataLines) == 0 {
		return 0, nil
	}

	if batch.UserID == "" {
		return 0, errors.New("userID is required")
	}

	projectObjID, err := bson.ObjectIDFromHex(batch.ProjectID)
	if err != nil {
		return 0, errutil.BadRequest("invalid project ID")
	}

	orgObjID, err := bson.ObjectIDFromHex(batch.OrganizationID)
	if err != nil {
		return 0, errutil.BadRequest("invalid organization ID")
	}

	userObjID, err := bson.ObjectIDFromHex(batch.UserID)
	if err != nil {
		return 0, errutil.BadRequest("invalid user ID")
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
			return 0, errutil.NotFound("project not found in organization")
		}
		r.logger.Error("failed to verify project ownership",
			slog.String("projectId", batch.ProjectID),
			slog.String("organizationId", batch.OrganizationID),
			slog.Any("error", err),
		)
		return 0, errutil.Internal("failed to verify project ownership")
	}

	// Parse and filter non-empty lines
	var docs []any
	for _, line := range batch.DataLines {
		if line == "" {
			continue
		}

		// Parse JSON line to any
		var data any
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			r.logger.Warn("failed to parse JSON line, skipping",
				slog.String("batchId", batch.BatchID),
				slog.Any("error", err),
			)
			continue
		}

		doc := bson.M{
			"_id":            bson.NewObjectID(),
			"data":           data,
			"version":        batch.Version,
			"provider":       batch.Provider,
			"projectId":      projectObjID,
			"organizationId": orgObjID,
			"userId":         userObjID,
			"batchId":        batch.BatchID,
			"retryCount":     0,
			"status":         domain.EventStatusPending,
		}
		docs = append(docs, doc)
	}

	if len(docs) == 0 {
		return 0, nil
	}

	result, err := r.eventsColl.InsertMany(ctx, docs)
	if err != nil {
		r.logger.Error("failed to insert events",
			slog.String("projectId", batch.ProjectID),
			slog.String("organizationId", batch.OrganizationID),
			slog.String("batchId", batch.BatchID),
			slog.Int("count", len(docs)),
			slog.Any("error", err),
		)
		return 0, errutil.Internal("failed to insert events")
	}

	r.logger.Debug("inserted events to events collection",
		slog.String("projectId", batch.ProjectID),
		slog.String("organizationId", batch.OrganizationID),
		slog.String("batchId", batch.BatchID),
		slog.Int("count", len(result.InsertedIDs)),
	)

	return len(result.InsertedIDs), nil
}

// Interface verification
var _ repository.EventRepositoryPort = (*MongoEventRepository)(nil)
