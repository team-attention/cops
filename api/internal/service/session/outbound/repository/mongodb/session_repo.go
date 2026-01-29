package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/team-attention/cops/api/internal/platform/util/errutil"
	"github.com/team-attention/cops/api/internal/service/session/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
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
func (r *MongoSessionRepository) SaveBatch(ctx context.Context, projectID, userID domain.ID, sessions []*session.Session) error {
	if len(sessions) == 0 {
		return errutil.BadRequest("sessions cannot be empty")
	}

	projectObjID, _ := bson.ObjectIDFromHex(string(projectID))
	userObjID, _ := bson.ObjectIDFromHex(string(userID))

	documents := make([]any, 0, len(sessions))
	for _, s := range sessions {
		doc, err := sessionToDocument(s, projectObjID, userObjID)
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
func sessionToDocument(s *session.Session, projectID, userID bson.ObjectID) (bson.M, error) {
	// Use mongoschema.Session wrapper for proper BSON marshaling
	wrapper := mongoschema.Session{
		Session:   s,
		ProjectID: projectID,
		UserID:    userID,
	}

	data, err := bson.Marshal(wrapper)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	var doc bson.M
	if err := bson.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session to bson.M: %w", err)
	}

	return doc, nil
}

// Interface verification
var _ repository.SessionRepositoryPort = (*MongoSessionRepository)(nil)
