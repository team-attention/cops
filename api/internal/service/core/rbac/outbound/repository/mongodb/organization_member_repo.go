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
	logger      *slog.Logger
	membersColl *mongo.Collection
}

// NewMongoOrganizationMemberRepository creates a new MongoDB organization member repository.
func NewMongoOrganizationMemberRepository(l *slog.Logger, db *mongo.Database) *MongoOrganizationMemberRepository {
	return &MongoOrganizationMemberRepository{
		logger:      l.With(slog.String("name", "rbac.repository.mongodb.organization_member")),
		membersColl: db.Collection(mongoschema.OrganizationMemberCollectionName),
	}
}

// IsMember checks if a user is a member of an organization.
func (r *MongoOrganizationMemberRepository) IsMember(ctx context.Context, userID, organizationID string) (bool, error) {
	userObjectID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return false, err
	}

	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return false, err
	}

	filter := bson.M{
		mongoschema.OrganizationMemberUserIDField:         userObjectID,
		mongoschema.OrganizationMemberOrganizationIDField: orgObjectID,
	}

	count, err := r.membersColl.CountDocuments(ctx, filter)
	if err != nil {
		r.logger.Error("failed to check organization membership",
			slog.String("userID", userID),
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return false, err
	}

	return count > 0, nil
}

// Interface verification
var _ repository.OrganizationMemberRepositoryPort = (*MongoOrganizationMemberRepository)(nil)
