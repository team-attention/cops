package domain

// AccountProvider represents supported OAuth providers.
type AccountProvider string

const (
	AccountProviderGoogle AccountProvider = "google"
)

// Account represents an OAuth provider account linked to a user.
// Accounts are embedded within the User document, not stored separately.
type Account struct {
	Provider   AccountProvider `json:"provider" bson:"provider"`
	ProviderID string          `json:"providerId" bson:"providerId"`
}
