package mongodb

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/team-attention/cops/api/internal/service/user/outbound/repository"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoCascadeDeleteRepository implements CascadeDeleteRepositoryPort for MongoDB.
type MongoCascadeDeleteRepository struct {
	logger       *slog.Logger
	projectsColl *mongo.Collection
	eventsColl   *mongo.Collection
}

// NewMongoCascadeDeleteRepository creates a new MongoDB cascade delete repository.
func NewMongoCascadeDeleteRepository(l *slog.Logger, db *mongo.Database) *MongoCascadeDeleteRepository {
	return &MongoCascadeDeleteRepository{
		logger:       l.With(slog.String("name", "user.repository.mongodb.cascade_delete")),
		projectsColl: db.Collection(mongoschema.ProjectCollectionName),
		eventsColl:   db.Collection(mongoschema.EventCollectionName),
	}
}

// DeleteProjectsByOrganization permanently deletes all projects for an organization.
func (r *MongoCascadeDeleteRepository) DeleteProjectsByOrganization(ctx context.Context, organizationID string) error {
	// 1. Convert organizationID string to bson.ObjectID.
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	// 2. If conversion fails, return error.
	if err != nil {
		return err
	}

	// 3. Create filter with organizationId field matching the ObjectID.
	filter := bson.M{mongoschema.ProjectOrganizationIDField: orgObjectID}

	// 4. Execute DeleteMany on projects collection.
	result, err := r.projectsColl.DeleteMany(ctx, filter)
	// 5. If error occurs, log error with organizationID and return error.
	if err != nil {
		r.logger.Error("failed to delete projects by organization",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return err
	}

	// 6. Log info with deleted count.
	r.logger.Info("deleted projects by organization",
		slog.String("organizationID", organizationID),
		slog.Int64("deletedCount", result.DeletedCount),
	)

	// 7. Return nil.
	return nil
}

// DeleteEventsByOrganization permanently deletes all events for projects in an organization.
func (r *MongoCascadeDeleteRepository) DeleteEventsByOrganization(ctx context.Context, organizationID string) error {
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return err
	}

	filter := bson.M{mongoschema.ProjectOrganizationIDField: orgObjectID}
	cursor, err := r.projectsColl.Find(ctx, filter, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		r.logger.Error("failed to query projects for events deletion",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return err
	}
	defer cursor.Close(ctx)

	var projectIDs []bson.ObjectID
	for cursor.Next(ctx) {
		var doc struct {
			ID bson.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		projectIDs = append(projectIDs, doc.ID)
	}

	if len(projectIDs) == 0 {
		r.logger.Info("no projects found for organization, skipping events deletion",
			slog.String("organizationID", organizationID),
		)
		return nil
	}

	eventsFilter := bson.M{mongoschema.EventProjectIDField: bson.M{"$in": projectIDs}}

	result, err := r.eventsColl.DeleteMany(ctx, eventsFilter)
	if err != nil {
		r.logger.Error("failed to delete events by organization",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return err
	}

	r.logger.Info("deleted events by organization",
		slog.String("organizationID", organizationID),
		slog.Int64("deletedCount", result.DeletedCount),
	)

	return nil
}

// Interface verification
var _ repository.CascadeDeleteRepositoryPort = (*MongoCascadeDeleteRepository)(nil)
