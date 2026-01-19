package mongodb

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/team-attention/cops/api/internal/service/invitation/outbound/repository"
	"github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
)

// MongoInvitationRepository implements InvitationRepositoryPort.
type MongoInvitationRepository struct {
	logger *slog.Logger
	coll   *mongo.Collection
}

// NewMongoInvitationRepository creates a new MongoDB invitation repository.
func NewMongoInvitationRepository(l *slog.Logger, db *mongo.Database) *MongoInvitationRepository {
	return &MongoInvitationRepository{
		logger: l.With(slog.String("name", "invitation.repository.mongodb")),
		coll:   db.Collection(mongoschema.InvitationCollectionName),
	}
}

// Create persists a new invitation.
func (r *MongoInvitationRepository) Create(ctx context.Context, invitation *domain.Invitation) (*domain.Invitation, error) {
	var schema mongoschema.Invitation
	schema.FromDomain(invitation)

	result, err := r.coll.InsertOne(ctx, schema)
	if err != nil {
		r.logger.Error("failed to create invitation",
			slog.String("email", invitation.Email),
			slog.Any("error", err),
		)
		return nil, err
	}

	schema.ID = result.InsertedID.(bson.ObjectID)
	return schema.ToDomain(), nil
}

// GetByToken retrieves an invitation by its secure token.
func (r *MongoInvitationRepository) GetByToken(ctx context.Context, token string) (*domain.Invitation, error) {
	filter := bson.M{mongoschema.InvitationTokenField: token}

	var schema mongoschema.Invitation
	err := r.coll.FindOne(ctx, filter).Decode(&schema)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.logger.Error("failed to get invitation by token",
			slog.Any("error", err),
		)
		return nil, err
	}

	return schema.ToDomain(), nil
}

// GetByID retrieves an invitation by ID.
func (r *MongoInvitationRepository) GetByID(ctx context.Context, id string) (*domain.Invitation, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	filter := bson.M{mongoschema.InvitationIDField: objectID}

	var schema mongoschema.Invitation
	err = r.coll.FindOne(ctx, filter).Decode(&schema)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.logger.Error("failed to get invitation by ID",
			slog.String("id", id),
			slog.Any("error", err),
		)
		return nil, err
	}

	return schema.ToDomain(), nil
}

// GetByEmailAndOrg checks if an invitation already exists for email in org.
func (r *MongoInvitationRepository) GetByEmailAndOrg(ctx context.Context, email, organizationID string) (*domain.Invitation, error) {
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return nil, err
	}

	// Normalize email to lowercase
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	filter := bson.M{
		mongoschema.InvitationEmailField:          normalizedEmail,
		mongoschema.InvitationOrganizationIDField: orgObjectID,
		mongoschema.InvitationStatusField:         domain.InvitationStatusPending,
	}

	var schema mongoschema.Invitation
	err = r.coll.FindOne(ctx, filter).Decode(&schema)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		r.logger.Error("failed to get invitation by email and org",
			slog.String("email", normalizedEmail),
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return nil, err
	}

	return schema.ToDomain(), nil
}

// ListByOrganization retrieves all pending invitations for an organization.
func (r *MongoInvitationRepository) ListByOrganization(ctx context.Context, organizationID string) ([]*domain.Invitation, error) {
	orgObjectID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		mongoschema.InvitationOrganizationIDField: orgObjectID,
		mongoschema.InvitationStatusField:         domain.InvitationStatusPending,
	}

	opts := options.Find().SetSort(bson.M{mongoschema.InvitationCreatedAtField: -1})

	cursor, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		r.logger.Error("failed to list invitations by organization",
			slog.String("organizationID", organizationID),
			slog.Any("error", err),
		)
		return nil, err
	}
	defer cursor.Close(ctx)

	var invitations []*domain.Invitation
	for cursor.Next(ctx) {
		var schema mongoschema.Invitation
		if err := cursor.Decode(&schema); err != nil {
			r.logger.Error("failed to decode invitation",
				slog.Any("error", err),
			)
			return nil, err
		}
		invitations = append(invitations, schema.ToDomain())
	}

	if err := cursor.Err(); err != nil {
		r.logger.Error("cursor error while listing invitations",
			slog.Any("error", err),
		)
		return nil, err
	}

	if invitations == nil {
		invitations = []*domain.Invitation{}
	}

	return invitations, nil
}

// UpdateStatus updates an invitation's status.
func (r *MongoInvitationRepository) UpdateStatus(ctx context.Context, id string, status domain.InvitationStatus) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{mongoschema.InvitationIDField: objectID}

	update := bson.M{
		"$set": bson.M{
			mongoschema.InvitationStatusField: status,
		},
	}

	// If status is "accepted", also set acceptedAt
	if status == domain.InvitationStatusAccepted {
		now := time.Now()
		update["$set"].(bson.M)[mongoschema.InvitationAcceptedAtField] = now
	}

	_, err = r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("failed to update invitation status",
			slog.String("id", id),
			slog.String("status", string(status)),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

// Delete removes an invitation permanently.
func (r *MongoInvitationRepository) Delete(ctx context.Context, id string) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{mongoschema.InvitationIDField: objectID}

	_, err = r.coll.DeleteOne(ctx, filter)
	if err != nil {
		r.logger.Error("failed to delete invitation",
			slog.String("id", id),
			slog.Any("error", err),
		)
		return err
	}

	return nil
}

// Interface verification
var _ repository.InvitationRepositoryPort = (*MongoInvitationRepository)(nil)
