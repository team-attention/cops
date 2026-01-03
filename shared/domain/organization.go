package domain

// MemberRole represents the role of a user within an organization.
type MemberRole string

const (
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

// OrganizationMember represents a member entry within an organization.
type OrganizationMember struct {
	UserID ID         `json:"userId" bson:"userId"`
	Role   MemberRole `json:"role" bson:"role"`
}

// Organization represents a group that owns projects.
// Members are embedded as an array within the organization document.
type Organization struct {
	ID      ID                    `json:"id" bson:"-"`
	Name    string                `json:"name" bson:"name"`
	Slug    string                `json:"slug" bson:"slug"`
	Members []*OrganizationMember `json:"members" bson:"members"`
}
