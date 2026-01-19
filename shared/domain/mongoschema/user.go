package mongoschema

import (
	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	UserCollectionName = "users"
)

// User struct field constants
const (
	UserIDField              = "_id"
	UserEmailField           = "email"
	UserNameField            = "name"
	UserProfileImageURLField = "profileImageUrl"
	UserAccountsField        = "accounts"
)

// Account struct field constants (embedded type gets its own constants)
const (
	AccountProviderField   = "provider"
	AccountProviderIDField = "providerId"
)

type User struct {
	domain.User `bson:",inline"`
	ID          bson.ObjectID `bson:"_id,omitempty"`
}

func (s *User) FromDomain(d *domain.User) {
	if d == nil {
		return
	}

	s.User = *d

	if d.ID != "" {
		s.ID, _ = bson.ObjectIDFromHex(string(d.ID))
	}
}

func (s *User) ToDomain() *domain.User {
	if s == nil {
		return nil
	}

	s.User.ID = domain.ID(s.ID.Hex())

	return &s.User
}
