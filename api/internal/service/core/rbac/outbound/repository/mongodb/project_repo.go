package mongodb

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoProjectRepository implements ProjectRepositoryPort for MongoDB.
type MongoProjectRepository struct {
	logger       *slog.Logger
	projectsColl *mongo.Collection
}

// NewMongoProjectRepository creates a new MongoDB project repository for RBAC.
func NewMongoProjectRepository(l *slog.Logger, db *mongo.Database) *MongoProjectRepository {
	return &MongoProjectRepository{
		logger:       l.With(slog.String("name", "rbac.repository.mongodb.project")),
		projectsColl: db.Collection(mongoschema.ProjectCollectionName),
	}
}

// GetByID retrieves a project by its ID.
func (r *MongoProjectRepository) GetByID(ctx context.Context, projectID string) (*domain.Project, error) {
	objectID, err := bson.ObjectIDFromHex(projectID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{mongoschema.ProjectIDField: objectID}

	var schema mongoschema.Project
	err = r.projectsColl.FindOne(ctx, filter).Decode(&schema)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.logger.Error("failed to get project by ID",
			slog.String("projectID", projectID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return schema.ToDomain(), nil
}

// Interface verification
var _ repository.ProjectRepositoryPort = (*MongoProjectRepository)(nil)
