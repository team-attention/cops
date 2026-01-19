package mongodb

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/apikey/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoAPIKeyRepository implements APIKeyRepositoryPort for MongoDB.
type MongoAPIKeyRepository struct {
	logger      *slog.Logger
	apiKeysColl *mongo.Collection
}

// NewMongoAPIKeyRepository creates a new MongoDB API key repository.
func NewMongoAPIKeyRepository(l *slog.Logger, db *mongo.Database) *MongoAPIKeyRepository {
	return &MongoAPIKeyRepository{
		logger:      l.With(slog.String("name", "apikey.repository.mongodb")),
		apiKeysColl: db.Collection(mongoschema.APIKeyCollectionName),
	}
}

// Create stores a new API key and returns the created key with ID.
func (r *MongoAPIKeyRepository) Create(ctx context.Context, apiKey *domain.APIKey) (*domain.APIKey, error) {
	var schema mongoschema.APIKey
	schema.FromDomain(apiKey)

	result, err := r.apiKeysColl.InsertOne(ctx, &schema)
	if err != nil {
		r.logger.Error("failed to create API key",
			slog.Any("error", err),
		)
		return nil, err
	}

	objectID := result.InsertedID.(bson.ObjectID)
	apiKey.ID = domain.ID(objectID.Hex())

	return apiKey, nil
}

// GetByHash retrieves an API key by its hash value.
// Returns nil, nil if not found.
func (r *MongoAPIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	filter := bson.M{mongoschema.APIKeyKeyHashField: keyHash}

	var schema mongoschema.APIKey
	err := r.apiKeysColl.FindOne(ctx, filter).Decode(&schema)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.logger.Error("failed to get API key by hash",
			slog.Any("error", err),
		)
		return nil, err
	}

	return schema.ToDomain(), nil
}

// GetByID retrieves an API key by its ID.
// Returns nil, nil if not found.
func (r *MongoAPIKeyRepository) GetByID(ctx context.Context, keyID string) (*domain.APIKey, error) {
	objectID, err := bson.ObjectIDFromHex(keyID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{mongoschema.APIKeyIDField: objectID}

	var schema mongoschema.APIKey
	err = r.apiKeysColl.FindOne(ctx, filter).Decode(&schema)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.logger.Error("failed to get API key by ID",
			slog.String("keyID", keyID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return schema.ToDomain(), nil
}

// ListByUser retrieves all API keys for a user.
// If includeRevoked is false, only returns active keys.
func (r *MongoAPIKeyRepository) ListByUser(ctx context.Context, userID string, includeRevoked bool) ([]*domain.APIKey, error) {
	objectID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{mongoschema.APIKeyUserIDField: objectID}
	if !includeRevoked {
		filter[mongoschema.APIKeyRevokedAtField] = nil
	}

	cursor, err := r.apiKeysColl.Find(ctx, filter)
	if err != nil {
		r.logger.Error("failed to list API keys by user",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}
	defer cursor.Close(ctx)

	var apiKeys []*domain.APIKey
	for cursor.Next(ctx) {
		var schema mongoschema.APIKey
		if err := cursor.Decode(&schema); err != nil {
			r.logger.Error("failed to decode API key",
				slog.Any("error", err),
			)
			return nil, err
		}
		apiKeys = append(apiKeys, schema.ToDomain())
	}

	if err := cursor.Err(); err != nil {
		r.logger.Error("cursor error while listing API keys",
			slog.Any("error", err),
		)
		return nil, err
	}

	return apiKeys, nil
}

// Revoke marks an API key as revoked by setting revokedAt.
func (r *MongoAPIKeyRepository) Revoke(ctx context.Context, keyID string) error {
	objectID, err := bson.ObjectIDFromHex(keyID)
	if err != nil {
		return err
	}

	filter := bson.M{mongoschema.APIKeyIDField: objectID}
	update := bson.M{
		"$set": bson.M{
			mongoschema.APIKeyRevokedAtField: time.Now(),
		},
	}

	result, err := r.apiKeysColl.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("failed to revoke API key",
			slog.String("keyID", keyID),
			slog.Any("error", err),
		)
		return err
	}

	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// UpdateLastUsedAt updates the lastUsedAt timestamp.
func (r *MongoAPIKeyRepository) UpdateLastUsedAt(ctx context.Context, keyID string) error {
	objectID, err := bson.ObjectIDFromHex(keyID)
	if err != nil {
		return err
	}

	filter := bson.M{mongoschema.APIKeyIDField: objectID}
	update := bson.M{
		"$set": bson.M{
			mongoschema.APIKeyLastUsedAtField: time.Now(),
		},
	}

	_, err = r.apiKeysColl.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("failed to update last used at",
			slog.String("keyID", keyID),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

// Interface verification
var _ repository.APIKeyRepositoryPort = (*MongoAPIKeyRepository)(nil)
