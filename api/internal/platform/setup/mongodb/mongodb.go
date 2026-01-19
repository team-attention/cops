package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/team-attention/cops/api/internal/platform/setup/config"
)

// InitMongoDB creates a configured MongoDB database instance.
func InitMongoDB(cfg *config.Config, logger *slog.Logger) (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Configure BSON options to use map[string]interface{} for any type fields.
	// This ensures fields typed as `any` deserialize documents as maps instead of primitive.D,
	// which is required for proper type assertions in converter code.
	bsonOpts := &options.BSONOptions{
		DefaultDocumentM: true,
	}

	clientOpts := options.Client().
		SetBSONOptions(bsonOpts).
		ApplyURI(cfg.MongoDB.URI)

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	logger.Info("MongoDB initialized",
		slog.String("database", cfg.MongoDB.Database),
	)

	return client.Database(cfg.MongoDB.Database), nil
}
