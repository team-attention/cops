package mongoschema

import (
	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)


// Event field constants for MongoDB queries.
// Note: EventCollectionName and EventProjectIDField are in transcript.go for compatibility.
const (
	EventIDField             = "_id"
	EventDataField           = "data"
	EventVersionField        = "version"
	EventProviderField       = "provider"
	EventOrganizationIDField = "organizationId"
	EventUserIDField         = "userId"
	EventBatchIDField        = "batchId"
	EventRetryCountField     = "retryCount"
	EventStatusField         = "status"
)

// Event is the MongoDB schema for domain.Event.
type Event struct {
	domain.Event   `bson:",inline"`
	ID             bson.ObjectID `bson:"_id,omitempty"`
	ProjectID      bson.ObjectID `bson:"projectId"`
	OrganizationID bson.ObjectID `bson:"organizationId"`
	UserID         bson.ObjectID `bson:"userId"`
}

// FromDomain converts domain.Event to MongoDB schema.
func (s *Event) FromDomain(d *domain.Event) {
	if d == nil {
		return
	}

	s.Event = *d

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

// ToDomain converts MongoDB schema to domain.Event.
func (s *Event) ToDomain() *domain.Event {
	if s == nil {
		return nil
	}

	s.Event.ID = domain.ID(s.ID.Hex())
	s.Event.ProjectID = domain.ID(s.ProjectID.Hex())
	s.Event.OrganizationID = domain.ID(s.OrganizationID.Hex())
	s.Event.UserID = domain.ID(s.UserID.Hex())
	s.Event.Data = ConvertBsonToMap(s.Event.Data)

	return &s.Event
}
