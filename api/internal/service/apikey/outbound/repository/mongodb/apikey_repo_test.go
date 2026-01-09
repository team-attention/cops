package mongodb_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/team-attention/cops/api/internal/service/apikey/outbound/repository/mongodb"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// Note: These are integration tests that require a running MongoDB instance.
// They are skipped if MONGODB_URI is not set.

func getTestDB(t *testing.T) *mongo.Database {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		t.Skip("MONGODB_URI not set, skipping integration test")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("failed to connect to MongoDB: %v", err)
	}

	dbName := "cops_test_" + time.Now().Format("20060102150405")
	t.Cleanup(func() {
		_ = client.Database(dbName).Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})

	return client.Database(dbName)
}

func TestMongoAPIKeyRepository_Create(t *testing.T) {
	db := getTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := mongodb.NewMongoAPIKeyRepository(logger, db)

	t.Run("stores API key and returns with ID", func(t *testing.T) {
		apiKey := &domain.APIKey{
			UserID:    "507f1f77bcf86cd799439011",
			Name:      "Test Key",
			KeyPrefix: "abc12345",
			KeyHash:   "testhash123",
			CreatedAt: time.Now(),
		}

		created, err := repo.Create(context.Background(), apiKey)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if created.ID == "" {
			t.Error("Create() returned key without ID")
		}
	})
}

func TestMongoAPIKeyRepository_GetByHash(t *testing.T) {
	db := getTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := mongodb.NewMongoAPIKeyRepository(logger, db)

	t.Run("when key exists returns the API key", func(t *testing.T) {
		// Create a key first
		userID, _ := bson.ObjectIDFromHex("507f1f77bcf86cd799439011")
		schema := &mongoschema.APIKey{
			UserID: userID,
		}
		schema.Name = "Test Key"
		schema.KeyPrefix = "abc12345"
		schema.KeyHash = "uniquehash123"
		schema.CreatedAt = time.Now()

		_, err := db.Collection(mongoschema.APIKeyCollectionName).InsertOne(context.Background(), schema)
		if err != nil {
			t.Fatalf("failed to insert test key: %v", err)
		}

		// Find by hash
		found, err := repo.GetByHash(context.Background(), "uniquehash123")
		if err != nil {
			t.Fatalf("GetByHash() error = %v", err)
		}
		if found == nil {
			t.Fatal("GetByHash() returned nil for existing key")
		}
		if found.KeyHash != "uniquehash123" {
			t.Errorf("GetByHash() keyHash = %q, want %q", found.KeyHash, "uniquehash123")
		}
	})

	t.Run("when key does not exist returns nil, nil", func(t *testing.T) {
		found, err := repo.GetByHash(context.Background(), "nonexistenthash")
		if err != nil {
			t.Fatalf("GetByHash() error = %v", err)
		}
		if found != nil {
			t.Error("GetByHash() expected nil for non-existent key")
		}
	})
}

func TestMongoAPIKeyRepository_ListByUser(t *testing.T) {
	db := getTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := mongodb.NewMongoAPIKeyRepository(logger, db)

	userID, _ := bson.ObjectIDFromHex("507f1f77bcf86cd799439011")
	revokedAt := time.Now()

	// Create test keys
	keys := []any{
		&mongoschema.APIKey{
			UserID:   userID,
			APIKey:   domain.APIKey{Name: "Active Key 1", KeyPrefix: "abc", KeyHash: "hash1", CreatedAt: time.Now()},
		},
		&mongoschema.APIKey{
			UserID:   userID,
			APIKey:   domain.APIKey{Name: "Active Key 2", KeyPrefix: "def", KeyHash: "hash2", CreatedAt: time.Now()},
		},
		&mongoschema.APIKey{
			UserID:   userID,
			APIKey:   domain.APIKey{Name: "Revoked Key", KeyPrefix: "ghi", KeyHash: "hash3", CreatedAt: time.Now(), RevokedAt: &revokedAt},
		},
	}
	_, err := db.Collection(mongoschema.APIKeyCollectionName).InsertMany(context.Background(), keys)
	if err != nil {
		t.Fatalf("failed to insert test keys: %v", err)
	}

	t.Run("with includeRevoked=false returns only active keys", func(t *testing.T) {
		found, err := repo.ListByUser(context.Background(), "507f1f77bcf86cd799439011", false)
		if err != nil {
			t.Fatalf("ListByUser() error = %v", err)
		}
		if len(found) != 2 {
			t.Errorf("ListByUser() returned %d keys, want 2", len(found))
		}
	})

	t.Run("with includeRevoked=true returns all keys", func(t *testing.T) {
		found, err := repo.ListByUser(context.Background(), "507f1f77bcf86cd799439011", true)
		if err != nil {
			t.Fatalf("ListByUser() error = %v", err)
		}
		if len(found) != 3 {
			t.Errorf("ListByUser() returned %d keys, want 3", len(found))
		}
	})
}

func TestMongoAPIKeyRepository_Revoke(t *testing.T) {
	db := getTestDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := mongodb.NewMongoAPIKeyRepository(logger, db)

	t.Run("sets revokedAt timestamp", func(t *testing.T) {
		userID, _ := bson.ObjectIDFromHex("507f1f77bcf86cd799439011")
		schema := &mongoschema.APIKey{
			UserID: userID,
		}
		schema.Name = "Test Key"
		schema.KeyPrefix = "abc12345"
		schema.KeyHash = "revokehash"
		schema.CreatedAt = time.Now()

		result, err := db.Collection(mongoschema.APIKeyCollectionName).InsertOne(context.Background(), schema)
		if err != nil {
			t.Fatalf("failed to insert test key: %v", err)
		}

		keyID := result.InsertedID.(bson.ObjectID).Hex()

		err = repo.Revoke(context.Background(), keyID)
		if err != nil {
			t.Fatalf("Revoke() error = %v", err)
		}

		// Verify revokedAt is set
		var updated mongoschema.APIKey
		err = db.Collection(mongoschema.APIKeyCollectionName).FindOne(
			context.Background(),
			bson.M{mongoschema.APIKeyIDField: result.InsertedID},
		).Decode(&updated)
		if err != nil {
			t.Fatalf("failed to find updated key: %v", err)
		}

		if updated.RevokedAt == nil {
			t.Error("Revoke() did not set revokedAt")
		}
	})
}
