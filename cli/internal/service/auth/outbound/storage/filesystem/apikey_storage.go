package filesystem

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/team-attention/cops/cli/internal/service/auth/outbound/storage"
)

// authJSON represents the ~/.cops/auth.json file structure.
type authJSON struct {
	APIKey string `json:"apiKey"`
}

// FilesystemAPIKeyStorage implements APIKeyStoragePort using filesystem.
type FilesystemAPIKeyStorage struct {
	logger   *slog.Logger
	authPath string
}

// NewFilesystemAPIKeyStorage creates a new filesystem API key storage.
func NewFilesystemAPIKeyStorage(l *slog.Logger) *FilesystemAPIKeyStorage {
	// Get user home directory, default to "." if error
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	// Construct authPath as ~/.cops/auth.json
	authPath := filepath.Join(homeDir, ".cops", "auth.json")

	return &FilesystemAPIKeyStorage{
		logger:   l.With(slog.String("name", "auth.storage.filesystem")),
		authPath: authPath,
	}
}

// SaveAPIKey stores the API key to ~/.cops/auth.json.
func (s *FilesystemAPIKeyStorage) SaveAPIKey(ctx context.Context, apiKey string) error {
	// Create authJSON struct with apiKey
	auth := authJSON{
		APIKey: apiKey,
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}

	// Ensure ~/.cops directory exists (0700 permission - user only)
	dir := filepath.Dir(s.authPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Write file with 0600 permission (user read/write only)
	if err := os.WriteFile(s.authPath, data, 0600); err != nil {
		return err
	}

	s.logger.Debug("API key saved successfully",
		slog.String("path", s.authPath),
	)

	return nil
}

// HasAPIKey checks if an API key exists in local storage.
func (s *FilesystemAPIKeyStorage) HasAPIKey(ctx context.Context) (bool, error) {
	// Check if authPath file exists using os.Stat
	if _, err := os.Stat(s.authPath); os.IsNotExist(err) {
		return false, nil
	}

	// Read file
	data, err := os.ReadFile(s.authPath)
	if err != nil {
		return false, err
	}

	// Unmarshal file
	var auth authJSON
	if err := json.Unmarshal(data, &auth); err != nil {
		return false, err
	}

	// Return true if apiKey is not empty
	return auth.APIKey != "", nil
}

// Compile-time interface verification
var _ storage.APIKeyStoragePort = (*FilesystemAPIKeyStorage)(nil)
