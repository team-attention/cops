package mongodb

import (
	"context"
	"errors"
	"fmt"
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

// SaveLogBatch saves a batch of JSONL transcripts to events collection.
// Validates project belongs to organization before saving.
func (r *MongoEventRepository) SaveLogBatch(ctx context.Context, batch *repository.LogBatch) error {
	if len(batch.Transcripts) == 0 {
		return nil
	}

	if batch.UserID == "" {
		return errors.New("userID is required")
	}

	// Convert projectID, organizationID, and userID to ObjectID
	projectObjID, err := bson.ObjectIDFromHex(batch.ProjectID)
	if err != nil {
		return errutil.BadRequest("invalid project ID")
	}

	orgObjID, err := bson.ObjectIDFromHex(batch.OrganizationID)
	if err != nil {
		return errutil.BadRequest("invalid organization ID")
	}

	userObjID, err := bson.ObjectIDFromHex(batch.UserID)
	if err != nil {
		return errutil.BadRequest("invalid user ID")
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
	docs := make([]any, len(batch.Transcripts))
	for i, transcript := range batch.Transcripts {
		doc, err := transcriptToDocument(transcript, projectObjID, userObjID)
		if err != nil {
			r.logger.Error("failed to convert transcript to document",
				slog.Int("index", i),
				slog.String("type", string(transcript.Type)),
				slog.Any("error", err),
			)
			return fmt.Errorf("failed to convert transcript at index %d: %w", i, err)
		}
		docs[i] = doc
	}

	result, err := r.eventsColl.InsertMany(ctx, docs)
	if err != nil {
		r.logger.Error("failed to insert log transcripts",
			slog.String("projectId", batch.ProjectID),
			slog.String("organizationId", batch.OrganizationID),
			slog.Int("count", len(batch.Transcripts)),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to insert log transcripts: %w", err)
	}

	r.logger.Debug("inserted log transcripts to events collection",
		slog.String("projectId", batch.ProjectID),
		slog.String("organizationId", batch.OrganizationID),
		slog.Int("count", len(result.InsertedIDs)),
	)

	return nil
}

// transcriptToDocument converts a Transcript to a BSON document using mongoschema.Transcript.
func transcriptToDocument(transcript *domain.Transcript, projectObjID, userObjID bson.ObjectID) (bson.M, error) {
	var schema mongoschema.Transcript
	schema.FromDomain(projectObjID, userObjID, transcript)

	// Use bson.Marshal which calls MarshalBSON
	data, err := bson.Marshal(&schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transcript: %w", err)
	}

	var doc bson.M
	if err := bson.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transcript to bson.M: %w", err)
	}

	return doc, nil
}

// Interface verification
var _ repository.EventRepositoryPort = (*MongoEventRepository)(nil)
