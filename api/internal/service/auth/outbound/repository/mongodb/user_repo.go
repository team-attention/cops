package mongodb

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/auth/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

type MongoUserRepository struct {
	logger    *slog.Logger
	usersColl *mongo.Collection
}

func NewMongoUserRepository(l *slog.Logger, db *mongo.Database) *MongoUserRepository {
	return &MongoUserRepository{
		logger:    l.With(slog.String("name", "auth.repository.mongodb.user")),
		usersColl: db.Collection(mongoschema.UserCollectionName),
	}
}

func (r *MongoUserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	var schema mongoschema.User
	schema.FromDomain(user)

	result, err := r.usersColl.InsertOne(ctx, schema)
	if err != nil {
		r.logger.Error("failed to create user",
			slog.String("email", user.Email),
			slog.Any("error", err),
		)
		return nil, err
	}

	insertedID, ok := result.InsertedID.(bson.ObjectID)
	if !ok {
		return nil, mongo.ErrNoDocuments
	}

	schema.ID = insertedID
	return schema.ToDomain(), nil
}

func (r *MongoUserRepository) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	objectID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{mongoschema.UserIDField: objectID}

	var schema mongoschema.User
	err = r.usersColl.FindOne(ctx, filter).Decode(&schema)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, mongo.ErrNoDocuments
		}
		r.logger.Error("failed to get user by ID",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return schema.ToDomain(), nil
}

func (r *MongoUserRepository) FindByAccountProvider(ctx context.Context, provider domain.AccountProvider, providerID string) (*domain.User, error) {
	filter := bson.M{
		mongoschema.UserAccountsField: bson.M{
			"$elemMatch": bson.M{
				mongoschema.AccountProviderField:   string(provider),
				mongoschema.AccountProviderIDField: providerID,
			},
		},
	}

	var schema mongoschema.User
	err := r.usersColl.FindOne(ctx, filter).Decode(&schema)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.logger.Error("failed to find user by account provider",
			slog.String("provider", string(provider)),
			slog.String("providerID", providerID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return schema.ToDomain(), nil
}

var _ repository.UserRepositoryPort = (*MongoUserRepository)(nil)
