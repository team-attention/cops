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
		collection: db.Collection(mongoschema.SessionRecordCollectionName),
	}
}

// SaveBatch saves a batch of session records to MongoDB.
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
		r.logger.Error("failed to insert session records",
			slog.String("projectId", batch.ProjectID),
			slog.Int("count", len(batch.Records)),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to insert session records: %w", err)
	}

	r.logger.Debug("inserted session records",
		slog.String("projectId", batch.ProjectID),
		slog.Int("count", len(result.InsertedIDs)),
	)

	return nil
}

func toDocument(record shareddomain.SessionRecord, projectObjID bson.ObjectID) bson.M {
	doc := bson.M{
		mongoschema.SessionRecordUUIDField:        record.UUID,
		mongoschema.SessionRecordParentUUIDField:  record.ParentUUID,
		mongoschema.SessionRecordSessionIDField:   record.SessionID,
		mongoschema.SessionRecordProjectIDField:   projectObjID,
		mongoschema.SessionRecordTypeField:        string(record.Type),
		mongoschema.SessionRecordTimestampField:   record.Timestamp,
		mongoschema.SessionRecordCWDField:         record.CWD,
		mongoschema.SessionRecordGitBranchField:   record.GitBranch,
		mongoschema.SessionRecordVersionField:     record.Version,
		mongoschema.SessionRecordUserTypeField:    record.UserType,
		mongoschema.SessionRecordIsSidechainField: record.IsSidechain,
		mongoschema.SessionRecordIsMetaField:      record.IsMeta,
	}

	if record.Slug != "" {
		doc[mongoschema.SessionRecordSlugField] = record.Slug
	}
	if record.RequestID != "" {
		doc[mongoschema.SessionRecordRequestIDField] = record.RequestID
	}

	if record.Message != nil {
		msg := record.Message
		if msg.ID != "" {
			doc[mongoschema.SessionRecordMessageIDField] = msg.ID
		}
		if msg.Type != "" {
			doc[mongoschema.SessionRecordMessageTypeField] = msg.Type
		}
		if msg.Role != "" {
			doc[mongoschema.SessionRecordMessageRoleField] = msg.Role
		}
		if msg.Model != "" {
			doc[mongoschema.SessionRecordMessageModelField] = msg.Model
		}
		if msg.StopReason != "" {
			doc[mongoschema.SessionRecordStopReasonField] = msg.StopReason
		}
		if msg.Content != nil {
			contentBytes, err := sonic.Marshal(msg.Content)
			if err != nil {
				// Log warning but continue - don't fail the batch for one message
				slog.Warn("failed to serialize message content",
					slog.String("messageId", msg.ID),
					slog.Any("error", err),
				)
			} else {
				doc[mongoschema.SessionRecordMessageContentField] = string(contentBytes)
			}
		}
		if msg.Usage != nil {
			doc[mongoschema.SessionRecordInputTokensField] = msg.Usage.InputTokens
			doc[mongoschema.SessionRecordOutputTokensField] = msg.Usage.OutputTokens
			doc[mongoschema.SessionRecordCacheReadTokensField] = msg.Usage.CacheReadInputTokens
		}
	}

	return doc
}

// Compile-time interface verification.
var _ repository.SessionRecordRepositoryPort = (*MongoSessionRecordRepository)(nil)
