package mongoschema

import (
	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	DeviceCodeCollectionName = "deviceCodes"
)

// DeviceCode struct field constants
const (
	DeviceCodeIDField        = "_id"
	DeviceCodeUserCodeField  = "userCode"
	DeviceCodeUserIDField    = "userId"
	DeviceCodeApprovedField  = "approved"
	DeviceCodeExpiresAtField = "expiresAt"
)

type DeviceCode struct {
	domain.DeviceCode `bson:",inline"`
	ID                bson.ObjectID  `bson:"_id,omitempty"`
	UserID            *bson.ObjectID `bson:"userId,omitempty"`
}

func (s *DeviceCode) FromDomain(d *domain.DeviceCode) {
	if d == nil {
		return
	}

	s.DeviceCode = *d

	if d.ID != "" {
		s.ID, _ = bson.ObjectIDFromHex(string(d.ID))
	}

	if d.UserID != nil && *d.UserID != "" {
		oid, _ := bson.ObjectIDFromHex(string(*d.UserID))
		s.UserID = &oid
	}
}

func (s *DeviceCode) ToDomain() *domain.DeviceCode {
	if s == nil {
		return nil
	}

	s.DeviceCode.ID = domain.ID(s.ID.Hex())

	if s.UserID != nil {
		userID := domain.ID(s.UserID.Hex())
		s.DeviceCode.UserID = &userID
	}

	return &s.DeviceCode
}
