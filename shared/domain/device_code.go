package domain

import "time"

// DeviceCode represents a device authentication code for CLI login flow.
// Device codes are stored in MongoDB with automatic TTL expiration.
type DeviceCode struct {
	ID        ID        `json:"-" bson:"-"`
	UserCode  string    `json:"userCode" bson:"userCode"`
	UserID    *ID       `json:"userId,omitempty" bson:"-"`
	Approved  bool      `json:"approved" bson:"approved"`
	ExpiresAt time.Time `json:"expiresAt" bson:"expiresAt"`
}
