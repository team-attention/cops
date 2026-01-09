package mongoschema

import (
	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	APIKeyCollectionName = "apiKeys"
)

// APIKey field constants for MongoDB queries
const (
	APIKeyIDField         = "_id"
	APIKeyUserIDField     = "userId"
	APIKeyNameField       = "name"
	APIKeyKeyPrefixField  = "keyPrefix"
	APIKeyKeyHashField    = "keyHash"
	APIKeyCreatedAtField  = "createdAt"
	APIKeyLastUsedAtField = "lastUsedAt"
	APIKeyRevokedAtField  = "revokedAt"
	APIKeyExpiresAtField  = "expiresAt"
)

type APIKey struct {
	domain.APIKey `bson:",inline"`
	ID            bson.ObjectID `bson:"_id,omitempty"`
	UserID        bson.ObjectID `bson:"userId"`
}

// FromDomain converts a domain.APIKey to MongoDB schema.
func (s *APIKey) FromDomain(d *domain.APIKey) {
	if d == nil {
		return
	}

	s.APIKey = *d

	if d.ID != "" {
		s.ID, _ = bson.ObjectIDFromHex(string(d.ID))
	}

	if d.UserID != "" {
		s.UserID, _ = bson.ObjectIDFromHex(string(d.UserID))
	}
}

// ToDomain converts MongoDB schema to domain.APIKey.
func (s *APIKey) ToDomain() *domain.APIKey {
	if s == nil {
		return nil
	}

	s.APIKey.ID = domain.ID(s.ID.Hex())
	s.APIKey.UserID = domain.ID(s.UserID.Hex())

	return &s.APIKey
}
