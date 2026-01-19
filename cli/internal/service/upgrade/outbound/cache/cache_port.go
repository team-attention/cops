package cache

import (
	"context"
	"time"
)

// UpdateCheckCache represents cached update check information.
type UpdateCheckCache struct {
	CheckedAt      time.Time `json:"checked_at"`
	CurrentVersion string    `json:"current_version"`
	LatestVersion  string    `json:"latest_version"`
	HasUpdate      bool      `json:"has_update"`
}

// CachePort defines operations for caching update check results.
type CachePort interface {
	// Get retrieves the cached update check information.
	// Returns nil if cache doesn't exist or is invalid.
	Get(ctx context.Context) (*UpdateCheckCache, error)

	// Set stores the update check information in cache.
	Set(ctx context.Context, cache *UpdateCheckCache) error

	// IsValid checks if the cache is still valid (within TTL).
	IsValid(cache *UpdateCheckCache, ttl time.Duration) bool
}
