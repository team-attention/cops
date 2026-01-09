package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/team-attention/cops/cli/internal/platform/outbound/authstate"
)

// AuthState represents the local authentication state.
// Simplified to support API key only (no OAuth tokens).
type AuthState struct {
	APIKey string `json:"apiKey"`
}

// FilesystemAuthState implements AuthStatePort using filesystem storage.
type FilesystemAuthState struct {
	logger   *slog.Logger
	authPath string
}

// NewFilesystemAuthState creates a new filesystem-based auth state adapter.
func NewFilesystemAuthState(l *slog.Logger) authstate.AuthStatePort {
	// 1. Get user home directory, default to "." if error.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	// 2. Construct auth path as ~/.cops/auth.json.
	authPath := filepath.Join(homeDir, ".cops", "auth.json")
	// 3. Create and return FilesystemAuthState with logger binding.
	// 4. No apiClient field needed.
	return &FilesystemAuthState{
		logger:   l.With(slog.String("name", "platform.authstate.filesystem")),
		authPath: authPath,
	}
}

// GetAccessToken returns the API key from auth.json.
// Note: Returns API key as "access token" for backward compatibility.
// The API key is used as Bearer token in Authorization header.
func (a *FilesystemAuthState) GetAccessToken(ctx context.Context) (string, error) {
	// 1. Read auth state from file:
	//    a. Check if auth file exists, return "not logged in" error if not.
	//    b. Read and unmarshal JSON file.
	state, err := a.readAuthState()
	if err != nil {
		return "", fmt.Errorf("failed to read auth state: %w", err)
	}
	if state == nil {
		return "", fmt.Errorf("not logged in")
	}

	// 2. Validate API key exists:
	//    a. If empty, return "not logged in" error.
	if state.APIKey == "" {
		return "", fmt.Errorf("not logged in")
	}

	// 3. Return API key directly (no refresh logic needed).
	return state.APIKey, nil
}

// readAuthState reads auth state from file.
func (a *FilesystemAuthState) readAuthState() (*AuthState, error) {
	// 1. Check if file exists using os.Stat.
	if _, err := os.Stat(a.authPath); os.IsNotExist(err) {
		// 2. If not exists, return nil, nil (not logged in).
		return nil, nil
	}

	// 3. Read file contents.
	data, err := os.ReadFile(a.authPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth file: %w", err)
	}

	// 4. Unmarshal JSON into AuthState.
	var state AuthState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse auth file: %w", err)
	}

	// 5. Return state.
	return &state, nil
}

// Compile-time interface verification
var _ authstate.AuthStatePort = (*FilesystemAuthState)(nil)
