package auth

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
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
	authPath    string
	mu          sync.RWMutex
	cachedState *AuthState
	lastLoad    time.Time
}

// NewService creates a new daemon auth service.
func NewService(l *slog.Logger, homeDir string) *Service {
	return &Service{
		logger:   l.With(slog.String("name", "daemon.auth.service")),
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

// IsAuthenticated checks if valid auth state exists.
func (s *Service) IsAuthenticated() bool {
	_, err := s.GetAccessToken()
	return err == nil
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
