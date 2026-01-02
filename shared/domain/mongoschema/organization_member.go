package mongoschema

import (
	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	OrganizationMemberCollectionName = "organization_members"
)

const (
	OrganizationMemberIDField             = "_id"
	OrganizationMemberOrganizationIDField = "organizationId"
	OrganizationMemberUserIDField         = "userId"
	OrganizationMemberRoleField           = "role"
)

type OrganizationMember struct {
	domain.OrganizationMember `bson:",inline"`
	ID                        bson.ObjectID `bson:"_id,omitempty"`
	OrganizationID            bson.ObjectID `bson:"organizationId"`
	UserID                    bson.ObjectID `bson:"userId"`
}

func (s *OrganizationMember) FromDomain(d *domain.OrganizationMember) {
	if d == nil {
		return
	}

	s.OrganizationMember = *d

	if d.ID != "" {
		s.ID, _ = bson.ObjectIDFromHex(string(d.ID))
	}

	if d.OrganizationID != "" {
		s.OrganizationID, _ = bson.ObjectIDFromHex(string(d.OrganizationID))
	}

	if d.UserID != "" {
		s.UserID, _ = bson.ObjectIDFromHex(string(d.UserID))
	}
}

func (s *OrganizationMember) ToDomain() *domain.OrganizationMember {
	if s == nil {
		return nil
	}

	s.OrganizationMember.ID = domain.ID(s.ID.Hex())
	s.OrganizationMember.OrganizationID = domain.ID(s.OrganizationID.Hex())
	s.OrganizationMember.UserID = domain.ID(s.UserID.Hex())

	return &s.OrganizationMember
}
