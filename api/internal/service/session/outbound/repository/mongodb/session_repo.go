package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/team-attention/cops/api/internal/platform/util/errutil"
	"github.com/team-attention/cops/api/internal/service/session/outbound/repository"
	"github.com/team-attention/cops/shared/domain/mongoschema"
	session "github.com/team-attention/cops/shared/domain/v2"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// MongoSessionRepository implements SessionRepositoryPort using MongoDB.
type MongoSessionRepository struct {
	logger       *slog.Logger
	sessionsColl *mongo.Collection
}

// NewMongoSessionRepository creates a new MongoDB session repository adapter.
func NewMongoSessionRepository(l *slog.Logger, db *mongo.Database) *MongoSessionRepository {
	return &MongoSessionRepository{
		logger:       l.With(slog.String("name", "session.repository.mongodb")),
		sessionsColl: db.Collection(mongoschema.SessionsCollectionName),
	}
}

// SaveBatch saves multiple Session records to the sessions collection.
func (r *MongoSessionRepository) SaveBatch(ctx context.Context, projectID, userID bson.ObjectID, sessions []*session.Session) error {
	if len(sessions) == 0 {
		return errutil.BadRequest("sessions cannot be empty")
	}

	documents := make([]any, 0, len(sessions))
	for _, s := range sessions {
		doc, err := sessionToDocument(s, projectID, userID)
		if err != nil {
			return errutil.Wrap(errutil.ErrorTypeInternal, "failed to convert session", err)
		}
		documents = append(documents, doc)
	}

	result, err := r.sessionsColl.InsertMany(ctx, documents)
	if err != nil {
		return errutil.Wrap(errutil.ErrorTypeInternal, "failed to insert sessions", err)
	}

	r.logger.Debug("inserted sessions",
		slog.Int("count", len(result.InsertedIDs)),
	)

	return nil
}

// sessionToDocument converts a Session to a BSON document with project/user IDs.
// This follows the same pattern as Transcript.MarshalBSON in transcript.go:
// marshal to bytes, unmarshal to bson.M, add additional fields.
func sessionToDocument(s *session.Session, projectID, userID bson.ObjectID) (bson.M, error) {
	data, err := bson.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	var doc bson.M
	if err := bson.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session to bson.M: %w", err)
	}

	doc[mongoschema.SessionProjectIDField] = projectID
	doc[mongoschema.SessionUserIDField] = userID

	return doc, nil
}

// Interface verification
var _ repository.SessionRepositoryPort = (*MongoSessionRepository)(nil)
