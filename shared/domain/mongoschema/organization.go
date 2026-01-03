package mongoschema

import (
	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	OrganizationCollectionName = "organizations"
)

const (
	OrganizationIDField      = "_id"
	OrganizationNameField    = "name"
	OrganizationSlugField    = "slug"
	OrganizationMembersField = "members"
)

// OrganizationMember field constants for nested queries.
const (
	OrganizationMemberUserIDField = "userId"
	OrganizationMemberRoleField   = "role"
)

// OrganizationMember represents a member entry within an organization document.
type OrganizationMember struct {
	UserID bson.ObjectID     `bson:"userId"`
	Role   domain.MemberRole `bson:"role"`
}

// ToDomain converts schema OrganizationMember to domain OrganizationMember.
func (s *OrganizationMember) ToDomain() *domain.OrganizationMember {
	if s == nil {
		return nil
	}
	return &domain.OrganizationMember{
		UserID: domain.ID(s.UserID.Hex()),
		Role:   s.Role,
	}
}

// FromDomain converts domain OrganizationMember to schema OrganizationMember.
func (s *OrganizationMember) FromDomain(d *domain.OrganizationMember) {
	if d == nil {
		return
	}
	if d.UserID != "" {
		s.UserID, _ = bson.ObjectIDFromHex(string(d.UserID))
	}
	s.Role = d.Role
}

type Organization struct {
	ID      bson.ObjectID         `bson:"_id,omitempty"`
	Name    string                `bson:"name"`
	Slug    string                `bson:"slug"`
	Members []*OrganizationMember `bson:"members"`
}

func (s *Organization) FromDomain(d *domain.Organization) {
	if d == nil {
		return
	}

	s.Name = d.Name
	s.Slug = d.Slug

	if d.ID != "" {
		s.ID, _ = bson.ObjectIDFromHex(string(d.ID))
	}

	// Convert organization members
	if d.Members != nil {
		s.Members = make([]*OrganizationMember, len(d.Members))
		for i, m := range d.Members {
			s.Members[i] = &OrganizationMember{}
			s.Members[i].FromDomain(m)
		}
	}
}

func (s *Organization) ToDomain() *domain.Organization {
	if s == nil {
		return nil
	}

	org := &domain.Organization{
		ID:   domain.ID(s.ID.Hex()),
		Name: s.Name,
		Slug: s.Slug,
	}

	// Convert organization members
	if s.Members != nil {
		org.Members = make([]*domain.OrganizationMember, len(s.Members))
		for i, m := range s.Members {
			org.Members[i] = m.ToDomain()
		}
	}

	return org
}
