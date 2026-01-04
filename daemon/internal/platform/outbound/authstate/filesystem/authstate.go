package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/team-attention/cops/daemon/internal/platform/outbound/authstate"
	authapi "github.com/team-attention/cops/daemon/internal/service/auth/outbound/api"
)

// refreshBuffer is the duration before token expiry to trigger proactive refresh (5 minutes).
const refreshBuffer = int64(300)

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
	mu        sync.RWMutex
	refreshMu sync.Mutex
}

// NewFilesystemAuthState creates a new filesystem-based auth state adapter.
func NewFilesystemAuthState(l *slog.Logger, apiClient authapi.AuthAPIPort, homeDir string) authstate.AuthStatePort {
	authPath := filepath.Join(homeDir, ".cops", "auth.json")
	return &FilesystemAuthState{
		logger:    l.With(slog.String("name", "platform.authstate.filesystem")),
		authPath:  authPath,
		apiClient: apiClient,
	}
}

// GetAccessToken returns a valid access token, refreshing if needed.
func (a *FilesystemAuthState) GetAccessToken(ctx context.Context) (string, error) {
	a.mu.RLock()
	state, err := a.readAuthState()
	a.mu.RUnlock()

	if err != nil {
		return "", fmt.Errorf("failed to read auth state: %w", err)
	}

	if state == nil || state.Tokens == nil {
		return "", fmt.Errorf("not authenticated")
	}

	now := time.Now().Unix()
	tokenExpiry := state.Tokens.ExpiresAt

	if tokenExpiry-now > refreshBuffer {
		return state.Tokens.AccessToken, nil
	}

	a.logger.Info("access token near expiry or expired, refreshing")

	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()

	// Double-check: Re-read state and check expiry again (another goroutine may have refreshed)
	a.mu.RLock()
	state, err = a.readAuthState()
	a.mu.RUnlock()

	if err != nil {
		return "", fmt.Errorf("failed to read auth state: %w", err)
	}

	if state == nil || state.Tokens == nil {
		return "", fmt.Errorf("not authenticated")
	}

	now = time.Now().Unix()
	if state.Tokens.ExpiresAt-now > refreshBuffer {
		return state.Tokens.AccessToken, nil
	}

	result, err := a.apiClient.RefreshToken(ctx, state.Tokens.RefreshToken)
	if err != nil {
		a.logger.Error("failed to refresh token",
			slog.Any("error", err),
		)
		return "", fmt.Errorf("failed to refresh token: %w", err)
	}

	state.Tokens.AccessToken = result.AccessToken
	state.Tokens.RefreshToken = result.RefreshToken
	state.Tokens.ExpiresAt = result.ExpiresAt

	if err := a.saveAuthState(state); err != nil {
		a.logger.Error("failed to save refreshed tokens",
			slog.Any("error", err),
		)
		return "", fmt.Errorf("failed to save refreshed tokens: %w", err)
	}

	a.logger.Info("token refreshed successfully")

	return state.Tokens.AccessToken, nil
}

// readAuthState reads auth state from file (must hold mu.RLock or mu.Lock).
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

// saveAuthState writes auth state to file with secure permissions.
func (a *FilesystemAuthState) saveAuthState(state *AuthState) error {
	copsDir := filepath.Dir(a.authPath)
	if err := os.MkdirAll(copsDir, 0700); err != nil {
		return fmt.Errorf("failed to create .cops directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth state: %w", err)
	}

	if err := os.WriteFile(a.authPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write auth file: %w", err)
	}

	return nil
}

// Compile-time interface verification.
var _ authstate.AuthStatePort = (*FilesystemAuthState)(nil)
