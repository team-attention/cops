package mongoschema

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/team-attention/cops/shared/domain"
)

const (
	InvitationCollectionName = "invitations"
)

const (
	InvitationIDField             = "_id"
	InvitationOrganizationIDField = "organizationId"
	InvitationEmailField          = "email"
	InvitationTokenField          = "token"
	InvitationStatusField         = "status"
	InvitationInvitedByIDField    = "invitedById"
	InvitationCreatedAtField      = "createdAt"
	InvitationAcceptedAtField     = "acceptedAt"
)

// Invitation represents the MongoDB document structure.
type Invitation struct {
	ID             bson.ObjectID          `bson:"_id,omitempty"`
	OrganizationID bson.ObjectID          `bson:"organizationId"`
	Email          string                 `bson:"email"`
	Token          string                 `bson:"token"`
	Status         domain.InvitationStatus `bson:"status"`
	InvitedByID    bson.ObjectID          `bson:"invitedById"`
	CreatedAt      time.Time              `bson:"createdAt"`
	AcceptedAt     *time.Time             `bson:"acceptedAt,omitempty"`
}

// FromDomain converts domain Invitation to MongoDB schema.
func (s *Invitation) FromDomain(d *domain.Invitation) {
	if d == nil {
		return
	}

	s.Email = d.Email
	s.Token = d.Token
	s.Status = d.Status
	s.CreatedAt = d.CreatedAt
	s.AcceptedAt = d.AcceptedAt

	if d.ID != "" {
		s.ID, _ = bson.ObjectIDFromHex(string(d.ID))
	}

	if d.OrganizationID != "" {
		s.OrganizationID, _ = bson.ObjectIDFromHex(string(d.OrganizationID))
	}

	if d.InvitedByID != "" {
		s.InvitedByID, _ = bson.ObjectIDFromHex(string(d.InvitedByID))
	}
}

// ToDomain converts MongoDB schema to domain Invitation.
func (s *Invitation) ToDomain() *domain.Invitation {
	if s == nil {
		return nil
	}

	return &domain.Invitation{
		ID:             domain.ID(s.ID.Hex()),
		OrganizationID: domain.ID(s.OrganizationID.Hex()),
		Email:          s.Email,
		Token:          s.Token,
		Status:         s.Status,
		InvitedByID:    domain.ID(s.InvitedByID.Hex()),
		CreatedAt:      s.CreatedAt,
		AcceptedAt:     s.AcceptedAt,
	}
}
