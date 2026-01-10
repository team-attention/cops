package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/team-attention/cops/daemon/internal/platform/outbound/authstate"
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
func NewFilesystemAuthState(l *slog.Logger, homeDir string) authstate.AuthStatePort {
	authPath := filepath.Join(homeDir, ".cops", "auth.json")
	return &FilesystemAuthState{
		logger:   l.With(slog.String("name", "platform.authstate.filesystem")),
		authPath: authPath,
	}
}

// GetAccessToken returns the API key from auth.json.
// Note: Returns API key as "access token" for backward compatibility.
// The API key is used as Bearer token in Authorization header.
func (a *FilesystemAuthState) GetAccessToken(ctx context.Context) (string, error) {
	state, err := a.readAuthState()
	if err != nil {
		return "", fmt.Errorf("failed to read auth state: %w", err)
	}
	if state == nil {
		return "", fmt.Errorf("not logged in")
	}

	if state.APIKey == "" {
		return "", fmt.Errorf("not logged in")
	}

	return state.APIKey, nil
}

// readAuthState reads auth state from file.
func (a *FilesystemAuthState) readAuthState() (*AuthState, error) {
	if _, err := os.Stat(a.authPath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(a.authPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth file: %w", err)
	}

	var state AuthState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse auth file: %w", err)
	}

	return &state, nil
}

// Compile-time interface verification.
var _ authstate.AuthStatePort = (*FilesystemAuthState)(nil)
