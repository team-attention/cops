package filesystem

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/team-attention/cops/cli/internal/platform/outbound/apikey"
	"github.com/team-attention/cops/cli/internal/platform/util/errutil"
)

// EnvAPIKey is the environment variable name for API key override.
const EnvAPIKey = "COPS_API_KEY"

// authJSON represents the ~/.cops/auth.json file structure.
// Note: This is distinct from the OAuth tokens stored by FilesystemAuthState.
// auth.json contains the project API key for hook authentication.
type authJSON struct {
	APIKey string `json:"apiKey"`
}

// FilesystemAPIKey implements APIKeyPort using environment variable
// with filesystem fallback.
type FilesystemAPIKey struct {
	logger   *slog.Logger
	authPath string
}

// NewFilesystemAPIKey creates a new filesystem-based API key adapter.
func NewFilesystemAPIKey(l *slog.Logger, authPath string) apikey.APIKeyPort {
	return &FilesystemAPIKey{
		logger:   l.With(slog.String("name", "platform.apikey.filesystem")),
		authPath: authPath,
	}
}


// GetAPIKey returns the API key for authentication.
// Priority: 1) COPS_API_KEY environment variable, 2) ~/.cops/auth.json file.
// Returns error if no API key is available or the key is empty.
func (a *FilesystemAPIKey) GetAPIKey(ctx context.Context) (string, error) {
	// 1. Check COPS_API_KEY environment variable first.
	if envKey := os.Getenv(EnvAPIKey); envKey != "" {
		// 1a. Log that env var is being used (debug level).
		a.logger.Debug("using API key from environment variable")
		// 1b. Return the env var value (no empty check needed, already checked).
		return envKey, nil
	}

	// 2. Read from ~/.cops/auth.json file.
	// 2a. Check if file exists using os.Stat.
	if _, err := os.Stat(a.authPath); os.IsNotExist(err) {
		// 2b. File not found: return NotFound error with message "auth.json not found".
		return "", errutil.NotFound("auth.json not found")
	}

	// 2c. Read file contents using os.ReadFile.
	data, err := os.ReadFile(a.authPath)
	if err != nil {
		// 2d. Read error: return Internal error wrapping the cause.
		return "", errutil.Wrap(errutil.ErrorTypeInternal, "failed to read auth.json", err)
	}

	// 3. Parse JSON into authJSON struct.
	var auth authJSON
	if err := json.Unmarshal(data, &auth); err != nil {
		// 3a. Parse error: return BadRequest error with message "failed to parse auth.json".
		return "", errutil.Wrap(errutil.ErrorTypeBadRequest, "failed to parse auth.json", err)
	}

	// 4. Validate API key is not empty.
	if auth.APIKey == "" {
		// 4a. Empty key: return BadRequest error with message "apiKey is empty".
		return "", errutil.BadRequest("apiKey is empty")
	}

	// 5. Log success and return API key.
	a.logger.Debug("loaded API key from file", slog.String("path", a.authPath))
	return auth.APIKey, nil
}

// Compile-time interface verification
var _ apikey.APIKeyPort = (*FilesystemAPIKey)(nil)
