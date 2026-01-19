package domain

// User represents an authenticated user in the system.
// A user can have multiple linked OAuth accounts embedded in the Accounts array.
type User struct {
	ID              ID         `json:"-" bson:"-"`
	Email           string     `json:"email" bson:"email"`
	Name            string     `json:"name" bson:"name"`
	ProfileImageURL string     `json:"profileImageUrl,omitempty" bson:"profileImageUrl,omitempty"`
	Accounts        []*Account `json:"accounts" bson:"accounts"`
}
