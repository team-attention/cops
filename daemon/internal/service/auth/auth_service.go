package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/team-attention/cops/daemon/internal/service/auth/outbound/api"
)

// AuthState mirrors CLI auth state structure.
type AuthState struct {
	Tokens *TokenInfo `json:"tokens"`
}

type TokenInfo struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    int64  `json:"expiresAt"`
}

// Service manages daemon authentication state.
type Service struct {
	logger      *slog.Logger
	authAPI     api.AuthAPIPort
	authPath    string
	mu          sync.RWMutex
	refreshMu   sync.Mutex // Separate mutex for token refresh to prevent concurrent refresh attempts
	cachedState *AuthState
	lastLoad    time.Time
}

// NewService creates a new daemon auth service.
func NewService(l *slog.Logger, authAPI api.AuthAPIPort, homeDir string) *Service {
	return &Service{
		logger:   l.With(slog.String("name", "daemon.auth.service")),
		authAPI:  authAPI,
		authPath: filepath.Join(homeDir, ".cops", "auth.json"),
	}
}

// GetAccessToken returns current access token, reloading from file if stale.
func (s *Service) GetAccessToken() (string, error) {
	s.mu.RLock()
	shouldReload := s.cachedState == nil || time.Since(s.lastLoad) > 30*time.Second
	s.mu.RUnlock()

	if shouldReload {
		if err := s.reloadAuthState(); err != nil {
			return "", err
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.cachedState == nil || s.cachedState.Tokens == nil {
		return "", fmt.Errorf("not authenticated")
	}

	now := time.Now().Unix()
	if s.cachedState.Tokens.ExpiresAt <= now {
		s.logger.Warn("access token expired",
			slog.Int64("expiresAt", s.cachedState.Tokens.ExpiresAt),
			slog.Int64("now", now),
		)
		return "", fmt.Errorf("access token expired")
	}

	return s.cachedState.Tokens.AccessToken, nil
}

// GetRefreshToken returns the current refresh token.
func (s *Service) GetRefreshToken() (string, error) {
	s.mu.RLock()
	shouldReload := s.cachedState == nil || time.Since(s.lastLoad) > 30*time.Second
	s.mu.RUnlock()

	if shouldReload {
		if err := s.reloadAuthState(); err != nil {
			return "", err
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.cachedState == nil || s.cachedState.Tokens == nil {
		return "", fmt.Errorf("not authenticated")
	}

	return s.cachedState.Tokens.RefreshToken, nil
}

// RefreshAccessToken uses the refresh token to obtain a new access token.
// Thread-safe: only one refresh operation can occur at a time.
// Returns the new access token on success.
func (s *Service) RefreshAccessToken(ctx context.Context) (string, error) {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	refreshToken, err := s.GetRefreshToken()
	if err != nil {
		return "", err
	}

	s.logger.Info("refreshing access token")

	result, err := s.authAPI.RefreshToken(ctx, refreshToken)
	if err != nil {
		s.logger.Error("failed to refresh token",
			slog.Any("error", err),
		)
		return "", err
	}

	newState := &AuthState{
		Tokens: &TokenInfo{
			AccessToken:  result.AccessToken,
			RefreshToken: result.RefreshToken,
			ExpiresAt:    result.ExpiresAt,
		},
	}

	if err := s.saveAuthState(newState); err != nil {
		s.logger.Error("failed to save refreshed tokens",
			slog.Any("error", err),
		)
		return "", err
	}

	s.mu.Lock()
	s.cachedState = newState
	s.lastLoad = time.Now()
	s.mu.Unlock()

	s.logger.Info("token refreshed successfully")

	return newState.Tokens.AccessToken, nil
}

// IsAuthenticated checks if valid auth state exists.
func (s *Service) IsAuthenticated() bool {
	_, err := s.GetAccessToken()
	return err == nil
}

// InvalidateCache forces reload of auth state from file on next access.
func (s *Service) InvalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cachedState = nil
	s.lastLoad = time.Time{}
}

// reloadAuthState reloads auth state from file.
func (s *Service) reloadAuthState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.authPath); os.IsNotExist(err) {
		s.cachedState = nil
		s.lastLoad = time.Now()
		return fmt.Errorf("auth file not found")
	}

	data, err := os.ReadFile(s.authPath)
	if err != nil {
		return fmt.Errorf("failed to read auth file: %w", err)
	}

	var state AuthState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to parse auth file: %w", err)
	}

	s.cachedState = &state
	s.lastLoad = time.Now()

	return nil
}

// saveAuthState writes auth state to file with secure permissions.
func (s *Service) saveAuthState(state *AuthState) error {
	copsDir := filepath.Dir(s.authPath)
	if err := os.MkdirAll(copsDir, 0700); err != nil {
		return fmt.Errorf("failed to create .cops directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth state: %w", err)
	}

	if err := os.WriteFile(s.authPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write auth file: %w", err)
	}

	return nil
}
