package mongodb

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/user/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoUserRepository implements UserRepositoryPort for MongoDB.
type MongoUserRepository struct {
	logger    *slog.Logger
	usersColl *mongo.Collection
}

// NewMongoUserRepository creates a new MongoDB user repository.
func NewMongoUserRepository(l *slog.Logger, db *mongo.Database) *MongoUserRepository {
	return &MongoUserRepository{
		logger:    l.With(slog.String("name", "user.repository.mongodb.user")),
		usersColl: db.Collection(mongoschema.UserCollectionName),
	}
}

// GetByID retrieves a user by their ID.
func (r *MongoUserRepository) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	// 1. Convert userID string to bson.ObjectID using bson.ObjectIDFromHex.
	objectID, err := bson.ObjectIDFromHex(userID)
	// 2. If conversion fails, return nil, error.
	if err != nil {
		return nil, err
	}

	// 3. Create filter with _id field.
	filter := bson.M{mongoschema.UserIDField: objectID}

	// 4. Execute FindOne query on users collection.
	var schema mongoschema.User
	err = r.usersColl.FindOne(ctx, filter).Decode(&schema)
	// 5. If mongo.ErrNoDocuments, return nil, nil.
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		// 6. If other error, log and return nil, error.
		r.logger.Error("failed to get user by ID",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}

	// 7. Convert mongoschema.User to domain.User using ToDomain().
	// 8. Return user, nil.
	return schema.ToDomain(), nil
}

// Interface verification
var _ repository.UserRepositoryPort = (*MongoUserRepository)(nil)
