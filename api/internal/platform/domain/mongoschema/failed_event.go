package mongoschema

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/team-attention/cops/api/internal/platform/domain"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

// FailedEventCollectionName is the MongoDB collection name for failed events.
const FailedEventCollectionName = "failed_events"

// Field name constants for query building.
const (
	FailedEventIDField             = "_id"
	FailedEventStatusField         = "status"
	FailedEventRetryCountField     = "retryCount"
	FailedEventFailureReasonField  = "failureReason"
	FailedEventRawLineField        = "rawLine"
	FailedEventProjectIDField      = "projectId"
	FailedEventOrganizationIDField = "organizationId"
	FailedEventUserIDField         = "userId"
	FailedEventCreatedAtField      = "createdAt"
	FailedEventLastRetryAtField    = "lastRetryAt"
)

// FailedEvent is the MongoDB schema for platform/domain.FailedEvent.
// Embeds domain model with bson:",inline" tag following the standard pattern.
type FailedEvent struct {
	domain.FailedEvent `bson:",inline"`
	// ID is the MongoDB ObjectID for this document.
	ID bson.ObjectID `bson:"_id,omitempty"`
	// ProjectID is the MongoDB ObjectID reference to the project.
	ProjectID bson.ObjectID `bson:"projectId"`
	// OrganizationID is the MongoDB ObjectID reference to the organization.
	OrganizationID bson.ObjectID `bson:"organizationId"`
	// UserID is the MongoDB ObjectID reference to the user.
	UserID bson.ObjectID `bson:"userId"`
	// CreatedAt is schema-only metadata for when the failed event was created.
	CreatedAt time.Time `bson:"createdAt"`
	// LastRetryAt is schema-only metadata for when the last retry attempt occurred.
	LastRetryAt *time.Time `bson:"lastRetryAt,omitempty"`
}

// FromDomain converts a domain.FailedEvent to a mongoschema.FailedEvent.
func (s *FailedEvent) FromDomain(d *domain.FailedEvent) {
	if d == nil {
		return
	}

	s.FailedEvent = *d

	if d.ID != "" {
		s.ID, _ = bson.ObjectIDFromHex(string(d.ID))
	}

	if d.ProjectID != "" {
		s.ProjectID, _ = bson.ObjectIDFromHex(string(d.ProjectID))
	}

	if d.OrganizationID != "" {
		s.OrganizationID, _ = bson.ObjectIDFromHex(string(d.OrganizationID))
	}

	if d.UserID != "" {
		s.UserID, _ = bson.ObjectIDFromHex(string(d.UserID))
	}
}

// ToDomain converts a mongoschema.FailedEvent to a domain.FailedEvent.
func (s *FailedEvent) ToDomain() *domain.FailedEvent {
	if s == nil {
		return nil
	}

	s.FailedEvent.ID = shareddomain.ID(s.ID.Hex())
	s.FailedEvent.ProjectID = shareddomain.ID(s.ProjectID.Hex())
	s.FailedEvent.OrganizationID = shareddomain.ID(s.OrganizationID.Hex())
	s.FailedEvent.UserID = shareddomain.ID(s.UserID.Hex())

	return &s.FailedEvent
}
