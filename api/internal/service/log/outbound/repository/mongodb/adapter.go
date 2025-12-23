package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/log/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

const (
	sessionRecordCollection = "session_records"
)

// SessionRecordDocument represents a session record in MongoDB.
type SessionRecordDocument struct {
	UUID        string    `bson:"uuid"`
	ParentUUID  string    `bson:"parent_uuid"`
	SessionID   string    `bson:"session_id"`
	Type        string    `bson:"type"`
	Timestamp   time.Time `bson:"timestamp"`
	CWD         string    `bson:"cwd"`
	GitBranch   string    `bson:"git_branch"`
	Version     string    `bson:"version"`
	UserType    string    `bson:"user_type"`
	IsSidechain bool      `bson:"is_sidechain"`
	IsMeta      bool      `bson:"is_meta"`
	Slug        string    `bson:"slug,omitempty"`
	RequestID   string    `bson:"request_id,omitempty"`
	DaemonID    string    `bson:"daemon_id"`
	CreatedAt   time.Time `bson:"created_at"`

	// Nested message data (flattened for query efficiency)
	MessageID       string `bson:"message_id,omitempty"`
	MessageType     string `bson:"message_type,omitempty"`
	MessageRole     string `bson:"message_role,omitempty"`
	MessageModel    string `bson:"message_model,omitempty"`
	MessageContent  string `bson:"message_content,omitempty"`
	StopReason      string `bson:"stop_reason,omitempty"`
	InputTokens     int    `bson:"input_tokens,omitempty"`
	OutputTokens    int    `bson:"output_tokens,omitempty"`
	CacheReadTokens int    `bson:"cache_read_tokens,omitempty"`
}

// Adapter implements SessionRecordRepositoryPort using MongoDB.
type Adapter struct {
	logger     *slog.Logger
	collection *mongo.Collection
}

// NewAdapter creates a new MongoDB session record repository adapter.
func NewAdapter(l *slog.Logger, db *mongo.Database) *Adapter {
	return &Adapter{
		logger:     l.With(slog.String("name", "log.repository.mongodb")),
		collection: db.Collection(sessionRecordCollection),
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
		"uuid":         record.UUID,
		"parent_uuid":  record.ParentUUID,
		"session_id":   record.SessionID,
		"type":         string(record.Type),
		"timestamp":    record.Timestamp,
		"cwd":          record.CWD,
		"git_branch":   record.GitBranch,
		"version":      record.Version,
		"user_type":    record.UserType,
		"is_sidechain": record.IsSidechain,
		"is_meta":      record.IsMeta,
		"daemon_id":    daemonID,
		"created_at":   time.Now(),
	}

	if record.Slug != "" {
		doc["slug"] = record.Slug
	}
	if record.RequestID != "" {
		doc["request_id"] = record.RequestID
	}

	if record.Message != nil {
		msg := record.Message
		if msg.ID != "" {
			doc["message_id"] = msg.ID
		}
		if msg.Type != "" {
			doc["message_type"] = msg.Type
		}
		if msg.Role != "" {
			doc["message_role"] = msg.Role
		}
		if msg.Model != "" {
			doc["message_model"] = msg.Model
		}
		if msg.StopReason != "" {
			doc["stop_reason"] = msg.StopReason
		}
		if msg.Content != nil && !msg.Content.IsBlocks && msg.Content.Text != nil {
			doc["message_content"] = *msg.Content.Text
		}
		if msg.Usage != nil {
			doc["input_tokens"] = msg.Usage.InputTokens
			doc["output_tokens"] = msg.Usage.OutputTokens
			doc["cache_read_tokens"] = msg.Usage.CacheReadInputTokens
		}
	}

	return doc
}

// Compile-time interface verification.
var _ repository.SessionRecordRepositoryPort = (*Adapter)(nil)
