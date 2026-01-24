package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/platform/outbound/transcriptsaver"
	"github.com/team-attention/cops/api/internal/platform/util/errutil"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoTranscriptSaver implements TranscriptSaverPort using MongoDB.
type MongoTranscriptSaver struct {
	logger       *slog.Logger
	eventsColl   *mongo.Collection
	projectsColl *mongo.Collection
}

// NewMongoTranscriptSaver creates a new MongoDB transcript saver adapter.
// Returns concrete type; use fx.As in container module for interface conversion.
func NewMongoTranscriptSaver(l *slog.Logger, db *mongo.Database) *MongoTranscriptSaver {
	return &MongoTranscriptSaver{
		logger:       l.With(slog.String("name", "platform.outbound.transcriptsaver")),
		eventsColl:   db.Collection(mongoschema.EventCollectionName),
		projectsColl: db.Collection(mongoschema.ProjectCollectionName),
	}
}

// SaveTranscripts saves a batch of transcripts to the events collection.
func (s *MongoTranscriptSaver) SaveTranscripts(ctx context.Context, batch *transcriptsaver.TranscriptBatch) error {
	if len(batch.Transcripts) == 0 {
		return nil
	}

	if batch.UserID == "" {
		return errors.New("userID is required")
	}

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

	filter := bson.M{
		mongoschema.ProjectIDField:             projectObjID,
		mongoschema.ProjectOrganizationIDField: orgObjID,
	}

	var project bson.M
	err = s.projectsColl.FindOne(ctx, filter).Decode(&project)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errutil.NotFound("project not found in organization")
		}
		s.logger.Error("failed to verify project ownership",
			slog.String("projectId", batch.ProjectID),
			slog.String("organizationId", batch.OrganizationID),
			slog.Any("error", err),
		)
		return errutil.Internal("failed to verify project ownership")
	}

	docs := make([]any, len(batch.Transcripts))
	for i, transcript := range batch.Transcripts {
		doc, err := transcriptToDocument(transcript, projectObjID, userObjID)
		if err != nil {
			s.logger.Error("failed to convert transcript to document",
				slog.Int("index", i),
				slog.String("type", string(transcript.Type)),
				slog.Any("error", err),
			)
			return fmt.Errorf("failed to convert transcript at index %d: %w", i, err)
		}
		docs[i] = doc
	}

	result, err := s.eventsColl.InsertMany(ctx, docs)
	if err != nil {
		s.logger.Error("failed to insert transcripts",
			slog.String("projectId", batch.ProjectID),
			slog.String("organizationId", batch.OrganizationID),
			slog.Int("count", len(batch.Transcripts)),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to insert transcripts: %w", err)
	}

	s.logger.Debug("inserted transcripts to events collection",
		slog.String("projectId", batch.ProjectID),
		slog.String("organizationId", batch.OrganizationID),
		slog.Int("count", len(result.InsertedIDs)),
	)

	return nil
}

// transcriptToDocument converts a Transcript to a BSON document.
func transcriptToDocument(transcript *domain.Transcript, projectObjID, userObjID bson.ObjectID) (bson.M, error) {
	var schema mongoschema.Transcript
	schema.FromDomain(projectObjID, userObjID, transcript)

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

// Compile-time interface implementation check.
var _ transcriptsaver.TranscriptSaverPort = (*MongoTranscriptSaver)(nil)
