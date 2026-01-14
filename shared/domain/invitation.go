package domain

import "time"

// InvitationStatus represents the current state of an invitation.
type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusRevoked  InvitationStatus = "revoked"
)

// Invitation represents a pending member invitation to an organization.
type Invitation struct {
	ID             ID               `json:"id" bson:"-"`
	OrganizationID ID               `json:"organizationId" bson:"-"`
	Email          string           `json:"email" bson:"email"`
	Token          string           `json:"-" bson:"-"`
	Status         InvitationStatus `json:"status" bson:"status"`
	InvitedByID    ID               `json:"invitedById" bson:"-"`
	CreatedAt      time.Time        `json:"createdAt" bson:"createdAt"`
	AcceptedAt     *time.Time       `json:"acceptedAt,omitempty" bson:"acceptedAt,omitempty"`
}
