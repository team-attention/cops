package domain

// MemberRole represents the role of a user within an organization.
type MemberRole string

const (
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

// Organization represents a group that owns projects.
type Organization struct {
	ID   ID     `json:"id" bson:"-"`
	Name string `json:"name" bson:"name"`
	Slug string `json:"slug" bson:"slug"`
}

// OrganizationMember represents membership relationship between user and organization.
type OrganizationMember struct {
	ID             ID         `json:"-" bson:"-"`
	OrganizationID ID         `json:"-" bson:"-"`
	UserID         ID         `json:"-" bson:"-"`
	Role           MemberRole `json:"role" bson:"role"`
}
