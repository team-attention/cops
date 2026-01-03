package mongodb

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/user/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoOrganizationRepository implements OrganizationRepositoryPort for MongoDB.
type MongoOrganizationRepository struct {
	logger  *slog.Logger
	orgColl *mongo.Collection
}

// NewMongoOrganizationRepository creates a new MongoDB organization repository.
func NewMongoOrganizationRepository(l *slog.Logger, db *mongo.Database) *MongoOrganizationRepository {
	return &MongoOrganizationRepository{
		logger:  l.With(slog.String("name", "user.repository.mongodb.organization")),
		orgColl: db.Collection(mongoschema.OrganizationCollectionName),
	}
}

// GetUserOrganizations retrieves all organizations a user belongs to with their roles.
func (r *MongoOrganizationRepository) GetUserOrganizations(ctx context.Context, userID string) ([]*repository.UserOrganization, error) {
	// 1. Convert userID string to bson.ObjectID.
	userObjectID, err := bson.ObjectIDFromHex(userID)
	// 2. If conversion fails, return nil, error.
	if err != nil {
		return nil, err
	}

	// 3. Build filter to find organizations where user is a member:
	filter := bson.M{
		"members": bson.M{
			"$elemMatch": bson.M{
				"userId": userObjectID,
			},
		},
	}

	// 4. Execute Find query on organizations collection.
	cursor, err := r.orgColl.Find(ctx, filter)
	if err != nil {
		r.logger.Error("failed to get user organizations",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}
	defer cursor.Close(ctx)

	// 5. Iterate cursor, decode each result to mongoschema.Organization.
	var result []*repository.UserOrganization
	for cursor.Next(ctx) {
		var orgSchema mongoschema.Organization
		if err := cursor.Decode(&orgSchema); err != nil {
			r.logger.Error("failed to decode organization",
				slog.String("userID", userID),
				slog.Any("error", err),
			)
			return nil, err
		}

		// 6. For each organization:
		// a. Convert to domain.Organization using ToDomain().
		org := orgSchema.ToDomain()

		// b. Find the user's membership entry in Members slice.
		// c. Extract the user's role from that entry.
		var userRole domain.MemberRole
		for _, member := range org.Members {
			if string(member.UserID) == userObjectID.Hex() {
				userRole = member.Role
				break
			}
		}

		// d. Create repository.UserOrganization with org and role.
		result = append(result, &repository.UserOrganization{
			Organization: org,
			Role:         userRole,
		})
	}

	if err := cursor.Err(); err != nil {
		r.logger.Error("cursor error while getting user organizations",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}

	// 7. Return slice of UserOrganization, nil.
	return result, nil
}

// Interface verification
var _ repository.OrganizationRepositoryPort = (*MongoOrganizationRepository)(nil)
