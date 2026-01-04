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
	logger             *slog.Logger
	projectsColl       *mongo.Collection
	sessionRecordsColl *mongo.Collection
}

// NewMongoCascadeDeleteRepository creates a new MongoDB cascade delete repository.
func NewMongoCascadeDeleteRepository(l *slog.Logger, db *mongo.Database) *MongoCascadeDeleteRepository {
	// 1. Return &MongoCascadeDeleteRepository with:
	//    - logger bound with name "user.repository.mongodb.cascade_delete"
	//    - projectsColl from db.Collection(mongoschema.ProjectCollectionName)
	//    - sessionRecordsColl from db.Collection(mongoschema.RecordCollectionName)
	return &MongoCascadeDeleteRepository{
		logger:             l.With(slog.String("name", "user.repository.mongodb.cascade_delete")),
		projectsColl:       db.Collection(mongoschema.ProjectCollectionName),
		sessionRecordsColl: db.Collection(mongoschema.RecordCollectionName),
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

// DeleteSessionRecordsByOrganization permanently deletes all session records for projects in an organization.
func (r *MongoCascadeDeleteRepository) DeleteSessionRecordsByOrganization(ctx context.Context, organizationID string) error {
	// 1. Convert organizationID string to bson.ObjectID.
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	// 2. If conversion fails, return error.
	if err != nil {
		return err
	}

	// 3. Query projects collection to get all project IDs for this organization.
	//    a. Create filter with organizationId field.
	filter := bson.M{mongoschema.ProjectOrganizationIDField: orgObjectID}
	//    b. Use Find with projection for _id only.
	cursor, err := r.projectsColl.Find(ctx, filter, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		r.logger.Error("failed to query projects for session record deletion",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return err
	}
	defer cursor.Close(ctx)

	//    c. Collect project IDs into slice of bson.ObjectID.
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

	// 4. If no projects found, return nil (nothing to delete).
	if len(projectIDs) == 0 {
		r.logger.Info("no projects found for organization, skipping session record deletion",
			slog.String("organizationID", organizationID),
		)
		return nil
	}

	// 5. Create filter for session records with projectId in the collected project IDs.
	recordsFilter := bson.M{mongoschema.RecordProjectIDField: bson.M{"$in": projectIDs}}

	// 6. Execute DeleteMany on sessionRecords collection.
	result, err := r.sessionRecordsColl.DeleteMany(ctx, recordsFilter)
	// 7. If error occurs, log error with organizationID and return error.
	if err != nil {
		r.logger.Error("failed to delete session records by organization",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return err
	}

	// 8. Log info with deleted count.
	r.logger.Info("deleted session records by organization",
		slog.String("organizationID", organizationID),
		slog.Int64("deletedCount", result.DeletedCount),
	)

	// 9. Return nil.
	return nil
}

// Interface verification
var _ repository.CascadeDeleteRepositoryPort = (*MongoCascadeDeleteRepository)(nil)
