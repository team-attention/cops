package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/project/outbound/repository"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

type MongoProjectRepository struct {
	logger       *slog.Logger
	projectsColl *mongo.Collection
}

func NewMongoProjectRepository(l *slog.Logger, db *mongo.Database) *MongoProjectRepository {
	return &MongoProjectRepository{
		logger:       l.With(slog.String("name", "project.repository.mongodb")),
		projectsColl: db.Collection(mongoschema.ProjectCollectionName),
	}
}

// FindOrCreate finds existing project by ID or URLs, or creates a new one.
// No business logic - just builds query from provided parameters.
// Empty values are naturally filtered out when building conditions.
func (r *MongoProjectRepository) FindOrCreate(ctx context.Context, params repository.FindOrCreateParams) (*repository.FindOrCreateResult, error) {
	// Build $or conditions from provided parameters
	conditions := r.buildSearchConditions(params)

	// If we have search conditions, try to find existing project
	if len(conditions) > 0 {
		project, err := r.findByConditions(ctx, conditions)
		if err != nil {
			return nil, err
		}
		if project != nil {
			r.logger.Info("found existing project",
				slog.String("projectID", project.ProjectID))
			return project, nil
		}
	}

	// Not found, create new project
	return r.createProject(ctx, params)
}

// buildSearchConditions creates $or conditions from params.
// Only adds conditions for non-empty values.
func (r *MongoProjectRepository) buildSearchConditions(params repository.FindOrCreateParams) []bson.M {
	conditions := make([]bson.M, 0, 2)

	// Add ID condition if provided
	if params.ExistingID != "" {
		if objectID, err := bson.ObjectIDFromHex(params.ExistingID); err == nil {
			conditions = append(conditions, bson.M{mongoschema.ProjectIDField: objectID})
		}
	}

	// Add URL conditions if provided (using $in for multiple URLs)
	urls := r.collectNonEmptyURLs(params.ConfiguredURL, params.ActualURL)
	if len(urls) > 0 {
		conditions = append(conditions, bson.M{
			mongoschema.ProjectRemoteURLField: bson.M{"$in": urls},
		})
	}

	return conditions
}

// collectNonEmptyURLs gathers non-empty, unique URLs.
func (r *MongoProjectRepository) collectNonEmptyURLs(configuredURL, actualURL string) []string {
	urls := make([]string, 0, 2)
	if configuredURL != "" {
		urls = append(urls, configuredURL)
	}
	if actualURL != "" && actualURL != configuredURL {
		urls = append(urls, actualURL)
	}
	return urls
}

// findByConditions executes the $or query.
func (r *MongoProjectRepository) findByConditions(ctx context.Context, conditions []bson.M) (*repository.FindOrCreateResult, error) {
	filter := bson.M{"$or": conditions}
	var doc bson.M
	err := r.projectsColl.FindOne(ctx, filter).Decode(&doc)

	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		r.logger.Error("failed to find project", slog.Any("error", err))
		return nil, fmt.Errorf("failed to find project: %w", err)
	}

	return r.docToResult(doc), nil
}

// createProject inserts a new project document.
func (r *MongoProjectRepository) createProject(ctx context.Context, params repository.FindOrCreateParams) (*repository.FindOrCreateResult, error) {
	// Prefer configured URL, fallback to actual URL
	remoteURL := params.ConfiguredURL
	if remoteURL == "" {
		remoteURL = params.ActualURL
	}

	newDoc := bson.M{
		mongoschema.ProjectRemoteURLField:    remoteURL,
		mongoschema.ProjectNameField:         params.Name,
		mongoschema.ProjectIsGitProjectField: params.IsGitProject,
		mongoschema.ProjectRegisteredAtField: time.Now(),
	}

	// Add organizationId field if provided
	if params.OrganizationID != "" {
		if orgID, err := bson.ObjectIDFromHex(params.OrganizationID); err == nil {
			newDoc[mongoschema.ProjectOrganizationIDField] = orgID
		}
	}

	result, err := r.projectsColl.InsertOne(ctx, newDoc)
	if err != nil {
		r.logger.Error("failed to create project", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	newID := result.InsertedID.(bson.ObjectID).Hex()
	r.logger.Info("created new project",
		slog.String("projectID", newID),
		slog.String("name", params.Name))

	return &repository.FindOrCreateResult{
		ProjectID:      newID,
		IsNew:          true,
		Name:           params.Name,
		IsGitProject:   params.IsGitProject,
		OrganizationID: params.OrganizationID,
	}, nil
}

// docToResult converts a MongoDB document to FindOrCreateResult.
func (r *MongoProjectRepository) docToResult(doc bson.M) *repository.FindOrCreateResult {
	result := &repository.FindOrCreateResult{
		ProjectID:    doc[mongoschema.ProjectIDField].(bson.ObjectID).Hex(),
		IsNew:        false,
		Name:         doc[mongoschema.ProjectNameField].(string),
		IsGitProject: doc[mongoschema.ProjectIsGitProjectField].(bool),
	}

	// Extract OrganizationID if present
	if orgID, ok := doc[mongoschema.ProjectOrganizationIDField].(bson.ObjectID); ok {
		result.OrganizationID = orgID.Hex()
	}

	return result
}

var _ repository.ProjectRepositoryPort = (*MongoProjectRepository)(nil)
