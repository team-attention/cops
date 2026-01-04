package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/team-attention/cops/cli/internal/platform/outbound/authstate"
	authapi "github.com/team-attention/cops/cli/internal/service/auth/outbound/api"
)

// AuthState represents the local authentication state.
type AuthState struct {
	Tokens *TokenInfo `json:"tokens"`
}

// TokenInfo contains token data.
type TokenInfo struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    int64  `json:"expiresAt"`
}

// FilesystemAuthState implements AuthStatePort using filesystem storage.
type FilesystemAuthState struct {
	logger    *slog.Logger
	authPath  string
	apiClient authapi.AuthAPIPort
}

// NewFilesystemAuthState creates a new filesystem-based auth state adapter.
func NewFilesystemAuthState(l *slog.Logger, apiClient authapi.AuthAPIPort) authstate.AuthStatePort {
	// 1. Get user home directory, default to "." if error.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	// 2. Construct auth path as ~/.cops/auth.json.
	authPath := filepath.Join(homeDir, ".cops", "auth.json")
	// 3. Create and return FilesystemAuthState with logger binding.
	return &FilesystemAuthState{
		logger:    l.With(slog.String("name", "platform.authstate.filesystem")),
		authPath:  authPath,
		apiClient: apiClient,
	}
}

// GetAccessToken returns a valid access token, refreshing if needed.
func (a *FilesystemAuthState) GetAccessToken(ctx context.Context) (string, error) {
	// 1. Read auth state from file:
	state, err := a.readAuthState()
	if err != nil {
		return "", fmt.Errorf("failed to read auth state: %w", err)
	}
	//    a. Check if auth file exists, return "not logged in" error if not.
	//    b. Read and unmarshal JSON file.
	//    c. Validate state has tokens, return "not logged in" error if nil.
	if state == nil || state.Tokens == nil {
		return "", fmt.Errorf("not logged in")
	}

	// 2. Check token expiry:
	//    a. Calculate time until expiry (tokenExpiry - now).
	now := time.Now().Unix()
	tokenExpiry := state.Tokens.ExpiresAt
	refreshBuffer := int64(300)

	//    b. If more than 300 seconds (5 min buffer), return current access token.
	if tokenExpiry-now > refreshBuffer {
		return state.Tokens.AccessToken, nil
	}

	// 3. Refresh token if near expiry:
	//    a. Log that token is being refreshed.
	a.logger.Info("access token near expiry, refreshing")

	//    b. Call apiClient.RefreshToken(ctx, refreshToken).
	result, err := a.apiClient.RefreshToken(ctx, state.Tokens.RefreshToken)
	//    c. If error, log and return error.
	if err != nil {
		a.logger.Error("failed to refresh token",
			slog.Any("error", err),
		)
		return "", fmt.Errorf("failed to refresh token: %w", err)
	}

	//    d. Update state with new tokens.
	state.Tokens.AccessToken = result.AccessToken
	state.Tokens.RefreshToken = result.RefreshToken
	state.Tokens.ExpiresAt = result.ExpiresAt

	//    e. Save updated state to file.
	if err := a.saveAuthState(state); err != nil {
		a.logger.Error("failed to save refreshed tokens",
			slog.Any("error", err),
		)
		return "", fmt.Errorf("failed to save refreshed tokens: %w", err)
	}

	//    f. Log success.
	a.logger.Info("token refreshed successfully")

	// 4. Return access token.
	return state.Tokens.AccessToken, nil
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

// saveAuthState writes auth state to file with secure permissions.
func (a *FilesystemAuthState) saveAuthState(state *AuthState) error {
	// 1. Ensure .cops directory exists (os.MkdirAll with 0700).
	copsDir := filepath.Dir(a.authPath)
	if err := os.MkdirAll(copsDir, 0700); err != nil {
		return fmt.Errorf("failed to create .cops directory: %w", err)
	}

	// 2. Marshal state to JSON with indentation.
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth state: %w", err)
	}

	// 3. Write file with 0600 permissions.
	if err := os.WriteFile(a.authPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write auth file: %w", err)
	}

	return nil
}

// Compile-time interface verification
var _ authstate.AuthStatePort = (*FilesystemAuthState)(nil)
