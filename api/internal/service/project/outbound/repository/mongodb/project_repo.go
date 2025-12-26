package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/project/outbound/repository"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoProjectRepository implements ProjectRepositoryPort using MongoDB.
type MongoProjectRepository struct {
	logger       *slog.Logger
	projectsColl *mongo.Collection
}

// NewMongoProjectRepository creates a new MongoDB project repository adapter.
func NewMongoProjectRepository(l *slog.Logger, db *mongo.Database) *MongoProjectRepository {
	return &MongoProjectRepository{
		logger:       l.With(slog.String("name", "project.repository.mongodb")),
		projectsColl: db.Collection(mongoschema.ProjectCollectionName),
	}
}

// FindOrCreate finds existing project or creates new one.
// Search order:
// 1. By remote URL (either configured or actual)
// 2. By existing project ID (if provided)
// 3. Create new if not found
func (r *MongoProjectRepository) FindOrCreate(ctx context.Context, configuredURL, actualURL, existingID string) (*repository.FindOrCreateResult, error) {
	// Build $or conditions array with all search criteria
	conditions := []bson.M{}

	// Add remote URL conditions
	if configuredURL != "" {
		conditions = append(conditions, bson.M{mongoschema.ProjectRemoteURLField: configuredURL})
	}
	if actualURL != "" && actualURL != configuredURL {
		conditions = append(conditions, bson.M{mongoschema.ProjectRemoteURLField: actualURL})
	}

	// Add existing ID condition if valid
	if existingID != "" {
		if objectID, err := bson.ObjectIDFromHex(existingID); err == nil {
			conditions = append(conditions, bson.M{mongoschema.ProjectIDField: objectID})
		}
	}

	// Validate at least one condition exists
	if len(conditions) == 0 {
		return nil, fmt.Errorf("no search criteria provided: at least one of configuredURL, actualURL, or existingID must be valid")
	}

	// Execute single findOne with $or filter
	filter := bson.M{"$or": conditions}
	var doc bson.M
	err := r.projectsColl.FindOne(ctx, filter).Decode(&doc)

	// If found, return existing project
	if err == nil {
		projectID := doc[mongoschema.ProjectIDField].(bson.ObjectID).Hex()
		r.logger.Info("found existing project",
			slog.String("projectID", projectID))
		return &repository.FindOrCreateResult{
			ProjectID: projectID,
			IsNew:     false,
		}, nil
	}

	// If error is not "not found", return error
	if err != mongo.ErrNoDocuments {
		r.logger.Error("failed to find project", slog.Any("error", err))
		return nil, fmt.Errorf("failed to find project: %w", err)
	}

	// Not found, create new document
	// Prefer configured URL, fallback to actual URL
	remoteURL := configuredURL
	if remoteURL == "" {
		remoteURL = actualURL
	}

	newDoc := bson.M{
		mongoschema.ProjectRemoteURLField: remoteURL,
	}

	result, err := r.projectsColl.InsertOne(ctx, newDoc)
	if err != nil {
		r.logger.Error("failed to create project", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	newID := result.InsertedID.(bson.ObjectID).Hex()
	r.logger.Info("created new project",
		slog.String("projectID", newID),
		slog.String("remoteURL", remoteURL))

	return &repository.FindOrCreateResult{
		ProjectID: newID,
		IsNew:     true,
	}, nil
}

// Compile-time interface verification
var _ repository.ProjectRepositoryPort = (*MongoProjectRepository)(nil)
