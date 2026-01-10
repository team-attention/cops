package domain

import (
	"testing"
	"time"
)

func TestAPIKey_IsActive(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name     string
		apiKey   APIKey
		expected bool
	}{
		{
			name:     "active key with no expiry and no revocation",
			apiKey:   APIKey{RevokedAt: nil, ExpiresAt: nil},
			expected: true,
		},
		{
			name:     "revoked key",
			apiKey:   APIKey{RevokedAt: &now, ExpiresAt: nil},
			expected: false,
		},
		{
			name:     "expired key",
			apiKey:   APIKey{RevokedAt: nil, ExpiresAt: &past},
			expected: false,
		},
		{
			name:     "active key with future expiry",
			apiKey:   APIKey{RevokedAt: nil, ExpiresAt: &future},
			expected: true,
		},
		{
			name:     "revoked and expired key",
			apiKey:   APIKey{RevokedAt: &now, ExpiresAt: &past},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.apiKey.IsActive()
			if result != tt.expected {
				t.Errorf("IsActive() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestAPIKey_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name     string
		apiKey   APIKey
		expected bool
	}{
		{
			name:     "no expiration set",
			apiKey:   APIKey{ExpiresAt: nil},
			expected: false,
		},
		{
			name:     "expired key",
			apiKey:   APIKey{ExpiresAt: &past},
			expected: true,
		},
		{
			name:     "not expired key",
			apiKey:   APIKey{ExpiresAt: &future},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.apiKey.IsExpired()
			if result != tt.expected {
				t.Errorf("IsExpired() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
