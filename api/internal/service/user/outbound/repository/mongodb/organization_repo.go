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

// GetUserOrganizationsWithMemberCount retrieves all organizations a user belongs to with member counts.
func (r *MongoOrganizationRepository) GetUserOrganizationsWithMemberCount(ctx context.Context, userID string) ([]*repository.OrganizationWithMemberCount, error) {
	// 1. Convert userID string to bson.ObjectID.
	userObjectID, err := bson.ObjectIDFromHex(userID)
	// 2. If conversion fails, return nil, error.
	if err != nil {
		return nil, err
	}

	// 3. Build filter to find organizations where user is a member using $elemMatch.
	filter := bson.M{
		"members": bson.M{
			"$elemMatch": bson.M{
				"userId": userObjectID,
			},
		},
	}

	// 4. Execute Find query on organizations collection.
	cursor, err := r.orgColl.Find(ctx, filter)
	// 5. If error, log and return nil, error.
	if err != nil {
		r.logger.Error("failed to get user organizations with member count",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}
	defer cursor.Close(ctx)

	// 6. Iterate cursor, decode each result to mongoschema.Organization.
	var result []*repository.OrganizationWithMemberCount
	for cursor.Next(ctx) {
		var orgSchema mongoschema.Organization
		if err := cursor.Decode(&orgSchema); err != nil {
			r.logger.Error("failed to decode organization",
				slog.String("userID", userID),
				slog.Any("error", err),
			)
			return nil, err
		}

		// 7. For each organization:
		// a. Convert to domain.Organization using ToDomain().
		org := orgSchema.ToDomain()
		// b. Count members in Members slice.
		memberCount := len(org.Members)
		// c. Create OrganizationWithMemberCount with org and count.
		result = append(result, &repository.OrganizationWithMemberCount{
			Organization: org,
			MemberCount:  memberCount,
		})
	}

	if err := cursor.Err(); err != nil {
		r.logger.Error("cursor error while getting user organizations with member count",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return nil, err
	}

	// 8. Return slice of OrganizationWithMemberCount, nil.
	return result, nil
}

// RemoveUserFromOrganization removes a user from an organization's members array.
func (r *MongoOrganizationRepository) RemoveUserFromOrganization(ctx context.Context, organizationID, userID string) error {
	// 1. Convert organizationID string to bson.ObjectID.
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	// 2. If conversion fails, return error.
	if err != nil {
		return err
	}

	// 3. Convert userID string to bson.ObjectID.
	userObjectID, err := bson.ObjectIDFromHex(userID)
	// 4. If conversion fails, return error.
	if err != nil {
		return err
	}

	// 5. Create filter with organization _id.
	filter := bson.M{mongoschema.OrganizationIDField: orgObjectID}

	// 6. Create update using $pull to remove member with matching userId from members array.
	update := bson.M{
		"$pull": bson.M{
			mongoschema.OrganizationMembersField: bson.M{
				mongoschema.OrganizationMemberUserIDField: userObjectID,
			},
		},
	}

	// 7. Execute UpdateOne on organizations collection.
	_, err = r.orgColl.UpdateOne(ctx, filter, update)
	// 8. If error occurs, log error and return error.
	if err != nil {
		r.logger.Error("failed to remove user from organization",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return err
	}

	// 9. Return nil (success - idempotent operation).
	return nil
}

// DeleteOrganization permanently deletes an organization by ID.
func (r *MongoOrganizationRepository) DeleteOrganization(ctx context.Context, organizationID string) error {
	// 1. Convert organizationID string to bson.ObjectID.
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	// 2. If conversion fails, return error.
	if err != nil {
		return err
	}

	// 3. Create filter with _id field.
	filter := bson.M{mongoschema.OrganizationIDField: orgObjectID}

	// 4. Execute DeleteOne on organizations collection.
	_, err = r.orgColl.DeleteOne(ctx, filter)
	// 5. If error occurs, log error and return error.
	if err != nil {
		r.logger.Error("failed to delete organization",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return err
	}

	// 6. Return nil (success - idempotent operation).
	return nil
}

// Interface verification
var _ repository.OrganizationRepositoryPort = (*MongoOrganizationRepository)(nil)
