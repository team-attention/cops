package mongodb

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoOrganizationMemberRepository implements OrganizationMemberRepositoryPort for MongoDB.
type MongoOrganizationMemberRepository struct {
	logger  *slog.Logger
	orgColl *mongo.Collection
}

// NewMongoOrganizationMemberRepository creates a new MongoDB organization member repository.
func NewMongoOrganizationMemberRepository(l *slog.Logger, db *mongo.Database) *MongoOrganizationMemberRepository {
	return &MongoOrganizationMemberRepository{
		logger:  l.With(slog.String("name", "rbac.repository.mongodb.organization_member")),
		orgColl: db.Collection(mongoschema.OrganizationCollectionName),
	}
}

// IsMember checks if a user is a member of an organization.
func (r *MongoOrganizationMemberRepository) IsMember(ctx context.Context, userID, organizationID string) (bool, error) {
	// 1. Convert userID string to bson.ObjectID.
	userObjectID, err := bson.ObjectIDFromHex(userID)
	// 2. If conversion fails, return false, error.
	if err != nil {
		return false, err
	}

	// 3. Convert organizationID string to bson.ObjectID.
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	// 4. If conversion fails, return false, error.
	if err != nil {
		return false, err
	}

	// 5. Build filter to find organization with specific ID and user in members:
	filter := bson.M{
		"_id": orgObjectID,
		"members": bson.M{
			"$elemMatch": bson.M{
				"userId": userObjectID,
			},
		},
	}

	// 6. Execute CountDocuments query on organizations collection.
	count, err := r.orgColl.CountDocuments(ctx, filter)
	// 7. If error, log and return false, error.
	if err != nil {
		r.logger.Error("failed to check organization membership",
			slog.String("userID", userID),
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return false, err
	}

	// 8. Return count > 0, nil.
	return count > 0, nil
}

// Interface verification
var _ repository.OrganizationMemberRepositoryPort = (*MongoOrganizationMemberRepository)(nil)
