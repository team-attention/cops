package mongodb

import (
	"context"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/team-attention/cops/api/internal/service/organization/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoOrganizationRepository implements OrganizationRepositoryPort for MongoDB.
type MongoOrganizationRepository struct {
	logger    *slog.Logger
	orgColl   *mongo.Collection
	usersColl *mongo.Collection
}

// NewMongoOrganizationRepository creates a new MongoDB organization repository.
func NewMongoOrganizationRepository(l *slog.Logger, db *mongo.Database) *MongoOrganizationRepository {
	return &MongoOrganizationRepository{
		logger:    l.With(slog.String("name", "organization.repository.mongodb")),
		orgColl:   db.Collection(mongoschema.OrganizationCollectionName),
		usersColl: db.Collection(mongoschema.UserCollectionName),
	}
}

// Create persists a new organization.
func (r *MongoOrganizationRepository) Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
	var orgSchema mongoschema.Organization
	orgSchema.FromDomain(org)

	result, err := r.orgColl.InsertOne(ctx, orgSchema)
	if err != nil {
		r.logger.Error("failed to create organization",
			slog.String("name", org.Name),
			slog.String("slug", org.Slug),
			slog.Any("error", err),
		)
		return nil, err
	}

	orgSchema.ID = result.InsertedID.(bson.ObjectID)
	return orgSchema.ToDomain(), nil
}

// GetByID retrieves an organization by its ID.
func (r *MongoOrganizationRepository) GetByID(ctx context.Context, organizationID string) (*domain.Organization, error) {
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{mongoschema.OrganizationIDField: orgObjectID}

	var orgSchema mongoschema.Organization
	err = r.orgColl.FindOne(ctx, filter).Decode(&orgSchema)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.logger.Error("failed to get organization by ID",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return orgSchema.ToDomain(), nil
}

// GetBySlug retrieves an organization by its slug.
func (r *MongoOrganizationRepository) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	filter := bson.M{mongoschema.OrganizationSlugField: slug}

	var orgSchema mongoschema.Organization
	err := r.orgColl.FindOne(ctx, filter).Decode(&orgSchema)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.logger.Error("failed to get organization by slug",
			slog.String("slug", slug),
			slog.Any("error", err),
		)
		return nil, err
	}

	return orgSchema.ToDomain(), nil
}

// Update updates an organization's name and slug.
func (r *MongoOrganizationRepository) Update(ctx context.Context, organizationID, name, slug string) (*domain.Organization, error) {
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{mongoschema.OrganizationIDField: orgObjectID}
	update := bson.M{
		"$set": bson.M{
			mongoschema.OrganizationNameField: name,
			mongoschema.OrganizationSlugField: slug,
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var orgSchema mongoschema.Organization
	err = r.orgColl.FindOneAndUpdate(ctx, filter, update, opts).Decode(&orgSchema)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("organization not found")
		}
		r.logger.Error("failed to update organization",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return orgSchema.ToDomain(), nil
}

// GetMembersWithDetails retrieves all members with their user details.
func (r *MongoOrganizationRepository) GetMembersWithDetails(ctx context.Context, organizationID string) ([]*repository.MemberWithDetails, error) {
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return nil, err
	}

	pipeline := []bson.M{
		// Match organization by ID
		{
			"$match": bson.M{
				mongoschema.OrganizationIDField: orgObjectID,
			},
		},
		// Unwind members array
		{
			"$unwind": bson.M{
				"path":                       "$" + mongoschema.OrganizationMembersField,
				"preserveNullAndEmptyArrays": false,
			},
		},
		// Lookup user details
		{
			"$lookup": bson.M{
				"from":         mongoschema.UserCollectionName,
				"localField":   mongoschema.OrganizationMembersField + "." + mongoschema.OrganizationMemberUserIDField,
				"foreignField": mongoschema.UserIDField,
				"as":           "userDetails",
			},
		},
		// Unwind userDetails array (should only have one element)
		{
			"$unwind": bson.M{
				"path":                       "$userDetails",
				"preserveNullAndEmptyArrays": true,
			},
		},
		// Project fields
		{
			"$project": bson.M{
				"userId":    "$" + mongoschema.OrganizationMembersField + "." + mongoschema.OrganizationMemberUserIDField,
				"role":      "$" + mongoschema.OrganizationMembersField + "." + mongoschema.OrganizationMemberRoleField,
				"email":     "$userDetails." + mongoschema.UserEmailField,
				"name":      "$userDetails." + mongoschema.UserNameField,
				"avatarUrl": "$userDetails." + mongoschema.UserProfileImageURLField,
			},
		},
	}

	cursor, err := r.orgColl.Aggregate(ctx, pipeline)
	if err != nil {
		r.logger.Error("failed to get members with details",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return nil, err
	}
	defer cursor.Close(ctx)

	var members []*repository.MemberWithDetails
	for cursor.Next(ctx) {
		var result struct {
			UserID    bson.ObjectID     `bson:"userId"`
			Role      domain.MemberRole `bson:"role"`
			Email     string            `bson:"email"`
			Name      string            `bson:"name"`
			AvatarURL string            `bson:"avatarUrl"`
		}
		if err := cursor.Decode(&result); err != nil {
			r.logger.Error("failed to decode member details",
				slog.String("organizationID", organizationID),
				slog.Any("error", err),
			)
			return nil, err
		}

		members = append(members, &repository.MemberWithDetails{
			UserID:    result.UserID.Hex(),
			Email:     result.Email,
			Name:      result.Name,
			AvatarURL: result.AvatarURL,
			Role:      result.Role,
		})
	}

	if err := cursor.Err(); err != nil {
		r.logger.Error("cursor error while getting members with details",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return nil, err
	}

	if members == nil {
		members = []*repository.MemberWithDetails{}
	}

	return members, nil
}

// UpdateMemberRole updates a member's role in the organization.
func (r *MongoOrganizationRepository) UpdateMemberRole(ctx context.Context, organizationID, userID string, role domain.MemberRole) error {
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return err
	}

	userObjectID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	filter := bson.M{
		mongoschema.OrganizationIDField: orgObjectID,
		mongoschema.OrganizationMembersField + "." + mongoschema.OrganizationMemberUserIDField: userObjectID,
	}

	update := bson.M{
		"$set": bson.M{
			mongoschema.OrganizationMembersField + ".$." + mongoschema.OrganizationMemberRoleField: role,
		},
	}

	_, err = r.orgColl.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("failed to update member role",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
			slog.String("role", string(role)),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

// RemoveMember removes a member from the organization.
func (r *MongoOrganizationRepository) RemoveMember(ctx context.Context, organizationID, userID string) error {
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return err
	}

	userObjectID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	filter := bson.M{mongoschema.OrganizationIDField: orgObjectID}

	update := bson.M{
		"$pull": bson.M{
			mongoschema.OrganizationMembersField: bson.M{
				mongoschema.OrganizationMemberUserIDField: userObjectID,
			},
		},
	}

	_, err = r.orgColl.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("failed to remove member",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

// CountAdmins returns the number of admin members in an organization.
func (r *MongoOrganizationRepository) CountAdmins(ctx context.Context, organizationID string) (int, error) {
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return 0, err
	}

	pipeline := []bson.M{
		{
			"$match": bson.M{
				mongoschema.OrganizationIDField: orgObjectID,
			},
		},
		{
			"$unwind": "$" + mongoschema.OrganizationMembersField,
		},
		{
			"$match": bson.M{
				mongoschema.OrganizationMembersField + "." + mongoschema.OrganizationMemberRoleField: domain.MemberRoleAdmin,
			},
		},
		{
			"$count": "count",
		},
	}

	cursor, err := r.orgColl.Aggregate(ctx, pipeline)
	if err != nil {
		r.logger.Error("failed to count admins",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return 0, err
	}
	defer cursor.Close(ctx)

	var result struct {
		Count int `bson:"count"`
	}
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, err
		}
		return result.Count, nil
	}

	// No admins found
	return 0, nil
}

// GetUserOrganizationCount returns the number of organizations a user belongs to.
func (r *MongoOrganizationRepository) GetUserOrganizationCount(ctx context.Context, userID string) (int, error) {
	userObjectID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return 0, err
	}

	filter := bson.M{
		mongoschema.OrganizationMembersField: bson.M{
			"$elemMatch": bson.M{
				mongoschema.OrganizationMemberUserIDField: userObjectID,
			},
		},
	}

	count, err := r.orgColl.CountDocuments(ctx, filter)
	if err != nil {
		r.logger.Error("failed to get user organization count",
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return 0, err
	}

	return int(count), nil
}

// GetMemberRole retrieves a user's role in an organization.
func (r *MongoOrganizationRepository) GetMemberRole(ctx context.Context, organizationID, userID string) (domain.MemberRole, error) {
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return "", err
	}

	userObjectID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return "", err
	}

	filter := bson.M{
		mongoschema.OrganizationIDField: orgObjectID,
		mongoschema.OrganizationMembersField: bson.M{
			"$elemMatch": bson.M{
				mongoschema.OrganizationMemberUserIDField: userObjectID,
			},
		},
	}

	projection := bson.M{
		mongoschema.OrganizationMembersField: bson.M{
			"$elemMatch": bson.M{
				mongoschema.OrganizationMemberUserIDField: userObjectID,
			},
		},
	}

	var result struct {
		Members []*mongoschema.OrganizationMember `bson:"members"`
	}

	err = r.orgColl.FindOne(ctx, filter, options.FindOne().SetProjection(projection)).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", nil
		}
		r.logger.Error("failed to get member role",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return "", err
	}

	if len(result.Members) == 0 {
		return "", nil
	}

	return result.Members[0].Role, nil
}

// DeleteOrganization permanently deletes an organization.
func (r *MongoOrganizationRepository) DeleteOrganization(ctx context.Context, organizationID string) error {
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return err
	}

	filter := bson.M{mongoschema.OrganizationIDField: orgObjectID}

	_, err = r.orgColl.DeleteOne(ctx, filter)
	if err != nil {
		r.logger.Error("failed to delete organization",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

// AddMember adds a new member to an organization.
func (r *MongoOrganizationRepository) AddMember(ctx context.Context, organizationID, userID string, role domain.MemberRole) error {
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return err
	}

	userObjectID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	filter := bson.M{mongoschema.OrganizationIDField: orgObjectID}

	update := bson.M{
		"$push": bson.M{
			mongoschema.OrganizationMembersField: bson.M{
				mongoschema.OrganizationMemberUserIDField: userObjectID,
				mongoschema.OrganizationMemberRoleField:   role,
			},
		},
	}

	_, err = r.orgColl.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("failed to add member",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
			slog.String("role", string(role)),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

// CheckMemberExists checks if a user is already a member.
func (r *MongoOrganizationRepository) CheckMemberExists(ctx context.Context, organizationID, userID string) (bool, error) {
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return false, err
	}

	userObjectID, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return false, err
	}

	filter := bson.M{
		mongoschema.OrganizationIDField: orgObjectID,
		mongoschema.OrganizationMembersField: bson.M{
			"$elemMatch": bson.M{
				mongoschema.OrganizationMemberUserIDField: userObjectID,
			},
		},
	}

	count, err := r.orgColl.CountDocuments(ctx, filter)
	if err != nil {
		r.logger.Error("failed to check member exists",
			slog.String("organizationID", organizationID),
			slog.String("userID", userID),
			slog.Any("error", err),
		)
		return false, err
	}

	return count > 0, nil
}

// Interface verification
var _ repository.OrganizationRepositoryPort = (*MongoOrganizationRepository)(nil)
