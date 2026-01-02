package mongoschema

import (
	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	OrganizationCollectionName = "organizations"
)

const (
	OrganizationIDField   = "_id"
	OrganizationNameField = "name"
	OrganizationSlugField = "slug"
)

type Organization struct {
	domain.Organization `bson:",inline"`
	ID                  bson.ObjectID `bson:"_id,omitempty"`
}

func (s *Organization) FromDomain(d *domain.Organization) {
	if d == nil {
		return
	}

	s.Organization = *d

	if d.ID != "" {
		s.ID, _ = bson.ObjectIDFromHex(string(d.ID))
	}
}

func (s *Organization) ToDomain() *domain.Organization {
	if s == nil {
		return nil
	}

	s.Organization.ID = domain.ID(s.ID.Hex())

	return &s.Organization
}
