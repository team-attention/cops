package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/aggregation/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// Adapter implements SessionRecordRepositoryPort using MongoDB.
type Adapter struct {
	logger     *slog.Logger
	collection *mongo.Collection
}

// NewAdapter creates a new MongoDB session record repository adapter.
func NewAdapter(l *slog.Logger, db *mongo.Database) *Adapter {
	return &Adapter{
		logger:     l.With(slog.String("name", "aggregation.repository.mongodb")),
		collection: db.Collection(mongoschema.SessionRecordCollectionName),
	}
}

// SaveBatch saves a batch of session records to MongoDB.
func (a *Adapter) SaveBatch(ctx context.Context, batch *repository.LogBatch) error {
	if len(batch.Records) == 0 {
		return nil
	}

	docs := make([]interface{}, len(batch.Records))
	for i, record := range batch.Records {
		docs[i] = toDocument(record, batch.DaemonID)
	}

	result, err := a.collection.InsertMany(ctx, docs)
	if err != nil {
		a.logger.Error("failed to insert session records",
			slog.String("daemonId", batch.DaemonID),
			slog.Int("count", len(batch.Records)),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to insert session records: %w", err)
	}

	a.logger.Debug("inserted session records",
		slog.String("daemonId", batch.DaemonID),
		slog.Int("count", len(result.InsertedIDs)),
	)

	return nil
}

func toDocument(record shareddomain.SessionRecord, daemonID string) bson.M {
	doc := bson.M{
		mongoschema.SessionRecordUUIDField:        record.UUID,
		mongoschema.SessionRecordParentUUIDField:  record.ParentUUID,
		mongoschema.SessionRecordSessionIDField:   record.SessionID,
		mongoschema.SessionRecordTypeField:        string(record.Type),
		mongoschema.SessionRecordTimestampField:   record.Timestamp,
		mongoschema.SessionRecordCWDField:         record.CWD,
		mongoschema.SessionRecordGitBranchField:   record.GitBranch,
		mongoschema.SessionRecordVersionField:     record.Version,
		mongoschema.SessionRecordUserTypeField:    record.UserType,
		mongoschema.SessionRecordIsSidechainField: record.IsSidechain,
		mongoschema.SessionRecordIsMetaField:      record.IsMeta,
		mongoschema.SessionRecordDaemonIDField:    daemonID,
		mongoschema.SessionRecordCreatedAtField:   time.Now(),
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
		if msg.Content != nil && !msg.Content.IsBlocks && msg.Content.Text != nil {
			doc[mongoschema.SessionRecordMessageContentField] = *msg.Content.Text
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
var _ repository.SessionRecordRepositoryPort = (*Adapter)(nil)
