package mongodb

import (
	"context"
	"log/slog"
	"time"

	"github.com/team-attention/cops/api/internal/platform/util/errutil"
	"github.com/team-attention/cops/api/internal/service/session/outbound/repository"
	"github.com/team-attention/cops/shared/domain/mongoschema"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoResumeTokenRepository implements ResumeTokenRepositoryPort using MongoDB.
type MongoResumeTokenRepository struct {
	logger *slog.Logger
	coll   *mongo.Collection
}

// NewMongoResumeTokenRepository creates a new MongoDB resume token repository.
func NewMongoResumeTokenRepository(l *slog.Logger, db *mongo.Database) *MongoResumeTokenRepository {
	return &MongoResumeTokenRepository{
		logger: l.With(slog.String("name", "session.resume_token.mongodb")),
		coll:   db.Collection(mongoschema.ResumeTokensCollectionName),
	}
}

// GetResumeToken retrieves the stored resume token for the given key.
func (r *MongoResumeTokenRepository) GetResumeToken(ctx context.Context, key string) (bson.Raw, error) {
	filter := bson.M{mongoschema.ResumeTokenKeyField: key}

	var result bson.M
	err := r.coll.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, errutil.Wrap(errutil.ErrorTypeInternal, "failed to get resume token", err)
	}

	token, ok := result[mongoschema.ResumeTokenTokenField].(bson.Raw)
	if !ok {
		return nil, nil
	}

	return token, nil
}

// SaveResumeToken persists the resume token for the given key.
func (r *MongoResumeTokenRepository) SaveResumeToken(ctx context.Context, key string, token bson.Raw) error {
	filter := bson.M{mongoschema.ResumeTokenKeyField: key}

	update := bson.M{
		"$set": bson.M{
			mongoschema.ResumeTokenTokenField:     token,
			mongoschema.ResumeTokenUpdatedAtField: time.Now(),
		},
	}

	opts := options.UpdateOne().SetUpsert(true)
	_, err := r.coll.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return errutil.Wrap(errutil.ErrorTypeInternal, "failed to save resume token", err)
	}

	r.logger.Debug("saved resume token", slog.String("key", key))

	return nil
}

// Interface verification
var _ repository.ResumeTokenRepositoryPort = (*MongoResumeTokenRepository)(nil)
