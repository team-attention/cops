package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/auth/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

type MongoDeviceCodeRepository struct {
	logger *slog.Logger
	coll   *mongo.Collection
}

func NewMongoDeviceCodeRepository(l *slog.Logger, db *mongo.Database) *MongoDeviceCodeRepository {
	return &MongoDeviceCodeRepository{
		logger: l.With(slog.String("name", "auth.repository.mongodb.device_code")),
		coll:   db.Collection(mongoschema.DeviceCodeCollectionName),
	}
}

func (r *MongoDeviceCodeRepository) Create(ctx context.Context, deviceCode *domain.DeviceCode) (*domain.DeviceCode, error) {
	var schema mongoschema.DeviceCode
	schema.FromDomain(deviceCode)

	result, err := r.coll.InsertOne(ctx, schema)
	if err != nil {
		r.logger.Error("failed to insert device code",
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to insert device code: %w", err)
	}

	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		deviceCode.ID = domain.ID(oid.Hex())
	}

	return deviceCode, nil
}

func (r *MongoDeviceCodeRepository) GetByID(ctx context.Context, id string) (*domain.DeviceCode, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		r.logger.Warn("invalid device code ID format",
			slog.String("id", id),
			slog.Any("error", err),
		)
		return nil, nil
	}

	filter := bson.M{mongoschema.DeviceCodeIDField: objectID}

	var schema mongoschema.DeviceCode
	err = r.coll.FindOne(ctx, filter).Decode(&schema)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.logger.Error("failed to find device code by ID",
			slog.String("id", id),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to find device code: %w", err)
	}

	return schema.ToDomain(), nil
}

func (r *MongoDeviceCodeRepository) GetByUserCode(ctx context.Context, userCode string) (*domain.DeviceCode, error) {
	filter := bson.M{mongoschema.DeviceCodeUserCodeField: userCode}

	var schema mongoschema.DeviceCode
	err := r.coll.FindOne(ctx, filter).Decode(&schema)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.logger.Error("failed to find device code by user code",
			slog.String("userCode", userCode),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("failed to find device code: %w", err)
	}

	return schema.ToDomain(), nil
}

func (r *MongoDeviceCodeRepository) Approve(ctx context.Context, userCode string, userID domain.ID) error {
	userObjectID, err := bson.ObjectIDFromHex(string(userID))
	if err != nil {
		r.logger.Error("invalid user ID format",
			slog.String("userID", string(userID)),
			slog.Any("error", err),
		)
		return fmt.Errorf("invalid user ID: %w", err)
	}

	filter := bson.M{
		mongoschema.DeviceCodeUserCodeField:  userCode,
		mongoschema.DeviceCodeApprovedField:  false,
		mongoschema.DeviceCodeExpiresAtField: bson.M{"$gt": time.Now()},
	}

	update := bson.M{
		"$set": bson.M{
			mongoschema.DeviceCodeApprovedField: true,
			mongoschema.DeviceCodeUserIDField:   userObjectID,
		},
	}

	result, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("failed to approve device code",
			slog.String("userCode", userCode),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to approve device code: %w", err)
	}

	if result.MatchedCount == 0 {
		deviceCode, err := r.GetByUserCode(ctx, userCode)
		if err != nil {
			return fmt.Errorf("failed to check device code: %w", err)
		}

		if deviceCode == nil {
			return fmt.Errorf("device code not found")
		}

		if deviceCode.Approved {
			return fmt.Errorf("device code already approved")
		}

		if deviceCode.ExpiresAt.Before(time.Now()) {
			return fmt.Errorf("device code expired")
		}

		return fmt.Errorf("failed to approve device code")
	}

	r.logger.Info("device code approved",
		slog.String("userCode", userCode),
		slog.String("userID", string(userID)),
	)

	return nil
}

var _ repository.DeviceCodeRepositoryPort = (*MongoDeviceCodeRepository)(nil)
