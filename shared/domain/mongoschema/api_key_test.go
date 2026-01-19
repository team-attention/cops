package mongoschema

import (
	"testing"
	"time"

	"github.com/team-attention/cops/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestAPIKey_FromDomain(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	lastUsed := now.Add(-time.Hour)

	tests := []struct {
		name   string
		domain *domain.APIKey
		verify func(t *testing.T, s *APIKey)
	}{
		{
			name:   "nil input",
			domain: nil,
			verify: func(t *testing.T, s *APIKey) {
				if s.ID != bson.NilObjectID {
					t.Error("expected empty ID for nil input")
				}
			},
		},
		{
			name: "valid input with all fields",
			domain: &domain.APIKey{
				ID:         "507f1f77bcf86cd799439011",
				UserID:     "507f1f77bcf86cd799439012",
				Name:       "Production Key",
				KeyPrefix:  "cops_abc1",
				KeyHash:    "sha256hash",
				CreatedAt:  now,
				LastUsedAt: &lastUsed,
			},
			verify: func(t *testing.T, s *APIKey) {
				if s.ID.Hex() != "507f1f77bcf86cd799439011" {
					t.Errorf("ID = %s, expected 507f1f77bcf86cd799439011", s.ID.Hex())
				}
				if s.UserID.Hex() != "507f1f77bcf86cd799439012" {
					t.Errorf("UserID = %s, expected 507f1f77bcf86cd799439012", s.UserID.Hex())
				}
				if s.Name != "Production Key" {
					t.Errorf("Name = %s, expected Production Key", s.Name)
				}
				if s.KeyPrefix != "cops_abc1" {
					t.Errorf("KeyPrefix = %s, expected cops_abc1", s.KeyPrefix)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s APIKey
			s.FromDomain(tt.domain)
			tt.verify(t, &s)
		})
	}
}

func TestAPIKey_ToDomain(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	lastUsed := now.Add(-time.Hour)

	tests := []struct {
		name   string
		schema *APIKey
		verify func(t *testing.T, d *domain.APIKey)
	}{
		{
			name:   "nil schema",
			schema: nil,
			verify: func(t *testing.T, d *domain.APIKey) {
				if d != nil {
					t.Error("expected nil for nil schema")
				}
			},
		},
		{
			name: "valid schema",
			schema: &APIKey{
				ID:     mustObjectID("507f1f77bcf86cd799439011"),
				UserID: mustObjectID("507f1f77bcf86cd799439012"),
				APIKey: domain.APIKey{
					Name:       "Production Key",
					KeyPrefix:  "cops_abc1",
					KeyHash:    "sha256hash",
					CreatedAt:  now,
					LastUsedAt: &lastUsed,
				},
			},
			verify: func(t *testing.T, d *domain.APIKey) {
				if d.ID != "507f1f77bcf86cd799439011" {
					t.Errorf("ID = %s, expected 507f1f77bcf86cd799439011", d.ID)
				}
				if d.UserID != "507f1f77bcf86cd799439012" {
					t.Errorf("UserID = %s, expected 507f1f77bcf86cd799439012", d.UserID)
				}
				if d.Name != "Production Key" {
					t.Errorf("Name = %s, expected Production Key", d.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := tt.schema.ToDomain()
			tt.verify(t, d)
		})
	}
}

func TestAPIKey_RoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	lastUsed := now.Add(-time.Hour)

	original := &domain.APIKey{
		ID:         "507f1f77bcf86cd799439011",
		UserID:     "507f1f77bcf86cd799439012",
		Name:       "Production Key",
		KeyPrefix:  "cops_abc1",
		KeyHash:    "sha256hash",
		CreatedAt:  now,
		LastUsedAt: &lastUsed,
	}

	// Convert to MongoDB schema
	var schema APIKey
	schema.FromDomain(original)

	// Convert back to domain
	result := schema.ToDomain()

	// Verify round-trip integrity
	if result.ID != original.ID {
		t.Errorf("ID mismatch: got %s, expected %s", result.ID, original.ID)
	}
	if result.UserID != original.UserID {
		t.Errorf("UserID mismatch: got %s, expected %s", result.UserID, original.UserID)
	}
	if result.Name != original.Name {
		t.Errorf("Name mismatch: got %s, expected %s", result.Name, original.Name)
	}
	if result.KeyPrefix != original.KeyPrefix {
		t.Errorf("KeyPrefix mismatch: got %s, expected %s", result.KeyPrefix, original.KeyPrefix)
	}
	if result.KeyHash != original.KeyHash {
		t.Errorf("KeyHash mismatch: got %s, expected %s", result.KeyHash, original.KeyHash)
	}
}

func mustObjectID(hex string) bson.ObjectID {
	oid, err := bson.ObjectIDFromHex(hex)
	if err != nil {
		panic(err)
	}
	return oid
}
