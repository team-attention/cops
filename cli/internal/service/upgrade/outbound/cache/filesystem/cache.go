package filesystem

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/team-attention/cops/cli/internal/service/upgrade/outbound/cache"
)

const (
	cacheFileName = "update-check.json"
	cacheDirName  = ".cops/cache"
)

// FilesystemCache implements CachePort using filesystem storage.
type FilesystemCache struct {
	logger    *slog.Logger
	cachePath string
}

// NewFilesystemCache creates a new filesystem cache adapter.
func NewFilesystemCache(l *slog.Logger) (*FilesystemCache, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(homeDir, cacheDirName)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}

	return &FilesystemCache{
		logger:    l.With(slog.String("name", "upgrade.cache.filesystem")),
		cachePath: filepath.Join(cacheDir, cacheFileName),
	}, nil
}

// Get retrieves the cached update check information.
func (c *FilesystemCache) Get(ctx context.Context) (*cache.UpdateCheckCache, error) {
	data, err := os.ReadFile(c.cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cached cache.UpdateCheckCache
	if err := json.Unmarshal(data, &cached); err != nil {
		c.logger.Debug("invalid cache file, ignoring", slog.Any("error", err))
		return nil, nil
	}

	return &cached, nil
}

// Set stores the update check information in cache.
func (c *FilesystemCache) Set(ctx context.Context, cached *cache.UpdateCheckCache) error {
	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.cachePath, data, 0644)
}

// IsValid checks if the cache is still valid (within TTL).
func (c *FilesystemCache) IsValid(cached *cache.UpdateCheckCache, ttl time.Duration) bool {
	if cached == nil {
		return false
	}
	return time.Since(cached.CheckedAt) < ttl
}

var _ cache.CachePort = (*FilesystemCache)(nil)
