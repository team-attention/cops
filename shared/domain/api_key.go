package domain

import "time"

// APIKey represents an API key for user-based authentication.
// API keys are issued per user and used by CLI/Hook to authenticate with the API server.
type APIKey struct {
	ID         ID         `json:"-" bson:"-"`
	UserID     ID         `json:"-" bson:"-"`
	Name       string     `json:"name" bson:"name"`
	KeyPrefix  string     `json:"keyPrefix" bson:"keyPrefix"`   // First 8 chars for identification (e.g., "cops_abc1")
	KeyHash    string     `json:"-" bson:"keyHash"`             // SHA-256 hash of full key (never expose)
	CreatedAt  time.Time  `json:"createdAt" bson:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty" bson:"lastUsedAt,omitempty"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty" bson:"revokedAt,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty" bson:"expiresAt,omitempty"`
}

// IsActive returns true if the API key is not revoked and not expired.
func (k *APIKey) IsActive() bool {
	// Check if revoked
	if k.RevokedAt != nil {
		return false
	}

	// Check if expired
	if k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now()) {
		return false
	}

	return true
}

// IsExpired returns true if the API key has passed its expiration time.
func (k *APIKey) IsExpired() bool {
	// If no expiration set, key never expires
	if k.ExpiresAt == nil {
		return false
	}

	return k.ExpiresAt.Before(time.Now())
}
